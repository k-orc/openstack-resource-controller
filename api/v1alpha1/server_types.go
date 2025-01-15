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

// +kubebuilder:validation:MinProperties:=1
// +kubebuilder:validation:MaxProperties:=1
type ServerPortSpec struct {
	// portRef is a reference to a Port object. Server creation will wait for
	// this port to be created and available.
	PortRef *KubernetesNameRef `json:"portRef,omitempty"`
}

// ServerResourceSpec contains the desired state of a server
type ServerResourceSpec struct {
	// name will be the name of the created resource. If not specified, the
	// name of the ORC object will be used.
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	ImageRef KubernetesNameRef `json:"imageRef"`

	FlavorRef KubernetesNameRef `json:"flavorRef"`

	// userData specifies data which will be made available to the server at
	// boot time, either via the metadata service or a config drive. It is
	// typically read by a configuration service such as cloud-init or ignition.
	UserData *UserDataSpec `json:"userData,omitempty"`

	// ports defines a list of ports which will be attached to the server.
	// +listType=atomic
	// +kubebuilder:validation:MaxItems:=32
	Ports []ServerPortSpec `json:"ports,omitempty"`
}

// +kubebuilder:validation:MinProperties:=1
// +kubebuilder:validation:MaxProperties:=1
type UserDataSpec struct {
	// secretRef is a reference to a Secret containing the user data for this server.
	SecretRef *KubernetesNameRef `json:"secretRef,omitempty"`
}

// ServerFilter defines an existing resource by its properties
// +kubebuilder:validation:MinProperties:=1
type ServerFilter struct {
	// name of the existing resource
	// +optional
	Name *OpenStackName `json:"name,omitempty"`
}

// ServerResourceStatus represents the observed state of the resource.
type ServerResourceStatus struct {
	// name is the human-readable name of the resource. Might not be unique.
	// +optional
	Name string `json:"name,omitempty"`

	// hostID is the host where the server is located in the cloud.
	HostID string `json:"hostID,omitempty"`

	// status contains the current operational status of the server,
	// such as IN_PROGRESS or ACTIVE.
	Status string `json:"status,omitempty"`

	// accessIPv4 contains the IPv4 addresses of the server, suitable for
	// remote access for administration.
	AccessIPv4 string `json:"accessIPv4,omitempty"`

	// accessIPv6 contains the IPv6 addresses of the server, suitable for
	// remote access for administration.
	AccessIPv6 string `json:"accessIPv6,omitempty"`

	// imageID indicates the OS image used to deploy the server.
	ImageID string `json:"imageID,omitempty"`

	// keyName indicates which public key was injected into the server on launch.
	KeyName string `json:"keyName,omitempty"`

	// securityGroups includes the security groups that this instance has
	// applied to it.
	// +kubebuilder:validation:MaxItems:=32
	// +listType=set
	SecurityGroups []string `json:"securityGroups,omitempty"`
}
