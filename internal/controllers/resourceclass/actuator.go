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

package resourceclass

import (
	"context"
	"iter"

	"github.com/gophercloud/gophercloud/v2/openstack/placement/v1/resourceclasses"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v3/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v3/internal/controllers/generic/progress"
	"github.com/k-orc/openstack-resource-controller/v3/internal/osclients"
	orcerrors "github.com/k-orc/openstack-resource-controller/v3/internal/util/errors"
)

// OpenStack resource types
type (
	osResourceT = resourceclasses.ResourceClass

	createResourceActuator    = interfaces.CreateResourceActuator[orcObjectPT, orcObjectT, filterT, osResourceT]
	deleteResourceActuator    = interfaces.DeleteResourceActuator[orcObjectPT, orcObjectT, osResourceT]
	reconcileResourceActuator = interfaces.ReconcileResourceActuator[orcObjectPT, osResourceT]
	resourceReconciler        = interfaces.ResourceReconciler[orcObjectPT, osResourceT]
	helperFactory             = interfaces.ResourceHelperFactory[orcObjectPT, orcObjectT, resourceSpecT, filterT, osResourceT]
)

type resourceclassActuator struct {
	osClient  osclients.ResourceClassClient
	k8sClient client.Client
}

var _ createResourceActuator = resourceclassActuator{}
var _ deleteResourceActuator = resourceclassActuator{}
var _ reconcileResourceActuator = resourceclassActuator{}

func (resourceclassActuator) GetResourceID(osResource *osResourceT) string {
	return osResource.Name
}

func (actuator resourceclassActuator) GetOSResourceByID(ctx context.Context, id string) (*osResourceT, progress.ReconcileStatus) {
	resource, err := actuator.osClient.GetResourceClass(ctx, id)
	if err != nil {
		return nil, progress.WrapError(err)
	}
	return resource, nil
}

func (actuator resourceclassActuator) ListOSResourcesForAdoption(ctx context.Context, orcObject orcObjectPT) (iter.Seq2[*osResourceT, error], bool) {
	resourceSpec := orcObject.Spec.Resource
	if resourceSpec == nil {
		return nil, false
	}

	var filters []osclients.ResourceFilter[osResourceT]

	// Add client-side filters
	if resourceSpec.Name != nil {
		filters = append(filters, func(f *resourceclasses.ResourceClass) bool {
			return f.Name == string(*resourceSpec.Name)
		})
	}

	return actuator.listOSResources(ctx, filters), true
}

func (actuator resourceclassActuator) ListOSResourcesForImport(ctx context.Context, obj orcObjectPT, filter filterT) (iter.Seq2[*osResourceT, error], progress.ReconcileStatus) {
	filters := make([]osclients.ResourceFilter[osResourceT], 0, 1)

	// Add client-side filters
	if filter.Name != nil {
		filters = append(filters, func(f *resourceclasses.ResourceClass) bool {
			return f.Name == string(*filter.Name)
		})
	}

	return actuator.listOSResources(ctx, filters), nil
}

func (actuator resourceclassActuator) listOSResources(ctx context.Context, filters []osclients.ResourceFilter[osResourceT]) iter.Seq2[*resourceclasses.ResourceClass, error] {
	resourceClasses := actuator.osClient.ListResourceClasses(ctx)
	return osclients.Filter(resourceClasses, filters...)
}

func (actuator resourceclassActuator) CreateResource(ctx context.Context, obj orcObjectPT) (*osResourceT, progress.ReconcileStatus) {
	if obj.Spec.Resource.Name == nil {
		return nil, progress.WrapError(orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration creating resource: name must be set"))
	}

	createOpts := resourceclasses.CreateOpts{
		Name: getResourceName(obj),
	}

	osResource, err := actuator.osClient.CreateResourceClass(ctx, createOpts)
	if err != nil {
		if !orcerrors.IsRetryable(err) {
			err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration creating resource: "+err.Error(), err)
		}
		return nil, progress.WrapError(err)
	}

	return osResource, nil
}

func (actuator resourceclassActuator) DeleteResource(ctx context.Context, _ orcObjectPT, resource *osResourceT) progress.ReconcileStatus {
	return progress.WrapError(actuator.osClient.DeleteResourceClass(ctx, resource.Name))
}

func (actuator resourceclassActuator) GetResourceReconcilers(ctx context.Context, orcObject orcObjectPT, osResource *osResourceT, controller interfaces.ResourceController) ([]resourceReconciler, progress.ReconcileStatus) {
	return []resourceReconciler{}, nil
}

type resourceclassHelperFactory struct{}

var _ helperFactory = resourceclassHelperFactory{}

func newActuator(ctx context.Context, orcObject *orcv1alpha1.ResourceClass, controller interfaces.ResourceController) (resourceclassActuator, progress.ReconcileStatus) {
	log := ctrl.LoggerFrom(ctx)

	// Ensure credential secrets exist and have our finalizer
	_, reconcileStatus := credentialsDependency.RequireDependencies(ctx, controller.GetK8sClient(), orcObject, func(*corev1.Secret) bool { return true })
	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return resourceclassActuator{}, reconcileStatus
	}

	clientScope, err := controller.GetScopeFactory().NewClientScopeFromObject(ctx, controller.GetK8sClient(), log, orcObject)
	if err != nil {
		return resourceclassActuator{}, progress.WrapError(err)
	}
	osClient, err := clientScope.NewResourceClassClient()
	if err != nil {
		return resourceclassActuator{}, progress.WrapError(err)
	}

	return resourceclassActuator{
		osClient:  osClient,
		k8sClient: controller.GetK8sClient(),
	}, nil
}

func (resourceclassHelperFactory) NewAPIObjectAdapter(obj orcObjectPT) adapterI {
	return resourceclassAdapter{obj}
}

func (resourceclassHelperFactory) NewCreateActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (createResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, orcObject, controller)
}

func (resourceclassHelperFactory) NewDeleteActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (deleteResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, orcObject, controller)
}
