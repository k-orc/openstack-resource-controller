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

// Package noopenstackidref provides a linter that enforces ORC's API design
// philosophy of referencing ORC Kubernetes objects rather than OpenStack
// resources directly by UUID.
//
// # Overview
//
// ORC (OpenStack Resource Controller) manages OpenStack resources through
// Kubernetes custom resources. The API design philosophy states that spec
// fields should only reference other ORC objects, not OpenStack resources
// directly by UUID.
//
// # What this linter checks
//
// The linter flags fields in spec-related structs that:
//   - Have names matching OpenStack resource ID patterns (e.g., ProjectID, NetworkID)
//   - Are of type *string
//
// These should instead use *KubernetesNameRef with a 'Ref' suffix.
//
// # Examples
//
// Bad (will be flagged):
//
//	type UserResourceSpec struct {
//	    DefaultProjectID *string `json:"defaultProjectID,omitempty"`
//	}
//
// Good (correct pattern):
//
//	type UserResourceSpec struct {
//	    DefaultProjectRef *KubernetesNameRef `json:"defaultProjectRef,omitempty"`
//	}
//
// # Status structs are exempt
//
// Fields in status structs (ending with 'Status' or 'ResourceStatus') are
// allowed to have OpenStack IDs, as they report what OpenStack returned.
//
// See https://k-orc.cloud/development/architecture/#api-design-philosophy
// for more details on ORC's API design philosophy.
package noopenstackidref
