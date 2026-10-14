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

package group

import (
	"testing"

	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/groups"
	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/api/v1alpha1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/ptr"
)

func TestNeedsUpdate(t *testing.T) {
	testCases := []struct {
		name         string
		updateOpts   groups.UpdateOpts
		expectChange bool
	}{
		{
			name:         "Empty base opts",
			updateOpts:   groups.UpdateOpts{},
			expectChange: false,
		},
		{
			name:         "Updated opts",
			updateOpts:   groups.UpdateOpts{Name: "updated"},
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
	ptrToName := ptr.To[orcv1alpha1.KeystoneName]
	testCases := []struct {
		name          string
		newValue      *orcv1alpha1.KeystoneName
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
			resource := &orcv1alpha1.Group{}
			resource.Name = "object-name"
			resource.Spec = orcv1alpha1.GroupSpec{
				Resource: &orcv1alpha1.GroupResourceSpec{Name: tt.newValue},
			}
			osResource := &osResourceT{
				Group: groups.Group{
					Name: tt.existingValue,
				},
			}

			updateOpts := groups.UpdateOpts{}
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
			resource := &orcv1alpha1.GroupResourceSpec{Description: tt.newValue}
			osResource := &osResourceT{
				Group: groups.Group{
					Description: tt.existingValue},
			}

			updateOpts := groups.UpdateOpts{}
			handleDescriptionUpdate(&updateOpts, resource, osResource)

			got, _ := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})

	}
}

func TestHandleUserRefsUpdate(t *testing.T) {
	userName1 := "user1"
	userID1 := "05648b1ac8c44e699de20c378c3f8cc7"
	userName2 := "user2"
	userID2 := "7df7160f73d64c6ab6a08de445aae10b"

	testCases := []struct {
		name string

		orcUsers map[string]*orcv1alpha1.User
		osUsers  []string

		expectedToAdd    sets.Set[string]
		expectedToDelete sets.Set[string]
	}{
		{
			name:             "Both empty",
			orcUsers:         map[string]*orcv1alpha1.User{},
			osUsers:          []string{},
			expectedToAdd:    sets.New[string](),
			expectedToDelete: sets.New[string](),
		},
		{
			name:             "Both nil",
			orcUsers:         nil,
			osUsers:          nil,
			expectedToAdd:    sets.New[string](),
			expectedToDelete: sets.New[string](),
		},
		{
			name: "Equal",
			orcUsers: map[string]*orcv1alpha1.User{
				userName1: {
					Status: orcv1alpha1.UserStatus{
						ID: ptr.To(userID1),
					},
				},
			},
			osUsers: []string{
				userID1,
			},
			expectedToAdd:    sets.New[string](),
			expectedToDelete: sets.New[string](),
		},
		{
			name: "New user",
			orcUsers: map[string]*orcv1alpha1.User{
				userName1: {
					Status: orcv1alpha1.UserStatus{
						ID: ptr.To(userID1),
					},
				},
			},
			osUsers: []string{},
			expectedToAdd: func() sets.Set[string] {
				s := sets.New[string]()
				s.Insert(userID1)
				return s
			}(),
			expectedToDelete: sets.New[string](),
		},
		{
			name:     "Remove user",
			orcUsers: map[string]*orcv1alpha1.User{},
			osUsers: []string{
				userID1,
			},
			expectedToAdd: sets.New[string](),
			expectedToDelete: func() sets.Set[string] {
				s := sets.New[string]()
				s.Insert(userID1)
				return s
			}(),
		},
		{
			name: "Add more user",
			orcUsers: map[string]*orcv1alpha1.User{
				userName1: {
					Status: orcv1alpha1.UserStatus{
						ID: ptr.To(userID1),
					},
				},
				userName2: {
					Status: orcv1alpha1.UserStatus{
						ID: ptr.To(userID2),
					},
				},
			},
			osUsers: []string{
				userID1,
			},
			expectedToAdd: func() sets.Set[string] {
				s := sets.New[string]()
				s.Insert(userID2)
				return s
			}(),
			expectedToDelete: sets.New[string](),
		},
		{
			name: "Delete a user when there are more than one",
			orcUsers: map[string]*orcv1alpha1.User{
				userName1: {
					Status: orcv1alpha1.UserStatus{
						ID: ptr.To(userID1),
					},
				},
			},
			osUsers: []string{
				userID1,
				userID2,
			},
			expectedToAdd: sets.New[string](),
			expectedToDelete: func() sets.Set[string] {
				s := sets.New[string]()
				s.Insert(userID2)
				return s
			}(),
		},
		{
			name: "Add/delete users at the same time",
			orcUsers: map[string]*orcv1alpha1.User{
				userName1: {
					Status: orcv1alpha1.UserStatus{
						ID: ptr.To(userID1),
					},
				},
			},
			osUsers: []string{
				userID2,
			},
			expectedToAdd: func() sets.Set[string] {
				s := sets.New[string]()
				s.Insert(userID1)
				return s
			}(),
			expectedToDelete: func() sets.Set[string] {
				s := sets.New[string]()
				s.Insert(userID2)
				return s
			}(),
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			toAdd, toDelete := handleUserRefsUpdate(tt.orcUsers, tt.osUsers)

			if !tt.expectedToAdd.Equal(toAdd) {
				t.Errorf("Expected change: %v, got: %v", tt.expectedToAdd, toAdd)
			}

			if !tt.expectedToDelete.Equal(toDelete) {
				t.Errorf("Expected change: %v, got: %v", tt.expectedToDelete, toDelete)
			}
		})
	}
}
