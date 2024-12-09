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

package server

import (
	"context"

	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/generic"
	"github.com/k-orc/openstack-resource-controller/internal/osclients"
	"github.com/k-orc/openstack-resource-controller/internal/scope"
	orcerrors "github.com/k-orc/openstack-resource-controller/internal/util/errors"
)

type serverActuator struct {
	*orcv1alpha1.Server
	osClient osclients.ComputeClient
}

func newActuator(ctx context.Context, k8sClient client.Client, scopeFactory scope.Factory, orcObject *orcv1alpha1.Server) (serverActuator, error) {
	log := ctrl.LoggerFrom(ctx)

	clientScope, err := scopeFactory.NewClientScopeFromObject(ctx, k8sClient, log, orcObject)
	if err != nil {
		return serverActuator{}, err
	}
	osClient, err := clientScope.NewComputeClient()
	if err != nil {
		return serverActuator{}, err
	}

	return serverActuator{
		Server:   orcObject,
		osClient: osClient,
	}, nil
}

var _ generic.DeleteResourceActuator[*servers.Server] = serverActuator{}
var _ generic.CreateResourceActuator[*servers.Server] = serverActuator{}

func (obj serverActuator) GetManagementPolicy() orcv1alpha1.ManagementPolicy {
	return obj.Spec.ManagementPolicy
}

func (obj serverActuator) GetManagedOptions() *orcv1alpha1.ManagedOptions {
	return obj.Spec.ManagedOptions
}

func (serverActuator) GetResourceID(osResource *servers.Server) string {
	return osResource.ID
}

func (obj serverActuator) GetOSResourceByStatusID(ctx context.Context) (bool, *servers.Server, error) {
	if obj.Status.ID == nil {
		return false, nil, nil
	}

	osResource, err := obj.osClient.GetServer(ctx, *obj.Status.ID)
	return true, osResource, err
}

func (obj serverActuator) GetOSResourceBySpec(ctx context.Context) (*servers.Server, error) {
	if obj.Spec.Resource == nil {
		return nil, nil
	}

	return GetByFilter(ctx, obj.osClient, specToFilter(*obj.Spec.Resource))
}

func (obj serverActuator) GetOSResourceByImportID(ctx context.Context) (bool, *servers.Server, error) {
	if obj.Spec.Import == nil || obj.Spec.Import.ID == nil {
		return false, nil, nil
	}

	osResource, err := obj.osClient.GetServer(ctx, *obj.Spec.Import.ID)
	return true, osResource, err
}

func (obj serverActuator) GetOSResourceByImportFilter(ctx context.Context) (bool, *servers.Server, error) {
	if obj.Spec.Import == nil || obj.Spec.Import.Filter == nil {
		return false, nil, nil
	}

	osResource, err := GetByFilter(ctx, obj.osClient, *obj.Spec.Import.Filter)
	return true, osResource, err
}

func (obj serverActuator) CreateResource(ctx context.Context) ([]string, *servers.Server, error) {
	resource := obj.Spec.Resource
	if resource == nil {
		// Should have been caught by API validation
		return nil, nil, orcerrors.Terminal(orcv1alpha1.OpenStackConditionReasonInvalidConfiguration, "Creation requested, but spec.resource is not set")
	}

	createOpts := servers.CreateOpts{
		Name: string(getResourceName(obj.Server)),
	}

	schedulerHints := servers.SchedulerHintOpts{}

	osResource, err := obj.osClient.CreateServer(ctx, &createOpts, schedulerHints)

	// We should require the spec to be updated before retrying a create which returned a conflict
	if orcerrors.IsConflict(err) {
		return nil, nil, orcerrors.Terminal(orcv1alpha1.OpenStackConditionReasonInvalidConfiguration, "invalid configuration creating resource: "+err.Error(), err)
	}

	return nil, osResource, err
}

func (obj serverActuator) DeleteResource(ctx context.Context, osResource *servers.Server) error {
	return obj.osClient.DeleteServer(ctx, osResource.ID)
}

// getResourceName returns the name of the OpenStack resource we should use.
func getResourceName(orcObject *orcv1alpha1.Server) orcv1alpha1.OpenStackName {
	if orcObject.Spec.Resource.Name != nil {
		return *orcObject.Spec.Resource.Name
	}
	return orcv1alpha1.OpenStackName(orcObject.Name)
}
