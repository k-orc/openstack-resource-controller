/*
Copyright 2023 Red Hat

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

type OpenStackConditionType string

const (
	OpenStackConditionReady OpenStackConditionType = "Ready"
	OpenStackConditionError OpenStackConditionType = "Error"
)

// OpenStackResourceCommonStatus returns status fields common to all OpenStack resources
// +kubebuilder:object:generate=false
type OpenStackResourceCommonStatus interface {
	OpenStackCommonStatus() *CommonStatus
}

// CommonStatus defines fields common to all OpenStack resource statuses
type CommonStatus struct {
	// Represents the observations of an OpenStack resource's current state.
	// All resources must define: "Ready", "WaitingFor", "Error"
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}
