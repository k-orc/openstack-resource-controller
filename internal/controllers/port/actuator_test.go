package port

import (
	"testing"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/portsbinding"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/portsecurity"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	osclients "github.com/k-orc/openstack-resource-controller/v2/internal/osclients"
	"k8s.io/utils/ptr"
)

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
			resource := &orcv1alpha1.Port{}
			resource.Name = "object-name"
			resource.Spec = orcv1alpha1.PortSpec{
				Resource: &orcv1alpha1.PortResourceSpec{Name: tt.newValue},
			}
			port := &ports.Port{Name: tt.existingValue}
			osResource := &osclients.PortExt{Port: *port}

			updateOpts := ports.UpdateOpts{}
			var needsUpdate bool

			handleNameUpdate(&updateOpts, resource, osResource, &needsUpdate)

			if needsUpdate != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, needsUpdate)
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
			resource := &orcv1alpha1.PortResourceSpec{Description: tt.newValue}
			port := &ports.Port{Description: tt.existingValue}
			osResource := &osclients.PortExt{Port: *port}

			updateOpts := ports.UpdateOpts{}
			var needsUpdate bool

			handleDescriptionUpdate(&updateOpts, resource, osResource, &needsUpdate)

			if needsUpdate != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, needsUpdate)
			}
		})

	}
}

func TestHandleAllowedAddressPairsUpdate(t *testing.T) {
	testCases := []struct {
		name          string
		newValue      []orcv1alpha1.AllowedAddressPair
		existingValue []ports.AddressPair
		expectChange  bool
	}{
		{
			name: "Identical",
			newValue: []orcv1alpha1.AllowedAddressPair{
				{IP: orcv1alpha1.IPvAny("192.168.100.1"), MAC: ptr.To(orcv1alpha1.MAC("00:1A:2B:3C:4D:5E"))},
				{IP: orcv1alpha1.IPvAny("192.168.200.1"), MAC: ptr.To(orcv1alpha1.MAC("00:1A:2B:3C:4D:6E"))},
			},
			existingValue: []ports.AddressPair{
				{IPAddress: "192.168.100.1", MACAddress: "00:1A:2B:3C:4D:5E"},
				{IPAddress: "192.168.200.1", MACAddress: "00:1A:2B:3C:4D:6E"},
			},
			expectChange: false,
		},
		{
			name: "Different entry",
			newValue: []orcv1alpha1.AllowedAddressPair{
				{IP: orcv1alpha1.IPvAny("192.168.100.1"), MAC: ptr.To(orcv1alpha1.MAC("00:1A:2B:3C:4D:5E"))},
				{IP: orcv1alpha1.IPvAny("192.168.500.1"), MAC: ptr.To(orcv1alpha1.MAC("00:1A:2B:3C:4D:9E"))},
			},
			existingValue: []ports.AddressPair{
				{IPAddress: "192.168.100.1", MACAddress: "00:1A:2B:3C:4D:5E"},
				{IPAddress: "192.168.200.1", MACAddress: "00:1A:2B:3C:4D:6E"},
			},
			expectChange: true,
		},
		{
			name: "Identical, out of order",
			newValue: []orcv1alpha1.AllowedAddressPair{
				{IP: orcv1alpha1.IPvAny("192.168.100.1"), MAC: ptr.To(orcv1alpha1.MAC("00:1A:2B:3C:4D:5E"))},
				{IP: orcv1alpha1.IPvAny("192.168.200.1"), MAC: ptr.To(orcv1alpha1.MAC("00:1A:2B:3C:4D:6E"))},
			},
			existingValue: []ports.AddressPair{
				{IPAddress: "192.168.200.1", MACAddress: "00:1A:2B:3C:4D:6E"},
				{IPAddress: "192.168.100.1", MACAddress: "00:1A:2B:3C:4D:5E"},
			},
			expectChange: false,
		},
		{
			name: "Identical, with duplicates",
			newValue: []orcv1alpha1.AllowedAddressPair{
				{IP: orcv1alpha1.IPvAny("192.168.100.1"), MAC: ptr.To(orcv1alpha1.MAC("00:1A:2B:3C:4D:5E"))},
				{IP: orcv1alpha1.IPvAny("192.168.200.1"), MAC: ptr.To(orcv1alpha1.MAC("00:1A:2B:3C:4D:6E"))},
				{IP: orcv1alpha1.IPvAny("192.168.200.1"), MAC: ptr.To(orcv1alpha1.MAC("00:1A:2B:3C:4D:6E"))},
			},
			existingValue: []ports.AddressPair{
				{IPAddress: "192.168.100.1", MACAddress: "00:1A:2B:3C:4D:5E"},
				{IPAddress: "192.168.200.1", MACAddress: "00:1A:2B:3C:4D:6E"},
			},
			expectChange: false,
		},
		{
			name: "Removing an entry",
			newValue: []orcv1alpha1.AllowedAddressPair{
				{IP: orcv1alpha1.IPvAny("192.168.100.1"), MAC: ptr.To(orcv1alpha1.MAC("00:1A:2B:3C:4D:5E"))},
			},
			existingValue: []ports.AddressPair{
				{IPAddress: "192.168.100.1", MACAddress: "00:1A:2B:3C:4D:5E"},
				{IPAddress: "192.168.200.1", MACAddress: "00:1A:2B:3C:4D:6E"},
			},
			expectChange: true,
		},
		{
			name: "Adding an entry",
			newValue: []orcv1alpha1.AllowedAddressPair{
				{IP: orcv1alpha1.IPvAny("192.168.100.1"), MAC: ptr.To(orcv1alpha1.MAC("00:1A:2B:3C:4D:5E"))},
				{IP: orcv1alpha1.IPvAny("192.168.200.1"), MAC: ptr.To(orcv1alpha1.MAC("00:1A:2B:3C:4D:6E"))},
			},
			existingValue: []ports.AddressPair{
				{IPAddress: "192.168.100.1", MACAddress: "00:1A:2B:3C:4D:5E"},
			},
			expectChange: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resource := &orcv1alpha1.PortResourceSpec{AllowedAddressPairs: tt.newValue}
			port := &ports.Port{AllowedAddressPairs: tt.existingValue}
			osResource := &osclients.PortExt{Port: *port}

			updateOpts := ports.UpdateOpts{}
			var needsUpdate bool

			handleAllowedAddressPairsUpdate(&updateOpts, resource, osResource, &needsUpdate)

			if needsUpdate != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, needsUpdate)
			}
		})

	}
}

func TestHandlePortBindingUpdate(t *testing.T) {
	testCases := []struct {
		name          string
		newValue      string
		existingValue string
		expectChange  bool
	}{
		{name: "Identical", newValue: "normal", existingValue: "normal", expectChange: false},
		{name: "Different", newValue: "direct", existingValue: "normal", expectChange: true},
		{name: "Updating to empty string", newValue: "", existingValue: "normal", expectChange: false},
		{name: "Updating from empty string", newValue: "normal", existingValue: "", expectChange: true},
		{name: "Both are empty strings", newValue: "", existingValue: "", expectChange: false},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resource := &orcv1alpha1.PortResourceSpec{VNICType: tt.newValue}
			osResource := &osclients.PortExt{
				PortsBindingExt: portsbinding.PortsBindingExt{
					VNICType: tt.existingValue,
				},
			}

			updateOpts := &ports.UpdateOpts{}
			needsUpdate := false

			finalOpts := handlePortBindingUpdate(updateOpts, resource, osResource, &needsUpdate)

			if needsUpdate != tt.expectChange {
				t.Errorf("expected needsUpdate=%v, got %v", tt.expectChange, needsUpdate)
			}

			_ = finalOpts
		})
	}
}

func TestHandlePortSecurityUpdate(t *testing.T) {
	testCases := []struct {
		name          string
		newValue      orcv1alpha1.PortSecurityState
		existingValue bool
		expectChange  bool
	}{
		{name: "Enabled when already enabled", newValue: orcv1alpha1.PortSecurityEnabled, existingValue: true, expectChange: false},
		{name: "Enabled when was disabled", newValue: orcv1alpha1.PortSecurityEnabled, existingValue: false, expectChange: true},

		{name: "Disabled when already disabled", newValue: orcv1alpha1.PortSecurityDisabled, existingValue: false, expectChange: false},
		{name: "Disabled when was enabled", newValue: orcv1alpha1.PortSecurityDisabled, existingValue: true, expectChange: true},

		{name: "Inherit when was enabled", newValue: orcv1alpha1.PortSecurityInherit, existingValue: true, expectChange: false},
		{name: "Inherit when was disabled", newValue: orcv1alpha1.PortSecurityInherit, existingValue: false, expectChange: false},

		{name: "Default (empty string) when was enabled", newValue: "", existingValue: true, expectChange: false},
		{name: "Invalid string when was enabled", newValue: "foo", existingValue: true, expectChange: false},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			// Setup the Kubernetes resource spec and the OpenStack resource state
			resource := &orcv1alpha1.PortResourceSpec{PortSecurity: tt.newValue}
			osResource := &osclients.PortExt{
				PortSecurityExt: portsecurity.PortSecurityExt{
					PortSecurityEnabled: tt.existingValue,
				},
			}

			// Initialize test variables
			updateOpts := &ports.UpdateOpts{}
			needsUpdate := false

			// Call the handler
			finalOpts := handlePortSecurityUpdate(updateOpts, resource, osResource, &needsUpdate)

			// Assert the outcome
			if needsUpdate != tt.expectChange {
				t.Errorf("expected needsUpdate=%v, got %v", tt.expectChange, needsUpdate)
			}

			// Ensure the builder is used to prevent compiler errors
			_ = finalOpts
		})
	}
}
