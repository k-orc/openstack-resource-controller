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

package registeredlimit

import (
	"context"
	"iter"

	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/registeredlimits"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/progress"
	"github.com/k-orc/openstack-resource-controller/v2/internal/logging"
	"github.com/k-orc/openstack-resource-controller/v2/internal/osclients"
	orcerrors "github.com/k-orc/openstack-resource-controller/v2/internal/util/errors"
)

// OpenStack resource types
type (
	osResourceT = registeredlimits.RegisteredLimit

	createResourceActuator = interfaces.CreateResourceActuator[orcObjectPT, orcObjectT, filterT, osResourceT]
	deleteResourceActuator = interfaces.DeleteResourceActuator[orcObjectPT, orcObjectT, osResourceT]
	resourceReconciler     = interfaces.ResourceReconciler[orcObjectPT, osResourceT]
	helperFactory          = interfaces.ResourceHelperFactory[orcObjectPT, orcObjectT, resourceSpecT, filterT, osResourceT]
)

type registeredlimitActuator struct {
	osClient  osclients.RegisteredLimitClient
	k8sClient client.Client
}

var _ createResourceActuator = registeredlimitActuator{}
var _ deleteResourceActuator = registeredlimitActuator{}

func (registeredlimitActuator) GetResourceID(osResource *osResourceT) string {
	return osResource.ID
}

func (actuator registeredlimitActuator) GetOSResourceByID(ctx context.Context, id string) (*osResourceT, progress.ReconcileStatus) {
	resource, err := actuator.osClient.GetRegisteredLimit(ctx, id)
	if err != nil {
		return nil, progress.WrapError(err)
	}
	return resource, nil
}

func (actuator registeredlimitActuator) ListOSResourcesForAdoption(ctx context.Context, orcObject orcObjectPT) (iter.Seq2[*osResourceT, error], bool) {
	resourceSpec := orcObject.Spec.Resource
	if resourceSpec == nil {
		return nil, false
	}

	// TODO(scaffolding) If you need to filter resources on fields that the List() function
	// of gophercloud does not support, it's possible to perform client-side filtering.
	// Check osclients.ResourceFilter

	listOpts := registeredlimits.ListOpts{
		// TODO(scaffolding): Add import filters
	}

	return actuator.osClient.ListRegisteredLimits(ctx, listOpts), true
}

func (actuator registeredlimitActuator) ListOSResourcesForImport(ctx context.Context, obj orcObjectPT, filter filterT) (iter.Seq2[*osResourceT, error], progress.ReconcileStatus) {
	// TODO(scaffolding) If you need to filter resources on fields that the List() function
	// of gophercloud does not support, it's possible to perform client-side filtering.
	// Check osclients.ResourceFilter

	listOpts := registeredlimits.ListOpts{
		// TODO(scaffolding): Add import filters
	}

	return actuator.osClient.ListRegisteredLimits(ctx, listOpts), nil
}

func (actuator registeredlimitActuator) CreateResource(ctx context.Context, obj orcObjectPT) (*osResourceT, progress.ReconcileStatus) {
	resource := obj.Spec.Resource

	if resource == nil {
		// Should have been caught by API validation
		return nil, progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Creation requested, but spec.resource is not set"))
	}
	var reconcileStatus progress.ReconcileStatus

	var serviceID string
	service, serviceDepRS := serviceDependency.GetDependency(
		ctx, actuator.k8sClient, obj, orcv1alpha1.IsAvailable,
	)
	reconcileStatus = reconcileStatus.WithReconcileStatus(serviceDepRS)
	if service != nil {
		serviceID = ptr.Deref(service.Status.ID, "")
	}
	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return nil, reconcileStatus
	}
	createOpts := registeredlimits.CreateOpts{
		Description: ptr.Deref(resource.Description, ""),
		ServiceID:   serviceID,
		// TODO(scaffolding): Add more fields
	}
	batchCreateOpts := registeredlimits.BatchCreateOpts{
		createOpts,
	}

	osResource, err := actuator.osClient.CreateRegisteredLimit(ctx, batchCreateOpts)
	if err != nil {
		if !orcerrors.IsRetryable(err) {
			err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration creating resource: "+err.Error(), err)
		}
		return nil, progress.WrapError(err)
	}

	return osResource, nil
}

func (actuator registeredlimitActuator) DeleteResource(ctx context.Context, _ orcObjectPT, resource *osResourceT) progress.ReconcileStatus {
	return progress.WrapError(actuator.osClient.DeleteRegisteredLimit(ctx, resource.ID))
}

func (actuator registeredlimitActuator) updateResource(ctx context.Context, obj orcObjectPT, osResource *osResourceT) progress.ReconcileStatus {
	log := ctrl.LoggerFrom(ctx)
	resource := obj.Spec.Resource
	if resource == nil {
		// Should have been caught by API validation
		return progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Update requested, but spec.resource is not set"))
	}

	updateOpts := registeredlimits.UpdateOpts{}

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

	_, err = actuator.osClient.UpdateRegisteredLimit(ctx, osResource.ID, updateOpts)

	if err != nil {
		if !orcerrors.IsRetryable(err) {
			err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration updating resource: "+err.Error(), err)
		}
		return progress.WrapError(err)
	}

	return progress.NeedsRefresh()
}

func needsUpdate(updateOpts registeredlimits.UpdateOpts) (bool, error) {
	updateOptsMap, err := updateOpts.ToRegisteredLimitUpdateMap()
	if err != nil {
		return false, err
	}

	updateMap, ok := updateOptsMap["registered_limits"].(map[string]any)
	if !ok {
		updateMap = make(map[string]any)
	}

	return len(updateMap) > 0, nil
}

func handleDescriptionUpdate(updateOpts *registeredlimits.UpdateOpts, resource *resourceSpecT, osResource *osResourceT) {
	description := ptr.Deref(resource.Description, "")
	if osResource.Description != description {
		updateOpts.Description = &description
	}
}

func (actuator registeredlimitActuator) GetResourceReconcilers(ctx context.Context, orcObject orcObjectPT, osResource *osResourceT, controller interfaces.ResourceController) ([]resourceReconciler, progress.ReconcileStatus) {
	return []resourceReconciler{
		actuator.updateResource,
	}, nil
}

type registeredlimitHelperFactory struct{}

var _ helperFactory = registeredlimitHelperFactory{}

func newActuator(ctx context.Context, orcObject *orcv1alpha1.RegisteredLimit, controller interfaces.ResourceController) (registeredlimitActuator, progress.ReconcileStatus) {
	log := ctrl.LoggerFrom(ctx)

	// Ensure credential secrets exist and have our finalizer
	_, reconcileStatus := credentialsDependency.GetDependencies(ctx, controller.GetK8sClient(), orcObject, func(*corev1.Secret) bool { return true })
	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return registeredlimitActuator{}, reconcileStatus
	}

	clientScope, err := controller.GetScopeFactory().NewClientScopeFromObject(ctx, controller.GetK8sClient(), log, orcObject)
	if err != nil {
		return registeredlimitActuator{}, progress.WrapError(err)
	}
	osClient, err := clientScope.NewRegisteredLimitClient()
	if err != nil {
		return registeredlimitActuator{}, progress.WrapError(err)
	}

	return registeredlimitActuator{
		osClient:  osClient,
		k8sClient: controller.GetK8sClient(),
	}, nil
}

func (registeredlimitHelperFactory) NewAPIObjectAdapter(obj orcObjectPT) adapterI {
	return registeredlimitAdapter{obj}
}

func (registeredlimitHelperFactory) NewCreateActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (createResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, orcObject, controller)
}

func (registeredlimitHelperFactory) NewDeleteActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (deleteResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, orcObject, controller)
}
