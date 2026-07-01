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

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:validation:Enum:=PRIMARY;SECONDARY
type DNSZoneType string

const (
	DNSZoneTypePrimary   DNSZoneType = "PRIMARY"
	DNSZoneTypeSecondary DNSZoneType = "SECONDARY"
)

// DNSZoneResourceSpec contains the desired state of the resource.
// +kubebuilder:validation:XValidation:rule="self.type == 'PRIMARY' ? (has(self.email) && self.email != \"\") : true",message="email is required for PRIMARY zones"
// +kubebuilder:validation:XValidation:rule="self.type == 'SECONDARY' ? (has(self.masters) && self.masters.size() > 0) : true",message="masters: required when type is SECONDARY"
// +kubebuilder:validation:XValidation:rule="self.type == 'PRIMARY' ? !has(self.masters) : true",message="masters: must not be specified when type is PRIMARY"
// +kubebuilder:validation:XValidation:rule="self.type == 'SECONDARY' ? !has(self.email) : true",message="email: must not be specified when type is SECONDARY"
type DNSZoneResourceSpec struct {
	// name will be the name of the created resource. If not specified, the
	// name of the ORC object will be used.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="name is immutable"
	// +kubebuilder:validation:XValidation:rule="self.endsWith('.')",message="zone name must end with a period"
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	// email is the email address of the administrator for the zone.
	// +kubebuilder:validation:Format:=email
	// +kubebuilder:validation:MaxLength:=255
	// +optional
	Email *string `json:"email,omitempty"`

	// description is a human-readable description for the resource.
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=255
	// +optional
	Description *string `json:"description,omitempty"`

	// ttl is the Time To Live for the zone in seconds.
	// +kubebuilder:validation:Minimum:=1
	// +kubebuilder:validation:Maximum:=2147483647
	// +optional
	TTL *int32 `json:"ttl,omitempty"`

	// type is the type of the zone.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="type is immutable"
	// +kubebuilder:default:="PRIMARY"
	// +optional
	Type DNSZoneType `json:"type,omitempty"`

	// masters specifies zone masters if this is a secondary zone.
	// +kubebuilder:validation:MaxItems:=32
	// +kubebuilder:validation:items:MaxLength:=255
	// +listType=atomic
	// +optional
	Masters []string `json:"masters,omitempty"`
}

// DNSZoneFilter defines an existing resource by its properties
// +kubebuilder:validation:MinProperties:=1
type DNSZoneFilter struct {
	// name of the existing resource
	// +kubebuilder:validation:XValidation:rule="self.endsWith('.')",message="name must end with a period"
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	// email of the existing resource
	// +kubebuilder:validation:Format:=email
	// +kubebuilder:validation:MaxLength:=255
	// +optional
	Email *string `json:"email,omitempty"`

	// description of the existing resource
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=255
	// +optional
	Description *string `json:"description,omitempty"`

	// ttl of the existing resource
	// +kubebuilder:validation:Minimum:=1
	// +kubebuilder:validation:Maximum:=2147483647
	// +optional
	TTL *int32 `json:"ttl,omitempty"`

	// type of the existing resource
	// +optional
	Type *DNSZoneType `json:"type,omitempty"`

	// masters of the existing resource
	// +kubebuilder:validation:MaxItems:=32
	// +kubebuilder:validation:items:MaxLength:=255
	// +listType=atomic
	// +optional
	Masters []string `json:"masters,omitempty"`
}

// DNSZoneResourceStatus represents the observed state of the resource.
type DNSZoneResourceStatus struct {
	// name is a Human-readable name for the resource. Might not be unique.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Name string `json:"name,omitempty"`

	// email is the email contact of the zone.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Email string `json:"email,omitempty"`

	// description is a human-readable description for the resource.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Description string `json:"description,omitempty"`

	// ttl is the Time to Live for the zone in seconds.
	// +optional
	TTL *int32 `json:"ttl,omitempty"`

	// type is the type of the zone.
	// +kubebuilder:validation:MaxLength=255
	// +optional
	Type string `json:"type,omitempty"`

	// masters specifies zone masters if this is a secondary zone.
	// +kubebuilder:validation:MaxItems:=32
	// +kubebuilder:validation:items:MaxLength:=255
	// +listType=atomic
	// +optional
	Masters []string `json:"masters,omitempty"`

	// transferredAt is the last time an update was retrieved from the master servers.
	// +optional
	TransferredAt *metav1.Time `json:"transferredAt,omitempty"`

	// status is the status of the resource.
	// +kubebuilder:validation:MaxLength=255
	// +optional
	Status string `json:"status,omitempty"`
}
