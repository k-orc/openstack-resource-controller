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

// LBPoolProtocol represents the protocol used by a pool.
// +kubebuilder:validation:Enum=HTTP;HTTPS;PROXY;PROXYV2;SCTP;TCP;UDP
type LBPoolProtocol string

const (
	LBPoolProtocolHTTP    LBPoolProtocol = "HTTP"
	LBPoolProtocolHTTPS   LBPoolProtocol = "HTTPS"
	LBPoolProtocolPROXY   LBPoolProtocol = "PROXY"
	LBPoolProtocolPROXYV2 LBPoolProtocol = "PROXYV2"
	LBPoolProtocolSCTP    LBPoolProtocol = "SCTP"
	LBPoolProtocolTCP     LBPoolProtocol = "TCP"
	LBPoolProtocolUDP     LBPoolProtocol = "UDP"
)

// LBPoolLBAlgorithm represents the load balancing algorithm used by a pool.
// +kubebuilder:validation:Enum=LEAST_CONNECTIONS;ROUND_ROBIN;SOURCE_IP;SOURCE_IP_PORT
type LBPoolLBAlgorithm string

const (
	LBPoolLBAlgorithmLeastConnections LBPoolLBAlgorithm = "LEAST_CONNECTIONS"
	LBPoolLBAlgorithmRoundRobin       LBPoolLBAlgorithm = "ROUND_ROBIN"
	LBPoolLBAlgorithmSourceIP         LBPoolLBAlgorithm = "SOURCE_IP"
	LBPoolLBAlgorithmSourceIPPort     LBPoolLBAlgorithm = "SOURCE_IP_PORT"
)

// LBPoolSessionPersistenceType represents the type of session persistence.
// +kubebuilder:validation:Enum=APP_COOKIE;HTTP_COOKIE;SOURCE_IP
type LBPoolSessionPersistenceType string

const (
	// LBPoolSessionPersistenceAppCookie relies on a cookie established by the backend application.
	LBPoolSessionPersistenceAppCookie LBPoolSessionPersistenceType = "APP_COOKIE"
	// LBPoolSessionPersistenceHTTPCookie causes the load balancer to create a cookie on first request.
	LBPoolSessionPersistenceHTTPCookie LBPoolSessionPersistenceType = "HTTP_COOKIE"
	// LBPoolSessionPersistenceSourceIP routes connections from the same source IP to the same member.
	LBPoolSessionPersistenceSourceIP LBPoolSessionPersistenceType = "SOURCE_IP"
)

// LBPoolSessionPersistence represents session persistence configuration for a pool.
type LBPoolSessionPersistence struct {
	// type is the type of session persistence.
	// +required
	Type LBPoolSessionPersistenceType `json:"type,omitempty"`

	// cookieName is the name of the cookie if persistence type is APP_COOKIE.
	// Required when type is APP_COOKIE.
	// +kubebuilder:validation:MaxLength=255
	// +optional
	CookieName *string `json:"cookieName,omitempty"`
}

// +kubebuilder:validation:MinLength:=1
// +kubebuilder:validation:MaxLength:=255
type LBPoolTag string

// LBPoolMemberSpec defines a member of an LB pool.
type LBPoolMemberSpec struct {
	// name is a human-readable name for the member.
	// +kubebuilder:validation:MaxLength=255
	// +optional
	Name *string `json:"name,omitempty"`

	// address is the IP address of the member to receive traffic.
	// +required
	Address IPvAny `json:"address,omitempty"`

	// protocolPort is the port on which the member application is listening.
	// +required
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	ProtocolPort int32 `json:"protocolPort,omitempty"`

	// subnetRef is a reference to the ORC Subnet where the member resides.
	// +optional
	SubnetRef *KubernetesNameRef `json:"subnetRef,omitempty"`

	// weight is the relative portion of traffic this member should receive.
	// A member with weight 10 receives 5x the traffic of a member with weight 2.
	// Default is 1.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=256
	// +optional
	Weight *int32 `json:"weight,omitempty"`

	// backup indicates whether this is a backup member. Backup members only
	// receive traffic when all non-backup members are down.
	// +optional
	Backup *bool `json:"backup,omitempty"`

	// adminStateUp is the administrative state of the member (up=true, down=false).
	// +optional
	AdminStateUp *bool `json:"adminStateUp,omitempty"`
}

// LBPoolMemberStatus represents the observed state of a pool member.
type LBPoolMemberStatus struct {
	// id is the unique identifier of the member in OpenStack.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	ID string `json:"id,omitempty"`

	// name is the human-readable name for the member.
	// +kubebuilder:validation:MaxLength=255
	// +optional
	Name string `json:"name,omitempty"`

	// address is the IP address of the member.
	// +kubebuilder:validation:MaxLength=64
	// +optional
	Address string `json:"address,omitempty"`

	// protocolPort is the port on which the member is listening.
	// +optional
	ProtocolPort int32 `json:"protocolPort,omitempty"`

	// subnetID is the ID of the subnet the member is on.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	SubnetID string `json:"subnetID,omitempty"`

	// weight is the weight of the member for load balancing.
	// +optional
	Weight int32 `json:"weight,omitempty"`

	// backup indicates whether this is a backup member.
	// +optional
	Backup bool `json:"backup,omitempty"`

	// adminStateUp is the administrative state of the member.
	// +optional
	AdminStateUp bool `json:"adminStateUp,omitempty"`

	// provisioningStatus is the provisioning status of the member.
	// +kubebuilder:validation:MaxLength=64
	// +optional
	ProvisioningStatus string `json:"provisioningStatus,omitempty"`

	// operatingStatus is the operating status of the member.
	// +kubebuilder:validation:MaxLength=64
	// +optional
	OperatingStatus string `json:"operatingStatus,omitempty"`
}

// LBPoolResourceSpec contains the desired state of the resource.
// +kubebuilder:validation:XValidation:rule="has(self.loadBalancerRef) || has(self.listenerRef)",message="either loadBalancerRef or listenerRef must be specified"
type LBPoolResourceSpec struct {
	// name will be the name of the created resource. If not specified, the
	// name of the ORC object will be used.
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	// description is a human-readable description for the resource.
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=255
	// +optional
	Description *string `json:"description,omitempty"`

	// lbAlgorithm is the load balancing algorithm used to distribute traffic
	// to the pool's members.
	// +required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="lbAlgorithm is immutable"
	LBAlgorithm LBPoolLBAlgorithm `json:"lbAlgorithm,omitempty"`

	// protocol is the protocol used by the pool and its members for traffic.
	// +required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="protocol is immutable"
	Protocol LBPoolProtocol `json:"protocol,omitempty"`

	// loadBalancerRef is a reference to the ORC LoadBalancer which this pool
	// is associated with. Either loadBalancerRef or listenerRef must be specified.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="loadBalancerRef is immutable"
	LoadBalancerRef *KubernetesNameRef `json:"loadBalancerRef,omitempty"`

	// listenerRef is a reference to the ORC Listener which this pool is
	// associated with as the default pool. Either loadBalancerRef or listenerRef
	// must be specified.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="listenerRef is immutable"
	ListenerRef *KubernetesNameRef `json:"listenerRef,omitempty"`

	// projectRef is a reference to the ORC Project which this resource is associated with.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="projectRef is immutable"
	ProjectRef *KubernetesNameRef `json:"projectRef,omitempty"`

	// adminStateUp is the administrative state of the pool, which is up (true)
	// or down (false).
	// +optional
	AdminStateUp *bool `json:"adminStateUp,omitempty"`

	// sessionPersistence is the session persistence configuration for the pool.
	// +optional
	SessionPersistence *LBPoolSessionPersistence `json:"sessionPersistence,omitempty"`

	// tlsEnabled enables backend re-encryption when set to true. Requires
	// TERMINATED_HTTPS or HTTPS protocol on the listener.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="tlsEnabled is immutable"
	TLSEnabled *bool `json:"tlsEnabled,omitempty"`

	// tlsContainerRef is a reference to a secret containing a PKCS12 format
	// certificate/key bundle for backend re-encryption.
	// +kubebuilder:validation:MaxLength=255
	// +optional
	TLSContainerRef *string `json:"tlsContainerRef,omitempty"`

	// caTLSContainerRef is a reference to a secret containing the CA
	// certificate for backend re-encryption.
	// +kubebuilder:validation:MaxLength=255
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="caTLSContainerRef is immutable"
	CATLSContainerRef *string `json:"caTLSContainerRef,omitempty"`

	// crlContainerRef is a reference to a secret containing the CA
	// revocation list for backend re-encryption.
	// +kubebuilder:validation:MaxLength=255
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="crlContainerRef is immutable"
	CRLContainerRef *string `json:"crlContainerRef,omitempty"`

	// tlsCiphers is a colon-separated list of ciphers for backend TLS connections.
	// +kubebuilder:validation:MaxLength=2048
	// +optional
	TLSCiphers *string `json:"tlsCiphers,omitempty"`

	// tlsVersions is a list of TLS protocol versions to be used for backend
	// TLS connections.
	// +listType=set
	// +optional
	// +kubebuilder:validation:MaxItems:=10
	// +kubebuilder:validation:items:MaxLength=32
	TLSVersions []string `json:"tlsVersions,omitempty"`

	// alpnProtocols is a list of ALPN protocols for backend TLS connections.
	// Available protocols: h2, http/1.0, http/1.1.
	// +listType=set
	// +optional
	// +kubebuilder:validation:MaxItems:=10
	// +kubebuilder:validation:items:MaxLength=32
	ALPNProtocols []string `json:"alpnProtocols,omitempty"`

	// tags is a list of tags which will be applied to the pool.
	// +kubebuilder:validation:MaxItems:=64
	// +listType=set
	// +optional
	Tags []LBPoolTag `json:"tags,omitempty"`

	// members is a list of backend members for this pool.
	// +kubebuilder:validation:MaxItems:=256
	// +listType=atomic
	// +optional
	Members []LBPoolMemberSpec `json:"members,omitempty"`
}

// LBPoolFilter defines an existing resource by its properties
// +kubebuilder:validation:MinProperties:=1
type LBPoolFilter struct {
	// name of the existing resource
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	// description of the existing resource
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=255
	// +optional
	Description *string `json:"description,omitempty"`

	// loadBalancerRef filters by the LoadBalancer this pool is associated with.
	// +optional
	LoadBalancerRef *KubernetesNameRef `json:"loadBalancerRef,omitempty"`

	// listenerRef filters by the Listener this pool is associated with.
	// +optional
	ListenerRef *KubernetesNameRef `json:"listenerRef,omitempty"`

	// projectRef filters by the Project this pool is associated with.
	// +optional
	ProjectRef *KubernetesNameRef `json:"projectRef,omitempty"`

	// lbAlgorithm filters by the load balancing algorithm.
	// +optional
	LBAlgorithm *LBPoolLBAlgorithm `json:"lbAlgorithm,omitempty"`

	// protocol filters by the protocol used by the pool.
	// +optional
	Protocol *LBPoolProtocol `json:"protocol,omitempty"`

	// tags is a list of tags to filter by. If specified, the resource must
	// have all of the tags specified to be included in the result.
	// +listType=set
	// +optional
	// +kubebuilder:validation:MaxItems:=64
	Tags []LBPoolTag `json:"tags,omitempty"`

	// tagsAny is a list of tags to filter by. If specified, the resource
	// must have at least one of the tags specified to be included in the
	// result.
	// +listType=set
	// +optional
	// +kubebuilder:validation:MaxItems:=64
	TagsAny []LBPoolTag `json:"tagsAny,omitempty"`

	// notTags is a list of tags to filter by. If specified, resources which
	// contain all of the given tags will be excluded from the result.
	// +listType=set
	// +optional
	// +kubebuilder:validation:MaxItems:=64
	NotTags []LBPoolTag `json:"notTags,omitempty"`

	// notTagsAny is a list of tags to filter by. If specified, resources
	// which contain any of the given tags will be excluded from the result.
	// +listType=set
	// +optional
	// +kubebuilder:validation:MaxItems:=64
	NotTagsAny []LBPoolTag `json:"notTagsAny,omitempty"`
}

// LBPoolResourceStatus represents the observed state of the resource.
type LBPoolResourceStatus struct {
	// name is a Human-readable name for the resource. Might not be unique.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Name string `json:"name,omitempty"`

	// description is a human-readable description for the resource.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Description string `json:"description,omitempty"`

	// lbAlgorithm is the load balancing algorithm used by the pool.
	// +kubebuilder:validation:MaxLength=64
	// +optional
	LBAlgorithm string `json:"lbAlgorithm,omitempty"`

	// protocol is the protocol used by the pool.
	// +kubebuilder:validation:MaxLength=64
	// +optional
	Protocol string `json:"protocol,omitempty"`

	// loadBalancerIDs is the list of LoadBalancer IDs this pool is associated with.
	// +listType=atomic
	// +optional
	// +kubebuilder:validation:MaxItems:=64
	// +kubebuilder:validation:items:MaxLength=1024
	LoadBalancerIDs []string `json:"loadBalancerIDs,omitempty"`

	// listenerIDs is the list of Listener IDs this pool is associated with.
	// +listType=atomic
	// +optional
	// +kubebuilder:validation:MaxItems:=64
	// +kubebuilder:validation:items:MaxLength=1024
	ListenerIDs []string `json:"listenerIDs,omitempty"`

	// projectID is the ID of the Project this pool is associated with.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	ProjectID string `json:"projectID,omitempty"`

	// adminStateUp is the administrative state of the pool,
	// which is up (true) or down (false).
	// +optional
	AdminStateUp *bool `json:"adminStateUp,omitempty"`

	// provisioningStatus is the provisioning status of the pool.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	ProvisioningStatus string `json:"provisioningStatus,omitempty"`

	// operatingStatus is the operating status of the pool.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	OperatingStatus string `json:"operatingStatus,omitempty"`

	// healthMonitorID is the ID of the health monitor associated with this pool.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	HealthMonitorID string `json:"healthMonitorID,omitempty"`

	// members is the list of members in this pool with their details.
	// +listType=atomic
	// +optional
	// +kubebuilder:validation:MaxItems:=256
	Members []LBPoolMemberStatus `json:"members,omitempty"`

	// sessionPersistence is the session persistence configuration.
	// +optional
	SessionPersistence *LBPoolSessionPersistence `json:"sessionPersistence,omitempty"`

	// tlsEnabled indicates whether backend re-encryption is enabled.
	// +optional
	TLSEnabled *bool `json:"tlsEnabled,omitempty"`

	// tlsContainerRef is the reference to the TLS container.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	TLSContainerRef string `json:"tlsContainerRef,omitempty"`

	// caTLSContainerRef is the reference to the CA TLS container.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	CATLSContainerRef string `json:"caTLSContainerRef,omitempty"`

	// crlContainerRef is the reference to the CRL container.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	CRLContainerRef string `json:"crlContainerRef,omitempty"`

	// tlsCiphers is the list of TLS ciphers for backend connections.
	// +kubebuilder:validation:MaxLength=2048
	// +optional
	TLSCiphers string `json:"tlsCiphers,omitempty"`

	// tlsVersions is the list of TLS versions for backend connections.
	// +listType=atomic
	// +optional
	// +kubebuilder:validation:MaxItems:=10
	// +kubebuilder:validation:items:MaxLength=32
	TLSVersions []string `json:"tlsVersions,omitempty"`

	// alpnProtocols is the list of ALPN protocols for backend connections.
	// +listType=atomic
	// +optional
	// +kubebuilder:validation:MaxItems:=10
	// +kubebuilder:validation:items:MaxLength=32
	ALPNProtocols []string `json:"alpnProtocols,omitempty"`

	// tags is the list of tags on the resource.
	// +listType=atomic
	// +optional
	// +kubebuilder:validation:MaxItems:=64
	// +kubebuilder:validation:items:MaxLength=255
	Tags []string `json:"tags,omitempty"`
}
