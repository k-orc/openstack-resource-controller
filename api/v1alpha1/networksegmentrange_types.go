/*
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

// NetworkSegmentRangeResourceSpec contains the desired state of a network segment range
type NetworkSegmentRangeResourceSpec struct {
	// name will be the name of the created resource. If not specified, the
	// name of the ORC object will be used.
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	// networkType is the type of physical network. Valid values are vlan, vxlan, gre, geneve.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=vlan;vxlan;gre;geneve
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="networkType is immutable"
	NetworkType string `json:"networkType"`

	// physicalNetwork is the physical network where this network segment range is implemented.
	// Required for VLAN network type.
	// +optional
	// +kubebuilder:validation:MaxLength=1024
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="physicalNetwork is immutable"
	PhysicalNetwork *string `json:"physicalNetwork,omitempty"`

	// minimum is the minimum segmentation ID for this range.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="minimum is immutable"
	Minimum int `json:"minimum"`

	// maximum is the maximum segmentation ID for this range.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="maximum is immutable"
	Maximum int `json:"maximum"`

	// shared indicates whether this resource is shared across all projects.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="shared is immutable"
	Shared *bool `json:"shared,omitempty"`

	// projectRef is a reference to the ORC Project this resource is associated with.
	// Only valid when shared is false.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="projectRef is immutable"
	ProjectRef *KubernetesNameRef `json:"projectRef,omitempty"`
}

// NetworkSegmentRangeFilter defines an existing resource by its properties
// +kubebuilder:validation:MinProperties:=1
type NetworkSegmentRangeFilter struct {
	// name of the existing resource
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	// networkType is the type of physical network. Valid values are vlan, vxlan, gre, geneve.
	// +optional
	// +kubebuilder:validation:Enum=vlan;vxlan;gre;geneve
	NetworkType *string `json:"networkType,omitempty"`

	// physicalNetwork is the physical network where this network segment range is implemented.
	// +optional
	PhysicalNetwork *string `json:"physicalNetwork,omitempty"`

	// shared indicates whether this resource is shared across all projects.
	// +optional
	Shared *bool `json:"shared,omitempty"`

	// projectRef is a reference to the ORC Project this resource is associated with.
	// +optional
	ProjectRef *KubernetesNameRef `json:"projectRef,omitempty"`
}

// NetworkSegmentRangeResourceStatus represents the observed state of the resource.
type NetworkSegmentRangeResourceStatus struct {
	// name is a Human-readable name for the network segment range.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Name string `json:"name,omitempty"`

	// networkType is the type of physical network.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	NetworkType string `json:"networkType,omitempty"`

	// physicalNetwork is the physical network where this network segment range is implemented.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	PhysicalNetwork string `json:"physicalNetwork,omitempty"`

	// minimum is the minimum segmentation ID for this range.
	// +optional
	Minimum *int `json:"minimum,omitempty"`

	// maximum is the maximum segmentation ID for this range.
	// +optional
	Maximum *int `json:"maximum,omitempty"`

	// shared indicates whether this resource is shared across all projects.
	// +optional
	Shared *bool `json:"shared,omitempty"`

	// default indicates whether this is a default range from configuration.
	// +optional
	Default *bool `json:"default,omitempty"`

	// projectID is the project owner of the network segment range.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	ProjectID string `json:"projectID,omitempty"`
}
