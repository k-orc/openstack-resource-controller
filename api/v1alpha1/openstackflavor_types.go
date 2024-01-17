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

	"github.com/gofrs/uuid/v5"
)

// OpenStackFlavorResourceSpec defines the desired state of OpenStackFlavor
type OpenStackFlavorResourceSpec struct {
	// ID is the flavor's unique ID.
	ID string `json:"id,omitempty"`

	// Name is the name of the flavor.
	Name string `json:"name,omitempty"`

	// RAM is the memory of the flavor, measured in MB.
	RAM int `json:"ram,omitempty"`

	// VCPUs is the number of vcpus for the flavor.
	VCPUs int `json:"vcpus,omitempty"`

	// Disk is the size of the root disk that will be created in GiB. If 0
	// the root disk will be set to exactly the size of the image used to
	// deploy the instance. However, in this case the scheduler cannot
	// select the compute host based on the virtual image size. Therefore,
	// 0 should only be used for volume booted instances or for testing
	// purposes. Volume-backed instances can be enforced for flavors with
	// zero root disk via the
	// os_compute_api:servers:create:zero_disk_flavor policy rule.
	Disk int `json:"disk,omitempty"`

	// Swap is the size of a dedicated swap disk that will be allocated, in
	// MiB. If 0 (the default), no dedicated swap disk will be created.
	Swap int `json:"swap,omitempty"`

	// RxTxFactor is the receive / transmit factor (as a float) that will
	// be set on ports if the network backend supports the QOS extension.
	// Otherwise it will be ignored. It defaults to 1.0.
	RxTxFactor string `json:"rxtxFactor,omitempty"`

	// IsPublic flags a flavor as being available to all projects or not.
	IsPublic *bool `json:"isPublic,omitempty"`

	// Ephemeral is the size of the ephemeral disk that will be created, in GiB.
	// Ephemeral disks may be written over on server state changes. So should only
	// be used as a scratch space for applications that are aware of its
	// limitations. Defaults to 0.
	Ephemeral int `json:"ephemeral,omitempty"`

	// Description is a free form description of the flavor. Limited to
	// 65535 characters in length. Only printable characters are allowed.
	// New in version 2.55
	Description string `json:"description,omitempty"`
}

// OpenStackFlavorResourceStatus defines the observed state of OpenStackFlavor
type OpenStackFlavorResourceStatus struct {
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

type OpenStackFlavorSpec struct {
	CommonSpec `json:",inline"`

	// ID is the UUID of the existing OpenStack resource to be adopted. If
	// left empty, the controller will create a new resource using the
	// information in the "resource" stanza.
	ID string `json:"id,omitempty"`

	Resource *OpenStackFlavorResourceSpec `json:"resource,omitempty"`
}

type OpenStackFlavorStatus struct {
	CommonStatus `json:",inline"`

	Resource OpenStackFlavorResourceStatus `json:"resource,omitempty"`
}

// Implement OpenStackResourceCommonStatus interface
func (c *OpenStackFlavor) OpenStackCommonStatus() *CommonStatus {
	return &c.Status.CommonStatus
}

var _ OpenStackResourceCommonStatus = &OpenStackFlavor{}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
//+kubebuilder:printcolumn:name="Error",type=string,JSONPath=`.status.conditions[?(@.type=="Error")].status`
//+kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].message`
//+kubebuilder:printcolumn:name="OpenStackID",type=string,JSONPath=`.status.resource.id`

// OpenStackFlavor is the Schema for the openstackflavors API
type OpenStackFlavor struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OpenStackFlavorSpec   `json:"spec,omitempty"`
	Status OpenStackFlavorStatus `json:"status,omitempty"`
}

func (r *OpenStackFlavor) ComputedSpecID() string {
	if r.Spec.Resource.ID != "" {
		return r.Spec.Resource.ID
	}
	return uuid.NewV5(UuidNamespace, r.GetCreationTimestamp().String()+r.GetName()).String()
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
