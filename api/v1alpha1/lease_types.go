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

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LeaseHostReservation reserves whole compute hosts for the duration of the
// lease.
// +kubebuilder:validation:XValidation:rule="self.max >= self.min",message="max must be greater than or equal to min"
type LeaseHostReservation struct {
	// min is the smallest number of hosts the lease can be satisfied with.
	// +kubebuilder:validation:Minimum:=1
	// +required
	Min int32 `json:"min,omitempty"`

	// max is the largest number of hosts to reserve.
	// +kubebuilder:validation:Minimum:=1
	// +required
	Max int32 `json:"max,omitempty"`

	// hypervisorProperties filters the candidate hosts on the properties Nova
	// reports, e.g. `[">=", "$vcpus", "4"]`.
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=1024
	// +optional
	HypervisorProperties *string `json:"hypervisorProperties,omitempty"`

	// resourceProperties filters the candidate hosts on their extra
	// capabilities, e.g. `["==", "$gpu", "a100"]`.
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=1024
	// +optional
	ResourceProperties *string `json:"resourceProperties,omitempty"`
}

// LeaseInstanceReservation reserves capacity for instances of a given size
// for the duration of the lease.
type LeaseInstanceReservation struct {
	// amount is the number of instances to reserve capacity for.
	// +kubebuilder:validation:Minimum:=1
	// +required
	Amount int32 `json:"amount,omitempty"`

	// vcpus is the number of virtual CPUs per instance.
	// +kubebuilder:validation:Minimum:=1
	// +required
	Vcpus int32 `json:"vcpus,omitempty"`

	// memoryMB is the amount of memory per instance, in megabytes.
	// +kubebuilder:validation:Minimum:=1
	// +required
	MemoryMB int32 `json:"memoryMB,omitempty"`

	// diskGB is the amount of disk per instance, in gigabytes.
	// +kubebuilder:validation:Minimum:=0
	// +required
	DiskGB *int32 `json:"diskGB,omitempty"`

	// affinity places the instances on the same host if true, or on distinct
	// hosts if false. If not specified, placement is left to Nova.
	// +optional
	Affinity *bool `json:"affinity,omitempty"`

	// resourceProperties filters the candidate hosts on their extra
	// capabilities, e.g. `["==", "$gpu", "a100"]`.
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=1024
	// +optional
	ResourceProperties *string `json:"resourceProperties,omitempty"`
}

// LeaseFlavorInstanceReservation reserves capacity for instances of an
// existing flavor for the duration of the lease.
type LeaseFlavorInstanceReservation struct {
	// amount is the number of instances to reserve capacity for.
	// +kubebuilder:validation:Minimum:=1
	// +required
	Amount int32 `json:"amount,omitempty"`

	// flavorRef is a reference to the ORC Flavor to reserve instances of.
	// Blazar derives the size and the resource properties of the
	// reservation from the flavor when the lease is created.
	// +orc:kustomize:ref=Flavor
	// +required
	FlavorRef KubernetesNameRef `json:"flavorRef,omitempty"`
}

// LeaseReservation is a single reservation within a lease. Exactly one of
// host, instance or flavorInstance must be specified.
// +kubebuilder:validation:XValidation:rule="[has(self.host), has(self.instance), has(self.flavorInstance)].filter(x, x).size() == 1",message="exactly one of host, instance or flavorInstance must be set"
type LeaseReservation struct {
	// host reserves whole compute hosts.
	// +optional
	Host *LeaseHostReservation `json:"host,omitempty"`

	// instance reserves capacity for instances of a given size.
	// +optional
	Instance *LeaseInstanceReservation `json:"instance,omitempty"`

	// flavorInstance reserves capacity for instances of an existing flavor.
	// +optional
	FlavorInstance *LeaseFlavorInstanceReservation `json:"flavorInstance,omitempty"`
}

// LeaseResourceSpec contains the desired state of the resource.
//
// A lease is only Available while Blazar reports it as ACTIVE. A lease which
// starts in the future is created immediately but stays Progressing until its
// start date. A lease which has ended (TERMINATED) or failed (ERROR) can no
// longer become Available, and is reported as a terminal error.
// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="LeaseResourceSpec is immutable"
// +kubebuilder:validation:XValidation:rule="!has(self.startDate) || timestamp(self.endDate) > timestamp(self.startDate)",message="endDate must be later than startDate"
type LeaseResourceSpec struct {
	// name will be the name of the created resource. If not specified, the
	// name of the ORC object will be used, which must then be at most 80
	// characters long.
	// +optional
	Name *BlazarName `json:"name,omitempty"`

	// startDate is the time at which the lease starts. If not specified, the
	// lease starts as soon as it is created. Blazar truncates it to the
	// minute.
	// +optional
	StartDate *metav1.Time `json:"startDate,omitempty"`

	// endDate is the time at which the lease ends. Blazar truncates it to the
	// minute.
	// +required
	EndDate metav1.Time `json:"endDate,omitempty"`

	// reservations are the resources reserved for the duration of the lease.
	// +kubebuilder:validation:MinItems:=1
	// +kubebuilder:validation:MaxItems:=32
	// +listType=atomic
	// +required
	Reservations []LeaseReservation `json:"reservations,omitempty"`
}

// LeaseFilter defines an existing resource by its properties
// +kubebuilder:validation:MinProperties:=1
type LeaseFilter struct {
	// name of the existing resource
	// +optional
	Name *BlazarName `json:"name,omitempty"`
}

// LeaseReservationStatus represents the observed state of a reservation
// within a lease.
type LeaseReservationStatus struct {
	// id is the ID of the reservation. For host reservations it is passed to
	// Nova as the `reservation` scheduler hint.
	// +kubebuilder:validation:MaxLength=36
	// +optional
	ID string `json:"id,omitempty"`

	// resourceType is the type of resource reserved: `physical:host`,
	// `virtual:instance` or `flavor:instance`.
	// +kubebuilder:validation:MaxLength=66
	// +optional
	ResourceType string `json:"resourceType,omitempty"`

	// status is the status of the reservation.
	// +kubebuilder:validation:MaxLength=13
	// +optional
	Status string `json:"status,omitempty"`

	// flavorID is the ID of the flavor Blazar created for an instance
	// reservation. Instances must be created with this flavor to consume the
	// reservation.
	// +kubebuilder:validation:MaxLength=36
	// +optional
	FlavorID string `json:"flavorID,omitempty"`

	// serverGroupID is the ID of the server group Blazar created for an
	// instance reservation.
	// +kubebuilder:validation:MaxLength=36
	// +optional
	ServerGroupID string `json:"serverGroupID,omitempty"`

	// missingResources is true if some of the reserved resources are no
	// longer available.
	// +optional
	MissingResources *bool `json:"missingResources,omitempty"`

	// resourcesChanged is true if some of the reserved resources were
	// replaced.
	// +optional
	ResourcesChanged *bool `json:"resourcesChanged,omitempty"`

	// min is the smallest number of hosts of a host reservation.
	// +optional
	Min *int32 `json:"min,omitempty"`

	// max is the largest number of hosts of a host reservation.
	// +optional
	Max *int32 `json:"max,omitempty"`

	// hypervisorProperties filters the candidate hosts of a host reservation
	// on the properties Nova reports.
	// +kubebuilder:validation:MaxLength=4096
	// +optional
	HypervisorProperties string `json:"hypervisorProperties,omitempty"`

	// resourceProperties filters the candidate hosts on their extra
	// capabilities.
	// +kubebuilder:validation:MaxLength=4096
	// +optional
	ResourceProperties string `json:"resourceProperties,omitempty"`

	// amount is the number of instances of an instance reservation.
	// +optional
	Amount *int32 `json:"amount,omitempty"`

	// vcpus is the number of virtual CPUs per instance of an instance
	// reservation.
	// +optional
	Vcpus *int32 `json:"vcpus,omitempty"`

	// memoryMB is the amount of memory per instance of an instance
	// reservation, in megabytes.
	// +optional
	MemoryMB *int32 `json:"memoryMB,omitempty"`

	// diskGB is the amount of disk per instance of an instance reservation,
	// in gigabytes.
	// +optional
	DiskGB *int32 `json:"diskGB,omitempty"`

	// affinity is the placement policy of the instances of an instance
	// reservation: on the same host if true, on distinct hosts if false.
	// +optional
	Affinity *bool `json:"affinity,omitempty"`
}

// LeaseResourceStatus represents the observed state of the resource.
type LeaseResourceStatus struct {
	// name is a Human-readable name for the resource. Might not be unique.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Name string `json:"name,omitempty"`

	// status is the status of the lease.
	// +kubebuilder:validation:MaxLength=255
	// +optional
	Status string `json:"status,omitempty"`

	// degraded is true if some of the reserved resources are unavailable.
	// +optional
	Degraded *bool `json:"degraded,omitempty"`

	// startDate is the time at which the lease starts.
	// +optional
	StartDate *metav1.Time `json:"startDate,omitempty"`

	// endDate is the time at which the lease ends.
	// +optional
	EndDate *metav1.Time `json:"endDate,omitempty"`

	// projectID is the ID of the project that owns the lease.
	// +kubebuilder:validation:MaxLength=255
	// +optional
	ProjectID string `json:"projectID,omitempty"`

	// userID is the ID of the user that created the lease.
	// +kubebuilder:validation:MaxLength=255
	// +optional
	UserID string `json:"userID,omitempty"`

	// createdAt shows the date and time when the resource was created.
	// +optional
	CreatedAt *metav1.Time `json:"createdAt,omitempty"`

	// updatedAt shows the date and time when the resource was updated.
	// +optional
	UpdatedAt *metav1.Time `json:"updatedAt,omitempty"`

	// reservations are the reservations within the lease.
	// +kubebuilder:validation:MaxItems:=32
	// +listType=atomic
	// +optional
	Reservations []LeaseReservationStatus `json:"reservations,omitempty"`
}
