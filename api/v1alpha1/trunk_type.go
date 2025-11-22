/*
Copyright 2025 The ORC Authors.

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

// TrunkFilter specifies a filter to select a trunk. At least one parameter must be specified.
// +kubebuilder:validation:MinProperties:=1
type TrunkFilter struct {
	// name of the existing resource
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	// description of the existing resource
	// +optional
	Description *NeutronDescription `json:"description,omitempty"`

	// portRef is a reference to the ORC Port which this trunk is associated with.
	// +optional
	PortRef *KubernetesNameRef `json:"portRef,omitempty"`

	// projectRef is a reference to the ORC Project this resource is associated with.
	// Typically, only used by admin.
	// +optional
	ProjectRef *KubernetesNameRef `json:"projectRef,omitempty"`

	FilterByNeutronTags `json:",inline"`
}

// Subport represents a subport that will be attached to a trunk.
type Subport struct {
	// portRef is a reference to the ORC Port which will be used as a subport.
	// +required
	PortRef KubernetesNameRef `json:"portRef"`

	// segmentationType is the type of segmentation to use (e.g., "vlan").
	// +required
	// +kubebuilder:validation:MaxLength=64
	SegmentationType string `json:"segmentationType"`

	// segmentationID is the segmentation identifier (e.g., VLAN ID).
	// +required
	SegmentationID int32 `json:"segmentationID"`
}

// SubportStatus represents the observed state of a subport.
type SubportStatus struct {
	// portID is the ID of the port used as a subport.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	PortID string `json:"portID,omitempty"`

	// segmentationType is the type of segmentation used.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	SegmentationType string `json:"segmentationType,omitempty"`

	// segmentationID is the segmentation identifier.
	// +optional
	SegmentationID *int32 `json:"segmentationID,omitempty"`
}

// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="portRef is immutable"
type TrunkResourceSpec struct {
	// name is a human-readable name of the trunk. If not set, the object's name will be used.
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	// description is a human-readable description for the resource.
	// +optional
	Description *NeutronDescription `json:"description,omitempty"`

	// portRef is a reference to the ORC Port which will be used as the parent port for this trunk.
	// +required
	PortRef KubernetesNameRef `json:"portRef"`

	// tags is a list of tags which will be applied to the trunk.
	// +kubebuilder:validation:MaxItems:=64
	// +listType=set
	// +optional
	Tags []NeutronTag `json:"tags,omitempty"`

	// adminStateUp is the administrative state of the trunk, which is up (true) or down (false).
	// +optional
	AdminStateUp *bool `json:"adminStateUp,omitempty"`

	// subports are the subports that will be attached to this trunk.
	// +kubebuilder:validation:MaxItems:=128
	// +listType=atomic
	// +optional
	Subports []Subport `json:"subports,omitempty"`

	// projectRef is a reference to the ORC Project this resource is associated with.
	// Typically, only used by admin.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="projectRef is immutable"
	ProjectRef *KubernetesNameRef `json:"projectRef,omitempty"`
}

type TrunkResourceStatus struct {
	// name is the human-readable name of the resource. Might not be unique.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Name string `json:"name,omitempty"`

	// description is a human-readable description for the resource.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Description string `json:"description,omitempty"`

	// portID is the ID of the parent port.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	PortID string `json:"portID,omitempty"`

	// projectID is the project owner of the resource.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	ProjectID string `json:"projectID,omitempty"`

	// status indicates the current status of the resource.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Status string `json:"status,omitempty"`

	// tags is the list of tags on the resource.
	// +kubebuilder:validation:MaxItems=64
	// +kubebuilder:validation:items:MaxLength=1024
	// +listType=atomic
	// +optional
	Tags []string `json:"tags,omitempty"`

	// adminStateUp is the administrative state of the trunk,
	// which is up (true) or down (false).
	// +optional
	AdminStateUp *bool `json:"adminStateUp,omitempty"`

	// subports is a list of subports attached to this trunk.
	// +kubebuilder:validation:MaxItems=128
	// +listType=atomic
	// +optional
	Subports []SubportStatus `json:"subports,omitempty"`

	NeutronStatusMetadata `json:",inline"`
}

