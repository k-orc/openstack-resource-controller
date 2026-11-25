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

// SwiftContainerResourceSpec contains the desired state of a Swift container.
// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="SwiftContainerResourceSpec is immutable"
type SwiftContainerResourceSpec struct {
	// name will be the name of the created resource. If not specified, the
	// name of the ORC object will be used.
	// +kubebuilder:validation:XValidation:rule="!self.contains('/')",message="Swift container names cannot contain '/'"
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	// metadata contains the X-Container-Meta-* properties of the container.
	// +kubebuilder:validation:MaxItems=90
	// +listType=map
	// +listMapKey=name
	// +optional
	Metadata []SwiftContainerMetadata `json:"metadata,omitempty"`

	// readACL is the X-Container-Read access control list.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	ReadACL string `json:"readACL,omitempty"`

	// writeACL is the X-Container-Write access control list.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	WriteACL string `json:"writeACL,omitempty"`

	// storagePolicy is the name of the Swift storage policy for the container.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=255
	// +optional
	StoragePolicy string `json:"storagePolicy,omitempty"`
}

// SwiftContainerFilter defines an existing resource by its properties
// +kubebuilder:validation:MinProperties:=1
type SwiftContainerFilter struct {
	// name is the exact name of the Swift container.
	// +kubebuilder:validation:XValidation:rule="!self.contains('/')",message="Swift container names cannot contain '/'"
	// +optional
	Name *OpenStackName `json:"name,omitempty"`
}

// SwiftContainerMetadata is a container metadata property.
type SwiftContainerMetadata struct {
	// name is the name of the metadata property, without the
	// X-Container-Meta- prefix.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=128
	// +required
	Name string `json:"name,omitempty"`

	// value is the value of the metadata property.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=256
	// +required
	Value string `json:"value,omitempty"`
}

// SwiftContainerMetadataStatus is an observed container metadata property.
type SwiftContainerMetadataStatus struct {
	// name is the name of the metadata property.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Name string `json:"name,omitempty"`

	// value is the value of the metadata property.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Value string `json:"value,omitempty"`
}

// SwiftContainerResourceStatus represents the observed state of the resource.
type SwiftContainerResourceStatus struct {
	// name is the name of the container in Swift.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Name string `json:"name,omitempty"`

	// metadata contains the X-Container-Meta-* properties of the container.
	// TempURL keys are not reported.
	// +kubebuilder:validation:MaxItems=128
	// +listType=atomic
	// +optional
	Metadata []SwiftContainerMetadataStatus `json:"metadata,omitempty"`

	// readACL is the X-Container-Read access control list.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	ReadACL string `json:"readACL,omitempty"`

	// writeACL is the X-Container-Write access control list.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	WriteACL string `json:"writeACL,omitempty"`

	// storagePolicy is the name of the Swift storage policy for the container.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	StoragePolicy string `json:"storagePolicy,omitempty"`

	// bytesUsed is the total number of bytes stored in the container.
	// +optional
	BytesUsed *int64 `json:"bytesUsed,omitempty"`

	// objectCount is the number of objects stored in the container.
	// +optional
	ObjectCount *int64 `json:"objectCount,omitempty"`
}
