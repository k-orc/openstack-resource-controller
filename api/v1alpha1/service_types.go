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

// ServiceResourceSpec contains the desired state of the resource.
type ServiceResourceSpec struct {
	// type indicates which resource the service is responsible for.
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=255
	// +optional
	Type *string `json:"type,omitempty"`

	// enabled indicates whether the service is enabled or not.
	// +kubebuilder:default=true
	Enabled *bool `json:"enabled,omitempty"`

	// extra indicates key-value information about the service.
	Extra map[string]any `json:"extra,omitempty"`
}

// ServiceFilter defines an existing resource by its properties
// +kubebuilder:validation:MinProperties:=1
type ServiceFilter struct {
	// name of the existing resource
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	// type of the existing resource
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=255
	// +optional
	Type *string `json:"type,omitempty"`
}

// ServiceResourceStatus represents the observed state of the resource.
type ServiceResourceStatus struct {
	// description is a human-readable description for the resource.
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=255
	// +optional
	Type *string `json:"type,omitempty"`

	// enabled indicates whether the service is enabled or not.
	// +kubebuilder:default=true
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// extra indicates key-value information about the service.
	// +optional
	Extra map[string]any `json:"extra,omitempty"`
}
