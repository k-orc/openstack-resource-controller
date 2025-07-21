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

func TestNeedsUpdate(t *testing.T) {
	testCases := []struct {
		name         string
		updateOpts   ports.UpdateOptsBuilder
		expectChange bool
	}{
		{
			name:         "Empty base opts",
			updateOpts:   ports.UpdateOpts{},
			expectChange: false,
		},
		{
			name:         "Empty base opts with revision number",
			updateOpts:   ports.UpdateOpts{RevisionNumber: ptr.To(4)},
			expectChange: false,
		},
		{
			name:         "Updated opts",
			updateOpts:   ports.UpdateOpts{Name: ptr.To("updated")},
			expectChange: true,
		},
		{
			name: "Empty extended opts",
			updateOpts: portsecurity.PortUpdateOptsExt{
				UpdateOptsBuilder: ports.UpdateOpts{},
			},
			expectChange: false,
		},
		{
			name: "Updated extended opts",
			updateOpts: portsecurity.PortUpdateOptsExt{
				UpdateOptsBuilder:   ports.UpdateOpts{},
				PortSecurityEnabled: ptr.To(true),
			},
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
			resource := &orcv1alpha1.Port{}
			resource.Name = "object-name"
			resource.Spec = orcv1alpha1.PortSpec{
				Resource: &orcv1alpha1.PortResourceSpec{Name: tt.newValue},
			}
			port := &ports.Port{Name: tt.existingValue}
			osResource := &osclients.PortExt{Port: *port}

			updateOpts := ports.UpdateOpts{}
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
			resource := &orcv1alpha1.PortResourceSpec{Description: tt.newValue}
			port := &ports.Port{Description: tt.existingValue}
			osResource := &osclients.PortExt{Port: *port}

			updateOpts := ports.UpdateOpts{}
			handleDescriptionUpdate(&updateOpts, resource, osResource)

			got, _ := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
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
			handleAllowedAddressPairsUpdate(&updateOpts, resource, osResource)

			got, _ := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})

	}
}

func makeSecGroupWithID(id string) *orcv1alpha1.SecurityGroup {
	return &orcv1alpha1.SecurityGroup{
		Status: orcv1alpha1.SecurityGroupStatus{
			ID: &id,
		},
	}
}

func TestHandleSecurityGroupRefsUpdate(t *testing.T) {
	sgWebName := orcv1alpha1.OpenStackName("sg-web")
	sgDbName := orcv1alpha1.OpenStackName("sg-db")

	idWeb := "d564a44b-346c-4f71-92b1-5899b8979374"
	idDb := "1d23d83b-2a78-4c12-9e55-0a6e026dd201"
	idOther := "7e8a3b8d-6c17-4581-80a5-a4b8b64f9b0c"

	testCases := []struct {
		name          string
		newValue      []orcv1alpha1.OpenStackName
		existingValue []string
		secGroupMap   map[string]*orcv1alpha1.SecurityGroup
		expectChange  bool
	}{
		{
			name:          "Identical",
			newValue:      []orcv1alpha1.OpenStackName{sgWebName, sgDbName},
			existingValue: []string{idWeb, idDb},
			secGroupMap: map[string]*orcv1alpha1.SecurityGroup{
				string(sgWebName): makeSecGroupWithID(idWeb),
				string(sgDbName):  makeSecGroupWithID(idDb),
			},
			expectChange: false,
		},
		{
			name:          "Identical but different order",
			newValue:      []orcv1alpha1.OpenStackName{sgDbName, sgWebName},
			existingValue: []string{idWeb, idDb},
			secGroupMap: map[string]*orcv1alpha1.SecurityGroup{
				string(sgWebName): makeSecGroupWithID(idWeb),
				string(sgDbName):  makeSecGroupWithID(idDb),
			},
			expectChange: false,
		},
		{
			name:          "Add a security group",
			newValue:      []orcv1alpha1.OpenStackName{sgWebName, sgDbName},
			existingValue: []string{idWeb},
			secGroupMap: map[string]*orcv1alpha1.SecurityGroup{
				string(sgWebName): makeSecGroupWithID(idWeb),
				string(sgDbName):  makeSecGroupWithID(idDb),
			},
			expectChange: true,
		},
		{
			name:          "Remove a security group",
			newValue:      []orcv1alpha1.OpenStackName{sgWebName},
			existingValue: []string{idWeb, idDb},
			secGroupMap: map[string]*orcv1alpha1.SecurityGroup{
				string(sgWebName): makeSecGroupWithID(idWeb),
				string(sgDbName):  makeSecGroupWithID(idDb),
			},
			expectChange: true,
		},
		{
			name:          "Replace a security group",
			newValue:      []orcv1alpha1.OpenStackName{sgWebName, sgDbName},
			existingValue: []string{idWeb, idOther},
			secGroupMap: map[string]*orcv1alpha1.SecurityGroup{
				string(sgWebName): makeSecGroupWithID(idWeb),
				string(sgDbName):  makeSecGroupWithID(idDb),
			},
			expectChange: true,
		},
		{
			name:          "Remove all security groups",
			newValue:      []orcv1alpha1.OpenStackName{},
			existingValue: []string{idWeb, idDb},
			secGroupMap:   map[string]*orcv1alpha1.SecurityGroup{},
			expectChange:  true,
		},
		{
			name:          "Add to empty list",
			newValue:      []orcv1alpha1.OpenStackName{sgWebName},
			existingValue: []string{},
			secGroupMap: map[string]*orcv1alpha1.SecurityGroup{
				string(sgWebName): makeSecGroupWithID(idWeb),
			},
			expectChange: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resource := &orcv1alpha1.PortResourceSpec{
				SecurityGroupRefs: tt.newValue,
			}

			port := &ports.Port{SecurityGroups: tt.existingValue}
			osResource := &osclients.PortExt{Port: *port}

			updateOpts := ports.UpdateOpts{}
			handleSecurityGroupRefsUpdate(&updateOpts, resource, osResource, tt.secGroupMap)

			got, _ := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
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

			updateOpts := handlePortBindingUpdate(&ports.UpdateOpts{}, resource, osResource)

			got, _ := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("expected needsUpdate=%v, got %v", tt.expectChange, got)
			}
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
			resource := &orcv1alpha1.PortResourceSpec{PortSecurity: tt.newValue}
			osResource := &osclients.PortExt{
				PortSecurityExt: portsecurity.PortSecurityExt{
					PortSecurityEnabled: tt.existingValue,
				},
			}

			updateOpts := handlePortSecurityUpdate(&ports.UpdateOpts{}, resource, osResource)

			got, _ := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("expected needsUpdate=%v, got %v", tt.expectChange, got)
			}
		})
	}
}
