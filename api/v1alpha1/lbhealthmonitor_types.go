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

// LBHealthMonitorType is the type of a health monitor.
// +kubebuilder:validation:Enum=HTTP;HTTPS;PING;SCTP;TCP;TLS-HELLO;UDP-CONNECT
type LBHealthMonitorType string

// LBHealthMonitorHTTPMethod is the HTTP method a health monitor uses.
// +kubebuilder:validation:Enum=CONNECT;DELETE;GET;HEAD;OPTIONS;PATCH;POST;PUT;TRACE
type LBHealthMonitorHTTPMethod string

// LBHealthMonitorHTTPVersion is the HTTP version a health monitor uses.
// +kubebuilder:validation:Enum="1.0";"1.1"
type LBHealthMonitorHTTPVersion string

// LBHealthMonitorResourceSpec contains the desired state of the resource.
// +kubebuilder:validation:XValidation:rule="self.timeout < self.delay",message="timeout must be less than delay"
type LBHealthMonitorResourceSpec struct {
	// name will be the name of the created resource. If not specified, the
	// name of the ORC object will be used.
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	// poolRef is a reference to the ORC LBPool which this health monitor checks.
	// A pool has at most one health monitor.
	// +required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="poolRef is immutable"
	// +orc:kustomize:ref=LBPool
	PoolRef KubernetesNameRef `json:"poolRef,omitempty"`

	// type is the type of health monitor.
	// +required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="type is immutable"
	Type LBHealthMonitorType `json:"type,omitempty"`

	// delay is the time, in seconds, between probes to members.
	// +required
	// +kubebuilder:validation:Minimum=1
	Delay int32 `json:"delay,omitempty"`

	// timeout is the maximum time, in seconds, that a probe waits to connect.
	// It must be less than delay.
	// +required
	// +kubebuilder:validation:Minimum=1
	Timeout int32 `json:"timeout,omitempty"`

	// maxRetries is the number of successful checks before a member's
	// operating status changes to ONLINE.
	// +required
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=10
	MaxRetries int32 `json:"maxRetries,omitempty"`

	// maxRetriesDown is the number of failed checks before a member's
	// operating status changes to ERROR. Octavia's default is 3.
	// +optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=10
	MaxRetriesDown *int32 `json:"maxRetriesDown,omitempty"`

	// httpMethod is the HTTP method of the probe, for HTTP and HTTPS monitors.
	// Octavia's default is GET.
	// +optional
	HTTPMethod *LBHealthMonitorHTTPMethod `json:"httpMethod,omitempty"`

	// httpVersion is the HTTP version of the probe, for HTTP and HTTPS
	// monitors. Octavia's default is 1.0.
	// +optional
	HTTPVersion *LBHealthMonitorHTTPVersion `json:"httpVersion,omitempty"`

	// expectedCodes are the HTTP status codes of a healthy member: a single
	// code (200), a list (200,202) or a range (200-204). Octavia's default is
	// 200.
	// +kubebuilder:validation:MaxLength=64
	// +kubebuilder:validation:Pattern=`^[0-9]{3}([-,][0-9]{3})*$`
	// +optional
	ExpectedCodes *string `json:"expectedCodes,omitempty"`

	// urlPath is the path of the probe, for HTTP and HTTPS monitors.
	// Octavia's default is /.
	// +kubebuilder:validation:MaxLength=255
	// +kubebuilder:validation:Pattern=`^/`
	// +optional
	URLPath *string `json:"urlPath,omitempty"`

	// domainName is injected into the Host header of the probe, for HTTP and
	// HTTPS monitors. Requires Octavia API 2.10.
	// +kubebuilder:validation:MaxLength=255
	// +optional
	DomainName *string `json:"domainName,omitempty"`

	// adminStateUp is the administrative state of the health monitor, which is
	// up (true) or down (false).
	// +optional
	AdminStateUp *bool `json:"adminStateUp,omitempty"`

	// tags is a list of tags which will be applied to the health monitor.
	// +kubebuilder:validation:MaxItems:=64
	// +listType=set
	// +optional
	Tags []LBPoolTag `json:"tags,omitempty"`
}

// LBHealthMonitorFilter defines an existing resource by its properties
// +kubebuilder:validation:MinProperties:=1
type LBHealthMonitorFilter struct {
	// name of the existing resource
	// +optional
	Name *OpenStackName `json:"name,omitempty"`

	// poolRef filters by the LBPool the health monitor checks.
	// +optional
	// +orc:kustomize:ref=LBPool
	PoolRef *KubernetesNameRef `json:"poolRef,omitempty"`

	// type filters by the type of health monitor.
	// +optional
	Type *LBHealthMonitorType `json:"type,omitempty"`

	// tags is a list of tags to filter by. If specified, the resource must
	// have all of the tags specified to be included in the result.
	// +listType=set
	// +optional
	// +kubebuilder:validation:MaxItems:=64
	Tags []LBPoolTag `json:"tags,omitempty"`
}

// LBHealthMonitorResourceStatus represents the observed state of the resource.
type LBHealthMonitorResourceStatus struct {
	// name is a Human-readable name for the resource. Might not be unique.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Name string `json:"name,omitempty"`

	// poolID is the ID of the pool the health monitor checks.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	PoolID string `json:"poolID,omitempty"`

	// type is the type of health monitor.
	// +kubebuilder:validation:MaxLength=64
	// +optional
	Type string `json:"type,omitempty"`

	// delay is the time, in seconds, between probes.
	// +optional
	Delay int32 `json:"delay,omitempty"`

	// timeout is the maximum time, in seconds, that a probe waits to connect.
	// +optional
	Timeout int32 `json:"timeout,omitempty"`

	// maxRetries is the number of successful checks before a member is ONLINE.
	// +optional
	MaxRetries int32 `json:"maxRetries,omitempty"`

	// maxRetriesDown is the number of failed checks before a member is in ERROR.
	// +optional
	MaxRetriesDown int32 `json:"maxRetriesDown,omitempty"`

	// httpMethod is the HTTP method of the probe.
	// +kubebuilder:validation:MaxLength=16
	// +optional
	HTTPMethod string `json:"httpMethod,omitempty"`

	// httpVersion is the HTTP version of the probe.
	// +kubebuilder:validation:MaxLength=8
	// +optional
	HTTPVersion string `json:"httpVersion,omitempty"`

	// expectedCodes are the HTTP status codes of a healthy member.
	// +kubebuilder:validation:MaxLength=64
	// +optional
	ExpectedCodes string `json:"expectedCodes,omitempty"`

	// urlPath is the path of the probe.
	// +kubebuilder:validation:MaxLength=255
	// +optional
	URLPath string `json:"urlPath,omitempty"`

	// domainName is the Host header of the probe.
	// +kubebuilder:validation:MaxLength=255
	// +optional
	DomainName string `json:"domainName,omitempty"`

	// adminStateUp is the administrative state of the health monitor.
	// +optional
	AdminStateUp *bool `json:"adminStateUp,omitempty"`

	// provisioningStatus is the provisioning status of the health monitor. See
	// https://docs.openstack.org/api-ref/load-balancer/v2/#provisioning-status-codes
	// +kubebuilder:validation:MaxLength=64
	// +optional
	ProvisioningStatus string `json:"provisioningStatus,omitempty"`

	// operatingStatus is the operating status of the health monitor. See
	// https://docs.openstack.org/api-ref/load-balancer/v2/#operating-status-codes
	// +kubebuilder:validation:MaxLength=64
	// +optional
	OperatingStatus string `json:"operatingStatus,omitempty"`

	// tags is the list of tags on the resource.
	// +listType=atomic
	// +optional
	// +kubebuilder:validation:MaxItems:=64
	// +kubebuilder:validation:items:MaxLength=255
	Tags []string `json:"tags,omitempty"`
}
