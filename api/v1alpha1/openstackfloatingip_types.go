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

// OpenStackFloatingIPResourceSpec defines the desired state of OpenStackFloatingIP
type OpenStackFloatingIPResourceSpec struct {
	Description string `json:"description,omitempty"`

	// FloatingNetwork is the external OpenStackNetwork where the floating
	// IP is to be created.
	// +kubebuilder:validation:Required
	FloatingNetwork string `json:"floatingNetwork,omitempty"`

	FloatingIPAddress string `json:"floatingIPAddress,omitempty"`
	Port              string `json:"port,omitempty"`
	FixedIPAddress    string `json:"fixedIPAddress,omitempty"`
	Subnet            string `json:"subnetID,omitempty"`
	TenantID          string `json:"tenantID,omitempty"`
	ProjectID         string `json:"projectID,omitempty"`
}

// OpenStackFloatingIPResourceStatus defines the observed state of OpenStackFloatingIP
type OpenStackFloatingIPResourceStatus struct {
	// ID is the unique identifier for the floating IP instance.
	ID string `json:"id,omitempty"`

	// Description for the floating IP instance.
	Description string `json:"description,omitempty"`

	// FloatingNetworkID is the UUID of the external network where the floating
	// IP is to be created.
	FloatingNetworkID string `json:"floatingNetworkID,omitempty"`

	// FloatingIP is the address of the floating IP on the external network.
	FloatingIP string `json:"floatingIPAddress,omitempty"`

	// PortID is the UUID of the port on an internal network that is associated
	// with the floating IP.
	PortID string `json:"portIP,omitempty"`

	// FixedIP is the specific IP address of the internal port which should be
	// associated with the floating IP.
	FixedIP string `json:"fixedIPAddress,omitempty"`

	// TenantID is the project owner of the floating IP. Only admin users can
	// specify a project identifier other than its own.
	TenantID string `json:"tenantID,omitempty"`

	// UpdatedAt contains the timestamp of when the resource was last
	// changed.
	UpdatedAt string `json:"updatedAt,omitempty"`

	// CreatedAt contains the timestamp of when the resource was created.
	CreatedAt string `json:"createdAt,omitempty"`

	// ProjectID is the project owner of the floating IP.
	ProjectID string `json:"projectID,omitempty"`

	// Status is the condition of the API resource.
	Status string `json:"status,omitempty"`

	// RouterID is the ID of the router used for this floating IP.
	RouterID string `json:"routerID,omitempty"`

	// Tags optionally set via extensions/attributestags
	Tags []string `json:"tags,omitempty"`
}

type OpenStackFloatingIPSpec struct {
	CommonSpec `json:",inline"`

	// ID is the UUID of the existing OpenStack resource to be adopted. If
	// left empty, the controller will create a new resource using the
	// information in the "resource" stanza.
	ID string `json:"id,omitempty"`

	Resource *OpenStackFloatingIPResourceSpec `json:"resource,omitempty"`
}

type OpenStackFloatingIPStatus struct {
	CommonStatus `json:",inline"`

	Resource OpenStackFloatingIPResourceStatus `json:"resource,omitempty"`
}

// Implement OpenStackResourceCommonStatus interface
func (c *OpenStackFloatingIP) OpenStackCommonStatus() *CommonStatus {
	return &c.Status.CommonStatus
}

var _ OpenStackResourceCommonStatus = &OpenStackFlavor{}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
//+kubebuilder:printcolumn:name="Error",type=string,JSONPath=`.status.conditions[?(@.type=="Error")].status`
//+kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].message`

// OpenStackFloatingIP is the Schema for the openstackfloatingips API
type OpenStackFloatingIP struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OpenStackFloatingIPSpec   `json:"spec,omitempty"`
	Status OpenStackFloatingIPStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OpenStackFloatingIPList contains a list of OpenStackFloatingIP
type OpenStackFloatingIPList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OpenStackFloatingIP `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OpenStackFloatingIP{}, &OpenStackFloatingIPList{})
}
