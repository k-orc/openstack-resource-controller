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

// +kubebuilder:validation:Enum:=flat;vlan;vxlan;gre
// +kubebuilder:validation:MinLength:=1
// +kubebuilder:validation:MaxLength:=16
type ProviderNetworkType string

// +kubebuilder:validation:MinLength:=1
// +kubebuilder:validation:MaxLength:=128
type PhysicalNetwork string

// ProviderProperties contains provider-network properties. Currently only
// available in status.
type ProviderProperties struct {
	// networkType is the type of physical network that this
	// network should be mapped to. Supported values are flat, vlan, vxlan, and gre.
	// Valid values depend on the networking back-end.
	NetworkType *ProviderNetworkType `json:"networkType,omitempty"`

	// physicalNetwork is the physical network where this network
	// should be implemented. The Networking API v2.0 does not provide a
	// way to list available physical networks. For example, the Open
	// vSwitch plug-in configuration file defines a symbolic name that maps
	// to specific bridges on each compute host.
	PhysicalNetwork *PhysicalNetwork `json:"physicalNetwork,omitempty"`

	// segmentationID is the ID of the isolated segment on the
	// physical network. The network_type attribute defines the
	// segmentation model. For example, if the network_type value is vlan,
	// this ID is a vlan identifier. If the network_type value is gre, this
	// ID is a gre key.
	SegmentationID *int32 `json:"segmentationID,omitempty"`
}

// TODO: Much better DNSDomain validation

// +kubebuilder:validation:MinLength:=1
// +kubebuilder:validation:MaxLength:=265
type DNSDomain string

// +kubebuilder:validation:Minimum:=68
type MTU int32

// NetworkResourceSpec contains the desired state of a network
type NetworkResourceSpec struct {
	// name will be the name of the created resource. If not specified, the
	// name of the ORC object will be used.
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	// +optional
	Description *OpenStackDescription `json:"description,omitempty"`

	// tags is a list of tags which will be applied to the network.
	// +kubebuilder:validation:MaxItems:=32
	// +listType=set
	Tags []NeutronTag `json:"tags,omitempty"`

	// adminStateUp is the administrative state of the network, which is up (true) or down (false)
	// +optional
	AdminStateUp *bool `json:"adminStateUp,omitempty"`

	// dnsDomain is the DNS domain of the network
	// +optional
	DNSDomain *DNSDomain `json:"dnsDomain,omitempty"`

	// mtu is the the maximum transmission unit value to address
	// fragmentation. Minimum value is 68 for IPv4, and 1280 for IPv6.
	// +optional
	MTU *MTU `json:"mtu,omitempty"`

	// portSecurityEnabled is the port security status of the network.
	// Valid values are enabled (true) and disabled (false). This value is
	// used as the default value of port_security_enabled field of a newly
	// created port.
	// +optional
	PortSecurityEnabled *bool `json:"portSecurityEnabled,omitempty"`

	// external indicates whether the network has an external routing
	// facility that’s not managed by the networking service.
	// +optional
	External *bool `json:"external,omitempty"`

	// shared indicates whether this resource is shared across all
	// projects. By default, only administrative users can change this
	// value.
	// +optional
	Shared *bool `json:"shared,omitempty"`

	// availabilityZoneHints is the availability zone candidate for the network.
	// +listType=set
	// +optional
	AvailabilityZoneHints []AvailabilityZoneHint `json:"availabilityZoneHints,omitempty"`
}

// NetworkFilter defines an existing resource by its properties
// +kubebuilder:validation:MinProperties:=1
type NetworkFilter struct {
	// name of the existing resource
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	// description of the existing resource
	// +optional
	Description *OpenStackDescription `json:"description,omitempty"`

	// external indicates whether the network has an external routing
	// facility that’s not managed by the networking service.
	// +optional
	External *bool `json:"external,omitempty"`

	// projectID specifies the ID of the project which owns the network.
	// +optional
	ProjectID *UUID `json:"projectID,omitempty"`

	FilterByNeutronTags `json:",inline"`
}

// NetworkResourceStatus represents the observed state of the resource.
type NetworkResourceStatus struct {
	// name is a Human-readable name for the network. Might not be unique.
	// +optional
	Name string `json:"name,omitempty"`

	// description is a human-readable description for the resource.
	// +optional
	Description string `json:"description,omitempty"`

	// projectID is the project owner of the network.
	// +optional
	ProjectID string `json:"projectID,omitempty"`

	// Indicates whether network is currently operational. Possible values
	// include `ACTIVE', `DOWN', `BUILD', or `ERROR'. Plug-ins might define
	// additional values.
	// +optional
	Status string `json:"status,omitempty"`

	// tags is the list of tags on the resource.
	// +listType=atomic
	// +optional
	Tags []string `json:"tags,omitempty"`

	NeutronStatusMetadata `json:",inline"`

	// adminStateUp is the administrative state of the network,
	// which is up (true) or down (false).
	AdminStateUp bool `json:"adminStateUp"`

	// availabilityZoneHints is the availability zone candidate for the
	// network.
	// +listType=atomic
	// +optional
	AvailabilityZoneHints []string `json:"availabilityZoneHints,omitempty"`

	DNSDomain string `json:"dnsDomain,omitempty"`

	// mtu is the the maximum transmission unit value to address
	// fragmentation. Minimum value is 68 for IPv4, and 1280 for IPv6.
	// +optional
	MTU int32 `json:"mtu,omitempty"`

	// portSecurityEnabled is the port security status of the network.
	// Valid values are enabled (true) and disabled (false). This value is
	// used as the default value of port_security_enabled field of a newly
	// created port.
	// +optional
	PortSecurityEnabled *bool `json:"portSecurityEnabled,omitempty"`

	// +optional
	Provider *ProviderProperties `json:"provider,omitempty"`

	// external defines whether the network may be used for creation of
	// floating IPs. Only networks with this flag may be an external
	// gateway for routers. The network must have an external routing
	// facility that is not managed by the networking service. If the
	// network is updated from external to internal the unused floating IPs
	// of this network are automatically deleted when extension
	// floatingip-autodelete-internal is present.
	// +optional
	External bool `json:"external,omitempty"`

	// Specifies whether the network resource can be accessed by any tenant.
	// +optional
	Shared bool `json:"shared,omitempty"`

	// subnets associated with this network.
	// +listType=atomic
	// +optional
	Subnets []string `json:"subnets,omitempty"`
}
