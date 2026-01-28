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

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// UserResourceSpec contains the desired state of the resource.
type UserResourceSpec struct {
	// name will be the name of the created resource. If not specified, the
	// name of the ORC object will be used.
	Name *KeystoneName `json:"name,omitempty"`

	// description is a human-readable description for the resource.
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=255
	// +optional
	Description *string `json:"description,omitempty"`

	// defaultProjectID is the ID of the default project for the user
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=255
	// +optional
	DefaultProjectID *string `json:"defaultProjectID,omitempty"`

	// domainRef is a reference to the ORC Domain which this resource is associated with.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="domainRef is immutable"
	DomainRef *KubernetesNameRef `json:"domainRef,omitempty"`

	// password is created by the user
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=255
	// +optional
	Password *string `json:"password,omitempty"`

	// enabled indicates whether the user is enabled or not
	// +kubebuilder:default=true
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
}

// UserFilter defines an existing resource by its properties
// +kubebuilder:validation:MinProperties:=1
type UserFilter struct {
	// name of the existing resource
	// +optional
	Name *KeystoneName `json:"name,omitempty"`

	// domainRef is a reference to the ORC Domain which this resource is associated with.
	// +optional
	DomainRef *KubernetesNameRef `json:"domainRef,omitempty"`
}

// UserResourceStatus represents the observed state of the resource.
type UserResourceStatus struct {
	// name is a Human-readable name for the resource. Might not be unique.
	// +kubebuilder:validation:MaxLength=1024
	Name string `json:"name,omitempty"`

	// description is a human-readable description for the resource.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Description string `json:"description,omitempty"`

	// defaultProjectID is the ID of the default project for the user
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=255
	// +optional
	DefaultProjectID string `json:"defaultProjectID,omitempty"`

	// domainID is the ID of the Domain to which the resource is associated.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	DomainID string `json:"domainID,omitempty"`

	// enabled indicates whether the user is enabled or not
	// +kubebuilder:default=true
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// passwordExpiresAt shows the date and time when the user's password will expire.
	// The date and time stamp format follows ISO 8601
	// Example: 0000-00-00T00:00:00.000000
	// +optional
	PasswordExpiresAt *metav1.Time `json:"passwordExpiresAt,omitempty"`
}
