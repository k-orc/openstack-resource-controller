/*
Copyright 2025 The ORC Authors.

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

// LBPoolResourceSpec contains the desired state of the resource.
type LBPoolResourceSpec struct {
	// name will be the name of the created resource. If not specified, the
	// name of the ORC object will be used.
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	// description is a human-readable description for the resource.
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=255
	// +optional
	Description *string `json:"description,omitempty"`

	// loadBalancerRef is a reference to the ORC LoadBalancer which this resource is associated with.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="loadBalancerRef is immutable"
	LoadBalancerRef *KubernetesNameRef `json:"loadBalancerRef,omitempty"`

	// listenerRef is a reference to the ORC Listener which this resource is associated with.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="listenerRef is immutable"
	ListenerRef *KubernetesNameRef `json:"listenerRef,omitempty"`

	// projectRef is a reference to the ORC Project which this resource is associated with.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="projectRef is immutable"
	ProjectRef *KubernetesNameRef `json:"projectRef,omitempty"`

	// TODO(scaffolding): Add more types.
	// To see what is supported, you can take inspiration from the CreateOpts structure from
	// github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/pools
	//
	// Until you have implemented mutability for the field, you must add a CEL validation
	// preventing the field being modified:
	// `// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="<fieldname> is immutable"`
}

// LBPoolFilter defines an existing resource by its properties
// +kubebuilder:validation:MinProperties:=1
type LBPoolFilter struct {
	// name of the existing resource
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	// description of the existing resource
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=255
	// +optional
	Description *string `json:"description,omitempty"`

	// loadBalancerRef is a reference to the ORC LoadBalancer which this resource is associated with.
	// +optional
	LoadBalancerRef *KubernetesNameRef `json:"loadBalancerRef,omitempty"`

	// listenerRef is a reference to the ORC Listener which this resource is associated with.
	// +optional
	ListenerRef *KubernetesNameRef `json:"listenerRef,omitempty"`

	// projectRef is a reference to the ORC Project which this resource is associated with.
	// +optional
	ProjectRef *KubernetesNameRef `json:"projectRef,omitempty"`

	// TODO(scaffolding): Add more types.
	// To see what is supported, you can take inspiration from the ListOpts structure from
	// github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/pools
}

// LBPoolResourceStatus represents the observed state of the resource.
type LBPoolResourceStatus struct {
	// name is a Human-readable name for the resource. Might not be unique.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Name string `json:"name,omitempty"`

	// description is a human-readable description for the resource.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Description string `json:"description,omitempty"`

	// loadBalancerID is the ID of the LoadBalancer to which the resource is associated.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	LoadBalancerID string `json:"loadBalancerID,omitempty"`

	// listenerID is the ID of the Listener to which the resource is associated.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	ListenerID string `json:"listenerID,omitempty"`

	// projectID is the ID of the Project to which the resource is associated.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	ProjectID string `json:"projectID,omitempty"`

	// TODO(scaffolding): Add more types.
	// To see what is supported, you can take inspiration from the Pool structure from
	// github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/pools
}
