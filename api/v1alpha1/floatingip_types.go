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

// FloatingIPFilter specifies a query to select an OpenStack floatingip. At least one property must be set.
// +kubebuilder:validation:MinProperties:=1
type FloatingIPFilter struct {
	// floatingIP is the floatingip address.
	// +optional
	FloatingIP *IPvAny `json:"floatingIP,omitempty"`

	// description of the existing resource
	// +optional
	Description *NeutronDescription `json:"description,omitempty"`

	// networkRef is a reference to the ORC Network which this resource is associated with.l
	// +optional
	NetworkRef KubernetesNameRef `json:"networkRef"`

	// portRef is a reference to the ORC Port which this resource is associated with.
	// +optional
	PortRef KubernetesNameRef `json:"portRef,omitempty"`

	// status is the status of the floatingip.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Status string `json:"status,omitempty"`

	FilterByNeutronTags `json:",inline"`
}

// FloatingIPResourceSpec contains the desired state of a floating IP
// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="FloatingIPResourceSpec is immutable"
type FloatingIPResourceSpec struct {
	// description is a human-readable description for the resource.
	// +optional
	Description *NeutronDescription `json:"description,omitempty"`

	// tags is a list of tags which will be applied to the floatingip.
	// +kubebuilder:validation:MaxItems:=32
	// +listType=set
	// +optional
	Tags []NeutronTag `json:"tags,omitempty"`

	// networkRef references the network to which the floatingip is associated.
	// +required
	NetworkRef KubernetesNameRef `json:"networkRef"`

	// subnetRef references the subnet to which the floatingip is associated.
	// +optional
	SubnetRef *KubernetesNameRef `json:"subnetRef,omitempty"`

	// floatingIP is the IP that will be assigned to the floatingip. If not set, it will
	// be assigned automatically.
	// +optional
	FloatingIP *IPvAny `json:"floatingIP"`

	// portRef is a reference to the ORC Port which this resource is associated with.
	// +optional
	PortRef *KubernetesNameRef `json:"portRef,omitempty"`

	// fixedIP is the IP address of the port to which the floatingip is associated.
	// +optional
	FixedIP *IPvAny `json:"fixedIP,omitempty"`
}

type FloatingIPResourceStatus struct {
	// description is a human-readable description for the resource.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Description string `json:"description,omitempty"`

	// floatingNetworkID is the ID of the network to which the floatingip is associated.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	FloatingNetworkID string `json:"floatingNetworkID,omitempty"`

	// floatingIP is the IP address of the floatingip.
	// +optional
	FloatingIP *IPvAny `json:"floatingIP,omitempty"`

	// portID is the ID of the port to which the floatingip is associated.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	PortID *string `json:"portID,omitempty"`

	// fixedIP is the IP address of the port to which the floatingip is associated.
	// +optional
	FixedIP *IPvAny `json:"fixedIP,omitempty"`

	// tenantID is the project owner of the resource.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	TenantID string `json:"tenantID,omitempty"`

	// projectID is the project owner of the resource.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	ProjectID string `json:"projectID,omitempty"`

	// status indicates the current status of the resource.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Status string `json:"status,omitempty"`

	// routerID is the ID of the router to which the floatingip is associated.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	RouterID string `json:"routerID,omitempty"`

	// tags is the list of tags on the resource.
	// +kubebuilder:validation:MaxItems:=32
	// +kubebuilder:validation:items:MaxLength=1024
	// +listType=atomic
	// +optional
	Tags []string `json:"tags,omitempty"`

	NeutronStatusMetadata `json:",inline"`
}
