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

package trunk

import (
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"
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
			name:         "Updated opts",
			updateOpts:   trunks.UpdateOpts{Name: ptr.To("updated")},
			expectChange: true,
		},
		{
			name:         "RevisionNumber only should not require update",
			updateOpts:   trunks.UpdateOpts{RevisionNumber: ptr.To(10)},
			expectChange: false,
		},
		{
			name:         "Name + RevisionNumber should require update",
			updateOpts:   trunks.UpdateOpts{Name: ptr.To("updated"), RevisionNumber: ptr.To(10)},
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
			resource := &orcv1alpha1.Trunk{}
			resource.Name = "object-name"
			resource.Spec = orcv1alpha1.TrunkSpec{
				Resource: &orcv1alpha1.TrunkResourceSpec{Name: tt.newValue},
			}
			osResource := &osResourceT{Name: tt.existingValue}

			updateOpts := trunks.UpdateOpts{}
			handleNameUpdate(&updateOpts, resource, osResource)

			got, _ := needsUpdate(updateOpts)
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
			osResource := &osResourceT{Description: tt.existingValue}

			updateOpts := trunks.UpdateOpts{}
			handleDescriptionUpdate(&updateOpts, resource, osResource)

			got, _ := needsUpdate(updateOpts)
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
		{name: "Identical true", newValue: ptrToBool(true), existingValue: true, expectChange: false},
		{name: "Identical false", newValue: ptrToBool(false), existingValue: false, expectChange: false},
		{name: "Different (true -> false)", newValue: ptrToBool(false), existingValue: true, expectChange: true},
		{name: "Different (false -> true)", newValue: ptrToBool(true), existingValue: false, expectChange: true},
		{name: "Nil means default true (existing true)", newValue: nil, existingValue: true, expectChange: false},
		{name: "Nil means default true (existing false)", newValue: nil, existingValue: false, expectChange: true},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resource := &orcv1alpha1.TrunkResourceSpec{AdminStateUp: tt.newValue}
			osResource := &osResourceT{AdminStateUp: tt.existingValue}

			updateOpts := trunks.UpdateOpts{}
			handleAdminStateUpUpdate(&updateOpts, resource, osResource)

			got, _ := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})
	}
}

func TestReconcileSubportsLogic(t *testing.T) {
	testCases := []struct {
		name                   string
		desiredSubports        map[string]*orcv1alpha1.TrunkSubportSpec
		actualSubports         map[string]trunks.Subport
		expectedSubportsToAdd  []trunks.Subport
		expectedSubportsRemove []trunks.Subport
	}{
		{
			name: "No changes needed",
			desiredSubports: map[string]*orcv1alpha1.TrunkSubportSpec{
				"port1": {SegmentationID: 100, SegmentationType: "vlan"},
			},
			actualSubports: map[string]trunks.Subport{
				"port1": {PortID: "port1", SegmentationID: 100, SegmentationType: "vlan"},
			},
			expectedSubportsToAdd:  []trunks.Subport{},
			expectedSubportsRemove: []trunks.Subport{},
		},
		{
			name: "Add new subport",
			desiredSubports: map[string]*orcv1alpha1.TrunkSubportSpec{
				"port1": {SegmentationID: 100, SegmentationType: "vlan"},
				"port2": {SegmentationID: 200, SegmentationType: "vlan"},
			},
			actualSubports: map[string]trunks.Subport{
				"port1": {PortID: "port1", SegmentationID: 100, SegmentationType: "vlan"},
			},
			expectedSubportsToAdd: []trunks.Subport{
				{PortID: "port2", SegmentationID: 200, SegmentationType: "vlan"},
			},
			expectedSubportsRemove: []trunks.Subport{},
		},
		{
			name: "Remove subport",
			desiredSubports: map[string]*orcv1alpha1.TrunkSubportSpec{
				"port1": {SegmentationID: 100, SegmentationType: "vlan"},
			},
			actualSubports: map[string]trunks.Subport{
				"port1": {PortID: "port1", SegmentationID: 100, SegmentationType: "vlan"},
				"port2": {PortID: "port2", SegmentationID: 200, SegmentationType: "vlan"},
			},
			expectedSubportsToAdd: []trunks.Subport{},
			expectedSubportsRemove: []trunks.Subport{
				{PortID: "port2"},
			},
		},
		{
			name: "Update segmentation",
			desiredSubports: map[string]*orcv1alpha1.TrunkSubportSpec{
				"port1": {SegmentationID: 150, SegmentationType: "vlan"},
			},
			actualSubports: map[string]trunks.Subport{
				"port1": {PortID: "port1", SegmentationID: 100, SegmentationType: "vlan"},
			},
			expectedSubportsToAdd: []trunks.Subport{
				{PortID: "port1", SegmentationID: 150, SegmentationType: "vlan"},
			},
			expectedSubportsRemove: []trunks.Subport{
				{PortID: "port1"},
			},
		},
		{
			name: "Update segmentation type",
			desiredSubports: map[string]*orcv1alpha1.TrunkSubportSpec{
				"port1": {SegmentationID: 100, SegmentationType: "inherit"},
			},
			actualSubports: map[string]trunks.Subport{
				"port1": {PortID: "port1", SegmentationID: 100, SegmentationType: "vlan"},
			},
			expectedSubportsToAdd: []trunks.Subport{
				{PortID: "port1", SegmentationID: 100, SegmentationType: "inherit"},
			},
			expectedSubportsRemove: []trunks.Subport{
				{PortID: "port1"},
			},
		},
		{
			name:            "Remove all subports",
			desiredSubports: map[string]*orcv1alpha1.TrunkSubportSpec{},
			actualSubports: map[string]trunks.Subport{
				"port1": {PortID: "port1", SegmentationID: 100, SegmentationType: "vlan"},
				"port2": {PortID: "port2", SegmentationID: 200, SegmentationType: "vlan"},
			},
			expectedSubportsToAdd: []trunks.Subport{},
			expectedSubportsRemove: []trunks.Subport{
				{PortID: "port1"},
				{PortID: "port2"},
			},
		},
		{
			name: "Complex update: add, remove, and modify",
			desiredSubports: map[string]*orcv1alpha1.TrunkSubportSpec{
				"port1": {SegmentationID: 150, SegmentationType: "vlan"}, // modified
				"port3": {SegmentationID: 300, SegmentationType: "vlan"}, // new
			},
			actualSubports: map[string]trunks.Subport{
				"port1": {PortID: "port1", SegmentationID: 100, SegmentationType: "vlan"},
				"port2": {PortID: "port2", SegmentationID: 200, SegmentationType: "vlan"}, // removed
			},
			expectedSubportsToAdd: []trunks.Subport{
				{PortID: "port1", SegmentationID: 150, SegmentationType: "vlan"}, // modified
				{PortID: "port3", SegmentationID: 300, SegmentationType: "vlan"}, // new
			},
			expectedSubportsRemove: []trunks.Subport{
				{PortID: "port1"}, // for modification
				{PortID: "port2"}, // removed
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			subportsToAdd := []trunks.Subport{}
			subportsToRemove := []trunks.Subport{}

			// Find subports to add (in desired but not in actual, or different segmentation)
			for portID, desiredSpec := range tt.desiredSubports {
				actual, exists := tt.actualSubports[portID]
				if !exists {
					// Need to add this subport
					subportsToAdd = append(subportsToAdd, trunks.Subport{
						PortID:           portID,
						SegmentationID:   int(desiredSpec.SegmentationID),
						SegmentationType: desiredSpec.SegmentationType,
					})
				} else if actual.SegmentationID != int(desiredSpec.SegmentationID) || actual.SegmentationType != desiredSpec.SegmentationType {
					// Segmentation changed - need to remove and re-add
					subportsToRemove = append(subportsToRemove, trunks.Subport{PortID: portID})
					subportsToAdd = append(subportsToAdd, trunks.Subport{
						PortID:           portID,
						SegmentationID:   int(desiredSpec.SegmentationID),
						SegmentationType: desiredSpec.SegmentationType,
					})
				}
			}

			// Find subports to remove (in actual but not in desired)
			for portID := range tt.actualSubports {
				if _, exists := tt.desiredSubports[portID]; !exists {
					subportsToRemove = append(subportsToRemove, trunks.Subport{PortID: portID})
				}
			}

			// Sort slices by PortID for deterministic comparison
			sortByPortID := func(a, b trunks.Subport) int {
				if a.PortID < b.PortID {
					return -1
				}
				if a.PortID > b.PortID {
					return 1
				}
				return 0
			}
			slices.SortFunc(subportsToAdd, sortByPortID)
			slices.SortFunc(subportsToRemove, sortByPortID)
			slices.SortFunc(tt.expectedSubportsToAdd, sortByPortID)
			slices.SortFunc(tt.expectedSubportsRemove, sortByPortID)

			if diff := cmp.Diff(tt.expectedSubportsToAdd, subportsToAdd); diff != "" {
				t.Errorf("Subports to add mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.expectedSubportsRemove, subportsToRemove); diff != "" {
				t.Errorf("Subports to remove mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
