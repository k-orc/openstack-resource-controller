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

package a

// KubernetesNameRef is a reference to a Kubernetes object by name.
type KubernetesNameRef string

// HostID is a custom struct type (simulating ORC's HostID pattern).
// The bare ID field inside is flagged and requires a nolint comment if intentional.
type HostID struct {
	ID        string            `json:"id,omitempty"` // want `field HostID.ID references OpenStack resource by ID in spec`
	ServerRef KubernetesNameRef `json:"serverRef,omitempty"`
}

// ---- Spec structs: OpenStack IDs should be flagged ----

// UserResourceSpec is a spec struct that should be checked.
type UserResourceSpec struct {
	// Name is fine, not an OpenStack ID reference.
	Name *string `json:"name,omitempty"`

	DefaultProjectID *string `json:"defaultProjectID,omitempty"` // want `field UserResourceSpec.DefaultProjectID references OpenStack resource by ID in spec`

	// DomainRef is good - uses KubernetesNameRef.
	DomainRef *KubernetesNameRef `json:"domainRef,omitempty"`
}

// PortResourceSpec has multiple violations.
type PortResourceSpec struct {
	NetworkID *string `json:"networkID,omitempty"` // want `field PortResourceSpec.NetworkID references OpenStack resource by ID in spec`

	SubnetID *string `json:"subnetID,omitempty"` // want `field PortResourceSpec.SubnetID references OpenStack resource by ID in spec`

	// ProjectRef is correct.
	ProjectRef *KubernetesNameRef `json:"projectRef,omitempty"`
}

// ServerSpec tests shortened Spec suffix.
type ServerSpec struct {
	ImageID *string `json:"imageID,omitempty"` // want `field ServerSpec.ImageID references OpenStack resource by ID in spec`

	FlavorID *string `json:"flavorID,omitempty"` // want `field ServerSpec.FlavorID references OpenStack resource by ID in spec`
}

// ---- Filter structs: OpenStack IDs should also be flagged ----

// NetworkFilter is a filter struct that should be checked.
type NetworkFilter struct {
	ProjectID *string `json:"projectID,omitempty"` // want `field NetworkFilter.ProjectID references OpenStack resource by ID in spec`

	// Name is fine.
	Name *string `json:"name,omitempty"`
}

// ---- Status structs: OpenStack IDs are allowed ----

// UserResourceStatus is a status struct where OpenStack IDs are expected.
type UserResourceStatus struct {
	// DefaultProjectID is allowed in status - it reports what OpenStack returned.
	DefaultProjectID string `json:"defaultProjectID,omitempty"`

	// DomainID is allowed in status.
	DomainID string `json:"domainID,omitempty"`
}

// PortStatus tests shortened Status suffix.
type PortStatus struct {
	// NetworkID is allowed in status.
	NetworkID string `json:"networkID,omitempty"`
}

// ---- Nested types used in specs: should also be flagged ----

// ServerBlockDevice is a nested type used in ServerResourceSpec.
type ServerBlockDevice struct {
	VolumeID *string `json:"volumeID,omitempty"` // want `field ServerBlockDevice.VolumeID references OpenStack resource by ID in spec`

	// Device is fine.
	Device *string `json:"device,omitempty"`
}

// SecurityGroupRule is a nested type.
type SecurityGroupRule struct {
	RemoteGroupID *string `json:"remoteGroupID,omitempty"` // want `field SecurityGroupRule.RemoteGroupID references OpenStack resource by ID in spec`
}

// ---- Edge cases ----

// NonPointerIDSpec has non-pointer ID fields which should also be flagged.
type NonPointerIDSpec struct {
	ProjectID string `json:"projectID,omitempty"` // want `field NonPointerIDSpec.ProjectID references OpenStack resource by ID in spec`
}

// UnrelatedIDStruct has ID fields that don't look like OpenStack resources,
// but they are still flagged because any *ID pattern could be a reference.
// Users should add //nolint:noopenstackidref if these are intentional.
type UnrelatedIDStruct struct {
	ExternalID *string `json:"externalID,omitempty"` // want `field UnrelatedIDStruct.ExternalID references OpenStack resource by ID in spec`
}

// StructTypeIDSpec tests that struct-typed ID fields are also flagged.
type StructTypeIDSpec struct {
	HostID *HostID `json:"hostID,omitempty"` // want `field StructTypeIDSpec.HostID references OpenStack resource by ID in spec`
}

// BareIDSpec tests that bare "ID" field is also flagged.
// Use //nolint:noopenstackidref for legitimate cases like spec.import.id.
type BareIDSpec struct {
	ID *string `json:"id,omitempty"` // want `field BareIDSpec.ID references OpenStack resource by ID in spec`
}

// WrongNameCorrectTypeSpec tests that *KubernetesNameRef with wrong name is allowed.
// This is acceptable because the type is correct even if naming is unconventional.
type WrongNameCorrectTypeSpec struct {
	// ProjectID with *KubernetesNameRef type is allowed (type takes precedence).
	ProjectID *KubernetesNameRef `json:"projectID,omitempty"`
}

// ---- Plural ID fields: should also be flagged ----

// PluralIDsSpec tests that plural IDs fields are flagged.
type PluralIDsSpec struct {
	NetworkIDs []string `json:"networkIDs,omitempty"` // want `field PluralIDsSpec.NetworkIDs references OpenStack resource by ID in spec`

	SubnetIDs []string `json:"subnetIDs,omitempty"` // want `field PluralIDsSpec.SubnetIDs references OpenStack resource by ID in spec`

	// SecurityGroupRefs is correct - uses the Refs suffix.
	SecurityGroupRefs []KubernetesNameRef `json:"securityGroupRefs,omitempty"`
}

// PluralIDsStatus tests that plural IDs in status are allowed.
type PluralIDsStatus struct {
	// NetworkIDs is allowed in status.
	NetworkIDs []string `json:"networkIDs,omitempty"`
}

// ---- Ref/Refs fields with wrong type: should be flagged ----

// OpenStackName simulates the ORC OpenStackName type (wrong type for Refs).
type OpenStackName string

// WrongTypeRefSpec tests that Ref fields with wrong type are flagged.
type WrongTypeRefSpec struct {
	// ProjectRef with *string type is wrong - should use *KubernetesNameRef.
	ProjectRef *string `json:"projectRef,omitempty"` // want `field WrongTypeRefSpec.ProjectRef has Ref suffix but does not use KubernetesNameRef type`

	// NetworkRef with OpenStackName type is wrong - should use KubernetesNameRef.
	NetworkRef OpenStackName `json:"networkRef,omitempty"` // want `field WrongTypeRefSpec.NetworkRef has Ref suffix but does not use KubernetesNameRef type`

	// SubnetRef is correct - uses KubernetesNameRef.
	SubnetRef KubernetesNameRef `json:"subnetRef,omitempty"`

	// RouterRef is correct - uses *KubernetesNameRef.
	RouterRef *KubernetesNameRef `json:"routerRef,omitempty"`
}

// WrongTypeRefsSpec tests that plural Refs fields with wrong type are flagged.
type WrongTypeRefsSpec struct {
	// SecurityGroupRefs with []OpenStackName type is wrong - should use []KubernetesNameRef.
	SecurityGroupRefs []OpenStackName `json:"securityGroupRefs,omitempty"` // want `field WrongTypeRefsSpec.SecurityGroupRefs has Ref suffix but does not use KubernetesNameRef type`

	// NetworkRefs with []string type is wrong - should use []KubernetesNameRef.
	NetworkRefs []string `json:"networkRefs,omitempty"` // want `field WrongTypeRefsSpec.NetworkRefs has Ref suffix but does not use KubernetesNameRef type`

	// SubnetRefs is correct - uses []KubernetesNameRef.
	SubnetRefs []KubernetesNameRef `json:"subnetRefs,omitempty"`
}

// WrongTypeRefsStatus tests that Refs in status with wrong type are allowed.
type WrongTypeRefsStatus struct {
	// SecurityGroupRefs is allowed in status even with wrong type.
	SecurityGroupRefs []OpenStackName `json:"securityGroupRefs,omitempty"`
}
