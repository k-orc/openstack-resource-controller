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

type OpenStackPortResourceSpec struct {
	// The administrative state of the resource, which is up (true) or down (false). Default is true.
	AdminStateUp *bool `json:"adminStateUp,omitempty"`

	// A set of zero or more allowed address pair objects each where
	// address pair object contains an ip_address and mac_address. While
	// the ip_address is required, the mac_address will be taken from the
	// port if not specified. The value of ip_address can be an IP Address
	// or a CIDR (if supported by the underlying extension plugin). A
	// server connected to the port can send a packet with source address
	// which matches one of the specified allowed address pairs.
	AllowedAddressPairs []OpenStackPortAllowedAddressPair `json:"allowedAddressPairs,omitempty"`

	// The ID of the host where the port resides. The default is an empty
	// string.
	// TODO: not in the Gophercloud spec
	// HostID string `json:"hostID,omitempty"`

	// A dictionary that enables the application running on the specific
	// host to pass and receive vif port information specific to the
	// networking back-end. This field is only meant for machine-machine
	// communication for compute services like Nova, Ironic or Zun to pass
	// information to a Neutron back-end. It should not be used by multiple
	// services concurrently or by cloud end users. The existing
	// counterexamples (capabilities: [switchdev] for Open vSwitch hardware
	// offload and trusted=true for Trusted Virtual Functions) are due to
	// be cleaned up. The networking API does not define a specific format
	// of this field. The default is an empty dictionary. If you update it
	// with null then it is treated like {} in the response. Since the
	// port-mac-address-override extension the device_mac_address field of
	// the binding:profile can be used to provide the MAC address of the
	// physical device a direct-physical port is being bound to. If
	// provided, then the mac_address field of the port resource will be
	// updated to the MAC from the active binding.
	// Profile     string `json:"profile,omitempty"`

	// The type of vNIC which this port should be attached to. This is used
	// to determine which mechanism driver(s) to be used to bind the port.
	// The valid values are normal, macvtap, direct, baremetal,
	// direct-physical, virtio-forwarder, smart-nic and remote-managed.
	// What type of vNIC is actually available depends on deployments. The
	// default is normal.
	// TODO: not in the Gophercloud spec
	// VNICType string `json:"vnicType,omitempty"`

	// A human-readable description for the resource. Default is an empty
	// string.
	Description string `json:"description,omitempty"`

	// DeviceID is not exposed in this API. Define the port attachment on
	// the receiving device.

	// The entity type that uses this port. For example, compute:nova
	// (server instance), network:dhcp (DHCP agent) or
	// network:router_interface (router interface).
	DeviceOwner string `json:"deviceOwner,omitempty"`

	// A valid DNS domain.
	// TODO: not in the Gophercloud spec
	// DNSDomain string `json:"dnsDomain,omitempty"`

	// A valid DNS name.
	// TODO: not in the Gophercloud spec
	// DNSName string `json:"dnsName,omitempty"`

	// A set of zero or more extra DHCP option pairs. An option pair
	// consists of an option value and name.
	// TODO: not in the Gophercloud spec
	// ExtraDHCPOpts []OpenStackPortDHCPOption `json:"extraDHCPOpts,omitempty"`

	// The IP addresses for the port. If you would like to assign multiple
	// IP addresses for the port, specify multiple entries in this field.
	// Each entry consists of IP address (ip_address) and the subnet ID
	// from which the IP address is assigned (subnet_id). If you specify
	// both a subnet ID and an IP address, OpenStack Networking tries to
	// allocate the IP address on that subnet to the port. If you specify
	// only a subnet ID, OpenStack Networking allocates an available IP
	// from that subnet to the port. If you specify only an IP address,
	// OpenStack Networking tries to allocate the IP address if the address
	// is a valid IP for any of the subnets on the specified network.
	FixedIPs []FixedIP `json:"fixedIPs,omitempty"`

	// Admin-only. A dict, at the top level keyed by mechanism driver
	// aliases (as defined in setup.cfg). To following values can be used
	// to control Open vSwitch’s Userspace Tx packet steering feature:
	//
	//     {"openvswitch": {"other_config": {"tx-steering": "hash"}}}
	//     {"openvswitch": {"other_config": {"tx-steering": "thread"}}}
	//
	// If omitted the default is defined by Open vSwitch. The field cannot
	// be longer than 4095 characters.
	// Hints

	// The MAC address of the port. If unspecified, a MAC address is
	// automatically generated.
	MACAddress string `json:"macAddress,omitempty"`

	// Human-readable name of the resource. Default is an empty string.
	Name string `json:"name,omitempty"`

	// The name of the attached OpenStackNetwork
	Network string `json:"network,omitempty"`

	// The port NUMA affinity policy requested during the virtual machine
	// scheduling. Values: None, required, preferred or legacy.
	// TODO: not in the Gophercloud spec
	// NumaAffinityPolicy string `json:"numaAffinityPolicy,omitempty"`

	// The port security status. A valid value is enabled (true) or
	// disabled (false). If port security is enabled for the port, security
	// group rules and anti-spoofing rules are applied to the traffic on
	// the port. If disabled, no such rules are applied.
	// TODO: not in the Gophercloud spec
	// PortSecurityEnabled *bool `json:"portSecurityEnabled,omitempty"`

	// The ID of the project that owns the resource. Only administrative
	// and users with advsvc role can specify a project ID other than their
	// own. You cannot change this value through authorization policies.
	ProjectID string `json:"projectID,omitempty"`

	// QoS policy associated with the port.
	// TODO: not in the Gophercloud spec
	// QOSPolicyID string `json:"qosPolicyID,omitempty"`

	// The OpenStackSecurityGroups applied to the port.
	SecurityGroups []string `json:"securityGroups,omitempty"`

	// The ID of the project that owns the resource. Only administrative
	// and users with advsvc role can specify a project ID other than their
	// own. You cannot change this value through authorization policies.
	TenantID string `json:"tenantID,omitempty"`

	// The uplink status propagation of the port. Valid values are enabled
	// (true) and disabled (false).
	PropagateUplinkStatus *bool `json:"propagateUplinkStatus,omitempty"`

	// A boolean value that indicates if MAC Learning is enabled on the
	// associated port.
	// TODO: not in the Gophercloud spec
	// MACLearningEnabled *bool `json:"macLearningEnabled,omitempty"`
}

type OpenStackPortResourceStatus struct {
	// UUID for the port.
	ID string `json:"id,omitempty"`

	// Network that this port is associated with.
	NetworkID string `json:"networkID,omitempty"`

	// Human-readable name for the port. Might not be unique.
	Name string `json:"name,omitempty"`

	// Describes the port.
	Description string `json:"description,omitempty"`

	// Administrative state of port. If false (down), port does not forward
	// packets.
	AdminStateUp bool `json:"adminStateUp,omitempty"`

	// Indicates whether network is currently operational. Possible values include
	// `ACTIVE', `DOWN', `BUILD', or `ERROR'. Plug-ins might define additional
	// values.
	Status string `json:"status,omitempty"`

	// Mac address to use on this port.
	MACAddress string `json:"macAddress,omitempty"`

	// Specifies IP addresses for the port thus associating the port itself with
	// the subnets where the IP addresses are picked from
	FixedIPs []OpenStackPortStatusFixedIP `json:"fixedIPs,omitempty"`

	// TenantID is the project owner of the port.
	TenantID string `json:"tenantID,omitempty"`

	// ProjectID is the project owner of the port.
	ProjectID string `json:"projectID,omitempty"`

	// Identifies the entity (e.g.: dhcp agent) using this port.
	DeviceOwner string `json:"deviceOwner,omitempty"`

	// Specifies the IDs of any security groups associated with a port.
	SecurityGroups []string `json:"securityGroups,omitempty"`

	// Identifies the device (e.g., virtual server) using this port.
	DeviceID string `json:"deviceID,omitempty"`

	// Identifies the list of IP addresses the port will recognize/accept
	AllowedAddressPairs []OpenStackPortAllowedAddressPair `json:"allowedAddressPairs,omitempty"`

	// Tags optionally set via extensions/attributestags
	Tags []string `json:"tags,omitempty"`

	// PropagateUplinkStatus enables/disables propagate uplink status on the port.
	PropagateUplinkStatus bool `json:"propagateUplinkStatus,omitempty"`

	// Extra parameters to include in the request.
	ValueSpecs map[string]string `json:"valueSpecs,omitempty"`

	// RevisionNumber optionally set via extensions/standard-attr-revisions
	RevisionNumber int `json:"revisionNumber,omitempty"`

	// Timestamp when the port was created
	CreatedAt string `json:"createdAt,omitempty"`

	// Timestamp when the port was last updated
	UpdatedAt string `json:"updatedAt,omitempty"`
}

type OpenStackPortStatusFixedIP struct {
	IPAddress string `json:"ipAddress,omitempty"`
	SubnetID  string `json:"subnetID,omitempty"`
}
type OpenStackPortAllowedAddressPair struct {
	IPAddress  string `json:"ipAddress"`
	MACAddress string `json:"macAddress,omitempty"`
}

type OpenStackPortDHCPOption struct {
	OptValue  string `json:"optValue,omitempty"`
	IpVersion int    `json:"ipVersion,omitempty"`
	OptName   string `json:"optName,omitempty"`
}

// OpenStackPortSpec defines the desired state of OpenStackPort
type OpenStackPortSpec struct {
	CommonSpec `json:",inline"`

	// ID is the UUID of the existing OpenStack resource to be adopted. If
	// left empty, the controller will create a new resource using the
	// information in the "resource" stanza.
	ID string `json:"id,omitempty"`

	Resource *OpenStackPortResourceSpec `json:"resource,omitempty"`
}

// OpenStackPortStatus defines the observed state of OpenStackPort
type OpenStackPortStatus struct {
	CommonStatus `json:",inline"`

	Resource OpenStackPortResourceStatus `json:"resource,omitempty"`
}

// Implement OpenStackResourceCommonStatus interface
func (c *OpenStackPort) OpenStackCommonStatus() *CommonStatus {
	return &c.Status.CommonStatus
}

var _ OpenStackResourceCommonStatus = &OpenStackPort{}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
//+kubebuilder:printcolumn:name="Error",type=string,JSONPath=`.status.conditions[?(@.type=="Error")].status`
//+kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].message`

// OpenStackPort is the Schema for the openstackports API
type OpenStackPort struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OpenStackPortSpec   `json:"spec,omitempty"`
	Status OpenStackPortStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OpenStackPortList contains a list of OpenStackPort
type OpenStackPortList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OpenStackPort `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OpenStackPort{}, &OpenStackPortList{})
}
