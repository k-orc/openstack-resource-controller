/*
Copyright 2026 The ORC Authors.

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

// QosPolicyResourceSpec contains the desired state of a QoS Policy
type QosPolicyResourceSpec struct {
	// name will be the name of the created resource. If not specified, the
	// name of the ORC object will be used.
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	// description is a human-readable description for the resource.
	// +optional
	Description *NeutronDescription `json:"description,omitempty"`

	// tags is a list of tags which will be applied to the QoS policy.
	// +kubebuilder:validation:MaxItems:=64
	// +listType=set
	// +optional
	Tags []NeutronTag `json:"tags,omitempty"`

	// shared indicates whether this resource is shared across all
	// projects. By default, only administrative users can change this
	// value.
	// +optional
	Shared *bool `json:"shared,omitempty"`

	// bandwidthlimitrules is a list of bandwidth limit rules belonging to this QoS.
	// +kubebuilder:validation:MaxItems:=256
	// +listType=atomic
	// +optional
	BandWidthLimitRules []QosBandwidthLimitRule `json:"bandwidthlimitrules,omitempty"`

	// dscpmarkingrules is a list of DSCP marking rules belonging to this QoS.
	// +kubebuilder:validation:MaxItems:=256
	// +listType=atomic
	// +optional
	DscpMarkingRules []QosDscpMarkingRule `json:"dscpmarkingrules,omitempty"`

	// minimumbandwidthrules is a list of minimum bandwidth rules belonging to this QoS.
	// +kubebuilder:validation:MaxItems:=256
	// +listType=atomic
	// +optional
	MinimumBandwidthRules []QosMinimumBandwidthRule `json:"minimumbandwidthrules,omitempty"`

	// minimumpacketraterules is a list of minimum packet rate rules belonging to this QoS.
	// +kubebuilder:validation:MaxItems:=256
	// +listType=atomic
	// +optional
	MinimumPacketRateRules []QosMinimumPacketRateRule `json:"minimumpacketraterules,omitempty"`

	// minimumpacketratelimitrules is a list of minimum packet rate limit rules belonging to this QoS.
	// +kubebuilder:validation:MaxItems:=256
	// +listType=atomic
	// +optional
	MinimumPacketRateLimitRules []QosMinimumPacketRateLimitRule `json:"minimumpacketratelimitrules,omitempty"`

	// projectRef is a reference to the ORC Project this resource is associated with.
	// Typically, only used by admin.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="projectRef is immutable"
	// +orc:kustomize:ref=Project
	ProjectRef *KubernetesNameRef `json:"projectRef,omitempty"`
}

// QosBandwidthLimitRule defines a QoS bandwidth limit rule
// +kubebuilder:validation:MinProperties:=1
// +kubebuilder:validation:XValidation:rule="!has(self.maxburstkbps) || has(self.maxkbps)",message="maxkbps is required when maxburstkbps is specified"
type QosBandwidthLimitRule struct {
	// description is a human-readable description for the resource.
	// +optional
	Description *NeutronDescription `json:"description,omitempty"`

	// direction represents the direction in which the rule
	// is applied. Can be ingress or egress.
	// +optional
	Direction *RuleDirection `json:"direction,omitempty"`

	// maxkbps is the maximum KBPS (kilobits per second) value.
	// +optional
	// +kubebuilder:validation:Minimum:=0
	Maxkbps *int32 `json:"maxkbps,omitempty"`

	// maxburstkbps is the maximum burst size (in kilobits).
	// +optional
	// +kubebuilder:validation:Minimum:=0
	MaxBurstkbps *int32 `json:"maxburstkbps,omitempty"`
}

// QosDscpMarkingRule defines a QoS DSCP Marking rule
// +kubebuilder:validation:MinProperties:=1
// +kubebuilder:validation:XValidation:rule="self.dscpmark in [0, 8, 10, 12, 14, 16, 18, 20, 22, 24, 26, 28, 30, 32, 34, 36, 38, 40, 46, 48, 56]",message="dscpmark must be a valid DSCP value"
type QosDscpMarkingRule struct {
	// dscpmark is the DSCP mark value to apply to packets.
	// Valid values are: 0, 8, 10, 12, 14, 16, 18, 20, 22, 24, 26, 28, 30, 32, 34, 36, 38, 40, 46, 48, 56.
	// +optional
	DscpMark *int32 `json:"dscpmark,omitempty"`
}

// QosMinimumBandwidthRule defines a QoS Minimum Bandwidth rule
// +kubebuilder:validation:MinProperties:=1
type QosMinimumBandwidthRule struct {
	// direction represents the direction in which the rule
	// is applied. Can be ingress or egress.
	// +optional
	Direction *RuleDirection `json:"direction,omitempty"`

	// minkbps is the minimum KBPS (kilobits per second) value which should be available for the port.
	// +optional
	// +kubebuilder:validation:Minimum:=0
	Minkbps *int32 `json:"minkbps,omitempty"`
}

// QosMinimumPacketRateRule defines a QoS Minimum Packet Rate rule
// +kubebuilder:validation:MinProperties:=1
type QosMinimumPacketRateRule struct {
	// direction represents the direction in which the rule
	// is applied. Can be ingress or egress.
	// +optional
	Direction *RuleDirection `json:"direction,omitempty"`

	// minkpps is the minimum KPPS (kilo packets per second) value which should be available for the port.
	// +optional
	// +kubebuilder:validation:Minimum:=0
	Minkpps *int32 `json:"minkpps,omitempty"`
}

// QosMinimumPacketRateLimitRule defines a QoS Packet Rate Limit rule
// +kubebuilder:validation:MinProperties:=1
// +kubebuilder:validation:XValidation:rule="!has(self.maxburstkpps) || has(self.maxkpps)",message="maxkpps is required when maxburstkpps is specified"
type QosMinimumPacketRateLimitRule struct {
	// direction represents the direction in which the rule
	// is applied. Can be ingress or egress.
	// +optional
	Direction *RuleDirection `json:"direction,omitempty"`

	// maxkpps is the maximum KPPS (kilo packets per second) value.
	// +optional
	// +kubebuilder:validation:Minimum:=0
	Maxkpps *int32 `json:"maxkpps,omitempty"`

	// maxburstkpps is the maximum burst size (in kilo packets per second).
	// +optional
	// +kubebuilder:validation:Minimum:=0
	MaxBurstkpps *int32 `json:"maxburstkpps,omitempty"`
}

type QosBandwidthLimitRuleStatus struct {
	// id is the ID of the bandwidth limit rule.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	ID string `json:"id,omitempty"`

	// description is a human-readable description for the resource.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Description string `json:"description,omitempty"`

	// direction represents the direction in which the rule is applied.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Direction string `json:"direction,omitempty"`

	// maxkbps is the maximum KBPS (kilobits per second) value.
	// +optional
	Maxkbps int32 `json:"maxkbps,omitempty"`

	// maxburstkbps is the maximum burst size (in kilobits).
	// +optional
	MaxBurstkbps int32 `json:"maxburstkbps,omitempty"`
}

type QosDscpMarkingRuleStatus struct {
	// id is the ID of the DSCP marking rule.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	ID string `json:"id,omitempty"`

	// dscpmark is the DSCP mark value.
	// +optional
	DscpMark int32 `json:"dscpmark,omitempty"`
}

type QosMinimumBandwidthRuleStatus struct {
	// id is the ID of the minimum bandwidth rule.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	ID string `json:"id,omitempty"`

	// direction represents the direction in which the rule is applied.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Direction string `json:"direction,omitempty"`

	// minkbps is the minimum KBPS (kilobits per second) value.
	// +optional
	Minkbps int32 `json:"minkbps,omitempty"`
}

type QosMinimumPacketRateRuleStatus struct {
	// id is the ID of the minimum packet rate rule.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	ID string `json:"id,omitempty"`

	// direction represents the direction in which the rule is applied.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Direction string `json:"direction,omitempty"`

	// minkpps is the minimum KPPS (kilo packets per second) value.
	// +optional
	Minkpps int32 `json:"minkpps,omitempty"`
}

type QosMinimumPacketRateLimitRuleStatus struct {
	// id is the ID of the packet rate limit rule.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	ID string `json:"id,omitempty"`

	// direction represents the direction in which the rule is applied.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Direction string `json:"direction,omitempty"`

	// maxkpps is the maximum KPPS (kilo packets per second) value.
	// +optional
	Maxkpps int32 `json:"maxkpps,omitempty"`

	// maxburstkpps is the maximum burst size (in kilo packets per second).
	// +optional
	MaxBurstkpps int32 `json:"maxburstkpps,omitempty"`
}

// QosPolicyFilter defines an existing resource by its properties
// +kubebuilder:validation:MinProperties:=1
type QosPolicyFilter struct {
	// name of the existing resource
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	// description of the existing resource
	// +optional
	Description *NeutronDescription `json:"description,omitempty"`

	// projectRef is a reference to the ORC Project this resource is associated with.
	// Typically, only used by admin.
	// +optional
	// +orc:kustomize:ref=Project
	ProjectRef *KubernetesNameRef `json:"projectRef,omitempty"`

	FilterByNeutronTags `json:",inline"`
}

// QosPolicyResourceStatus represents the observed state of the resource.
type QosPolicyResourceStatus struct {
	// name is a Human-readable name for the QoS Policy. Might not be unique.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Name string `json:"name,omitempty"`

	// description is a human-readable description for the resource.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Description string `json:"description,omitempty"`

	// projectID is the project owner of the QoS Policy.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	ProjectID string `json:"projectID,omitempty"`

	// tags is the list of tags on the resource.
	// +kubebuilder:validation:MaxItems:=64
	// +kubebuilder:validation:items:MaxLength=1024
	// +listType=atomic
	// +optional
	Tags []string `json:"tags,omitempty"`

	// shared indicates whether this resource is shared across all
	// projects. By default, only administrative users can change this
	// value.
	// +optional
	Shared *bool `json:"shared,omitempty"`

	// bandwidthlimitrules is a list of bandwidth limit rules belonging to this QoS.
	// +kubebuilder:validation:MaxItems:=256
	// +listType=atomic
	// +optional
	BandWidthLimitRules []QosBandwidthLimitRuleStatus `json:"bandwidthlimitrules,omitempty"`

	// dscpmarkingrules is a list of DSCP marking rules belonging to this QoS.
	// +kubebuilder:validation:MaxItems:=256
	// +listType=atomic
	// +optional
	DscpMarkingRules []QosDscpMarkingRuleStatus `json:"dscpmarkingrules,omitempty"`

	// minimumbandwidthrules is a list of minimum bandwidth rules belonging to this QoS.
	// +kubebuilder:validation:MaxItems:=256
	// +listType=atomic
	// +optional
	MinimumBandwidthRules []QosMinimumBandwidthRuleStatus `json:"minimumbandwidthrules,omitempty"`

	// minimumpacketraterules is a list of minimum packet rate rules belonging to this QoS.
	// +kubebuilder:validation:MaxItems:=256
	// +listType=atomic
	// +optional
	MinimumPacketRateRules []QosMinimumPacketRateRuleStatus `json:"minimumpacketraterules,omitempty"`

	// minimumpacketratelimitrules is a list of packet rate limit rules belonging to this QoS.
	// +kubebuilder:validation:MaxItems:=256
	// +listType=atomic
	// +optional
	MinimumPacketRateLimitRules []QosMinimumPacketRateLimitRuleStatus `json:"minimumpacketratelimitrules,omitempty"`

	NeutronStatusMetadata `json:",inline"`
}
