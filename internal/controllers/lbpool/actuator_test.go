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

package lbpool

import (
	"testing"

	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/pools"
	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
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

func TestHandleTLSContainerRefUpdate(t *testing.T) {
	testCases := []struct {
		name          string
		newValue      *string
		existingValue string
		expectChange  bool
	}{
		{name: "Identical", newValue: ptr.To("ref"), existingValue: "ref", expectChange: false},
		{name: "Different", newValue: ptr.To("new-ref"), existingValue: "ref", expectChange: true},
		{name: "Add ref", newValue: ptr.To("ref"), existingValue: "", expectChange: true},
		{name: "Remove ref", newValue: nil, existingValue: "ref", expectChange: true},
		{name: "Both empty", newValue: nil, existingValue: "", expectChange: false},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resource := &orcv1alpha1.LBPoolResourceSpec{TLSContainerRef: tt.newValue}
			osResource := &osResourceT{TLSContainerRef: tt.existingValue}

			updateOpts := pools.UpdateOpts{}
			handleTLSContainerRefUpdate(&updateOpts, resource, osResource)

			got, _ := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})
	}
}

func TestHandleTLSCiphersUpdate(t *testing.T) {
	testCases := []struct {
		name          string
		newValue      *string
		existingValue string
		expectChange  bool
	}{
		{name: "Identical", newValue: ptr.To("AES256:AES128"), existingValue: "AES256:AES128", expectChange: false},
		{name: "Different", newValue: ptr.To("AES256"), existingValue: "AES128", expectChange: true},
		{name: "Add ciphers", newValue: ptr.To("AES256"), existingValue: "", expectChange: true},
		{name: "Remove ciphers", newValue: nil, existingValue: "AES256", expectChange: true},
		{name: "Both empty", newValue: nil, existingValue: "", expectChange: false},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resource := &orcv1alpha1.LBPoolResourceSpec{TLSCiphers: tt.newValue}
			osResource := &osResourceT{TLSCiphers: tt.existingValue}

			updateOpts := pools.UpdateOpts{}
			handleTLSCiphersUpdate(&updateOpts, resource, osResource)

			got, _ := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})
	}
}

func TestHandleTLSVersionsUpdate(t *testing.T) {
	testCases := []struct {
		name          string
		newValue      []string
		existingValue []string
		expectChange  bool
	}{
		{name: "Identical", newValue: []string{"TLSv1.2", "TLSv1.3"}, existingValue: []string{"TLSv1.2", "TLSv1.3"}, expectChange: false},
		{name: "Different", newValue: []string{"TLSv1.3"}, existingValue: []string{"TLSv1.2"}, expectChange: true},
		{name: "Add versions", newValue: []string{"TLSv1.2"}, existingValue: nil, expectChange: true},
		{name: "Remove versions", newValue: nil, existingValue: []string{"TLSv1.2"}, expectChange: true},
		{name: "Both empty", newValue: nil, existingValue: nil, expectChange: false},
		{name: "Different order", newValue: []string{"TLSv1.3", "TLSv1.2"}, existingValue: []string{"TLSv1.2", "TLSv1.3"}, expectChange: true},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resource := &orcv1alpha1.LBPoolResourceSpec{TLSVersions: tt.newValue}
			osResource := &osResourceT{TLSVersions: tt.existingValue}

			updateOpts := pools.UpdateOpts{}
			handleTLSVersionsUpdate(&updateOpts, resource, osResource)

			got, _ := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})
	}
}

func TestHandleALPNProtocolsUpdate(t *testing.T) {
	testCases := []struct {
		name          string
		newValue      []string
		existingValue []string
		expectChange  bool
	}{
		{name: "Identical", newValue: []string{"h2", "http/1.1"}, existingValue: []string{"h2", "http/1.1"}, expectChange: false},
		{name: "Different", newValue: []string{"h2"}, existingValue: []string{"http/1.1"}, expectChange: true},
		{name: "Add protocols", newValue: []string{"h2"}, existingValue: nil, expectChange: true},
		{name: "Remove protocols", newValue: nil, existingValue: []string{"h2"}, expectChange: true},
		{name: "Both empty", newValue: nil, existingValue: nil, expectChange: false},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resource := &orcv1alpha1.LBPoolResourceSpec{ALPNProtocols: tt.newValue}
			osResource := &osResourceT{ALPNProtocols: tt.existingValue}

			updateOpts := pools.UpdateOpts{}
			handleALPNProtocolsUpdate(&updateOpts, resource, osResource)

			got, _ := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})
	}
}
