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

// +kubebuilder:validation:Enum:=ingress;egress
type RuleDirection string

// +kubebuilder:validation:Enum:=ah;dccp;egp;esp;gre;icmp;icmpv6;igmp;ipip;ipv6-encap;ipv6-frag;ipv6-icmp;ipv6-nonxt;ipv6-opts;ipv6-route;ospf;pgm;rsvp;sctp;tcp;udp;udplite;vrrp
type Protocol string

const (
	ProtocolAH        Protocol = "ah"
	ProtocolDCCP      Protocol = "dccp"
	ProtocolEGP       Protocol = "egp"
	ProtocolESP       Protocol = "esp"
	ProtocolGRE       Protocol = "gre"
	ProtocolICMP      Protocol = "icmp"
	ProtocolICMPV6    Protocol = "icmpv6"
	ProtocolIGMP      Protocol = "igmp"
	ProtocolIPIP      Protocol = "ipip"
	ProtocolIPV6ENCAP Protocol = "ipv6-encap"
	ProtocolIPV6FRAG  Protocol = "ipv6-frag"
	ProtocolIPV6ICMP  Protocol = "ipv6-icmp"
	ProtocolIPV6NONXT Protocol = "ipv6-nonxt"
	ProtocolIPV6OPTS  Protocol = "ipv6-opts"
	ProtocolIPV6ROUTE Protocol = "ipv6-route"
	ProtocolOSPF      Protocol = "ospf"
	ProtocolPGM       Protocol = "pgm"
	ProtocolRSVP      Protocol = "rsvp"
	ProtocolSCTP      Protocol = "sctp"
	ProtocolTCP       Protocol = "tcp"
	ProtocolUDP       Protocol = "udp"
	ProtocolUDPLITE   Protocol = "udplite"
	ProtocolVRRP      Protocol = "vrrp"
)

// +kubebuilder:validation:Enum:=IPv4;IPv6
// +required
type Ethertype string

const (
	EthertypeIPv4 Ethertype = "IPv4"
	EthertypeIPv6 Ethertype = "IPv6"
)

// +kubebuilder:validation:Minimum:=0
// +kubebuilder:validation:Maximum:=65535
type PortNumber int32

type PortRangeSpec struct {
	// +required
	Min PortNumber `json:"min"`
	// +required
	Max PortNumber `json:"max"`
}

type PortRangeStatus struct {
	Min int32 `json:"min"`
	Max int32 `json:"max"`
}

// SecurityGroupRule defines a Security Group rule
// +kubebuilder:validation:MinProperties:=1
// +kubebuilder:validation:XValidation:rule="(!has(self.portRange)|| !(self.protocol == 'tcp'|| self.protocol == 'udp' || self.protocol == 'dccp' || self.protocol == 'sctp' || self.protocol == 'udplite') || (self.portRange.min <= self.portRange.max))",message="portRangeMax should be equal or greater than portRange.min"
// +kubebuilder:validation:XValidation:rule="!(self.protocol == 'icmp' || self.protocol == 'icmpv6') || !has(self.portRange)|| (self.portRange.min >= 0 && self.portRange.min <= 255)",message="When protocol is ICMP or ICMPv6 portRange.min should be between 0 and 255"
// +kubebuilder:validation:XValidation:rule="!(self.protocol == 'icmp' || self.protocol == 'icmpv6') || !has(self.portRange)|| (self.portRange.max >= 0 && self.portRange.max <= 255)",message="When protocol is ICMP or ICMPv6 portRange.max should be between 0 and 255"
// +kubebuilder:validation:XValidation:rule="!has(self.remoteIPPrefix) || (isCIDR(self.remoteIPPrefix) && cidr(self.remoteIPPrefix).ip().family() == 4 && self.ethertype == 'IPv4') || (isCIDR(self.remoteIPPrefix) && cidr(self.remoteIPPrefix).ip().family() == 6 && self.ethertype == 'IPv6')",message="remoteIPPrefix should be a valid CIDR and match the ethertype"
type SecurityGroupRule struct {
	// Description of the existing resource
	// +optional
	Description *OpenStackDescription `json:"description,omitempty"`

	// Direction represents the direction in which the security group rule
	// is applied. Can be ingress or egress.
	// +optional
	Direction *RuleDirection `json:"direction,omitempty"`

	// RemoteIPPrefix is an IP address block. Should match the Ethertype (IPv4 or IPv6)
	// +optional
	RemoteIPPrefix *CIDR `json:"remoteIPPrefix,omitempty"`

	// Protocol is the IP protocol can be represented by a string or an
	// integer represented as a string.
	// +optional
	Protocol *Protocol `json:"protocol,omitempty"`

	// Ethertype must be IPv4 or IPv6, and addresses represented in CIDR
	// must match the ingress or egress rules.
	// +kubebuilder:validation:Required
	Ethertype Ethertype `json:"ethertype"`
	// If the protocol is [tcp, udp, dccp sctp,udplite] PortRange.Min must be less than
	// or equal to the PortRange.Max attribute value.
	// If the protocol is ICMP, this PortRamge.Min must be an ICMP code and PortRange.Max
	// should be an ICMP type
	// +optional
	PortRange *PortRangeSpec `json:"portRange,omitempty"`
}

type SecurityGroupRuleStatus struct {
	// ID is the ID of the security group rule.
	// +required
	ID string `json:"id,omitempty"`

	// Description of the existing resource
	// +optional
	Description string `json:"description,omitempty"`

	// Direction represents the direction in which the security group rule
	// is applied. Can be ingress or egress.
	Direction string `json:"direction,omitempty"`

	// RemoteAddressGroupId (Not in gophercloud)

	// RemoteGroupID
	RemoteGroupID string `json:"remoteGroupID,omitempty"`

	// RemoteIPPrefix
	RemoteIPPrefix string `json:"remoteIPPrefix,omitempty"`

	// Protocol is the IP protocol can be represented by a string, an
	// integer, or null
	Protocol string `json:"protocol,omitempty"`

	// Ethertype must be IPv4 or IPv6, and addresses represented in CIDR
	// must match the ingress or egress rules.
	Ethertype string `json:"ethertype,omitempty"`

	PortRange *PortRangeStatus `json:"portRange,omitempty"`
	// FIXME(mandre) This field is not yet returned by gophercloud
	// BelongsToDefaultSG bool `json:"belongsToDefaultSG,omitempty"`

	// FIXME(mandre) Technically, the neutron status metadata are returned
	// for SG rules. Should we include them? Gophercloud does not
	// implements this yet.
	// NeutronStatusMetadata `json:",inline"`
}

// SecurityGroupResourceSpec contains the desired state of a security group
type SecurityGroupResourceSpec struct {
	// Name will be the name of the created resource. If not specified, the
	// name of the ORC object will be used.
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	// +optional
	Description *OpenStackDescription `json:"description,omitempty"`

	// Tags is a list of tags which will be applied to the security group.
	// +kubebuilder:validation:MaxItems:=32
	// +listType=set
	Tags []NeutronTag `json:"tags,omitempty"`

	// Stateful indicates if the security group is stateful or stateless.
	// +optional
	Stateful *bool `json:"stateful,omitempty"`

	// Rules is a list of security group rules belonging to this SG.
	// +kubebuilder:validation:MaxItems:=256
	// +listType=atomic
	Rules []SecurityGroupRule `json:"rules,omitempty"`
}

// SecurityGroupFilter defines an existing resource by its properties
// +kubebuilder:validation:MinProperties:=1
type SecurityGroupFilter struct {
	// Name of the existing resource
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	// Description of the existing resource
	// +optional
	Description *OpenStackDescription `json:"description,omitempty"`

	// ProjectID specifies the ID of the project which owns the security group.
	// +optional
	ProjectID *UUID `json:"projectID,omitempty"`

	FilterByNeutronTags `json:",inline"`
}

// SecurityGroupResourceStatus represents the observed state of the resource.
type SecurityGroupResourceStatus struct {
	// Human-readable name for the security group. Might not be unique.
	// +optional
	Name string `json:"name,omitempty"`

	// Description is a human-readable description for the resource.
	// +optional
	Description string `json:"description,omitempty"`

	// ProjectID is the project owner of the security group.
	// +optional
	ProjectID string `json:"projectID,omitempty"`

	// Tags is the list of tags on the resource.
	// +listType=atomic
	// +optional
	Tags []string `json:"tags,omitempty"`

	// Stateful indicates if the security group is stateful or stateless.
	// +optional
	Stateful bool `json:"stateful,omitempty"`

	// Rules is a list of security group rules belonging to this SG.
	// +listType=atomic
	Rules []SecurityGroupRuleStatus `json:"rules,omitempty"`

	NeutronStatusMetadata `json:",inline"`
}
