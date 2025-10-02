package floatingip

import (
	"testing"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/floatingips"
	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	"k8s.io/utils/ptr"
)

func TestNeedsUpdate(t *testing.T) {
	testCases := []struct {
		name         string
		updateOpts   floatingips.UpdateOptsBuilder
		expectChange bool
	}{
		{
			name:         "Empty base opts",
			updateOpts:   floatingips.UpdateOpts{},
			expectChange: false,
		},
		{
			name:         "Empty base opts with revision number",
			updateOpts:   floatingips.UpdateOpts{RevisionNumber: ptr.To(4)},
			expectChange: false,
		},
		{
			name:         "Updated opts",
			updateOpts:   floatingips.UpdateOpts{Description: ptr.To("updated")},
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
			resource := &orcv1alpha1.FloatingIPResourceSpec{Description: tt.newValue}
			osResource := &floatingips.FloatingIP{Description: tt.existingValue}

			updateOpts := floatingips.UpdateOpts{}
			handleDescriptionUpdate(&updateOpts, resource, osResource)

			got, _ := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})

	}
}

func TestHandleFixedIPUpdate(t *testing.T) {
	ptrToIPvAny := ptr.To[orcv1alpha1.IPvAny]
	testCases := []struct {
		name          string
		newValue      *orcv1alpha1.IPvAny
		existingValue string
		expectChange  bool
	}{
		{name: "Identical", newValue: ptrToIPvAny("192.168.1.10"), existingValue: "192.168.1.10", expectChange: false},
		{name: "Different", newValue: ptrToIPvAny("192.168.1.20"), existingValue: "192.168.1.10", expectChange: true},
		// FIXME(dkokkino): Unable to clear the field when it has been set issue tracked by https://github.com/gophercloud/gophercloud/issues/3520
		// {name: "No value provided, existing set", newValue: nil, existingValue: "192.168.1.10", expectChange: true},
		{name: "No value provided, existing empty", newValue: nil, existingValue: "", expectChange: false},
		{name: "Value provided, existing empty", newValue: ptrToIPvAny("10.0.0.1"), existingValue: "", expectChange: true},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resource := &orcv1alpha1.FloatingIPResourceSpec{FixedIP: tt.newValue}
			osResource := &floatingips.FloatingIP{FixedIP: tt.existingValue}

			updateOpts := floatingips.UpdateOpts{}
			handleFixedIPUpdate(&updateOpts, resource, osResource)

			got, _ := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})

	}
}
