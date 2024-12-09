/*
Copyright 2024 The ORC Authors.

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

// NetworkResourceSpec contains the desired state of a network
type FlavorResourceSpec struct {
	// Name will be the name of the created resource. If not specified, the
	// name of the ORC object will be used.
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	// +optional
	Description *OpenStackDescription `json:"description,omitempty"`

	// RAM is the memory of the flavor, measured in MB.
	// +kubebuilder:validation:Minimum=1
	RAM int32 `json:"ram"`

	// Vcpus is the number of vcpus for the flavor.
	// +kubebuilder:validation:Minimum=1
	Vcpus int32 `json:"vcpus"`

	// Disk is the size of the root disk that will be created in GiB. If 0
	// the root disk will be set to exactly the size of the image used to
	// deploy the instance. However, in this case the scheduler cannot
	// select the compute host based on the virtual image size. Therefore,
	// 0 should only be used for volume booted instances or for testing
	// purposes. Volume-backed instances can be enforced for flavors with
	// zero root disk via the
	// os_compute_api:servers:create:zero_disk_flavor policy rule.
	Disk int32 `json:"disk,omitempty"`

	// Swap is the size of a dedicated swap disk that will be allocated, in
	// MiB. If 0 (the default), no dedicated swap disk will be created.
	Swap int32 `json:"swap,omitempty"`

	// IsPublic flags a flavor as being available to all projects or not.
	IsPublic *bool `json:"isPublic,omitempty"`

	// Ephemeral is the size of the ephemeral disk that will be created, in GiB.
	// Ephemeral disks may be written over on server state changes. So should only
	// be used as a scratch space for applications that are aware of its
	// limitations. Defaults to 0.
	Ephemeral int32 `json:"ephemeral,omitempty"`
}

// FlavorFilter defines an existing resource by its properties
// +kubebuilder:validation:MinProperties:=1
type FlavorFilter struct {
	// Name of the existing resource
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	// RAM is the memory of the flavor, measured in MB.
	// +optional
	RAM *int32 `json:"ram,omitempty"`

	// Disk is the size of the root disk in GiB.
	// +optional
	Disk *int32 `json:"disk,omitempty"`
}

// FlavorResourceStatus represents the observed state of the resource.
type FlavorResourceStatus struct {
	// Human-readable name for the flavor. Might not be unique.
	// +optional
	Name string `json:"name,omitempty"`

	// Description is a human-readable description for the resource.
	// +optional
	Description *string `json:"description,omitempty"`

	// RAM is the memory of the flavor, measured in MB.
	// +optional
	RAM *int32 `json:"ram,omitempty"`

	// Vcpus is the number of vcpus for the flavor.
	// +optional
	Vcpus *int32 `json:"vcpus,omitempty"`

	// Disk is the size of the root disk that will be created in GiB.
	// +optional
	Disk *int32 `json:"disk,omitempty"`

	// Swap is the size of a dedicated swap disk that will be allocated, in
	// MiB.
	// +optional
	Swap *int32 `json:"swap,omitempty"`

	// IsPublic flags a flavor as being available to all projects or not.
	// +optional
	IsPublic *bool `json:"isPublic,omitempty"`

	// Ephemeral is the size of the ephemeral disk, in GiB.
	// +optional
	Ephemeral *int32 `json:"ephemeral,omitempty"`
}
