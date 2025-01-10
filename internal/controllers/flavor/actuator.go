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

package flavor

import (
	"context"

	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/flavors"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/generic"
	osclients "github.com/k-orc/openstack-resource-controller/internal/osclients"
	orcerrors "github.com/k-orc/openstack-resource-controller/internal/util/errors"
)

type flavorActuator struct {
	*orcv1alpha1.Flavor
	osClient   osclients.ComputeClient
	controller generic.ResourceControllerCommon
}

func newActuator(ctx context.Context, controller generic.ResourceControllerCommon, orcObject *orcv1alpha1.Flavor) (flavorActuator, error) {
	log := ctrl.LoggerFrom(ctx)

	clientScope, err := controller.GetScopeFactory().NewClientScopeFromObject(ctx, controller.GetK8sClient(), log, orcObject)
	if err != nil {
		return flavorActuator{}, err
	}
	osClient, err := clientScope.NewComputeClient()
	if err != nil {
		return flavorActuator{}, err
	}

	return flavorActuator{
		Flavor:     orcObject,
		osClient:   osClient,
		controller: controller,
	}, nil
}

var _ generic.CreateResourceActuator[*flavors.Flavor] = flavorActuator{}
var _ generic.DeleteResourceActuator[*flavors.Flavor] = flavorActuator{}

func (flavorActuator) GetControllerName() string {
	return "flavor"
}

func (obj flavorActuator) GetObject() client.Object {
	return obj.Flavor
}

func (obj flavorActuator) GetController() generic.ResourceControllerCommon {
	return obj.controller
}

func (obj flavorActuator) GetManagementPolicy() orcv1alpha1.ManagementPolicy {
	return obj.Spec.ManagementPolicy
}

func (obj flavorActuator) GetManagedOptions() *orcv1alpha1.ManagedOptions {
	return obj.Spec.ManagedOptions
}

func (flavorActuator) GetResourceID(osResource *flavors.Flavor) string {
	return osResource.ID
}

func (obj flavorActuator) GetOSResourceByStatusID(ctx context.Context) (bool, *flavors.Flavor, error) {
	if obj.Status.ID == nil {
		return false, nil, nil
	}
	flavor, err := obj.osClient.GetFlavor(ctx, *obj.Status.ID)
	return true, flavor, err
}

func (obj flavorActuator) GetOSResourceBySpec(ctx context.Context) (*flavors.Flavor, error) {
	if obj.Spec.Resource == nil {
		return nil, nil
	}
	return GetByFilter(ctx, obj.osClient, specToFilter(*obj.Spec.Resource))
}

func (obj flavorActuator) GetOSResourceByImportID(ctx context.Context) (bool, *flavors.Flavor, error) {
	if obj.Spec.Import == nil {
		return false, nil, nil
	}
	if obj.Spec.Import.ID == nil {
		return false, nil, nil
	}
	flavor, err := obj.osClient.GetFlavor(ctx, *obj.Spec.Import.ID)
	return true, flavor, err
}

func (obj flavorActuator) GetOSResourceByImportFilter(ctx context.Context) (bool, *flavors.Flavor, error) {
	if obj.Spec.Import == nil {
		return false, nil, nil
	}
	if obj.Spec.Import.Filter == nil {
		return false, nil, nil
	}
	flavor, err := GetByFilter(ctx, obj.osClient, *obj.Spec.Import.Filter)
	return true, flavor, err
}

func (obj flavorActuator) CreateResource(ctx context.Context) ([]generic.WaitingOnEvent, *flavors.Flavor, error) {
	resource := obj.Spec.Resource

	if resource == nil {
		// Should have been caught by API validation
		return nil, nil, orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Creation requested, but spec.resource is not set")
	}

	createOpts := flavors.CreateOpts{
		Name:        string(getResourceName(obj.Flavor)),
		RAM:         int(resource.RAM),
		VCPUs:       int(resource.Vcpus),
		Disk:        ptr.To(int(resource.Disk)),
		Swap:        ptr.To(int(resource.Swap)),
		IsPublic:    resource.IsPublic,
		Ephemeral:   ptr.To(int(resource.Ephemeral)),
		Description: string(ptr.Deref(resource.Description, "")),
	}

	osResource, err := obj.osClient.CreateFlavor(ctx, createOpts)
	if err != nil {
		// We should require the spec to be updated before retrying a create which returned a conflict
		if orcerrors.IsConflict(err) {
			err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration creating resource: "+err.Error(), err)
		}
		return nil, nil, err
	}

	return nil, osResource, nil
}

func (obj flavorActuator) DeleteResource(ctx context.Context, flavor *flavors.Flavor) ([]generic.WaitingOnEvent, error) {
	return nil, obj.osClient.DeleteFlavor(ctx, flavor.ID)
}

// getResourceName returns the name of the OpenStack resource we should use.
func getResourceName(orcObject *orcv1alpha1.Flavor) orcv1alpha1.OpenStackName {
	if orcObject.Spec.Resource.Name != nil {
		return *orcObject.Spec.Resource.Name
	}
	return orcv1alpha1.OpenStackName(orcObject.Name)
}
