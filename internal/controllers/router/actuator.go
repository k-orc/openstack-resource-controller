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
	"fmt"

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

type osResourcePT = *routers.Router

type routerActuator struct {
	obj        *orcv1alpha1.Router
	osClient   osclients.NetworkClient
	controller generic.ResourceController
}

type routerCreateActuator struct {
	routerActuator
}

var _ generic.DeleteResourceActuator[osResourcePT] = routerActuator{}
var _ generic.CreateResourceActuator[osResourcePT] = routerCreateActuator{}

func (actuator routerActuator) GetObject() client.Object {
	return actuator.obj
}

func (actuator routerActuator) GetController() generic.ResourceController {
	return actuator.controller
}

func (actuator routerActuator) GetManagementPolicy() orcv1alpha1.ManagementPolicy {
	return actuator.obj.Spec.ManagementPolicy
}

func (actuator routerActuator) GetManagedOptions() *orcv1alpha1.ManagedOptions {
	return actuator.obj.Spec.ManagedOptions
}

func (routerActuator) GetResourceID(osResource *routers.Router) string {
	return osResource.ID
}

func (actuator routerActuator) GetStatusID() *string {
	return actuator.obj.Status.ID
}

func (actuator routerActuator) GetOSResourceByStatusID(ctx context.Context) (bool, *routers.Router, error) {
	if actuator.obj.Status.ID == nil {
		return false, nil, nil
	}

	port, err := actuator.osClient.GetRouter(ctx, *actuator.obj.Status.ID)
	return true, port, err
}

func (actuator routerActuator) GetOSResourceBySpec(ctx context.Context) (*routers.Router, error) {
	if actuator.obj.Spec.Resource == nil {
		return nil, nil
	}

	listOpts := listOptsFromCreation(actuator.obj)
	return getResourceFromList(ctx, listOpts, actuator.osClient)
}

func (actuator routerCreateActuator) GetOSResourceByImportID(ctx context.Context) (bool, *routers.Router, error) {
	if actuator.obj.Spec.Import == nil {
		return false, nil, nil
	}
	if actuator.obj.Spec.Import.ID == nil {
		return false, nil, nil
	}

	port, err := actuator.osClient.GetRouter(ctx, *actuator.obj.Spec.Import.ID)
	return true, port, err
}

func (actuator routerCreateActuator) GetOSResourceByImportFilter(ctx context.Context) (bool, *routers.Router, error) {
	if actuator.obj.Spec.Import == nil {
		return false, nil, nil
	}
	if actuator.obj.Spec.Import.Filter == nil {
		return false, nil, nil
	}

	listOpts := listOptsFromImportFilter(actuator.obj.Spec.Import.Filter)
	osResource, err := getResourceFromList(ctx, listOpts, actuator.osClient)
	return true, osResource, err
}

func (actuator routerCreateActuator) CreateResource(ctx context.Context) ([]generic.WaitingOnEvent, *routers.Router, error) {
	resource := actuator.obj.Spec.Resource
	if resource == nil {
		// Should have been caught by API validation
		return nil, nil, orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Creation requested, but spec.resource is not set")
	}

	var waitEvents []generic.WaitingOnEvent

	var gatewayInfo *routers.GatewayInfo
	for name, result := range externalGWDep.GetDependencies(ctx, actuator.controller.GetK8sClient(), actuator.obj) {
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

func (actuator routerActuator) DeleteResource(ctx context.Context, router *routers.Router) ([]generic.WaitingOnEvent, error) {
	return nil, actuator.osClient.DeleteRouter(ctx, router.ID)
}

func listOptsFromImportFilter(filter *orcv1alpha1.RouterFilter) routers.ListOpts {
	listOpts := routers.ListOpts{
		Name:        string(ptr.Deref(filter.Name, "")),
		Description: string(ptr.Deref(filter.Description, "")),
		Tags:        neutrontags.Join(filter.FilterByNeutronTags.Tags),
		TagsAny:     neutrontags.Join(filter.FilterByNeutronTags.TagsAny),
		NotTags:     neutrontags.Join(filter.FilterByNeutronTags.NotTags),
		NotTagsAny:  neutrontags.Join(filter.FilterByNeutronTags.NotTagsAny),
	}
	return listOpts
}

// listOptsFromCreation returns a listOpts which will return the OpenStack
// resource which would have been created from the current spec and hopefully no
// other. Its purpose is to automatically adopt a resource that we created but
// failed to write to status.id.
func listOptsFromCreation(osResource *orcv1alpha1.Router) routers.ListOpts {
	return routers.ListOpts{Name: string(getResourceName(osResource))}
}

func getResourceFromList(ctx context.Context, listOpts routers.ListOpts, networkClient osclients.NetworkClient) (*routers.Router, error) {
	osResources, err := networkClient.ListRouter(ctx, listOpts)
	if err != nil {
		return nil, err
	}

	if len(osResources) == 1 {
		return &osResources[0], nil
	}

	// No resource found
	if len(osResources) == 0 {
		return nil, nil
	}

	// Multiple resources found
	return nil, orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, fmt.Sprintf("Expected to find exactly one OpenStack resource to import. Found %d", len(osResources)))
}

var _ generic.UpdateResourceActuator[orcObjectPT, osResourcePT] = routerActuator{}

type resourceUpdater = generic.ResourceUpdater[orcObjectPT, osResourcePT]

func (actuator routerActuator) GetResourceUpdaters(ctx context.Context, orcObject orcObjectPT, osResource osResourcePT, controller generic.ResourceController) ([]resourceUpdater, error) {
	return []resourceUpdater{
		actuator.updateTags,
	}, nil
}

func (actuator routerActuator) updateTags(ctx context.Context, orcObject orcObjectPT, osResource osResourcePT) ([]generic.WaitingOnEvent, orcObjectPT, osResourcePT, error) {
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
	return nil, orcObject, osResource, err
}

type routerActuatorFactory struct{}

var _ generic.ActuatorFactory[orcObjectPT, osResourcePT] = routerActuatorFactory{}

func (routerActuatorFactory) NewCreateActuator(ctx context.Context, orcObject orcObjectPT, controller generic.ResourceController) ([]generic.WaitingOnEvent, generic.CreateResourceActuator[osResourcePT], error) {
	actuator, err := newCreateActuator(ctx, orcObject, controller)
	return nil, actuator, err
}

func (routerActuatorFactory) NewDeleteActuator(ctx context.Context, orcObject orcObjectPT, controller generic.ResourceController) ([]generic.WaitingOnEvent, generic.DeleteResourceActuator[osResourcePT], error) {
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
		obj:        orcObject,
		osClient:   osClient,
		controller: controller,
	}, nil
}

func newCreateActuator(ctx context.Context, orcObject *orcv1alpha1.Router, controller generic.ResourceController) (routerCreateActuator, error) {
	routerActuator, err := newActuator(ctx, orcObject, controller)
	if err != nil {
		return routerCreateActuator{}, err
	}

	return routerCreateActuator{
		routerActuator: routerActuator,
	}, nil
}
