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
	"fmt"

	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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

type serverCreateActuator struct {
	serverActuator
	k8sClient client.Client
}

func newActuator(ctx context.Context, k8sClient client.Client, scopeFactory scope.Factory, orcObject *orcv1alpha1.Server) (serverActuator, error) {
	if orcObject == nil {
		return serverActuator{}, fmt.Errorf("orcObject may not be nil")
	}

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

func newCreateActuator(ctx context.Context, k8sClient client.Client, scopeFactory scope.Factory, orcObject *orcv1alpha1.Server) (serverCreateActuator, error) {
	actuator, err := newActuator(ctx, k8sClient, scopeFactory, orcObject)
	if err != nil {
		return serverCreateActuator{}, err
	}
	return serverCreateActuator{
		serverActuator: actuator,
		k8sClient:      k8sClient,
	}, nil
}

var _ generic.DeleteResourceActuator[*servers.Server] = serverActuator{}
var _ generic.CreateResourceActuator[*servers.Server] = serverCreateActuator{}

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

	return GetByFilter(ctx, obj.osClient, specToFilter(obj.Server))
}

func (obj serverCreateActuator) GetOSResourceByImportID(ctx context.Context) (bool, *servers.Server, error) {
	if obj.Spec.Import == nil || obj.Spec.Import.ID == nil {
		return false, nil, nil
	}

	osResource, err := obj.osClient.GetServer(ctx, *obj.Spec.Import.ID)
	return true, osResource, err
}

func (obj serverCreateActuator) GetOSResourceByImportFilter(ctx context.Context) (bool, *servers.Server, error) {
	if obj.Spec.Import == nil || obj.Spec.Import.Filter == nil {
		return false, nil, nil
	}

	osResource, err := GetByFilter(ctx, obj.osClient, *obj.Spec.Import.Filter)
	return true, osResource, err
}

func (obj serverCreateActuator) CreateResource(ctx context.Context) ([]generic.WaitingOnEvent, *servers.Server, error) {
	resource := obj.Spec.Resource
	if resource == nil {
		// Should have been caught by API validation
		return nil, nil, orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Creation requested, but spec.resource is not set")
	}

	var waitEvents []generic.WaitingOnEvent

	image := &orcv1alpha1.Image{}
	{
		imageKey := client.ObjectKey{Name: string(resource.ImageRef), Namespace: obj.Namespace}
		if err := obj.k8sClient.Get(ctx, imageKey, image); err != nil {
			if apierrors.IsNotFound(err) {
				waitEvents = append(waitEvents, generic.WaitingOnORCExist("Image", imageKey.Name))
			} else {
				return nil, nil, fmt.Errorf("fetching image %s: %w", imageKey.Name, err)
			}
		}
		if !orcv1alpha1.IsAvailable(image) || image.Status.ID == nil {
			waitEvents = append(waitEvents, generic.WaitingOnORCReady("Image", imageKey.Name))
		}
	}

	flavor := &orcv1alpha1.Flavor{}
	{
		flavorKey := client.ObjectKey{Name: string(resource.FlavorRef), Namespace: obj.Namespace}
		if err := obj.k8sClient.Get(ctx, flavorKey, flavor); err != nil {
			if apierrors.IsNotFound(err) {
				waitEvents = append(waitEvents, generic.WaitingOnORCExist("Flavor", flavorKey.Name))
			} else {
				return nil, nil, fmt.Errorf("fetching flavor %s: %w", flavorKey.Name, err)
			}
		}
		if !orcv1alpha1.IsAvailable(flavor) || flavor.Status.ID == nil {
			waitEvents = append(waitEvents, generic.WaitingOnORCReady("Flavor", flavorKey.Name))
		}
	}

	portList := make([]servers.Network, len(resource.Ports))
	{
		for i := range resource.Ports {
			portSpec := &resource.Ports[i]
			port := &portList[i]

			if portSpec.PortRef == nil {
				// Should have been caught by API validation
				return nil, nil, orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "empty port spec")
			}

			portObject := &orcv1alpha1.Port{}
			portKey := client.ObjectKey{Name: string(*portSpec.PortRef), Namespace: obj.Namespace}
			if err := obj.k8sClient.Get(ctx, portKey, portObject); err != nil {
				if apierrors.IsNotFound(err) {
					waitEvents = append(waitEvents, generic.WaitingOnORCExist("Port", portKey.Name))
					continue
				}
				return nil, nil, fmt.Errorf("fetching port %s: %w", portKey.Name, err)
			}
			if !orcv1alpha1.IsAvailable(portObject) || portObject.Status.ID == nil {
				waitEvents = append(waitEvents, generic.WaitingOnORCReady("Port", portKey.Name))
				continue
			}

			port.Port = *portObject.Status.ID
		}
	}

	if len(waitEvents) > 0 {
		return waitEvents, nil, nil
	}

	createOpts := servers.CreateOpts{
		Name:      string(getResourceName(obj.Server)),
		ImageRef:  *image.Status.ID,
		FlavorRef: *flavor.Status.ID,
		Networks:  portList,
	}

	schedulerHints := servers.SchedulerHintOpts{}

	osResource, err := obj.osClient.CreateServer(ctx, &createOpts, schedulerHints)

	// We should require the spec to be updated before retrying a create which returned a non-retryable error
	if err != nil && !orcerrors.IsRetryable(err) {
		return nil, nil, orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration creating resource: "+err.Error(), err)
	}

	return nil, osResource, err
}

func (obj serverActuator) DeleteResource(ctx context.Context, osResource *servers.Server) ([]generic.WaitingOnEvent, error) {
	return nil, obj.osClient.DeleteServer(ctx, osResource.ID)
}

// getResourceName returns the name of the OpenStack resource we should use.
func getResourceName(orcObject *orcv1alpha1.Server) orcv1alpha1.OpenStackName {
	if orcObject.Spec.Resource.Name != nil {
		return *orcObject.Spec.Resource.Name
	}
	return orcv1alpha1.OpenStackName(orcObject.Name)
}
