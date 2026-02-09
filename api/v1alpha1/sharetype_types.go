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

// ShareTypeResourceSpec contains the desired state of the resource.
type ShareTypeResourceSpec struct {
	// name will be the name of the created resource. If not specified, the
	// name of the ORC object will be used.
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	// isPublic indicates whether a share type is publicly accessible
	// +optional
	IsPublic *bool `json:"isPublic,omitempty"`

	// extraSpecs defines the extra specifications for the share type.
	// +required
	ExtraSpecs ShareTypeExtraSpecRequired `json:"extraSpecs,omitempty"`
}

// ShareTypeFilter defines an existing resource by its properties
// +kubebuilder:validation:MinProperties:=1
type ShareTypeFilter struct {
	// name of the existing resource
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	// isPublic indicated whether the ShareType is public.
	// +optional
	IsPublic *bool `json:"isPublic,omitempty"`
}

type ShareTypeExtraSpecRequired struct {
	// driver_handles_shares_servers is REQUIRED.
	// It defines the driver mode for share server lifecycle management.
	// +required
	DriverHandlesShareServers bool `json:"driverHandlesShareServers"`

	// Any other key-value pairs can be added here
	// +optional
	// +listType=atomic
	// +kubebuilder:validation:MaxItems:=64
	OtherSpecs []ShareTypeExtraSpec `json:"otherSpecs,omitempty"`
}

// ShareTypeExtraSpec is a generic key-value pair for additional specs.
type ShareTypeExtraSpec struct {
	// name is the key of the extra spec.
	// +kubebuilder:validation:MaxLength:=255
	// +required
	Name string `json:"name"`

	// value is the value of the extra spec.
	// +kubebuilder:validation:MaxLength:=255
	// +required
	Value string `json:"value"`
}

// ShareTypeResourceStatus represents the observed state of the resource.
type ShareTypeResourceStatus struct {
	// name is a Human-readable name for the resource. Might not be unique.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Name string `json:"name,omitempty"`

	// description is a human-readable description for the resource.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Description string `json:"description,omitempty"`

	// extraSpecs is a map of key-value pairs that define extra specifications for the share type.
	// +kubebuilder:validation:MaxItems:=64
	// +listType=atomic
	// +optional
	ExtraSpecs []ShareTypeExtraSpecStatus `json:"extraSpecs"`

	// isPublic indicates whether the ShareType is public.
	// +optional
	IsPublic *bool `json:"isPublic"`
}

type ShareTypeExtraSpecStatus struct {
	// name is the name of the extraspec
	// +kubebuilder:validation:MaxLength:=255
	// +optional
	Name string `json:"name,omitempty"`

	// value is the value of the extraspec
	// +kubebuilder:validation:MaxLength:=255
	// +optional
	Value string `json:"value,omitempty"`
}
