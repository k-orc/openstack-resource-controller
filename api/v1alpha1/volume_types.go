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

// VolumeResourceSpec contains the desired state of a volume
// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="VolumeResourceSpec is immutable"
type VolumeResourceSpec struct {
	// name will be the name of the created resource. If not specified, the
	// name of the ORC object will be used.
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	// description contains a free form description of the volume.
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=65535
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
	// +kubebuilder:validation:MaxLength:=65535
	// +optional
	Description string `json:"description,omitempty"`

	// size is the size of the volume in GiB.
	// +optional
	Size *int32 `json:"size,omitempty"`
}
