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

// +kubebuilder:validation:MinLength:=1
// +kubebuilder:validation:MaxLength:=80
type ServerTag string

type FilterByServerTags struct {
	// tags is a list of tags to filter by. If specified, the resource must
	// have all of the tags specified to be included in the result.
	// +listType=set
	// +optional
	// +kubebuilder:validation:MaxItems:=32
	Tags []ServerTag `json:"tags,omitempty"`

	// tagsAny is a list of tags to filter by. If specified, the resource
	// must have at least one of the tags specified to be included in the
	// result.
	// +listType=set
	// +optional
	// +kubebuilder:validation:MaxItems:=32
	TagsAny []ServerTag `json:"tagsAny,omitempty"`

	// notTags is a list of tags to filter by. If specified, resources which
	// contain all of the given tags will be excluded from the result.
	// +listType=set
	// +optional
	// +kubebuilder:validation:MaxItems:=32
	NotTags []ServerTag `json:"notTags,omitempty"`

	// notTagsAny is a list of tags to filter by. If specified, resources
	// which contain any of the given tags will be excluded from the result.
	// +listType=set
	// +optional
	// +kubebuilder:validation:MaxItems:=32
	NotTagsAny []ServerTag `json:"notTagsAny,omitempty"`
}

// +kubebuilder:validation:MinProperties:=1
// +kubebuilder:validation:MaxProperties:=1
type ServerPortSpec struct {
	// portRef is a reference to a Port object. Server creation will wait for
	// this port to be created and available.
	// +optional
	PortRef *KubernetesNameRef `json:"portRef,omitempty"`
}

// ServerResourceSpec contains the desired state of a server
type ServerResourceSpec struct {
	// name will be the name of the created resource. If not specified, the
	// name of the ORC object will be used.
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	// imageRef references the image to use for the server instance.
	// NOTE: This is not required in case of boot from volume.
	// +required
	ImageRef KubernetesNameRef `json:"imageRef"`

	// flavorRef references the flavor to use for the server instance.
	// +required
	FlavorRef KubernetesNameRef `json:"flavorRef"`

	// userData specifies data which will be made available to the server at
	// boot time, either via the metadata service or a config drive. It is
	// typically read by a configuration service such as cloud-init or ignition.
	// +optional
	UserData *UserDataSpec `json:"userData,omitempty"`

	// ports defines a list of ports which will be attached to the server.
	// +kubebuilder:validation:MaxItems:=32
	// +listType=atomic
	// +optional
	Ports []ServerPortSpec `json:"ports,omitempty"`

	// tags is a list of tags which will be applied to the server.
	// +kubebuilder:validation:MaxItems:=32
	// +listType=set
	// +optional
	Tags []ServerTag `json:"tags,omitempty"`
}

// +kubebuilder:validation:MinProperties:=1
// +kubebuilder:validation:MaxProperties:=1
type UserDataSpec struct {
	// secretRef is a reference to a Secret containing the user data for this server.
	// +optional
	SecretRef *KubernetesNameRef `json:"secretRef,omitempty"`
}

// ServerFilter defines an existing resource by its properties
// +kubebuilder:validation:MinProperties:=1
type ServerFilter struct {
	// name of the existing resource
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	FilterByServerTags `json:",inline"`
}

// ServerResourceStatus represents the observed state of the resource.
type ServerResourceStatus struct {
	// name is the human-readable name of the resource. Might not be unique.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Name string `json:"name,omitempty"`

	// hostID is the host where the server is located in the cloud.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	HostID string `json:"hostID,omitempty"`

	// status contains the current operational status of the server,
	// such as IN_PROGRESS or ACTIVE.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Status string `json:"status,omitempty"`

	// accessIPv4 contains the IPv4 addresses of the server, suitable for
	// remote access for administration.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	AccessIPv4 string `json:"accessIPv4,omitempty"`

	// accessIPv6 contains the IPv6 addresses of the server, suitable for
	// remote access for administration.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	AccessIPv6 string `json:"accessIPv6,omitempty"`

	// imageID indicates the OS image used to deploy the server.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	ImageID string `json:"imageID,omitempty"`

	// keyName indicates which public key was injected into the server on launch.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	KeyName string `json:"keyName,omitempty"`

	// securityGroups includes the security groups that this instance has
	// applied to it.
	// +kubebuilder:validation:MaxItems:=32
	// +kubebuilder:validation:items:MaxLength=1024
	// +listType=atomic
	// +optional
	SecurityGroups []string `json:"securityGroups,omitempty"`

	// tags is the list of tags on the resource.
	// +kubebuilder:validation:MaxItems:=32
	// +kubebuilder:validation:items:MaxLength=1024
	// +listType=atomic
	// +optional
	Tags []string `json:"tags,omitempty"`
}
