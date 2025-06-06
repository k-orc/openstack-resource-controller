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

// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1alpha1

import (
	apiv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
)

// ImageSpecApplyConfiguration represents a declarative configuration of the ImageSpec type for use
// with apply.
type ImageSpecApplyConfiguration struct {
	Import              *ImageImportApplyConfiguration               `json:"import,omitempty"`
	Resource            *ImageResourceSpecApplyConfiguration         `json:"resource,omitempty"`
	ManagementPolicy    *apiv1alpha1.ManagementPolicy                `json:"managementPolicy,omitempty"`
	ManagedOptions      *ManagedOptionsApplyConfiguration            `json:"managedOptions,omitempty"`
	CloudCredentialsRef *CloudCredentialsReferenceApplyConfiguration `json:"cloudCredentialsRef,omitempty"`
}

// ImageSpecApplyConfiguration constructs a declarative configuration of the ImageSpec type for use with
// apply.
func ImageSpec() *ImageSpecApplyConfiguration {
	return &ImageSpecApplyConfiguration{}
}

// WithImport sets the Import field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Import field is set to the value of the last call.
func (b *ImageSpecApplyConfiguration) WithImport(value *ImageImportApplyConfiguration) *ImageSpecApplyConfiguration {
	b.Import = value
	return b
}

// WithResource sets the Resource field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Resource field is set to the value of the last call.
func (b *ImageSpecApplyConfiguration) WithResource(value *ImageResourceSpecApplyConfiguration) *ImageSpecApplyConfiguration {
	b.Resource = value
	return b
}

// WithManagementPolicy sets the ManagementPolicy field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ManagementPolicy field is set to the value of the last call.
func (b *ImageSpecApplyConfiguration) WithManagementPolicy(value apiv1alpha1.ManagementPolicy) *ImageSpecApplyConfiguration {
	b.ManagementPolicy = &value
	return b
}

// WithManagedOptions sets the ManagedOptions field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ManagedOptions field is set to the value of the last call.
func (b *ImageSpecApplyConfiguration) WithManagedOptions(value *ManagedOptionsApplyConfiguration) *ImageSpecApplyConfiguration {
	b.ManagedOptions = value
	return b
}

// WithCloudCredentialsRef sets the CloudCredentialsRef field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the CloudCredentialsRef field is set to the value of the last call.
func (b *ImageSpecApplyConfiguration) WithCloudCredentialsRef(value *CloudCredentialsReferenceApplyConfiguration) *ImageSpecApplyConfiguration {
	b.CloudCredentialsRef = value
	return b
}
