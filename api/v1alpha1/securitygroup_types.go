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
// +kubebuilder:validation:MinLength:=1
// +kubebuilder:validation:MaxLength:=16
type RuleDirection string

// +kubebuilder:validation:Pattern:=\b([01]?[0-9][0-9]?|2[0-4][0-9]|25[0-5])\b|any|ah|dccp|egp|esp|gre|icmp|icmpv6|igmp|ipip|ipv6-encap|ipv6-frag|ipv6-icmp|ipv6-nonxt|ipv6-opts|ipv6-route|ospf|pgm|rsvp|sctp|tcp|udp|udplite|vrrp
// +kubebuilder:validation:MinLength:=1
// +kubebuilder:validation:MaxLength:=16
type Protocol string

// +kubebuilder:validation:Enum:=IPv4;IPv6
// +kubebuilder:validation:MinLength:=1
// +kubebuilder:validation:MaxLength:=16
type Ethertype string

// SecurityGroupRule defines a Security Group rule
// +kubebuilder:validation:MinProperties:=1
type SecurityGroupRule struct {
	// Description of the existing resource
	// +optional
	Description *OpenStackDescription `json:"description,omitempty"`

	// Direction represents the direction in which the security group rule
	// is applied. Can be ingress or egress.
	Direction *RuleDirection `json:"direction,omitempty"`

	// RemoteAddressGroupId (Not in gophercloud)

	// RemoteGroupID
	RemoteGroupID *UUID `json:"remoteGroupID,omitempty"`

	// RemoteIPPrefix
	RemoteIPPrefix *CIDR `json:"remoteIPPrefix,omitempty"`

	// Protocol is the IP protocol can be represented by a string, an
	// integer, or null
	Protocol *Protocol `json:"protocol,omitempty"`

	// EtherType must be IPv4 or IPv6, and addresses represented in CIDR
	// must match the ingress or egress rules.
	Ethertype *Ethertype `json:"ethertype,omitempty"`

	PortRangeMin *int32 `json:"portRangeMin,omitempty"`
	PortRangeMax *int32 `json:"portRangeMax,omitempty"`
}

type SecurityGroupRuleStatus struct {
	// ID is the ID of the security group rule.
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

	PortRangeMin int `json:"portRangeMin,omitempty"`
	PortRangeMax int `json:"portRangeMax,omitempty"`

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
