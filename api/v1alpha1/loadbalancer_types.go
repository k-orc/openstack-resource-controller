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

// Octavia provisioning status values
const (
	LoadbalancerProvisioningStatusActive        = "ACTIVE"
	LoadbalancerProvisioningStatusError         = "ERROR"
	LoadbalancerProvisioningStatusPendingCreate = "PENDING_CREATE"
	LoadbalancerProvisioningStatusPendingUpdate = "PENDING_UPDATE"
	LoadbalancerProvisioningStatusPendingDelete = "PENDING_DELETE"
)

// +kubebuilder:validation:MinLength:=1
// +kubebuilder:validation:MaxLength:=255
type LoadBalancerTag string

// LoadBalancerResourceSpec contains the desired state of the resource.
// +kubebuilder:validation:XValidation:rule="has(self.vipSubnetRef) || has(self.vipNetworkRef) || has(self.vipPortRef)",message="at least one of vipSubnetRef, vipNetworkRef, or vipPortRef must be specified"
type LoadBalancerResourceSpec struct {
	// name will be the name of the created resource. If not specified, the
	// name of the ORC object will be used.
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	// description is a human-readable description for the resource.
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=255
	// +optional
	Description *string `json:"description,omitempty"`

	// vipSubnetRef is the subnet on which to allocate the load balancer's address.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="vipSubnetRef is immutable"
	VipSubnetRef *KubernetesNameRef `json:"vipSubnetRef,omitempty"`

	// vipNetworkRef is the network on which to allocate the load balancer's address.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="vipNetworkRef is immutable"
	VipNetworkRef *KubernetesNameRef `json:"vipNetworkRef,omitempty"`

	// vipPortRef is a reference to a neutron port to use for the VIP. If the port
	// has more than one subnet you must specify either vipSubnetRef or vipAddress
	// to clarify which address should be used for the VIP.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="vipPortRef is immutable"
	VipPortRef *KubernetesNameRef `json:"vipPortRef,omitempty"`

	// flavorRef is a reference to the ORC Flavor which this resource is associated with.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="flavorRef is immutable"
	FlavorRef *KubernetesNameRef `json:"flavorRef,omitempty"`

	// projectRef is a reference to the ORC Project which this resource is associated with.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="projectRef is immutable"
	ProjectRef *KubernetesNameRef `json:"projectRef,omitempty"`

	// adminStateUp is the administrative state of the load balancer, which is up (true) or down (false)
	// +optional
	AdminStateUp *bool `json:"adminStateUp,omitempty"`

	// availabilityZone is the availability zone in which to create the load balancer.
	// +kubebuilder:validation:MaxLength=255
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="availabilityZone is immutable"
	AvailabilityZone string `json:"availabilityZone,omitempty"`

	// provider is the name of the load balancer provider.
	// +kubebuilder:validation:MaxLength=255
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="provider is immutable"
	Provider string `json:"provider,omitempty"`

	// vipAddress is the specific IP address to use for the VIP (optional).
	// If not specified, one is allocated automatically from the subnet.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="vipAddress is immutable"
	VipAddress *IPvAny `json:"vipAddress,omitempty"`

	// tags is a list of tags which will be applied to the load balancer.
	// +kubebuilder:validation:MaxItems:=64
	// +listType=set
	// +optional
	Tags []LoadBalancerTag `json:"tags,omitempty"`
}

// LoadBalancerFilter defines an existing resource by its properties
// +kubebuilder:validation:MinProperties:=1
type LoadBalancerFilter struct {
	// name of the existing resource
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	// description of the existing resource
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=255
	// +optional
	Description *string `json:"description,omitempty"`

	// projectRef is a reference to the ORC Project this resource is associated with.
	// Typically, only used by admin.
	// +optional
	ProjectRef *KubernetesNameRef `json:"projectRef,omitempty"`

	// vipSubnetRef filters by the subnet on which the load balancer's address is allocated.
	// +optional
	VipSubnetRef *KubernetesNameRef `json:"vipSubnetRef,omitempty"`

	// vipNetworkRef filters by the network on which the load balancer's address is allocated.
	// +optional
	VipNetworkRef *KubernetesNameRef `json:"vipNetworkRef,omitempty"`

	// vipPortRef filters by the neutron port used for the VIP.
	// +optional
	VipPortRef *KubernetesNameRef `json:"vipPortRef,omitempty"`

	// availabilityZone is the availability zone in which to create the load balancer.
	// +kubebuilder:validation:MaxLength=255
	// +optional
	AvailabilityZone string `json:"availabilityZone,omitempty"`

	// provider filters by the name of the load balancer provider.
	// +kubebuilder:validation:MaxLength=255
	// +optional
	Provider string `json:"provider,omitempty"`

	// vipAddress filters by the IP address of the load balancer's VIP.
	// +kubebuilder:validation:MaxLength=64
	// +optional
	VipAddress string `json:"vipAddress,omitempty"`

	// tags is a list of tags to filter by. If specified, the resource must
	// have all of the tags specified to be included in the result.
	// +listType=set
	// +optional
	// +kubebuilder:validation:MaxItems:=64
	Tags []LoadBalancerTag `json:"tags,omitempty"`

	// tagsAny is a list of tags to filter by. If specified, the resource
	// must have at least one of the tags specified to be included in the
	// result.
	// +listType=set
	// +optional
	// +kubebuilder:validation:MaxItems:=64
	TagsAny []LoadBalancerTag `json:"tagsAny,omitempty"`

	// notTags is a list of tags to filter by. If specified, resources which
	// contain all of the given tags will be excluded from the result.
	// +listType=set
	// +optional
	// +kubebuilder:validation:MaxItems:=64
	NotTags []LoadBalancerTag `json:"notTags,omitempty"`

	// notTagsAny is a list of tags to filter by. If specified, resources
	// which contain any of the given tags will be excluded from the result.
	// +listType=set
	// +optional
	// +kubebuilder:validation:MaxItems:=64
	NotTagsAny []LoadBalancerTag `json:"notTagsAny,omitempty"`
}

// LoadBalancerResourceStatus represents the observed state of the resource.
type LoadBalancerResourceStatus struct {
	// name is a Human-readable name for the resource. Might not be unique.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Name string `json:"name,omitempty"`

	// description is a human-readable description for the resource.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Description string `json:"description,omitempty"`

	// vipSubnetID is the ID of the Subnet to which the resource is associated.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	VipSubnetID string `json:"vipSubnetID,omitempty"`

	// vipNetworkID is the ID of the Network to which the resource is associated.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	VipNetworkID string `json:"vipNetworkID,omitempty"`

	// vipPortID is the ID of the Port to which the resource is associated.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	VipPortID string `json:"vipPortID,omitempty"`

	// flavorID is the ID of the Flavor to which the resource is associated.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	FlavorID string `json:"flavorID,omitempty"`

	// projectID is the ID of the Project to which the resource is associated.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	ProjectID string `json:"projectID,omitempty"`

	// adminStateUp is the administrative state of the load balancer,
	// which is up (true) or down (false).
	// +optional
	AdminStateUp *bool `json:"adminStateUp,omitempty"`

	// tags is the list of tags on the resource.
	// +listType=atomic
	// +optional
	// +kubebuilder:validation:MaxItems:=64
	// +kubebuilder:validation:items:MaxLength=255
	Tags []string `json:"tags,omitempty"`

	// availabilityZone is the availability zone where the load balancer is located.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	AvailabilityZone string `json:"availabilityZone,omitempty"`

	// provisioningStatus is the provisioning status of the load balancer.
	// This value is ACTIVE, PENDING_CREATE or ERROR.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	ProvisioningStatus string `json:"provisioningStatus,omitempty"`

	// operatingStatus is the operating status of the load balancer,
	// such as ONLINE or OFFLINE.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	OperatingStatus string `json:"operatingStatus,omitempty"`

	// provider is the name of the load balancer provider.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Provider string `json:"provider,omitempty"`

	// vipAddress is the IP address of the load balancer's VIP.
	// +optional
	// +kubebuilder:validation:MaxLength=64
	VipAddress string `json:"vipAddress,omitempty"`
}
