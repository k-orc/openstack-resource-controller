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

// OpenStackSubnetSpec defines the desired state of OpenStackSubnet
type OpenStackSubnetResourceSpec struct {
	// NetworkID is the OpenStackNetwork the subnet will be associated with.
	Network string `json:"network,omitempty"`

	// CIDR is the address CIDR of the subnet.
	CIDR string `json:"cidr,omitempty"`

	// Name is a human-readable name of the subnet.
	Name string `json:"name,omitempty"`

	// Description of the subnet.
	Description string `json:"description,omitempty"`

	// AllocationPools are IP Address pools that will be available for DHCP.
	AllocationPools []OpenStackSubnetAllocationPool `json:"allocationPools,omitempty"`

	// GatewayIP sets gateway information for the subnet. Setting to nil will
	// cause a default gateway to automatically be created. Setting to an empty
	// string will cause the subnet to be created with no gateway. Setting to
	// an explicit address will set that address as the gateway.
	GatewayIP *string `json:"gatewayIP,omitempty"`

	// IPVersion is the IP version for the subnet.
	IPVersion string `json:"ipVersion,omitempty"`

	// EnableDHCP will either enable to disable the DHCP service.
	EnableDHCP *bool `json:"enableDHCP,omitempty"`

	// DNSNameservers are the nameservers to be set via DHCP.
	DNSNameservers []string `json:"dnsNameservers,omitempty"`

	// ServiceTypes are the service types associated with the subnet.
	ServiceTypes []string `json:"serviceTypes,omitempty"`

	// HostRoutes are any static host routes to be set via DHCP.
	HostRoutes []OpenStackSubnetHostRoute `json:"hostRoutes,omitempty"`

	// The IPv6 address modes specifies mechanisms for assigning IPv6 IP addresses.
	IPv6AddressMode string `json:"ipv6AddressMode,omitempty"`

	// The IPv6 router advertisement specifies whether the networking service
	// should transmit ICMPv6 packets.
	IPv6RAMode string `json:"ipv6RAMode,omitempty"`

	// SubnetPoolID is the id of the subnet pool that subnet should be associated to.
	// SubnetPoolID string `json:"subnetPoolID,omitempty"`

	// Prefixlen is used when user creates a subnet from the subnetpool. It will
	// overwrite the "default_prefixlen" value of the referenced subnetpool.
	// Prefixlen int `json:"prefixlen,omitempty"`
}

// OpenStackSubnetAllocationPool represents a sub-range of cidr available for
// dynamic allocation to ports, e.g. {Start: "10.0.0.2", End: "10.0.0.254"}
type OpenStackSubnetAllocationPool struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// OpenStackSubnetHostRoute represents a route that should be used by devices with IPs from
// a subnet (not including local subnet route).
type OpenStackSubnetHostRoute struct {
	DestinationCIDR string `json:"destination"`
	NextHop         string `json:"nexthop"`
}

// OpenStackSubnetStatus defines the observed state of OpenStackSubnet
type OpenStackSubnetResourceStatus struct {
	// UUID representing the subnet.
	ID string `json:"id,omitempty"`

	// UUID of the parent network.
	NetworkID string `json:"networkID,omitempty"`

	// Human-readable name for the subnet. Might not be unique.
	Name string `json:"name,omitempty"`

	// Description for the subnet.
	Description string `json:"description,omitempty"`

	// IP version, either `4' or `6'.
	IPVersion int `json:"ipVersion,omitempty"`

	// CIDR representing IP range for this subnet, based on IP version.
	CIDR string `json:"cidr,omitempty"`

	// Default gateway used by devices in this subnet.
	GatewayIP string `json:"gatewayIP,omitempty"`

	// DNS name servers used by hosts in this subnet.
	DNSNameservers []string `json:"dnsNameservers,omitempty"`

	// Service types associated with the subnet.
	ServiceTypes []string `json:"serviceTypes,omitempty"`

	// Sub-ranges of CIDR available for dynamic allocation to ports.
	// See AllocationPool.
	AllocationPools []OpenStackSubnetAllocationPool `json:"allocationPools,omitempty"`

	// Routes that should be used by devices with IPs from this subnet
	// (not including local subnet route).
	HostRoutes []OpenStackSubnetHostRoute `json:"hostRoutes,omitempty"`

	// Specifies whether DHCP is enabled for this subnet or not.
	EnableDHCP bool `json:"enableDHCP,omitempty"`

	// TenantID is the project owner of the subnet.
	TenantID string `json:"tenantID,omitempty"`

	// ProjectID is the project owner of the subnet.
	ProjectID string `json:"projectID,omitempty"`

	// The IPv6 address modes specifies mechanisms for assigning IPv6 IP addresses.
	IPv6AddressMode string `json:"ipv6AddressMode,omitempty"`

	// The IPv6 router advertisement specifies whether the networking service
	// should transmit ICMPv6 packets.
	IPv6RAMode string `json:"ipv6RAMode,omitempty"`

	// SubnetPoolID is the id of the subnet pool associated with the subnet.
	SubnetPoolID string `json:"subnetpoolID,omitempty"`

	// Tags optionally set via extensions/attributestags
	Tags []string `json:"tags,omitempty"`

	// RevisionNumber optionally set via extensions/standard-attr-revisions
	RevisionNumber int `json:"revisionNumber,omitempty"`
}

// OpenStackSubnetSpec defines the desired state of OpenStackPort
type OpenStackSubnetSpec struct {
	CommonSpec `json:",inline"`

	// ID is the UUID of the existing OpenStack resource to be adopted. If
	// left empty, the controller will create a new resource using the
	// information in the "resource" stanza.
	ID string `json:"id,omitempty"`

	Resource *OpenStackSubnetResourceSpec `json:"resource,omitempty"`
}

// OpenStackSubnetStatus defines the observed state of OpenStackPort
type OpenStackSubnetStatus struct {
	CommonStatus `json:",inline"`

	Resource OpenStackSubnetResourceStatus `json:"resource,omitempty"`
}

// Implement OpenStackResourceCommonStatus interface
func (c *OpenStackSubnet) OpenStackCommonStatus() *CommonStatus {
	return &c.Status.CommonStatus
}

var _ OpenStackResourceCommonStatus = &OpenStackSubnet{}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
//+kubebuilder:printcolumn:name="Error",type=string,JSONPath=`.status.conditions[?(@.type=="Error")].status`
//+kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].message`

// OpenStackSubnet is the Schema for the openstacksubnets API
type OpenStackSubnet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OpenStackSubnetSpec   `json:"spec,omitempty"`
	Status OpenStackSubnetStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OpenStackSubnetList contains a list of OpenStackSubnet
type OpenStackSubnetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OpenStackSubnet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OpenStackSubnet{}, &OpenStackSubnetList{})
}
