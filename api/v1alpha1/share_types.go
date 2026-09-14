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

// ShareResourceSpec contains the desired state of the resource.
type ShareResourceSpec struct {
	// name will be the name of the created resource. If not specified, the
	// name of the ORC object will be used.
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	// description is a human-readable description for the resource.
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=255
	// +optional
	Description *string `json:"description,omitempty"`

	// shareProto is the shared file system protocol (e.g., NFS, CIFS).
	// +required
	// +kubebuilder:validation:MaxLength=255
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="shareProto is immutable"
	ShareProto string `json:"shareProto"`

	// size is the size of the share in GB.
	// +required
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="size is immutable"
	Size int32 `json:"size,omitempty"`

	// shareNetworkRef is a reference to the ORC ShareNetwork which this resource is associated with.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="shareNetworkRef is immutable"
	ShareNetworkRef *KubernetesNameRef `json:"shareNetworkRef,omitempty"`

	// availabilityZone is the availability zone in which to create the share.
	// +optional
	// +kubebuilder:validation:MaxLength=255
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="availabilityZone is immutable"
	AvailabilityZone *string `json:"availabilityZone,omitempty"`

	// shareType is the share type to use. If not specified, the default share type is used.
	// +optional
	// +kubebuilder:validation:MaxLength=255
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="shareType is immutable"
	ShareType *string `json:"shareType,omitempty"`

	// metadata contains key-value pairs of metadata for the share.
	// +optional
	Metadata map[string]string `json:"metadata,omitempty"`

	// isPublic determines whether the share is public.
	// +optional
	IsPublic *bool `json:"isPublic,omitempty"`
}

// ShareFilter defines an existing resource by its properties
// +kubebuilder:validation:MinProperties:=1
type ShareFilter struct {
	// name of the existing resource
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	// description of the existing resource
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=255
	// +optional
	Description *string `json:"description,omitempty"`

	// status filters by share status
	// +optional
	// +kubebuilder:validation:MaxLength=255
	Status *string `json:"status,omitempty"`

	// shareProto filters by share protocol
	// +optional
	// +kubebuilder:validation:MaxLength=255
	ShareProto *string `json:"shareProto,omitempty"`
}

// ShareResourceStatus represents the observed state of the resource.
type ShareResourceStatus struct {
	// name is a Human-readable name for the resource. Might not be unique.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Name string `json:"name,omitempty"`

	// description is a human-readable description for the resource.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Description string `json:"description,omitempty"`

	// status is the current status of the share.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Status string `json:"status,omitempty"`

	// shareProto is the shared file system protocol.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	ShareProto string `json:"shareProto,omitempty"`

	// size is the size of the share in GB.
	// +optional
	Size *int32 `json:"size,omitempty"`

	// availabilityZone is the availability zone of the share.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	AvailabilityZone string `json:"availabilityZone,omitempty"`

	// shareType is the UUID of the share type.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	ShareType string `json:"shareType,omitempty"`

	// shareTypeName is the name of the share type.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	ShareTypeName string `json:"shareTypeName,omitempty"`

	// shareNetworkID is the ID of the ShareNetwork to which the resource is associated.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	ShareNetworkID string `json:"shareNetworkID,omitempty"`

	// isPublic indicates the visibility of the share.
	// +optional
	IsPublic *bool `json:"isPublic,omitempty"`

	// metadata contains key-value pairs of custom metadata.
	// +optional
	Metadata map[string]string `json:"metadata,omitempty"`

	// exportLocations contains the export locations for mounting the share.
	// +listType=atomic
	// +kubebuilder:validation:MaxItems=100
	// +kubebuilder:validation:items:MaxLength=1024
	// +optional
	ExportLocations []string `json:"exportLocations,omitempty"`
}
