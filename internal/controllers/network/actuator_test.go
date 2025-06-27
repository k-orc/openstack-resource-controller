package network

import (
	"testing"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/dns"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/external"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/mtu"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/portsecurity"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks"
	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v2/internal/osclients"
	"k8s.io/utils/ptr"
)

func TestNeedsUpdate(t *testing.T) {
	testCases := []struct {
		name         string
		updateOpts   networks.UpdateOptsBuilder
		expectChange bool
	}{
		{
			name:         "Empty base opts",
			updateOpts:   networks.UpdateOpts{},
			expectChange: false,
		},
		{
			name:         "Empty base opts with revision number",
			updateOpts:   networks.UpdateOpts{RevisionNumber: ptr.To(4)},
			expectChange: false,
		},
		{
			name:         "Updated opts",
			updateOpts:   networks.UpdateOpts{Name: ptr.To("updated")},
			expectChange: true,
		},
		{
			name: "Empty extended opts",
			updateOpts: portsecurity.NetworkUpdateOptsExt{
				UpdateOptsBuilder: networks.UpdateOpts{},
			},
			expectChange: false,
		},
		{
			name: "Updated extended opts",
			updateOpts: portsecurity.NetworkUpdateOptsExt{
				UpdateOptsBuilder:   networks.UpdateOpts{},
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
		{name: "No value provided, existing is set", newValue: nil, existingValue: false, expectChange: true},
		{name: "No value provided, existing is default", newValue: nil, existingValue: true, expectChange: false},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resource := &orcv1alpha1.NetworkResourceSpec{AdminStateUp: tt.newValue}
			net := &networks.Network{AdminStateUp: tt.existingValue}
			osResource := &osclients.NetworkExt{Network: *net}

			updateOpts := networks.UpdateOpts{}
			handleAdminStateUpUpdate(&updateOpts, resource, osResource)

			got, _ := needsUpdate(updateOpts)
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
			resource := &orcv1alpha1.Network{}
			resource.Name = "object-name"
			resource.Spec = orcv1alpha1.NetworkSpec{
				Resource: &orcv1alpha1.NetworkResourceSpec{Name: tt.newValue},
			}
			net := &networks.Network{Name: tt.existingValue}
			osResource := &osclients.NetworkExt{Network: *net}

			updateOpts := networks.UpdateOpts{}
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
			resource := &orcv1alpha1.NetworkResourceSpec{Description: tt.newValue}
			net := &networks.Network{Description: tt.existingValue}
			osResource := &osclients.NetworkExt{Network: *net}

			updateOpts := networks.UpdateOpts{}
			handleDescriptionUpdate(&updateOpts, resource, osResource)

			got, _ := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})

	}
}

func TestHandleSharedUpdate(t *testing.T) {
	ptrToBool := ptr.To[bool]
	testCases := []struct {
		name          string
		newValue      *bool
		existingValue bool
		expectChange  bool
	}{
		{name: "Identical", newValue: ptrToBool(true), existingValue: true, expectChange: false},
		{name: "Different", newValue: ptrToBool(true), existingValue: false, expectChange: true},
		{name: "No value provided, existing is set", newValue: nil, existingValue: true, expectChange: true},
		{name: "No value provided, existing is default", newValue: nil, existingValue: false, expectChange: false},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resource := &orcv1alpha1.NetworkResourceSpec{Shared: tt.newValue}
			net := &networks.Network{Shared: tt.existingValue}
			osResource := &osclients.NetworkExt{Network: *net}

			updateOpts := networks.UpdateOpts{}
			handleSharedUpdate(&updateOpts, resource, osResource)

			got, _ := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})

	}
}

func TestHandlePortSecurityEnabledUpdate(t *testing.T) {
	testCases := []struct {
		name          string
		newValue      *bool
		existingValue bool
		expectChange  bool
	}{
		{name: "Identical", newValue: ptr.To(true), existingValue: true, expectChange: false},
		{name: "Different", newValue: ptr.To(false), existingValue: true, expectChange: true},
		{name: "No value provided, existing is cleared", newValue: nil, existingValue: false, expectChange: false},
		{name: "No value provided, existing is default", newValue: nil, existingValue: true, expectChange: false},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resource := &orcv1alpha1.NetworkResourceSpec{PortSecurityEnabled: tt.newValue}
			osResource := &osclients.NetworkExt{
				PortSecurityExt: portsecurity.PortSecurityExt{
					PortSecurityEnabled: tt.existingValue,
				},
			}

			updateOpts := handlePortSecurityEnabledUpdate(&networks.UpdateOpts{}, resource, osResource)

			got, _ := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})
	}
}

func TestHandleMTUUpdate(t *testing.T) {
	testCases := []struct {
		name          string
		newValue      *orcv1alpha1.MTU
		existingValue int
		expectChange  bool
	}{
		{name: "Identical", newValue: ptr.To(orcv1alpha1.MTU(1500)), existingValue: 1500, expectChange: false},
		{name: "Different", newValue: ptr.To(orcv1alpha1.MTU(1400)), existingValue: 1500, expectChange: true},
		{name: "No value provided, existing is set", newValue: nil, existingValue: 1500, expectChange: false},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resource := &orcv1alpha1.NetworkResourceSpec{MTU: tt.newValue}
			osResource := &osclients.NetworkExt{
				NetworkMTUExt: mtu.NetworkMTUExt{
					MTU: tt.existingValue,
				},
			}

			updateOpts := handleMTUUpdate(&networks.UpdateOpts{}, resource, osResource)

			got, _ := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})
	}
}

func TestHandleExternalUpdate(t *testing.T) {
	testCases := []struct {
		name          string
		newValue      *bool
		existingValue bool
		expectChange  bool
	}{
		{name: "Identical", newValue: ptr.To(true), existingValue: true, expectChange: false},
		{name: "Different", newValue: ptr.To(false), existingValue: true, expectChange: true},
		{name: "No value provided, existing is set", newValue: nil, existingValue: true, expectChange: false},
		{name: "No value provided, existing is default", newValue: nil, existingValue: false, expectChange: false},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resource := &orcv1alpha1.NetworkResourceSpec{External: tt.newValue}
			osResource := &osclients.NetworkExt{
				NetworkExternalExt: external.NetworkExternalExt{
					External: tt.existingValue,
				},
			}

			updateOpts := handleExternalUpdate(&networks.UpdateOpts{}, resource, osResource)

			got, _ := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})
	}
}

func TestHandleDNSDomainUpdate(t *testing.T) {
	testCases := []struct {
		name          string
		newValue      *orcv1alpha1.DNSDomain
		existingValue string
		expectChange  bool
	}{
		{name: "Identical", newValue: ptr.To(orcv1alpha1.DNSDomain("foo.com")), existingValue: "foo.com", expectChange: false},
		{name: "Different", newValue: ptr.To(orcv1alpha1.DNSDomain("bar.com")), existingValue: "foo.com", expectChange: true},
		{name: "No value provided, existing is set", newValue: nil, existingValue: "foo.com", expectChange: false},
		{name: "No value provided, existing is empty", newValue: nil, existingValue: "", expectChange: false},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resource := &orcv1alpha1.NetworkResourceSpec{DNSDomain: tt.newValue}
			osResource := &osclients.NetworkExt{
				NetworkDNSExt: dns.NetworkDNSExt{
					DNSDomain: tt.existingValue,
				},
			}

			updateOpts := handleDNSDomainUpdate(&networks.UpdateOpts{}, resource, osResource)

			got, _ := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v, updateOpts: %v", tt.expectChange, got, updateOpts)
			}
		})
	}
}
