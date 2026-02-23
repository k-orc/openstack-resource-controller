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

// ShareNetworkResourceSpec contains the desired state of the resource.
type ShareNetworkResourceSpec struct {
	// name will be the name of the created resource. If not specified, the
	// name of the ORC object will be used.
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	// description is a human-readable description for the resource.
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=255
	// +optional
	Description *string `json:"description,omitempty"`

	// networkRef is a reference to the ORC Network which this resource is associated with.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="networkRef is immutable"
	NetworkRef *KubernetesNameRef `json:"networkRef,omitempty"`

	// subnetRef is a reference to the ORC Subnet which this resource is associated with.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="subnetRef is immutable"
	SubnetRef *KubernetesNameRef `json:"subnetRef,omitempty"`

	// TODO(scaffolding): Add more types.
	// To see what is supported, you can take inspiration from the CreateOpts structure from
	// github.com/gophercloud/gophercloud/v2/openstack/sharedfilesystems/v2/sharenetworks
	//
	// Until you have implemented mutability for the field, you must add a CEL validation
	// preventing the field being modified:
	// `// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="<fieldname> is immutable"`
}

// ShareNetworkFilter defines an existing resource by its properties
// +kubebuilder:validation:MinProperties:=1
type ShareNetworkFilter struct {
	// name of the existing resource
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	// description of the existing resource
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=255
	// +optional
	Description *string `json:"description,omitempty"`

	// TODO(scaffolding): Add more types.
	// To see what is supported, you can take inspiration from the ListOpts structure from
	// github.com/gophercloud/gophercloud/v2/openstack/sharedfilesystems/v2/sharenetworks
}

// ShareNetworkResourceStatus represents the observed state of the resource.
type ShareNetworkResourceStatus struct {
	// name is a Human-readable name for the resource. Might not be unique.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Name string `json:"name,omitempty"`

	// description is a human-readable description for the resource.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Description string `json:"description,omitempty"`

	// networkID is the ID of the Network to which the resource is associated.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	NetworkID string `json:"networkID,omitempty"`

	// subnetID is the ID of the Subnet to which the resource is associated.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	SubnetID string `json:"subnetID,omitempty"`

	// TODO(scaffolding): Add more types.
	// To see what is supported, you can take inspiration from the ShareNetwork structure from
	// github.com/gophercloud/gophercloud/v2/openstack/sharedfilesystems/v2/sharenetworks
}
