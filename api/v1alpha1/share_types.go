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

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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

	// size is the size of the share, in gibibytes (GiB).
	// +kubebuilder:validation:Minimum=1
	// +required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="size is immutable"
	Size int32 `json:"size,omitempty"`

	// shareProto is the file system protocol for the share.
	// Valid values are NFS, CIFS, GlusterFS, HDFS, CephFS, or MAPRFS.
	// +kubebuilder:validation:Enum=NFS;CIFS;GlusterFS;HDFS;CephFS;MAPRFS
	// +required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="shareProto is immutable"
	ShareProto string `json:"shareProto,omitempty"`

	// availabilityZone is the availability zone in which to create the share.
	// +kubebuilder:validation:MaxLength:=255
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="availabilityZone is immutable"
	AvailabilityZone string `json:"availabilityZone,omitempty"`

	// metadata key and value pairs to be associated with the share.
	// +kubebuilder:validation:MaxItems:=64
	// +listType=atomic
	// +optional
	Metadata []ShareMetadata `json:"metadata,omitempty"`

	// isPublic defines whether the share is publicly visible.
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

	// shareProto is the file system protocol to filter by
	// +kubebuilder:validation:Enum=NFS;CIFS;GlusterFS;HDFS;CephFS;MAPRFS
	// +optional
	ShareProto *string `json:"shareProto,omitempty"`

	// status is the share status to filter by
	// +kubebuilder:validation:Enum=creating;available;deleting;error;error_deleting;manage_starting;manage_error;unmanage_starting;unmanage_error;extending;extending_error;shrinking;shrinking_error
	// +optional
	Status *string `json:"status,omitempty"`

	// isPublic filters by public visibility
	// +optional
	IsPublic *bool `json:"isPublic,omitempty"`
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

	// size is the size of the share in GiB.
	// +optional
	Size *int32 `json:"size,omitempty"`

	// shareProto is the file system protocol.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	ShareProto string `json:"shareProto,omitempty"`

	// status represents the current status of the share.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Status string `json:"status,omitempty"`

	// availabilityZone is which availability zone the share is in.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	AvailabilityZone string `json:"availabilityZone,omitempty"`

	// exportLocations contains paths for accessing the share.
	// +kubebuilder:validation:MaxItems:=32
	// +listType=atomic
	// +optional
	ExportLocations []ShareExportLocation `json:"exportLocations,omitempty"`

	// metadata key and value pairs associated with the share.
	// +kubebuilder:validation:MaxItems:=64
	// +listType=atomic
	// +optional
	Metadata []ShareMetadataStatus `json:"metadata,omitempty"`

	// isPublic indicates whether the share is publicly visible.
	// +optional
	IsPublic *bool `json:"isPublic,omitempty"`

	// createdAt shows the date and time when the resource was created. The date and time stamp format is ISO 8601.
	// +optional
	CreatedAt *metav1.Time `json:"createdAt,omitempty"`

	// projectID is the ID of the project that owns the share.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	ProjectID string `json:"projectID,omitempty"`
}

type ShareMetadata struct {
	// name is the name of the metadata
	// +kubebuilder:validation:MaxLength:=255
	// +required
	Name string `json:"name"`

	// value is the value of the metadata
	// +kubebuilder:validation:MaxLength:=255
	// +required
	Value string `json:"value"`
}

type ShareMetadataStatus struct {
	// name is the name of the metadata
	// +kubebuilder:validation:MaxLength:=255
	// +optional
	Name string `json:"name,omitempty"`

	// value is the value of the metadata
	// +kubebuilder:validation:MaxLength:=255
	// +optional
	Value string `json:"value,omitempty"`
}

type ShareExportLocation struct {
	// path is the export path for accessing the share
	// +kubebuilder:validation:MaxLength:=1024
	// +optional
	Path string `json:"path,omitempty"`

	// preferred indicates if this is the preferred export location
	// +optional
	Preferred *bool `json:"preferred,omitempty"`
}
