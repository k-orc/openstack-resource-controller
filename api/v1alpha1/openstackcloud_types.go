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

const (
	OpenStackCloudSecretNameLabel = "openstack.gopherkube.dev/secret-ref"

	OpenStackCloudCredentialsSourceTypeSecret = "secret"
	OpenStackCloudCredentialsSourceInvalid    = "SourceTypeInvalid"
)

// OpenStackCloudSpec defines the desired state of OpenStackCloud
type OpenStackCloudSpec struct {
	// Cloud is the key to look for in the "clouds" object in clouds.yaml.
	Cloud string `json:"cloud"`

	// Credentials defines where to find clouds.yaml.
	Credentials OpenStackCloudCredentials `json:"credentials"`
}

type OpenStackCloudCredentials struct {
	// Source defines the source type of the credentials. The only supported value is "secret".
	// +kubebuilder:validation:Enum=secret
	Source string `json:"source"`

	// SecretRef defines the reference to the secret containing the credentials.
	SecretRef OpenStackCloudCredentialsSecretRef `json:"secretRef"`
}

type OpenStackCloudCredentialsSecretRef struct {
	// Name is the name of the secret containing the credentials.
	Name string `json:"name"`

	// Key is the key in the secret containing the credentials.
	Key string `json:"key"`
}

// OpenStackCloudStatus defines the observed state of OpenStackCloud
type OpenStackCloudStatus struct {
	CommonStatus `json:",inline"`
}

// Implement OpenStackResourceCommonStatus interface

func (c *OpenStackCloud) OpenStackCommonStatus() *CommonStatus {
	return &c.Status.CommonStatus
}

var _ OpenStackResourceCommonStatus = &OpenStackCloud{}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
//+kubebuilder:printcolumn:name="Error",type=string,JSONPath=`.status.conditions[?(@.type=="Error")].status`
//+kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].message`

// OpenStackCloud is the Schema for the openstackclouds API
type OpenStackCloud struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OpenStackCloudSpec   `json:"spec,omitempty"`
	Status OpenStackCloudStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OpenStackCloudList contains a list of OpenStackCloud
type OpenStackCloudList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OpenStackCloud `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OpenStackCloud{}, &OpenStackCloudList{})
}
