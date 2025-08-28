package project

import (
	"testing"

	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/projects"
	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	"k8s.io/utils/ptr"
)

func TestNeedsUpdate(t *testing.T) {
	testCases := []struct {
		name         string
		updateOpts   projects.UpdateOptsBuilder
		expectChange bool
	}{
		{
			name:         "Empty base opts",
			updateOpts:   projects.UpdateOpts{},
			expectChange: false,
		},
		{
			name:         "Updated opts",
			updateOpts:   projects.UpdateOpts{Name: "updated"},
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
			resource := &orcv1alpha1.Project{}
			resource.Name = "object-name"
			resource.Spec = orcv1alpha1.ProjectSpec{
				Resource: &orcv1alpha1.ProjectResourceSpec{Name: tt.newValue},
			}
			osResource := &projects.Project{Name: tt.existingValue}

			updateOpts := projects.UpdateOpts{}
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
			resource := &orcv1alpha1.ProjectResourceSpec{Description: tt.newValue}
			osResource := &projects.Project{Description: tt.existingValue}

			updateOpts := projects.UpdateOpts{}
			handleDescriptionUpdate(&updateOpts, resource, osResource)

			got, _ := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})

	}
}

func TestHandleEnabledUpdate(t *testing.T) {
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
			resource := &orcv1alpha1.ProjectResourceSpec{Enabled: tt.newValue}
			osResource := &projects.Project{Enabled: tt.existingValue}

			updateOpts := projects.UpdateOpts{}
			handleEnabledUpdate(&updateOpts, resource, osResource)

			got, _ := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})
	}
}

func TestHandleTagsUpdate(t *testing.T) {
	testCases := []struct {
		name         string
		newTags      []orcv1alpha1.KeystoneTag
		existingTags []string
		expectChange bool
	}{
		{
			name:         "Identical tags",
			newTags:      []orcv1alpha1.KeystoneTag{"tag1", "tag2"},
			existingTags: []string{"tag1", "tag2"},
			expectChange: false,
		},
		{
			name:         "Different tags",
			newTags:      []orcv1alpha1.KeystoneTag{"tag1", "tag2"},
			existingTags: []string{"tag1", "tag3"},
			expectChange: true,
		},
		{
			name:         "Tags out of order",
			newTags:      []orcv1alpha1.KeystoneTag{"tag2", "tag1"},
			existingTags: []string{"tag1", "tag2"},
			expectChange: false,
		},
		{
			name:         "Extra tag in existing",
			newTags:      []orcv1alpha1.KeystoneTag{"tag1"},
			existingTags: []string{"tag1", "tag2"},
			expectChange: true,
		},
		{
			name:         "Extra tag in new",
			newTags:      []orcv1alpha1.KeystoneTag{"tag1", "tag2"},
			existingTags: []string{"tag1"},
			expectChange: true,
		},
		{
			name:         "Empty tags",
			newTags:      []orcv1alpha1.KeystoneTag{},
			existingTags: []string{},
			expectChange: false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resource := &orcv1alpha1.ProjectResourceSpec{Tags: tt.newTags}
			osResource := &projects.Project{Tags: tt.existingTags}

			updateOpts := projects.UpdateOpts{}
			handleTagsUpdate(&updateOpts, resource, osResource)

			got, _ := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})
	}
}
