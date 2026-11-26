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

package swiftcontainer

import (
	"context"
	"iter"
	"time"

	"github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/containers"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v3/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v3/internal/controllers/generic/progress"
	"github.com/k-orc/openstack-resource-controller/v3/internal/osclients"
	orcerrors "github.com/k-orc/openstack-resource-controller/v3/internal/util/errors"
)

// OpenStack resource types
type (
	osResourceT = osclients.SwiftContainer

	createResourceActuator    = interfaces.CreateResourceActuator[orcObjectPT, orcObjectT, filterT, osResourceT]
	deleteResourceActuator    = interfaces.DeleteResourceActuator[orcObjectPT, orcObjectT, osResourceT]
	reconcileResourceActuator = interfaces.ReconcileResourceActuator[orcObjectPT, osResourceT]
	resourceReconciler        = interfaces.ResourceReconciler[orcObjectPT, osResourceT]
	helperFactory             = interfaces.ResourceHelperFactory[orcObjectPT, orcObjectT, resourceSpecT, filterT, osResourceT]
)

// The frequency to poll when waiting for a container to be emptied
const swiftcontainerDeletingPollingPeriod = 10 * time.Second

type swiftcontainerActuator struct {
	osClient osclients.SwiftContainerClient
}

var (
	_ createResourceActuator    = swiftcontainerActuator{}
	_ deleteResourceActuator    = swiftcontainerActuator{}
	_ reconcileResourceActuator = swiftcontainerActuator{}
)

func (swiftcontainerActuator) GetResourceID(osResource *osResourceT) string {
	return osResource.Name
}

func (actuator swiftcontainerActuator) GetOSResourceByID(ctx context.Context, id string) (*osResourceT, progress.ReconcileStatus) {
	resource, err := actuator.osClient.GetSwiftContainer(ctx, id)
	if err != nil {
		return nil, progress.WrapError(err)
	}
	return resource, nil
}

func (actuator swiftcontainerActuator) ListOSResourcesForAdoption(ctx context.Context, orcObject orcObjectPT) (iter.Seq2[*osResourceT, error], bool) {
	if orcObject.Spec.Resource == nil {
		return nil, false
	}

	return actuator.listOSResources(ctx, getResourceName(orcObject)), true
}

func (actuator swiftcontainerActuator) ListOSResourcesForImport(ctx context.Context, obj orcObjectPT, filter filterT) (iter.Seq2[*osResourceT, error], progress.ReconcileStatus) {
	return actuator.listOSResources(ctx, string(ptr.Deref(filter.Name, ""))), nil
}

// listOSResources returns the container with the given name. Swift can only
// filter container listings by prefix.
func (actuator swiftcontainerActuator) listOSResources(ctx context.Context, name string) iter.Seq2[*osResourceT, error] {
	listOpts := containers.ListOpts{Prefix: name}
	filters := []osclients.ResourceFilter[osResourceT]{
		func(container *osResourceT) bool {
			return container.Name == name
		},
	}
	return osclients.Filter(actuator.osClient.ListSwiftContainers(ctx, listOpts), filters...)
}

func (actuator swiftcontainerActuator) CreateResource(ctx context.Context, obj orcObjectPT) (*osResourceT, progress.ReconcileStatus) {
	resource := obj.Spec.Resource

	metadata := make(map[string]string, len(resource.Metadata))
	for _, m := range resource.Metadata {
		metadata[m.Name] = m.Value
	}

	createOpts := containers.CreateOpts{
		Metadata:       metadata,
		ContainerRead:  resource.ReadACL,
		ContainerWrite: resource.WriteACL,
		StoragePolicy:  resource.StoragePolicy,
	}

	osResource, err := actuator.osClient.CreateSwiftContainer(ctx, getResourceName(obj), createOpts)
	if err != nil {
		if !orcerrors.IsRetryable(err) {
			err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration creating resource: "+err.Error(), err)
		}
		return nil, progress.WrapError(err)
	}

	return osResource, nil
}

func (actuator swiftcontainerActuator) DeleteResource(ctx context.Context, _ orcObjectPT, resource *osResourceT) progress.ReconcileStatus {
	err := actuator.osClient.DeleteSwiftContainer(ctx, resource.Name)
	// Swift returns a conflict when the container is not empty. ORC never
	// deletes objects, so wait for the container to be emptied.
	if orcerrors.IsConflict(err) {
		return progress.NewReconcileStatus().
			WithProgressMessage("Waiting for the Swift container to be emptied before deletion").
			WithRequeue(swiftcontainerDeletingPollingPeriod)
	}
	return progress.WrapError(err)
}

func (actuator swiftcontainerActuator) GetResourceReconcilers(ctx context.Context, orcObject orcObjectPT, osResource *osResourceT, controller interfaces.ResourceController) ([]resourceReconciler, progress.ReconcileStatus) {
	// SwiftContainer is immutable
	return nil, nil
}

type swiftcontainerHelperFactory struct{}

var _ helperFactory = swiftcontainerHelperFactory{}

func newActuator(ctx context.Context, orcObject *orcv1alpha1.SwiftContainer, controller interfaces.ResourceController) (swiftcontainerActuator, progress.ReconcileStatus) {
	log := ctrl.LoggerFrom(ctx)

	// Ensure credential secrets exist and have our finalizer
	_, reconcileStatus := credentialsDependency.RequireDependencies(ctx, controller.GetK8sClient(), orcObject, func(*corev1.Secret) bool { return true })
	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return swiftcontainerActuator{}, reconcileStatus
	}

	clientScope, err := controller.GetScopeFactory().NewClientScopeFromObject(ctx, controller.GetK8sClient(), log, orcObject)
	if err != nil {
		return swiftcontainerActuator{}, progress.WrapError(err)
	}
	osClient, err := clientScope.NewSwiftContainerClient()
	if err != nil {
		return swiftcontainerActuator{}, progress.WrapError(err)
	}

	return swiftcontainerActuator{
		osClient: osClient,
	}, nil
}

func (swiftcontainerHelperFactory) NewAPIObjectAdapter(obj orcObjectPT) adapterI {
	return swiftcontainerAdapter{obj}
}

func (swiftcontainerHelperFactory) NewCreateActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (createResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, orcObject, controller)
}

func (swiftcontainerHelperFactory) NewDeleteActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (deleteResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, orcObject, controller)
}
