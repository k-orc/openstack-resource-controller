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

package dnszone

import (
	"context"
	"errors"
	"iter"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/dns/v2/zones"
	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v2/internal/osclients/mock"
	orcerrors "github.com/k-orc/openstack-resource-controller/v2/internal/util/errors"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

var (
	errTest = errors.New("test error")
)

const testZoneName = "example.com."

func mockListZones(zonesList []zones.Zone) iter.Seq2[*zones.Zone, error] {
	return func(yield func(*zones.Zone, error) bool) {
		for i := range zonesList {
			if !yield(&zonesList[i], nil) {
				return
			}
		}
	}
}

type zoneResult struct {
	zone *zones.Zone
	err  error
}

func TestGetResourceID(t *testing.T) {
	actuator := dnsZoneActuator{}
	zone := &zones.Zone{ID: "test-zone-id"}
	if got := actuator.GetResourceID(zone); got != "test-zone-id" {
		t.Errorf("Expected test-zone-id, got %s", got)
	}
}

func TestGetOSResourceByID(t *testing.T) {
	ctx := context.Background()
	mockctrl := gomock.NewController(t)
	defer mockctrl.Finish()
	mockClient := mock.NewMockDNSZoneClient(mockctrl)

	mockClient.EXPECT().GetZone(ctx, "found").Return(&zones.Zone{ID: "found", Name: testZoneName}, nil)
	mockClient.EXPECT().GetZone(ctx, "notfound").Return(nil, errTest)

	actuator := dnsZoneActuator{osClient: mockClient}

	// Case 1: success
	res, status := actuator.GetOSResourceByID(ctx, "found")
	if status != nil {
		t.Errorf("Expected nil status, got %v", status)
	}
	if res == nil || res.ID != "found" {
		t.Errorf("Expected zone with ID 'found', got %v", res)
	}

	// Case 2: error
	res, status = actuator.GetOSResourceByID(ctx, "notfound")
	if status == nil {
		t.Errorf("Expected error status, got nil")
	}
	if res != nil {
		t.Errorf("Expected nil zone, got %v", res)
	}
}

func TestListOSResourcesForAdoption(t *testing.T) {
	for _, tc := range [...]struct {
		name         string
		resourceSpec orcv1alpha1.DNSZoneResourceSpec
		zones        []zones.Zone
		expectCount  int
		expectIDs    []string
	}{
		{
			name: "exact match",
			resourceSpec: orcv1alpha1.DNSZoneResourceSpec{
				Name:        ptr.To[orcv1alpha1.OpenStackName](testZoneName),
				Email:       ptr.To("admin@example.com"),
				Description: ptr.To("desc"),
				TTL:         ptr.To[int32](3600),
				Type:        orcv1alpha1.DNSZoneTypePrimary,
			},
			zones: []zones.Zone{
				{ID: "1", Name: testZoneName, Email: "admin@example.com", Description: "desc", TTL: 3600, Type: "PRIMARY"},
				{ID: "2", Name: testZoneName, Email: "other@example.com", Description: "desc", TTL: 3600, Type: "PRIMARY"},
			},
			expectCount: 1,
			expectIDs:   []string{"1"},
		},
		{
			name: "no spec description, matches empty description",
			resourceSpec: orcv1alpha1.DNSZoneResourceSpec{
				Name:  ptr.To[orcv1alpha1.OpenStackName](testZoneName),
				Email: ptr.To("admin@example.com"),
				Type:  orcv1alpha1.DNSZoneTypePrimary,
			},
			zones: []zones.Zone{
				{ID: "1", Name: testZoneName, Email: "admin@example.com", Description: "", Type: "PRIMARY"},
				{ID: "2", Name: testZoneName, Email: "admin@example.com", Description: "some-desc", Type: "PRIMARY"},
			},
			expectCount: 1,
			expectIDs:   []string{"1"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			mockctrl := gomock.NewController(t)
			defer mockctrl.Finish()
			mockClient := mock.NewMockDNSZoneClient(mockctrl)

			mockClient.EXPECT().ListZones(ctx, zones.ListOpts{Name: testZoneName}).Return(mockListZones(tc.zones))

			actuator := dnsZoneActuator{osClient: mockClient}

			obj := &orcv1alpha1.DNSZone{
				ObjectMeta: metav1.ObjectMeta{
					Name: testZoneName,
				},
				Spec: orcv1alpha1.DNSZoneSpec{
					Resource: &tc.resourceSpec,
				},
			}

			iter, ok := actuator.ListOSResourcesForAdoption(ctx, obj)
			if !ok {
				t.Fatalf("Expected ok to be true")
			}

			var results []zoneResult
			for zone, err := range iter {
				results = append(results, zoneResult{zone, err})
			}

			if len(results) != tc.expectCount {
				t.Errorf("Expected %d results, got %d", tc.expectCount, len(results))
			}

			for i, id := range tc.expectIDs {
				if i < len(results) && results[i].zone.ID != id {
					t.Errorf("Expected ID %s, got %s", id, results[i].zone.ID)
				}
			}
		})
	}
}

func TestListOSResourcesForAdoption_NilSpec(t *testing.T) {
	ctx := context.Background()
	actuator := dnsZoneActuator{}
	_, ok := actuator.ListOSResourcesForAdoption(ctx, &orcv1alpha1.DNSZone{})
	if ok {
		t.Errorf("Expected ok to be false with nil spec")
	}
}

func TestListOSResourcesForImport(t *testing.T) {
	for _, tc := range [...]struct {
		name         string
		filter       orcv1alpha1.DNSZoneFilter
		zones        []zones.Zone
		expectCount  int
		expectIDs    []string
		expectedOpts zones.ListOpts
	}{
		{
			name: "match name and email",
			filter: orcv1alpha1.DNSZoneFilter{
				Name:  ptr.To[orcv1alpha1.OpenStackName](testZoneName),
				Email: ptr.To("admin@example.com"),
			},
			zones: []zones.Zone{
				{ID: "1", Name: testZoneName, Email: "admin@example.com"},
				{ID: "2", Name: testZoneName, Email: "other@example.com"},
			},
			expectCount:  1,
			expectIDs:    []string{"1"},
			expectedOpts: zones.ListOpts{Name: testZoneName},
		},
		{
			name: "match TTL and Type",
			filter: orcv1alpha1.DNSZoneFilter{
				TTL:  ptr.To[int32](1800),
				Type: ptr.To(orcv1alpha1.DNSZoneTypePrimary),
			},
			zones: []zones.Zone{
				{ID: "1", Name: testZoneName, TTL: 1800, Type: "PRIMARY"},
				{ID: "2", Name: testZoneName, TTL: 3600, Type: "PRIMARY"},
				{ID: "3", Name: testZoneName, TTL: 1800, Type: "SECONDARY"},
			},
			expectCount:  1,
			expectIDs:    []string{"1"},
			expectedOpts: zones.ListOpts{},
		},
		{
			name: "match description",
			filter: orcv1alpha1.DNSZoneFilter{
				Description: ptr.To("special zone"),
			},
			zones: []zones.Zone{
				{ID: "1", Name: testZoneName, Description: "special zone"},
				{ID: "2", Name: testZoneName, Description: "other zone"},
			},
			expectCount:  1,
			expectIDs:    []string{"1"},
			expectedOpts: zones.ListOpts{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			mockctrl := gomock.NewController(t)
			defer mockctrl.Finish()
			mockClient := mock.NewMockDNSZoneClient(mockctrl)

			mockClient.EXPECT().ListZones(ctx, tc.expectedOpts).Return(mockListZones(tc.zones))

			actuator := dnsZoneActuator{osClient: mockClient}

			iter, status := actuator.ListOSResourcesForImport(ctx, &orcv1alpha1.DNSZone{}, tc.filter)
			if status != nil {
				t.Fatalf("Expected nil status, got %v", status)
			}

			var results []zoneResult
			for zone, err := range iter {
				results = append(results, zoneResult{zone, err})
			}

			if len(results) != tc.expectCount {
				t.Errorf("Expected %d results, got %d", tc.expectCount, len(results))
			}

			for i, id := range tc.expectIDs {
				if i < len(results) && results[i].zone.ID != id {
					t.Errorf("Expected ID %s, got %s", id, results[i].zone.ID)
				}
			}
		})
	}
}

func TestCreateResource(t *testing.T) {
	ctx := context.Background()

	obj := &orcv1alpha1.DNSZone{
		Spec: orcv1alpha1.DNSZoneSpec{
			Resource: &orcv1alpha1.DNSZoneResourceSpec{
				Name:        ptr.To[orcv1alpha1.OpenStackName](testZoneName),
				Email:       ptr.To("admin@example.com"),
				Description: ptr.To("desc"),
				TTL:         ptr.To[int32](3600),
				Type:        orcv1alpha1.DNSZoneTypePrimary,
			},
		},
	}

	expectedCreateOpts := zones.CreateOpts{
		Name:        testZoneName,
		Email:       "admin@example.com",
		Description: "desc",
		Type:        "PRIMARY",
		TTL:         3600,
	}

	// Case 1: Success
	{
		mockctrl := gomock.NewController(t)
		mockClient := mock.NewMockDNSZoneClient(mockctrl)
		mockClient.EXPECT().CreateZone(ctx, expectedCreateOpts).Return(&zones.Zone{
			ID:          "created-id",
			Name:        testZoneName,
			Email:       "admin@example.com",
			Description: "desc",
			TTL:         3600,
			Type:        "PRIMARY",
		}, nil)

		actuator := dnsZoneActuator{osClient: mockClient}
		res, status := actuator.CreateResource(ctx, obj)
		if status != nil {
			t.Fatalf("Expected nil status, got %v", status)
		}
		if res.ID != "created-id" || res.Name != testZoneName || res.Email != "admin@example.com" || res.Description != "desc" || res.TTL != 3600 || res.Type != "PRIMARY" {
			t.Errorf("Created resource does not match: %v", res)
		}
		mockctrl.Finish()
	}

	// Case 2: Conflict (already exists)
	{
		conflictErr := gophercloud.ErrUnexpectedResponseCode{
			URL:      "http://designate/zones",
			Method:   "POST",
			Expected: []int{201},
			Actual:   409,
			Body:     []byte(`{"message": "Zone already exists"}`),
		}

		mockctrl := gomock.NewController(t)
		mockClient := mock.NewMockDNSZoneClient(mockctrl)
		mockClient.EXPECT().CreateZone(ctx, expectedCreateOpts).Return(nil, conflictErr)

		actuatorConflict := dnsZoneActuator{osClient: mockClient}
		_, status := actuatorConflict.CreateResource(ctx, obj)
		if status == nil {
			t.Fatalf("Expected non-nil status on conflict")
		}
		needsReschedule, err := status.NeedsReschedule()
		if !needsReschedule {
			t.Errorf("Expected needsReschedule on error")
		}
		if err == nil {
			t.Errorf("Expected error from status, got nil")
		}
		if !orcerrors.IsConflict(err) {
			t.Errorf("Expected conflict error, got %v", err)
		}
		if orcerrors.IsRetryable(err) {
			t.Errorf("Expected conflict error to be terminal (not retryable)")
		}
		var terminalError *orcerrors.TerminalError
		if !errors.As(err, &terminalError) {
			t.Errorf("Expected error to contain a *TerminalError, got %T", err)
		} else if terminalError.Reason != string(orcv1alpha1.ConditionReasonUnrecoverableError) {
			t.Errorf("Expected TerminalError reason %s, got %s", orcv1alpha1.ConditionReasonUnrecoverableError, terminalError.Reason)
		}
		mockctrl.Finish()
	}

	// Case 3: Other errors (transient API errors)
	{
		mockctrl := gomock.NewController(t)
		mockClient := mock.NewMockDNSZoneClient(mockctrl)
		mockClient.EXPECT().CreateZone(ctx, expectedCreateOpts).Return(nil, errTest)

		actuatorError := dnsZoneActuator{osClient: mockClient}
		_, status := actuatorError.CreateResource(ctx, obj)
		if status == nil {
			t.Fatalf("Expected non-nil status on generic API error")
		}
		needsReschedule, err := status.NeedsReschedule()
		if !needsReschedule {
			t.Errorf("Expected needsReschedule to be true")
		}
		if err == nil {
			t.Errorf("Expected error from status, got nil")
		}
		if !errors.Is(err, errTest) {
			t.Errorf("Expected error %v, got %v", errTest, err)
		}
		mockctrl.Finish()
	}
}

func TestCreateResource_NilSpec(t *testing.T) {
	ctx := context.Background()
	actuator := dnsZoneActuator{}
	_, status := actuator.CreateResource(ctx, &orcv1alpha1.DNSZone{})
	if status == nil {
		t.Fatalf("Expected status to be non-nil when resource is nil")
	}
	err := status.GetError()
	if err == nil {
		t.Fatalf("Expected error when resource is nil")
	}
	var terminalError *orcerrors.TerminalError
	if !errors.As(err, &terminalError) {
		t.Errorf("Expected error to be a terminal error, got %T", err)
	}
}

func TestDeleteResource(t *testing.T) {
	ctx := context.Background()
	mockctrl := gomock.NewController(t)
	defer mockctrl.Finish()
	mockClient := mock.NewMockDNSZoneClient(mockctrl)

	mockClient.EXPECT().DeleteZone(ctx, "delete-me").Return(nil)

	actuator := dnsZoneActuator{osClient: mockClient}
	zone := &zones.Zone{ID: "delete-me"}

	status := actuator.DeleteResource(ctx, &orcv1alpha1.DNSZone{}, zone)
	if status != nil {
		t.Errorf("Expected nil status, got %v", status)
	}
}

func TestUpdateResource(t *testing.T) {
	ctx := context.Background()

	obj := &orcv1alpha1.DNSZone{
		Spec: orcv1alpha1.DNSZoneSpec{
			Resource: &orcv1alpha1.DNSZoneResourceSpec{
				Name:        ptr.To[orcv1alpha1.OpenStackName](testZoneName),
				Email:       ptr.To("new-admin@example.com"),
				Description: ptr.To("new-desc"),
				TTL:         ptr.To[int32](7200),
				Type:        orcv1alpha1.DNSZoneTypePrimary,
			},
		},
	}
	osResource := &zones.Zone{
		ID:          "zone-id",
		Name:        testZoneName,
		Email:       "admin@example.com",
		Description: "desc",
		TTL:         3600,
		Type:        "PRIMARY",
	}

	expectedUpdateOpts := zones.UpdateOpts{
		Email:       "new-admin@example.com",
		Description: ptr.To("new-desc"),
		TTL:         7200,
	}

	// Case 1: Progress (change requires update)
	{
		mockctrl := gomock.NewController(t)
		mockClient := mock.NewMockDNSZoneClient(mockctrl)
		mockClient.EXPECT().UpdateZone(ctx, "zone-id", expectedUpdateOpts).Return(&zones.Zone{ID: "zone-id"}, nil)

		actuator := dnsZoneActuator{osClient: mockClient}
		status := actuator.updateResource(ctx, obj, osResource)
		if status == nil {
			t.Fatalf("Expected progress status, got nil")
		}
		needsReschedule, err := status.NeedsReschedule()
		if !needsReschedule {
			t.Errorf("Expected needsReschedule to be true")
		}
		if err != nil {
			t.Errorf("Expected nil error, got %v", err)
		}
		mockctrl.Finish()
	}

	// Case 2: No change (no update call should be made)
	{
		mockctrl := gomock.NewController(t)
		mockClient := mock.NewMockDNSZoneClient(mockctrl)
		// Expect no call to UpdateZone

		actuator := dnsZoneActuator{osClient: mockClient}
		unchangedObj := &orcv1alpha1.DNSZone{
			Spec: orcv1alpha1.DNSZoneSpec{
				Resource: &orcv1alpha1.DNSZoneResourceSpec{
					Name:        ptr.To[orcv1alpha1.OpenStackName](testZoneName),
					Email:       ptr.To("admin@example.com"),
					Description: ptr.To("desc"),
					TTL:         ptr.To[int32](3600),
					Type:        orcv1alpha1.DNSZoneTypePrimary,
				},
			},
		}
		status := actuator.updateResource(ctx, unchangedObj, osResource)
		if status != nil {
			t.Errorf("Expected nil status when no update needed, got %v", status)
		}
		mockctrl.Finish()
	}

	// Case 3: Error during update (transient error)
	{
		mockctrl := gomock.NewController(t)
		mockClient := mock.NewMockDNSZoneClient(mockctrl)
		mockClient.EXPECT().UpdateZone(ctx, "zone-id", expectedUpdateOpts).Return(nil, errTest)

		actuator := dnsZoneActuator{osClient: mockClient}
		status := actuator.updateResource(ctx, obj, osResource)
		if status == nil {
			t.Fatalf("Expected progress status on error, got nil")
		}
		needsReschedule, err := status.NeedsReschedule()
		if !needsReschedule {
			t.Errorf("Expected needsReschedule to be true")
		}
		if !errors.Is(err, errTest) {
			t.Errorf("Expected error %v, got %v", errTest, err)
		}
		mockctrl.Finish()
	}
}

func TestUpdateResource_NilSpec(t *testing.T) {
	ctx := context.Background()
	actuator := dnsZoneActuator{}
	status := actuator.updateResource(ctx, &orcv1alpha1.DNSZone{}, &zones.Zone{})
	if status == nil {
		t.Fatalf("Expected status to be non-nil when resource is nil")
	}
	err := status.GetError()
	if err == nil {
		t.Fatalf("Expected error when resource is nil")
	}
	var terminalError *orcerrors.TerminalError
	if !errors.As(err, &terminalError) {
		t.Errorf("Expected error to be a terminal error, got %T", err)
	}
}

func TestNeedsUpdate(t *testing.T) {
	testCases := []struct {
		name         string
		updateOpts   zones.UpdateOpts
		expectChange bool
	}{
		{
			name:         "Empty base opts",
			updateOpts:   zones.UpdateOpts{},
			expectChange: false,
		},
		{
			name:         "Updated opts",
			updateOpts:   zones.UpdateOpts{Description: ptr.To("updated")},
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
			resource := &orcv1alpha1.DNSZoneResourceSpec{Description: tt.newValue}
			osResource := &osResourceT{Description: tt.existingValue}

			updateOpts := zones.UpdateOpts{}
			handleDescriptionUpdate(&updateOpts, resource, osResource)

			got, _ := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})

	}
}

func TestHandleEmailUpdate(t *testing.T) {
	testCases := []struct {
		name          string
		newValue      string
		existingValue string
		expectChange  bool
	}{
		{name: "Identical", newValue: "admin@example.com", existingValue: "admin@example.com", expectChange: false},
		{name: "Different", newValue: "new-admin@example.com", existingValue: "admin@example.com", expectChange: true},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resource := &orcv1alpha1.DNSZoneResourceSpec{Email: ptr.To(tt.newValue)}
			osResource := &osResourceT{Email: tt.existingValue}

			updateOpts := zones.UpdateOpts{}
			handleEmailUpdate(&updateOpts, resource, osResource)

			got, _ := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})
	}
}

func TestHandleTTLUpdate(t *testing.T) {
	testCases := []struct {
		name          string
		newValue      *int32
		existingValue int
		expectChange  bool
	}{
		{name: "Identical", newValue: ptr.To[int32](3600), existingValue: 3600, expectChange: false},
		{name: "Different", newValue: ptr.To[int32](1800), existingValue: 3600, expectChange: true},
		{name: "Nil value", newValue: nil, existingValue: 3600, expectChange: false},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resource := &orcv1alpha1.DNSZoneResourceSpec{TTL: tt.newValue}
			osResource := &osResourceT{TTL: tt.existingValue}

			updateOpts := zones.UpdateOpts{}
			handleTTLUpdate(&updateOpts, resource, osResource)

			got, _ := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})
	}
}

func TestHandleMastersUpdate(t *testing.T) {
	testCases := []struct {
		name          string
		newValue      []string
		existingValue []string
		expectChange  bool
	}{
		{name: "Identical", newValue: []string{"1.2.3.4"}, existingValue: []string{"1.2.3.4"}, expectChange: false},
		{name: "Different length", newValue: []string{"1.2.3.4", "5.6.7.8"}, existingValue: []string{"1.2.3.4"}, expectChange: true},
		{name: "Different value", newValue: []string{"1.2.3.4"}, existingValue: []string{"5.6.7.8"}, expectChange: true},
		{name: "Both empty", newValue: nil, existingValue: nil, expectChange: false},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resource := &orcv1alpha1.DNSZoneResourceSpec{Masters: tt.newValue}
			osResource := &osResourceT{Masters: tt.existingValue}

			updateOpts := zones.UpdateOpts{}
			handleMastersUpdate(&updateOpts, resource, osResource)

			got, _ := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})
	}
}

func TestGetResourceReconcilers(t *testing.T) {
	actuator := dnsZoneActuator{}
	reconcilers, status := actuator.GetResourceReconcilers(context.Background(), &orcv1alpha1.DNSZone{}, &zones.Zone{}, nil)
	if status != nil {
		t.Errorf("Expected nil status, got %v", status)
	}
	if len(reconcilers) != 1 {
		t.Errorf("Expected 1 reconciler, got %d", len(reconcilers))
	}
}

func TestHelperFactory_NewAPIObjectAdapter(t *testing.T) {
	factory := dnszoneHelperFactory{}
	obj := &orcv1alpha1.DNSZone{
		Spec: orcv1alpha1.DNSZoneSpec{
			ManagementPolicy: orcv1alpha1.ManagementPolicyManaged,
			ManagedOptions: &orcv1alpha1.ManagedOptions{
				OnDelete: orcv1alpha1.OnDeleteDelete,
			},
			Resource: &orcv1alpha1.DNSZoneResourceSpec{
				Name: ptr.To[orcv1alpha1.OpenStackName](testZoneName),
			},
			Import: &orcv1alpha1.DNSZoneImport{
				ID: ptr.To("imported-id"),
				Filter: &orcv1alpha1.DNSZoneFilter{
					Name: ptr.To[orcv1alpha1.OpenStackName](testZoneName),
				},
			},
		},
		Status: orcv1alpha1.DNSZoneStatus{
			ID: ptr.To("status-id"),
		},
	}
	adapter := factory.NewAPIObjectAdapter(obj)
	if adapter.GetObject() != obj {
		t.Errorf("Expected GetObject to return the original object")
	}
	if adapter.GetManagementPolicy() != orcv1alpha1.ManagementPolicyManaged {
		t.Errorf("Expected GetManagementPolicy to match")
	}
	if adapter.GetManagedOptions().OnDelete != orcv1alpha1.OnDeleteDelete {
		t.Errorf("Expected GetManagedOptions to match")
	}
	if *adapter.GetStatusID() != "status-id" {
		t.Errorf("Expected GetStatusID to return 'status-id'")
	}
	if adapter.GetResourceSpec().Name == nil || string(*adapter.GetResourceSpec().Name) != testZoneName {
		t.Errorf("Expected GetResourceSpec Name to match")
	}
	if *adapter.GetImportID() != "imported-id" {
		t.Errorf("Expected GetImportID to return 'imported-id'")
	}
	if adapter.GetImportFilter().Name == nil || string(*adapter.GetImportFilter().Name) != testZoneName {
		t.Errorf("Expected GetImportFilter Name to match")
	}
}

func TestHelperFactory_NewAPIObjectAdapter_NilImport(t *testing.T) {
	factory := dnszoneHelperFactory{}
	obj := &orcv1alpha1.DNSZone{
		Spec: orcv1alpha1.DNSZoneSpec{},
	}
	adapter := factory.NewAPIObjectAdapter(obj)
	if adapter.GetImportID() != nil {
		t.Errorf("Expected GetImportID to be nil")
	}
	if adapter.GetImportFilter() != nil {
		t.Errorf("Expected GetImportFilter to be nil")
	}
}

func TestGetDNSZoneName(t *testing.T) {
	testCases := []struct {
		name         string
		specName     *string
		objName      string
		expectedName string
	}{
		{
			name:         "spec name ends with dot",
			specName:     ptr.To("example.com."),
			objName:      "my-dnszone",
			expectedName: "example.com.",
		},
		{
			name:         "spec name has no dot",
			specName:     ptr.To("example.com"),
			objName:      "my-dnszone",
			expectedName: "example.com.",
		},
		{
			name:         "fallback to obj name with dot",
			specName:     nil,
			objName:      "example.org.",
			expectedName: "example.org.",
		},
		{
			name:         "fallback to obj name without dot",
			specName:     nil,
			objName:      "example.org",
			expectedName: "example.org.",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			obj := &orcv1alpha1.DNSZone{}
			obj.Name = tt.objName
			if tt.specName != nil {
				obj.Spec.Resource = &orcv1alpha1.DNSZoneResourceSpec{
					Name: ptr.To[orcv1alpha1.OpenStackName](orcv1alpha1.OpenStackName(*tt.specName)),
				}
			} else {
				obj.Spec.Resource = &orcv1alpha1.DNSZoneResourceSpec{}
			}

			got := getDNSZoneName(obj)
			if got != tt.expectedName {
				t.Errorf("Expected name %q, got %q", tt.expectedName, got)
			}
		})
	}
}
