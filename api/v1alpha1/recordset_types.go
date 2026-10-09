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

// RecordSetType is the RRTYPE of a DNS recordset.
// +kubebuilder:validation:Enum:=A;AAAA;CNAME;MX;NS;PTR;SPF;SRV;SSHFP;TXT;CAA
type RecordSetType string

// RecordSetResourceSpec contains the desired state of the resource.
type RecordSetResourceSpec struct {
	// name is the name of the recordset, e.g. "www.example.com.". Must end with a period, per
	// Designate's own convention.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="name is immutable"
	// +required
	Name *DesignateFQDN `json:"name,omitempty"` //nolint:kubeapilinter // always populated; pointer kept so the shared getResourceName helper compiles

	// description is a human-readable description for the resource.
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=255
	// +optional
	Description *string `json:"description,omitempty"`

	// zoneRef is a reference to the ORC DNSZone this recordset belongs to.
	// +required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="zoneRef is immutable"
	// +orc:kustomize:ref=DNSZone
	ZoneRef KubernetesNameRef `json:"zoneRef,omitempty"`

	// type is the RRTYPE of the recordset, e.g. A, CNAME, TXT. Immutable - Designate has no
	// update path for a recordset's type, only its records/ttl/description.
	// +required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="type is immutable"
	Type RecordSetType `json:"type,omitempty"`

	// records are the record data for this recordset, in Designate's own format for the given
	// type (e.g. an IP address for A/AAAA, a hostname for CNAME/MX/NS, free text for TXT). Not
	// further validated here - record data syntax varies by type and Designate's own API is the
	// source of truth for what's acceptable.
	// +required
	// +kubebuilder:validation:MinItems:=1
	// +kubebuilder:validation:MaxItems:=64
	// +kubebuilder:validation:items:MaxLength:=4096
	// +listType=set
	Records []string `json:"records,omitempty"`

	// ttl is the Time To Live for the recordset, in seconds. If not specified, the zone's own
	// default TTL applies.
	// +kubebuilder:validation:Minimum:=1
	// +kubebuilder:validation:Maximum:=2147483647
	// +optional
	TTL *int32 `json:"ttl,omitempty"`
}

// RecordSetFilter defines an existing resource by its properties
// +kubebuilder:validation:MinProperties:=2
type RecordSetFilter struct {
	// zoneRef is a reference to the ORC DNSZone to look for the recordset under - required
	// because every Designate recordset operation, including list, is scoped to a specific zone
	// (see RecordSetResourceSpec.zoneRef's doc comment for the same constraint on the managed
	// path).
	// +required
	// +orc:kustomize:ref=DNSZone
	ZoneRef KubernetesNameRef `json:"zoneRef,omitempty"`

	// name of the existing resource
	// +optional
	Name *DesignateFQDN `json:"name,omitempty"`

	// description of the existing resource
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=255
	// +optional
	Description *string `json:"description,omitempty"`

	// type of the existing resource
	// +optional
	Type *RecordSetType `json:"type,omitempty"`
}

// RecordSetResourceStatus represents the observed state of the resource.
type RecordSetResourceStatus struct {
	// name is the name of the recordset, e.g. "www.example.com.".
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Name string `json:"name,omitempty"`

	// description is a human-readable description for the resource.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Description string `json:"description,omitempty"`

	// zoneID is the ID of the DNSZone this recordset belongs to.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	ZoneID string `json:"zoneID,omitempty"`

	// type is the RRTYPE of the recordset.
	// +kubebuilder:validation:MaxLength=255
	// +optional
	Type string `json:"type,omitempty"`

	// records are the record data for this recordset.
	// +kubebuilder:validation:MaxItems:=64
	// +kubebuilder:validation:items:MaxLength:=4096
	// +listType=set
	// +optional
	Records []string `json:"records,omitempty"`

	// ttl is the Time To Live for the recordset, in seconds.
	// +optional
	TTL *int32 `json:"ttl,omitempty"`

	// projectID is the ID of the project that owns this recordset (inherited from its zone).
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	ProjectID string `json:"projectID,omitempty"`
}
