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
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/generic"
	osclients "github.com/k-orc/openstack-resource-controller/internal/osclients"
	orcerrors "github.com/k-orc/openstack-resource-controller/internal/util/errors"
)

type routerActuator struct {
	*orcv1alpha1.Router
	osClient  osclients.NetworkClient
	k8sClient client.Client
}

var _ generic.ResourceActuator[*routers.Router] = routerActuator{}

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

func (obj routerActuator) GetOSResourceByImportID(ctx context.Context) (bool, *routers.Router, error) {
	if obj.Spec.Import == nil {
		return false, nil, nil
	}
	if obj.Spec.Import.ID == nil {
		return false, nil, nil
	}

	port, err := obj.osClient.GetRouter(ctx, *obj.Spec.Import.ID)
	return true, port, err
}

func (obj routerActuator) GetOSResourceByImportFilter(ctx context.Context) (bool, *routers.Router, error) {
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

func (obj routerActuator) GetOSResourceBySpec(ctx context.Context) (*routers.Router, error) {
	listOpts := listOptsFromCreation(obj.Router)
	return getResourceFromList(ctx, listOpts, obj.osClient)
}

func (obj routerActuator) CreateResource(ctx context.Context) ([]string, *routers.Router, error) {
	resource := obj.Router.Spec.Resource
	if resource == nil {
		// Should have been caught by API validation
		return nil, nil, orcerrors.Terminal(orcv1alpha1.OpenStackConditionReasonInvalidConfiguration, "Creation requested, but spec.resource is not set")
	}

	var gatewayInfo *routers.GatewayInfo
	for name, result := range externalGWDep.GetDependencies(ctx, obj.k8sClient, obj.Router) {
		err := result.Err()
		if err != nil {
			if apierrors.IsNotFound(err) {
				return []string{waitingOnCreationMsg("network", name)}, nil, nil
			}
			return nil, nil, err
		}

		network := result.Ok()
		if !orcv1alpha1.IsAvailable(network) {
			return []string{waitingOnAvailableMsg("network", name)}, nil, nil
		}

		if network.Status.ID == nil {
			// Programming error, but lets not panic
			return []string{fmt.Sprintf("network %s is available but status.id is nil", name)}, nil, nil
		}

		gatewayInfo = &routers.GatewayInfo{
			NetworkID: *network.Status.ID,
		}
		break
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
		err = orcerrors.Terminal(orcv1alpha1.OpenStackConditionReasonInvalidConfiguration, "invalid configuration creating resource: "+err.Error(), err)
	}

	return nil, osResource, err
}

func (obj routerActuator) DeleteResource(ctx context.Context, router *routers.Router) error {
	return obj.osClient.DeleteRouter(ctx, router.ID)
}
