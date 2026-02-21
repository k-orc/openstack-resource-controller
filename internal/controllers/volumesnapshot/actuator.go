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

package volumesnapshot

import (
	"context"
	"iter"
	"time"

	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/snapshots"
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
	osResourceT = snapshots.Snapshot

	createResourceActuator = interfaces.CreateResourceActuator[orcObjectPT, orcObjectT, filterT, osResourceT]
	deleteResourceActuator = interfaces.DeleteResourceActuator[orcObjectPT, orcObjectT, osResourceT]
	resourceReconciler     = interfaces.ResourceReconciler[orcObjectPT, osResourceT]
	helperFactory          = interfaces.ResourceHelperFactory[orcObjectPT, orcObjectT, resourceSpecT, filterT, osResourceT]
)

const (
	volumesnapshotAvailablePollingPeriod = 15 * time.Second
	volumesnapshotDeletingPollingPeriod  = 15 * time.Second
)

type volumesnapshotActuator struct {
	osClient  osclients.VolumeSnapshotClient
	k8sClient client.Client
}

var _ createResourceActuator = volumesnapshotActuator{}
var _ deleteResourceActuator = volumesnapshotActuator{}

func (volumesnapshotActuator) GetResourceID(osResource *osResourceT) string {
	return osResource.ID
}

func (actuator volumesnapshotActuator) GetOSResourceByID(ctx context.Context, id string) (*osResourceT, progress.ReconcileStatus) {
	resource, err := actuator.osClient.GetVolumeSnapshot(ctx, id)
	if err != nil {
		return nil, progress.WrapError(err)
	}
	return resource, nil
}

func (actuator volumesnapshotActuator) ListOSResourcesForAdoption(ctx context.Context, orcObject orcObjectPT) (iter.Seq2[*osResourceT, error], bool) {
	resourceSpec := orcObject.Spec.Resource
	if resourceSpec == nil {
		return nil, false
	}

	var filters []osclients.ResourceFilter[osResourceT]

	if resourceSpec.Description != nil {
		filters = append(filters, func(s *snapshots.Snapshot) bool {
			return s.Description == *resourceSpec.Description
		})
	}

	listOpts := snapshots.ListOpts{
		Name: getResourceName(orcObject),
	}

	return actuator.listOSResources(ctx, filters, listOpts), true
}

func (actuator volumesnapshotActuator) ListOSResourcesForImport(ctx context.Context, obj orcObjectPT, filter filterT) (iter.Seq2[*osResourceT, error], progress.ReconcileStatus) {
	var filters []osclients.ResourceFilter[osResourceT]

	if filter.Description != nil {
		filters = append(filters, func(s *snapshots.Snapshot) bool {
			return s.Description == *filter.Description
		})
	}

	listOpts := snapshots.ListOpts{
		Name:     string(ptr.Deref(filter.Name, "")),
		Status:   ptr.Deref(filter.Status, ""),
		VolumeID: ptr.Deref(filter.VolumeID, ""),
	}

	return actuator.listOSResources(ctx, filters, listOpts), nil
}

func (actuator volumesnapshotActuator) listOSResources(ctx context.Context, filters []osclients.ResourceFilter[osResourceT], listOpts snapshots.ListOptsBuilder) iter.Seq2[*snapshots.Snapshot, error] {
	results := actuator.osClient.ListVolumeSnapshots(ctx, listOpts)
	return osclients.Filter(results, filters...)
}

func (actuator volumesnapshotActuator) CreateResource(ctx context.Context, obj orcObjectPT) (*osResourceT, progress.ReconcileStatus) {
	resource := obj.Spec.Resource

	if resource == nil {
		// Should have been caught by API validation
		return nil, progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Creation requested, but spec.resource is not set"))
	}
	var reconcileStatus progress.ReconcileStatus

	var volumeID string
	volume, volumeDepRS := volumeDependency.GetDependency(
		ctx, actuator.k8sClient, obj, func(dep *orcv1alpha1.Volume) bool {
			return orcv1alpha1.IsAvailable(dep) && dep.Status.ID != nil
		},
	)
	reconcileStatus = reconcileStatus.WithReconcileStatus(volumeDepRS)
	if volume != nil {
		volumeID = ptr.Deref(volume.Status.ID, "")
	}
	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return nil, reconcileStatus
	}
	metadata := make(map[string]string)
	for _, m := range resource.Metadata {
		metadata[m.Name] = m.Value
	}

	createOpts := snapshots.CreateOpts{
		Name:        getResourceName(obj),
		Description: ptr.Deref(resource.Description, ""),
		VolumeID:    volumeID,
		Force:       ptr.Deref(resource.Force, false),
		Metadata:    metadata,
	}

	osResource, err := actuator.osClient.CreateVolumeSnapshot(ctx, createOpts)
	if err != nil {
		// We should require the spec to be updated before retrying a create which returned a conflict
		if !orcerrors.IsRetryable(err) {
			err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration creating resource: "+err.Error(), err)
		}
		return nil, progress.WrapError(err)
	}

	return osResource, nil
}

func (actuator volumesnapshotActuator) DeleteResource(ctx context.Context, _ orcObjectPT, resource *osResourceT) progress.ReconcileStatus {
	if resource.Status == SnapshotStatusDeleting {
		return progress.WaitingOnOpenStack(progress.WaitingOnReady, volumesnapshotDeletingPollingPeriod)
	}
	return progress.WrapError(actuator.osClient.DeleteVolumeSnapshot(ctx, resource.ID))
}

func (actuator volumesnapshotActuator) updateResource(ctx context.Context, obj orcObjectPT, osResource *osResourceT) progress.ReconcileStatus {
	log := ctrl.LoggerFrom(ctx)
	resource := obj.Spec.Resource
	if resource == nil {
		// Should have been caught by API validation
		return progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Update requested, but spec.resource is not set"))
	}

	updateOpts := snapshots.UpdateOpts{}

	handleNameUpdate(&updateOpts, obj, osResource)
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

	_, err = actuator.osClient.UpdateVolumeSnapshot(ctx, osResource.ID, updateOpts)

	// We should require the spec to be updated before retrying an update which returned a conflict
	if orcerrors.IsConflict(err) {
		err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration updating resource: "+err.Error(), err)
	}

	if err != nil {
		return progress.WrapError(err)
	}

	return progress.NeedsRefresh()
}

func needsUpdate(updateOpts snapshots.UpdateOpts) (bool, error) {
	updateOptsMap, err := updateOpts.ToSnapshotUpdateMap()
	if err != nil {
		return false, err
	}

	updateMap, ok := updateOptsMap["snapshot"].(map[string]any)
	if !ok {
		updateMap = make(map[string]any)
	}

	return len(updateMap) > 0, nil
}

func handleNameUpdate(updateOpts *snapshots.UpdateOpts, obj orcObjectPT, osResource *osResourceT) {
	name := getResourceName(obj)
	if osResource.Name != name {
		updateOpts.Name = &name
	}
}

func handleDescriptionUpdate(updateOpts *snapshots.UpdateOpts, resource *resourceSpecT, osResource *osResourceT) {
	description := ptr.Deref(resource.Description, "")
	if osResource.Description != description {
		updateOpts.Description = &description
	}
}

func (actuator volumesnapshotActuator) GetResourceReconcilers(ctx context.Context, orcObject orcObjectPT, osResource *osResourceT, controller interfaces.ResourceController) ([]resourceReconciler, progress.ReconcileStatus) {
	return []resourceReconciler{
		actuator.updateResource,
	}, nil
}

type volumesnapshotHelperFactory struct{}

var _ helperFactory = volumesnapshotHelperFactory{}

func newActuator(ctx context.Context, orcObject *orcv1alpha1.VolumeSnapshot, controller interfaces.ResourceController) (volumesnapshotActuator, progress.ReconcileStatus) {
	log := ctrl.LoggerFrom(ctx)

	// Ensure credential secrets exist and have our finalizer
	_, reconcileStatus := credentialsDependency.GetDependencies(ctx, controller.GetK8sClient(), orcObject, func(*corev1.Secret) bool { return true })
	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return volumesnapshotActuator{}, reconcileStatus
	}

	clientScope, err := controller.GetScopeFactory().NewClientScopeFromObject(ctx, controller.GetK8sClient(), log, orcObject)
	if err != nil {
		return volumesnapshotActuator{}, progress.WrapError(err)
	}
	osClient, err := clientScope.NewVolumeSnapshotClient()
	if err != nil {
		return volumesnapshotActuator{}, progress.WrapError(err)
	}

	return volumesnapshotActuator{
		osClient:  osClient,
		k8sClient: controller.GetK8sClient(),
	}, nil
}

func (volumesnapshotHelperFactory) NewAPIObjectAdapter(obj orcObjectPT) adapterI {
	return volumesnapshotAdapter{obj}
}

func (volumesnapshotHelperFactory) NewCreateActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (createResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, orcObject, controller)
}

func (volumesnapshotHelperFactory) NewDeleteActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (deleteResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, orcObject, controller)
}
