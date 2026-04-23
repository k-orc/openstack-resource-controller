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

// Package reconciler contains unit tests for external deletion handling in
// GetOrCreateOSResource.
//
// These tests cover all combinations of management policy and import status
// when a resource's OpenStack counterpart is not found (404), verifying that:
//
//   - Managed, ORC-created resources trigger recreation (return nil, nil)
//   - Managed, imported-by-ID resources return a terminal error
//   - Managed, imported-by-filter resources return a terminal error
//   - Unmanaged resources return a terminal error
//   - Managed, existing resources continue through the normal update flow
package reconciler

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	orcerrors "github.com/k-orc/openstack-resource-controller/v2/internal/util/errors"
)

// --------------------------------------------------------------------------
// Helpers for building test Flavors used in external-deletion tests.
//
// All helpers pre-set the controller finalizer so that GetOrCreateOSResource
// does not attempt to call the Kubernetes client to add it.
// --------------------------------------------------------------------------

// managedFlavorImportedByFilter builds a managed Flavor that was imported via
// a filter (has a non-nil import.Filter) and whose status.ID is already set.
// When GetOSResourceByID returns a not-found error, IsImported() returns true
// because GetImportFilter() is non-nil, so the controller must return a
// terminal error rather than triggering recreation.
func managedFlavorImportedByFilter(statusID string) fakeAdapter {
	return fakeAdapter{
		Flavor: &orcv1alpha1.Flavor{
			ObjectMeta: metav1.ObjectMeta{
				Name:       "test-flavor",
				Namespace:  "default",
				Finalizers: []string{finalizerFor()},
			},
			Spec: orcv1alpha1.FlavorSpec{
				ManagementPolicy: orcv1alpha1.ManagementPolicyManaged,
				Import: &orcv1alpha1.FlavorImport{
					Filter: &orcv1alpha1.FlavorFilter{
						Name: ptr.To[orcv1alpha1.OpenStackName]("my-flavor"),
					},
				},
			},
			Status: orcv1alpha1.FlavorStatus{
				ID: ptr.To(statusID),
			},
		},
	}
}

// --------------------------------------------------------------------------
// External deletion tests — all use GetOrCreateOSResource directly.
// --------------------------------------------------------------------------

// TestGetOrCreateOSResource_ExternalDeletion_ManagedOrcCreated verifies that
// when a managed, ORC-created resource (not imported) is externally deleted,
// GetOrCreateOSResource returns (nil, nil) to signal the caller should clear
// status.ID and trigger recreation on the next reconcile.
func TestGetOrCreateOSResource_ExternalDeletion_ManagedOrcCreated(t *testing.T) {
	t.Parallel()

	const resourceID = "orc-created-flavor-id"

	actuator := &noWriteActuator{t: t, readByIDErr: notFoundErr()}
	adapter := managedFlavorWithStatusID(resourceID)

	got, rs := GetOrCreateOSResource(context.Background(), logr.Discard(), &fakeResourceController{}, adapter, actuator)

	// Expect (nil, nil): caller will clear status.id and trigger recreation.
	needsReschedule, err := rs.NeedsReschedule()
	if needsReschedule {
		t.Fatalf("expected no rescheduling for externally-deleted managed resource (recreation path), got needsReschedule=%v err=%v", needsReschedule, err)
	}
	if got != nil {
		t.Errorf("expected nil osResource for recreation path, got %v", got)
	}
	if !actuator.getByIDCalled {
		t.Error("GetOSResourceByID was not called: controller must attempt to fetch the resource")
	}
}

// TestGetOrCreateOSResource_ExternalDeletion_ManagedImportedByID verifies that
// when a managed resource originally imported by ID is externally deleted, the
// controller returns a terminal error. Recreation is not possible for imported
// resources because we cannot know the original creation parameters.
func TestGetOrCreateOSResource_ExternalDeletion_ManagedImportedByID(t *testing.T) {
	t.Parallel()

	const resourceID = "imported-by-id-flavor-id"

	actuator := &noWriteActuator{t: t, readByIDErr: notFoundErr()}
	adapter := managedFlavorImportedByID(resourceID)

	_, rs := GetOrCreateOSResource(context.Background(), logr.Discard(), &fakeResourceController{}, adapter, actuator)

	_, err := rs.NeedsReschedule()
	if err == nil {
		t.Fatal("expected a terminal error for externally-deleted imported-by-ID resource, got nil")
	}

	var termErr *orcerrors.TerminalError
	if !errors.As(err, &termErr) {
		t.Errorf("expected a TerminalError for externally-deleted imported-by-ID resource, got %T: %v", err, err)
	}
	if !actuator.getByIDCalled {
		t.Error("GetOSResourceByID was not called")
	}
}

// TestGetOrCreateOSResource_ExternalDeletion_ManagedImportedByFilter verifies
// that when a managed resource originally imported via a filter is externally
// deleted, the controller returns a terminal error. As with import-by-ID,
// recreation is not possible.
func TestGetOrCreateOSResource_ExternalDeletion_ManagedImportedByFilter(t *testing.T) {
	t.Parallel()

	const resourceID = "imported-by-filter-flavor-id"

	actuator := &noWriteActuator{t: t, readByIDErr: notFoundErr()}
	adapter := managedFlavorImportedByFilter(resourceID)

	_, rs := GetOrCreateOSResource(context.Background(), logr.Discard(), &fakeResourceController{}, adapter, actuator)

	_, err := rs.NeedsReschedule()
	if err == nil {
		t.Fatal("expected a terminal error for externally-deleted imported-by-filter resource, got nil")
	}

	var termErr *orcerrors.TerminalError
	if !errors.As(err, &termErr) {
		t.Errorf("expected a TerminalError for externally-deleted imported-by-filter resource, got %T: %v", err, err)
	}
	if !actuator.getByIDCalled {
		t.Error("GetOSResourceByID was not called")
	}
}

// TestGetOrCreateOSResource_ExternalDeletion_Unmanaged verifies that when an
// unmanaged resource is externally deleted (404), the controller returns a
// terminal error instead of calling CreateResource.
func TestGetOrCreateOSResource_ExternalDeletion_Unmanaged(t *testing.T) {
	t.Parallel()

	const resourceID = "unmanaged-deleted-flavor-id"

	actuator := &noWriteActuator{t: t, readByIDErr: notFoundErr()}
	adapter := unmanagedFlavorWithStatusID(resourceID)

	_, rs := GetOrCreateOSResource(context.Background(), logr.Discard(), &fakeResourceController{}, adapter, actuator)

	_, err := rs.NeedsReschedule()
	if err == nil {
		t.Fatal("expected a terminal error for externally-deleted unmanaged resource, got nil")
	}

	var termErr *orcerrors.TerminalError
	if !errors.As(err, &termErr) {
		t.Errorf("expected a TerminalError for externally-deleted unmanaged resource, got %T: %v", err, err)
	}
	if !actuator.getByIDCalled {
		t.Error("GetOSResourceByID was not called: controller must attempt to fetch the resource")
	}
}

// TestGetOrCreateOSResource_ExternalDeletion_ManagedResourceExists verifies
// the normal update flow: when a managed, ORC-created resource still exists in
// OpenStack, GetOrCreateOSResource returns the resource with a nil reconcile
// status so the caller proceeds with reconciliation (no recreation, no error).
func TestGetOrCreateOSResource_ExternalDeletion_ManagedResourceExists(t *testing.T) {
	t.Parallel()

	const resourceID = "existing-managed-flavor-id"
	osResource := &fakeOSResource{ID: resourceID}

	actuator := &noWriteActuator{t: t, readByIDResult: osResource}
	adapter := managedFlavorWithStatusID(resourceID)

	got, rs := GetOrCreateOSResource(context.Background(), logr.Discard(), &fakeResourceController{}, adapter, actuator)

	needsReschedule, err := rs.NeedsReschedule()
	if needsReschedule {
		t.Fatalf("expected no rescheduling for existing resource, got needsReschedule=%v err=%v", needsReschedule, err)
	}
	if got == nil || got.ID != resourceID {
		t.Errorf("expected osResource with ID=%q, got %v", resourceID, got)
	}
	if !actuator.getByIDCalled {
		t.Error("GetOSResourceByID was not called")
	}
}
