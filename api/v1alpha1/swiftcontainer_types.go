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

// SwiftContainerName is the name of a Swift container. It must be between 1
// and 256 characters long and must not contain forward slashes.
// +kubebuilder:validation:MinLength:=1
// +kubebuilder:validation:MaxLength:=256
// +kubebuilder:validation:XValidation:rule="!self.contains('/')",message="name must not contain forward slashes"
// +kubebuilder:validation:XValidation:rule="self.size() <= 256",message="name must not exceed 256 UTF-8 bytes"
type SwiftContainerName string

// SwiftContainerMetadata defines a key-value pair to be set as a Swift
// container metadata header (X-Container-Meta-<key>: <value>).
type SwiftContainerMetadata struct {
	// key is the key of the metadata item. It will be used as the suffix of
	// the X-Container-Meta-* header.
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=255
	// +required
	Key string `json:"key,omitempty"`

	// value is the value of the metadata item.
	// +kubebuilder:validation:MaxLength:=255
	// +required
	Value string `json:"value"`
}

// SwiftContainerMetadataStatus represents an observed metadata key-value pair
// on a Swift container.
type SwiftContainerMetadataStatus struct {
	// key is the key of the metadata item.
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=255
	// +required
	Key string `json:"key,omitempty"`

	// value is the value of the metadata item.
	// +kubebuilder:validation:MaxLength:=255
	// +optional
	Value string `json:"value,omitempty"`
}

// SwiftContainerFilter defines an existing resource query.
// +kubebuilder:validation:MinProperties:=1
type SwiftContainerFilter struct {
	// prefix filters containers by name prefix. Only containers whose names
	// begin with this prefix will be considered.
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=256
	// +optional
	Prefix *string `json:"prefix,omitempty"`
}

// SwiftContainerImport specifies an existing resource which will be imported
// instead of creating a new one. Swift containers are identified by name
// rather than UUID.
// +kubebuilder:validation:MinProperties:=1
// +kubebuilder:validation:MaxProperties:=1
type SwiftContainerImport struct {
	// name contains the name of an existing Swift container to import. Note
	// that when specifying an import by name, the resource MUST already exist.
	// The ORC object will enter an error state if the resource does not exist.
	// +optional
	Name *SwiftContainerName `json:"name,omitempty"`

	// filter contains a resource query which is expected to return a single
	// result. The controller will continue to retry if filter returns no
	// results. If filter returns multiple results the controller will set an
	// error state and will not continue to retry.
	// +optional
	Filter *SwiftContainerFilter `json:"filter,omitempty"`
}

// SwiftContainerResourceSpec contains the desired state of a Swift container.
type SwiftContainerResourceSpec struct {
	// name will be the name of the created Swift container. If not specified,
	// the name of the ORC object will be used. The name must be unique within
	// the account and must not contain forward slashes.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="name is immutable"
	Name *SwiftContainerName `json:"name,omitempty"`

	// metadata is a list of key-value pairs which will be set as
	// X-Container-Meta-* headers on the Swift container.
	// +kubebuilder:validation:MaxItems:=64
	// +listType=map
	// +listMapKey=key
	// +optional
	Metadata []SwiftContainerMetadata `json:"metadata,omitempty"`

	// containerRead sets the X-Container-Read ACL header which defines who
	// can read objects in the container. Common values include ".r:*" for
	// public read access or a comma-separated list of account/container
	// combinations.
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=256
	// +optional
	ContainerRead string `json:"containerRead,omitempty"`

	// containerWrite sets the X-Container-Write ACL header which defines who
	// can write objects to the container. Common values include a
	// comma-separated list of account/container combinations.
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=256
	// +optional
	ContainerWrite string `json:"containerWrite,omitempty"`

	// storagePolicy is the name of the storage policy to use for this
	// container. If not specified, the cluster's default storage policy will
	// be used. This field is immutable after creation.
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=255
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="storagePolicy is immutable"
	StoragePolicy string `json:"storagePolicy,omitempty"`
}

// SwiftContainerResourceStatus represents the observed state of a Swift container.
type SwiftContainerResourceStatus struct {
	// name is the name of the Swift container.
	// +kubebuilder:validation:MaxLength=256
	// +optional
	Name string `json:"name,omitempty"`

	// bytesUsed is the total number of bytes stored in the container.
	// +optional
	BytesUsed int64 `json:"bytesUsed,omitempty"`

	// objectCount is the number of objects stored in the container.
	// +optional
	ObjectCount int64 `json:"objectCount,omitempty"`

	// metadata is the list of observed metadata key-value pairs on the container.
	// +kubebuilder:validation:MaxItems:=64
	// +listType=map
	// +listMapKey=key
	// +optional
	Metadata []SwiftContainerMetadataStatus `json:"metadata,omitempty"`

	// containerRead is the current X-Container-Read ACL, defining who can
	// read objects in the container.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	ContainerRead string `json:"containerRead,omitempty"`

	// containerWrite is the current X-Container-Write ACL, defining who can
	// write objects to the container.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	ContainerWrite string `json:"containerWrite,omitempty"`

	// storagePolicy is the name of the storage policy assigned to the container.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	StoragePolicy string `json:"storagePolicy,omitempty"`

	// versions is the container where object versions are stored, if versioning
	// is enabled on this container.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Versions string `json:"versions,omitempty"`
}
