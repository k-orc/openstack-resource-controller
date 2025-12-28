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

// ListenerProtocol represents the protocol used by a listener.
// +kubebuilder:validation:Enum=HTTP;HTTPS;SCTP;PROMETHEUS;TCP;TERMINATED_HTTPS;UDP
type ListenerProtocol string

const (
	ListenerProtocolHTTP            ListenerProtocol = "HTTP"
	ListenerProtocolHTTPS           ListenerProtocol = "HTTPS"
	ListenerProtocolSCTP            ListenerProtocol = "SCTP"
	ListenerProtocolPROMETHEUS      ListenerProtocol = "PROMETHEUS"
	ListenerProtocolTCP             ListenerProtocol = "TCP"
	ListenerProtocolTerminatedHTTPS ListenerProtocol = "TERMINATED_HTTPS"
	ListenerProtocolUDP             ListenerProtocol = "UDP"
)

// ListenerClientAuthentication represents TLS client authentication mode.
// +kubebuilder:validation:Enum=NONE;OPTIONAL;MANDATORY
type ListenerClientAuthentication string

const (
	ListenerClientAuthNone      ListenerClientAuthentication = "NONE"
	ListenerClientAuthOptional  ListenerClientAuthentication = "OPTIONAL"
	ListenerClientAuthMandatory ListenerClientAuthentication = "MANDATORY"
)

// +kubebuilder:validation:MinLength:=1
// +kubebuilder:validation:MaxLength:=255
type ListenerTag string

// ListenerHSTS represents HTTP Strict Transport Security configuration.
type ListenerHSTS struct {
	// maxAge is the maximum time in seconds that the browser should remember
	// that this site is only to be accessed using HTTPS.
	// +kubebuilder:validation:Minimum=0
	// +optional
	MaxAge *int32 `json:"maxAge,omitempty"`

	// includeSubDomains specifies whether this rule applies to all subdomains.
	// +optional
	IncludeSubDomains *bool `json:"includeSubDomains,omitempty"`

	// preload specifies whether the domain should be included in browsers' preload list.
	// +optional
	Preload *bool `json:"preload,omitempty"`
}

// ListenerResourceSpec contains the desired state of the resource.
type ListenerResourceSpec struct {
	// name will be the name of the created resource. If not specified, the
	// name of the ORC object will be used.
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	// description is a human-readable description for the resource.
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=255
	// +optional
	Description *string `json:"description,omitempty"`

	// loadBalancerRef is a reference to the LoadBalancer this listener belongs to.
	// +required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="loadBalancerRef is immutable"
	LoadBalancerRef KubernetesNameRef `json:"loadBalancerRef,omitempty"`

	// protocol is the protocol the listener will use.
	// +required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="protocol is immutable"
	Protocol ListenerProtocol `json:"protocol,omitempty"`

	// protocolPort is the port on which the listener will accept connections.
	// +required
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="protocolPort is immutable"
	ProtocolPort int32 `json:"protocolPort,omitempty"`

	// adminStateUp is the administrative state of the listener, which is up (true) or down (false).
	// +optional
	AdminStateUp *bool `json:"adminStateUp,omitempty"`

	// connectionLimit is the maximum number of connections permitted for this listener.
	// Default value is -1 which represents infinite connections.
	// +kubebuilder:validation:Minimum=-1
	// +optional
	ConnectionLimit *int32 `json:"connectionLimit,omitempty"`

	// defaultTLSContainerRef is a reference to a secret containing a PKCS12 format
	// certificate/key bundle for TERMINATED_HTTPS listeners.
	// +kubebuilder:validation:MaxLength=255
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="defaultTLSContainerRef is immutable"
	DefaultTLSContainerRef *string `json:"defaultTLSContainerRef,omitempty"`

	// sniContainerRefs is a list of references to secrets containing PKCS12 format
	// certificate/key bundles for TERMINATED_HTTPS listeners using SNI.
	// +listType=set
	// +optional
	// +kubebuilder:validation:MaxItems:=25
	// +kubebuilder:validation:items:MaxLength=255
	SNIContainerRefs []string `json:"sniContainerRefs,omitempty"`

	// defaultPoolRef is a reference to the default Pool for this listener.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="defaultPoolRef is immutable"
	DefaultPoolRef *KubernetesNameRef `json:"defaultPoolRef,omitempty"`

	// insertHeaders is a dictionary of optional headers to insert into the request
	// before it is sent to the backend member.
	// +optional
	InsertHeaders map[string]string `json:"insertHeaders,omitempty"`

	// timeoutClientData is the frontend client inactivity timeout in milliseconds.
	// +kubebuilder:validation:Minimum=0
	// +optional
	TimeoutClientData *int32 `json:"timeoutClientData,omitempty"`

	// timeoutMemberConnect is the backend member connection timeout in milliseconds.
	// +kubebuilder:validation:Minimum=0
	// +optional
	TimeoutMemberConnect *int32 `json:"timeoutMemberConnect,omitempty"`

	// timeoutMemberData is the backend member inactivity timeout in milliseconds.
	// +kubebuilder:validation:Minimum=0
	// +optional
	TimeoutMemberData *int32 `json:"timeoutMemberData,omitempty"`

	// timeoutTCPInspect is the time in milliseconds to wait for additional TCP packets
	// for content inspection.
	// +kubebuilder:validation:Minimum=0
	// +optional
	TimeoutTCPInspect *int32 `json:"timeoutTCPInspect,omitempty"`

	// allowedCIDRs is a list of IPv4/IPv6 CIDRs that are permitted to connect to this listener.
	// +listType=set
	// +optional
	// +kubebuilder:validation:MaxItems:=256
	// +kubebuilder:validation:items:MaxLength=64
	AllowedCIDRs []string `json:"allowedCIDRs,omitempty"`

	// tlsCiphers is a colon-separated list of ciphers for TLS-terminated listeners.
	// +kubebuilder:validation:MaxLength=2048
	// +optional
	TLSCiphers *string `json:"tlsCiphers,omitempty"`

	// tlsVersions is a list of TLS protocol versions to be used by the listener.
	// +listType=set
	// +optional
	// +kubebuilder:validation:MaxItems:=10
	// +kubebuilder:validation:items:MaxLength=32
	TLSVersions []string `json:"tlsVersions,omitempty"`

	// alpnProtocols is a list of ALPN protocols for TLS-enabled listeners.
	// +listType=set
	// +optional
	// +kubebuilder:validation:MaxItems:=10
	// +kubebuilder:validation:items:MaxLength=32
	ALPNProtocols []string `json:"alpnProtocols,omitempty"`

	// clientAuthentication is the TLS client authentication mode.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="clientAuthentication is immutable"
	ClientAuthentication *ListenerClientAuthentication `json:"clientAuthentication,omitempty"`

	// clientCATLSContainerRef is a reference to a secret containing the CA certificate
	// for client authentication.
	// +kubebuilder:validation:MaxLength=255
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="clientCATLSContainerRef is immutable"
	ClientCATLSContainerRef *string `json:"clientCATLSContainerRef,omitempty"`

	// clientCRLContainerRef is a reference to a secret containing the CA revocation list
	// for client authentication.
	// +kubebuilder:validation:MaxLength=255
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="clientCRLContainerRef is immutable"
	ClientCRLContainerRef *string `json:"clientCRLContainerRef,omitempty"`

	// hsts is the HTTP Strict Transport Security configuration.
	// +optional
	HSTS *ListenerHSTS `json:"hsts,omitempty"`

	// tags is a list of tags which will be applied to the listener.
	// +kubebuilder:validation:MaxItems:=64
	// +listType=set
	// +optional
	Tags []ListenerTag `json:"tags,omitempty"`
}

// ListenerFilter defines an existing resource by its properties.
// +kubebuilder:validation:MinProperties:=1
type ListenerFilter struct {
	// name of the existing resource.
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	// description of the existing resource.
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=255
	// +optional
	Description *string `json:"description,omitempty"`

	// loadBalancerRef filters by the LoadBalancer this listener belongs to.
	// +optional
	LoadBalancerRef *KubernetesNameRef `json:"loadBalancerRef,omitempty"`

	// protocol filters by the protocol used by the listener.
	// +optional
	Protocol *ListenerProtocol `json:"protocol,omitempty"`

	// protocolPort filters by the port used by the listener.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +optional
	ProtocolPort *int32 `json:"protocolPort,omitempty"`

	// tags is a list of tags to filter by. If specified, the resource must
	// have all of the tags specified to be included in the result.
	// +listType=set
	// +optional
	// +kubebuilder:validation:MaxItems:=64
	Tags []ListenerTag `json:"tags,omitempty"`

	// tagsAny is a list of tags to filter by. If specified, the resource
	// must have at least one of the tags specified to be included in the
	// result.
	// +listType=set
	// +optional
	// +kubebuilder:validation:MaxItems:=64
	TagsAny []ListenerTag `json:"tagsAny,omitempty"`

	// notTags is a list of tags to filter by. If specified, resources which
	// contain all of the given tags will be excluded from the result.
	// +listType=set
	// +optional
	// +kubebuilder:validation:MaxItems:=64
	NotTags []ListenerTag `json:"notTags,omitempty"`

	// notTagsAny is a list of tags to filter by. If specified, resources
	// which contain any of the given tags will be excluded from the result.
	// +listType=set
	// +optional
	// +kubebuilder:validation:MaxItems:=64
	NotTagsAny []ListenerTag `json:"notTagsAny,omitempty"`
}

// ListenerResourceStatus represents the observed state of the resource.
type ListenerResourceStatus struct {
	// name is a human-readable name for the resource.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Name string `json:"name,omitempty"`

	// description is a human-readable description for the resource.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Description string `json:"description,omitempty"`

	// loadBalancerID is the ID of the LoadBalancer this listener belongs to.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	LoadBalancerID string `json:"loadBalancerID,omitempty"`

	// protocol is the protocol used by the listener.
	// +kubebuilder:validation:MaxLength=64
	// +optional
	Protocol string `json:"protocol,omitempty"`

	// protocolPort is the port used by the listener.
	// +optional
	ProtocolPort int32 `json:"protocolPort,omitempty"`

	// adminStateUp is the administrative state of the listener,
	// which is up (true) or down (false).
	// +optional
	AdminStateUp *bool `json:"adminStateUp,omitempty"`

	// connectionLimit is the maximum number of connections permitted for this listener.
	// +optional
	ConnectionLimit int32 `json:"connectionLimit,omitempty"`

	// defaultPoolID is the ID of the default pool for this listener.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	DefaultPoolID string `json:"defaultPoolID,omitempty"`

	// provisioningStatus is the provisioning status of the listener.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	ProvisioningStatus string `json:"provisioningStatus,omitempty"`

	// operatingStatus is the operating status of the listener.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	OperatingStatus string `json:"operatingStatus,omitempty"`

	// allowedCIDRs is the list of CIDRs permitted to connect to this listener.
	// +listType=atomic
	// +optional
	// +kubebuilder:validation:MaxItems:=256
	// +kubebuilder:validation:items:MaxLength=64
	AllowedCIDRs []string `json:"allowedCIDRs,omitempty"`

	// timeoutClientData is the frontend client inactivity timeout in milliseconds.
	// +optional
	TimeoutClientData int32 `json:"timeoutClientData,omitempty"`

	// timeoutMemberConnect is the backend member connection timeout in milliseconds.
	// +optional
	TimeoutMemberConnect int32 `json:"timeoutMemberConnect,omitempty"`

	// timeoutMemberData is the backend member inactivity timeout in milliseconds.
	// +optional
	TimeoutMemberData int32 `json:"timeoutMemberData,omitempty"`

	// timeoutTCPInspect is the time to wait for additional TCP packets in milliseconds.
	// +optional
	TimeoutTCPInspect int32 `json:"timeoutTCPInspect,omitempty"`

	// insertHeaders is a dictionary of headers inserted into the request.
	// +optional
	InsertHeaders map[string]string `json:"insertHeaders,omitempty"`

	// tags is the list of tags on the resource.
	// +listType=atomic
	// +optional
	// +kubebuilder:validation:MaxItems:=64
	// +kubebuilder:validation:items:MaxLength=255
	Tags []string `json:"tags,omitempty"`
}
