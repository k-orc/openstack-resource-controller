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

package lbpool

import (
	"testing"

	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/pools"
	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/api/v1alpha1"
	"k8s.io/utils/ptr"
)

func TestNeedsUpdate(t *testing.T) {
	testCases := []struct {
		name         string
		updateOpts   pools.UpdateOpts
		expectChange bool
	}{
		{
			name:         "Empty base opts",
			updateOpts:   pools.UpdateOpts{},
			expectChange: false,
		},
		{
			name:         "Updated opts",
			updateOpts:   pools.UpdateOpts{Name: ptr.To("updated")},
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
			resource := &orcv1alpha1.LBPool{}
			resource.Name = "object-name"
			resource.Spec = orcv1alpha1.LBPoolSpec{
				Resource: &orcv1alpha1.LBPoolResourceSpec{Name: tt.newValue},
			}
			osResource := &osResourceT{Name: tt.existingValue}

			updateOpts := pools.UpdateOpts{}
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
			resource := &orcv1alpha1.LBPoolResourceSpec{Description: tt.newValue}
			osResource := &osResourceT{Description: tt.existingValue}

			updateOpts := pools.UpdateOpts{}
			handleDescriptionUpdate(&updateOpts, resource, osResource)

			got, _ := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})
	}
}

func TestHandleAdminStateUpUpdate(t *testing.T) {
	testCases := []struct {
		name          string
		newValue      *bool
		existingValue bool
		expectChange  bool
	}{
		{name: "Identical true", newValue: ptr.To(true), existingValue: true, expectChange: false},
		{name: "Identical false", newValue: ptr.To(false), existingValue: false, expectChange: false},
		{name: "Different true to false", newValue: ptr.To(false), existingValue: true, expectChange: true},
		{name: "Different false to true", newValue: ptr.To(true), existingValue: false, expectChange: true},
		{name: "No value provided", newValue: nil, existingValue: true, expectChange: false},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resource := &orcv1alpha1.LBPoolResourceSpec{AdminStateUp: tt.newValue}
			osResource := &osResourceT{AdminStateUp: tt.existingValue}

			updateOpts := pools.UpdateOpts{}
			handleAdminStateUpUpdate(&updateOpts, resource, osResource)

			got, _ := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})
	}
}

func TestHandleSessionPersistenceUpdate(t *testing.T) {
	testCases := []struct {
		name          string
		newValue      *orcv1alpha1.LBPoolSessionPersistence
		existingValue pools.SessionPersistence
		expectChange  bool
	}{
		{
			name:          "Identical SOURCE_IP",
			newValue:      &orcv1alpha1.LBPoolSessionPersistence{Type: orcv1alpha1.LBPoolSessionPersistenceSourceIP},
			existingValue: pools.SessionPersistence{Type: "SOURCE_IP"},
			expectChange:  false,
		},
		{
			name:          "Different type",
			newValue:      &orcv1alpha1.LBPoolSessionPersistence{Type: orcv1alpha1.LBPoolSessionPersistenceHTTPCookie},
			existingValue: pools.SessionPersistence{Type: "SOURCE_IP"},
			expectChange:  true,
		},
		{
			name:          "Add session persistence",
			newValue:      &orcv1alpha1.LBPoolSessionPersistence{Type: orcv1alpha1.LBPoolSessionPersistenceSourceIP},
			existingValue: pools.SessionPersistence{},
			expectChange:  true,
		},
		{
			name:          "Remove session persistence",
			newValue:      nil,
			existingValue: pools.SessionPersistence{Type: "SOURCE_IP"},
			expectChange:  true,
		},
		{
			name:          "Both nil/empty",
			newValue:      nil,
			existingValue: pools.SessionPersistence{},
			expectChange:  false,
		},
		{
			name:          "Identical APP_COOKIE with cookie name",
			newValue:      &orcv1alpha1.LBPoolSessionPersistence{Type: orcv1alpha1.LBPoolSessionPersistenceAppCookie, CookieName: ptr.To("mycookie")},
			existingValue: pools.SessionPersistence{Type: "APP_COOKIE", CookieName: "mycookie"},
			expectChange:  false,
		},
		{
			name:          "Different cookie name",
			newValue:      &orcv1alpha1.LBPoolSessionPersistence{Type: orcv1alpha1.LBPoolSessionPersistenceAppCookie, CookieName: ptr.To("newcookie")},
			existingValue: pools.SessionPersistence{Type: "APP_COOKIE", CookieName: "oldcookie"},
			expectChange:  true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resource := &orcv1alpha1.LBPoolResourceSpec{SessionPersistence: tt.newValue}
			osResource := &osResourceT{Persistence: tt.existingValue}

			updateOpts := pools.UpdateOpts{}
			handleSessionPersistenceUpdate(&updateOpts, resource, osResource)

			got, _ := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})
	}
}

func TestHandleLBAlgorithmUpdate(t *testing.T) {
	testCases := []struct {
		name          string
		newValue      orcv1alpha1.LBPoolLBAlgorithm
		existingValue string
		expectChange  bool
	}{
		{name: "Identical", newValue: orcv1alpha1.LBPoolLBAlgorithmRoundRobin, existingValue: "ROUND_ROBIN", expectChange: false},
		{name: "Different", newValue: orcv1alpha1.LBPoolLBAlgorithmLeastConnections, existingValue: "ROUND_ROBIN", expectChange: true},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resource := &orcv1alpha1.LBPoolResourceSpec{LBAlgorithm: tt.newValue}
			osResource := &osResourceT{LBMethod: tt.existingValue}

			updateOpts := pools.UpdateOpts{}
			handleLBAlgorithmUpdate(&updateOpts, resource, osResource)

			got, _ := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})
	}
}

func TestHandleTLSUpdate(t *testing.T) {
	testCases := []struct {
		name         string
		resource     orcv1alpha1.LBPoolResourceSpec
		osResource   osResourceT
		expectChange bool
	}{
		{name: "Nothing set", expectChange: false},
		{name: "Unset settings keep the Octavia defaults", osResource: osResourceT{TLSCiphers: "AES256", TLSVersions: []string{"TLSv1.2", "TLSv1.3"}, ALPNProtocols: []string{"h2"}}, expectChange: false},
		{name: "TLS enabled", resource: orcv1alpha1.LBPoolResourceSpec{TLSEnabled: ptr.To(true)}, expectChange: true},
		{name: "TLS enabled identical", resource: orcv1alpha1.LBPoolResourceSpec{TLSEnabled: ptr.To(true)}, osResource: osResourceT{TLSEnabled: true}, expectChange: false},
		{name: "Container identical", resource: orcv1alpha1.LBPoolResourceSpec{TLSContainerURL: ptr.To("ref")}, osResource: osResourceT{TLSContainerRef: "ref"}, expectChange: false},
		{name: "Container different", resource: orcv1alpha1.LBPoolResourceSpec{TLSContainerURL: ptr.To("new-ref")}, osResource: osResourceT{TLSContainerRef: "ref"}, expectChange: true},
		{name: "CA container added", resource: orcv1alpha1.LBPoolResourceSpec{CATLSContainerURL: ptr.To("ca")}, expectChange: true},
		{name: "CRL container different", resource: orcv1alpha1.LBPoolResourceSpec{CRLContainerURL: ptr.To("crl2")}, osResource: osResourceT{CRLContainerRef: "crl"}, expectChange: true},
		{name: "Ciphers different", resource: orcv1alpha1.LBPoolResourceSpec{TLSCiphers: ptr.To("AES256")}, osResource: osResourceT{TLSCiphers: "AES128"}, expectChange: true},
		{name: "Versions in another order", resource: orcv1alpha1.LBPoolResourceSpec{TLSVersions: []orcv1alpha1.LBPoolTLSVersion{"TLSv1.3", "TLSv1.2"}}, osResource: osResourceT{TLSVersions: []string{"TLSv1.2", "TLSv1.3"}}, expectChange: false},
		{name: "Versions different", resource: orcv1alpha1.LBPoolResourceSpec{TLSVersions: []orcv1alpha1.LBPoolTLSVersion{"TLSv1.3"}}, osResource: osResourceT{TLSVersions: []string{"TLSv1.2"}}, expectChange: true},
		{name: "ALPN different", resource: orcv1alpha1.LBPoolResourceSpec{ALPNProtocols: []string{"h2"}}, osResource: osResourceT{ALPNProtocols: []string{"http/1.1"}}, expectChange: true},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			updateOpts := pools.UpdateOpts{}
			handleTLSUpdate(&updateOpts, &tt.resource, &tt.osResource)

			got, _ := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})
	}
}
