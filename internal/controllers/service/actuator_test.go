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

package service

import (
	"testing"

	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/services"
	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	"k8s.io/utils/ptr"
)

func TestNeedsUpdate(t *testing.T) {
	testCases := []struct {
		name         string
		updateOpts   services.UpdateOpts
		expectChange bool
	}{
		{
			name:         "Empty base opts",
			updateOpts:   services.UpdateOpts{},
			expectChange: false,
		},
		{
			name:         "Updated type opt",
			updateOpts:   services.UpdateOpts{Type: "updated"},
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

func TestHandleTypeUpdate(t *testing.T) {
	testCases := []struct {
		name           string
		newValue       *string
		existingValue  string
		expectedChange bool
	}{
		{name: "Identical", newValue: ptr.To("service"), existingValue: "service", expectedChange: false},
		{name: "Different", newValue: ptr.To("new-service"), existingValue: "service", expectedChange: true},
		{name: "No value provided, existing is set", newValue: nil, existingValue: "service", expectedChange: false},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resource := &orcv1alpha1.ServiceResourceSpec{Type: tt.newValue}
			osResource := &osResourceT{Type: tt.existingValue}

			updateOpts := services.UpdateOpts{}
			handleTypeUpdate(&updateOpts, resource, osResource)

			if got, _ := needsUpdate(updateOpts); got != tt.expectedChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectedChange, got)
			}
		})
	}
}

func TestHandleEnabledUpdate(t *testing.T) {
	testCases := []struct {
		name           string
		newValue       *bool
		existingValue  bool
		expectedChange bool
	}{
		{name: "Identical", newValue: ptr.To(true), existingValue: true, expectedChange: false},
		{name: "Different", newValue: ptr.To(false), existingValue: true, expectedChange: true},
		{name: "No value provided, existing is set", newValue: nil, existingValue: true, expectedChange: false},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resource := &orcv1alpha1.ServiceResourceSpec{Enabled: tt.newValue}
			osResource := &osResourceT{Enabled: tt.existingValue}

			updateOpts := services.UpdateOpts{}
			handleEnabledUpdate(&updateOpts, resource, osResource)

			if got, _ := needsUpdate(updateOpts); got != tt.expectedChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectedChange, got)
			}
		})
	}
}

func TestHandleExtraUpdate(t *testing.T) {
	testCases := []struct {
		name           string
		newValue       map[string]any
		existingValue  map[string]any
		expectedChange bool
	}{
		{name: "Identical", newValue: map[string]any{"name": "service"}, existingValue: map[string]any{"name": "service"}, expectedChange: false},
		{name: "Different", newValue: map[string]any{"name": "new-service"}, existingValue: map[string]any{"name": "service"}, expectedChange: true},
		{name: "No value provided, existing is set", newValue: nil, existingValue: map[string]any{"name": "service"}, expectedChange: false},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resource := &orcv1alpha1.ServiceResourceSpec{Extra: tt.newValue}
			osResource := &osResourceT{Extra: tt.existingValue}

			updateOpts := services.UpdateOpts{}
			handleExtraUpdate(&updateOpts, resource, osResource)

			if got, _ := needsUpdate(updateOpts); got != tt.expectedChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectedChange, got)
			}
		})
	}
}