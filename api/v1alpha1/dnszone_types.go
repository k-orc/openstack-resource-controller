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

// DNSZoneType is the type of a DNS zone - whether this ORC object owns the zone's data
// (PRIMARY) or replicates it from external master servers over AXFR (SECONDARY). This is
// Designate's own DNS-protocol zone transfer concept, unrelated to project ownership - see
// DNSZoneShare for sharing access to a zone with another OpenStack project.
// +kubebuilder:validation:Enum:=PRIMARY;SECONDARY
type DNSZoneType string

const (
	DNSZoneTypePrimary   DNSZoneType = "PRIMARY"
	DNSZoneTypeSecondary DNSZoneType = "SECONDARY"
)

// DNSZoneResourceSpec contains the desired state of the resource.
// +kubebuilder:validation:XValidation:rule="self.type == 'PRIMARY' ? has(self.email) : true",message="email is required for PRIMARY zones"
// +kubebuilder:validation:XValidation:rule="self.type == 'SECONDARY' ? (has(self.masters) && self.masters.size() > 0) : true",message="masters is required for SECONDARY zones"
// +kubebuilder:validation:XValidation:rule="self.type == 'PRIMARY' ? !has(self.masters) : true",message="masters must not be set for PRIMARY zones"
// +kubebuilder:validation:XValidation:rule="self.type == 'SECONDARY' ? !has(self.email) : true",message="email must not be set for SECONDARY zones"
type DNSZoneResourceSpec struct {
	// name is the name of the zone, e.g. "example.com.". Must end with a period, per Designate's
	// own convention. If not specified, the name of the ORC object is used.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="name is immutable"
	// +kubebuilder:validation:XValidation:rule="self.endsWith('.')",message="zone name must end with a period"
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	// email is the email address of the administrator for the zone. Required for PRIMARY zones,
	// not applicable to SECONDARY zones (Designate rejects both the missing-when-required and the
	// present-when-not-applicable cases - enforced here too via CEL rather than only server-side).
	// +kubebuilder:validation:Format:=email
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=255
	// +optional
	Email *string `json:"email,omitempty"`

	// description is a human-readable description for the resource.
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=255
	// +optional
	Description *string `json:"description,omitempty"`

	// ttl is the default Time To Live for the zone's recordsets, in seconds.
	// +kubebuilder:validation:Minimum:=1
	// +kubebuilder:validation:Maximum:=2147483647
	// +optional
	TTL *int32 `json:"ttl,omitempty"`

	// type is PRIMARY (this zone's data is authoritative here) or SECONDARY (replicated from
	// masters over AXFR). Immutable - Designate has no API to convert between the two in place.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="type is immutable"
	// +kubebuilder:default:="PRIMARY"
	// +optional
	Type DNSZoneType `json:"type,omitempty"`

	// masters are the master server IPs to transfer a SECONDARY zone's records from over AXFR.
	// Required when type is SECONDARY, must not be set when type is PRIMARY. Typed as IPvAny
	// (not a plain string) so malformed entries are rejected at admission rather than accepted
	// and only failing later against the real Designate API - a real gap found while reviewing
	// #825's draft implementation, which left this as an unvalidated []string.
	// +kubebuilder:validation:MaxItems:=32
	// +listType=set
	// +optional
	Masters []IPvAny `json:"masters,omitempty"`
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

	// type of the existing resource
	// +optional
	Type *DNSZoneType `json:"type,omitempty"`
}

// DNSZoneResourceStatus represents the observed state of the resource.
type DNSZoneResourceStatus struct {
	// name is the name of the zone, e.g. "example.com.".
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

	// ttl is the default Time To Live for the zone's recordsets, in seconds.
	// +optional
	TTL *int32 `json:"ttl,omitempty"`

	// type is PRIMARY or SECONDARY.
	// +kubebuilder:validation:MaxLength=255
	// +optional
	Type string `json:"type,omitempty"`

	// masters are the master server IPs this SECONDARY zone transfers its records from.
	// +kubebuilder:validation:MaxItems:=32
	// +kubebuilder:validation:items:MaxLength=1024
	// +listType=set
	// +optional
	Masters []string `json:"masters,omitempty"`

	// serial is the zone's current SOA serial number.
	// +optional
	Serial *int64 `json:"serial,omitempty"`

	// transferredAt is the last time this SECONDARY zone's records were refreshed from its
	// masters. Unset for PRIMARY zones.
	// +optional
	TransferredAt *metav1.Time `json:"transferredAt,omitempty"`

	// projectID is the ID of the OpenStack project that owns this zone. Not to be confused with
	// a DNSZoneShare's targetProjectID, which grants a *different* project access without
	// changing ownership.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	ProjectID string `json:"projectID,omitempty"`
}
