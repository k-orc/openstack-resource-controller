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

package endpoint

import (
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/endpoints"
	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	"k8s.io/utils/ptr"
)

func TestNeedsUpdate(t *testing.T) {
	testCases := []struct {
		name         string
		updateOpts   endpoints.UpdateOpts
		expectChange bool
	}{
		{
			name:         "Empty base opts",
			updateOpts:   endpoints.UpdateOpts{},
			expectChange: false,
		},
		{
			name:         "Updated opts",
			updateOpts:   endpoints.UpdateOpts{URL: "http://updated.com"},
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

func TestHandleInterfaceUpdate(t *testing.T) {
	testCases := []struct {
		name          string
		newValue      *string
		existingValue string
		expectChange  bool
	}{
		{name: "Identical", newValue: ptr.To("internal"), existingValue: "internal", expectChange: false},
		{name: "Different", newValue: ptr.To("public"), existingValue: "internal", expectChange: true},
		{name: "No value provided, existing is kept", newValue: nil, existingValue: "internal", expectChange: false},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resourceSpec := &orcv1alpha1.EndpointResourceSpec{Interface: ptr.Deref(tt.newValue, "")}
			osResource := &osResourceT{Availability: gophercloud.Availability(tt.existingValue)}

			updateOpts := endpoints.UpdateOpts{}
			handleInterfaceUpdate(&updateOpts, resourceSpec, osResource)

			got, _ := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})

	}
}

func TestHandleURLUpdate(t *testing.T) {
	testCases := []struct {
		name          string
		newValue      *string
		existingValue string
		expectChange  bool
	}{
		{name: "Identical", newValue: ptr.To("http://same.com"), existingValue: "http://same.com", expectChange: false},
		{name: "Different", newValue: ptr.To("http://different.com"), existingValue: "http://same.com", expectChange: true},
		{name: "No value provided, existing is kept", newValue: nil, existingValue: "http://same.com", expectChange: false},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resourceSpec := &orcv1alpha1.EndpointResourceSpec{URL: ptr.Deref(tt.newValue, "")}
			osResource := &osResourceT{URL: tt.existingValue}

			updateOpts := endpoints.UpdateOpts{}
			handleURLUpdate(&updateOpts, resourceSpec, osResource)

			got, _ := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})

	}
}
