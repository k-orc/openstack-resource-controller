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

package a

// KubernetesNameRef is a reference to a Kubernetes object by name.
type KubernetesNameRef string

// ---- Spec structs: missing markers should be flagged ----

// ProjectResourceSpec has a marked field and an unmarked field.
type ProjectResourceSpec struct {
	// name is fine, not a reference field.
	Name *string `json:"name,omitempty"`

	// domainRef is correctly marked.
	// +orc:kustomize:ref=Domain
	DomainRef *KubernetesNameRef `json:"domainRef,omitempty"`

	// projectRef has no marker and should be flagged.
	ProjectRef *KubernetesNameRef `json:"projectRef,omitempty"` // want `field ProjectResourceSpec.ProjectRef references another object by KubernetesNameRef but has no \+orc:kustomize:ref=<Kind> marker`
}

// PortResourceSpec tests the non-pointer and slice variants.
type PortResourceSpec struct {
	// networkRef is correctly marked and non-pointer.
	// +orc:kustomize:ref=Network
	NetworkRef KubernetesNameRef `json:"networkRef,omitempty"`

	// securityGroupRefs is correctly marked and a slice.
	// +orc:kustomize:ref=SecurityGroup
	SecurityGroupRefs []KubernetesNameRef `json:"securityGroupRefs,omitempty"`

	// subnetRef is a slice with no marker and should be flagged.
	SubnetRefs []KubernetesNameRef `json:"subnetRefs,omitempty"` // want `field PortResourceSpec.SubnetRefs references another object by KubernetesNameRef but has no \+orc:kustomize:ref=<Kind> marker`
}

// ---- Filter structs: missing markers should also be flagged ----

// NetworkFilter is a filter struct that should be checked.
type NetworkFilter struct {
	// name is fine.
	Name *string `json:"name,omitempty"`

	// projectRef has no marker and should be flagged.
	ProjectRef *KubernetesNameRef `json:"projectRef,omitempty"` // want `field NetworkFilter.ProjectRef references another object by KubernetesNameRef but has no \+orc:kustomize:ref=<Kind> marker`
}

// ---- Status structs: exempt ----

// ProjectResourceStatus is a status struct, exempt even without a marker.
type ProjectResourceStatus struct {
	// domainRef is allowed without a marker in status.
	DomainRef KubernetesNameRef `json:"domainRef,omitempty"`
}

// ---- Edge cases ----

// EmptyMarkerSpec tests that a marker with no value is treated as missing.
type EmptyMarkerSpec struct {
	// +orc:kustomize:ref
	ProjectRef *KubernetesNameRef `json:"projectRef,omitempty"` // want `field EmptyMarkerSpec.ProjectRef references another object by KubernetesNameRef but has no \+orc:kustomize:ref=<Kind> marker`
}

// UnrelatedFieldSpec has non-KubernetesNameRef fields which are never flagged.
type UnrelatedFieldSpec struct {
	ProjectID *string `json:"projectID,omitempty"`
}
