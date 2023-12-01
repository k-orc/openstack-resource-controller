/*
Copyright 2023.

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

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OpenStackImageSpec defines the desired state of OpenStackImage
type OpenStackImageResourceSpec struct {
	// ContainerFormat is the format of the
	// container. Valid values are ami, ari, aki, bare, and ovf.
	ContainerFormat string `json:"containerFormat,omitempty"`

	// DiskFormat is the format of the disk. If set,
	// valid values are ami, ari, aki, vhd, vmdk, raw, qcow2, vdi,
	// and iso.
	DiskFormat string `json:"diskFormat,omitempty"`

	// ID is the OpenStack UUID of the resource. If left empty, the
	// controller will create a new resource and populate this field. If
	// manually populated, the controller will adopt the corresponding
	// resource.
	ID string `json:"id,omitempty"`

	// MinDisk is the amount of disk space in GB that is required to boot
	// the image.
	MinDisk int `json:"minDisk,omitempty"`

	// MinRAM is the amount of RAM in MB that is required to boot the
	// image.
	MinRAM int `json:"minRam,omitempty"`

	// Name of the OpenStack resource.
	Name string `json:"name,omitempty"`

	// protected is whether the image is not deletable.
	Protected *bool `json:"protected,omitempty"`

	// Tags is a set of image tags.
	// Each tag is a string of at most 255 chars.
	Tags []string `json:"tags,omitempty"`

	// Visibility defines who can see/use the image.
	// +kubebuilder:validation:Enum:="public";"private";"shared";"community"
	Visibility *string `json:"visibility,omitempty"`
}

// OpenStackImageStatus defines the observed state of OpenStackImage
type OpenStackImageResourceStatus struct {
	// ID is the image UUID.
	ID string `json:"id"`

	// Name is the human-readable display name for the image.
	Name string `json:"name"`

	// Status is the image status. It can be "queued" or "active"
	// See imageservice/v2/images/type.go
	Status string `json:"status,omitempty"`

	// Tags is a list of image tags. Tags are arbitrarily defined strings
	// attached to an image.
	Tags []string `json:"tags,omitempty"`

	// ContainerFormat is the format of the container.
	// Valid values are ami, ari, aki, bare, and ovf.
	ContainerFormat string `json:"containerFormat,omitempty"`

	// DiskFormat is the format of the disk.
	// If set, valid values are ami, ari, aki, vhd, vmdk, raw, qcow2, vdi,
	// and iso.
	DiskFormat string `json:"diskFormat,omitempty"`

	// MinDisk is the amount of disk space in GB that is required to boot
	// the image.
	MinDisk int `json:"minDisk,omitempty"`

	// MinRAM is the amount of RAM in MB that is required to boot the
	// image.
	MinRAM int `json:"minRam,omitempty"`

	// Owner is the tenant ID the image belongs to.
	Owner string `json:"owner,omitempty"`

	// Protected is whether the image is deletable or not.
	Protected bool `json:"protected,omitempty"`

	// Visibility defines who can see/use the image.
	Visibility string `json:"visibility,omitempty"`

	// Hidden is whether the image is listed in default image list or not.
	Hidden bool `json:"hidden,omitempty"`

	// Checksum is the checksum of the data that's associated with the
	// image.
	Checksum string `json:"checksum,omitempty"`

	// Size is the size in bytes of the data that's associated with the
	// image.
	Size int64 `json:"size,omitempty"`

	// Metadata is a set of metadata associated with the image.
	// Image metadata allow for meaningfully define the image properties
	// and tags.
	// See http://docs.openstack.org/developer/glance/metadefs-concepts.html.
	Metadata map[string]string `json:"metadata,omitempty"`

	// Properties is a set of key-value pairs, if any, that are associated with
	// the image.
	Properties map[string]string `json:"properties,omitempty"`

	// UpdatedAt contains the timestamp of when the resource was last
	// changed.
	UpdatedAt string `json:"updatedAt,omitempty"`

	// CreatedAt contains the timestamp of when the resource was created.
	CreatedAt string `json:"createdAt,omitempty"`

	// File is the trailing path after the glance endpoint that represent the
	// location of the image or the path to retrieve it.
	File string `json:"file,omitempty"`

	// Schema is the path to the JSON-schema that represent the image or image
	// entity.
	Schema string `json:"schema,omitempty"`

	// VirtualSize is the virtual size of the image
	VirtualSize int64 `json:"virtualSize,omitempty"`

	// OpenStackImageImportMethods is a slice listing the types of import
	// methods available in the cloud.
	ImportMethods []string `json:"importMethods,omitempty"`

	// StoreIDs is a slice listing the store IDs available in the cloud.
	StoreIDs []string `json:"storeIDs,omitempty"`
}

// OpenStackImageSpec defines the desired state of OpenStackImage
type OpenStackImageSpec struct {
	CommonSpec `json:",inline"`

	// ID is the UUID of the existing OpenStack resource to be adopted. If
	// left empty, the controller will create a new resource using the
	// information in the "resource" stanza.
	ID string `json:"id,omitempty"`

	Resource *OpenStackImageResourceSpec `json:"resource,omitempty"`
}

// OpenStackImageStatus defines the observed state of OpenStackImage
type OpenStackImageStatus struct {
	CommonStatus `json:",inline"`

	Resource OpenStackImageResourceStatus `json:"resource,omitempty"`
}

// Implement OpenStackResourceCommonStatus interface
func (c *OpenStackImage) OpenStackCommonStatus() *CommonStatus {
	return &c.Status.CommonStatus
}

var _ OpenStackResourceCommonStatus = &OpenStackImage{}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
//+kubebuilder:printcolumn:name="Error",type=string,JSONPath=`.status.conditions[?(@.type=="Error")].status`
//+kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].message`

// OpenStackImage is the Schema for the openstackimages API
type OpenStackImage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OpenStackImageSpec   `json:"spec,omitempty"`
	Status OpenStackImageStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OpenStackImageList contains a list of OpenStackImage
type OpenStackImageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OpenStackImage `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OpenStackImage{}, &OpenStackImageList{})
}
