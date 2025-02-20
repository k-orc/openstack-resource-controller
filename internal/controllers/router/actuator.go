/*
Copyright 2024 The ORC Authors.

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

package router

import (
	"context"
	"iter"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/routers"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/generic"
	osclients "github.com/k-orc/openstack-resource-controller/internal/osclients"
	orcerrors "github.com/k-orc/openstack-resource-controller/internal/util/errors"
	"github.com/k-orc/openstack-resource-controller/internal/util/neutrontags"
)

type (
	osResourceT = routers.Router

	createResourceActuator    = generic.CreateResourceActuator[orcObjectPT, orcObjectT, filterT, osResourceT]
	deleteResourceActuator    = generic.DeleteResourceActuator[orcObjectPT, orcObjectT, osResourceT]
	reconcileResourceActuator = generic.ReconcileResourceActuator[orcObjectPT, osResourceT]
	resourceReconciler        = generic.ResourceReconciler[orcObjectPT, osResourceT]
	helperFactory             = generic.ResourceHelperFactory[orcObjectPT, orcObjectT, resourceSpecT, filterT, osResourceT]
	routerIterator            = iter.Seq2[*osResourceT, error]
)

type routerActuator struct {
	osClient osclients.NetworkClient
}

type routerCreateActuator struct {
	routerActuator
	k8sClient client.Client
}

var _ createResourceActuator = routerCreateActuator{}
var _ deleteResourceActuator = routerActuator{}

func (routerActuator) GetResourceID(osResource *routers.Router) string {
	return osResource.ID
}

func (actuator routerActuator) GetOSResourceByID(ctx context.Context, id string) (*routers.Router, error) {
	return actuator.osClient.GetRouter(ctx, id)
}

func (actuator routerActuator) ListOSResourcesForAdoption(ctx context.Context, obj *orcv1alpha1.Router) (routerIterator, bool) {
	if obj.Spec.Resource == nil {
		return nil, false
	}

	listOpts := routers.ListOpts{Name: string(getResourceName(obj))}
	return actuator.osClient.ListRouter(ctx, listOpts), true
}

func (actuator routerCreateActuator) ListOSResourcesForImport(ctx context.Context, filter filterT) routerIterator {
	listOpts := routers.ListOpts{
		Name:        string(ptr.Deref(filter.Name, "")),
		Description: string(ptr.Deref(filter.Description, "")),
		Tags:        neutrontags.Join(filter.FilterByNeutronTags.Tags),
		TagsAny:     neutrontags.Join(filter.FilterByNeutronTags.TagsAny),
		NotTags:     neutrontags.Join(filter.FilterByNeutronTags.NotTags),
		NotTagsAny:  neutrontags.Join(filter.FilterByNeutronTags.NotTagsAny),
	}

	return actuator.osClient.ListRouter(ctx, listOpts)
}

func (actuator routerCreateActuator) CreateResource(ctx context.Context, obj *orcv1alpha1.Router) ([]generic.ProgressStatus, *routers.Router, error) {
	resource := obj.Spec.Resource
	if resource == nil {
		// Should have been caught by API validation
		return nil, nil, orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Creation requested, but spec.resource is not set")
	}

	gatewayInfo := &routers.GatewayInfo{}

	if len(resource.ExternalGateways) > 0 {
		var progressStatus []generic.ProgressStatus

		// Fetch dependencies and ensure they have our finalizer
		externalGW, progressStatus, err := externalGWDep.GetDependency(
			ctx, actuator.k8sClient, obj, func(dep *orcv1alpha1.Network) bool {
				return orcv1alpha1.IsAvailable(dep) && dep.Status.ID != nil
			},
		)
		if len(progressStatus) != 0 || err != nil {
			return progressStatus, nil, err
		}
		gatewayInfo.NetworkID = *externalGW.Status.ID
	}

	createOpts := routers.CreateOpts{
		Name:         string(ptr.Deref(resource.Name, "")),
		Description:  string(ptr.Deref(resource.Description, "")),
		AdminStateUp: resource.AdminStateUp,
		Distributed:  resource.Distributed,
		GatewayInfo:  gatewayInfo,
	}

	if len(resource.AvailabilityZoneHints) > 0 {
		createOpts.AvailabilityZoneHints = make([]string, len(resource.AvailabilityZoneHints))
		for i := range resource.AvailabilityZoneHints {
			createOpts.AvailabilityZoneHints[i] = string(resource.AvailabilityZoneHints[i])
		}
	}

	osResource, err := actuator.osClient.CreateRouter(ctx, &createOpts)

	// We should require the spec to be updated before retrying a create which returned a conflict
	if orcerrors.IsConflict(err) {
		err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration creating resource: "+err.Error(), err)
	}

	return nil, osResource, err
}

func (actuator routerActuator) DeleteResource(ctx context.Context, _ orcObjectPT, router *routers.Router) ([]generic.ProgressStatus, error) {
	return nil, actuator.osClient.DeleteRouter(ctx, router.ID)
}

var _ reconcileResourceActuator = routerActuator{}

func (actuator routerActuator) GetResourceReconcilers(ctx context.Context, orcObject orcObjectPT, osResource *osResourceT, controller generic.ResourceController) ([]resourceReconciler, error) {
	return []resourceReconciler{
		neutrontags.ReconcileTags[orcObjectPT, osResourceT](actuator.osClient, "routers", osResource.ID, orcObject.Spec.Resource.Tags, osResource.Tags),
	}, nil
}

type routerHelperFactory struct{}

var _ helperFactory = routerHelperFactory{}

func (routerHelperFactory) NewAPIObjectAdapter(obj orcObjectPT) adapterI {
	return routerAdapter{obj}
}

func (routerHelperFactory) NewCreateActuator(ctx context.Context, orcObject orcObjectPT, controller generic.ResourceController) ([]generic.ProgressStatus, createResourceActuator, error) {
	actuator, progressStatus, err := newCreateActuator(ctx, orcObject, controller)
	return progressStatus, actuator, err
}

func (routerHelperFactory) NewDeleteActuator(ctx context.Context, orcObject orcObjectPT, controller generic.ResourceController) ([]generic.ProgressStatus, deleteResourceActuator, error) {
	actuator, progressStatus, err := newActuator(ctx, orcObject, controller)
	return progressStatus, actuator, err
}

func newActuator(ctx context.Context, orcObject *orcv1alpha1.Router, controller generic.ResourceController) (routerActuator, []generic.ProgressStatus, error) {
	log := ctrl.LoggerFrom(ctx)

	// Ensure credential secrets exist and have our finalizer
	_, progressStatus, err := credentialsDependency.GetDependencies(ctx, controller.GetK8sClient(), orcObject, func(*corev1.Secret) bool { return true })
	if len(progressStatus) > 0 || err != nil {
		return routerActuator{}, progressStatus, err
	}

	clientScope, err := controller.GetScopeFactory().NewClientScopeFromObject(ctx, controller.GetK8sClient(), log, orcObject)
	if err != nil {
		return routerActuator{}, nil, err
	}
	osClient, err := clientScope.NewNetworkClient()
	if err != nil {
		return routerActuator{}, nil, err
	}

	return routerActuator{
		osClient: osClient,
	}, nil, nil
}

func newCreateActuator(ctx context.Context, orcObject *orcv1alpha1.Router, controller generic.ResourceController) (routerCreateActuator, []generic.ProgressStatus, error) {
	routerActuator, progressStatus, err := newActuator(ctx, orcObject, controller)
	if len(progressStatus) > 0 || err != nil {
		return routerCreateActuator{}, progressStatus, err
	}

	return routerCreateActuator{
		routerActuator: routerActuator,
		k8sClient:      controller.GetK8sClient(),
	}, nil, nil
}
