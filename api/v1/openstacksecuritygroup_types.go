/*
Copyright 2023.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OpenStackSecurityGroupSpec defines the desired state of OpenStackSecurityGroup
type OpenStackSecurityGroupSpec struct {
	// Cloud is the OpenStackCloud hosting this resource
	Cloud string `json:"cloud"`

	// ID is the OpenStack UUID of the resource. If left empty, the
	// controller will create a new resource and populate this field. If
	// manually populated, the controller will adopt the corresponding
	// resource.
	ID string `json:"id,omitempty"`

	// Name of the OpenStack resource.
	Name string `json:"name,omitempty"`

	Description string `json:"description,omitempty"`

	// Unmanaged, when true, means that no action will be performed in
	// OpenStack against this resource. This is false by default, except
	// for pre-existing resources that are adopted by passing ID on
	// creation.
	Unmanaged *bool `json:"unmanaged,omitempty"`
}

// OpenStackSecurityGroupStatus defines the observed state of OpenStackSecurityGroup
type OpenStackSecurityGroupStatus struct {
	// The UUID for the security group.
	ID string `json:"id"`

	// Human-readable name for the security group. Might not be unique.
	// Cannot be named "default" as that is automatically created for a tenant.
	Name string `json:"name"`

	// The security group description.
	Description string `json:"description,omitempty"`

	// A slice of security group rule IDs that dictate the permitted
	// behaviour for traffic entering and leaving the group.
	Rules []string `json:"securityGroupRulesID,omitempty"`

	// TenantID is the project owner of the security group.
	TenantID string `json:"tenantID,omitempty"`

	// UpdatedAt contains the timestamp of when the resource was last
	// changed.
	UpdatedAt string `json:"updatedAt,omitempty"`

	// CreatedAt contains the timestamp of when the resource was created.
	CreatedAt string `json:"createdAt,omitempty"`

	// ProjectID is the project owner of the security group.
	ProjectID string `json:"projectID,omitempty"`

	// Tags optionally set via extensions/attributestags
	Tags []string `json:"tags,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// OpenStackSecurityGroup is the Schema for the openstacksecuritygroups API
type OpenStackSecurityGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OpenStackSecurityGroupSpec   `json:"spec,omitempty"`
	Status OpenStackSecurityGroupStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OpenStackSecurityGroupList contains a list of OpenStackSecurityGroup
type OpenStackSecurityGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OpenStackSecurityGroup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OpenStackSecurityGroup{}, &OpenStackSecurityGroupList{})
}
