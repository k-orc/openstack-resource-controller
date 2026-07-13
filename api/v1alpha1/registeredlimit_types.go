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

// RegisteredLimitResourceSpec contains the desired state of the resource.
type RegisteredLimitResourceSpec struct {
	// description is a human-readable description for the resource.
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=255
	// +optional
	Description *string `json:"description,omitempty"`

	// serviceRef is a reference to the ORC Service which this resource is associated with.
	// +required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="serviceRef is immutable"
	ServiceRef KubernetesNameRef `json:"serviceRef,omitempty"`

	// resourceName is name of the resource to be limited.
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=255
	// +required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="resourceName is immutable"
	ResourceName string `json:"resourceName,omitempty"`

	// defaultLimit is limit of the specified resource in the given context.
	// +kubebuilder:validation:Minimum=-1
	// +kubebuilder:validation:Maximum=2147483647
	// +required
	DefaultLimit *int32 `json:"defaultLimit,omitempty"`
}

// RegisteredLimitFilter defines an existing resource by its properties
// +kubebuilder:validation:MinProperties:=1
type RegisteredLimitFilter struct {
	// description of the existing resource
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=255
	// +optional
	Description *string `json:"description,omitempty"`

	// serviceRef is a reference to the ORC Service which this resource is associated with.
	// +optional
	ServiceRef *KubernetesNameRef `json:"serviceRef,omitempty"`

	// resourceName is name of the resource to be limited.
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=255
	// +optional
	ResourceName *string `json:"resourceName,omitempty"`
}

// RegisteredLimitResourceStatus represents the observed state of the resource.
type RegisteredLimitResourceStatus struct {
	// description is a human-readable description for the resource.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Description string `json:"description,omitempty"`

	// resourceName is name of the resource to be limited.
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=255
	// +optional
	ResourceName string `json:"resourceName,omitempty"`

	// regionID is the ID of the region that contains the service endpoint.
	// +kubebuilder:validation:MaxLength:=1024
	// +optional
	RegionID string `json:"regionID,omitempty"`

	// serviceID is a reference to the ORC Service which this resource is associated with.
	// +kubebuilder:validation:MaxLength:=1024
	// +optional
	ServiceID string `json:"serviceID,omitempty"`

	// defaultLimit is limit of the specified resource in the given context.
	// +optional
	DefaultLimit int32 `json:"defaultLimit,omitempty"`
}
