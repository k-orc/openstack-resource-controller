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

// VolumeSnapshotResourceSpec contains the desired state of the resource.
type VolumeSnapshotResourceSpec struct {
	// name will be the name of the created resource. If not specified, the
	// name of the ORC object will be used.
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	// description is a human-readable description for the resource.
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=255
	// +optional
	Description *string `json:"description,omitempty"`

	// volumeRef is a reference to the ORC Volume to create a snapshot from.
	// +required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="volumeRef is immutable"
	VolumeRef KubernetesNameRef `json:"volumeRef,omitempty"`

	// force allows creating a snapshot even if the volume is attached.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="force is immutable"
	Force *bool `json:"force,omitempty"`

	// metadata key and value pairs to be associated with the snapshot.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="metadata is immutable"
	// +kubebuilder:validation:MaxItems:=64
	// +listType=atomic
	// +optional
	Metadata []VolumeSnapshotMetadata `json:"metadata,omitempty"`
}

// VolumeSnapshotFilter defines an existing resource by its properties
// +kubebuilder:validation:MinProperties:=1
type VolumeSnapshotFilter struct {
	// name of the existing resource
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	// description of the existing resource
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=255
	// +optional
	Description *string `json:"description,omitempty"`

	// status of the existing resource
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=255
	// +optional
	Status *string `json:"status,omitempty"`

	// volumeID is the ID of the volume the snapshot was created from
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=255
	// +optional
	VolumeID *string `json:"volumeID,omitempty"`
}

// VolumeSnapshotResourceStatus represents the observed state of the resource.
type VolumeSnapshotResourceStatus struct {
	// name is a human-readable name for the resource. Might not be unique.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Name string `json:"name,omitempty"`

	// description is a human-readable description for the resource.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Description string `json:"description,omitempty"`

	// status represents the current status of the snapshot.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Status string `json:"status,omitempty"`

	// size is the size of the snapshot in GiB.
	// +optional
	Size *int32 `json:"size,omitempty"`

	// volumeID is the ID of the volume the snapshot was created from.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	VolumeID string `json:"volumeID,omitempty"`

	// metadata key and value pairs associated with the snapshot.
	// +kubebuilder:validation:MaxItems:=64
	// +listType=atomic
	// +optional
	Metadata []VolumeSnapshotMetadataStatus `json:"metadata,omitempty"`

	// progress is the percentage of completion of the snapshot creation.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Progress string `json:"progress,omitempty"`

	// projectID is the ID of the project that owns the snapshot.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	ProjectID string `json:"projectID,omitempty"`

	// userID is the ID of the user who created the snapshot.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	UserID string `json:"userID,omitempty"`

	// groupSnapshotID is the ID of the group snapshot, if applicable.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	GroupSnapshotID string `json:"groupSnapshotID,omitempty"`

	// consumesQuota indicates whether the snapshot consumes quota.
	// +optional
	ConsumesQuota *bool `json:"consumesQuota,omitempty"`

	// createdAt shows the date and time when the resource was created. The date and time stamp format is ISO 8601.
	// +optional
	CreatedAt *metav1.Time `json:"createdAt,omitempty"`

	// updatedAt shows the date and time when the resource was updated. The date and time stamp format is ISO 8601.
	// +optional
	UpdatedAt *metav1.Time `json:"updatedAt,omitempty"`
}

type VolumeSnapshotMetadata struct {
	// name is the name of the metadata
	// +kubebuilder:validation:MaxLength:=255
	// +required
	Name string `json:"name"`

	// value is the value of the metadata
	// +kubebuilder:validation:MaxLength:=255
	// +required
	Value string `json:"value"`
}

type VolumeSnapshotMetadataStatus struct {
	// name is the name of the metadata
	// +kubebuilder:validation:MaxLength:=255
	// +optional
	Name string `json:"name,omitempty"`

	// value is the value of the metadata
	// +kubebuilder:validation:MaxLength:=255
	// +optional
	Value string `json:"value,omitempty"`
}
