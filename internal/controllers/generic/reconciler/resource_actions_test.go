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

// Package reconciler contains unit tests for external deletion handling in
// GetOrCreateOSResource.
//
// These tests cover management policy behaviour when a resource's OpenStack
// counterpart is not found (404), verifying that:
//
//   - Managed, ORC-created resources trigger recreation (IsExternallyDeleted)
//   - Unmanaged resources return a terminal error
//   - Managed, existing resources continue through the normal update flow
package reconciler

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr"

	orcerrors "github.com/k-orc/openstack-resource-controller/v2/internal/util/errors"
)

// --------------------------------------------------------------------------
// External deletion tests — all use GetOrCreateOSResource directly.
// --------------------------------------------------------------------------

// TestGetOrCreateOSResource_ExternalDeletion_ManagedOrcCreated verifies that
// when a managed, ORC-created resource (not imported) is externally deleted,
// GetOrCreateOSResource returns IsExternallyDeleted to signal the caller should
// clear status.ID and trigger recreation on the next reconcile.
func TestGetOrCreateOSResource_ExternalDeletion_ManagedOrcCreated(t *testing.T) {
	t.Parallel()

	const resourceID = "orc-created-flavor-id"

	actuator := &noWriteActuator{t: t, readByIDErr: notFoundErr()}
	adapter := managedFlavorWithStatusID(resourceID)

	got, rs := GetOrCreateOSResource(context.Background(), logr.Discard(), &fakeResourceController{}, adapter, actuator)

	if !rs.IsExternallyDeleted() {
		t.Fatal("expected IsExternallyDeleted for externally-deleted managed resource")
	}
	if got != nil {
		t.Errorf("expected nil osResource for recreation path, got %v", got)
	}
	if !actuator.getByIDCalled {
		t.Error("GetOSResourceByID was not called: controller must attempt to fetch the resource")
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

// TestGetOrCreateOSResource_ExternalDeletion_ManagedImported verifies that
// imported resources are not recreated after external deletion, even if the
// management policy is managed.
func TestGetOrCreateOSResource_ExternalDeletion_ManagedImported(t *testing.T) {
	t.Parallel()

	const resourceID = "managed-imported-deleted-flavor-id"

	actuator := &noWriteActuator{t: t, readByIDErr: notFoundErr()}
	adapter := managedImportedFlavorWithStatusID(resourceID)

	_, rs := GetOrCreateOSResource(context.Background(), logr.Discard(), &fakeResourceController{}, adapter, actuator)

	if rs.IsExternallyDeleted() {
		t.Fatal("expected imported resource deletion to be terminal, not externally-deleted recreation")
	}

	_, err := rs.NeedsReschedule()
	if err == nil {
		t.Fatal("expected a terminal error for externally-deleted imported resource, got nil")
	}

	var termErr *orcerrors.TerminalError
	if !errors.As(err, &termErr) {
		t.Errorf("expected a TerminalError for externally-deleted imported resource, got %T: %v", err, err)
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
