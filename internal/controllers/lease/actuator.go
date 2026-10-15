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

package lease

import (
	"context"
	"fmt"
	"iter"
	"slices"
	"time"

	"github.com/gophercloud/gophercloud/v2/openstack/reservation/v1/leases"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v3/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v3/internal/controllers/generic/progress"
	"github.com/k-orc/openstack-resource-controller/v3/internal/osclients"
	"github.com/k-orc/openstack-resource-controller/v3/internal/util/dependency"
	orcerrors "github.com/k-orc/openstack-resource-controller/v3/internal/util/errors"
)

// OpenStack resource types
type (
	osResourceT = leases.Lease

	createResourceActuator    = interfaces.CreateResourceActuator[orcObjectPT, orcObjectT, filterT, osResourceT]
	deleteResourceActuator    = interfaces.DeleteResourceActuator[orcObjectPT, orcObjectT, osResourceT]
	reconcileResourceActuator = interfaces.ReconcileResourceActuator[orcObjectPT, osResourceT]
	resourceReconciler        = interfaces.ResourceReconciler[orcObjectPT, osResourceT]
	helperFactory             = interfaces.ResourceHelperFactory[orcObjectPT, orcObjectT, resourceSpecT, filterT, osResourceT]
)

// The frequency to poll when waiting for the resource to become available
const leaseAvailablePollingPeriod = 15 * time.Second

// The longest time to wait before checking on a lease which has not started yet
const leasePendingMaxPollingPeriod = 10 * time.Minute

// The frequency to poll when waiting for the resource to be deleted
const leaseDeletingPollingPeriod = 15 * time.Second

// Blazar stores lease names in a column of this length
const leaseNameMaxLength = 80

type leaseActuator struct {
	osClient  osclients.LeaseClient
	k8sClient client.Client
}

var _ createResourceActuator = leaseActuator{}
var _ deleteResourceActuator = leaseActuator{}
var _ reconcileResourceActuator = leaseActuator{}

func (leaseActuator) GetResourceID(osResource *osResourceT) string {
	return osResource.ID
}

func (actuator leaseActuator) GetOSResourceByID(ctx context.Context, id string) (*osResourceT, progress.ReconcileStatus) {
	resource, err := actuator.osClient.GetLease(ctx, id)
	if err != nil {
		return nil, progress.WrapError(err)
	}
	return resource, nil
}

// Blazar ignores the query parameters of the list request, so all filtering
// is done client side.
func (actuator leaseActuator) listOSResources(ctx context.Context, filters []osclients.ResourceFilter[osResourceT]) iter.Seq2[*osResourceT, error] {
	return osclients.Filter(actuator.osClient.ListLeases(ctx, leases.ListOpts{}), filters...)
}

func (actuator leaseActuator) ListOSResourcesForAdoption(ctx context.Context, orcObject orcObjectPT) (iter.Seq2[*osResourceT, error], bool) {
	resourceSpec := orcObject.Spec.Resource
	if resourceSpec == nil {
		return nil, false
	}

	name := getResourceName(orcObject)
	endDate := toBlazarDate(resourceSpec.EndDate.Time)
	reservationTypes := specReservationTypes(resourceSpec)

	// Blazar allows duplicate lease names, so also match the dates and the
	// types of the reservations to avoid adopting an unrelated lease.
	filters := []osclients.ResourceFilter[osResourceT]{
		func(l *leases.Lease) bool {
			if l.Name != name || !l.EndDate.Equal(endDate) {
				return false
			}
			if resourceSpec.StartDate != nil && !l.StartDate.Equal(toBlazarDate(resourceSpec.StartDate.Time)) {
				return false
			}
			return slices.Equal(osReservationTypes(l), reservationTypes)
		},
	}

	return actuator.listOSResources(ctx, filters), true
}

func (actuator leaseActuator) ListOSResourcesForImport(ctx context.Context, obj orcObjectPT, filter filterT) (iter.Seq2[*osResourceT, error], progress.ReconcileStatus) {
	var filters []osclients.ResourceFilter[osResourceT]

	if filter.Name != nil {
		filters = append(filters, func(l *leases.Lease) bool { return l.Name == string(*filter.Name) })
	}

	return actuator.listOSResources(ctx, filters), nil
}

func (actuator leaseActuator) CreateResource(ctx context.Context, obj orcObjectPT) (*osResourceT, progress.ReconcileStatus) {
	resource := obj.Spec.Resource

	if resource == nil {
		// Should have been caught by API validation
		return nil, progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Creation requested, but spec.resource is not set"))
	}

	// spec.resource.name is validated by the API, but the object name it
	// falls back to may be longer than Blazar allows.
	name := getResourceName(obj)
	if len(name) > leaseNameMaxLength {
		return nil, progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration,
				fmt.Sprintf("lease name %q is longer than %d characters: set spec.resource.name", name, leaseNameMaxLength)))
	}

	// Resolve the flavors of all flavor reservations before creating anything
	var reconcileStatus progress.ReconcileStatus
	flavorIDs := make(map[orcv1alpha1.KubernetesNameRef]string)
	for i := range resource.Reservations {
		flavorInstance := resource.Reservations[i].FlavorInstance
		if flavorInstance == nil {
			continue
		}
		flavor, flavorReconcileStatus := dependency.FetchDependency[*orcv1alpha1.Flavor](
			ctx, actuator.k8sClient, obj.Namespace,
			&flavorInstance.FlavorRef, "Flavor",
			orcv1alpha1.IsAvailable,
		)
		reconcileStatus = reconcileStatus.WithReconcileStatus(flavorReconcileStatus)
		flavorIDs[flavorInstance.FlavorRef] = ptr.Deref(flavor.Status.ID, "")
	}
	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return nil, reconcileStatus
	}

	reservations := make([]leases.ReservationOptsBuilder, len(resource.Reservations))
	for i := range resource.Reservations {
		reservations[i] = reservationOpts(&resource.Reservations[i], flavorIDs)
	}

	createOpts := leases.CreateOpts{
		Name:         name,
		EndDate:      resource.EndDate.Time,
		Reservations: reservations,
	}
	// A zero start date starts the lease immediately
	if resource.StartDate != nil {
		createOpts.StartDate = resource.StartDate.Time
	}

	osResource, err := actuator.osClient.CreateLease(ctx, createOpts)
	if err != nil {
		if !orcerrors.IsRetryable(err) {
			err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration creating resource: "+err.Error(), err)
		}
		return nil, progress.WrapError(err)
	}

	return osResource, nil
}

func reservationOpts(reservation *orcv1alpha1.LeaseReservation, flavorIDs map[orcv1alpha1.KubernetesNameRef]string) leases.ReservationOptsBuilder {
	if flavorInstance := reservation.FlavorInstance; flavorInstance != nil {
		return leases.FlavorInstanceReservationOpts{
			Amount:   int(flavorInstance.Amount),
			FlavorID: flavorIDs[flavorInstance.FlavorRef],
		}
	}

	if host := reservation.Host; host != nil {
		return leases.HostReservationOpts{
			Min:                  int(host.Min),
			Max:                  int(host.Max),
			HypervisorProperties: ptr.Deref(host.HypervisorProperties, ""),
			ResourceProperties:   ptr.Deref(host.ResourceProperties, ""),
		}
	}

	// API validation guarantees that exactly one reservation type is set
	instance := reservation.Instance
	return leases.InstanceReservationOpts{
		Amount:             int(instance.Amount),
		VCPUs:              int(instance.Vcpus),
		MemoryMB:           int(instance.MemoryMB),
		DiskGB:             int(ptr.Deref(instance.DiskGB, 0)),
		Affinity:           instance.Affinity,
		ResourceProperties: ptr.Deref(instance.ResourceProperties, ""),
	}
}

// specReservationTypes returns the sorted Blazar resource types of the
// reservations in the spec.
func specReservationTypes(resourceSpec *resourceSpecT) []string {
	types := make([]string, len(resourceSpec.Reservations))
	for i := range resourceSpec.Reservations {
		switch reservation := &resourceSpec.Reservations[i]; {
		case reservation.Host != nil:
			types[i] = leases.ResourceTypeHost
		case reservation.FlavorInstance != nil:
			types[i] = leases.ResourceTypeFlavorInstance
		default:
			types[i] = leases.ResourceTypeInstance
		}
	}
	slices.Sort(types)
	return types
}

// osReservationTypes returns the sorted resource types of the reservations of
// a lease.
func osReservationTypes(l *leases.Lease) []string {
	types := make([]string, len(l.Reservations))
	for i := range l.Reservations {
		types[i] = l.Reservations[i].ResourceType
	}
	slices.Sort(types)
	return types
}

// toBlazarDate returns the date as Blazar stores it: in UTC, truncated to the
// minute.
func toBlazarDate(t time.Time) time.Time {
	return t.UTC().Truncate(time.Minute)
}

func (actuator leaseActuator) DeleteResource(ctx context.Context, _ orcObjectPT, resource *osResourceT) progress.ReconcileStatus {
	if resource.Status == LeaseStatusDeleting {
		return progress.WaitingOnOpenStack(progress.WaitingOnReady, leaseDeletingPollingPeriod)
	}
	return progress.WrapError(actuator.osClient.DeleteLease(ctx, resource.ID))
}

func (actuator leaseActuator) GetResourceReconcilers(ctx context.Context, orcObject orcObjectPT, osResource *osResourceT, controller interfaces.ResourceController) ([]resourceReconciler, progress.ReconcileStatus) {
	return []resourceReconciler{
		actuator.checkStatus,
	}, nil
}

// checkStatus stops reconciling a lease once it can no longer become
// available. Waiting for a lease to start is handled by the status writer.
func (leaseActuator) checkStatus(_ context.Context, _ orcObjectPT, osResource *osResourceT) progress.ReconcileStatus {
	switch osResource.Status {
	case LeaseStatusError:
		return progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonUnrecoverableError, "Lease is in ERROR state"))
	case LeaseStatusTerminated:
		return progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonUnrecoverableError, "Lease has ended"))
	default:
		return nil
	}
}

type leaseHelperFactory struct{}

var _ helperFactory = leaseHelperFactory{}

func newActuator(ctx context.Context, orcObject *orcv1alpha1.Lease, controller interfaces.ResourceController) (leaseActuator, progress.ReconcileStatus) {
	log := ctrl.LoggerFrom(ctx)

	// Ensure credential secrets exist and have our finalizer
	_, reconcileStatus := credentialsDependency.RequireDependencies(ctx, controller.GetK8sClient(), orcObject, func(*corev1.Secret) bool { return true })
	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return leaseActuator{}, reconcileStatus
	}

	clientScope, err := controller.GetScopeFactory().NewClientScopeFromObject(ctx, controller.GetK8sClient(), log, orcObject)
	if err != nil {
		return leaseActuator{}, progress.WrapError(err)
	}
	osClient, err := clientScope.NewLeaseClient()
	if err != nil {
		return leaseActuator{}, progress.WrapError(err)
	}

	return leaseActuator{
		osClient:  osClient,
		k8sClient: controller.GetK8sClient(),
	}, nil
}

func (leaseHelperFactory) NewAPIObjectAdapter(obj orcObjectPT) adapterI {
	return leaseAdapter{obj}
}

func (leaseHelperFactory) NewCreateActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (createResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, orcObject, controller)
}

func (leaseHelperFactory) NewDeleteActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (deleteResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, orcObject, controller)
}
