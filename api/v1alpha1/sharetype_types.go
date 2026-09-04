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

// ShareTypeResourceSpec contains the desired state of the resource.
// All fields are immutable after creation.
// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="ShareTypeResourceSpec is immutable"
type ShareTypeResourceSpec struct {
	// name will be the name of the created resource. If not specified, the
	// name of the ORC object will be used.
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	// isPublic indicates whether a share type is publicly accessible.
	// +kubebuilder:default:=true
	// +optional
	IsPublic *bool `json:"isPublic,omitempty"`

	// driverHandlesShareServers defines the driver mode for share server, or storage, life cycle management.
	// This is a required extra specification for share types.
	// +kubebuilder:default:=true
	// +optional
	DriverHandlesShareServers *bool `json:"driverHandlesShareServers,omitempty"`

	// snapshotSupport filters back ends by whether they do or do not support share snapshots.
	// +optional
	SnapshotSupport *bool `json:"snapshotSupport,omitempty"`
}

// ShareTypeFilter defines an existing resource by its properties
// +kubebuilder:validation:MinProperties:=1
type ShareTypeFilter struct {
	// name of the existing resource
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	// isPublic selects public types, private types, or both
	// +optional
	IsPublic *bool `json:"isPublic,omitempty"`
}

// ShareTypeResourceStatus represents the observed state of the resource.
type ShareTypeResourceStatus struct {
	// name is a Human-readable name for the resource. Might not be unique.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Name string `json:"name,omitempty"`

	// isPublic indicates whether a share type is publicly accessible.
	// +optional
	IsPublic bool `json:"isPublic,omitempty"`

	// extraSpecs contains the extra specifications for the share type.
	// +optional
	ExtraSpecs map[string]string `json:"extraSpecs,omitempty"`
}
