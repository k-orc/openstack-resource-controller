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

// LimitResourceSpec contains the desired state of the resource.
// +kubebuilder:validation:XValidation:rule="has(self.projectRef) || has(self.domainRef)",message="either projectRef or domainRef must be specified"
// +kubebuilder:validation:XValidation:rule="!(has(self.projectRef) && has(self.domainRef))",message="projectRef and domainRef are mutually exclusive"
type LimitResourceSpec struct {
	// description is a human-readable description for the resource.
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=255
	// +optional
	Description *string `json:"description,omitempty"`

	// serviceRef is a reference to the ORC Service which this resource is associated with.
	// +required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="serviceRef is immutable"
	ServiceRef KubernetesNameRef `json:"serviceRef"`

	// projectRef is a reference to the ORC Project which this resource is associated with.
	// Either Domain ID or Project ID must be provided.
	// https://opendev.org/openstack/keystone/src/commit/30ef2ffa65a3486ef882f00538e20f2253c57d4c/keystone/limit/schema.py#L323-L340
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="projectRef is immutable"
	ProjectRef *KubernetesNameRef `json:"projectRef,omitempty"`

	// domainRef is a reference to the ORC Domain which this resource is associated with.
	// Either Domain ID or Project ID must be provided.
	// https://opendev.org/openstack/keystone/src/commit/30ef2ffa65a3486ef882f00538e20f2253c57d4c/keystone/limit/schema.py#L323-L340
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="domainRef is immutable"
	DomainRef *KubernetesNameRef `json:"domainRef,omitempty"`

	// resourceName is the name of the resource this limit is associated with.
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=255
	// +kubebuilder:validation:Pattern=`^[\S]+$`
	// +required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="resourceName is immutable"
	ResourceName string `json:"resourceName"`

	// resourceLimit is the override value of the limit.
	// +kubebuilder:validation:Minimum=-1
	// +required
	ResourceLimit int32 `json:"resourceLimit"`
}

// LimitFilter defines an existing resource by its properties
// +kubebuilder:validation:MinProperties:=1
type LimitFilter struct {
	// description of the existing resource
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=255
	// +optional
	Description *string `json:"description,omitempty"`

	// serviceRef is a reference to the ORC Service which this resource is associated with.
	// +optional
	ServiceRef *KubernetesNameRef `json:"serviceRef,omitempty"`

	// projectRef is a reference to the ORC Project which this resource is associated with.
	// +optional
	ProjectRef *KubernetesNameRef `json:"projectRef,omitempty"`

	// domainRef is a reference to the ORC Domain which this resource is associated with.
	// +optional
	DomainRef *KubernetesNameRef `json:"domainRef,omitempty"`

	// resourceName is the name of the resource this limit is associated with.
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=255
	// +kubebuilder:validation:Pattern=`^[\S]+$`
	// +optional
	ResourceName string `json:"resourceName,omitempty"`
}

// LimitResourceStatus represents the observed state of the resource.
type LimitResourceStatus struct {
	// description is a human-readable description for the resource.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Description string `json:"description,omitempty"`

	// serviceID is the ID of the Service to which the resource is associated.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	ServiceID string `json:"serviceID,omitempty"`

	// projectID is the ID of the Project to which the resource is associated.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	ProjectID string `json:"projectID,omitempty"`

	// domainID is the ID of the Domain to which the resource is associated.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	DomainID string `json:"domainID,omitempty"`

	// resourceLimit is the override value of the limit.
	// +optional
	ResourceLimit int32 `json:"resourceLimit,omitempty"`

	// resourceName is the name of the resource this limit is associated with.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	ResourceName string `json:"resourceName,omitempty"`
}
