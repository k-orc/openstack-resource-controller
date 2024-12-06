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

package port

import (
	"context"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/generic"
	osclients "github.com/k-orc/openstack-resource-controller/internal/osclients"
)

type portActuator struct {
	*orcv1alpha1.Port
	osClient osclients.NetworkClient

	networkID             orcv1alpha1.UUID
	subnetsMapping        map[orcv1alpha1.OpenStackName]orcv1alpha1.UUID
	securityGroupsMapping map[orcv1alpha1.OpenStackName]orcv1alpha1.UUID
}

var _ generic.ResourceActuator[*ports.Port] = portActuator{}

func (obj portActuator) GetManagementPolicy() orcv1alpha1.ManagementPolicy {
	return obj.Spec.ManagementPolicy
}

func (obj portActuator) GetManagedOptions() *orcv1alpha1.ManagedOptions {
	return obj.Spec.ManagedOptions
}

func (portActuator) GetResourceID(osResource *ports.Port) string {
	return osResource.ID
}

func (obj portActuator) GetOSResourceByStatusID(ctx context.Context) (bool, *ports.Port, error) {
	if obj.Status.ID == nil {
		return false, nil, nil
	}

	port, err := obj.osClient.GetPort(ctx, *obj.Status.ID)
	return true, port, err
}

func (obj portActuator) GetOSResourceByImportID(ctx context.Context) (bool, *ports.Port, error) {
	if obj.Spec.Import == nil {
		return false, nil, nil
	}
	if obj.Spec.Import.ID == nil {
		return false, nil, nil
	}

	port, err := obj.osClient.GetPort(ctx, *obj.Spec.Import.ID)
	return true, port, err
}

func (obj portActuator) GetOSResourceByImportFilter(ctx context.Context) (bool, *ports.Port, error) {
	if obj.Spec.Import == nil {
		return false, nil, nil
	}
	if obj.Spec.Import.Filter == nil {
		return false, nil, nil
	}

	listOpts := listOptsFromImportFilter(obj.Spec.Import.Filter, obj.networkID)
	port, err := getResourceFromList(ctx, listOpts, obj.osClient)
	return true, port, err
}

func (obj portActuator) GetOSResourceBySpec(ctx context.Context) (*ports.Port, error) {
	listOpts := listOptsFromCreation(obj.Port)
	return getResourceFromList(ctx, listOpts, obj.osClient)
}

func (obj portActuator) CreateResource(ctx context.Context) ([]string, *ports.Port, error) {
	port, err := createResource(ctx, obj.Port, obj.networkID, obj.subnetsMapping, obj.securityGroupsMapping, obj.osClient)
	return nil, port, err
}

func (obj portActuator) DeleteResource(ctx context.Context, flavor *ports.Port) error {
	return obj.osClient.DeletePort(ctx, flavor.ID)
}
