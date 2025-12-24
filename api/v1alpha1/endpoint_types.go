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

// EndpointResourceSpec contains the desired state of the resource.
type EndpointResourceSpec struct {
	// name will be the name of the created resource. If not specified, the
	// name of the ORC object will be used.
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	// enabled indicates whether the endpoint is enabled or not.
	// +kubebuilder:default:=true
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// interface indicates the visibility of the endpoint.
	// +kubebuilder:validation:Enum:=admin;internal;public
	// +required
	Interface string `json:"interface,omitempty"`

	// url is the endpoint URL.
	// +kubebuilder:validation:MaxLength=1024
	// +required
	URL string `json:"url"`

	// serviceRef is a reference to the ORC Service which this resource is associated with.
	// +required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="serviceRef is immutable"
	ServiceRef KubernetesNameRef `json:"serviceRef,omitempty"`
}

// EndpointFilter defines an existing resource by its properties
// +kubebuilder:validation:MinProperties:=1
type EndpointFilter struct {
	// interface of the existing endpoint.
	// +kubebuilder:validation:Enum:=admin;internal;public
	// +optional
	Interface string `json:"interface,omitempty"`

	// serviceRef is a reference to the ORC Service which this resource is associated with.
	// +optional
	ServiceRef *KubernetesNameRef `json:"serviceRef,omitempty"`

	// url is the URL of the existing endpoint.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	URL string `json:"url,omitempty"`
}

// EndpointResourceStatus represents the observed state of the resource.
type EndpointResourceStatus struct {
	// name is a Human-readable name for the resource. Might not be unique.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Name string `json:"name,omitempty"`

	// enabled indicates whether the endpoint is enabled or not.
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// interface indicates the visibility of the endpoint.
	// +kubebuilder:validation:Enum:=admin;internal;public
	// +optional
	Interface string `json:"interface,omitempty"`

	// url is the endpoint URL.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	URL string `json:"url,omitempty"`

	// serviceID is the ID of the Service to which the resource is associated.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	ServiceID string `json:"serviceID,omitempty"`
}
