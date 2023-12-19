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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type OpenStackKeypairResourceSpec struct {
	// Name of the OpenStack resource.
	Name string `json:"name,omitempty"`

	// PublicKey is the public ssh key to import. Was optional before
	// microversion 2.92 : if you were omitting this value, a keypair was
	// generated for you.
	PublicKey string `json:"publicKey"`

	// Type is the type of the keypair. Allowed values are ssh or x509. New
	// in version 2.2
	// +kubebuilder:validation:Enum:="";"ssh";"x509"
	Type string `json:"type,omitempty"`

	// UserID is the user_id for a keypair. This allows administrative
	// users to upload keys for other users than themselves. New in version
	// 2.10
	UserID string `json:"userID,omitempty"`
}

type OpenStackKeypairResourceStatus struct {
	Name        string `json:"name,omitempty"`
	PublicKey   string `json:"publicKey,omitempty"`
	Fingerprint string `json:"fingerprint,omitempty"`
	UserID      string `json:"userID,omitempty"`
	Type        string `json:"type,omitempty"`
}

// OpenStackKeypairSpec defines the desired state of OpenStackKeypair
type OpenStackKeypairSpec struct {
	CommonSpec `json:",inline"`

	// Name is the identifier of the existing OpenStack resource to be
	// adopted. If left empty, the controller will create a new resource
	// using the information in the "resource" stanza.
	Name string `json:"name,omitempty"`

	Resource *OpenStackKeypairResourceSpec `json:"resource,omitempty"`
}

// OpenStackKeypairStatus defines the observed state of OpenStackKeypair
type OpenStackKeypairStatus struct {
	CommonStatus `json:",inline"`

	Resource OpenStackKeypairResourceStatus `json:"resource,omitempty"`
}

// Implement OpenStackResourceCommonStatus interface
func (c *OpenStackKeypair) OpenStackCommonStatus() *CommonStatus {
	return &c.Status.CommonStatus
}

var _ OpenStackResourceCommonStatus = &OpenStackKeypair{}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
//+kubebuilder:printcolumn:name="Error",type=string,JSONPath=`.status.conditions[?(@.type=="Error")].status`
//+kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].message`
//+kubebuilder:printcolumn:name="OpenStackID",type=string,JSONPath=`.status.resource.name`

// OpenStackKeypair is the Schema for the openstackkeypairs API
type OpenStackKeypair struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OpenStackKeypairSpec   `json:"spec,omitempty"`
	Status OpenStackKeypairStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OpenStackKeypairList contains a list of OpenStackKeypair
type OpenStackKeypairList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OpenStackKeypair `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OpenStackKeypair{}, &OpenStackKeypairList{})
}
