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

// OpenStackCloudSpec defines the desired state of OpenStackCloud
type OpenStackCloudSpec struct {
	// Cloud is the key to look for in the "clouds" object in clouds.yaml.
	Cloud string `json:"cloud"`

	// Credentials defines where to find clouds.yaml.
	Credentials OpenStackCloudCredentials `json:"credentials"`
}

type OpenStackCloudCredentials struct {
	Source    string                             `json:"source"`
	SecretRef OpenStackCloudCredentialsSecretRef `json:"secretRef"`
}

type OpenStackCloudCredentialsSecretRef struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

// OpenStackCloudStatus defines the observed state of OpenStackCloud
type OpenStackCloudStatus struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

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
