/*
Copyright The ORC Authors.

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

// SubnetPoolResourceSpec contains the desired state of the resource.
//
// +kubebuilder:validation:XValidation:rule="self.minPrefixLength <= self.maxPrefixLength",message="minPrefixLength must be less than or equal to maxPrefixLength"
// +kubebuilder:validation:XValidation:rule="!has(self.defaultPrefixLength) || (self.defaultPrefixLength >= self.minPrefixLength && self.defaultPrefixLength <= self.maxPrefixLength)",message="defaultPrefixLength must be between minPrefixLength and maxPrefixLength"
type SubnetPoolResourceSpec struct {
	// name will be the name of the created resource. If not specified, the
	// name of the ORC object will be used.
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	// description is a human-readable description for the resource.
	// +optional
	Description *NeutronDescription `json:"description,omitempty"`

	// projectRef is a reference to the ORC Project which this resource is associated with.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="projectRef is immutable"
	// +orc:kustomize:ref=Project
	ProjectRef *KubernetesNameRef `json:"projectRef,omitempty"`

	// addressScopeRef is a reference to the ORC AddressScope which this resource is associated with.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="addressScopeRef is immutable"
	// +orc:kustomize:ref=AddressScope
	AddressScopeRef *KubernetesNameRef `json:"addressScopeRef,omitempty"`

	// prefixes is the list of subnet prefixes to assign to the subnet
	// pool. The API merges adjacent prefixes and treats them as a
	// single prefix. Each subnet prefix must be unique across all
	// subnet pools associated with the address scope.
	// +kubebuilder:validation:MinItems:=1
	// +kubebuilder:validation:MaxItems:=64
	// +listType=set
	// +required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="prefixes is immutable"
	Prefixes []CIDR `json:"prefixes,omitempty"`

	// minPrefixLength is the smallest prefix that can be allocated
	// from a subnet pool. For IPv4 subnet pools, default is 8. For
	// IPv6 subnet pools, default is 64. Must be less than or equal to
	// maxPrefixLength.
	// +kubebuilder:validation:Minimum=1
	// +required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="minPrefixLength is immutable"
	MinPrefixLength int32 `json:"minPrefixLength,omitempty"`

	// maxPrefixLength is the maximum prefix size that can be
	// allocated from the subnet pool. For IPv4 subnet pools, default
	// is 32. For IPv6 subnet pools, default is 128. Must be greater
	// than or equal to minPrefixLength.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=128
	// +required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="maxPrefixLength is immutable"
	MaxPrefixLength int32 `json:"maxPrefixLength,omitempty"`

	// shared indicates whether this resource is shared across all projects.
	// By default, it is false, and only administrative users can
	// change this value.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="shared is immutable"
	Shared *bool `json:"shared,omitempty"`

	// defaultPrefixLength is the size of the prefix to allocate when
	// the CIDR or prefixlen attributes are omitted when you create
	// the subnet. Default is minPrefixLength. Must be between
	// minPrefixLength and maxPrefixLength, inclusive.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="defaultPrefixLength is immutable"
	DefaultPrefixLength int32 `json:"defaultPrefixLength,omitempty"`

	// isDefault defines whether the subnet pool is the default pool
	// or not.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="isDefault is immutable"
	IsDefault *bool `json:"isDefault,omitempty"`

	// tags is a list of tags which will be applied to the subnet pool.
	// +kubebuilder:validation:MaxItems:=64
	// +listType=set
	// +optional
	Tags []NeutronTag `json:"tags,omitempty"`
}

// SubnetPoolFilter defines an existing resource by its properties
// +kubebuilder:validation:MinProperties:=1
type SubnetPoolFilter struct {
	// name of the existing resource
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	// description of the existing resource
	// +optional
	Description *NeutronDescription `json:"description,omitempty"`

	// projectRef is a reference to the ORC Project which this resource is associated with.
	// +optional
	// +orc:kustomize:ref=Project
	ProjectRef *KubernetesNameRef `json:"projectRef,omitempty"`

	// addressScopeRef is a reference to the ORC AddressScope which this resource is associated with.
	// +optional
	// +orc:kustomize:ref=AddressScope
	AddressScopeRef *KubernetesNameRef `json:"addressScopeRef,omitempty"`

	// minPrefixLength allows filtering the subnet pool list result by
	// the smallest prefix that can be allocated from a subnet pool.
	// +optional
	MinPrefixLength int32 `json:"minPrefixLength,omitempty"`

	// maxPrefixLength allows filtering the subnet pool list result by
	// the maximum prefix size that can be allocated from the subnet
	// pool.
	// +optional
	MaxPrefixLength int32 `json:"maxPrefixLength,omitempty"`

	// ipVersion is the IP protocol version. It can be either 4 or 6
	// +optional
	IPVersion *IPVersion `json:"ipVersion,omitempty"`

	// shared allows filtering the list result based on whether the
	// resource is shared across all projects. This field is
	// admin-only.
	// +optional
	Shared *bool `json:"shared,omitempty"`

	// defaultPrefixLength allows filtering the subnet pool list
	// result by the size of the prefix to allocate when the cidr or
	// prefixlen attributes are omitted when you create the subnet.
	// +optional
	DefaultPrefixLength int32 `json:"defaultPrefixLength,omitempty"`

	// isDefault allows filtering the subnet pool list result based on
	// whether it is a default pool or not.
	// +optional
	IsDefault *bool `json:"isDefault,omitempty"`

	FilterByNeutronTags `json:",inline"`
}

// SubnetPoolResourceStatus represents the observed state of the resource.
type SubnetPoolResourceStatus struct {
	// name is a Human-readable name for the resource. Might not be unique.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Name string `json:"name,omitempty"`

	// description is a human-readable description for the resource.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Description string `json:"description,omitempty"`

	// projectID is the ID of the Project to which the resource is associated.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	ProjectID string `json:"projectID,omitempty"`

	// addressScopeID is the ID of the AddressScope to which the resource is associated.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	AddressScopeID string `json:"addressScopeID,omitempty"`

	// prefixes is a list of prefixes assigned to the SubnetPool.
	// +listType=atomic
	// +kubebuilder:validation:MaxItems:=64
	// +kubebuilder:validation:items:MaxLength:=64
	// +optional
	Prefixes []string `json:"prefixes,omitempty"`

	// minPrefixLength is the smallest prefix that can be allocated
	// from the subnet pool.
	// +optional
	MinPrefixLength int32 `json:"minPrefixLength,omitempty"`

	// maxPrefixLength is the maximum prefix size that can be
	// allocated from the subnet pool.
	// +optional
	MaxPrefixLength int32 `json:"maxPrefixLength,omitempty"`

	// defaultPrefixLength is the size of the prefix to allocate when
	// the cidr or prefixlen attributes are omitted when you create
	// the subnet.
	// +optional
	DefaultPrefixLength int32 `json:"defaultPrefixLength,omitempty"`

	// isDefault indicates whether the SubnetPool is the default pool
	// when creating subnets.
	// +optional
	IsDefault *bool `json:"isDefault,omitempty"`

	// shared indicates whether the SubnetPool is shared across all projects.
	// +optional
	Shared *bool `json:"shared,omitempty"`

	// ipVersion is the IP protocol version. It can be either 4 or 6
	// +optional
	IPVersion int32 `json:"ipVersion,omitempty"`

	// tags is the list of tags on the resource.
	// +kubebuilder:validation:MaxItems=64
	// +kubebuilder:validation:items:MaxLength=1024
	// +listType=atomic
	// +optional
	Tags []string `json:"tags,omitempty"`

	NeutronStatusMetadata `json:",inline"`
}
