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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OpenStackFlavorSpec defines the desired state of OpenStackFlavor
type OpenStackFlavorSpec struct {
	// Cloud is the OpenStackCloud hosting this resource
	Cloud string `json:"cloud"`

	// ID is the OpenStack UUID of the resource. If left empty, the
	// controller will create a new resource and populate this field. If
	// manually populated, the controller will adopt the corresponding
	// resource.
	ID string `json:"id,omitempty"`

	// Name is the name of the flavor.
	Name string `json:"name,omitempty"`

	// RAM is the memory of the flavor, measured in MB.
	RAM int `json:"ram,omitempty"`

	// VCPUs is the number of vcpus for the flavor.
	VCPUs int `json:"vcpus,omitempty"`

	// Disk the amount of root disk space, measured in GB.
	Disk *int `json:"disk,omitempty"`

	// Swap is the amount of swap space for the flavor, measured in MB.
	Swap *int `json:"swap,omitempty"`

	// RxTxFactor alters the network bandwidth of a flavor.
	RxTxFactor string `json:"rxtxFactor,omitempty"`

	// IsPublic flags a flavor as being available to all projects or not.
	IsPublic *bool `json:"isPublic,omitempty"`

	// Ephemeral is the amount of ephemeral disk space, measured in GB.
	Ephemeral *int `json:"ephemeral,omitempty"`

	// Description is a free form description of the flavor. Limited to
	// 65535 characters in length. Only printable characters are allowed.
	// New in version 2.55
	Description string `json:"description,omitempty"`

	// Unmanaged, when true, means that no action will be performed in
	// OpenStack against this resource. This is false by default, except
	// for pre-existing resources that are adopted by passing ID on
	// creation.
	Unmanaged *bool `json:"unmanaged,omitempty"`
}

// OpenStackFlavorStatus defines the observed state of OpenStackFlavor
type OpenStackFlavorStatus struct {
	// ID is the flavor's unique ID.
	ID string `json:"id,omitempty"`

	// Disk is the amount of root disk, measured in GB.
	Disk int `json:"disk,omitempty"`

	// RAM is the amount of memory, measured in MB.
	RAM int `json:"ram,omitempty"`

	// Name is the name of the flavor.
	Name string `json:"name,omitempty"`

	// RxTxFactor describes bandwidth alterations of the flavor.
	RxTxFactor string `json:"rxtxFactor,omitempty"`

	// Swap is the amount of swap space, measured in MB.
	Swap int `json:"swap,omitempty"`

	// VCPUs indicates how many (virtual) CPUs are available for this flavor.
	VCPUs int `json:"vcpus,omitempty"`

	// IsPublic indicates whether the flavor is public.
	IsPublic bool `json:"isPublic,omitempty"`

	// Ephemeral is the amount of ephemeral disk space, measured in GB.
	Ephemeral int `json:"ephemeral,omitempty"`

	// Description is a free form description of the flavor. Limited to
	// 65535 characters in length. Only printable characters are allowed.
	// New in version 2.55
	Description string `json:"description,omitempty"`

	// Properties is a dictionary of the flavor’s extra-specs key-and-value
	// pairs. This will only be included if the user is allowed by policy to
	// index flavor extra_specs
	// New in version 2.61
	ExtraSpecs map[string]string `json:"extraSpecs,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// OpenStackFlavor is the Schema for the openstackflavors API
type OpenStackFlavor struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OpenStackFlavorSpec   `json:"spec,omitempty"`
	Status OpenStackFlavorStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OpenStackFlavorList contains a list of OpenStackFlavor
type OpenStackFlavorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OpenStackFlavor `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OpenStackFlavor{}, &OpenStackFlavorList{})
}
