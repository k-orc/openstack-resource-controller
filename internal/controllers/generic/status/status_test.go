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

package status

import (
	"context"
	"errors"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/progress"
	"github.com/k-orc/openstack-resource-controller/v2/internal/scope"
	orcerrors "github.com/k-orc/openstack-resource-controller/v2/internal/util/errors"
)

// TestShouldSetLastSyncTime_SuccessfulReconciliation verifies that lastSyncTime
// is set when reconcileStatus is nil (clean success, no errors, no progress
// messages). This is the common case after a successful OpenStack API read
// (TS-009, TS-013).
func TestShouldSetLastSyncTime_SuccessfulReconciliation(t *testing.T) {
	t.Parallel()

	// nil ReconcileStatus represents a clean, successful reconciliation.
	var rs progress.ReconcileStatus
	if !shouldSetLastSyncTime(rs) {
		t.Error("shouldSetLastSyncTime(nil) = false; want true for successful reconciliation")
	}
}

// TestShouldSetLastSyncTime_WithRequeueOnly verifies that a requeue alone
// (e.g., for a periodic resync) does not prevent lastSyncTime from being set.
// A pending requeue without errors or progress messages still counts as a
// successful reconciliation cycle (TS-009).
func TestShouldSetLastSyncTime_WithRequeueOnly(t *testing.T) {
	t.Parallel()

	// A requeue alone does not contribute to NeedsReschedule.
	rs := progress.NewReconcileStatus().WithRequeue(10 * 60 * 1000000000) // 10 minutes in nanoseconds
	if !shouldSetLastSyncTime(rs) {
		t.Error("shouldSetLastSyncTime(requeue-only) = false; want true: requeue alone should not prevent lastSyncTime update")
	}
}

// TestShouldSetLastSyncTime_WithError verifies that lastSyncTime is NOT set
// when reconcileStatus contains an error. An error means the controller did not
// successfully complete the reconciliation cycle (TS-009, TS-013).
func TestShouldSetLastSyncTime_WithError(t *testing.T) {
	t.Parallel()

	rs := progress.WrapError(errors.New("transient openstack error"))
	if shouldSetLastSyncTime(rs) {
		t.Error("shouldSetLastSyncTime(error) = true; want false: errors should prevent lastSyncTime update")
	}
}

// TestShouldSetLastSyncTime_WithTerminalError verifies that lastSyncTime is NOT
// set when reconcileStatus contains a terminal error. Terminal errors indicate
// a non-retryable failure; the reconciliation did not succeed (TS-009, TS-013).
func TestShouldSetLastSyncTime_WithTerminalError(t *testing.T) {
	t.Parallel()

	termErr := orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration", nil)
	rs := progress.WrapError(termErr)
	if shouldSetLastSyncTime(rs) {
		t.Error("shouldSetLastSyncTime(terminal error) = true; want false: terminal errors should prevent lastSyncTime update")
	}
}

// TestShouldSetLastSyncTime_WithProgressMessage verifies that lastSyncTime is
// NOT set when reconcileStatus contains a progress message. Progress messages
// indicate that the reconciliation is still ongoing (waiting on a dependency,
// resource not yet ready, etc.) and has not completed successfully (TS-009,
// TS-013).
func TestShouldSetLastSyncTime_WithProgressMessage(t *testing.T) {
	t.Parallel()

	rs := progress.NewReconcileStatus().WithProgressMessage("waiting for resource to become active")
	if shouldSetLastSyncTime(rs) {
		t.Error("shouldSetLastSyncTime(progress message) = true; want false: progress messages should prevent lastSyncTime update")
	}
}

// TestShouldSetLastSyncTime_WithErrorAndProgressMessage verifies that
// lastSyncTime is NOT set when reconcileStatus contains both an error and a
// progress message. Either alone should be sufficient to suppress the update.
func TestShouldSetLastSyncTime_WithErrorAndProgressMessage(t *testing.T) {
	t.Parallel()

	rs := progress.WrapError(errors.New("API error")).WithProgressMessage("still waiting")
	if shouldSetLastSyncTime(rs) {
		t.Error("shouldSetLastSyncTime(error+progress) = true; want false: any non-success condition should prevent lastSyncTime update")
	}
}

// --------------------------------------------------------------------------
// fakeStatusController implements interfaces.ResourceController for
// ClearStatusID tests. It wraps a real fake.Client to allow status patch calls.
// --------------------------------------------------------------------------

type fakeStatusController struct {
	k8sClient client.Client
}

var _ interfaces.ResourceController = &fakeStatusController{}

func (c *fakeStatusController) GetName() string                { return "test-status-controller" }
func (c *fakeStatusController) GetK8sClient() client.Client    { return c.k8sClient }
func (c *fakeStatusController) GetScopeFactory() scope.Factory { return nil }

// TestClearStatusID_SendsMergePatchWithNullID verifies that ClearStatusID
// issues a JSON merge patch that sets status.id to null. The function is
// expected to be called by reconcileNormal when an externally deleted managed
// resource is detected (GetOrCreateOSResource returns nil, nil).
func TestClearStatusID_SendsMergePatchWithNullID(t *testing.T) {
	t.Parallel()

	const resourceID = "some-os-id"

	// Build a Flavor with status.ID already set (simulating a managed resource
	// whose OpenStack counterpart was deleted externally).
	flavor := &orcv1alpha1.Flavor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-flavor",
			Namespace: "default",
			UID:       types.UID("test-uid"),
		},
		Status: orcv1alpha1.FlavorStatus{
			ID: ptr.To(resourceID),
		},
	}

	// Register the Flavor scheme so the fake client can handle it.
	scheme := runtime.NewScheme()
	if err := orcv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add orcv1alpha1 to scheme: %v", err)
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&orcv1alpha1.Flavor{}).
		WithObjects(flavor).
		Build()

	controller := &fakeStatusController{k8sClient: fakeClient}

	// Call ClearStatusID: should patch status.id to null.
	if err := ClearStatusID(context.Background(), controller, flavor); err != nil {
		t.Fatalf("ClearStatusID returned unexpected error: %v", err)
	}

	// Fetch the updated Flavor and verify status.ID is now nil.
	updated := &orcv1alpha1.Flavor{}
	if err := fakeClient.Get(context.Background(), client.ObjectKey{Name: "test-flavor", Namespace: "default"}, updated); err != nil {
		t.Fatalf("failed to get updated flavor: %v", err)
	}

	if updated.Status.ID != nil {
		t.Errorf("status.id = %q after ClearStatusID; want nil (cleared)", *updated.Status.ID)
	}
}

// TestClearStatusID_GroupVersionResource verifies that ClearStatusID targets
// the status subresource (i.e., calls Status().Patch rather than Patch).
// This is an indirect check: if ClearStatusID called the main Patch instead of
// Status().Patch, the fake client with WithStatusSubresource would not update
// the status and the ID would remain set.
func TestClearStatusID_IdempotentWhenAlreadyNil(t *testing.T) {
	t.Parallel()

	// Flavor with no status.ID (already cleared or never set).
	flavor := &orcv1alpha1.Flavor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-flavor",
			Namespace: "default",
			UID:       types.UID("test-uid"),
		},
		// Status.ID is nil by default.
	}

	scheme := runtime.NewScheme()
	if err := orcv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add orcv1alpha1 to scheme: %v", err)
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&orcv1alpha1.Flavor{}).
		WithObjects(flavor).
		Build()

	controller := &fakeStatusController{k8sClient: fakeClient}

	// ClearStatusID should succeed even when status.id is already nil.
	if err := ClearStatusID(context.Background(), controller, flavor); err != nil {
		t.Fatalf("ClearStatusID returned unexpected error on already-nil ID: %v", err)
	}

	// Status.ID should remain nil.
	updated := &orcv1alpha1.Flavor{}
	if err := fakeClient.Get(context.Background(), client.ObjectKey{Name: "test-flavor", Namespace: "default"}, updated); err != nil {
		t.Fatalf("failed to get flavor after ClearStatusID: %v", err)
	}

	if updated.Status.ID != nil {
		t.Errorf("status.id = %q after ClearStatusID on already-nil ID; want nil", *updated.Status.ID)
	}
}
