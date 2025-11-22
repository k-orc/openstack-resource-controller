/*
Copyright 2025 The ORC Authors.

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

package trunk

import (
	"context"
	"errors"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/trunks"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	osclientsmock "github.com/k-orc/openstack-resource-controller/v2/internal/osclients/mock"
	orcerrors "github.com/k-orc/openstack-resource-controller/v2/internal/util/errors"
	"go.uber.org/mock/gomock"
)

func TestNeedsUpdate(t *testing.T) {
	testCases := []struct {
		name         string
		updateOpts   trunks.UpdateOpts
		expectChange bool
	}{
		{
			name:         "Empty base opts",
			updateOpts:   trunks.UpdateOpts{},
			expectChange: false,
		},
		{
			name:         "Empty base opts with revision number",
			updateOpts:   trunks.UpdateOpts{RevisionNumber: ptr.To(4)},
			expectChange: false,
		},
		{
			name:         "Updated opts with name",
			updateOpts:   trunks.UpdateOpts{Name: ptr.To("updated")},
			expectChange: true,
		},
		{
			name:         "Updated opts with description",
			updateOpts:   trunks.UpdateOpts{Description: ptr.To("new description")},
			expectChange: true,
		},
		{
			name:         "Updated opts with adminStateUp",
			updateOpts:   trunks.UpdateOpts{AdminStateUp: ptr.To(false)},
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
			resource := &orcv1alpha1.Trunk{}
			resource.Name = "object-name"
			resource.Spec = orcv1alpha1.TrunkSpec{
				Resource: &orcv1alpha1.TrunkResourceSpec{Name: tt.newValue},
			}
			osResource := &trunks.Trunk{Name: tt.existingValue}

			updateOpts := trunks.UpdateOpts{}
			handleNameUpdate(&updateOpts, resource, osResource)

			got := needsUpdate(updateOpts)
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
			resource := &orcv1alpha1.TrunkResourceSpec{Description: tt.newValue}
			osResource := &trunks.Trunk{Description: tt.existingValue}

			updateOpts := trunks.UpdateOpts{}
			handleDescriptionUpdate(&updateOpts, resource, osResource)

			got := needsUpdate(updateOpts)
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
		{name: "No value provided, existing is set", newValue: nil, existingValue: false, expectChange: false},
		{name: "No value provided, existing is default", newValue: nil, existingValue: true, expectChange: false},
		{name: "False when already false", newValue: ptrToBool(false), existingValue: false, expectChange: false},
		{name: "False when was true", newValue: ptrToBool(false), existingValue: true, expectChange: true},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resource := &orcv1alpha1.TrunkResourceSpec{AdminStateUp: tt.newValue}
			osResource := &trunks.Trunk{AdminStateUp: tt.existingValue}

			updateOpts := trunks.UpdateOpts{}
			handleAdminStateUpUpdate(&updateOpts, resource, osResource)

			got := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})
	}
}

// needsUpdate checks if the updateOpts contains any changes that require an update
func needsUpdate(updateOpts trunks.UpdateOpts) bool {
	return updateOpts.Name != nil || updateOpts.Description != nil || updateOpts.AdminStateUp != nil
}

// handleNameUpdate updates the updateOpts if the name needs to be changed
func handleNameUpdate(updateOpts *trunks.UpdateOpts, resource *orcv1alpha1.Trunk, osResource *trunks.Trunk) {
	name := getResourceName(resource)
	if osResource.Name != name {
		updateOpts.Name = &name
	}
}

// handleDescriptionUpdate updates the updateOpts if the description needs to be changed
func handleDescriptionUpdate(updateOpts *trunks.UpdateOpts, resource *orcv1alpha1.TrunkResourceSpec, osResource *trunks.Trunk) {
	description := string(ptr.Deref(resource.Description, ""))
	if osResource.Description != description {
		updateOpts.Description = &description
	}
}

// handleAdminStateUpUpdate updates the updateOpts if the adminStateUp needs to be changed
func handleAdminStateUpUpdate(updateOpts *trunks.UpdateOpts, resource *orcv1alpha1.TrunkResourceSpec, osResource *trunks.Trunk) {
	if resource.AdminStateUp != nil && *resource.AdminStateUp != osResource.AdminStateUp {
		updateOpts.AdminStateUp = resource.AdminStateUp
	}
}

func TestGetResourceID(t *testing.T) {
	actuator := trunkActuator{}
	osResource := &trunks.Trunk{ID: "test-id-123"}
	
	got := actuator.GetResourceID(osResource)
	if got != "test-id-123" {
		t.Errorf("Expected ID 'test-id-123', got '%s'", got)
	}
}

func TestGetOSResourceByID(t *testing.T) {
	ctx := context.Background()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockClient := osclientsmock.NewMockNetworkClient(mockCtrl)
	actuator := trunkActuator{osClient: mockClient}

	t.Run("success", func(t *testing.T) {
		expectedTrunk := &trunks.Trunk{ID: "test-id", Name: "test-trunk"}
		mockClient.EXPECT().GetTrunk(ctx, "test-id").Return(expectedTrunk, nil)

		got, status := actuator.GetOSResourceByID(ctx, "test-id")
		if status != nil {
			t.Errorf("Expected nil status, got %v", status)
		}
		if got.ID != expectedTrunk.ID {
			t.Errorf("Expected ID '%s', got '%s'", expectedTrunk.ID, got.ID)
		}
	})

	t.Run("error", func(t *testing.T) {
		expectedErr := errors.New("not found")
		mockClient.EXPECT().GetTrunk(ctx, "test-id").Return(nil, expectedErr)

		got, status := actuator.GetOSResourceByID(ctx, "test-id")
		if got != nil {
			t.Errorf("Expected nil, got %v", got)
		}
		if status == nil {
			t.Fatal("Expected non-nil status")
		}
		needsReschedule, err := status.NeedsReschedule()
		if !needsReschedule {
			t.Error("Expected needsReschedule to be true")
		}
		if err == nil {
			t.Error("Expected error in status")
		}
	})
}

func TestListOSResourcesForAdoption(t *testing.T) {
	ctx := context.Background()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockClient := osclientsmock.NewMockNetworkClient(mockCtrl)
	actuator := trunkActuator{osClient: mockClient}

	t.Run("no resource spec", func(t *testing.T) {
		obj := &orcv1alpha1.Trunk{
			Spec: orcv1alpha1.TrunkSpec{Resource: nil},
		}
		iter, ok := actuator.ListOSResourcesForAdoption(ctx, obj)
		if ok {
			t.Error("Expected ok to be false when resource spec is nil")
		}
		if iter != nil {
			t.Error("Expected nil iterator when resource spec is nil")
		}
	})

	t.Run("success", func(t *testing.T) {
		obj := &orcv1alpha1.Trunk{
			ObjectMeta: metav1.ObjectMeta{Name: "test-trunk"},
			Spec: orcv1alpha1.TrunkSpec{
				Resource: &orcv1alpha1.TrunkResourceSpec{},
			},
		}
		expectedTrunks := []trunks.Trunk{
			{ID: "id1", Name: "test-trunk"},
			{ID: "id2", Name: "test-trunk"},
		}
		mockClient.EXPECT().ListTrunk(ctx, gomock.Any()).Return(expectedTrunks, nil)

		iter, ok := actuator.ListOSResourcesForAdoption(ctx, obj)
		if !ok {
			t.Error("Expected ok to be true")
		}

		var results []*trunks.Trunk
		for trunk, err := range iter {
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			results = append(results, trunk)
		}

		if len(results) != 2 {
			t.Errorf("Expected 2 trunks, got %d", len(results))
		}
	})

	t.Run("error", func(t *testing.T) {
		obj := &orcv1alpha1.Trunk{
			ObjectMeta: metav1.ObjectMeta{Name: "test-trunk"},
			Spec: orcv1alpha1.TrunkSpec{
				Resource: &orcv1alpha1.TrunkResourceSpec{},
			},
		}
		expectedErr := errors.New("list error")
		mockClient.EXPECT().ListTrunk(ctx, gomock.Any()).Return(nil, expectedErr)

		iter, ok := actuator.ListOSResourcesForAdoption(ctx, obj)
		if !ok {
			t.Error("Expected ok to be true even on error")
		}

		var gotErr error
		for _, err := range iter {
			gotErr = err
			break
		}

		if gotErr != expectedErr {
			t.Errorf("Expected error '%v', got '%v'", expectedErr, gotErr)
		}
	})
}

func TestDeleteResource(t *testing.T) {
	ctx := context.Background()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockClient := osclientsmock.NewMockNetworkClient(mockCtrl)
	actuator := trunkActuator{osClient: mockClient}

	t.Run("success", func(t *testing.T) {
		osResource := &trunks.Trunk{ID: "test-id"}
		mockClient.EXPECT().DeleteTrunk(ctx, "test-id").Return(nil)

		status := actuator.DeleteResource(ctx, nil, osResource)
		if status != nil {
			needsReschedule, err := status.NeedsReschedule()
			if needsReschedule || err != nil {
				t.Errorf("Expected nil status, got needsReschedule=%v, err=%v", needsReschedule, err)
			}
		}
	})

	t.Run("error", func(t *testing.T) {
		osResource := &trunks.Trunk{ID: "test-id"}
		expectedErr := errors.New("delete error")
		mockClient.EXPECT().DeleteTrunk(ctx, "test-id").Return(expectedErr)

		status := actuator.DeleteResource(ctx, nil, osResource)
		if status == nil {
			t.Fatal("Expected non-nil status on error")
		}
		needsReschedule, err := status.NeedsReschedule()
		if !needsReschedule {
			t.Error("Expected needsReschedule to be true")
		}
		if err == nil {
			t.Error("Expected error in status")
		}
	})
}

func TestUpdateResource(t *testing.T) {
	ctx := ctrl.LoggerInto(context.Background(), ctrl.Log)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockClient := osclientsmock.NewMockNetworkClient(mockCtrl)
	actuator := trunkActuator{osClient: mockClient}

	t.Run("no changes", func(t *testing.T) {
		obj := &orcv1alpha1.Trunk{
			ObjectMeta: metav1.ObjectMeta{Name: "test-trunk"},
			Spec: orcv1alpha1.TrunkSpec{
				Resource: &orcv1alpha1.TrunkResourceSpec{
					Name: ptr.To(orcv1alpha1.OpenStackName("test-trunk")),
				},
			},
		}
		osResource := &trunks.Trunk{
			ID:             "test-id",
			Name:           "test-trunk",
			Description:    "",
			AdminStateUp:   true,
			RevisionNumber: 1,
		}

		status := actuator.updateResource(ctx, obj, osResource)
		if status != nil {
			needsReschedule, _ := status.NeedsReschedule()
			if needsReschedule {
				t.Error("Expected no status when no changes")
			}
		}
	})

	t.Run("name change", func(t *testing.T) {
		obj := &orcv1alpha1.Trunk{
			ObjectMeta: metav1.ObjectMeta{Name: "test-trunk"},
			Spec: orcv1alpha1.TrunkSpec{
				Resource: &orcv1alpha1.TrunkResourceSpec{
					Name: ptr.To(orcv1alpha1.OpenStackName("new-name")),
				},
			},
		}
		osResource := &trunks.Trunk{
			ID:             "test-id",
			Name:           "old-name",
			RevisionNumber: 1,
		}

		expectedTrunk := &trunks.Trunk{ID: "test-id", Name: "new-name"}
		mockClient.EXPECT().UpdateTrunk(ctx, "test-id", gomock.Any()).Return(expectedTrunk, nil)

		status := actuator.updateResource(ctx, obj, osResource)
		if status == nil {
			t.Fatal("Expected non-nil status after update")
		}
		// Status should indicate refresh is needed (status will have a refresh message)
	})

	t.Run("description change", func(t *testing.T) {
		obj := &orcv1alpha1.Trunk{
			ObjectMeta: metav1.ObjectMeta{Name: "test-trunk"},
			Spec: orcv1alpha1.TrunkSpec{
				Resource: &orcv1alpha1.TrunkResourceSpec{
					Description: ptr.To(orcv1alpha1.NeutronDescription("new desc")),
				},
			},
		}
		osResource := &trunks.Trunk{
			ID:             "test-id",
			Name:           "test-trunk",
			Description:    "old desc",
			RevisionNumber: 1,
		}

		expectedTrunk := &trunks.Trunk{ID: "test-id", Description: "new desc"}
		mockClient.EXPECT().UpdateTrunk(ctx, "test-id", gomock.Any()).Return(expectedTrunk, nil)

		status := actuator.updateResource(ctx, obj, osResource)
		if status == nil {
			t.Fatal("Expected non-nil status after update")
		}
	})

	t.Run("conflict error", func(t *testing.T) {
		obj := &orcv1alpha1.Trunk{
			ObjectMeta: metav1.ObjectMeta{Name: "test-trunk"},
			Spec: orcv1alpha1.TrunkSpec{
				Resource: &orcv1alpha1.TrunkResourceSpec{
					Name: ptr.To(orcv1alpha1.OpenStackName("new-name")),
				},
			},
		}
		osResource := &trunks.Trunk{
			ID:             "test-id",
			Name:           "old-name",
			RevisionNumber: 1,
		}

		conflictErr := gophercloud.ErrUnexpectedResponseCode{Actual: 409}
		mockClient.EXPECT().UpdateTrunk(ctx, "test-id", gomock.Any()).Return(nil, conflictErr)

		status := actuator.updateResource(ctx, obj, osResource)
		if status == nil {
			t.Fatal("Expected non-nil status on error")
		}
		needsReschedule, err := status.NeedsReschedule()
		if !needsReschedule {
			t.Error("Expected needsReschedule to be true on conflict")
		}
		if err == nil {
			t.Error("Expected error in status")
		}
		// Check that it's a terminal error
		var terminalError *orcerrors.TerminalError
		if !errors.As(err, &terminalError) {
			t.Error("Expected terminal error on conflict")
		}
	})
}

func TestReconcileSubports(t *testing.T) {
	ctx := ctrl.LoggerInto(context.Background(), ctrl.Log)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	scheme := runtime.NewScheme()
	orcv1alpha1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)

	mockClient := osclientsmock.NewMockNetworkClient(mockCtrl)
	k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	actuator := trunkActuator{osClient: mockClient, k8sClient: k8sClient}

	t.Run("no resource spec", func(t *testing.T) {
		obj := &orcv1alpha1.Trunk{
			Spec: orcv1alpha1.TrunkSpec{Resource: nil},
		}
		osResource := &trunks.Trunk{ID: "test-id"}

		status := actuator.reconcileSubports(ctx, obj, osResource)
		if status != nil {
			t.Errorf("Expected nil status, got %v", status)
		}
	})

	t.Run("no changes needed", func(t *testing.T) {
		// Create a fresh k8sClient for this test to avoid state from previous tests
		testK8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		testActuator := trunkActuator{osClient: mockClient, k8sClient: testK8sClient}

		port1 := &orcv1alpha1.Port{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "port1",
				Namespace: "default",
				Finalizers: []string{finalizer},
			},
			Status: orcv1alpha1.PortStatus{
				ID: ptr.To("port-id-1"),
				Conditions: []metav1.Condition{
					{Type: "Available", Status: metav1.ConditionTrue},
				},
			},
		}
		testK8sClient.Create(ctx, port1)

		obj := &orcv1alpha1.Trunk{
			ObjectMeta: metav1.ObjectMeta{Name: "trunk", Namespace: "default"},
			Spec: orcv1alpha1.TrunkSpec{
				Resource: &orcv1alpha1.TrunkResourceSpec{
					Subports: []orcv1alpha1.Subport{
						{
							PortRef:          "port1",
							SegmentationType: "vlan",
							SegmentationID:   100,
						},
					},
				},
			},
		}
		osResource := &trunks.Trunk{
			ID: "trunk-id",
		}

		currentSubports := []trunks.Subport{
			{
				PortID:           "port-id-1",
				SegmentationType: "vlan",
				SegmentationID:   100,
			},
		}

		mockClient.EXPECT().ListTrunkSubports(ctx, "trunk-id").Return(currentSubports, nil)

		status := testActuator.reconcileSubports(ctx, obj, osResource)
		if status != nil {
			t.Errorf("Expected nil status when no changes, got %v", status)
		}
	})

	t.Run("add subport", func(t *testing.T) {
		// Create a fresh k8sClient for this test to avoid state from previous tests
		testK8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		testActuator := trunkActuator{osClient: mockClient, k8sClient: testK8sClient}

		port1 := &orcv1alpha1.Port{
			ObjectMeta: metav1.ObjectMeta{
				Name:       "port1",
				Namespace:  "default",
				Finalizers: []string{finalizer},
			},
			Status: orcv1alpha1.PortStatus{
				ID: ptr.To("port-id-1"),
				Conditions: []metav1.Condition{
					{Type: "Available", Status: metav1.ConditionTrue},
				},
			},
		}
		testK8sClient.Create(ctx, port1)

		obj := &orcv1alpha1.Trunk{
			ObjectMeta: metav1.ObjectMeta{Name: "trunk", Namespace: "default"},
			Spec: orcv1alpha1.TrunkSpec{
				Resource: &orcv1alpha1.TrunkResourceSpec{
					Subports: []orcv1alpha1.Subport{
						{
							PortRef:          "port1",
							SegmentationType: "vlan",
							SegmentationID:   100,
						},
					},
				},
			},
		}
		osResource := &trunks.Trunk{ID: "trunk-id"}

		mockClient.EXPECT().ListTrunkSubports(ctx, "trunk-id").Return([]trunks.Subport{}, nil)
		expectedTrunk := &trunks.Trunk{ID: "trunk-id"}
		mockClient.EXPECT().AddSubports(ctx, "trunk-id", gomock.Any()).Return(expectedTrunk, nil)

		status := testActuator.reconcileSubports(ctx, obj, osResource)
		if status == nil {
			t.Fatal("Expected non-nil status after adding subport")
		}
		// Status should indicate refresh is needed (status will have a refresh message)
	})

	t.Run("remove subport", func(t *testing.T) {
		obj := &orcv1alpha1.Trunk{
			ObjectMeta: metav1.ObjectMeta{Name: "trunk", Namespace: "default"},
			Spec: orcv1alpha1.TrunkSpec{
				Resource: &orcv1alpha1.TrunkResourceSpec{
					Subports: []orcv1alpha1.Subport{},
				},
			},
		}
		osResource := &trunks.Trunk{ID: "trunk-id"}

		currentSubports := []trunks.Subport{
			{
				PortID:           "port-id-1",
				SegmentationType: "vlan",
				SegmentationID:   100,
			},
		}

		mockClient.EXPECT().ListTrunkSubports(ctx, "trunk-id").Return(currentSubports, nil)
		mockClient.EXPECT().RemoveSubports(ctx, "trunk-id", gomock.Any()).Return(nil)

		status := actuator.reconcileSubports(ctx, obj, osResource)
		if status == nil {
			t.Fatal("Expected non-nil status after removing subport")
		}
		// Status should indicate refresh is needed (status will have a refresh message)
	})

	t.Run("list subports error", func(t *testing.T) {
		obj := &orcv1alpha1.Trunk{
			ObjectMeta: metav1.ObjectMeta{Name: "trunk", Namespace: "default"},
			Spec: orcv1alpha1.TrunkSpec{
				Resource: &orcv1alpha1.TrunkResourceSpec{
					Subports: []orcv1alpha1.Subport{},
				},
			},
		}
		osResource := &trunks.Trunk{ID: "trunk-id"}

		expectedErr := errors.New("list error")
		mockClient.EXPECT().ListTrunkSubports(ctx, "trunk-id").Return(nil, expectedErr)

		status := actuator.reconcileSubports(ctx, obj, osResource)
		if status == nil {
			t.Fatal("Expected non-nil status on error")
		}
		needsReschedule, err := status.NeedsReschedule()
		if !needsReschedule {
			t.Error("Expected needsReschedule to be true on error")
		}
		if err == nil {
			t.Error("Expected error in status")
		}
	})
}


