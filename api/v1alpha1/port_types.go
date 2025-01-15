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

type PortRefs struct {
	// networkRef is a reference to the ORC Network which this port is associated with.
	// +required
	NetworkRef KubernetesNameRef `json:"networkRef"`
}

// PortFilter specifies a filter to select a port. At least one parameter must be specified.
// +kubebuilder:validation:MinProperties:=1
type PortFilter struct {
	Name        *OpenStackName        `json:"name,omitempty"`
	Description *OpenStackDescription `json:"description,omitempty"`
	ProjectID   *UUID                 `json:"projectID,omitempty"`

	FilterByNeutronTags `json:",inline"`
}

type AllowedAddressPair struct {
	IP  *IPvAny `json:"ip"`
	MAC *MAC    `json:"mac,omitempty"`
}

type AllowedAddressPairStatus struct {
	IP  string `json:"ip"`
	MAC string `json:"mac,omitempty"`
}

type Address struct {
	IP        *IPvAny        `json:"ip,omitempty"`
	SubnetRef *OpenStackName `json:"subnetRef"`
}

type FixedIPStatus struct {
	IP       string `json:"ip"`
	SubnetID string `json:"subnetID,omitempty"`
}

type PortResourceSpec struct {
	// name is a human-readable name of the port. If not set, the object's name will be used.
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	// description of the port.
	// +optional
	Description *OpenStackDescription `json:"description,omitempty"`

	// tags is a list of tags which will be applied to the port.
	// +kubebuilder:validation:MaxItems:=32
	// +listType=set
	// +optional
	Tags []NeutronTag `json:"tags,omitempty"`

	// projectID is the unique ID of the project which owns the Port. Only
	// administrative users can specify a project UUID other than their own.
	// +optional
	ProjectID *UUID `json:"projectID,omitempty"`

	// allowedAddressPairs are allowed addresses associated with this port.
	// +kubebuilder:validation:MaxItems:=32
	// +listType=atomic
	// +optional
	AllowedAddressPairs []AllowedAddressPair `json:"allowedAddressPairs,omitempty"`

	// addresses are the IP addresses for the port.
	// +kubebuilder:validation:MaxItems:=32
	// +listType=atomic
	// +optional
	Addresses []Address `json:"addresses,omitempty"`

	// securityGroupRefs are the names of the security groups associated
	// with this port.
	// +listType=atomic
	SecurityGroupRefs []OpenStackName `json:"securityGroupRefs,omitempty"`
}

type PortResourceStatus struct {
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

	// adminStateUp is the administrative state of the port,
	// which is up (true) or down (false).
	// +optional
	AdminStateUp bool `json:"adminStateUp,omitempty"`

	// macAddress is the MAC address of the port.
	// +optional
	MACAddress string `json:"macAddress,omitempty"`

	// deviceID is the ID of the device that uses this port.
	// +optional
	DeviceID string `json:"deviceID,omitempty"`

	// deviceOwner is the entity type that uses this port.
	// +optional
	DeviceOwner string `json:"deviceOwner,omitempty"`

	// allowedAddressPairs is a set of zero or more allowed address pair
	// objects each where address pair object contains an IP address and
	// MAC address.
	// +listType=atomic
	// +optional
	AllowedAddressPairs []AllowedAddressPairStatus `json:"allowedAddressPairs,omitempty"`

	// fixedIPs is a set of zero or more fixed IP objects each where fixed
	// IP object contains an IP address and subnet ID from which the IP
	// address is assigned.
	// +listType=atomic
	// +optional
	FixedIPs []FixedIPStatus `json:"fixedIPs,omitempty"`

	// securityGroups contains the IDs of security groups applied to the port.
	// +listType=atomic
	// +optional
	SecurityGroups []string `json:"securityGroups,omitempty"`

	// propagateUplinkStatus represents the uplink status propagation of
	// the port.
	// +optional
	PropagateUplinkStatus bool `json:"propagateUplinkStatus,omitempty"`

	NeutronStatusMetadata `json:",inline"`
}
