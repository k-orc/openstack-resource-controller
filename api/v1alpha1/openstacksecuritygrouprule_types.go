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

// OpenStackSecurityGroupRuleResourceSpec defines the desired state of OpenStackSecurityGroupRule
type OpenStackSecurityGroupRuleResourceSpec struct {
	// The direction in which the security group rule is applied. The only values
	// allowed are "ingress" or "egress". For a compute instance, an ingress
	// security group rule is applied to incoming (ingress) traffic for that
	// instance. An egress rule is applied to traffic leaving the instance.
	Direction string `json:"direction,omitempty"`

	// Description of the rule
	Description string `json:"description,omitempty"`

	// Must be IPv4 or IPv6, and addresses represented in CIDR must match the
	// ingress or egress rules.
	EtherType string `json:"etherType,omitempty"`

	// The OpenStackSecrurityGroup to associate with this security group rule.
	SecurityGroup string `json:"securityGroup,omitempty"`

	// The minimum port number in the range that is matched by the security group
	// rule. If the protocol is TCP or UDP, this value must be less than or equal
	// to the value of the PortRangeMax attribute. If the protocol is ICMP, this
	// value must be an ICMP type.
	PortRangeMin int `json:"portRangeMin,omitempty"`

	// The maximum port number in the range that is matched by the security group
	// rule. The PortRangeMin attribute constrains the PortRangeMax attribute. If
	// the protocol is ICMP, this value must be an ICMP type.
	PortRangeMax int `json:"portRangeMax,omitempty"`

	// The protocol that is matched by the security group rule. Valid values are
	// "tcp", "udp", "icmp" or an empty string.
	Protocol string `json:"protocol,omitempty"`

	// The remote group ID to be associated with this security group rule. You
	// can specify either RemoteGroupID or RemoteIPPrefix.
	RemoteGroupID string `json:"remoteGroupID,omitempty"`

	// The remote IP prefix to be associated with this security group rule. You
	// can specify either RemoteGroupID or RemoteIPPrefix . This attribute
	// matches the specified IP prefix as the source IP address of the IP packet.
	RemoteIPPrefix string `json:"remoteIPPrefix,omitempty"`

	// TenantID is the project owner of this security group rule.
	TenantID string `json:"tenantID,omitempty"`

	// ProjectID is the project owner of this security group rule.
	ProjectID string `json:"projectID,omitempty"`

	// Unmanaged, when true, means that no action will be performed in
	// OpenStack against this resource. This is false by default, except
	// for pre-existing resources that are adopted by passing ID on
	// creation.
	Unmanaged *bool `json:"unmanaged,omitempty"`
}

// OpenStackSecurityGroupRuleStatus defines the observed state of OpenStackSecurityGroupRule
type OpenStackSecurityGroupRuleResourceStatus struct {
	// The UUID for the security group.
	ID string `json:"id"`

	// The direction in which the security group rule is applied. The only values
	// allowed are "ingress" or "egress". For a compute instance, an ingress
	// security group rule is applied to incoming (ingress) traffic for that
	// instance. An egress rule is applied to traffic leaving the instance.
	Direction string `json:"direction,omitempty"`

	// Description of the rule
	Description string `json:"description,omitempty"`

	// Must be IPv4 or IPv6, and addresses represented in CIDR must match the
	// ingress or egress rules.
	EtherType string `json:"etherType,omitempty"`

	// The security group ID to associate with this security group rule.
	SecurityGroupID string `json:"securityGroupID,omitempty"`

	// The minimum port number in the range that is matched by the security group
	// rule. If the protocol is TCP or UDP, this value must be less than or equal
	// to the value of the PortRangeMax attribute. If the protocol is ICMP, this
	// value must be an ICMP type.
	PortRangeMin int `json:"portRangeMin,omitempty"`

	// The maximum port number in the range that is matched by the security group
	// rule. The PortRangeMin attribute constrains the PortRangeMax attribute. If
	// the protocol is ICMP, this value must be an ICMP type.
	PortRangeMax int `json:"portRangeMax,omitempty"`

	// The protocol that is matched by the security group rule. Valid values are
	// "tcp", "udp", "icmp" or an empty string.
	Protocol string `json:"protocol,omitempty"`

	// The remote group ID to be associated with this security group rule. You
	// can specify either RemoteGroupID or RemoteIPPrefix.
	RemoteGroupID string `json:"remoteGroupID,omitempty"`

	// The remote IP prefix to be associated with this security group rule. You
	// can specify either RemoteGroupID or RemoteIPPrefix . This attribute
	// matches the specified IP prefix as the source IP address of the IP packet.
	RemoteIPPrefix string `json:"remoteIPPrefix,omitempty"`

	// TenantID is the project owner of this security group rule.
	TenantID string `json:"tenantID,omitempty"`

	// ProjectID is the project owner of this security group rule.
	ProjectID string `json:"projectID,omitempty"`
}

// OpenStackPortSpec defines the desired state of OpenStackSecurityGroupRule
type OpenStackSecurityGroupRuleSpec struct {
	CommonSpec `json:",inline"`

	// ID is the UUID of the existing OpenStack resource to be adopted. If
	// left empty, the controller will create a new resource using the
	// information in the "resource" stanza.
	ID string `json:"id,omitempty"`

	Resource *OpenStackSecurityGroupRuleResourceSpec `json:"resource,omitempty"`
}

// OpenStackSecurityGroupRuleStatus defines the observed state of OpenStackSecurityGroupRule
type OpenStackSecurityGroupRuleStatus struct {
	CommonStatus `json:",inline"`

	Resource OpenStackSecurityGroupRuleResourceStatus `json:"resource,omitempty"`
}

// Implement OpenStackResourceCommonStatus interface
func (c *OpenStackSecurityGroupRule) OpenStackCommonStatus() *CommonStatus {
	return &c.Status.CommonStatus
}

var _ OpenStackResourceCommonStatus = &OpenStackSecurityGroupRule{}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
//+kubebuilder:printcolumn:name="Error",type=string,JSONPath=`.status.conditions[?(@.type=="Error")].status`
//+kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].message`
//+kubebuilder:printcolumn:name="OpenStackID",type=string,JSONPath=`.status.resource.id`

// OpenStackSecurityGroupRule is the Schema for the openstacksecuritygrouprules API
type OpenStackSecurityGroupRule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OpenStackSecurityGroupRuleSpec   `json:"spec,omitempty"`
	Status OpenStackSecurityGroupRuleStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OpenStackSecurityGroupRuleList contains a list of OpenStackSecurityGroupRule
type OpenStackSecurityGroupRuleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OpenStackSecurityGroupRule `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OpenStackSecurityGroupRule{}, &OpenStackSecurityGroupRuleList{})
}
