/*
Copyright The ORC Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package kustomizeref

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
	"sigs.k8s.io/kube-api-linter/pkg/analysis/helpers/extractjsontags"
	"sigs.k8s.io/kube-api-linter/pkg/analysis/helpers/inspector"
	"sigs.k8s.io/kube-api-linter/pkg/analysis/helpers/markers"
	"sigs.k8s.io/kube-api-linter/pkg/analysis/initializer"
	"sigs.k8s.io/kube-api-linter/pkg/analysis/registry"
)

const (
	name = "kustomizeref"

	// kustomizeRefMarker is the identifier of the marker which must be
	// present on every field which references another ORC object (or a
	// core Secret) by KubernetesNameRef, naming the Kind it refers to.
	//
	// It is consumed by cmd/kustomizeconfig-generator to generate
	// examples/components/kustomizeconfig/kustomizeconfig.yaml.
	kustomizeRefMarker = "orc:kustomize:ref"

	doc = `Requires every KubernetesNameRef field to carry an +orc:kustomize:ref=<Kind> marker.

examples/components/kustomizeconfig/kustomizeconfig.yaml is generated from
these markers (see cmd/kustomizeconfig-generator) so that kustomize can
correctly rewrite cross-object name references (e.g. when a nameSuffix or
nameReference transformer is applied to the examples). A missing marker
produces no diff in the generated file, so this can't be caught by
'make verify-generated' alone.

Fields declared on a struct whose name contains "Status" are exempt, since
status fields are server-observed and not subject to kustomize name
substitution.

See: https://k-orc.cloud/development/api-design/`
)

// Analyzer is the analyzer for the kustomizeref linter.
var Analyzer = &analysis.Analyzer{
	Name:     name,
	Doc:      doc,
	Run:      run,
	Requires: []*analysis.Analyzer{inspector.Analyzer},
}

func init() {
	registry.DefaultRegistry().RegisterLinter(initializer.NewInitializer(
		name,
		Analyzer,
		false, // not enabled by default - must be explicitly enabled
	))
}

func run(pass *analysis.Pass) (any, error) {
	inspect, ok := pass.ResultOf[inspector.Analyzer].(inspector.Inspector)
	if !ok {
		return nil, nil
	}

	inspect.InspectFieldsIncludingListTypes(func(field *ast.Field, _ extractjsontags.FieldTagInfo, markersAccess markers.Markers, qualifiedFieldName string) {
		checkField(pass, field, markersAccess, qualifiedFieldName)
	})

	return nil, nil
}

func checkField(pass *analysis.Pass, field *ast.Field, markersAccess markers.Markers, qualifiedFieldName string) {
	// qualifiedFieldName is in the form "StructName.FieldName"
	parts := strings.SplitN(qualifiedFieldName, ".", 2)
	if len(parts) != 2 {
		return
	}

	structName := parts[0]

	// Status fields are server-observed and are never rewritten by
	// kustomize name substitution.
	if strings.Contains(structName, "Status") {
		return
	}

	if !isKubernetesNameRefType(field.Type) {
		return
	}

	if hasKustomizeRefMarker(markersAccess.FieldMarkers(field)) {
		return
	}

	pass.Reportf(field.Pos(),
		"field %s references another object by KubernetesNameRef but has no +%s=<Kind> marker; "+
			"see https://k-orc.cloud/development/api-design/",
		qualifiedFieldName, kustomizeRefMarker)
}

func hasKustomizeRefMarker(fieldMarkers markers.MarkerSet) bool {
	for _, m := range fieldMarkers.Get(kustomizeRefMarker) {
		if m.Payload.Value != "" {
			return true
		}
	}

	return false
}

// isKubernetesNameRefType checks if the expression is KubernetesNameRef,
// *KubernetesNameRef, or []KubernetesNameRef.
func isKubernetesNameRefType(expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name == "KubernetesNameRef"
	case *ast.StarExpr:
		return isKubernetesNameRefType(e.X)
	case *ast.ArrayType:
		return isKubernetesNameRefType(e.Elt)
	default:
		return false
	}
}
