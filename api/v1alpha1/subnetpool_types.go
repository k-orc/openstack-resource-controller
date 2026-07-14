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
type SubnetPoolResourceSpec struct {
	// name will be the name of the created resource. If not specified, the
	// name of the ORC object will be used.
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	// description is a human-readable description for the resource.
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=255
	// +optional
	Description *string `json:"description,omitempty"`

	// projectRef is a reference to the ORC Project which this resource is associated with.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="projectRef is immutable"
	ProjectRef *KubernetesNameRef `json:"projectRef,omitempty"`

	// addressScopeRef is a reference to the ORC AddressScope which this resource is associated with.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="addressScopeRef is immutable"
	AddressScopeRef *KubernetesNameRef `json:"addressScopeRef,omitempty"`

	// defaultQuota is a per-project quota on the prefix space that
	// can be allocated from the SubnetPool for project subnets. Its
	// value represents the number of absolute addresses any given
	// project is allowed to consume from the pool.
	// +optional
	// TODO(winiciusallan): Add this field when fixing bug in gophercloud
	// DefaultQuota int32 `json:"defaultQuota,omitempty"`

	// prefixes is the list of subnet prefixes to assign to the subnet
	// pool. The API merges adjacent prefixes and treats them as a
	// single prefix. Each subnet prefix must be unique across all
	// subnet pools associated with address scope.
	// +kubebuilder:validation:MinItems:=1
	// +kubebuilder:validation:MaxItems:=64
	// +listType=set
	// +required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="prefixes is immutable"
	Prefixes []CIDR `json:"prefixes,omitempty"`

	// minPrefixLength is the smallest prefix that can be allocated
	// from a subnet pool. For IPv4 subnet pools, default is 8. For
	// IPv6 subnet pools, default is 64.
	// +kubebuilder:validation:Minimum=1
	// +required
	MinPrefixLength int32 `json:"minPrefixLength,omitempty"`

	// maxPrefixLength is the maximum prefix size that can be allocated
	// from the subnet pool. For IPv4 subnet pools, default is 32. For
	// IPv6 subnet pools, default is 128.
	// +kubebuilder:validation:Minimum=1
	// +required
	MaxPrefixLength int32 `json:"maxPrefixLength,omitempty"`

	// shared indicates whether this resource is shared across all projects.
	// By default, it is false, and only administrative users can
	// change this value.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="shared is immutable"
	Shared *bool `json:"shared,omitempty"`

	// defaultPrefixLength is the size of the prefix to allocate when
	// the cidr or prefixlen attributes are omitted when you create
	// the subnet. Default is MinPrefixLength.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="defaultPrefixLength is immutable"
	// +optional
	DefaultPrefixLength int32 `json:"defaultPrefixLength,omitempty"`

	// isDefault defines whether the subnetpool is default pool or
	// not.
	// +optional
	IsDefault *bool `json:"isDefault,omitempty"`
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
	ProjectRef *KubernetesNameRef `json:"projectRef,omitempty"`

	// addressScopeRef is a reference to the ORC AddressScope which this resource is associated with.
	// +optional
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
	IPVersion IPVersion `json:"ipVersion,omitempty"`

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
	// if it is a default pool or not.
	// +optional
	IsDefault *bool `json:"isDefault,omitempty"`

	// revisionNumber allows filtering the list result by the revision
	// number of the resource.
	// +optional
	RevisionNumber int64 `json:"revisionNumber,omitempty"`

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

	// prefixes is a list of prefixes to assign to the SubnetPool.
	// +listType=atomic
	// +kubebuilder:validation:MaxItems:=1024
	// +kubebuilder:validation:items:MaxLength:=64
	// +optional
	Prefixes []string `json:"prefixes,omitempty"`

	// defaultQuota is a per-project quota on the prefix space that
	// can be allocated from the SubnetPool for project subnets.
	// +optional
	DefaultQuota int32 `json:"defaultQuota,omitempty"`

	// minPrefixLength is the smallest prefix that can be allocated
	// from a subnet pool.
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
	IsDefault bool `json:"isDefault,omitempty"`

	// shared indicates whether the SubnetPool is shared across all projects.
	// +optional
	Shared bool `json:"shared,omitempty"`

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
