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

package network

import (
	"context"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/generic"
	"github.com/k-orc/openstack-resource-controller/internal/osclients"
)

type networkActuator struct {
	*orcv1alpha1.Network
	osClient osclients.NetworkClient
}

var _ generic.ResourceActuator[*networkExt] = networkActuator{}

func (obj networkActuator) GetManagementPolicy() orcv1alpha1.ManagementPolicy {
	return obj.Spec.ManagementPolicy
}

func (obj networkActuator) GetManagedOptions() *orcv1alpha1.ManagedOptions {
	return obj.Spec.ManagedOptions
}

func (networkActuator) GetResourceID(osResource *networkExt) string {
	return osResource.ID
}

func getNetworkByID(ctx context.Context, osClient osclients.NetworkClient, id string) (*networkExt, error) {
	osResource := &networkExt{}
	getResult := osClient.GetNetwork(ctx, id)
	err := getResult.ExtractInto(osResource)
	if err != nil {
		return nil, err
	}
	return osResource, nil
}

func (obj networkActuator) GetOSResourceByStatusID(ctx context.Context) (bool, *networkExt, error) {
	if obj.Status.ID == nil {
		return false, nil, nil
	}
	network, err := getNetworkByID(ctx, obj.osClient, *obj.Status.ID)
	return true, network, err
}

func (obj networkActuator) GetOSResourceByImportID(ctx context.Context) (bool, *networkExt, error) {
	if obj.Spec.Import == nil {
		return false, nil, nil
	}
	if obj.Spec.Import.ID == nil {
		return false, nil, nil
	}
	network, err := getNetworkByID(ctx, obj.osClient, *obj.Spec.Import.ID)
	return true, network, err
}

func (obj networkActuator) GetOSResourceByImportFilter(ctx context.Context) (bool, *networkExt, error) {
	if obj.Spec.Import == nil {
		return false, nil, nil
	}

	if obj.Spec.Import.Filter == nil {
		return false, nil, nil
	}

	listOpts := listOptsFromImportFilter(obj.Spec.Import.Filter)
	osResource, err := getResourceFromList(ctx, listOpts, obj.osClient)
	if err != nil {
		return true, nil, err
	}
	if osResource == nil {
		return true, nil, nil
	}
	return true, osResource, nil
}

func (obj networkActuator) GetOSResourceBySpec(ctx context.Context) (*networkExt, error) {
	listOpts := listOptsFromCreation(obj.Network)
	return getResourceFromList(ctx, listOpts, obj.osClient)
}

func (obj networkActuator) CreateResource(ctx context.Context) ([]string, *networkExt, error) {
	network, err := createResource(ctx, obj.Network, obj.osClient)
	return nil, network, err
}

func (obj networkActuator) DeleteResource(ctx context.Context, network *networkExt) error {
	return obj.osClient.DeleteNetwork(ctx, network.ID).ExtractErr()
}
