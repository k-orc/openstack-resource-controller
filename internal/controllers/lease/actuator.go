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
	"iter"
	"time"

	"github.com/gophercloud/gophercloud/v2/openstack/reservation/v1/leases"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v3/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v3/internal/controllers/generic/progress"
	"github.com/k-orc/openstack-resource-controller/v3/internal/logging"
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
// The frequency to poll when waiting for the resource to be deleted
const leaseDeletingPollingPeriod = 15 * time.Second

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

func (actuator leaseActuator) ListOSResourcesForAdoption(ctx context.Context, orcObject orcObjectPT) (iter.Seq2[*osResourceT, error], bool) {
	resourceSpec := orcObject.Spec.Resource
	if resourceSpec == nil {
		return nil, false
	}

	// TODO(scaffolding) If you need to filter resources on fields that the List() function
	// of gophercloud does not support, it's possible to perform client-side filtering.
	// Check osclients.ResourceFilter

	var rs progress.ReconcileStatus

	var flavorID string
	if resourceSpec.FlavorRef != nil {
		flavor, rs1 := dependency.FetchDependency[*orcv1alpha1.Flavor](
			ctx, actuator.k8sClient, orcObject.Namespace, resourceSpec.FlavorRef, "Flavor",
			orcv1alpha1.IsAvailable,
		)
		rs = rs.WithReconcileStatus(rs1)
		flavorID = ptr.Deref(flavor.Status.ID, "")
	}

	if needsReschedule, _ := rs.NeedsReschedule(); needsReschedule {
		return nil, false
	}

	listOpts := leases.ListOpts{
		Name:        getResourceName(orcObject),
		Description: ptr.Deref(resourceSpec.Description, ""),
		FlavorID:  flavorID,
		// TODO(scaffolding): Add more adoption filters
	}

	return actuator.osClient.ListLeases(ctx, listOpts), true
}

func (actuator leaseActuator) ListOSResourcesForImport(ctx context.Context, obj orcObjectPT, filter filterT) (iter.Seq2[*osResourceT, error], progress.ReconcileStatus) {
	// TODO(scaffolding) If you need to filter resources on fields that the List() function
	// of gophercloud does not support, it's possible to perform client-side filtering.
	// Check osclients.ResourceFilter

	listOpts := leases.ListOpts{
		Name:        string(ptr.Deref(filter.Name, "")),
		Description: string(ptr.Deref(filter.Description, "")),
		// TODO(scaffolding): Add more import filters
	}

	return actuator.osClient.ListLeases(ctx, listOpts), nil
}

func (actuator leaseActuator) CreateResource(ctx context.Context, obj orcObjectPT) (*osResourceT, progress.ReconcileStatus) {
	resource := obj.Spec.Resource

	if resource == nil {
		// Should have been caught by API validation
		return nil, progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Creation requested, but spec.resource is not set"))
	}
	var reconcileStatus progress.ReconcileStatus

	var flavorID string
	if resource.FlavorRef != nil {
		flavor, flavorDepRS := flavorDependency.RequireDependency(
			ctx, actuator.k8sClient, obj, orcv1alpha1.IsAvailable,
		)
		reconcileStatus = reconcileStatus.WithReconcileStatus(flavorDepRS)
		if flavor != nil {
			flavorID = ptr.Deref(flavor.Status.ID, "")
		}
	}
	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return nil, reconcileStatus
	}
	createOpts := leases.CreateOpts{
		Name:        getResourceName(obj),
		Description: ptr.Deref(resource.Description, ""),
		FlavorID:  flavorID,
		// TODO(scaffolding): Add more fields
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

func (actuator leaseActuator) DeleteResource(ctx context.Context, _ orcObjectPT, resource *osResourceT) progress.ReconcileStatus {
	if resource.Status == LeaseStatusDeleting {
		return progress.WaitingOnOpenStack(progress.WaitingOnReady, leaseDeletingPollingPeriod)
	}
	return progress.WrapError(actuator.osClient.DeleteLease(ctx, resource.ID))
}

func (actuator leaseActuator) updateResource(ctx context.Context, obj orcObjectPT, osResource *osResourceT) progress.ReconcileStatus {
	log := ctrl.LoggerFrom(ctx)
	resource := obj.Spec.Resource
	if resource == nil {
		// Should have been caught by API validation
		return progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Update requested, but spec.resource is not set"))
	}

	updateOpts := leases.UpdateOpts{}

	handleNameUpdate(&updateOpts, obj, osResource)
	handleDescriptionUpdate(&updateOpts, resource, osResource)

	// TODO(scaffolding): add handler for all fields supporting mutability

	needsUpdate, err := needsUpdate(updateOpts)
	if err != nil {
		return progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration updating resource: "+err.Error(), err))
	}
	if !needsUpdate {
		log.V(logging.Debug).Info("No changes")
		return nil
	}

	_, err = actuator.osClient.UpdateLease(ctx, osResource.ID, updateOpts)

	if err != nil {
		if !orcerrors.IsRetryable(err) {
			err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration updating resource: "+err.Error(), err)
		}
		return progress.WrapError(err)
	}

	return progress.NeedsRefresh()
}

func needsUpdate(updateOpts leases.UpdateOpts) (bool, error) {
	updateOptsMap, err := updateOpts.ToLeaseUpdateMap()
	if err != nil {
		return false, err
	}

	updateMap, ok := updateOptsMap["lease"].(map[string]any)
	if !ok {
		updateMap = make(map[string]any)
	}

	return len(updateMap) > 0, nil
}

func handleNameUpdate(updateOpts *leases.UpdateOpts, obj orcObjectPT, osResource *osResourceT) {
	name := getResourceName(obj)
	if osResource.Name != name {
		updateOpts.Name = &name
	}
}

func handleDescriptionUpdate(updateOpts *leases.UpdateOpts, resource *resourceSpecT, osResource *osResourceT) {
	description := ptr.Deref(resource.Description, "")
	if osResource.Description != description {
		updateOpts.Description = &description
	}
}

func (actuator leaseActuator) GetResourceReconcilers(ctx context.Context, orcObject orcObjectPT, osResource *osResourceT, controller interfaces.ResourceController) ([]resourceReconciler, progress.ReconcileStatus) {
	return []resourceReconciler{
		actuator.updateResource,
	}, nil
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
