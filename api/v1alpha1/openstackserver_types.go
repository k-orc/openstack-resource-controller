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

// OpenStackServerResourceSpec defines the desired state of OpenStackServer
type OpenStackServerResourceSpec struct {
	// Name contains the human-readable name for the server.
	Name string `json:"name,omitempty"`

	// Image indicates the OpenStackImage used to deploy the server.
	Image string `json:"image,omitempty"`

	// Flavor indicates the OpenStackFlavor of the deployed server.
	Flavor string `json:"flavor,omitempty"`

	// Networks indicates the OpenStackNetworks to attach the server to.
	Networks []OpenStackServerSpecNetworks `json:"networks"`

	// Metadata includes a list of all user-specified key-value pairs attached
	// to the server.
	Metadata map[string]string `json:"metadata,omitempty"`

	// Key indicates an OpenStackKey to injected into the server on launch.
	Key string `json:"key,omitempty"`

	// SecurityGroups sets the security groups to apply to this instance.
	SecurityGroups []string `json:"securityGroups,omitempty"`

	// UserData contains configuration information or scripts to use upon launch.
	// Create will base64-encode it for you, if it isn't already.
	UserData []byte `json:"userData,omitempty"`

	// AttachedVolumes sets the volume attachments of this instance.
	// AttachedVolumes []string `json:"volumesAttached,omitempty"`

	// Tags is a slice/list of string tags in a server.
	// The requires microversion 2.26 or later.
	Tags []string `json:"tags,omitempty"`

	// ServerGroups is a slice of strings containing the UUIDs of the
	// server groups to which the server belongs. Currently this can
	// contain at most one entry.
	// New in microversion 2.71
	// ServerGroups []string `json:"serverGroups,omitempty"`
}

type OpenStackServerSpecNetworks struct {
	Network string `json:"network,omitempty"`
	Port    string `json:"port,omitempty"`
	FixedIP string `json:"fixedIP,omitempty"`
	Tag     string `json:"tag,omitempty"`
}

// OpenStackServerResourceStatus defines the observed state of OpenStackServer
type OpenStackServerResourceStatus struct {
	// ID uniquely identifies this server amongst all other servers,
	// including those not accessible to the current tenant.
	ID string `json:"id"`

	// TenantID identifies the tenant owning this server resource.
	TenantID string `json:"tenantID,omitempty"`

	// UserID uniquely identifies the user account owning the tenant.
	UserID string `json:"userID,omitempty"`

	// Name contains the human-readable name for the server.
	Name string `json:"name,omitempty"`

	// UpdatedAt contains the timestamp of when the resource was last
	// changed.
	UpdatedAt string `json:"updatedAt,omitempty"`

	// CreatedAt contains the timestamp of when the resource was created.
	CreatedAt string `json:"createdAt,omitempty"`

	// HostID is the host where the server is located in the cloud.
	HostID string `json:"hostID,omitempty"`

	// Status contains the current operational status of the server,
	// such as IN_PROGRESS or ACTIVE.
	Status string `json:"status,omitempty"`

	// Progress ranges from 0..100.
	// A request made against the server completes only once Progress reaches 100.
	Progress int `json:"progress,omitempty"`

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

	// Links includes HTTP references to the itself, useful for passing along to
	// other APIs that might want a server reference.
	Links []string `json:"links,omitempty"`

	// KeyName indicates which public key was injected into the server on launch.
	KeyName string `json:"keyName,omitempty"`

	// SecurityGroupIDs includes the security groups that this instance has
	// applied to it.
	SecurityGroupIDs string `json:"securityGroupIDs,omitempty"`

	// AttachedVolumes includes the volume attachments of this instance
	AttachedVolumeIDs []string `json:"volumesAttached,omitempty"`

	// Fault contains failure information about a server.
	Fault string `json:"fault,omitempty"`

	// Tags is a slice/list of string tags in a server.
	// The requires microversion 2.26 or later.
	Tags []string `json:"tags,omitempty"`

	// ServerGroupIDs is a slice of strings containing the UUIDs of the
	// server groups to which the server belongs. Currently this can
	// contain at most one entry.
	// New in microversion 2.71
	ServerGroupIDs []string `json:"serverGroupIDs,omitempty"`
}

// OpenStackServerSpec defines the desired state of OpenStackPort
type OpenStackServerSpec struct {
	CommonSpec `json:",inline"`

	// ID is the UUID of the existing OpenStack resource to be adopted. If
	// left empty, the controller will create a new resource using the
	// information in the "resource" stanza.
	ID string `json:"id,omitempty"`

	Resource *OpenStackServerResourceSpec `json:"resource,omitempty"`
}

// OpenStackServerStatus defines the observed state of OpenStackPort
type OpenStackServerStatus struct {
	CommonStatus `json:",inline"`

	Resource OpenStackServerResourceStatus `json:"resource,omitempty"`
}

// Implement OpenStackResourceCommonStatus interface
func (c *OpenStackServer) OpenStackCommonStatus() *CommonStatus {
	return &c.Status.CommonStatus
}

var _ OpenStackResourceCommonStatus = &OpenStackPort{}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
//+kubebuilder:printcolumn:name="Error",type=string,JSONPath=`.status.conditions[?(@.type=="Error")].status`
//+kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].message`

// OpenStackServer is the Schema for the openstackservers API
type OpenStackServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OpenStackServerSpec   `json:"spec,omitempty"`
	Status OpenStackServerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OpenStackServerList contains a list of OpenStackServer
type OpenStackServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OpenStackServer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OpenStackServer{}, &OpenStackServerList{})
}
