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

package listener

import (
	"testing"

	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/listeners"
	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/api/v1alpha1"
	"k8s.io/utils/ptr"
)

func TestNeedsUpdate(t *testing.T) {
	testCases := []struct {
		name         string
		updateOpts   listeners.UpdateOpts
		expectChange bool
	}{
		{
			name:         "Empty base opts",
			updateOpts:   listeners.UpdateOpts{},
			expectChange: false,
		},
		{
			name:         "Updated opts",
			updateOpts:   listeners.UpdateOpts{Name: ptr.To("updated")},
			expectChange: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := needsUpdate(tt.updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})
	}
}

func TestHandleNameUpdate(t *testing.T) {
	ptrToName := ptr.To[orcv1alpha1.OpenStackName]
	testCases := []struct {
		name          string
		newValue      *orcv1alpha1.OpenStackName
		existingValue string
		expectChange  bool
	}{
		{name: "Identical", newValue: ptrToName("name"), existingValue: "name", expectChange: false},
		{name: "Different", newValue: ptrToName("new-name"), existingValue: "name", expectChange: true},
		{name: "No value provided, existing is identical to object name", newValue: nil, existingValue: "object-name", expectChange: false},
		{name: "No value provided, existing is different from object name", newValue: nil, existingValue: "different-from-object-name", expectChange: true},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resource := &orcv1alpha1.Listener{}
			resource.Name = "object-name"
			resource.Spec = orcv1alpha1.ListenerSpec{
				Resource: &orcv1alpha1.ListenerResourceSpec{Name: tt.newValue},
			}
			osResource := &osResourceT{Name: tt.existingValue}

			updateOpts := listeners.UpdateOpts{}
			handleNameUpdate(&updateOpts, resource, osResource)

			got, _ := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})

	}
}

func TestHandleDescriptionUpdate(t *testing.T) {
	ptrToDescription := ptr.To[string]
	testCases := []struct {
		name          string
		newValue      *string
		existingValue string
		expectChange  bool
	}{
		{name: "Identical", newValue: ptrToDescription("desc"), existingValue: "desc", expectChange: false},
		{name: "Different", newValue: ptrToDescription("new-desc"), existingValue: "desc", expectChange: true},
		{name: "No value provided, existing is set", newValue: nil, existingValue: "desc", expectChange: true},
		{name: "No value provided, existing is empty", newValue: nil, existingValue: "", expectChange: false},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resource := &orcv1alpha1.ListenerResourceSpec{Description: tt.newValue}
			osResource := &osResourceT{Description: tt.existingValue}

			updateOpts := listeners.UpdateOpts{}
			handleDescriptionUpdate(&updateOpts, resource, osResource)

			got, _ := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})

	}
}

func TestHandleUpdates(t *testing.T) {
	testCases := []struct {
		name         string
		resource     orcv1alpha1.ListenerResourceSpec
		osResource   osResourceT
		expectChange bool
	}{
		{name: "Defaults match OpenStack defaults", osResource: osResourceT{AdminStateUp: true, ConnLimit: -1}, expectChange: false},
		{name: "Admin state unset, listener down", osResource: osResourceT{AdminStateUp: false, ConnLimit: -1}, expectChange: true},
		{name: "Connection limit unset, limit set", osResource: osResourceT{AdminStateUp: true, ConnLimit: 100}, expectChange: true},
		{name: "Connection limit identical", resource: orcv1alpha1.ListenerResourceSpec{ConnectionLimit: ptr.To[int32](100)}, osResource: osResourceT{AdminStateUp: true, ConnLimit: 100}, expectChange: false},
		{name: "Timeout unset is left alone", osResource: osResourceT{AdminStateUp: true, ConnLimit: -1, TimeoutClientData: 50000}, expectChange: false},
		{name: "Timeout different", resource: orcv1alpha1.ListenerResourceSpec{TimeoutClientData: ptr.To[int32](1000)}, osResource: osResourceT{AdminStateUp: true, ConnLimit: -1, TimeoutClientData: 50000}, expectChange: true},
		{name: "Insert headers nil and empty", osResource: osResourceT{AdminStateUp: true, ConnLimit: -1, InsertHeaders: map[string]string{}}, expectChange: false},
		{name: "Insert headers removed", osResource: osResourceT{AdminStateUp: true, ConnLimit: -1, InsertHeaders: map[string]string{"X-Forwarded-For": "true"}}, expectChange: true},
		{name: "Allowed CIDRs in another order", resource: orcv1alpha1.ListenerResourceSpec{AllowedCIDRs: []string{"10.0.0.0/8", "192.168.0.0/16"}}, osResource: osResourceT{AdminStateUp: true, ConnLimit: -1, AllowedCIDRs: []string{"192.168.0.0/16", "10.0.0.0/8"}}, expectChange: false},
		{name: "Allowed CIDRs removed", osResource: osResourceT{AdminStateUp: true, ConnLimit: -1, AllowedCIDRs: []string{"10.0.0.0/8"}}, expectChange: true},
		{name: "TLS versions unset are left alone", osResource: osResourceT{AdminStateUp: true, ConnLimit: -1, TLSVersions: []string{"TLSv1.2", "TLSv1.3"}}, expectChange: false},
		{name: "TLS versions different", resource: orcv1alpha1.ListenerResourceSpec{TLSVersions: []orcv1alpha1.ListenerTLSVersion{"TLSv1.3"}}, osResource: osResourceT{AdminStateUp: true, ConnLimit: -1, TLSVersions: []string{"TLSv1.2", "TLSv1.3"}}, expectChange: true},
		{name: "Client authentication identical", resource: orcv1alpha1.ListenerResourceSpec{ClientAuthentication: ptr.To(orcv1alpha1.ListenerClientAuthNone)}, osResource: osResourceT{AdminStateUp: true, ConnLimit: -1, ClientAuthentication: "NONE"}, expectChange: false},
		{name: "SNI containers removed", osResource: osResourceT{AdminStateUp: true, ConnLimit: -1, SniContainerRefs: []string{"https://barbican/v1/containers/a"}}, expectChange: true},
		{name: "HSTS max age different", resource: orcv1alpha1.ListenerResourceSpec{HSTS: &orcv1alpha1.ListenerHSTS{MaxAge: ptr.To[int32](31536000)}}, osResource: osResourceT{AdminStateUp: true, ConnLimit: -1}, expectChange: true},
		{name: "Tags in another order", resource: orcv1alpha1.ListenerResourceSpec{Tags: []orcv1alpha1.ListenerTag{"a", "b"}}, osResource: osResourceT{AdminStateUp: true, ConnLimit: -1, Tags: []string{"b", "a"}}, expectChange: false},
		{name: "Tags removed", osResource: osResourceT{AdminStateUp: true, ConnLimit: -1, Tags: []string{"a"}}, expectChange: true},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			updateOpts := listeners.UpdateOpts{}
			handleAdminStateUpdate(&updateOpts, &tt.resource, &tt.osResource)
			handleConnectionLimitUpdate(&updateOpts, &tt.resource, &tt.osResource)
			handleTimeoutsUpdate(&updateOpts, &tt.resource, &tt.osResource)
			handleInsertHeadersUpdate(&updateOpts, &tt.resource, &tt.osResource)
			handleAllowedCIDRsUpdate(&updateOpts, &tt.resource, &tt.osResource)
			handleTLSUpdate(&updateOpts, &tt.resource, &tt.osResource)
			handleHSTSUpdate(&updateOpts, &tt.resource, &tt.osResource)
			handleTagsUpdate(&updateOpts, &tt.resource, &tt.osResource)

			got, _ := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})
	}
}
