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

// ServerResourceSpec contains the desired state of a server
type ServerResourceSpec struct {
	// Name will be the name of the created resource. If not specified, the
	// name of the ORC object will be used.
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	Image ORCNameRef `json:"image,omitempty"`

	Flavor ORCNameRef `json:"flavor,omitempty"`
}

// ServerFilter defines an existing resource by its properties
// +kubebuilder:validation:MinProperties:=1
type ServerFilter struct {
	// Name of the existing resource
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	Image ORCNameRef `json:"image,omitempty"`

	Flavor ORCNameRef `json:"flavor,omitempty"`
}

// ServerResourceStatus represents the observed state of the resource.
type ServerResourceStatus struct {
	// ID uniquely identifies this server amongst all other servers,
	// including those not accessible to the current tenant.
	ID string `json:"id"`

	// Name is the human-readable name of the resource. Might not be unique.
	// +optional
	Name string `json:"name,omitempty"`

	// HostID is the host where the server is located in the cloud.
	HostID string `json:"hostID,omitempty"`

	// Status contains the current operational status of the server,
	// such as IN_PROGRESS or ACTIVE.
	Status string `json:"status,omitempty"`

	// AccessIPv4 contains the IPv4 addresses of the server, suitable for
	// remote access for administration.
	AccessIPv4 string `json:"accessIPv4,omitempty"`

	// AccessIPv6 contains the IPv6 addresses of the server, suitable for
	// remote access for administration.
	AccessIPv6 string `json:"accessIPv6,omitempty"`

	// ImageID indicates the OS image used to deploy the server.
	ImageID string `json:"imageID,omitempty"`

	// FlavorID indicates the hardware configuration of the deployed server.
	FlavorID string `json:"flavorID,omitempty"`

	// Addresses includes a list of all IP addresses assigned to the server,
	// keyed by pool.
	Addresses string `json:"addresses,omitempty"`

	// Metadata includes all user-specified key-value pairs attached to the
	// server.
	Metadata string `json:"metadata,omitempty"`

	// KeyName indicates which public key was injected into the server on launch.
	KeyName string `json:"keyName,omitempty"`

	// SecurityGroupIDs includes the security groups that this instance has
	// applied to it.
	SecurityGroupIDs string `json:"securityGroupIDs,omitempty"`

	// Fault contains failure information about a server.
	Fault string `json:"fault,omitempty"`
}
