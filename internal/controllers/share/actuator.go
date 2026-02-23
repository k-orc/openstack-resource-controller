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

package share

import (
	"context"
	"iter"
	"time"

	"github.com/gophercloud/gophercloud/v2/openstack/sharedfilesystems/v2/shares"
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
	osResourceT = shares.Share

	createResourceActuator = interfaces.CreateResourceActuator[orcObjectPT, orcObjectT, filterT, osResourceT]
	deleteResourceActuator = interfaces.DeleteResourceActuator[orcObjectPT, orcObjectT, osResourceT]
	resourceReconciler     = interfaces.ResourceReconciler[orcObjectPT, osResourceT]
	helperFactory          = interfaces.ResourceHelperFactory[orcObjectPT, orcObjectT, resourceSpecT, filterT, osResourceT]
)

// The frequency to poll when waiting for the resource to become available
const shareAvailablePollingPeriod = 15 * time.Second

// The frequency to poll when waiting for the resource to be deleted
const shareDeletingPollingPeriod = 15 * time.Second

type shareActuator struct {
	osClient  osclients.ShareClient
	k8sClient client.Client
}

var _ createResourceActuator = shareActuator{}
var _ deleteResourceActuator = shareActuator{}

func (shareActuator) GetResourceID(osResource *osResourceT) string {
	return osResource.ID
}

func (actuator shareActuator) GetOSResourceByID(ctx context.Context, id string) (*osResourceT, progress.ReconcileStatus) {
	resource, err := actuator.osClient.GetShare(ctx, id)
	if err != nil {
		return nil, progress.WrapError(err)
	}
	return resource, nil
}

func (actuator shareActuator) ListOSResourcesForAdoption(ctx context.Context, orcObject orcObjectPT) (iter.Seq2[*osResourceT, error], bool) {
	resourceSpec := orcObject.Spec.Resource
	if resourceSpec == nil {
		return nil, false
	}

	listOpts := shares.ListOpts{
		Name: getResourceName(orcObject),
	}

	return actuator.osClient.ListShares(ctx, listOpts), true
}

func (actuator shareActuator) ListOSResourcesForImport(ctx context.Context, obj orcObjectPT, filter filterT) (iter.Seq2[*osResourceT, error], progress.ReconcileStatus) {
	var filters []osclients.ResourceFilter[osResourceT]

	// NOTE: The API doesn't support filtering by description or shareProto
	// we'll have to do it client-side.
	if filter.Description != nil {
		filters = append(filters, func(s *shares.Share) bool {
			return s.Description == *filter.Description
		})
	}
	if filter.ShareProto != nil {
		filters = append(filters, func(s *shares.Share) bool {
			return s.ShareProto == *filter.ShareProto
		})
	}

	listOpts := shares.ListOpts{
		Name:   string(ptr.Deref(filter.Name, "")),
		Status: ptr.Deref(filter.Status, ""),
	}

	if filter.IsPublic != nil {
		listOpts.IsPublic = filter.IsPublic
	}

	return actuator.listOSResources(ctx, filters, listOpts), nil
}

func (actuator shareActuator) listOSResources(ctx context.Context, filters []osclients.ResourceFilter[osResourceT], listOpts shares.ListOptsBuilder) iter.Seq2[*shares.Share, error] {
	shares := actuator.osClient.ListShares(ctx, listOpts)
	return osclients.Filter(shares, filters...)
}

func (actuator shareActuator) CreateResource(ctx context.Context, obj orcObjectPT) (*osResourceT, progress.ReconcileStatus) {
	resource := obj.Spec.Resource

	if resource == nil {
		// Should have been caught by API validation
		return nil, progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Creation requested, but spec.resource is not set"))
	}

	metadata := make(map[string]string)
	for _, m := range resource.Metadata {
		metadata[m.Name] = m.Value
	}

	createOpts := shares.CreateOpts{
		Name:             getResourceName(obj),
		Description:      ptr.Deref(resource.Description, ""),
		Size:             int(resource.Size),
		ShareProto:       resource.ShareProto,
		AvailabilityZone: resource.AvailabilityZone,
		Metadata:         metadata,
		IsPublic:         resource.IsPublic,
	}

	osResource, err := actuator.osClient.CreateShare(ctx, createOpts)
	if err != nil {
		// We should require the spec to be updated before retrying a create which returned a conflict
		if !orcerrors.IsRetryable(err) {
			err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration creating resource: "+err.Error(), err)
		}
		return nil, progress.WrapError(err)
	}

	return osResource, nil
}

func (actuator shareActuator) DeleteResource(ctx context.Context, _ orcObjectPT, resource *osResourceT) progress.ReconcileStatus {
	if resource.Status == ShareStatusDeleting {
		return progress.WaitingOnOpenStack(progress.WaitingOnReady, shareDeletingPollingPeriod)
	}
	return progress.WrapError(actuator.osClient.DeleteShare(ctx, resource.ID))
}

func (actuator shareActuator) updateResource(ctx context.Context, obj orcObjectPT, osResource *osResourceT) progress.ReconcileStatus {
	log := ctrl.LoggerFrom(ctx)
	resource := obj.Spec.Resource
	if resource == nil {
		// Should have been caught by API validation
		return progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Update requested, but spec.resource is not set"))
	}

	updateOpts := shares.UpdateOpts{}

	handleIsPublicUpdate(&updateOpts, resource, osResource)

	needsUpdate, err := needsUpdate(updateOpts)
	if err != nil {
		return progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration updating resource: "+err.Error(), err))
	}
	if !needsUpdate {
		log.V(logging.Debug).Info("No changes")
		return nil
	}

	_, err = actuator.osClient.UpdateShare(ctx, osResource.ID, updateOpts)

	// We should require the spec to be updated before retrying an update which returned a conflict
	if orcerrors.IsConflict(err) {
		err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration updating resource: "+err.Error(), err)
	}

	if err != nil {
		return progress.WrapError(err)
	}

	return progress.NeedsRefresh()
}

func needsUpdate(updateOpts shares.UpdateOpts) (bool, error) {
	updateOptsMap, err := updateOpts.ToShareUpdateMap()
	if err != nil {
		return false, err
	}

	updateMap, ok := updateOptsMap["share"].(map[string]any)
	if !ok {
		updateMap = make(map[string]any)
	}

	return len(updateMap) > 0, nil
}

// NOTE: Manila API doesn't support updating name or description
// func handleNameUpdate(updateOpts *shares.UpdateOpts, obj orcObjectPT, osResource *osResourceT) {
// 	name := getResourceName(obj)
// 	if osResource.Name != name {
// 		updateOpts.Name = &name
// 	}
// }

// func handleDescriptionUpdate(updateOpts *shares.UpdateOpts, resource *resourceSpecT, osResource *osResourceT) {
// 	description := ptr.Deref(resource.Description, "")
// 	if osResource.Description != description {
// 		updateOpts.Description = &description
// 	}
// }

func handleIsPublicUpdate(updateOpts *shares.UpdateOpts, resource *resourceSpecT, osResource *osResourceT) {
	isPublic := ptr.Deref(resource.IsPublic, false)
	if osResource.IsPublic != isPublic {
		updateOpts.IsPublic = &isPublic
	}
}

func (actuator shareActuator) GetResourceReconcilers(ctx context.Context, orcObject orcObjectPT, osResource *osResourceT, controller interfaces.ResourceController) ([]resourceReconciler, progress.ReconcileStatus) {
	return []resourceReconciler{
		actuator.updateResource,
	}, nil
}

type shareHelperFactory struct{}

var _ helperFactory = shareHelperFactory{}

func newActuator(ctx context.Context, orcObject *orcv1alpha1.Share, controller interfaces.ResourceController) (shareActuator, progress.ReconcileStatus) {
	log := ctrl.LoggerFrom(ctx)

	// Ensure credential secrets exist and have our finalizer
	_, reconcileStatus := credentialsDependency.GetDependencies(ctx, controller.GetK8sClient(), orcObject, func(*corev1.Secret) bool { return true })
	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return shareActuator{}, reconcileStatus
	}

	clientScope, err := controller.GetScopeFactory().NewClientScopeFromObject(ctx, controller.GetK8sClient(), log, orcObject)
	if err != nil {
		return shareActuator{}, progress.WrapError(err)
	}
	osClient, err := clientScope.NewShareClient()
	if err != nil {
		return shareActuator{}, progress.WrapError(err)
	}

	return shareActuator{
		osClient:  osClient,
		k8sClient: controller.GetK8sClient(),
	}, nil
}

func (shareHelperFactory) NewAPIObjectAdapter(obj orcObjectPT) adapterI {
	return shareAdapter{obj}
}

func (shareHelperFactory) NewCreateActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (createResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, orcObject, controller)
}

func (shareHelperFactory) NewDeleteActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (deleteResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, orcObject, controller)
}
