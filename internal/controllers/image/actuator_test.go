package image

import (
	"testing"

	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/images"
	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	"k8s.io/utils/ptr"
)

func TestNeedsUpdate(t *testing.T) {
	testCases := []struct {
		name         string
		updateOpts   images.UpdateOpts
		expectChange bool
	}{
		{
			name:         "No changes (nil slice)",
			updateOpts:   nil,
			expectChange: false,
		},
		{
			name:         "No changes (empty slice)",
			updateOpts:   images.UpdateOpts{},
			expectChange: false,
		},
		{
			name: "One change (name update)",
			updateOpts: images.UpdateOpts{
				images.ReplaceImageName{NewName: "updated-name"},
			},
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
			resource := &orcv1alpha1.Image{}
			resource.Name = "object-name"
			resource.Spec = orcv1alpha1.ImageSpec{
				Resource: &orcv1alpha1.ImageResourceSpec{
					Name: tt.newValue,
				},
			}

			osResource := &images.Image{Name: tt.existingValue}
			updateOpts := images.UpdateOpts{}

			updateOpts = handleNameUpdate(updateOpts, resource, osResource)

			got := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})
	}
}

func TestHandleVisibilityUpdate(t *testing.T) {
	ptrToVis := ptr.To[orcv1alpha1.ImageVisibility]

	testCases := []struct {
		name          string
		newValue      *orcv1alpha1.ImageVisibility
		existingValue images.ImageVisibility
		expectChange  bool
	}{
		{
			name:          "Identical",
			newValue:      ptrToVis("private"),
			existingValue: "private",
			expectChange:  false,
		},
		{
			name:          "Different",
			newValue:      ptrToVis("public"),
			existingValue: "private",
			expectChange:  true,
		},
		{
			name:          "Not specified in spec",
			newValue:      nil,
			existingValue: "private",
			expectChange:  false,
		},
		{
			name:          "Changing to community",
			newValue:      ptrToVis("community"),
			existingValue: "public",
			expectChange:  true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {

			resource := &orcv1alpha1.ImageResourceSpec{
				Visibility: tt.newValue,
			}
			osResource := &images.Image{Visibility: tt.existingValue}
			updateOpts := images.UpdateOpts{}

			updateOpts = handleVisibilityUpdate(updateOpts, resource, osResource)

			got := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})
	}
}

func TestHandleProtectedUpdate(t *testing.T) {
	ptrToBool := ptr.To[bool]

	testCases := []struct {
		name          string
		newValue      *bool
		existingValue bool
		expectChange  bool
	}{
		{
			name:          "Identical (true)",
			newValue:      ptrToBool(true),
			existingValue: true,
			expectChange:  false,
		},
		{
			name:          "Identical (false)",
			newValue:      ptrToBool(false),
			existingValue: false,
			expectChange:  false,
		},
		{
			name:          "Different (spec is true, existing is false)",
			newValue:      ptrToBool(true),
			existingValue: false,
			expectChange:  true,
		},
		{
			name:          "Different (spec is false, existing is true)",
			newValue:      ptrToBool(false),
			existingValue: true,
			expectChange:  true,
		},
		{
			name:          "Not specified in spec",
			newValue:      nil,
			existingValue: false,
			expectChange:  false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resource := &orcv1alpha1.ImageResourceSpec{
				Protected: tt.newValue,
			}

			osResource := &images.Image{Protected: tt.existingValue}

			updateOpts := images.UpdateOpts{}
			updateOpts = handleProtectedUpdate(updateOpts, resource, osResource)

			got := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})
	}
}

func TestHandleTagsUpdate(t *testing.T) {
	testCases := []struct {
		name          string
		newValue      []orcv1alpha1.ImageTag
		existingValue []string
		expectChange  bool
	}{
		{
			name:          "Identical tags in same order",
			newValue:      []orcv1alpha1.ImageTag{"tag1", "tag2"},
			existingValue: []string{"tag1", "tag2"},
			expectChange:  false,
		},
		{
			name:          "Identical tags in different order",
			newValue:      []orcv1alpha1.ImageTag{"tag2", "tag1"},
			existingValue: []string{"tag1", "tag2"},
			expectChange:  false,
		},
		{
			name:          "Adding a tag",
			newValue:      []orcv1alpha1.ImageTag{"tag1", "tag2", "tag3"},
			existingValue: []string{"tag1", "tag2"},
			expectChange:  true,
		},
		{
			name:          "Removing a tag",
			newValue:      []orcv1alpha1.ImageTag{"tag1"},
			existingValue: []string{"tag1", "tag2"},
			expectChange:  true,
		},
		{
			name:          "Completely different tags",
			newValue:      []orcv1alpha1.ImageTag{"alpha", "beta"},
			existingValue: []string{"tag1", "tag2"},
			expectChange:  true,
		},
		{
			name:          "Spec is empty, existing has tags (clearing tags)",
			newValue:      []orcv1alpha1.ImageTag{},
			existingValue: []string{"tag1", "tag2"},
			expectChange:  true,
		},
		{
			name:          "Spec has tags, existing is empty",
			newValue:      []orcv1alpha1.ImageTag{"tag1"},
			existingValue: []string{},
			expectChange:  true,
		},
		{
			name:          "Spec has tags, existing is nil",
			newValue:      []orcv1alpha1.ImageTag{"tag1"},
			existingValue: nil,
			expectChange:  true,
		},
		{
			name:          "Spec is nil, existing has tags (clearing tags)",
			newValue:      nil,
			existingValue: []string{"tag1"},
			expectChange:  true,
		},
		{
			name:          "Spec is nil, existing is empty",
			newValue:      nil,
			existingValue: []string{},
			expectChange:  false,
		},
		{
			name:          "Spec is empty, existing is nil",
			newValue:      []orcv1alpha1.ImageTag{},
			existingValue: nil,
			expectChange:  false,
		},
		{
			name:          "Both spec and existing are nil",
			newValue:      nil,
			existingValue: nil,
			expectChange:  false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resource := &orcv1alpha1.ImageResourceSpec{
				Tags: tt.newValue,
			}
			osResource := &images.Image{Tags: tt.existingValue}
			updateOpts := images.UpdateOpts{}

			updateOpts = handleTagsUpdate(updateOpts, resource, osResource)

			got := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})
	}
}
