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

// LoadBalancerFilter specifies a query to select an OpenStack load balancer. At least one property must be set.
// +kubebuilder:validation:MinProperties:=1
type LoadBalancerFilter struct {
	// name of the existing resource
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	// description of the existing resource
	// +optional
	Description *NeutronDescription `json:"description,omitempty"`

	// vipSubnetRef is a reference to the ORC Subnet on which the VIP is allocated.
	// +optional
	VIPSubnetRef *KubernetesNameRef `json:"vipSubnetRef,omitempty"`

	// vipNetworkRef is a reference to the ORC Network on which the VIP is allocated.
	// +optional
	VIPNetworkRef *KubernetesNameRef `json:"vipNetworkRef,omitempty"`

	// projectRef is a reference to the ORC Project this resource is associated with.
	// Typically, only used by admin.
	// +optional
	ProjectRef *KubernetesNameRef `json:"projectRef,omitempty"`

	FilterByNeutronTags `json:",inline"`
}

// LoadBalancerResourceSpec contains the desired state of a load balancer.
// +kubebuilder:validation:XValidation:rule="[has(self.subnetRef), has(self.networkRef), has(self.vipPortRef)].exists_one(x, x)",message="exactly one of subnetRef, networkRef, or vipPortRef must be set"
type LoadBalancerResourceSpec struct {
	// name is a human-readable name of the load balancer. If not set, the
	// object's name will be used.
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	// description is a human-readable description for the resource.
	// +optional
	Description *NeutronDescription `json:"description,omitempty"`

	// subnetRef is a reference to the ORC Subnet on which to allocate the VIP address.
	// Mutually exclusive with networkRef and vipPortRef.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="subnetRef is immutable"
	SubnetRef *KubernetesNameRef `json:"subnetRef,omitempty"`

	// networkRef is a reference to the ORC Network on which to allocate the VIP address.
	// Mutually exclusive with subnetRef and vipPortRef.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="networkRef is immutable"
	NetworkRef *KubernetesNameRef `json:"networkRef,omitempty"`

	// vipPortRef is a reference to an ORC Port to use as the VIP port for the load balancer.
	// Mutually exclusive with subnetRef and networkRef.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="vipPortRef is immutable"
	VIPPortRef *KubernetesNameRef `json:"vipPortRef,omitempty"`

	// vipAddress is the IP address of the VIP. If not specified, an available
	// IP from the specified subnet or network will be allocated.
	// +optional
	VIPAddress *IPvAny `json:"vipAddress,omitempty"`

	// adminStateUp is the administrative state of the load balancer,
	// which is up (true) or down (false). Default is true.
	// +optional
	AdminStateUp *bool `json:"adminStateUp,omitempty"`

	// flavor is the name of the flavor for the load balancer. This is
	// provider-specific.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="flavor is immutable"
	Flavor *OpenStackName `json:"flavor,omitempty"`

	// provider is the name of the provider driver to use for the load
	// balancer. If not specified, the default provider will be used.
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=255
	// +optional
	Provider *string `json:"provider,omitempty"`

	// projectRef is a reference to the ORC Project this resource is associated with.
	// Typically, only used by admin.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="projectRef is immutable"
	ProjectRef *KubernetesNameRef `json:"projectRef,omitempty"`

	// tags is a list of tags which will be applied to the load balancer.
	// +kubebuilder:validation:MaxItems:=64
	// +listType=set
	// +optional
	Tags []NeutronTag `json:"tags,omitempty"`
}

// LoadBalancerResourceStatus defines the observed state of a load balancer.
type LoadBalancerResourceStatus struct {
	// name is the human-readable name of the resource. Might not be unique.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Name string `json:"name,omitempty"`

	// description is a human-readable description for the resource.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Description string `json:"description,omitempty"`

	// adminStateUp is the administrative state of the load balancer,
	// which is up (true) or down (false).
	// +optional
	AdminStateUp *bool `json:"adminStateUp,omitempty"`

	// provisioningStatus is the provisioning status of the load balancer.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	ProvisioningStatus string `json:"provisioningStatus,omitempty"`

	// operatingStatus is the operating status of the load balancer.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	OperatingStatus string `json:"operatingStatus,omitempty"`

	// vipAddress is the IP address of the VIP.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	VIPAddress string `json:"vipAddress,omitempty"`

	// vipSubnetID is the ID of the subnet on which the VIP is allocated.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	VIPSubnetID string `json:"vipSubnetID,omitempty"`

	// vipNetworkID is the ID of the network on which the VIP is allocated.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	VIPNetworkID string `json:"vipNetworkID,omitempty"`

	// vipPortID is the ID of the port for the VIP.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	VIPPortID string `json:"vipPortID,omitempty"`

	// provider is the name of the provider driver for the load balancer.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Provider string `json:"provider,omitempty"`

	// flavorID is the ID of the flavor for the load balancer.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	FlavorID string `json:"flavorID,omitempty"`

	// projectID is the project owner of the resource.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	ProjectID string `json:"projectID,omitempty"`

	// tags is the list of tags on the resource.
	// +kubebuilder:validation:MaxItems:=64
	// +kubebuilder:validation:items:MaxLength=1024
	// +listType=atomic
	// +optional
	Tags []string `json:"tags,omitempty"`

	NeutronStatusMetadata `json:",inline"`
}
