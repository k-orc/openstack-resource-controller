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

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/routers"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/generic"
	osclients "github.com/k-orc/openstack-resource-controller/internal/osclients"
	"github.com/k-orc/openstack-resource-controller/internal/scope"
	orcerrors "github.com/k-orc/openstack-resource-controller/internal/util/errors"
	"github.com/k-orc/openstack-resource-controller/internal/util/neutrontags"
)

type routerActuator struct {
	*orcv1alpha1.Router
	osClient osclients.NetworkClient
}

type routerCreateActuator struct {
	routerActuator
	k8sClient client.Client
}

func newActuator(ctx context.Context, k8sClient client.Client, scopeFactory scope.Factory, orcObject *orcv1alpha1.Router) (routerActuator, error) {
	log := ctrl.LoggerFrom(ctx)

	clientScope, err := scopeFactory.NewClientScopeFromObject(ctx, k8sClient, log, orcObject)
	if err != nil {
		return routerActuator{}, err
	}
	osClient, err := clientScope.NewNetworkClient()
	if err != nil {
		return routerActuator{}, err
	}

	return routerActuator{
		Router:   orcObject,
		osClient: osClient,
	}, nil
}

func newCreateActuator(ctx context.Context, k8sClient client.Client, scopeFactory scope.Factory, orcObject *orcv1alpha1.Router) (routerCreateActuator, error) {
	routerActuator, err := newActuator(ctx, k8sClient, scopeFactory, orcObject)
	if err != nil {
		return routerCreateActuator{}, err
	}

	return routerCreateActuator{
		routerActuator: routerActuator,
		k8sClient:      k8sClient,
	}, nil
}

var _ generic.DeleteResourceActuator[*routers.Router] = routerActuator{}
var _ generic.CreateResourceActuator[*routers.Router] = routerCreateActuator{}

func (obj routerActuator) GetObject() client.Object {
	return obj.Router
}

func (obj routerActuator) GetManagementPolicy() orcv1alpha1.ManagementPolicy {
	return obj.Spec.ManagementPolicy
}

func (obj routerActuator) GetManagedOptions() *orcv1alpha1.ManagedOptions {
	return obj.Spec.ManagedOptions
}

func (routerActuator) GetResourceID(osResource *routers.Router) string {
	return osResource.ID
}

func (obj routerActuator) GetOSResourceByStatusID(ctx context.Context) (bool, *routers.Router, error) {
	if obj.Status.ID == nil {
		return false, nil, nil
	}

	port, err := obj.osClient.GetRouter(ctx, *obj.Status.ID)
	return true, port, err
}

func (obj routerActuator) GetOSResourceBySpec(ctx context.Context) (*routers.Router, error) {
	if obj.Spec.Resource == nil {
		return nil, nil
	}

	listOpts := listOptsFromCreation(obj.Router)
	return getResourceFromList(ctx, listOpts, obj.osClient)
}

func (obj routerCreateActuator) GetOSResourceByImportID(ctx context.Context) (bool, *routers.Router, error) {
	if obj.Spec.Import == nil {
		return false, nil, nil
	}
	if obj.Spec.Import.ID == nil {
		return false, nil, nil
	}

	port, err := obj.osClient.GetRouter(ctx, *obj.Spec.Import.ID)
	return true, port, err
}

func (obj routerCreateActuator) GetOSResourceByImportFilter(ctx context.Context) (bool, *routers.Router, error) {
	if obj.Spec.Import == nil {
		return false, nil, nil
	}
	if obj.Spec.Import.Filter == nil {
		return false, nil, nil
	}

	listOpts := listOptsFromImportFilter(obj.Spec.Import.Filter)
	osResource, err := getResourceFromList(ctx, listOpts, obj.osClient)
	return true, osResource, err
}

func (obj routerCreateActuator) CreateResource(ctx context.Context) ([]generic.WaitingOnEvent, *routers.Router, error) {
	resource := obj.Router.Spec.Resource
	if resource == nil {
		// Should have been caught by API validation
		return nil, nil, orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Creation requested, but spec.resource is not set")
	}

	var waitEvents []generic.WaitingOnEvent

	var gatewayInfo *routers.GatewayInfo
	for name, result := range externalGWDep.GetDependencies(ctx, obj.k8sClient, obj.Router) {
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
		Description:  string(resource.Description),
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

	osResource, err := obj.osClient.CreateRouter(ctx, &createOpts)

	// We should require the spec to be updated before retrying a create which returned a conflict
	if orcerrors.IsConflict(err) {
		err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration creating resource: "+err.Error(), err)
	}

	return nil, osResource, err
}

func (obj routerActuator) DeleteResource(ctx context.Context, router *routers.Router) ([]generic.WaitingOnEvent, error) {
	return nil, obj.osClient.DeleteRouter(ctx, router.ID)
}

// getResourceName returns the name of the OpenStack resource we should use.
func getResourceName(orcObject *orcv1alpha1.Router) orcv1alpha1.OpenStackName {
	if orcObject.Spec.Resource.Name != nil {
		return *orcObject.Spec.Resource.Name
	}
	return orcv1alpha1.OpenStackName(orcObject.Name)
}

func listOptsFromImportFilter(filter *orcv1alpha1.RouterFilter) routers.ListOpts {
	listOpts := routers.ListOpts{
		Name:        string(filter.Name),
		Description: string(filter.Description),
		ProjectID:   string(filter.ProjectID),
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
