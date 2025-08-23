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

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// VolumeResourceSpec contains the desired state of a volume
// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="VolumeResourceSpec is immutable"
type VolumeResourceSpec struct {
	// name will be the name of the created resource. If not specified, the
	// name of the ORC object will be used.
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	// description contains a free form description of the volume.
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=255
	// +optional
	Description *string `json:"description,omitempty"`

	// size is the size of the volume, in gibibytes (GiB).
	// +kubebuilder:validation:Minimum=1
	// +required
	Size int32 `json:"size"`
}

// VolumeFilter defines an existing resource by its properties
// +kubebuilder:validation:MinProperties:=1
type VolumeFilter struct {
	// name of the existing resource
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	// description contains a free form description of the volume.
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=255
	// +optional
	Description *string `json:"description,omitempty"`

	// size is the size of the volume in GiB.
	// +kubebuilder:validation:Minimum=1
	// +optional
	Size *int32 `json:"size,omitempty"`
}

// VolumeResourceStatus represents the observed state of the resource.
type VolumeResourceStatus struct {
	// name is a Human-readable name for the volume. Might not be unique.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Name string `json:"name,omitempty"`

	// description is a human-readable description for the resource.
	// +kubebuilder:validation:MaxLength:=255
	// +optional
	Description string `json:"description,omitempty"`

	// size is the size of the volume in GiB.
	// +optional
	Size *int32 `json:"size,omitempty"`

	// status represents the current status of the volume.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Status string `json:"status,omitempty"`

	// availabilityZone is which availability zone the volume is in.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	AvailabilityZone string `json:"availabilityZone,omitempty"`

	// Instances onto which the volume is attached.
	// Attachments []Attachment `json:"attachments"`

	// volumeType is the type of volume to create, either SATA or SSD.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	VolumeType string `json:"volumeType,omitempty"`

	// snapshotID is the ID of the snapshot from which the volume was created
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	SnapshotID string `json:"snapshotID,omitempty"`

	// sourceVolID is the ID of another block storage volume from which the current volume was created
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	SourceVolID string `json:"sourceVolID,omitempty"`

	// backupID is the ID of the backup from which the volume was restored
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	BackupID string `json:"backupID,omitempty"`

	// userID is the ID of the user who created the volume.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	UserID string `json:"userID,omitempty"`

	// bootable indicates whether this is a bootable volume.
	// +optional
	Bootable *bool `json:"bootable,omitempty"`

	// encrypted denotes if the volume is encrypted.
	// +optional
	Encrypted *bool `json:"encrypted,omitempty"`

	// replicationStatus is the status of replication.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	ReplicationStatus string `json:"replicationStatus,omitempty"`

	// consistencyGroupID is the consistency group ID.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	ConsistencyGroupID string `json:"consistencyGroupID,omitempty"`

	// multiattach denotes if the volume is multi-attach capable.
	// +optional
	Multiattach *bool `json:"multiattach,omitempty"`

	// host is the identifier of the host holding the volume.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Host string `json:"host,omitempty"`

	// tenantID is the ID of the project that owns the volume.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	TenantID string `json:"tenantID,omitempty"`

	// createdAt shows the date and time when the resource was created. The date and time stamp format is ISO 8601
	// +optional
	CreatedAt *metav1.Time `json:"createdAt,omitempty"`
	// updatedAt shows the date and time when the resource was updated. The date and time stamp format is ISO 8601
	// +optional
	UpdatedAt *metav1.Time `json:"updatedAt,omitempty"`
}
