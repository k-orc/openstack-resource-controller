/*
Copyright 2025 The ORC Authors.

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

// HostAggregateResourceSpec contains the desired state of the resource.
type HostAggregateResourceSpec struct {
	// TODO(stephenfin): Enforce that the name should not contain a colon.
	// name will be the name of the created resource. If not specified, the
	// name of the ORC object will be used.
	// +optional
	Name *OpenStackName `json:"name"`

	// availabilityZone is the availability zone of the host aggregate.
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=255
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="availabilityZone is immutable"
	// +optional
	AvailabilityZone *string `json:"availabilityZone,omitempty"`
}

// HostAggregateFilter defines an existing resource by its properties
// +kubebuilder:validation:MinProperties:=1
type HostAggregateFilter struct {
	// name of the existing resource
	// +optional
	Name *OpenStackName `json:"name,omitempty"`
}

// HostAggregateResourceStatus represents the observed state of the resource.
type HostAggregateResourceStatus struct {
	// The availability zone of the host aggregate.
	// +optional
	AvailabilityZone string `json:"availabilityZone"`

	// A list of host ids in this aggregate.
	//Hosts []string `json:"hosts"`

	// Metadata key and value pairs associate with the aggregate.
	// Metadata map[string]string `json:"metadata"`

	// name is a Human-readable name for the resource. Might not be unique.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Name string `json:"name"`

	// The date and time when the resource was created.
	// CreatedAt time.Time `json:"-"`

	// The date and time when the resource was updated,
	// if the resource has not been updated, this field will show as null.
	// UpdatedAt time.Time `json:"-"`

	// The date and time when the resource was deleted,
	// if the resource has not been deleted yet, this field will be null.
	// DeletedAt time.Time `json:"-"`

	// A boolean indicates whether this aggregate is deleted or not,
	// if it has not been deleted, false will appear.
	// Deleted bool `json:"deleted"`
}
