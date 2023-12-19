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

	AdminStateUp *bool `json:"adminStateUp,omitempty"`

	DNSDomain string `json:"dnsDomain,omitempty"`

	// MTU is the the maximum transmission unit value to address
	// fragmentation. Minimum value is 68 for IPv4, and 1280 for IPv6.
	MTU int32 `json:"mtu,omitempty"`

	// PortSecurityEnabled is the port security status of the network.
	// Valid values are enabled (true) and disabled (false). This value is
	// used as the default value of port_security_enabled field of a newly
	// created port.
	PortSecurityEnabled *bool `json:"portSecurityEnabled,omitempty"`

	// TenantID is the project owner of the resource. Only admin users can
	// specify a project identifier other than its own.
	TenantID string `json:"tenantID,omitempty"`

	// ProjectID is the project owner of the resource.
	ProjectID string `json:"projectID,omitempty"`

	// QoSPolicyID is the ID of the QoS policy associated with the network.
	QoSPolicyID string `json:"qosPolicyID,omitempty"`

	// External indicates whether the network has an external routing
	// facility that’s not managed by the networking service.
	External *bool `json:"external,omitempty"`

	Segment OpenStackNetworkSegment `json:",inline"`

	// Segment is a list of provider segment objects.
	Segments []OpenStackNetworkSegment `json:"segments,omitempty"`

	// Shared indicates whether this resource is shared across all
	// projects. By default, only administrative users can change this
	// value.
	Shared *bool `json:"shared,omitempty"`

	// VLANTransparent indicates the VLAN transparency mode of the network,
	// which is VLAN transparent (true) or not VLAN transparent (false).
	VLANTransparent *bool `json:"vlanTransparent,omitempty"`

	IsDefault *bool `json:"isDefault,omitempty"`

	// AvailabilityZoneHints is the availability zone candidate for the network.
	AvailabilityZoneHints []string `json:"availabilityZoneHints,omitempty"`
}

type OpenStackNetworkSegment struct {
	// ProviderNetworkType is the type of physical network that this
	// network should be mapped to. For example, flat, vlan, vxlan, or gre.
	// Valid values depend on a networking back-end.
	ProviderNetworkType string `json:"providerNetworkType,omitempty"`

	// ProviderPhysicalNetwork is the physical network where this network
	// should be implemented. The Networking API v2.0 does not provide a
	// way to list available physical networks. For example, the Open
	// vSwitch plug-in configuration file defines a symbolic name that maps
	// to specific bridges on each compute host.
	ProviderPhysicalNetwork string `json:"providerPhysicalNetwork,omitempty"`

	// ProviderSegmentationID is the ID of the isolated segment on the
	// physical network. The network_type attribute defines the
	// segmentation model. For example, if the network_type value is vlan,
	// this ID is a vlan identifier. If the network_type value is gre, this
	// ID is a gre key.
	ProviderSegmentationID int32 `json:"providerSegmentationID,omitempty"`
}

// OpenStackNetworkStatus defines the observed state of OpenStackNetwork
type OpenStackNetworkResourceStatus struct {
	// AdminStateUp is the administrative state of the network, which is up
	// (true) or down (false).
	AdminStateUp bool `json:"adminStateUp,omitempty"`

	// AvailabilityZoneHints is the availability zone candidate for the
	// network.
	AvailabilityZoneHints []string `json:"availabilityZoneHints,omitempty"`

	// Availability is the availability zone for the network.
	AvailabilityZones []string `json:"availabilityZones,omitempty"`

	// CreatedAt contains the timestamp of when the resource was created.
	CreatedAt string `json:"createdAt,omitempty"`

	DNSDomain string `json:"dnsDomain,omitempty"`

	// UUID for the network
	ID string `json:"id,omitempty"`

	// IPV4AddressScope is the ID of the IPv4 address scope that the
	// network is associated with.
	IPV4AddressScope string `json:"ipv4AddressScope,omitempty"`

	// IPV6AddressScope is the ID of the IPv6 address scope that the
	// network is associated with.
	IPV6AddressScope string `json:"ipv6AddressScope,omitempty"`

	// L2Adjacency indicates whether L2 connectivity is available
	// throughout the network.
	L2Adjacency *bool `json:"l2Adjacency,omitempty"`

	// MTU is the the maximum transmission unit value to address
	// fragmentation. Minimum value is 68 for IPv4, and 1280 for IPv6.
	MTU int32 `json:"mtu,omitempty"`

	// Human-readable name for the network. Might not be unique.
	Name string `json:"name,omitempty"`

	// PortSecurityEnabled is the port security status of the network.
	// Valid values are enabled (true) and disabled (false). This value is
	// used as the default value of port_security_enabled field of a newly
	// created port.
	PortSecurityEnabled *bool `json:"portSecurityEnabled,omitempty"`

	// ProjectID is the project owner of the network.
	ProjectID string `json:"projectID,omitempty"`

	Segment OpenStackNetworkSegment `json:",inline"`

	// QoSPolicyID is the ID of the QoS policy associated with the network.
	QoSPolicyID string `json:"qosPolicyID,omitempty"`

	// RevisionNumber is the revision number of the resource.
	RevisionNumber int32 `json:"revisionNumber,omitempty"`

	// External defines whether the network may be used for creation of
	// floating IPs. Only networks with this flag may be an external
	// gateway for routers. The network must have an external routing
	// facility that is not managed by the networking service. If the
	// network is updated from external to internal the unused floating IPs
	// of this network are automatically deleted when extension
	// floatingip-autodelete-internal is present.
	External bool `json:"external,omitempty"`

	// Segment is a list of provider segment objects.
	Segments []OpenStackNetworkSegment `json:"segments,omitempty"`

	// Specifies whether the network resource can be accessed by any tenant.
	Shared bool `json:"shared,omitempty"`

	// Indicates whether network is currently operational. Possible values
	// include `ACTIVE', `DOWN', `BUILD', or `ERROR'. Plug-ins might define
	// additional values.
	Status string `json:"status,omitempty"`

	// Subnets associated with this network.
	Subnets []string `json:"subnets,omitempty"`

	// TenantID is the project owner of the network.
	TenantID string `json:"tenantID,omitempty"`

	// UpdatedAt contains the timestamp of when the resource was last
	// changed.
	UpdatedAt string `json:"updatedAt,omitempty"`

	// VLANTransparent indicates the VLAN transparency mode of the network,
	// which is VLAN transparent (true) or not VLAN transparent (false).
	VLANTransparent bool `json:"vlanTransparent,omitempty"`

	// Description is a human-readable description for the resource.
	Description string `json:"description,omitempty"`

	IsDefault *bool `json:"isDefault,omitempty"`

	// Tags is the list of tags on the resource.
	Tags []string `json:"tags,omitempty"`
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
//+kubebuilder:printcolumn:name="OpenStackID",type=string,JSONPath=`.status.resource.id`

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
