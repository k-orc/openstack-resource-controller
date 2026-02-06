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

// Package orcapilinter is a golangci-lint plugin that extends kube-api-linter
// with ORC-specific API design rules.
//
// It imports the base kube-api-linter plugin and registers additional ORC-specific
// linters with the registry. This allows all linters to be configured through
// the kubeapilinter section in .golangci.yml.
package orcapilinter

import (
	pluginbase "sigs.k8s.io/kube-api-linter/pkg/plugin/base"

	// Import the default kube-api-linter linters.
	_ "sigs.k8s.io/kube-api-linter/pkg/registration"

	// Import ORC-specific linters to register them with the registry.
	_ "github.com/k-orc/openstack-resource-controller/v2/tools/orc-api-linter/pkg/analysis/noopenstackidref"
)

// New is the entrypoint for the plugin.
// We use the base kube-api-linter plugin which will include both the standard
// KAL linters and our ORC-specific linters registered via init().
//
//nolint:gochecknoglobals
var New = pluginbase.New
