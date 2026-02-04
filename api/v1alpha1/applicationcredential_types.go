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

// +kubebuilder:validation:Enum:=CONNECT;DELETE;GET;HEAD;OPTIONS;PATCH;POST;PUT;TRACE
type HTTPMethod string

const (
	HTTPMethodCONNECT HTTPMethod = "CONNECT"
	HTTPMethodDELETE  HTTPMethod = "DELETE"
	HTTPMethodGET     HTTPMethod = "GET"
	HTTPMethodHEAD    HTTPMethod = "HEAD"
	HTTPMethodOPTIONS HTTPMethod = "OPTIONS"
	HTTPMethodPATCH   HTTPMethod = "PATCH"
	HTTPMethodPOST    HTTPMethod = "POST"
	HTTPMethodPUT     HTTPMethod = "PUT"
	HTTPMethodTRACE   HTTPMethod = "TRACE"
)

// +kubebuilder:validation:MinProperties:=1
// +kubebuilder:validation:MaxProperties:=1
type ApplicationCredentialAccessRole struct {
	// name of an existing role
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	// id is the ID of an role
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	ID *string `json:"id,omitempty"`
}

// ApplicationCredentialAccessRule defines an access rule
// +kubebuilder:validation:MinProperties:=1
type ApplicationCredentialAccessRule struct {
	// API path that the application credential is permitted to access
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Path *string `json:"path,omitempty"`

	// request method that the application credential is permitted to use for a given API endpoint
	// +optional
	Method *HTTPMethod `json:"method,omitempty"`

	// service type identifier for the service that the application credential is permitted to access
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Service *string `json:"service,omitempty"`
}

// ApplicationCredentialResourceSpec contains the desired state of the resource.
type ApplicationCredentialResourceSpec struct {
	// name will be the name of the created resource. If not specified, the
	// name of the ORC object will be used.
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	// description is a human-readable description for the resource.
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=255
	// +optional
	Description *string `json:"description,omitempty"`

	// ID of the user the application credential belongs to
	// TODO: Replace with UserRef when ORC has support for User objects
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="UserID is immutable"
	// +kubebuilder:validation:MaxLength=1024
	// +required
	UserID string `json:"userID"`

	// flag indicating whether the application credential may be used for creation or destruction of other application credentials or trusts
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Unrestricted is immutable"
	// +optional
	Unrestricted *bool `json:"unrestricted,omitempty"`

	// TODO: Add description
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Secret is immutable"
	// +optional
	Secret *string `json:"secret,omitempty"`

	// list of role objects may only contain roles that the user has assigned on the project. If not provided, the roles assigned to the application credential will be the same as the roles in the current token.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Roles is immutable"
	// +kubebuilder:validation:MaxItems:=256
	// +listType=atomic
	// +optional
	Roles []ApplicationCredentialAccessRole `json:"roles,omitempty"`

	// list of fine grained access control rules
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Rules is immutable"
	// +kubebuilder:validation:MaxItems:=256
	// +listType=atomic
	// +optional
	AccessRules []ApplicationCredentialAccessRule `json:"accessRules,omitempty"`

	// expiry time for the application credential. If unset, the application credential does not expire.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="ExpiresAt is immutable"
	// +optional
	ExpiresAt *metav1.Time `json:"expiresAt,omitempty"`
}

// ApplicationCredentialFilter defines an existing resource by its properties
// +kubebuilder:validation:MinProperties:=1
type ApplicationCredentialFilter struct {
	// name of the existing resource
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	// ID of the user the application credential belongs to
	// +required
	UserID string `json:"userID"`
}

type ApplicationCredentialAccessRoleStatus struct {
	// name of an existing role
	// +optional
	Name *string `json:"name,omitempty"`

	// id is the ID of an role
	// +optional
	ID *string `json:"id,omitempty"`

	// id of the domain of this role
	// +optional
	DomainID *string `json:"domainID,omitempty"`
}

type ApplicationCredentialAccessRuleStatus struct {
	// id is the ID of this access rule
	// +optional
	ID *string `json:"id,omitempty"`

	// API path that the application credential is permitted to access
	// +optional
	Path *string `json:"path,omitempty"`

	// request method that the application credential is permitted to use for a given API endpoint
	// +optional
	Method *string `json:"method,omitempty"`

	// service type identifier for the service that the application credential is permitted to access
	// +optional
	Service *string `json:"service,omitempty"`
}

// ApplicationCredentialResourceStatus represents the observed state of the resource.
type ApplicationCredentialResourceStatus struct {
	// id is the ID of the application credential.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	ID string `json:"id,omitempty"`

	// name is a Human-readable name for the resource. Might not be unique.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Name string `json:"name,omitempty"`

	// description is a human-readable description for the resource.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Description string `json:"description,omitempty"`

	// flag indicating whether the application credential may be used for creation or destruction of other application credentials or trusts
	// +optional
	Unrestricted bool `json:"unrestricted,omitempty"`

	// TODO: Add description
	// +optional
	Secret string `json:"secret,omitempty"`

	// ID of the project the application credential was created for and that authentication requests using this application credential will be scoped to.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	ProjectID string `json:"projectID,omitempty"`

	// list of role objects may only contain roles that the user has assigned on the project
	// +listType=atomic
	// +optional
	Roles []ApplicationCredentialAccessRoleStatus `json:"roles"`

	// expiry time for the application credential
	// +optional
	ExpiresAt *metav1.Time `json:"expiresAt"`

	// list of fine grained access control rules
	// +listType=atomic
	// +optional
	AccessRules []ApplicationCredentialAccessRuleStatus `json:"accessRules,omitempty"`

	// Links contains referencing links to the application credential
	// +optional
	Links map[string]string `json:"links"`
}
