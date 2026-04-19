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

// Package reconciler contains integration tests verifying that unmanaged ORC
// resources do not invoke any OpenStack write operations during periodic resync.
//
// Acceptance criteria covered (TS-007):
//   - Unmanaged resources update status without invoking actuator updates.
//   - Unmanaged resources still fetch current OpenStack state (read-only).
//   - CreateResource is NEVER called for unmanaged resources.
//   - Actuator reconcilers (GetResourceReconcilers) are NEVER called for
//     unmanaged resources (this is enforced in reconcileNormal).
//
// The tests use thin mock types for the actuator so that any unexpected call to
// a write method (CreateResource) causes the test to fail immediately.
package reconciler

import (
	"context"
	"errors"
	"iter"
	"testing"

	"github.com/go-logr/logr"
	"github.com/gophercloud/gophercloud/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/progress"
	"github.com/k-orc/openstack-resource-controller/v2/internal/scope"
	orcerrors "github.com/k-orc/openstack-resource-controller/v2/internal/util/errors"
	orcstrings "github.com/k-orc/openstack-resource-controller/v2/internal/util/strings"
)

// --------------------------------------------------------------------------
// Minimal fake OpenStack resource type used in these tests.
// --------------------------------------------------------------------------

// fakeOSResource is a stand-in for any OpenStack resource (e.g. flavors.Flavor).
type fakeOSResource struct {
	ID string
}

// --------------------------------------------------------------------------
// Mock actuator that enforces "no write operations" (TS-007).
//
// GetOSResourceByID and ListOSResourcesForImport are read-only operations and
// are expected to be called for unmanaged resources. CreateResource is a write
// operation; calling it causes the test to fail immediately.
// --------------------------------------------------------------------------

type noWriteActuator struct {
	t *testing.T

	// readByIDResult is returned by GetOSResourceByID.
	readByIDResult *fakeOSResource
	readByIDErr    error

	// listResult is returned by ListOSResourcesForImport.
	listResult []*fakeOSResource

	// Track which read methods were called so tests can assert that OpenStack
	// state was actually fetched.
	getByIDCalled bool
	listCalled    bool
}

var _ interfaces.CreateResourceActuator[*orcv1alpha1.Flavor, orcv1alpha1.Flavor, orcv1alpha1.FlavorFilter, fakeOSResource] = &noWriteActuator{}

func (a *noWriteActuator) GetResourceID(r *fakeOSResource) string {
	return r.ID
}

// GetOSResourceByID is a read-only operation: allowed for unmanaged resources.
func (a *noWriteActuator) GetOSResourceByID(_ context.Context, _ string) (*fakeOSResource, progress.ReconcileStatus) {
	a.getByIDCalled = true
	if a.readByIDErr != nil {
		return nil, progress.WrapError(a.readByIDErr)
	}
	return a.readByIDResult, nil
}

// ListOSResourcesForAdoption is only called in the creation flow for managed
// resources; for unmanaged resources this path should not be reached when
// statusID or importID is set.
func (a *noWriteActuator) ListOSResourcesForAdoption(_ context.Context, _ *orcv1alpha1.Flavor) (iter.Seq2[*fakeOSResource, error], bool) {
	// Return false to signal "no adoption" — this is a safe, read-only path.
	return nil, false
}

// ListOSResourcesForImport is a read-only operation: allowed for unmanaged
// resources using filter-based import.
func (a *noWriteActuator) ListOSResourcesForImport(_ context.Context, _ *orcv1alpha1.Flavor, _ orcv1alpha1.FlavorFilter) (iter.Seq2[*fakeOSResource, error], progress.ReconcileStatus) {
	a.listCalled = true
	return func(yield func(*fakeOSResource, error) bool) {
		for _, r := range a.listResult {
			if !yield(r, nil) {
				return
			}
		}
	}, nil
}

// CreateResource is a write operation: MUST NOT be called for unmanaged resources (TS-007).
func (a *noWriteActuator) CreateResource(_ context.Context, _ *orcv1alpha1.Flavor) (*fakeOSResource, progress.ReconcileStatus) {
	a.t.Fatal("CreateResource was called for an unmanaged resource: this is a write operation and MUST NOT be invoked (TS-007)")
	return nil, nil
}

// --------------------------------------------------------------------------
// fakeAdapter implements interfaces.APIObjectAdapter for *orcv1alpha1.Flavor.
// It delegates all metav1.Object methods to the underlying Flavor (which
// embeds metav1.ObjectMeta and therefore implements metav1.Object).
// --------------------------------------------------------------------------

type fakeAdapter struct {
	*orcv1alpha1.Flavor
}

// Ensure fakeAdapter implements APIObjectAdapter at compile time.
var _ interfaces.APIObjectAdapter[*orcv1alpha1.Flavor, orcv1alpha1.FlavorResourceSpec, orcv1alpha1.FlavorFilter] = fakeAdapter{}

// metav1.Object — all methods delegate to the embedded Flavor (which embeds
// metav1.ObjectMeta). We override only the methods not provided by embedding
// because embedding a non-pointer would copy the object and lose write-backs.

func (a fakeAdapter) GetUID() types.UID           { return a.Flavor.GetUID() }
func (a fakeAdapter) SetUID(uid types.UID)        { a.Flavor.SetUID(uid) }
func (a fakeAdapter) GetResourceVersion() string  { return a.Flavor.GetResourceVersion() }
func (a fakeAdapter) SetResourceVersion(v string) { a.Flavor.SetResourceVersion(v) }
func (a fakeAdapter) GetGeneration() int64        { return a.Flavor.GetGeneration() }
func (a fakeAdapter) SetGeneration(gen int64)     { a.Flavor.SetGeneration(gen) }
func (a fakeAdapter) GetFinalizers() []string     { return a.Flavor.GetFinalizers() }
func (a fakeAdapter) SetFinalizers(f []string)    { a.Flavor.SetFinalizers(f) }

// APIObjectAdapter-specific methods.
func (a fakeAdapter) GetObject() *orcv1alpha1.Flavor { return a.Flavor }

func (a fakeAdapter) GetManagementPolicy() orcv1alpha1.ManagementPolicy {
	return a.Flavor.Spec.ManagementPolicy
}

func (a fakeAdapter) GetManagedOptions() *orcv1alpha1.ManagedOptions {
	return a.Flavor.Spec.ManagedOptions
}

func (a fakeAdapter) GetResyncPeriod() *metav1.Duration {
	return a.Flavor.Spec.ResyncPeriod
}

func (a fakeAdapter) GetLastSyncTime() *metav1.Time {
	return a.Flavor.Status.LastSyncTime
}

func (a fakeAdapter) GetStatusID() *string {
	return a.Flavor.Status.ID
}

func (a fakeAdapter) GetResourceSpec() *orcv1alpha1.FlavorResourceSpec {
	return a.Flavor.Spec.Resource
}

func (a fakeAdapter) GetImportID() *string {
	if a.Flavor.Spec.Import == nil {
		return nil
	}
	return a.Flavor.Spec.Import.ID
}

func (a fakeAdapter) GetImportFilter() *orcv1alpha1.FlavorFilter {
	if a.Flavor.Spec.Import == nil {
		return nil
	}
	return a.Flavor.Spec.Import.Filter
}

// --------------------------------------------------------------------------
// fakeResourceController satisfies ResourceController for tests that pre-set
// the finalizer on the ORC object so that no Kubernetes Patch is needed.
// --------------------------------------------------------------------------

type fakeResourceController struct{}

var _ ResourceController = &fakeResourceController{}

func (c *fakeResourceController) GetName() string { return "test-controller" }

// GetK8sClient returns nil. If a Kubernetes Patch call is reached during a
// test, the nil dereference will cause a panic — signalling a bug in either
// the test setup (finalizer not pre-set) or the reconciler.
func (c *fakeResourceController) GetK8sClient() client.Client { return nil }

func (c *fakeResourceController) GetScopeFactory() scope.Factory { return nil }

// --------------------------------------------------------------------------
// Helpers for building test Flavors.
//
// All helpers pre-set the controller finalizer so that GetOrCreateOSResource
// does not attempt to call the Kubernetes client to add it.
// --------------------------------------------------------------------------

const testControllerName = "test-controller"

// finalizerFor returns the controller finalizer string for testControllerName.
func finalizerFor() string {
	return orcstrings.GetFinalizerName(testControllerName)
}

// unmanagedFlavorWithStatusID builds an unmanaged Flavor whose status.ID is
// already set (the normal periodic-resync case).
func unmanagedFlavorWithStatusID(statusID string) fakeAdapter {
	return fakeAdapter{
		Flavor: &orcv1alpha1.Flavor{
			ObjectMeta: metav1.ObjectMeta{
				Name:       "test-flavor",
				Namespace:  "default",
				Finalizers: []string{finalizerFor()},
			},
			Spec: orcv1alpha1.FlavorSpec{
				ManagementPolicy: orcv1alpha1.ManagementPolicyUnmanaged,
				Import: &orcv1alpha1.FlavorImport{
					ID: ptr.To(statusID),
				},
			},
			Status: orcv1alpha1.FlavorStatus{
				ID: ptr.To(statusID),
			},
		},
	}
}

// unmanagedFlavorWithImportID builds an unmanaged Flavor that specifies an
// importID but has no statusID yet (before first reconcile).
func unmanagedFlavorWithImportID(importID string) fakeAdapter {
	return fakeAdapter{
		Flavor: &orcv1alpha1.Flavor{
			ObjectMeta: metav1.ObjectMeta{
				Name:       "test-flavor",
				Namespace:  "default",
				Finalizers: []string{finalizerFor()},
			},
			Spec: orcv1alpha1.FlavorSpec{
				ManagementPolicy: orcv1alpha1.ManagementPolicyUnmanaged,
				Import: &orcv1alpha1.FlavorImport{
					ID: ptr.To(importID),
				},
			},
		},
	}
}

// unmanagedFlavorWithFilter builds an unmanaged Flavor using filter-based
// import with no statusID.
func unmanagedFlavorWithFilter(filter orcv1alpha1.FlavorFilter) fakeAdapter {
	return fakeAdapter{
		Flavor: &orcv1alpha1.Flavor{
			ObjectMeta: metav1.ObjectMeta{
				Name:       "test-flavor",
				Namespace:  "default",
				Finalizers: []string{finalizerFor()},
			},
			Spec: orcv1alpha1.FlavorSpec{
				ManagementPolicy: orcv1alpha1.ManagementPolicyUnmanaged,
				Import: &orcv1alpha1.FlavorImport{
					Filter: &filter,
				},
			},
		},
	}
}

// unmanagedFlavorNoImport builds an unmanaged Flavor with neither statusID nor
// import — an invalid configuration that API validation normally prevents.
func unmanagedFlavorNoImport() fakeAdapter {
	return fakeAdapter{
		Flavor: &orcv1alpha1.Flavor{
			ObjectMeta: metav1.ObjectMeta{
				Name:       "test-flavor",
				Namespace:  "default",
				Finalizers: []string{finalizerFor()},
			},
			Spec: orcv1alpha1.FlavorSpec{
				ManagementPolicy: orcv1alpha1.ManagementPolicyUnmanaged,
			},
		},
	}
}

// notFoundErr returns a gophercloud not-found error that orcerrors.IsNotFound
// will recognise.
func notFoundErr() error {
	return gophercloud.ErrResourceNotFound{Name: "missing-id", ResourceType: "flavor"}
}

// --------------------------------------------------------------------------
// Tests
// --------------------------------------------------------------------------

// TestGetOrCreateOSResource_UnmanagedByStatusID verifies that for an unmanaged
// resource whose status.ID is already set (the periodic resync case), only
// GetOSResourceByID is called. No write operation (CreateResource) must be
// invoked (TS-007: unmanaged resources update status without invoking actuator
// updates).
func TestGetOrCreateOSResource_UnmanagedByStatusID(t *testing.T) {
	t.Parallel()

	const resourceID = "test-flavor-id"
	osResource := &fakeOSResource{ID: resourceID}

	actuator := &noWriteActuator{t: t, readByIDResult: osResource}
	adapter := unmanagedFlavorWithStatusID(resourceID)

	got, rs := GetOrCreateOSResource(context.Background(), logr.Discard(), &fakeResourceController{}, adapter, actuator)

	if needsReschedule, err := rs.NeedsReschedule(); needsReschedule {
		t.Fatalf("unexpected reconcile status: needsReschedule=%v err=%v", needsReschedule, err)
	}
	if got == nil || got.ID != resourceID {
		t.Errorf("got resource %v, want ID=%q", got, resourceID)
	}
	// Verify OpenStack state was fetched (acceptance criterion: unmanaged
	// resources still fetch current OpenStack state).
	if !actuator.getByIDCalled {
		t.Error("GetOSResourceByID was not called: unmanaged resources must still fetch current OpenStack state")
	}
	if actuator.listCalled {
		t.Error("ListOSResourcesForImport was unexpectedly called")
	}
}

// TestGetOrCreateOSResource_UnmanagedByImportID verifies that for an unmanaged
// resource using import-by-ID (no statusID yet), GetOSResourceByID is called
// as a read-only operation and CreateResource is never invoked (TS-007).
func TestGetOrCreateOSResource_UnmanagedByImportID(t *testing.T) {
	t.Parallel()

	const importID = "imported-flavor-id"
	osResource := &fakeOSResource{ID: importID}

	actuator := &noWriteActuator{t: t, readByIDResult: osResource}
	adapter := unmanagedFlavorWithImportID(importID)

	got, rs := GetOrCreateOSResource(context.Background(), logr.Discard(), &fakeResourceController{}, adapter, actuator)

	if needsReschedule, err := rs.NeedsReschedule(); needsReschedule {
		t.Fatalf("unexpected reconcile status: needsReschedule=%v err=%v", needsReschedule, err)
	}
	if got == nil || got.ID != importID {
		t.Errorf("got resource %v, want ID=%q", got, importID)
	}
	if !actuator.getByIDCalled {
		t.Error("GetOSResourceByID was not called: unmanaged resources must still fetch current OpenStack state")
	}
}

// TestGetOrCreateOSResource_UnmanagedByFilter verifies that for an unmanaged
// resource using filter-based import, ListOSResourcesForImport is called as a
// read-only operation and CreateResource is never invoked (TS-007).
func TestGetOrCreateOSResource_UnmanagedByFilter(t *testing.T) {
	t.Parallel()

	osResource := &fakeOSResource{ID: "filter-flavor-id"}
	filter := orcv1alpha1.FlavorFilter{
		Name: ptr.To[orcv1alpha1.OpenStackName]("my-flavor"),
	}

	actuator := &noWriteActuator{t: t, listResult: []*fakeOSResource{osResource}}
	adapter := unmanagedFlavorWithFilter(filter)

	got, rs := GetOrCreateOSResource(context.Background(), logr.Discard(), &fakeResourceController{}, adapter, actuator)

	if needsReschedule, err := rs.NeedsReschedule(); needsReschedule {
		t.Fatalf("unexpected reconcile status: needsReschedule=%v err=%v", needsReschedule, err)
	}
	if got == nil || got.ID != osResource.ID {
		t.Errorf("got resource %v, want ID=%q", got, osResource.ID)
	}
	if !actuator.listCalled {
		t.Error("ListOSResourcesForImport was not called: unmanaged resources must still fetch current OpenStack state")
	}
}

// TestGetOrCreateOSResource_UnmanagedNoImport verifies that an unmanaged
// resource with no import configuration returns a terminal error without calling
// CreateResource (TS-007). API validation should prevent this state in
// production, but the reconciler must handle it safely.
func TestGetOrCreateOSResource_UnmanagedNoImport(t *testing.T) {
	t.Parallel()

	actuator := &noWriteActuator{t: t}
	adapter := unmanagedFlavorNoImport()

	_, rs := GetOrCreateOSResource(context.Background(), logr.Discard(), &fakeResourceController{}, adapter, actuator)

	_, err := rs.NeedsReschedule()
	if err == nil {
		t.Fatal("expected a terminal error for unmanaged resource with no import, got nil")
	}

	// Verify it is a TerminalError so the controller does not retry uselessly.
	var termErr *orcerrors.TerminalError
	if !errors.As(err, &termErr) {
		t.Errorf("expected a TerminalError, got %T: %v", err, err)
	}

	if actuator.getByIDCalled {
		t.Error("GetOSResourceByID was unexpectedly called")
	}
	if actuator.listCalled {
		t.Error("ListOSResourcesForImport was unexpectedly called")
	}
}

// TestGetOrCreateOSResource_UnmanagedStatusIDDeleted verifies that when an
// unmanaged resource's OpenStack resource has been deleted externally (the
// controller receives a not-found error), the controller returns a terminal
// error rather than calling CreateResource (TS-007).
func TestGetOrCreateOSResource_UnmanagedStatusIDDeleted(t *testing.T) {
	t.Parallel()

	const resourceID = "deleted-flavor-id"

	// Simulate OpenStack returning a 404 / not-found.
	actuator := &noWriteActuator{t: t, readByIDErr: notFoundErr()}
	adapter := unmanagedFlavorWithStatusID(resourceID)

	_, rs := GetOrCreateOSResource(context.Background(), logr.Discard(), &fakeResourceController{}, adapter, actuator)

	_, err := rs.NeedsReschedule()
	if err == nil {
		t.Fatal("expected an error when OpenStack resource is not found, got nil")
	}

	// The error must be terminal: the resource was deleted externally and cannot
	// be recovered by the controller.
	var termErr *orcerrors.TerminalError
	if !errors.As(err, &termErr) {
		t.Errorf("expected a TerminalError for externally-deleted resource, got %T: %v", err, err)
	}

	// The controller must still have attempted to fetch the resource.
	if !actuator.getByIDCalled {
		t.Error("GetOSResourceByID was not called: the controller should attempt to fetch the resource before concluding it is gone")
	}
}
