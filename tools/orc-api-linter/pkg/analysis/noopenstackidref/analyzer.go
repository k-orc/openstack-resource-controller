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

package noopenstackidref

import (
	"go/ast"
	"regexp"
	"slices"
	"strings"

	"golang.org/x/tools/go/analysis"
	"sigs.k8s.io/kube-api-linter/pkg/analysis/helpers/extractjsontags"
	"sigs.k8s.io/kube-api-linter/pkg/analysis/helpers/inspector"
	"sigs.k8s.io/kube-api-linter/pkg/analysis/helpers/markers"
	"sigs.k8s.io/kube-api-linter/pkg/analysis/initializer"
	"sigs.k8s.io/kube-api-linter/pkg/analysis/registry"
)

const (
	name = "noopenstackidref"
	doc  = `Flags OpenStack ID references in spec structs.

ORC's API design philosophy states that spec fields should only reference
ORC Kubernetes objects, not OpenStack resources directly by UUID.

Fields ending with 'ID' or 'IDs' (like ProjectID, NetworkIDs) in spec structs should
instead use KubernetesNameRef type with a 'Ref' or 'Refs' suffix (like ProjectRef, NetworkRefs).

Additionally, fields ending with 'Ref' or 'Refs' must use the KubernetesNameRef type,
not other types like OpenStackName or string.

See: https://k-orc.cloud/development/api-design/`
)

// openstackIDPattern matches field names that end with "ID" or "IDs" and are likely
// references to OpenStack resources by UUID. These should instead use
// KubernetesNameRef with a "Ref" or "Refs" suffix to reference ORC objects.
var openstackIDPattern = regexp.MustCompile(`IDs?$`)

// refPattern matches field names that end with "Ref" or "Refs".
// These fields should use KubernetesNameRef type.
var refPattern = regexp.MustCompile(`Refs?$`)

// excludedIDPatterns contains field name patterns that end in "ID" or "IDs" but are
// not OpenStack resource references.
var excludedIDPatterns = []string{
	"SegmentationID", // VLAN segmentation ID, not an OpenStack resource
}

// excludedRefPatterns contains field name patterns that end in "Ref" or "Refs" but
// intentionally use a different type than KubernetesNameRef.
var excludedRefPatterns = []string{
	"CloudCredentialsRef", // References a credentials secret, not an ORC object
}

// excludedStructs contains struct names that should not be checked even though
// they don't have "Status" in their name. These are typically nested types used
// exclusively within status structs.
var excludedStructs = []string{
	"ServerInterfaceFixedIP", // Used only in ServerInterfaceStatus.FixedIPs
}

// Analyzer is the analyzer for the noopenstackidref linter.
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

	inspect.InspectFieldsIncludingListTypes(func(field *ast.Field, _ extractjsontags.FieldTagInfo, _ markers.Markers, qualifiedFieldName string) {
		checkField(pass, field, qualifiedFieldName)
	})

	return nil, nil
}

func checkField(pass *analysis.Pass, field *ast.Field, qualifiedFieldName string) {
	// qualifiedFieldName is in the form "StructName.FieldName"
	parts := strings.SplitN(qualifiedFieldName, ".", 2)
	if len(parts) != 2 {
		return
	}

	structName := parts[0]
	fieldName := parts[1]

	// Only check spec-related structs, not status structs
	if !isSpecStruct(structName) {
		return
	}

	// Check if field name ends in Ref/Refs but uses wrong type
	if refPattern.MatchString(fieldName) {
		// Check if field name is in the Ref exclusion list
		if slices.Contains(excludedRefPatterns, fieldName) {
			return
		}

		if !isKubernetesNameRefTypeOrSlice(field.Type) {
			pass.Reportf(field.Pos(),
				"field %s has Ref suffix but does not use KubernetesNameRef type; "+
					"see https://k-orc.cloud/development/api-design/",
				qualifiedFieldName)
		}
		return
	}

	// Check if field name matches OpenStack ID pattern
	if !openstackIDPattern.MatchString(fieldName) {
		return
	}

	// Check if field name is in the exclusion list
	if slices.Contains(excludedIDPatterns, fieldName) {
		return
	}

	// Allow *KubernetesNameRef type (correct type, even if name ends in ID)
	if isKubernetesNameRefType(field.Type) {
		return
	}

	// Generate the suggested Ref/Refs name based on singular/plural
	var suggestedRef string
	if strings.HasSuffix(fieldName, "IDs") {
		suggestedRef = strings.TrimSuffix(fieldName, "IDs") + "Refs"
	} else {
		suggestedRef = strings.TrimSuffix(fieldName, "ID") + "Ref"
	}

	pass.Reportf(field.Pos(),
		"field %s references OpenStack resource by ID in spec; "+
			"use *KubernetesNameRef with %s instead; "+
			"see https://k-orc.cloud/development/api-design/",
		qualifiedFieldName, suggestedRef)
}

// isSpecStruct returns true if the struct name indicates it's a spec-related struct
// (where OpenStack ID references should be flagged), not a status struct
// (where OpenStack IDs are expected and valid).
func isSpecStruct(structName string) bool {
	// Status structs are allowed to have OpenStack IDs
	if strings.HasSuffix(structName, "Status") ||
		strings.Contains(structName, "Status") {
		return false
	}

	// Check excluded structs (nested types used only in status contexts)
	if slices.Contains(excludedStructs, structName) {
		return false
	}

	// All other structs should use KubernetesNameRef for references
	return true
}

// isKubernetesNameRefType checks if the expression is KubernetesNameRef or *KubernetesNameRef.
// This is the only acceptable type for fields that might look like ID references.
func isKubernetesNameRefType(expr ast.Expr) bool {
	// Check for *KubernetesNameRef
	if starExpr, ok := expr.(*ast.StarExpr); ok {
		if ident, ok := starExpr.X.(*ast.Ident); ok {
			return ident.Name == "KubernetesNameRef"
		}
		return false
	}

	// Check for KubernetesNameRef (non-pointer)
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name == "KubernetesNameRef"
	}

	return false
}

// isKubernetesNameRefTypeOrSlice checks if the expression is KubernetesNameRef,
// *KubernetesNameRef, or []KubernetesNameRef. This is used for Ref/Refs fields
// which may be singular or plural.
func isKubernetesNameRefTypeOrSlice(expr ast.Expr) bool {
	// Check for []KubernetesNameRef
	if arrayType, ok := expr.(*ast.ArrayType); ok {
		return isKubernetesNameRefType(arrayType.Elt)
	}

	// Check for KubernetesNameRef or *KubernetesNameRef
	return isKubernetesNameRefType(expr)
}
