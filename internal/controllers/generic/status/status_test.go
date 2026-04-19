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
	"errors"
	"testing"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/progress"
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
