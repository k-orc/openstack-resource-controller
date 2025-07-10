package subnet

import (
	"reflect"
	"testing"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/subnets"
	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	"k8s.io/utils/ptr"
)

func TestNeedsUpdate(t *testing.T) {
	testCases := []struct {
		name         string
		updateOpts   subnets.UpdateOpts
		expectChange bool
	}{
		{
			name:         "Empty base opts",
			updateOpts:   subnets.UpdateOpts{},
			expectChange: false,
		},
		{
			name:         "Empty base opts with revision number",
			updateOpts:   subnets.UpdateOpts{RevisionNumber: ptr.To(4)},
			expectChange: false,
		},
		{
			name:         "Updated opts",
			updateOpts:   subnets.UpdateOpts{Name: ptr.To("updated")},
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
			resource := &orcv1alpha1.Subnet{}
			resource.Name = "object-name"
			resource.Spec = orcv1alpha1.SubnetSpec{
				Resource: &orcv1alpha1.SubnetResourceSpec{Name: tt.newValue},
			}
			osResource := &subnets.Subnet{Name: tt.existingValue}

			updateOpts := subnets.UpdateOpts{}
			handleNameUpdate(&updateOpts, resource, osResource)

			got := !reflect.ValueOf(updateOpts).IsZero()
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got %v", tt.expectChange, got)
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
			resource := &orcv1alpha1.SubnetResourceSpec{Description: tt.newValue}
			osResource := &subnets.Subnet{Description: tt.existingValue}

			updateOpts := subnets.UpdateOpts{}
			handleDescriptionUpdate(&updateOpts, resource, osResource)

			got, _ := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})

	}
}

func TestHandleAllocationPoolsUpdate(t *testing.T) {
	testCases := []struct {
		name          string
		newValue      []orcv1alpha1.AllocationPool
		existingValue []subnets.AllocationPool
		expectChange  bool
	}{
		{
			name: "Identical",
			newValue: []orcv1alpha1.AllocationPool{
				{Start: orcv1alpha1.IPvAny("192.168.1.1"), End: orcv1alpha1.IPvAny("192.168.1.10")},
				{Start: orcv1alpha1.IPvAny("192.168.2.1"), End: orcv1alpha1.IPvAny("192.168.2.10")},
			},
			existingValue: []subnets.AllocationPool{
				{Start: "192.168.1.1", End: "192.168.1.10"},
				{Start: "192.168.2.1", End: "192.168.2.10"},
			},
			expectChange: false,
		},
		{
			name: "Different entry",
			newValue: []orcv1alpha1.AllocationPool{
				{Start: orcv1alpha1.IPvAny("192.168.1.1"), End: orcv1alpha1.IPvAny("192.168.1.10")},
				{Start: orcv1alpha1.IPvAny("192.168.5.1"), End: orcv1alpha1.IPvAny("192.168.5.10")},
			},
			existingValue: []subnets.AllocationPool{
				{Start: "192.168.1.1", End: "192.168.1.10"},
				{Start: "192.168.2.1", End: "192.168.2.10"},
			},
			expectChange: true,
		},
		{
			name: "Identical, out of order",
			newValue: []orcv1alpha1.AllocationPool{
				{Start: orcv1alpha1.IPvAny("192.168.2.1"), End: orcv1alpha1.IPvAny("192.168.2.10")},
				{Start: orcv1alpha1.IPvAny("192.168.1.1"), End: orcv1alpha1.IPvAny("192.168.1.10")},
			},
			existingValue: []subnets.AllocationPool{
				{Start: "192.168.1.1", End: "192.168.1.10"},
				{Start: "192.168.2.1", End: "192.168.2.10"},
			},
			expectChange: false,
		},
		{
			name: "Identical, with duplicate",
			newValue: []orcv1alpha1.AllocationPool{
				{Start: orcv1alpha1.IPvAny("192.168.1.1"), End: orcv1alpha1.IPvAny("192.168.1.10")},
				{Start: orcv1alpha1.IPvAny("192.168.2.1"), End: orcv1alpha1.IPvAny("192.168.2.10")},
				{Start: orcv1alpha1.IPvAny("192.168.2.1"), End: orcv1alpha1.IPvAny("192.168.2.10")},
			},
			existingValue: []subnets.AllocationPool{
				{Start: "192.168.1.1", End: "192.168.1.10"},
				{Start: "192.168.2.1", End: "192.168.2.10"},
			},
			expectChange: false,
		},
		{
			name: "Removing an entry",
			newValue: []orcv1alpha1.AllocationPool{
				{Start: orcv1alpha1.IPvAny("192.168.1.1"), End: orcv1alpha1.IPvAny("192.168.1.10")},
			},
			existingValue: []subnets.AllocationPool{
				{Start: "192.168.1.1", End: "192.168.1.10"},
				{Start: "192.168.2.1", End: "192.168.2.10"},
			},
			expectChange: true,
		},
		{
			name: "Adding an entry",
			newValue: []orcv1alpha1.AllocationPool{
				{Start: orcv1alpha1.IPvAny("192.168.1.1"), End: orcv1alpha1.IPvAny("192.168.1.10")},
				{Start: orcv1alpha1.IPvAny("192.168.2.1"), End: orcv1alpha1.IPvAny("192.168.2.10")},
			},
			existingValue: []subnets.AllocationPool{
				{Start: "192.168.1.1", End: "192.168.1.10"},
			},
			expectChange: true,
		},
		{
			name:     "Default allocation pool",
			newValue: []orcv1alpha1.AllocationPool{},
			existingValue: []subnets.AllocationPool{
				{Start: "192.168.1.1", End: "192.168.1.10"},
			},
			expectChange: false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resource := &orcv1alpha1.SubnetResourceSpec{AllocationPools: tt.newValue}
			osResource := &subnets.Subnet{AllocationPools: tt.existingValue}

			updateOpts := subnets.UpdateOpts{}
			handleAllocationPoolsUpdate(&updateOpts, resource, osResource)

			got, _ := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})

	}
}

func TestHandleHostRoutesUpdate(t *testing.T) {
	testCases := []struct {
		name          string
		newValue      []orcv1alpha1.HostRoute
		existingValue []subnets.HostRoute
		expectChange  bool
	}{
		{
			name: "Identical",
			newValue: []orcv1alpha1.HostRoute{
				{Destination: orcv1alpha1.CIDR("192.168.1.0/24"), NextHop: orcv1alpha1.IPvAny("192.168.100.1")},
				{Destination: orcv1alpha1.CIDR("192.168.2.0/24"), NextHop: orcv1alpha1.IPvAny("192.168.200.1")},
			},
			existingValue: []subnets.HostRoute{
				{DestinationCIDR: "192.168.1.0/24", NextHop: "192.168.100.1"},
				{DestinationCIDR: "192.168.2.0/24", NextHop: "192.168.200.1"},
			},
			expectChange: false,
		},
		{
			name: "Different entry",
			newValue: []orcv1alpha1.HostRoute{
				{Destination: orcv1alpha1.CIDR("192.168.1.0/24"), NextHop: orcv1alpha1.IPvAny("192.168.100.1")},
				{Destination: orcv1alpha1.CIDR("192.168.5.0/24"), NextHop: orcv1alpha1.IPvAny("192.168.500.1")},
			},
			existingValue: []subnets.HostRoute{
				{DestinationCIDR: "192.168.1.0/24", NextHop: "192.168.100.1"},
				{DestinationCIDR: "192.168.2.0/24", NextHop: "192.168.200.1"},
			},
			expectChange: true,
		},
		{
			name: "Identical, out of order",
			newValue: []orcv1alpha1.HostRoute{
				{Destination: orcv1alpha1.CIDR("192.168.2.0/24"), NextHop: orcv1alpha1.IPvAny("192.168.200.1")},
				{Destination: orcv1alpha1.CIDR("192.168.1.0/24"), NextHop: orcv1alpha1.IPvAny("192.168.100.1")},
			},
			existingValue: []subnets.HostRoute{
				{DestinationCIDR: "192.168.1.0/24", NextHop: "192.168.100.1"},
				{DestinationCIDR: "192.168.2.0/24", NextHop: "192.168.200.1"},
			},
			expectChange: false,
		},
		{
			name: "Identical, with duplicates",
			newValue: []orcv1alpha1.HostRoute{
				{Destination: orcv1alpha1.CIDR("192.168.1.0/24"), NextHop: orcv1alpha1.IPvAny("192.168.100.1")},
				{Destination: orcv1alpha1.CIDR("192.168.2.0/24"), NextHop: orcv1alpha1.IPvAny("192.168.200.1")},
				{Destination: orcv1alpha1.CIDR("192.168.2.0/24"), NextHop: orcv1alpha1.IPvAny("192.168.200.1")},
			},
			existingValue: []subnets.HostRoute{
				{DestinationCIDR: "192.168.1.0/24", NextHop: "192.168.100.1"},
				{DestinationCIDR: "192.168.2.0/24", NextHop: "192.168.200.1"},
			},
			expectChange: false,
		},
		{
			name: "Removing an entry",
			newValue: []orcv1alpha1.HostRoute{
				{Destination: orcv1alpha1.CIDR("192.168.1.0/24"), NextHop: orcv1alpha1.IPvAny("192.168.100.1")},
			},
			existingValue: []subnets.HostRoute{
				{DestinationCIDR: "192.168.1.0/24", NextHop: "192.168.100.1"},
				{DestinationCIDR: "192.168.2.0/24", NextHop: "192.168.200.1"},
			},
			expectChange: true,
		},
		{
			name: "Adding an entry",
			newValue: []orcv1alpha1.HostRoute{
				{Destination: orcv1alpha1.CIDR("192.168.1.0/24"), NextHop: orcv1alpha1.IPvAny("192.168.100.1")},
				{Destination: orcv1alpha1.CIDR("192.168.2.0/24"), NextHop: orcv1alpha1.IPvAny("192.168.200.1")},
			},
			existingValue: []subnets.HostRoute{
				{DestinationCIDR: "192.168.1.0/24", NextHop: "192.168.100.1"},
			},
			expectChange: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resource := &orcv1alpha1.SubnetResourceSpec{HostRoutes: tt.newValue}
			osResource := &subnets.Subnet{HostRoutes: tt.existingValue}

			updateOpts := subnets.UpdateOpts{}
			handleHostRoutesUpdate(&updateOpts, resource, osResource)

			got, _ := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})

	}
}

func TestHandleDNSNameserversUpdate(t *testing.T) {
	testCases := []struct {
		name          string
		newValue      []orcv1alpha1.IPvAny
		existingValue []string
		expectChange  bool
	}{
		{
			name:          "Identical",
			newValue:      []orcv1alpha1.IPvAny{"192.168.1.1", "192.168.1.2"},
			existingValue: []string{"192.168.1.1", "192.168.1.2"},
			expectChange:  false,
		},
		{
			name:          "Change order",
			newValue:      []orcv1alpha1.IPvAny{"192.168.1.2", "192.168.1.1"},
			existingValue: []string{"192.168.1.1", "192.168.1.2"},
			expectChange:  true,
		},
		{
			name:          "Duplicate entry, with update",
			newValue:      []orcv1alpha1.IPvAny{"192.168.1.1", "192.168.1.1", "192.168.1.2"},
			existingValue: []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"},
			expectChange:  true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resource := &orcv1alpha1.SubnetResourceSpec{DNSNameservers: tt.newValue}
			osResource := &subnets.Subnet{DNSNameservers: tt.existingValue}

			updateOpts := subnets.UpdateOpts{}
			handleDNSNameserversUpdate(&updateOpts, resource, osResource)

			got, _ := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})

	}
}

func TestHandleEnableDHCPUpdate(t *testing.T) {
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
			resource := &orcv1alpha1.SubnetResourceSpec{EnableDHCP: tt.newValue}
			osResource := &subnets.Subnet{EnableDHCP: tt.existingValue}

			updateOpts := subnets.UpdateOpts{}
			handleEnableDHCPUpdate(&updateOpts, resource, osResource)

			got, _ := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})

	}
}

func TestHandleGatewayUpdate(t *testing.T) {
	ptrToGateway := ptr.To[orcv1alpha1.SubnetGateway]
	ptrToIP := ptr.To[orcv1alpha1.IPvAny]
	testCases := []struct {
		name          string
		newValue      *orcv1alpha1.SubnetGateway
		existingValue string
		expectChange  bool
	}{
		{
			name: "Identical",
			newValue: ptrToGateway(orcv1alpha1.SubnetGateway{
				Type: orcv1alpha1.SubnetGatewayTypeIP,
				IP:   ptrToIP("192.168.1.1"),
			}),
			existingValue: "192.168.1.1",
			expectChange:  false,
		},
		{
			name: "To different IP",
			newValue: ptrToGateway(orcv1alpha1.SubnetGateway{
				Type: orcv1alpha1.SubnetGatewayTypeIP,
				IP:   ptrToIP("192.168.1.2"),
			}),
			existingValue: "192.168.1.1",
			expectChange:  true,
		},
		{
			name: "Disable gateway",
			newValue: ptrToGateway(orcv1alpha1.SubnetGateway{
				Type: orcv1alpha1.SubnetGatewayTypeNone,
			}),
			existingValue: "192.168.1.1",
			expectChange:  true,
		},
		{
			name: "Disabled gateway",
			newValue: ptrToGateway(orcv1alpha1.SubnetGateway{
				Type: orcv1alpha1.SubnetGatewayTypeNone,
			}),
			existingValue: "",
			expectChange:  false,
		},
		{
			name: "Not updating when automatic",
			newValue: ptrToGateway(orcv1alpha1.SubnetGateway{
				Type: orcv1alpha1.SubnetGatewayTypeAutomatic,
			}),
			existingValue: "192.168.1.1",
			expectChange:  false,
		},
		{
			name:          "Not updating when no value provided",
			newValue:      nil,
			existingValue: "192.168.1.1",
			expectChange:  false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resource := &orcv1alpha1.SubnetResourceSpec{Gateway: tt.newValue}
			osResource := &subnets.Subnet{GatewayIP: tt.existingValue}

			updateOpts := subnets.UpdateOpts{}
			handleGatewayUpdate(&updateOpts, resource, osResource)

			got, _ := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})

	}
}
