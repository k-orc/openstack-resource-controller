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

package volume

import (
	"context"
	"iter"

	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/progress"
	"github.com/k-orc/openstack-resource-controller/v2/internal/osclients"
	orcerrors "github.com/k-orc/openstack-resource-controller/v2/internal/util/errors"
)

// OpenStack resource types
type (
	osResourceT = volumes.Volume

	createResourceActuator = interfaces.CreateResourceActuator[orcObjectPT, orcObjectT, filterT, osResourceT]
	deleteResourceActuator = interfaces.DeleteResourceActuator[orcObjectPT, orcObjectT, osResourceT]
	helperFactory          = interfaces.ResourceHelperFactory[orcObjectPT, orcObjectT, resourceSpecT, filterT, osResourceT]
)

type volumeActuator struct {
	osClient osclients.VolumeClient
}

var _ createResourceActuator = volumeActuator{}
var _ deleteResourceActuator = volumeActuator{}

func (volumeActuator) GetResourceID(osResource *volumes.Volume) string {
	return osResource.ID
}

func (actuator volumeActuator) GetOSResourceByID(ctx context.Context, id string) (*volumes.Volume, progress.ReconcileStatus) {
	volume, err := actuator.osClient.GetVolume(ctx, id)
	if err != nil {
		return nil, progress.WrapError(err)
	}
	return volume, nil
}

func (actuator volumeActuator) ListOSResourcesForAdoption(ctx context.Context, orcObject orcObjectPT) (iter.Seq2[*volumes.Volume, error], bool) {
	resourceSpec := orcObject.Spec.Resource
	if resourceSpec == nil {
		return nil, false
	}

	var filters []osclients.ResourceFilter[osResourceT]
	listOpts := volumes.ListOpts{}

	filters = append(filters,
		func(f *volumes.Volume) bool {
			name := getResourceName(orcObject)
			// Compare non-pointer values
			return f.Name == name &&
				f.Size == int(resourceSpec.Size)
		},
	)

	if resourceSpec.Description != nil {
		filters = append(filters, func(f *volumes.Volume) bool {
			return f.Description == *resourceSpec.Description
		})
	}

	return actuator.listOSResources(ctx, filters, &listOpts), true
}

func (actuator volumeActuator) ListOSResourcesForImport(ctx context.Context, obj orcObjectPT, filter filterT) (iter.Seq2[*osResourceT, error], progress.ReconcileStatus) {
	var filters []osclients.ResourceFilter[osResourceT]

	if filter.Name != nil {
		filters = append(filters, func(f *volumes.Volume) bool { return f.Name == string(*filter.Name) })
	}

	if filter.Description != nil {
		filters = append(filters, func(f *volumes.Volume) bool { return f.Description == *filter.Description })
	}

	if filter.Size != nil {
		filters = append(filters, func(f *volumes.Volume) bool { return f.Size == int(*filter.Size) })
	}

	return actuator.listOSResources(ctx, filters, &volumes.ListOpts{}), nil
}

func (actuator volumeActuator) listOSResources(ctx context.Context, filters []osclients.ResourceFilter[osResourceT], listOpts volumes.ListOptsBuilder) iter.Seq2[*volumes.Volume, error] {
	volumes := actuator.osClient.ListVolumes(ctx, listOpts)
	return osclients.Filter(volumes, filters...)
}

func (actuator volumeActuator) CreateResource(ctx context.Context, obj orcObjectPT) (*volumes.Volume, progress.ReconcileStatus) {
	resource := obj.Spec.Resource

	if resource == nil {
		// Should have been caught by API validation
		return nil, progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Creation requested, but spec.resource is not set"))
	}

	createOpts := volumes.CreateOpts{
		Name:        getResourceName(obj),
		Size:        int(resource.Size),
		Description: ptr.Deref(resource.Description, ""),
	}

	osResource, err := actuator.osClient.CreateVolume(ctx, createOpts)
	if err != nil {
		// We should require the spec to be updated before retrying a create which returned a conflict
		if !orcerrors.IsRetryable(err) {
			err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration creating resource: "+err.Error(), err)
		}
		return nil, progress.WrapError(err)
	}

	return osResource, nil
}

func (actuator volumeActuator) DeleteResource(ctx context.Context, _ orcObjectPT, volume *volumes.Volume) progress.ReconcileStatus {
	// FIXME(mandre) Make this optional
	deleteOpts := volumes.DeleteOpts{
		Cascade: false,
	}
	return progress.WrapError(actuator.osClient.DeleteVolume(ctx, volume.ID, deleteOpts))
}

type volumeHelperFactory struct{}

var _ helperFactory = volumeHelperFactory{}

func newActuator(ctx context.Context, orcObject *orcv1alpha1.Volume, controller interfaces.ResourceController) (volumeActuator, progress.ReconcileStatus) {
	log := ctrl.LoggerFrom(ctx)

	// Ensure credential secrets exist and have our finalizer
	_, reconcileStatus := credentialsDependency.GetDependencies(ctx, controller.GetK8sClient(), orcObject, func(*corev1.Secret) bool { return true })
	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return volumeActuator{}, reconcileStatus
	}

	clientScope, err := controller.GetScopeFactory().NewClientScopeFromObject(ctx, controller.GetK8sClient(), log, orcObject)
	if err != nil {
		return volumeActuator{}, progress.WrapError(err)
	}
	osClient, err := clientScope.NewVolumeClient()
	if err != nil {
		return volumeActuator{}, progress.WrapError(err)
	}

	return volumeActuator{
		osClient: osClient,
	}, nil
}

func (volumeHelperFactory) NewAPIObjectAdapter(obj orcObjectPT) adapterI {
	return volumeAdapter{obj}
}

func (volumeHelperFactory) NewCreateActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (createResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, orcObject, controller)
}

func (volumeHelperFactory) NewDeleteActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (deleteResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, orcObject, controller)
}
