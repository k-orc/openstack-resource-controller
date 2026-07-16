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

package region

import (
	"context"
	"iter"

	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/regions"
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
	osResourceT = regions.Region

	createResourceActuator = interfaces.CreateResourceActuator[orcObjectPT, orcObjectT, filterT, osResourceT]
	deleteResourceActuator = interfaces.DeleteResourceActuator[orcObjectPT, orcObjectT, osResourceT]
	resourceReconciler     = interfaces.ResourceReconciler[orcObjectPT, osResourceT]
	helperFactory          = interfaces.ResourceHelperFactory[orcObjectPT, orcObjectT, resourceSpecT, filterT, osResourceT]
)

type regionActuator struct {
	osClient  osclients.RegionClient
	k8sClient client.Client
}

var _ createResourceActuator = regionActuator{}
var _ deleteResourceActuator = regionActuator{}

func (regionActuator) GetResourceID(osResource *osResourceT) string {
	return osResource.ID
}

func (actuator regionActuator) GetOSResourceByID(ctx context.Context, id string) (*osResourceT, progress.ReconcileStatus) {
	resource, err := actuator.osClient.GetRegion(ctx, id)
	if err != nil {
		return nil, progress.WrapError(err)
	}
	return resource, nil
}

func (actuator regionActuator) ListOSResourcesForAdoption(ctx context.Context, orcObject orcObjectPT) (iter.Seq2[*osResourceT, error], bool) {
	resourceSpec := orcObject.Spec.Resource
	if resourceSpec == nil {
		return nil, false
	}

	// Filter by the expected resource name to avoid adopting wrong regions.
	// The OpenStack Region API does not support server-side filtering by name/id,
	// so we must use client-side filtering.
	filters := []osclients.ResourceFilter[osResourceT]{
		func(r *regions.Region) bool {
			return r.ID == string(orcObject.Spec.Resource.Name)
		},
	}

	if resourceSpec.Description != nil {
		filters = append(filters, func(r *regions.Region) bool {
			return r.Description == *resourceSpec.Description
		})
	}

	// TODO:
	// listOpts := regions.ListOpts{
	// ParentRegionID: ptr.Deref(resourceSpec.ParentRegionID),
	// }

	return actuator.listOSResources(ctx, filters, regions.ListOpts{}), true
}

func (actuator regionActuator) ListOSResourcesForImport(ctx context.Context, obj orcObjectPT, filter filterT) (iter.Seq2[*osResourceT, error], progress.ReconcileStatus) {
	// The OpenStack Region API does not support server-side filtering by name/id,
	// so we must use client-side filtering.
	var filters []osclients.ResourceFilter[osResourceT]

	if filter.Name != nil {
		filters = append(filters, func(r *regions.Region) bool {
			return r.ID == string(*filter.Name)
		})
	}

	if filter.Description != nil {
		filters = append(filters, func(r *regions.Region) bool {
			return r.Description == *filter.Description
		})
	}

	// TODO:
	// listOpts := regions.ListOpts{
	// ParentRegionID: ptr.Deref(resourceSpec.ParentRegionID),
	// }

	return actuator.listOSResources(ctx, filters, regions.ListOpts{}), nil
}

func (actuator regionActuator) listOSResources(ctx context.Context, filters []osclients.ResourceFilter[osResourceT], listOpts regions.ListOptsBuilder) iter.Seq2[*osResourceT, error] {
	regions := actuator.osClient.ListRegions(ctx, listOpts)
	return osclients.Filter(regions, filters...)
}

func (actuator regionActuator) CreateResource(ctx context.Context, obj orcObjectPT) (*osResourceT, progress.ReconcileStatus) {
	resource := obj.Spec.Resource

	if resource == nil {
		// Should have been caught by API validation
		return nil, progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Creation requested, but spec.resource is not set"))
	}
	createOpts := regions.CreateOpts{
		ID:          string(resource.Name),
		Description: ptr.Deref(resource.Description, ""),
		// TODO:
		// ParentRegionID: ptr.Deref(resource.ParentRegionID),
	}

	osResource, err := actuator.osClient.CreateRegion(ctx, createOpts)
	if err != nil {
		if !orcerrors.IsRetryable(err) {
			err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration creating resource: "+err.Error(), err)
		}
		return nil, progress.WrapError(err)
	}

	return osResource, nil
}

func (actuator regionActuator) DeleteResource(ctx context.Context, _ orcObjectPT, resource *osResourceT) progress.ReconcileStatus {
	return progress.WrapError(actuator.osClient.DeleteRegion(ctx, resource.ID))
}

func (actuator regionActuator) updateResource(ctx context.Context, obj orcObjectPT, osResource *osResourceT) progress.ReconcileStatus {
	log := ctrl.LoggerFrom(ctx)
	resource := obj.Spec.Resource
	if resource == nil {
		// Should have been caught by API validation
		return progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Update requested, but spec.resource is not set"))
	}

	updateOpts := regions.UpdateOpts{}

	handleDescriptionUpdate(&updateOpts, resource, osResource)

	needsUpdate, err := needsUpdate(updateOpts)
	if err != nil {
		return progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration updating resource: "+err.Error(), err))
	}
	if !needsUpdate {
		log.V(logging.Debug).Info("No changes")
		return nil
	}

	_, err = actuator.osClient.UpdateRegion(ctx, osResource.ID, updateOpts)

	if err != nil {
		if !orcerrors.IsRetryable(err) {
			err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration updating resource: "+err.Error(), err)
		}
		return progress.WrapError(err)
	}

	return progress.NeedsRefresh()
}

func needsUpdate(updateOpts regions.UpdateOpts) (bool, error) {
	updateOptsMap, err := updateOpts.ToRegionUpdateMap()
	if err != nil {
		return false, err
	}

	updateMap, ok := updateOptsMap["region"].(map[string]any)
	if !ok {
		updateMap = make(map[string]any)
	}

	return len(updateMap) > 0, nil
}

func handleDescriptionUpdate(updateOpts *regions.UpdateOpts, resource *resourceSpecT, osResource *osResourceT) {
	description := ptr.Deref(resource.Description, "")
	if osResource.Description != description {
		updateOpts.Description = &description
	}
}

func (actuator regionActuator) GetResourceReconcilers(ctx context.Context, orcObject orcObjectPT, osResource *osResourceT, controller interfaces.ResourceController) ([]resourceReconciler, progress.ReconcileStatus) {
	return []resourceReconciler{
		actuator.updateResource,
	}, nil
}

type regionHelperFactory struct{}

var _ helperFactory = regionHelperFactory{}

func newActuator(ctx context.Context, orcObject *orcv1alpha1.Region, controller interfaces.ResourceController) (regionActuator, progress.ReconcileStatus) {
	log := ctrl.LoggerFrom(ctx)

	// Ensure credential secrets exist and have our finalizer
	_, reconcileStatus := credentialsDependency.GetDependencies(ctx, controller.GetK8sClient(), orcObject, func(*corev1.Secret) bool { return true })
	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return regionActuator{}, reconcileStatus
	}

	clientScope, err := controller.GetScopeFactory().NewClientScopeFromObject(ctx, controller.GetK8sClient(), log, orcObject)
	if err != nil {
		return regionActuator{}, progress.WrapError(err)
	}
	osClient, err := clientScope.NewRegionClient()
	if err != nil {
		return regionActuator{}, progress.WrapError(err)
	}

	return regionActuator{
		osClient:  osClient,
		k8sClient: controller.GetK8sClient(),
	}, nil
}

func (regionHelperFactory) NewAPIObjectAdapter(obj orcObjectPT) adapterI {
	return regionAdapter{obj}
}

func (regionHelperFactory) NewCreateActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (createResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, orcObject, controller)
}

func (regionHelperFactory) NewDeleteActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (deleteResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, orcObject, controller)
}
