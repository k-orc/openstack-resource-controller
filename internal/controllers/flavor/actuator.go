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

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/generic"
	osclients "github.com/k-orc/openstack-resource-controller/internal/osclients"
)

type flavorActuator struct {
	*orcv1alpha1.Flavor
	osClient osclients.ComputeClient
}

var _ generic.ResourceActuator[*flavors.Flavor] = flavorActuator{}

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

func (obj flavorActuator) GetOSResourceBySpec(ctx context.Context) (*flavors.Flavor, error) {
	if obj.Spec.Resource == nil {
		return nil, nil
	}
	return GetByFilter(ctx, obj.osClient, specToFilter(*obj.Spec.Resource))
}

func (obj flavorActuator) CreateResource(ctx context.Context) (*flavors.Flavor, error) {
	return createResource(ctx, obj.Flavor, obj.osClient)
}

func (obj flavorActuator) DeleteResource(ctx context.Context, flavor *flavors.Flavor) error {
	return obj.osClient.DeleteFlavor(ctx, flavor.ID)
}
