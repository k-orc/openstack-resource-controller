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

// ResourceClassResourceSpec contains the desired state of the resource.
// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="ResourceClassResourceSpec is immutable"
type ResourceClassResourceSpec struct {
	// name will be the name of the created resource.
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=255
	// +kubebuilder:validation:Pattern=`^CUSTOM_[A-Z0-9_]+$`
	// +optional
	Name *string `json:"name,omitempty"`
}

// ResourceClassFilter defines an existing resource by its properties
// +kubebuilder:validation:MinProperties:=1
type ResourceClassFilter struct {
	// name of the existing resource
	// +optional
	Name *OpenStackName `json:"name,omitempty"`
}

// ResourceClassResourceStatus represents the observed state of the resource.
type ResourceClassResourceStatus struct {
	// name is a Human-readable name for the resource. Might not be unique.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Name string `json:"name,omitempty"`
}
