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
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/progress"
	orcerrors "github.com/k-orc/openstack-resource-controller/v2/internal/util/errors"
	orcapplyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/pkg/clients/applyconfiguration/api/v1alpha1"
)

// TestSetCommonConditions_TerminalErrorOverridesUnknownToFalse verifies that
// when ResourceAvailableStatus returns ConditionUnknown and a terminal error is
// present, SetCommonConditions overrides the Available condition to False.
//
// This is the fix for the network-external-deletion-import CI failure: when an
// imported network is externally deleted, ORC sets a terminal error but
// ResourceAvailableStatus returns ConditionUnknown (because Status.ID is set
// but the OS resource is nil). The Available condition must be False, not
// Unknown, when a terminal error is present.
func TestSetCommonConditions_TerminalErrorOverridesUnknownToFalse(t *testing.T) {
	t.Parallel()

	flavor := &orcv1alpha1.Flavor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-flavor",
			Namespace: "default",
		},
	}

	termErr := orcerrors.Terminal(orcv1alpha1.ConditionReasonUnrecoverableError, "resource has been deleted from OpenStack", nil)
	reconcileStatus := progress.WrapError(termErr)

	applyConfigStatus := orcapplyconfigv1alpha1.FlavorStatus()
	now := metav1.Now()

	// Call SetCommonConditions with ConditionUnknown (as ResourceAvailableStatus
	// returns when osResource==nil and Status.ID!=nil) and a terminal error.
	SetCommonConditions(flavor, applyConfigStatus, metav1.ConditionUnknown, reconcileStatus, now)

	// Find the Available condition in the resulting apply configuration.
	var availableCondition *metav1.ConditionStatus
	for i := range applyConfigStatus.Conditions {
		if applyConfigStatus.Conditions[i].Type != nil && *applyConfigStatus.Conditions[i].Type == orcv1alpha1.ConditionAvailable {
			availableCondition = applyConfigStatus.Conditions[i].Status
			break
		}
	}

	if availableCondition == nil {
		t.Fatal("Available condition not set in apply configuration")
	}

	if *availableCondition != metav1.ConditionFalse {
		t.Errorf("Available condition status = %q; want %q (terminal error should override Unknown → False)",
			*availableCondition, metav1.ConditionFalse)
	}
}

// TestSetCommonConditions_TerminalErrorWithFalseRemainingFalse verifies that
// when ResourceAvailableStatus already returns ConditionFalse and a terminal
// error is present, the Available condition remains False (not changed).
func TestSetCommonConditions_TerminalErrorWithFalseRemainingFalse(t *testing.T) {
	t.Parallel()

	flavor := &orcv1alpha1.Flavor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-flavor",
			Namespace: "default",
		},
	}

	termErr := orcerrors.Terminal(orcv1alpha1.ConditionReasonUnrecoverableError, "resource has been deleted from OpenStack", nil)
	reconcileStatus := progress.WrapError(termErr)

	applyConfigStatus := orcapplyconfigv1alpha1.FlavorStatus()
	now := metav1.Now()

	// Call SetCommonConditions with ConditionFalse (resource not found, no ID).
	SetCommonConditions(flavor, applyConfigStatus, metav1.ConditionFalse, reconcileStatus, now)

	var availableCondition *metav1.ConditionStatus
	for i := range applyConfigStatus.Conditions {
		if applyConfigStatus.Conditions[i].Type != nil && *applyConfigStatus.Conditions[i].Type == orcv1alpha1.ConditionAvailable {
			availableCondition = applyConfigStatus.Conditions[i].Status
			break
		}
	}

	if availableCondition == nil {
		t.Fatal("Available condition not set in apply configuration")
	}

	if *availableCondition != metav1.ConditionFalse {
		t.Errorf("Available condition status = %q; want %q", *availableCondition, metav1.ConditionFalse)
	}
}

// TestSetCommonConditions_NoTerminalErrorKeepsUnknown verifies that when
// ResourceAvailableStatus returns ConditionUnknown and no terminal error is
// present (e.g. a transient error or progress), the Available condition remains
// Unknown.
func TestSetCommonConditions_NoTerminalErrorKeepsUnknown(t *testing.T) {
	t.Parallel()

	flavor := &orcv1alpha1.Flavor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-flavor",
			Namespace: "default",
		},
	}

	// Transient error (not terminal) — Available should remain Unknown.
	reconcileStatus := progress.NewReconcileStatus().WithProgressMessage("waiting for OpenStack")

	applyConfigStatus := orcapplyconfigv1alpha1.FlavorStatus()
	now := metav1.Now()

	SetCommonConditions(flavor, applyConfigStatus, metav1.ConditionUnknown, reconcileStatus, now)

	var availableCondition *metav1.ConditionStatus
	for i := range applyConfigStatus.Conditions {
		if applyConfigStatus.Conditions[i].Type != nil && *applyConfigStatus.Conditions[i].Type == orcv1alpha1.ConditionAvailable {
			availableCondition = applyConfigStatus.Conditions[i].Status
			break
		}
	}

	if availableCondition == nil {
		t.Fatal("Available condition not set in apply configuration")
	}

	if *availableCondition != metav1.ConditionUnknown {
		t.Errorf("Available condition status = %q; want %q (no terminal error should not change Unknown)",
			*availableCondition, metav1.ConditionUnknown)
	}
}
