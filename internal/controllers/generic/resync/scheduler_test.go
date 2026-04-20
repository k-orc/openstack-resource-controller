/*
Copyright 2024 The ORC Authors.

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

package resync

import (
	"fmt"
	"testing"
	"time"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/progress"
	orcerrors "github.com/k-orc/openstack-resource-controller/v2/internal/util/errors"
)

// TestCalculateJitteredDuration_Range verifies that the returned duration is
// always within [base*1.0, base*1.2) across many calls (acceptance criterion).
func TestCalculateJitteredDuration_Range(t *testing.T) {
	t.Parallel()

	const (
		base    = 10 * time.Minute
		samples = 1000
	)

	lo := time.Duration(float64(base) * 1.0)
	hi := time.Duration(float64(base) * 1.2)

	for i := range samples {
		d := CalculateJitteredDuration(base)
		if d < lo || d > hi {
			t.Errorf("sample %d: CalculateJitteredDuration(%v) = %v, want in [%v, %v]", i, base, d, lo, hi)
		}
	}
}

// TestCalculateJitteredDuration_Uniformity verifies that the jitter
// distribution is statistically uniform by checking that all 10 buckets across
// [base*1.0, base*1.2) are populated with at least 1/20th of the expected
// frequency (very conservative check to avoid flakiness while still catching
// obvious bias).
func TestCalculateJitteredDuration_Uniformity(t *testing.T) {
	t.Parallel()

	const (
		base    = time.Hour
		samples = 10000
		buckets = 10
	)

	lo := float64(base) * 1.0
	hi := float64(base) * (1 + 2*jitterFactor)
	width := (hi - lo) / buckets

	counts := make([]int, buckets)
	for range samples {
		d := CalculateJitteredDuration(base)
		idx := int((float64(d) - lo) / width)
		// Clamp to handle floating-point edge at the top of the range.
		if idx >= buckets {
			idx = buckets - 1
		}
		if idx < 0 {
			idx = 0
		}
		counts[idx]++
	}

	// Each bucket should receive roughly samples/buckets hits. Require at
	// least 1/3 of the expected count to avoid flakiness while catching bias.
	minExpected := (samples / buckets) / 3
	for i, c := range counts {
		if c < minExpected {
			t.Errorf("bucket %d: count %d is below minimum expected %d (distribution is not uniform)", i, c, minExpected)
		}
	}
}

// TestCalculateJitteredDuration_Independence verifies that multiple resources
// receive independent jitter values (TS-011): calling the function twice with
// the same base should produce different values in the vast majority of cases.
func TestCalculateJitteredDuration_Independence(t *testing.T) {
	t.Parallel()

	const (
		base    = time.Hour
		samples = 100
	)

	unique := make(map[time.Duration]struct{}, samples)
	for range samples {
		d := CalculateJitteredDuration(base)
		unique[d] = struct{}{}
	}

	// Expect nearly all samples to be distinct. Allow for at most 5%
	// collisions (extremely conservative; in practice collisions are
	// essentially impossible with nanosecond precision).
	minUnique := samples * 95 / 100
	if len(unique) < minUnique {
		t.Errorf("CalculateJitteredDuration produced only %d unique values out of %d samples (expected >= %d); values may not be independent", len(unique), samples, minUnique)
	}
}

// TestCalculateJitteredDuration_ZeroBase verifies behaviour with a zero base.
func TestCalculateJitteredDuration_ZeroBase(t *testing.T) {
	t.Parallel()

	if d := CalculateJitteredDuration(0); d != 0 {
		t.Errorf("CalculateJitteredDuration(0) = %v, want 0", d)
	}
}

// TestShouldScheduleResync covers all documented return-false conditions and
// the happy path.
func TestShouldScheduleResync(t *testing.T) {
	t.Parallel()

	terminalErr := orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "bad config")
	transientErr := fmt.Errorf("transient error")

	tests := []struct {
		name            string
		resyncPeriod    time.Duration
		reconcileStatus progress.ReconcileStatus
		want            bool
	}{
		{
			// TS-007: resync disabled globally
			name:            "resyncPeriod 0, nil status, returns false",
			resyncPeriod:    0,
			reconcileStatus: nil,
			want:            false,
		},
		{
			// resyncPeriod negative: treated as disabled
			name:            "resyncPeriod negative, nil status, returns false",
			resyncPeriod:    -time.Second,
			reconcileStatus: nil,
			want:            false,
		},
		{
			// TS-008: terminal error → no resync
			name:            "terminal error in status, returns false",
			resyncPeriod:    time.Hour,
			reconcileStatus: progress.WrapError(terminalErr),
			want:            false,
		},
		{
			// TS-012: requeue already pending → resync is redundant
			name:            "requeue already pending in status, returns false",
			resyncPeriod:    time.Hour,
			reconcileStatus: progress.NewReconcileStatus().WithRequeue(5 * time.Second),
			want:            false,
		},
		{
			// Happy path: positive period, no terminal error, no pending requeue
			name:            "positive period, nil status, returns true",
			resyncPeriod:    time.Hour,
			reconcileStatus: nil,
			want:            true,
		},
		{
			// Happy path: transient (non-terminal) error should not suppress resync
			name:            "transient error in status, returns true",
			resyncPeriod:    time.Hour,
			reconcileStatus: progress.WrapError(transientErr),
			want:            true,
		},
		{
			// Happy path: progress message with no requeue should not suppress resync
			name:            "progress message only, no requeue, returns true",
			resyncPeriod:    time.Hour,
			reconcileStatus: progress.NewReconcileStatus().WithProgressMessage("waiting for dependency"),
			want:            true,
		},
		{
			// Terminal error takes precedence even when period is positive
			name:            "terminal error with progress message, returns false",
			resyncPeriod:    time.Hour,
			reconcileStatus: progress.WrapError(terminalErr).WithProgressMessage("some message"),
			want:            false,
		},
		{
			// Requeue takes precedence when period is positive
			name:            "requeue pending with progress message, returns false",
			resyncPeriod:    30 * time.Minute,
			reconcileStatus: progress.NewReconcileStatus().WithRequeue(10 * time.Second).WithProgressMessage("waiting"),
			want:            false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ShouldScheduleResync(tt.resyncPeriod, tt.reconcileStatus)
			if got != tt.want {
				t.Errorf("ShouldScheduleResync(%v, ...) = %v, want %v", tt.resyncPeriod, got, tt.want)
			}
		})
	}
}
