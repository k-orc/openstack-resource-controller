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

package reconciler

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/progress"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/resync"
	orcerrors "github.com/k-orc/openstack-resource-controller/v2/internal/util/errors"
)

// makeObj creates a Flavor object with the given generation and conditions,
// satisfying orcv1alpha1.ObjectWithConditions.
func makeObj(generation int64, conditions []metav1.Condition) orcv1alpha1.ObjectWithConditions {
	f := &orcv1alpha1.Flavor{}
	f.Generation = generation
	f.Status.Conditions = conditions
	return f
}

// makeProgressingCondition returns a Progressing condition with the given
// status and observedGeneration.
func makeProgressingCondition(status metav1.ConditionStatus, observedGeneration int64) metav1.Condition { //nolint:unparam
	return metav1.Condition{
		Type:               orcv1alpha1.ConditionProgressing,
		Status:             status,
		ObservedGeneration: observedGeneration,
		Reason:             "Test",
	}
}

// agoPtr returns a *metav1.Time that is d in the past.
func agoPtr(d time.Duration) *metav1.Time {
	t := metav1.NewTime(time.Now().Add(-d))
	return &t
}

// nowPtr returns a *metav1.Time set to approximately now.
func nowPtr() *metav1.Time {
	t := metav1.Now()
	return &t
}

func TestShouldReconcile_ConditionBased(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		generation   int64
		conditions   []metav1.Condition
		lastSyncTime *metav1.Time
		resyncPeriod time.Duration
		want         bool
	}{
		{
			name:       "no conditions: should reconcile",
			generation: 1,
			conditions: nil,
			want:       true,
		},
		{
			name:       "Progressing=True up-to-date: should reconcile",
			generation: 1,
			conditions: []metav1.Condition{
				makeProgressingCondition(metav1.ConditionTrue, 1),
			},
			want: true,
		},
		{
			name:       "Progressing=False up-to-date resync disabled: should not reconcile",
			generation: 1,
			conditions: []metav1.Condition{
				makeProgressingCondition(metav1.ConditionFalse, 1),
			},
			resyncPeriod: 0,
			want:         false,
		},
		{
			name:       "Progressing=False stale generation: should reconcile",
			generation: 2,
			conditions: []metav1.Condition{
				makeProgressingCondition(metav1.ConditionFalse, 1),
			},
			want: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			obj := makeObj(tc.generation, tc.conditions)
			got := shouldReconcile(obj, tc.lastSyncTime, tc.resyncPeriod)
			if got != tc.want {
				t.Errorf("shouldReconcile() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestShouldReconcile_ResyncDisabled(t *testing.T) {
	t.Parallel()

	// An up-to-date Progressing=False condition prevents reconciliation when
	// resync is disabled (resyncPeriod <= 0).
	obj := makeObj(1, []metav1.Condition{
		makeProgressingCondition(metav1.ConditionFalse, 1),
	})

	tests := []struct {
		name         string
		resyncPeriod time.Duration
		lastSyncTime *metav1.Time
	}{
		{
			name:         "resyncPeriod=0 nil lastSyncTime",
			resyncPeriod: 0,
			lastSyncTime: nil,
		},
		{
			name:         "resyncPeriod=0 old lastSyncTime",
			resyncPeriod: 0,
			lastSyncTime: agoPtr(24 * time.Hour),
		},
		{
			name:         "negative resyncPeriod",
			resyncPeriod: -1 * time.Minute,
			lastSyncTime: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := shouldReconcile(obj, tc.lastSyncTime, tc.resyncPeriod)
			if got {
				t.Errorf("shouldReconcile() = true; want false when resync disabled (resyncPeriod=%v)", tc.resyncPeriod)
			}
		})
	}
}

func TestShouldReconcile_ResyncEnabled_NilLastSyncTime(t *testing.T) {
	t.Parallel()

	// When resyncPeriod > 0 and lastSyncTime is nil (never synced), reconcile
	// immediately (TS-015: persisted time is absent → treat as overdue).
	obj := makeObj(1, []metav1.Condition{
		makeProgressingCondition(metav1.ConditionFalse, 1),
	})

	got := shouldReconcile(obj, nil, 10*time.Minute)
	if !got {
		t.Error("shouldReconcile() = false; want true when lastSyncTime is nil and resyncPeriod > 0")
	}
}

func TestShouldReconcile_ResyncEnabled_PeriodElapsed(t *testing.T) {
	t.Parallel()

	// When time.Since(lastSyncTime) >= resyncPeriod, a resync is due.
	obj := makeObj(1, []metav1.Condition{
		makeProgressingCondition(metav1.ConditionFalse, 1),
	})

	// Last synced 20 minutes ago, period is 10 minutes.
	got := shouldReconcile(obj, agoPtr(20*time.Minute), 10*time.Minute)
	if !got {
		t.Error("shouldReconcile() = false; want true when time.Since(lastSyncTime) >= resyncPeriod")
	}
}

func TestShouldReconcile_ResyncEnabled_PeriodNotElapsed(t *testing.T) {
	t.Parallel()

	// When time.Since(lastSyncTime) < resyncPeriod, no resync is due.
	obj := makeObj(1, []metav1.Condition{
		makeProgressingCondition(metav1.ConditionFalse, 1),
	})

	// Last synced 2 minutes ago, period is 10 minutes.
	got := shouldReconcile(obj, agoPtr(2*time.Minute), 10*time.Minute)
	if got {
		t.Error("shouldReconcile() = true; want false when time.Since(lastSyncTime) < resyncPeriod")
	}
}

func TestShouldReconcile_ResyncEnabled_JustPastPeriod(t *testing.T) {
	t.Parallel()

	// Boundary condition: just past the period should trigger resync (>= semantics).
	obj := makeObj(1, []metav1.Condition{
		makeProgressingCondition(metav1.ConditionFalse, 1),
	})

	resyncPeriod := 10 * time.Minute
	// Add a small extra to ensure we're past the boundary even accounting for
	// time elapsed during test execution.
	lastSyncTime := agoPtr(resyncPeriod + 100*time.Millisecond)

	got := shouldReconcile(obj, lastSyncTime, resyncPeriod)
	if !got {
		t.Error("shouldReconcile() = false; want true when time.Since(lastSyncTime) is just past resyncPeriod")
	}
}

func TestShouldReconcile_ResyncEnabled_ProgressingTrue_IgnoresResyncNotElapsed(t *testing.T) {
	t.Parallel()

	// Progressing=True always triggers reconciliation even if resync period has
	// not elapsed yet (condition-based logic takes priority for positive cases).
	obj := makeObj(1, []metav1.Condition{
		makeProgressingCondition(metav1.ConditionTrue, 1),
	})

	// lastSyncTime is very recent, so resync would say "false".
	// But Progressing=True means we must reconcile anyway.
	got := shouldReconcile(obj, nowPtr(), time.Hour)
	if !got {
		t.Error("shouldReconcile() = false; want true when Progressing=True regardless of resync period")
	}
}

func TestShouldReconcile_ResyncEnabled_ControllerRestart_PersistsLastSyncTime(t *testing.T) {
	t.Parallel()

	// Thundering-herd prevention (TS-015): after a controller restart,
	// lastSyncTime is read from the persisted Kubernetes status. If the
	// persisted time is recent, shouldReconcile should return false so the
	// controller does not immediately hammer OpenStack for all resources at once.
	obj := makeObj(1, []metav1.Condition{
		makeProgressingCondition(metav1.ConditionFalse, 1),
	})

	resyncPeriod := 30 * time.Minute
	// Simulated: last sync was 5 minutes ago (persisted from before restart).
	lastSyncTime := agoPtr(5 * time.Minute)

	got := shouldReconcile(obj, lastSyncTime, resyncPeriod)
	if got {
		t.Error("shouldReconcile() = true; want false: controller should respect persisted lastSyncTime after restart (TS-015)")
	}
}

func TestShouldReconcile_ExistingBehaviorUnchanged_ResyncPeriodZero(t *testing.T) {
	t.Parallel()

	// When resyncPeriod is 0 (disabled), shouldReconcile behaves exactly as it
	// did before the resync feature was added: only condition-based logic applies.
	tests := []struct {
		name       string
		generation int64
		conditions []metav1.Condition
		want       bool
	}{
		{
			name:       "no conditions",
			generation: 1,
			want:       true,
		},
		{
			name:       "progressing true",
			generation: 1,
			conditions: []metav1.Condition{makeProgressingCondition(metav1.ConditionTrue, 1)},
			want:       true,
		},
		{
			name:       "progressing false up-to-date",
			generation: 1,
			conditions: []metav1.Condition{makeProgressingCondition(metav1.ConditionFalse, 1)},
			want:       false,
		},
		{
			name:       "progressing false stale",
			generation: 2,
			conditions: []metav1.Condition{makeProgressingCondition(metav1.ConditionFalse, 1)},
			want:       true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			obj := makeObj(tc.generation, tc.conditions)
			// resyncPeriod=0 and nil lastSyncTime: pure condition-based behaviour.
			got := shouldReconcile(obj, nil, 0)
			if got != tc.want {
				t.Errorf("shouldReconcile() = %v, want %v (existing behaviour should be unchanged)", got, tc.want)
			}
		})
	}
}

// scheduleResyncRequeue simulates the resync scheduling logic added to the end
// of reconcileNormal:
//
//	if resync.ShouldScheduleResync(effectiveResyncPeriod, reconcileStatus) {
//	    reconcileStatus = reconcileStatus.WithRequeue(resync.CalculateJitteredDuration(effectiveResyncPeriod))
//	}
//
// This helper allows the following tests to verify the combined behaviour of
// ShouldScheduleResync and CalculateJitteredDuration without requiring a full
// Kubernetes environment.
func scheduleResyncRequeue(reconcileStatus progress.ReconcileStatus, period time.Duration) progress.ReconcileStatus {
	if resync.ShouldScheduleResync(period, reconcileStatus) {
		reconcileStatus = reconcileStatus.WithRequeue(resync.CalculateJitteredDuration(period))
	}
	return reconcileStatus
}

// TestResyncRequeue_ScheduledWhenPeriodPositive verifies that a resync requeue
// is added to a clean ReconcileStatus when resyncPeriod > 0 (TS-006).
func TestResyncRequeue_ScheduledWhenPeriodPositive(t *testing.T) {
	t.Parallel()

	const period = 10 * time.Minute

	// A clean (nil) ReconcileStatus represents a successful reconciliation with
	// no errors and no pending requeue.
	var rs progress.ReconcileStatus
	rs = scheduleResyncRequeue(rs, period)

	requeue := rs.GetRequeue()
	if requeue == 0 {
		t.Fatal("expected a non-zero requeue duration after resync scheduling; got 0")
	}

	// The requeue must be within the jitter range [period*0.9, period*1.1] (TS-006).
	lo := time.Duration(float64(period) * 0.9)
	hi := time.Duration(float64(period) * 1.1)
	if requeue < lo || requeue > hi {
		t.Errorf("resync requeue %v is outside jitter range [%v, %v]", requeue, lo, hi)
	}
}

// TestResyncRequeue_NotScheduledWhenPeriodZero verifies that no resync requeue
// is added when resyncPeriod is zero (disabled).
func TestResyncRequeue_NotScheduledWhenPeriodZero(t *testing.T) {
	t.Parallel()

	var rs progress.ReconcileStatus
	rs = scheduleResyncRequeue(rs, 0)

	if requeue := rs.GetRequeue(); requeue != 0 {
		t.Errorf("expected no requeue when resyncPeriod=0; got %v", requeue)
	}
}

// TestResyncRequeue_NotScheduledWhenPeriodNegative verifies that no resync
// requeue is added when resyncPeriod is negative (effectively disabled).
func TestResyncRequeue_NotScheduledWhenPeriodNegative(t *testing.T) {
	t.Parallel()

	var rs progress.ReconcileStatus
	rs = scheduleResyncRequeue(rs, -1*time.Minute)

	if requeue := rs.GetRequeue(); requeue != 0 {
		t.Errorf("expected no requeue when resyncPeriod<0; got %v", requeue)
	}
}

// TestResyncRequeue_NotScheduledWhenTerminalError verifies that no resync
// requeue is scheduled when the ReconcileStatus contains a terminal error
// (TS-008). Terminal errors indicate the resource is in a non-retryable state;
// resyncing would be pointless and wasteful.
func TestResyncRequeue_NotScheduledWhenTerminalError(t *testing.T) {
	t.Parallel()

	const period = 10 * time.Minute

	termErr := orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid config", nil)
	rs := progress.WrapError(termErr)
	rs = scheduleResyncRequeue(rs, period)

	if requeue := rs.GetRequeue(); requeue != 0 {
		t.Errorf("expected no resync requeue on terminal error; got %v", requeue)
	}
}

// TestResyncRequeue_NotScheduledWhenRequeueAlreadyPending verifies that no
// additional resync requeue is scheduled when one is already set (TS-012).
// This prevents redundant requeues when the reconciler is already waiting on
// an OpenStack event or dependency.
func TestResyncRequeue_NotScheduledWhenRequeueAlreadyPending(t *testing.T) {
	t.Parallel()

	const period = 10 * time.Minute
	const existingRequeue = 5 * time.Second

	// Simulate a reconcile status that already has a short requeue (e.g., waiting
	// for an OpenStack resource to become ready).
	rs := progress.NewReconcileStatus().WithRequeue(existingRequeue)
	rs = scheduleResyncRequeue(rs, period)

	// The existing requeue must be preserved unchanged; no extra requeue added.
	if requeue := rs.GetRequeue(); requeue != existingRequeue {
		t.Errorf("expected existing requeue %v to be preserved; got %v", existingRequeue, requeue)
	}
}

// TestResyncRequeue_JitterIsApplied verifies that multiple scheduling calls
// with the same period produce different requeue durations (jitter is random),
// and all values are within the expected ±10% range (TS-006).
func TestResyncRequeue_JitterIsApplied(t *testing.T) {
	t.Parallel()

	const samples = 200
	period := time.Hour

	lo := time.Duration(float64(period) * 0.9)
	hi := time.Duration(float64(period) * 1.1)

	unique := make(map[time.Duration]struct{}, samples)
	for i := range samples {
		var rs progress.ReconcileStatus
		rs = scheduleResyncRequeue(rs, period)
		d := rs.GetRequeue()
		if d < lo || d > hi {
			t.Errorf("sample %d: requeue %v outside jitter range [%v, %v]", i, d, lo, hi)
		}
		unique[d] = struct{}{}
	}

	// With 200 samples from a continuous distribution, we expect nearly all
	// values to be distinct. Require at least 90% uniqueness.
	minUnique := samples * 9 / 10
	if len(unique) < minUnique {
		t.Errorf("jitter appears non-random: only %d unique values out of %d samples (want >= %d)", len(unique), samples, minUnique)
	}
}

// TestResyncRequeue_RequeueTimingRange verifies the requeue timing over many
// samples remains within the ±10% jitter window, functioning as an integration
// check of the scheduling logic used in reconcileNormal (TS-006).
func TestResyncRequeue_RequeueTimingRange(t *testing.T) {
	t.Parallel()

	periods := []time.Duration{
		time.Minute,
		10 * time.Minute,
		time.Hour,
		24 * time.Hour,
	}

	for _, period := range periods {
		t.Run(period.String(), func(t *testing.T) {
			t.Parallel()

			lo := time.Duration(float64(period) * 0.9)
			hi := time.Duration(float64(period) * 1.1)

			for i := range 50 {
				var rs progress.ReconcileStatus
				rs = scheduleResyncRequeue(rs, period)
				d := rs.GetRequeue()
				if d < lo || d > hi {
					t.Errorf("sample %d: period=%v requeue=%v outside [%v, %v]",
						i, period, d, lo, hi)
				}
			}
		})
	}
}
