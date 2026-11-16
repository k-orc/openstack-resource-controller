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

package v1alpha1

// DNSZoneShareResourceSpec contains the desired state of the resource: a Designate zone share,
// which lets another project manage the zone's recordsets while the zone's project keeps owning it.
type DNSZoneShareResourceSpec struct {
	// zoneRef is a reference to the ORC DNSZone to share.
	// +required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="zoneRef is immutable"
	// +orc:kustomize:ref=DNSZone
	ZoneRef KubernetesNameRef `json:"zoneRef,omitempty"`

	// targetProjectID is the ID of the project to share the zone with.
	// +required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="targetProjectID is immutable"
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=64
	TargetProjectID string `json:"targetProjectID,omitempty"`
}

// DNSZoneShareFilter defines an existing resource by its properties
// +kubebuilder:validation:MinProperties:=1
type DNSZoneShareFilter struct {
	// zoneRef is a reference to the ORC DNSZone the share belongs to. Designate lists shares per
	// zone only.
	// +required
	// +orc:kustomize:ref=DNSZone
	ZoneRef KubernetesNameRef `json:"zoneRef,omitempty"`

	// targetProjectID of the existing resource
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=64
	// +optional
	TargetProjectID *string `json:"targetProjectID,omitempty"`
}

// DNSZoneShareResourceStatus represents the observed state of the resource.
type DNSZoneShareResourceStatus struct {
	// zoneID is the ID of the DNSZone this share applies to.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	ZoneID string `json:"zoneID,omitempty"`

	// targetProjectID is the OpenStack project ID this share grants access to.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	TargetProjectID string `json:"targetProjectID,omitempty"`

	// projectID is the ID of the project that owns the zone.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	ProjectID string `json:"projectID,omitempty"`
}
