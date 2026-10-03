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

// DNSZoneShareResourceSpec contains the desired state of the resource.
//
// Designate has two separate mechanisms for giving another project access to a zone: zone
// *transfer* (transfer_requests/transfer_accepts), which moves full ownership and needs both the
// owner's and the recipient's credentials in one flow; and zone *share*, modeled here, which
// grants another project co-management rights over a zone's recordsets while the original
// project keeps ownership - a single-credential operation, only the owner's. See
// RBACPolicy's own doc comment for the same shape applied to Neutron network sharing; this is
// the Designate equivalent. Confirmed live against a real OpenStack deployment (a create/list/
// delete round-trip) that this API exists and works as gophercloud's bindings describe, not just
// assumed from reading the client library.
type DNSZoneShareResourceSpec struct {
	// zoneRef is a reference to the ORC DNSZone this share grants access to. Immutable -
	// Designate's zone-share API has no update path for which zone a share applies to, only
	// create and delete.
	// +required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="zoneRef is immutable"
	// +orc:kustomize:ref=DNSZone
	ZoneRef KubernetesNameRef `json:"zoneRef,omitempty"`

	// targetProjectID is the OpenStack project ID to grant access to. A raw OpenStack ID, not a
	// KubernetesNameRef to an ORC Project object - Project creation itself may not be usable on
	// every cloud (some providers gate identity/project provisioning behind their own control
	// plane, outside Keystone, so no corresponding ORC Project object may ever exist to
	// reference). Immutable - Designate's zone-share API has no update operation at all; changing
	// the target means deleting this share and creating a new one.
	// +required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="targetProjectID is immutable"
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=64
	TargetProjectID string `json:"targetProjectID,omitempty"`
}

// DNSZoneShareFilter defines an existing resource by its properties
// +kubebuilder:validation:MinProperties:=1
type DNSZoneShareFilter struct {
	// zoneRef is a reference to the ORC DNSZone to look for the share under - required because
	// every Designate zone-share operation, including list, is scoped to a specific zone (see
	// DNSZoneShareResourceSpec.zoneRef's doc comment for the same constraint on the managed path).
	// +required
	// +orc:kustomize:ref=DNSZone
	ZoneRef KubernetesNameRef `json:"zoneRef,omitempty"`

	// targetProjectID of the existing resource. If not specified, matches any target project -
	// which is only unambiguous if the referenced zone has exactly one share.
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=64
	// +optional
	TargetProjectID *string `json:"targetProjectID,omitempty"`
}

// DNSZoneShareResourceStatus represents the observed state of the resource.
type DNSZoneShareResourceStatus struct {
	// zoneID is the ID of the DNSZone this share applies to.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	ZoneID string `json:"zoneID,omitempty"`

	// targetProjectID is the OpenStack project ID this share grants access to.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	TargetProjectID string `json:"targetProjectID,omitempty"`

	// projectID is the ID of the project that owns the shared zone (and therefore this share) -
	// not to be confused with targetProjectID, the project being granted access.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	ProjectID string `json:"projectID,omitempty"`
}
