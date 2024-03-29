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
	"github.com/gofrs/uuid/v5"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	UuidNamespace = uuid.UUID{56, 237, 119, 111, 66, 211, 75, 156, 170, 37, 100, 120, 183, 56, 27, 191}
)

type OpenStackConditionType string

const (
	OpenStackConditionReady OpenStackConditionType = "Ready"
	OpenStackConditionError OpenStackConditionType = "Error"

	OpenStackErrorReasonInvalidSpec = "InvalidSpec"

	OpenStackLabelPrefix = "openstack.k-orc.cloud/"
)

func OpenStackDependencyLabelImage(name string) string {
	return openStackDependencyLabel("image", name)
}

func OpenStackDependencyLabelSecret(name string) string {
	return openStackDependencyLabel("secret", name)
}

func OpenStackDependencyLabelFlavor(name string) string {
	return openStackDependencyLabel("flavor", name)
}

func OpenStackDependencyLabelCloud(name string) string {
	return openStackDependencyLabel("cloud", name)
}

func OpenStackDependencyLabelNetwork(name string) string {
	return openStackDependencyLabel("network", name)
}

func OpenStackDependencyLabelSecurityGroup(name string) string {
	return openStackDependencyLabel("secgroup", name)
}

func OpenStackDependencyLabelSubnet(name string) string {
	return openStackDependencyLabel("subnet", name)
}

func OpenStackDependencyLabelPort(name string) string {
	return openStackDependencyLabel("port", name)
}

func OpenStackDependencyLabelKey(name string) string {
	return openStackDependencyLabel("key", name)
}

func openStackDependencyLabel(resource, name string) string {
	return resource + "." + OpenStackLabelPrefix + name
}

type CommonSpec struct {
	// Cloud is the OpenStackCloud hosting this resource
	Cloud string `json:"cloud"`

	// Unmanaged, when true, means that no action will be performed in
	// OpenStack against this resource.
	Unmanaged bool `json:"unmanaged,omitempty"`
}

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

// FixedIP is a data structure used in multiple resources to identify an IP
// address on a subnet.
type FixedIP struct {
	IPAddress string `json:"ipAddress,omitempty"`
	Subnet    string `json:"subnet,omitempty"`
}
