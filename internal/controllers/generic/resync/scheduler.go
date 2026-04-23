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
	"errors"
	"math/rand/v2"
	"time"

	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/progress"
	orcerrors "github.com/k-orc/openstack-resource-controller/v2/internal/util/errors"
)

const (
	// jitterFactor is the fraction added to the base duration as positive-only
	// jitter. A value of 0.1 means the jitter range is [0%, +20%], producing
	// values in [base*1.0, base*1.2). Positive-only jitter ensures the requeue
	// always fires after resyncPeriod has elapsed, so shouldReconcile always
	// returns true when the requeue fires.
	jitterFactor = 0.1
)

// CalculateJitteredDuration returns a duration in the range [base*1.0, base*1.2)
// using uniform random positive-only jitter. Jitter prevents thundering-herd
// problems when many resources share the same resync period: each resource
// independently picks a slightly different schedule so they do not all reconcile
// at once.
//
// Positive-only jitter guarantees the returned duration is always >= base,
// ensuring the requeue fires after resyncPeriod has elapsed and shouldReconcile
// returns true when the requeue fires.
//
// Each call produces an independent random value, so multiple resources calling
// this function with the same base will receive different durations (TS-011).
func CalculateJitteredDuration(base time.Duration) time.Duration {
	// rand.Float64() returns a value in [0.0, 1.0).
	// Multiplying by 2*jitterFactor gives [0.0, 0.2).
	// Adding 1.0 gives a multiplier in [1.0, 1.2).
	// This ensures the result is always >= base.
	multiplier := 1.0 + rand.Float64()*2*jitterFactor //nolint:gosec // math/rand/v2 is fine for jitter
	return time.Duration(float64(base) * multiplier)
}

// ShouldScheduleResync reports whether a periodic resync should be scheduled
// based on the effective resync period and the current reconcile status.
//
// It returns false (do not schedule) when:
//   - resyncPeriod <= 0: periodic resync is disabled (TS-007).
//   - reconcileStatus contains a terminal error: the resource is in a
//     non-retryable error state; resync would be pointless (TS-008).
//   - reconcileStatus already requests a requeue: another reconcile is
//     already pending so a resync requeue would be redundant (TS-012).
//
// When it returns true, the caller should schedule a requeue after
// CalculateJitteredDuration(resyncPeriod).
func ShouldScheduleResync(resyncPeriod time.Duration, reconcileStatus progress.ReconcileStatus) bool {
	// Resync disabled.
	if resyncPeriod <= 0 {
		return false
	}

	// Terminal error: no further reconciles will help.
	if err := reconcileStatus.GetError(); err != nil {
		var terminalError *orcerrors.TerminalError
		if errors.As(err, &terminalError) {
			return false
		}
	}

	// Another requeue is already pending; avoid adding a redundant one.
	if reconcileStatus.GetRequeue() > 0 {
		return false
	}

	return true
}
