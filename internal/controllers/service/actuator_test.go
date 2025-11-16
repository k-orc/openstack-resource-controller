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
		name          string
		newValue      string
		existingValue string
		expectChange  bool
	}{
		{name: "Identical", newValue: "service", existingValue: "service", expectChange: false},
		{name: "Different", newValue: "new-service", existingValue: "service", expectChange: true},
		{name: "No value provided, existing is set", newValue: "", existingValue: "service", expectChange: false},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resource := &orcv1alpha1.ServiceResourceSpec{Type: tt.newValue}
			osResource := &osResourceT{Type: tt.existingValue}

			updateOpts := services.UpdateOpts{}
			handleTypeUpdate(&updateOpts, resource, osResource)

			if got, _ := needsUpdate(updateOpts); got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})
	}
}

func TestHandleEnabledUpdate(t *testing.T) {
	testCases := []struct {
		name          string
		newValue      *bool
		existingValue bool
		expectChange  bool
	}{
		{name: "Identical", newValue: ptr.To(true), existingValue: true, expectChange: false},
		{name: "Different", newValue: ptr.To(false), existingValue: true, expectChange: true},
		{name: "No value provided, existing is set", newValue: nil, existingValue: true, expectChange: false},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resource := &orcv1alpha1.ServiceResourceSpec{Enabled: tt.newValue}
			osResource := &osResourceT{Enabled: tt.existingValue}

			updateOpts := services.UpdateOpts{}
			handleEnabledUpdate(&updateOpts, resource, osResource)

			if got, _ := needsUpdate(updateOpts); got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})
	}
}

func TestHandleNameUpdate(t *testing.T) {
	testCases := []struct {
		name          string
		newValue      *string
		existingValue string
		expectChange  bool
	}{
		{name: "Identical", newValue: ptr.To("same-name"), existingValue: "same-name", expectChange: false},
		{name: "Different", newValue: ptr.To("new-name"), existingValue: "same-name", expectChange: true},
		{name: "No value provided, existing is set", newValue: nil, existingValue: "service-name", expectChange: true},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			obj := &orcv1alpha1.Service{Spec: orcv1alpha1.ServiceSpec{
				Resource: &orcv1alpha1.ServiceResourceSpec{
					Name: (*orcv1alpha1.OpenStackName)(tt.newValue)},
			},
			}
			osResource := &osResourceT{Extra: map[string]any{"name": tt.existingValue}}

			updateOpts := services.UpdateOpts{Extra: make(map[string]any)}
			handleNameUpdate(&updateOpts, obj, osResource)

			if got, _ := needsUpdate(updateOpts); got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})
	}
}

func TestHandleDescriptionUpdate(t *testing.T) {
	testCases := []struct {
		name          string
		newValue      *string
		existingValue string
		expectChange  bool
	}{
		{name: "Identical", newValue: ptr.To("same-description"), existingValue: "same-description", expectChange: false},
		{name: "Different", newValue: ptr.To("new-description"), existingValue: "same-description", expectChange: true},
		{name: "No value provided, existing is set", newValue: nil, existingValue: "description", expectChange: true},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resource := &orcv1alpha1.ServiceResourceSpec{
				Description: tt.newValue,
			}
			osResource := &osResourceT{Extra: map[string]any{"description": tt.existingValue}}

			updateOpts := services.UpdateOpts{Extra: make(map[string]any)}
			handleDescriptionUpdate(&updateOpts, resource, osResource)

			if got, _ := needsUpdate(updateOpts); got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})
	}
}
