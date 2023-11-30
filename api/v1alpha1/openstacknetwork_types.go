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

// OpenStackNetworkSpec defines the desired state of OpenStackNetwork
type OpenStackNetworkResourceSpec struct {
	// ID is the OpenStack UUID of the resource. If left empty, the
	// controller will create a new resource and populate this field. If
	// manually populated, the controller will adopt the corresponding
	// resource.
	ID string `json:"id,omitempty"`

	// Name of the OpenStack resource.
	Name string `json:"name,omitempty"`

	Description string `json:"description,omitempty"`
}

// OpenStackNetworkStatus defines the observed state of OpenStackNetwork
type OpenStackNetworkResourceStatus struct {
	// UUID for the network
	ID string `json:"id"`

	// Human-readable name for the network. Might not be unique.
	Name string `json:"name"`

	// Description for the network
	Description string `json:"description,omitempty"`

	// The administrative state of network. If false (down), the network does not
	// forward packets.
	AdminStateUp bool `json:"adminStateUp,omitempty"`

	// Indicates whether network is currently operational. Possible values include
	// `ACTIVE', `DOWN', `BUILD', or `ERROR'. Plug-ins might define additional
	// values.
	Status string `json:"status,omitempty"`

	// Subnets associated with this network.
	Subnets []string `json:"subnets,omitempty"`

	// TenantID is the project owner of the network.
	TenantID string `json:"tenantID,omitempty"`

	// UpdatedAt contains the timestamp of when the resource was last
	// changed.
	UpdatedAt string `json:"updatedAt,omitempty"`

	// CreatedAt contains the timestamp of when the resource was created.
	CreatedAt string `json:"createdAt,omitempty"`

	// ProjectID is the project owner of the network.
	ProjectID string `json:"projectID,omitempty"`

	// Specifies whether the network resource can be accessed by any tenant.
	Shared bool `json:"shared,omitempty"`

	// Availability zone hints groups network nodes that run services like DHCP, L3, FW, and others.
	// Used to make network resources highly available.
	AvailabilityZoneHints []string `json:"availabilityZoneHints,omitempty"`

	// Tags optionally set via extensions/attributestags
	Tags []string `json:"tags,omitempty"`

	// RevisionNumber optionally set via extensions/standard-attr-revisions
	RevisionNumber int `json:"revisionNumber,omitempty"`
}

// OpenStackNetworkSpec defines the desired state of OpenStackNetwork
type OpenStackNetworkSpec struct {
	CommonSpec `json:",inline"`

	// ID is the UUID of the existing OpenStack resource to be adopted. If
	// left empty, the controller will create a new resource using the
	// information in the "resource" stanza.
	ID string `json:"id,omitempty"`

	Resource *OpenStackNetworkResourceSpec `json:"resource,omitempty"`
}

// OpenStackNetworkStatus defines the observed state of OpenStackNetwork
type OpenStackNetworkStatus struct {
	CommonStatus `json:",inline"`

	Resource OpenStackNetworkResourceStatus `json:"resource,omitempty"`
}

// Implement OpenStackResourceCommonStatus interface
func (c *OpenStackNetwork) OpenStackCommonStatus() *CommonStatus {
	return &c.Status.CommonStatus
}

var _ OpenStackResourceCommonStatus = &OpenStackNetwork{}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
//+kubebuilder:printcolumn:name="Error",type=string,JSONPath=`.status.conditions[?(@.type=="Error")].status`
//+kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].message`

// OpenStackNetwork is the Schema for the openstacknetworks API
type OpenStackNetwork struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OpenStackNetworkSpec   `json:"spec,omitempty"`
	Status OpenStackNetworkStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OpenStackNetworkList contains a list of OpenStackNetwork
type OpenStackNetworkList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OpenStackNetwork `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OpenStackNetwork{}, &OpenStackNetworkList{})
}
