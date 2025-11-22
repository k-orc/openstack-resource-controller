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

package trunk

import (
	"testing"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/trunks"
	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	"k8s.io/utils/ptr"
)

func TestNeedsUpdate(t *testing.T) {
	testCases := []struct {
		name         string
		updateOpts   trunks.UpdateOpts
		expectChange bool
	}{
		{
			name:         "Empty base opts",
			updateOpts:   trunks.UpdateOpts{},
			expectChange: false,
		},
		{
			name:         "Empty base opts with revision number",
			updateOpts:   trunks.UpdateOpts{RevisionNumber: ptr.To(4)},
			expectChange: false,
		},
		{
			name:         "Updated opts with name",
			updateOpts:   trunks.UpdateOpts{Name: ptr.To("updated")},
			expectChange: true,
		},
		{
			name:         "Updated opts with description",
			updateOpts:   trunks.UpdateOpts{Description: ptr.To("new description")},
			expectChange: true,
		},
		{
			name:         "Updated opts with adminStateUp",
			updateOpts:   trunks.UpdateOpts{AdminStateUp: ptr.To(false)},
			expectChange: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			got := needsUpdate(tt.updateOpts)
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
			resource := &orcv1alpha1.Trunk{}
			resource.Name = "object-name"
			resource.Spec = orcv1alpha1.TrunkSpec{
				Resource: &orcv1alpha1.TrunkResourceSpec{Name: tt.newValue},
			}
			osResource := &trunks.Trunk{Name: tt.existingValue}

			updateOpts := trunks.UpdateOpts{}
			handleNameUpdate(&updateOpts, resource, osResource)

			got := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})
	}
}

func TestHandleDescriptionUpdate(t *testing.T) {
	ptrToDescription := ptr.To[orcv1alpha1.NeutronDescription]
	testCases := []struct {
		name          string
		newValue      *orcv1alpha1.NeutronDescription
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
			resource := &orcv1alpha1.TrunkResourceSpec{Description: tt.newValue}
			osResource := &trunks.Trunk{Description: tt.existingValue}

			updateOpts := trunks.UpdateOpts{}
			handleDescriptionUpdate(&updateOpts, resource, osResource)

			got := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})
	}
}

func TestHandleAdminStateUpUpdate(t *testing.T) {
	ptrToBool := ptr.To[bool]
	testCases := []struct {
		name          string
		newValue      *bool
		existingValue bool
		expectChange  bool
	}{
		{name: "Identical", newValue: ptrToBool(true), existingValue: true, expectChange: false},
		{name: "Different", newValue: ptrToBool(true), existingValue: false, expectChange: true},
		{name: "No value provided, existing is set", newValue: nil, existingValue: false, expectChange: false},
		{name: "No value provided, existing is default", newValue: nil, existingValue: true, expectChange: false},
		{name: "False when already false", newValue: ptrToBool(false), existingValue: false, expectChange: false},
		{name: "False when was true", newValue: ptrToBool(false), existingValue: true, expectChange: true},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resource := &orcv1alpha1.TrunkResourceSpec{AdminStateUp: tt.newValue}
			osResource := &trunks.Trunk{AdminStateUp: tt.existingValue}

			updateOpts := trunks.UpdateOpts{}
			handleAdminStateUpUpdate(&updateOpts, resource, osResource)

			got := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})
	}
}

// needsUpdate checks if the updateOpts contains any changes that require an update
func needsUpdate(updateOpts trunks.UpdateOpts) bool {
	return updateOpts.Name != nil || updateOpts.Description != nil || updateOpts.AdminStateUp != nil
}

// handleNameUpdate updates the updateOpts if the name needs to be changed
func handleNameUpdate(updateOpts *trunks.UpdateOpts, resource *orcv1alpha1.Trunk, osResource *trunks.Trunk) {
	name := getResourceName(resource)
	if osResource.Name != name {
		updateOpts.Name = &name
	}
}

// handleDescriptionUpdate updates the updateOpts if the description needs to be changed
func handleDescriptionUpdate(updateOpts *trunks.UpdateOpts, resource *orcv1alpha1.TrunkResourceSpec, osResource *trunks.Trunk) {
	description := string(ptr.Deref(resource.Description, ""))
	if osResource.Description != description {
		updateOpts.Description = &description
	}
}

// handleAdminStateUpUpdate updates the updateOpts if the adminStateUp needs to be changed
func handleAdminStateUpUpdate(updateOpts *trunks.UpdateOpts, resource *orcv1alpha1.TrunkResourceSpec, osResource *trunks.Trunk) {
	if resource.AdminStateUp != nil && *resource.AdminStateUp != osResource.AdminStateUp {
		updateOpts.AdminStateUp = resource.AdminStateUp
	}
}

// getResourceName returns the name of the OpenStack resource we should use.
// This is a test helper that mirrors the function in the generated adapter file.
func getResourceName(orcObject *orcv1alpha1.Trunk) string {
	if orcObject.Spec.Resource != nil && orcObject.Spec.Resource.Name != nil {
		return string(*orcObject.Spec.Resource.Name)
	}
	return orcObject.Name
}

