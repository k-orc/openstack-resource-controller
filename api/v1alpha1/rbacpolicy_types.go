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

// RBACPolicyResourceSpec contains the desired state of the resource. Only network policies
// are supported.
// +kubebuilder:validation:XValidation:rule="has(self.targetProjectRef) != has(self.targetProjectID)",message="exactly one of targetProjectRef and targetProjectID must be set"
type RBACPolicyResourceSpec struct {
	// networkRef is a reference to the ORC Network this policy grants access to.
	// +required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="networkRef is immutable"
	// +orc:kustomize:ref=Network
	NetworkRef KubernetesNameRef `json:"networkRef,omitempty"`

	// action is the type of access being granted to the target project.
	// +required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="action is immutable"
	Action RBACPolicyAction `json:"action,omitempty"`

	// targetProjectRef is a reference to the ORC Project to grant access to.
	// +optional
	// +orc:kustomize:ref=Project
	TargetProjectRef *KubernetesNameRef `json:"targetProjectRef,omitempty"`

	// targetProjectID is the ID of the project to grant access to, for when it can't be
	// referenced: the target is another project, which the network owner's credentials usually
	// can't read, so it can't be imported with them.
	// +optional
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=64
	TargetProjectID *string `json:"targetProjectID,omitempty"`
}

// RBACPolicyFilter defines an existing resource by its properties
// +kubebuilder:validation:MinProperties:=1
// +kubebuilder:validation:XValidation:rule="!(has(self.targetProjectRef) && has(self.targetProjectID))",message="only one of targetProjectRef and targetProjectID can be set"
type RBACPolicyFilter struct {
	// action of the existing resource
	// +optional
	Action *RBACPolicyAction `json:"action,omitempty"`

	// targetProjectRef is a reference to the ORC Project of the existing resource's target
	// +optional
	// +orc:kustomize:ref=Project
	TargetProjectRef *KubernetesNameRef `json:"targetProjectRef,omitempty"`

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

	// projectID is the ID of the project that owns this policy.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	ProjectID string `json:"projectID,omitempty"`
}
