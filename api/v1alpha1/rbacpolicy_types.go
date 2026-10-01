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

// RBACPolicyAction is the type of access being granted by an RBAC policy.
// +kubebuilder:validation:Enum:=access_as_shared;access_as_external
type RBACPolicyAction string

const (
	// RBACPolicyActionAccessShared grants the target project permission to attach ports to
	// (use) the network, without granting any ability to manage the network itself.
	RBACPolicyActionAccessShared RBACPolicyAction = "access_as_shared"

	// RBACPolicyActionAccessExternal grants the target project permission to use the network
	// as an external gateway.
	RBACPolicyActionAccessExternal RBACPolicyAction = "access_as_external"
)

// RBACPolicyResourceSpec contains the desired state of the resource.
//
// Neutron's RBAC policy API only supports sharing a Network in this initial implementation -
// object_type is implicitly "network" (via networkRef) for every RBACPolicy. Neutron's RBAC API
// also covers qos-policy and security-group as other possible object_types; adding those as
// alternatives to networkRef (a discriminated union, the same pattern used by
// RouterInterfaceSpec's type/subnetRef) is a natural follow-up once there's a concrete use case,
// deliberately left out of this first pass to keep it minimal and reviewable.
type RBACPolicyResourceSpec struct {
	// networkRef is a reference to the ORC Network this policy grants access to. Immutable -
	// Neutron's own RBAC policy API has no update path for which object a policy applies to,
	// only for its targetProjectID (see gophercloud's UpdateOpts, which carries TargetTenant
	// only).
	// +required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="networkRef is immutable"
	// +orc:kustomize:ref=Network
	NetworkRef KubernetesNameRef `json:"networkRef,omitempty"`

	// action is the type of access being granted to targetProjectID. Immutable for the same
	// reason as networkRef - not present in Neutron's UpdateOpts.
	// +required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="action is immutable"
	Action RBACPolicyAction `json:"action,omitempty"`

	// targetProjectID is the OpenStack project ID to grant access to. A raw OpenStack id, not
	// a KubernetesNameRef to an ORC Project object - Project creation itself may not be usable
	// on every cloud (some providers gate identity/project provisioning behind their own
	// control plane, outside Keystone, so no corresponding ORC Project object may ever exist
	// to reference). Mutable - matches Neutron's own UpdateOpts, which only allows changing
	// the target project of an existing policy, nothing else.
	//
	// Deliberately fails the kube-api-linter noopenstackidref check (confirmed: this linter
	// doesn't support //nolint suppression, "unknown linters in //nolint directives") - raised
	// as an open question for maintainers rather than silently worked around, see the PR
	// description.
	// +required
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=64
	TargetProjectID string `json:"targetProjectID,omitempty"`
}

// RBACPolicyFilter defines an existing resource by its properties
// +kubebuilder:validation:MinProperties:=1
type RBACPolicyFilter struct {
	// action of the existing resource
	// +optional
	Action *RBACPolicyAction `json:"action,omitempty"`

	// targetProjectID of the existing resource
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=64
	// +optional
	TargetProjectID *string `json:"targetProjectID,omitempty"`
}

// RBACPolicyResourceStatus represents the observed state of the resource.
type RBACPolicyResourceStatus struct {
	// networkID is the ID of the Network this policy applies to.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	NetworkID string `json:"networkID,omitempty"`

	// action is the type of access granted to targetProjectID.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Action string `json:"action,omitempty"`

	// targetProjectID is the OpenStack project ID this policy grants access to.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	TargetProjectID string `json:"targetProjectID,omitempty"`

	// projectID is the ID of the project that owns the shared network (and therefore this
	// policy) - not to be confused with targetProjectID, the project being granted access.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	ProjectID string `json:"projectID,omitempty"`
}
