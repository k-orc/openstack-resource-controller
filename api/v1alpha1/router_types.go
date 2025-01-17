/*
Copyright 2024 The ORC Authors.

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

// RouterFilter specifies a query to select an OpenStack router. At least one property must be set.
// +kubebuilder:validation:MinProperties:=1
type RouterFilter struct {
	Name *OpenStackName `json:"name,omitempty"`

	// description of the existing resource
	// +optional
	Description *NeutronDescription `json:"description,omitempty"`

	FilterByNeutronTags `json:",inline"`
}

type ExternalGateway struct {
	NetworkRef KubernetesNameRef `json:"networkRef"`
}

type ExternalGatewayStatus struct {
	NetworkID string `json:"networkID"`
}

type RouterResourceSpec struct {
	// name is the human-readable name of the router. Might not be unique.
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	// description is a human-readable description for the resource.
	// +optional
	Description *NeutronDescription `json:"description,omitempty"`

	// tags optionally set via extensions/attributestags
	// +listType=set
	Tags []NeutronTag `json:"tags,omitempty"`

	AdminStateUp *bool `json:"adminStateUp,omitempty"`

	// +listType=atomic
	// +optional
	ExternalGateways []ExternalGateway `json:"externalGateways,omitempty"`

	Distributed *bool `json:"distributed,omitempty"`

	// +listType=set
	// +optional
	AvailabilityZoneHints []AvailabilityZoneHint `json:"availabilityZoneHints,omitempty"`

	NeutronStatusMetadata `json:",inline"`
}

type RouterResourceStatus struct {
	// name is the human-readable name of the resource. Might not be unique.
	// +optional
	Name string `json:"name,omitempty"`

	// description is a human-readable description for the resource.
	// +optional
	Description string `json:"description,omitempty"`

	// projectID is the project owner of the resource.
	// +optional
	ProjectID string `json:"projectID,omitempty"`

	// status indicates the current status of the resource.
	// +optional
	Status string `json:"status,omitempty"`

	// tags is the list of tags on the resource.
	// +listType=atomic
	// +optional
	Tags []string `json:"tags,omitempty"`

	AdminStateUp bool `json:"adminStateUp"`

	// +listType=atomic
	// +optional
	ExternalGateways []ExternalGatewayStatus `json:"externalGateways,omitempty"`

	// +listType=atomic
	// +optional
	AvailabilityZoneHints []string `json:"availabilityZoneHints,omitempty"`
}
