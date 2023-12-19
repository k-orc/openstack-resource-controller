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

// OpenStackRouterSpec defines the desired state of OpenStackRouter
type OpenStackRouterResourceSpec struct {
	// Name of the OpenStack resource.
	Name string `json:"name,omitempty"`

	Description string `json:"description,omitempty"`

	ExternalGateway *OpenStackRouterSpecExternalGateway `json:"externalGatewayInfo,omitempty"`

	// AdminStateUp is the administrative state of the router.
	AdminStateUp *bool `json:"adminStateUp,omitempty"`

	// Distributed is whether router is distributed or not.
	Distributed *bool `json:"distributed,omitempty"`

	// Availability zone hints groups router nodes.
	// Used to make router resources highly available.
	AvailabilityZoneHints []string `json:"availabilityZoneHints,omitempty"`

	// TenantID is the project owner of the router. Only admin users can
	// specify a project identifier other than its own.
	TenantID string `json:"tenantID,omitempty"`

	// ProjectID is the project owner of the router.
	ProjectID string `json:"projectID,omitempty"`

	// All the ports that will be added as router interfaces
	Ports []string `json:"ports,omitempty"`
}

// OpenStackRouterStatus defines the observed state of OpenStackRouter
type OpenStackRouterResourceStatus struct {
	// Status indicates whether or not a router is currently operational.
	Status string `json:"status,omitempty"`

	// GatewayInfo provides information on external gateway for the router.
	GatewayInfo OpenStackRouterStatusExternalGatewayInfo `json:"externalGatewayInfo,omitempty"`

	// Ports provides information on the interfaces connected to this router
	Ports []string `json:"ports,omitempty"`

	// AdminStateUp is the administrative state of the router.
	AdminStateUp bool `json:"adminStateUp,omitempty"`

	// Distributed is whether router is disitrubted or not.
	Distributed bool `json:"distributed,omitempty"`

	// Name is the human readable name for the router. It does not have to be
	// unique.
	Name string `json:"name,omitempty"`

	// Description for the router.
	Description string `json:"description,omitempty"`

	// ID is the unique identifier for the router.
	ID string `json:"ID,omitempty"`

	// TenantID is the project owner of the router. Only admin users can
	// specify a project identifier other than its own.
	TenantID string `json:"tenantID,omitempty"`

	// ProjectID is the project owner of the router.
	ProjectID string `json:"projectID,omitempty"`

	// Routes are a collection of static routes that the router will host.
	Routes []OpenStackRouterRoute `json:"routes,omitempty"`

	// Availability zone hints groups network nodes that run services like DHCP, L3, FW, and others.
	// Used to make network resources highly available.
	AvailabilityZoneHints []string `json:"availabilityZoneHints,omitempty"`

	// Tags optionally set via extensions/attributestags
	Tags []string `json:"tags,omitempty"`
}

// OpenStackRouterSpecExternalGateway represents the information of an external gateway for any
// particular network router.
type OpenStackRouterSpecExternalGateway struct {
	Network          string                           `json:"network,omitempty"`
	EnableSNAT       *bool                            `json:"enableSNAT,omitempty"`
	ExternalFixedIPs []OpenStackRouterExternalFixedIP `json:"externalFixedIps,omitempty"`
}

// OpenStackRouterStatusExternalGatewayInfo represents the information of an external gateway for any
// particular network router.
type OpenStackRouterStatusExternalGatewayInfo struct {
	NetworkID        string                           `json:"networkID,omitempty"`
	EnableSNAT       *bool                            `json:"enableSNAT,omitempty"`
	ExternalFixedIPs []OpenStackRouterExternalFixedIP `json:"externalFixedIps,omitempty"`
}

// OpenStackRouterExternalFixedIP is the IP address and subnet of the external gateway of a
// router.
type OpenStackRouterExternalFixedIP struct {
	IPAddress string `json:"ipAddress,omitempty"`
	Subnet    string `json:"subnet,omitempty"`
}

// OpenStackRouterRoute is a possible route in a router.
type OpenStackRouterRoute struct {
	NextHop         string `json:"nextHop"`
	DestinationCIDR string `json:"destination"`
}

// OpenStackRouterInterfaceInfo represents information about a particular router interface. As
// mentioned above, in order for a router to forward to a subnet, it needs an
// interface.
type OpenStackRouterInterfaceInfo struct {
	// SubnetID is the ID of the subnet which this interface is associated with.
	SubnetID string `json:"subnetID,omitempty"`

	// PortID is the ID of the port that is a part of the subnet.
	PortID string `json:"portID,omitempty"`

	// ID is the UUID of the interface.
	ID string `json:"ID,omitempty"`

	// TenantID is the owner of the interface.
	TenantID string `json:"tenantID,omitempty"`
}

// OpenStackRouterSpec defines the desired state of OpenStackPort
type OpenStackRouterSpec struct {
	CommonSpec `json:",inline"`

	// ID is the UUID of the existing OpenStack resource to be adopted. If
	// left empty, the controller will create a new resource using the
	// information in the "resource" stanza.
	ID string `json:"id,omitempty"`

	Resource *OpenStackRouterResourceSpec `json:"resource,omitempty"`
}

// OpenStackRouterStatus defines the observed state of OpenStackPort
type OpenStackRouterStatus struct {
	CommonStatus `json:",inline"`

	Resource OpenStackRouterResourceStatus `json:"resource,omitempty"`
}

// Implement OpenStackResourceCommonStatus interface
func (c *OpenStackRouter) OpenStackCommonStatus() *CommonStatus {
	return &c.Status.CommonStatus
}

var _ OpenStackResourceCommonStatus = &OpenStackRouter{}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
//+kubebuilder:printcolumn:name="Error",type=string,JSONPath=`.status.conditions[?(@.type=="Error")].status`
//+kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].message`
//+kubebuilder:printcolumn:name="OpenStackID",type=string,JSONPath=`.status.resource.id`

// OpenStackRouter is the Schema for the openstackrouters API
type OpenStackRouter struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OpenStackRouterSpec   `json:"spec,omitempty"`
	Status OpenStackRouterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OpenStackRouterList contains a list of OpenStackRouter
type OpenStackRouterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OpenStackRouter `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OpenStackRouter{}, &OpenStackRouterList{})
}
