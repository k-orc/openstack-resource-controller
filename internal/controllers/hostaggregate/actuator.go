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

package hostaggregate

import (
	"context"
	"iter"
	"strconv"

	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/aggregates"
	corev1 "k8s.io/api/core/v1"
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
	osResourceT = aggregates.Aggregate

	createResourceActuator = interfaces.CreateResourceActuator[orcObjectPT, orcObjectT, filterT, osResourceT]
	deleteResourceActuator = interfaces.DeleteResourceActuator[orcObjectPT, orcObjectT, osResourceT]
	resourceReconciler     = interfaces.ResourceReconciler[orcObjectPT, osResourceT]
	helperFactory          = interfaces.ResourceHelperFactory[orcObjectPT, orcObjectT, resourceSpecT, filterT, osResourceT]
)

type hostaggregateActuator struct {
	osClient  osclients.HostAggregateClient
	k8sClient client.Client
}

var _ createResourceActuator = hostaggregateActuator{}
var _ deleteResourceActuator = hostaggregateActuator{}

// TODO(stephenfin): I suspect we need to change the interface since Nova expects integer IDs
func (hostaggregateActuator) GetResourceID(osResource *osResourceT) string {
	return strconv.Itoa(osResource.ID)
}

func (actuator hostaggregateActuator) GetOSResourceByID(ctx context.Context, id string) (*osResourceT, progress.ReconcileStatus) {
	iid, err := strconv.Atoi(id)
	if err != nil {
		return nil, progress.WrapError(err)
	}
	resource, err := actuator.osClient.GetHostAggregate(ctx, iid)
	if err != nil {
		return nil, progress.WrapError(err)
	}
	return resource, nil
}

func (actuator hostaggregateActuator) ListOSResourcesForAdoption(ctx context.Context, orcObject orcObjectPT) (iter.Seq2[*osResourceT, error], bool) {
	resourceSpec := orcObject.Spec.Resource
	if resourceSpec == nil {
		return nil, false
	}

	var filters []osclients.ResourceFilter[osResourceT]

	// NOTE: The API doesn't allow filtering by name or description, we'll have to do it client-side.
	filters = append(filters,
		func(f *aggregates.Aggregate) bool {
			name := getResourceName(orcObject)
			// Compare non-pointer values
			return f.Name == name
		},
	)

	return actuator.listOSResources(ctx, filters), true
}

func (actuator hostaggregateActuator) ListOSResourcesForImport(ctx context.Context, obj orcObjectPT, filter filterT) (iter.Seq2[*osResourceT, error], progress.ReconcileStatus) {
	var filters []osclients.ResourceFilter[osResourceT]

	// NOTE: The API doesn't allow filtering by name or description, we'll have to do it client-side.
	if filter.Name != nil {
		filters = append(filters, func(f *aggregates.Aggregate) bool {
			return f.Name == string(*filter.Name)
		})
	}

	return actuator.listOSResources(ctx, filters), nil
}

func (actuator hostaggregateActuator) listOSResources(ctx context.Context, filters []osclients.ResourceFilter[osResourceT]) iter.Seq2[*aggregates.Aggregate, error] {
	volumetypes := actuator.osClient.ListHostAggregates(ctx)
	return osclients.Filter(volumetypes, filters...)
}

func (actuator hostaggregateActuator) CreateResource(ctx context.Context, obj orcObjectPT) (*osResourceT, progress.ReconcileStatus) {
	resource := obj.Spec.Resource

	if resource == nil {
		// Should have been caught by API validation
		return nil, progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Creation requested, but spec.resource is not set"))
	}
	createOpts := aggregates.CreateOpts{
		Name: getResourceName(obj),
	}

	osResource, err := actuator.osClient.CreateHostAggregate(ctx, createOpts)
	if err != nil {
		// We should require the spec to be updated before retrying a create which returned a conflict
		if !orcerrors.IsRetryable(err) {
			err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration creating resource: "+err.Error(), err)
		}
		return nil, progress.WrapError(err)
	}

	return osResource, nil
}

func (actuator hostaggregateActuator) DeleteResource(ctx context.Context, _ orcObjectPT, resource *osResourceT) progress.ReconcileStatus {
	return progress.WrapError(actuator.osClient.DeleteHostAggregate(ctx, resource.ID))
}

func (actuator hostaggregateActuator) updateResource(ctx context.Context, obj orcObjectPT, osResource *osResourceT) progress.ReconcileStatus {
	log := ctrl.LoggerFrom(ctx)
	resource := obj.Spec.Resource
	if resource == nil {
		// Should have been caught by API validation
		return progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Update requested, but spec.resource is not set"))
	}

	updateOpts := aggregates.UpdateOpts{}

	handleNameUpdate(&updateOpts, obj, osResource)

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

	_, err = actuator.osClient.UpdateHostAggregate(ctx, osResource.ID, updateOpts)

	// We should require the spec to be updated before retrying an update which returned a conflict
	if orcerrors.IsConflict(err) {
		err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration updating resource: "+err.Error(), err)
	}

	if err != nil {
		return progress.WrapError(err)
	}

	return progress.NeedsRefresh()
}

func needsUpdate(updateOpts aggregates.UpdateOpts) (bool, error) {
	updateOptsMap, err := updateOpts.ToAggregatesUpdateMap()
	if err != nil {
		return false, err
	}

	updateMap, ok := updateOptsMap["aggregate"].(map[string]any)
	if !ok {
		updateMap = make(map[string]any)
	}

	return len(updateMap) > 0, nil
}

func handleNameUpdate(updateOpts *aggregates.UpdateOpts, obj orcObjectPT, osResource *osResourceT) {
	name := getResourceName(obj)
	if osResource.Name != name {
		updateOpts.Name = name
	}
}

func (actuator hostaggregateActuator) GetResourceReconcilers(ctx context.Context, orcObject orcObjectPT, osResource *osResourceT, controller interfaces.ResourceController) ([]resourceReconciler, progress.ReconcileStatus) {
	return []resourceReconciler{
		actuator.updateResource,
	}, nil
}

type hostaggregateHelperFactory struct{}

var _ helperFactory = hostaggregateHelperFactory{}

func newActuator(ctx context.Context, orcObject *orcv1alpha1.HostAggregate, controller interfaces.ResourceController) (hostaggregateActuator, progress.ReconcileStatus) {
	log := ctrl.LoggerFrom(ctx)

	// Ensure credential secrets exist and have our finalizer
	_, reconcileStatus := credentialsDependency.GetDependencies(ctx, controller.GetK8sClient(), orcObject, func(*corev1.Secret) bool { return true })
	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return hostaggregateActuator{}, reconcileStatus
	}

	clientScope, err := controller.GetScopeFactory().NewClientScopeFromObject(ctx, controller.GetK8sClient(), log, orcObject)
	if err != nil {
		return hostaggregateActuator{}, progress.WrapError(err)
	}
	osClient, err := clientScope.NewHostAggregateClient()
	if err != nil {
		return hostaggregateActuator{}, progress.WrapError(err)
	}

	return hostaggregateActuator{
		osClient:  osClient,
		k8sClient: controller.GetK8sClient(),
	}, nil
}

func (hostaggregateHelperFactory) NewAPIObjectAdapter(obj orcObjectPT) adapterI {
	return hostaggregateAdapter{obj}
}

func (hostaggregateHelperFactory) NewCreateActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (createResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, orcObject, controller)
}

func (hostaggregateHelperFactory) NewDeleteActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (deleteResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, orcObject, controller)
}
