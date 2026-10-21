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

package volumetype

import (
	"testing"

	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumetypes"
	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/api/v1alpha1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/ptr"
)

func TestNeedsUpdate(t *testing.T) {
	testCases := []struct {
		name         string
		updateOpts   volumetypes.UpdateOpts
		expectChange bool
	}{
		{
			name:         "Empty base opts",
			updateOpts:   volumetypes.UpdateOpts{},
			expectChange: false,
		},
		{
			name:         "Updated opts",
			updateOpts:   volumetypes.UpdateOpts{Name: ptr.To("updated")},
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
			resource := &orcv1alpha1.VolumeType{}
			resource.Name = "object-name"
			resource.Spec = orcv1alpha1.VolumeTypeSpec{
				Resource: &orcv1alpha1.VolumeTypeResourceSpec{Name: tt.newValue},
			}
			osResource := &osResourceT{VolumeType: volumetypes.VolumeType{Name: tt.existingValue}}

			updateOpts := volumetypes.UpdateOpts{}
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
			resource := &orcv1alpha1.VolumeTypeResourceSpec{Description: tt.newValue}
			osResource := &osResourceT{VolumeType: volumetypes.VolumeType{Description: tt.existingValue}}

			updateOpts := volumetypes.UpdateOpts{}
			handleDescriptionUpdate(&updateOpts, resource, osResource)

			got, _ := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})

	}
}

func TestHandleIsPublicUpdate(t *testing.T) {
	ptrToBool := ptr.To[bool]
	testCases := []struct {
		name          string
		newValue      *bool
		existingValue bool
		expectChange  bool
	}{
		{name: "Identical", newValue: ptrToBool(true), existingValue: true, expectChange: false},
		{name: "Different", newValue: ptrToBool(true), existingValue: false, expectChange: true},
		{name: "No value provided, existing is set", newValue: nil, existingValue: false, expectChange: true},
		{name: "No value provided, existing is default", newValue: nil, existingValue: true, expectChange: false},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resource := &orcv1alpha1.VolumeTypeResourceSpec{IsPublic: tt.newValue}
			osResource := &osResourceT{VolumeType: volumetypes.VolumeType{IsPublic: tt.existingValue}}
			updateOpts := volumetypes.UpdateOpts{}
			handleIsPublicUpdate(&updateOpts, resource, osResource)
			got, _ := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})
	}
}

func TestHandleProjectRefsUpdate(t *testing.T) {
	projectName1 := "project1"
	projectID1 := "05648b1ac8c44e699de20c378c3f8cc7"
	projectName2 := "project2"
	projectID2 := "7df7160f73d64c6ab6a08de445aae10b"

	testCases := []struct {
		name string

		orcProjects map[string]*orcv1alpha1.Project
		osProjects  []string

		expectedToAdd    sets.Set[string]
		expectedToDelete sets.Set[string]
	}{
		{
			name:             "Both empty",
			orcProjects:      map[string]*orcv1alpha1.Project{},
			osProjects:       []string{},
			expectedToAdd:    sets.New[string](),
			expectedToDelete: sets.New[string](),
		},
		{
			name:             "Both nil",
			orcProjects:      nil,
			osProjects:       nil,
			expectedToAdd:    sets.New[string](),
			expectedToDelete: sets.New[string](),
		},
		{
			name: "Equal",
			orcProjects: map[string]*orcv1alpha1.Project{
				projectName1: {
					Status: orcv1alpha1.ProjectStatus{
						ID: ptr.To(projectID1),
					},
				},
			},
			osProjects: []string{
				projectID1,
			},
			expectedToAdd:    sets.New[string](),
			expectedToDelete: sets.New[string](),
		},
		{
			name: "New project",
			orcProjects: map[string]*orcv1alpha1.Project{
				projectName1: {
					Status: orcv1alpha1.ProjectStatus{
						ID: ptr.To(projectID1),
					},
				},
			},
			osProjects: []string{},
			expectedToAdd: func() sets.Set[string] {
				s := sets.New[string]()
				s.Insert(projectID1)
				return s
			}(),
			expectedToDelete: sets.New[string](),
		},
		{
			name:        "Remove project",
			orcProjects: map[string]*orcv1alpha1.Project{},
			osProjects: []string{
				projectID1,
			},
			expectedToAdd: sets.New[string](),
			expectedToDelete: func() sets.Set[string] {
				s := sets.New[string]()
				s.Insert(projectID1)
				return s
			}(),
		},
		{
			name: "Add more projects",
			orcProjects: map[string]*orcv1alpha1.Project{
				projectName1: {
					Status: orcv1alpha1.ProjectStatus{
						ID: ptr.To(projectID1),
					},
				},
				projectName2: {
					Status: orcv1alpha1.ProjectStatus{
						ID: ptr.To(projectID2),
					},
				},
			},
			osProjects: []string{
				projectID1,
			},
			expectedToAdd: func() sets.Set[string] {
				s := sets.New[string]()
				s.Insert(projectID2)
				return s
			}(),
			expectedToDelete: sets.New[string](),
		},
		{
			name: "Delete a project when there are more than one",
			orcProjects: map[string]*orcv1alpha1.Project{
				projectName1: {
					Status: orcv1alpha1.ProjectStatus{
						ID: ptr.To(projectID1),
					},
				},
			},
			osProjects: []string{
				projectID1,
				projectID2,
			},
			expectedToAdd: sets.New[string](),
			expectedToDelete: func() sets.Set[string] {
				s := sets.New[string]()
				s.Insert(projectID2)
				return s
			}(),
		},
		{
			name: "Add/delete projects at the same time",
			orcProjects: map[string]*orcv1alpha1.Project{
				projectName1: {
					Status: orcv1alpha1.ProjectStatus{
						ID: ptr.To(projectID1),
					},
				},
			},
			osProjects: []string{
				projectID2,
			},
			expectedToAdd: func() sets.Set[string] {
				s := sets.New[string]()
				s.Insert(projectID1)
				return s
			}(),
			expectedToDelete: func() sets.Set[string] {
				s := sets.New[string]()
				s.Insert(projectID2)
				return s
			}(),
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			toAdd, toDelete := handleProjectRefsUpdate(tt.orcProjects, tt.osProjects)

			if !tt.expectedToAdd.Equal(toAdd) {
				t.Errorf("Expected change: %v, got: %v", tt.expectedToAdd, toAdd)
			}

			if !tt.expectedToDelete.Equal(toDelete) {
				t.Errorf("Expected change: %v, got: %v", tt.expectedToDelete, toDelete)
			}
		})
	}
}
