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

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/attributestags"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/routers"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/utils/ptr"
	"k8s.io/utils/set"
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

	var waitEvents []generic.ProgressStatus

	var gatewayInfo *routers.GatewayInfo
	for name, result := range externalGWDep.GetDependencies(ctx, actuator.k8sClient, obj) {
		err := result.Err()
		if err != nil {
			if apierrors.IsNotFound(err) {
				waitEvents = append(waitEvents, generic.WaitingOnORCExist("Network", name))
				continue
			}
			return nil, nil, err
		}

		network := result.Ok()
		if !orcv1alpha1.IsAvailable(network) || network.Status.ID == nil {
			waitEvents = append(waitEvents, generic.WaitingOnORCReady("Network", name))
			continue
		}

		gatewayInfo = &routers.GatewayInfo{
			NetworkID: *network.Status.ID,
		}
		break
	}

	if len(waitEvents) > 0 {
		return waitEvents, nil, nil
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
		actuator.updateTags,
	}, nil
}

func (actuator routerActuator) updateTags(ctx context.Context, orcObject orcObjectPT, osResource *osResourceT) ([]generic.ProgressStatus, error) {
	resourceTagSet := set.New[string](osResource.Tags...)
	objectTagSet := set.New[string]()
	for i := range orcObject.Spec.Resource.Tags {
		objectTagSet.Insert(string(orcObject.Spec.Resource.Tags[i]))
	}
	var err error
	if !objectTagSet.Equal(resourceTagSet) {
		opts := attributestags.ReplaceAllOpts{Tags: objectTagSet.SortedList()}
		_, err = actuator.osClient.ReplaceAllAttributesTags(ctx, "routers", osResource.ID, &opts)
	}
	return nil, err
}

type routerHelperFactory struct{}

var _ helperFactory = routerHelperFactory{}

func (routerHelperFactory) NewAPIObjectAdapter(obj orcObjectPT) adapterI {
	return routerAdapter{obj}
}

func (routerHelperFactory) NewCreateActuator(ctx context.Context, orcObject orcObjectPT, controller generic.ResourceController) ([]generic.ProgressStatus, createResourceActuator, error) {
	actuator, err := newCreateActuator(ctx, orcObject, controller)
	return nil, actuator, err
}

func (routerHelperFactory) NewDeleteActuator(ctx context.Context, orcObject orcObjectPT, controller generic.ResourceController) ([]generic.ProgressStatus, deleteResourceActuator, error) {
	actuator, err := newActuator(ctx, orcObject, controller)
	return nil, actuator, err
}

func newActuator(ctx context.Context, orcObject *orcv1alpha1.Router, controller generic.ResourceController) (routerActuator, error) {
	log := ctrl.LoggerFrom(ctx)

	clientScope, err := controller.GetScopeFactory().NewClientScopeFromObject(ctx, controller.GetK8sClient(), log, orcObject)
	if err != nil {
		return routerActuator{}, err
	}
	osClient, err := clientScope.NewNetworkClient()
	if err != nil {
		return routerActuator{}, err
	}

	return routerActuator{
		osClient: osClient,
	}, nil
}

func newCreateActuator(ctx context.Context, orcObject *orcv1alpha1.Router, controller generic.ResourceController) (routerCreateActuator, error) {
	routerActuator, err := newActuator(ctx, orcObject, controller)
	if err != nil {
		return routerCreateActuator{}, err
	}

	return routerCreateActuator{
		routerActuator: routerActuator,
		k8sClient:      controller.GetK8sClient(),
	}, nil
}
