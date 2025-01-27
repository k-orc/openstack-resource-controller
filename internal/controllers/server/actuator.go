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
	"time"

	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/generic"
	"github.com/k-orc/openstack-resource-controller/internal/osclients"
	orcerrors "github.com/k-orc/openstack-resource-controller/internal/util/errors"
)

type osResourcePT = *servers.Server

const (
	// The frequency to poll when waiting for a server to become active
	serverActivePollingPeriod = 15 * time.Second
)

type serverActuator struct {
	obj        *orcv1alpha1.Server
	osClient   osclients.ComputeClient
	controller generic.ResourceController
}

var _ generic.DeleteResourceActuator[osResourcePT] = serverActuator{}
var _ generic.CreateResourceActuator[osResourcePT] = serverActuator{}

func (actuator serverActuator) GetObject() client.Object {
	return actuator.obj
}

func (actuator serverActuator) GetController() generic.ResourceController {
	return actuator.controller
}

func (actuator serverActuator) GetManagementPolicy() orcv1alpha1.ManagementPolicy {
	return actuator.obj.Spec.ManagementPolicy
}

func (actuator serverActuator) GetManagedOptions() *orcv1alpha1.ManagedOptions {
	return actuator.obj.Spec.ManagedOptions
}

func (serverActuator) GetResourceID(osResource *servers.Server) string {
	return osResource.ID
}

func (actuator serverActuator) GetStatusID() *string {
	return actuator.obj.Status.ID
}

func (actuator serverActuator) GetOSResourceByStatusID(ctx context.Context) (bool, *servers.Server, error) {
	if actuator.obj.Status.ID == nil {
		return false, nil, nil
	}

	osResource, err := actuator.osClient.GetServer(ctx, *actuator.obj.Status.ID)
	return true, osResource, err
}

func (actuator serverActuator) GetOSResourceBySpec(ctx context.Context) (*servers.Server, error) {
	if actuator.obj.Spec.Resource == nil {
		return nil, nil
	}

	return GetByFilter(ctx, actuator.osClient, specToFilter(actuator.obj))
}

func (actuator serverActuator) GetOSResourceByImportID(ctx context.Context) (bool, *servers.Server, error) {
	if actuator.obj.Spec.Import == nil || actuator.obj.Spec.Import.ID == nil {
		return false, nil, nil
	}

	osResource, err := actuator.osClient.GetServer(ctx, *actuator.obj.Spec.Import.ID)
	return true, osResource, err
}

func (actuator serverActuator) GetOSResourceByImportFilter(ctx context.Context) (bool, *servers.Server, error) {
	if actuator.obj.Spec.Import == nil || actuator.obj.Spec.Import.Filter == nil {
		return false, nil, nil
	}

	osResource, err := GetByFilter(ctx, actuator.osClient, *actuator.obj.Spec.Import.Filter)
	return true, osResource, err
}

func (actuator serverActuator) CreateResource(ctx context.Context) ([]generic.WaitingOnEvent, *servers.Server, error) {
	resource := actuator.obj.Spec.Resource
	if resource == nil {
		// Should have been caught by API validation
		return nil, nil, orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Creation requested, but spec.resource is not set")
	}

	var waitEvents []generic.WaitingOnEvent
	k8sClient := actuator.controller.GetK8sClient()

	image := &orcv1alpha1.Image{}
	{
		imageKey := client.ObjectKey{Name: string(resource.ImageRef), Namespace: actuator.obj.Namespace}
		if err := k8sClient.Get(ctx, imageKey, image); err != nil {
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
		flavorKey := client.ObjectKey{Name: string(resource.FlavorRef), Namespace: actuator.obj.Namespace}
		if err := k8sClient.Get(ctx, flavorKey, flavor); err != nil {
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
			portKey := client.ObjectKey{Name: string(*portSpec.PortRef), Namespace: actuator.obj.Namespace}
			if err := k8sClient.Get(ctx, portKey, portObject); err != nil {
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

	var userData []byte
	if resource.UserData != nil && resource.UserData.SecretRef != nil {
		secret := &corev1.Secret{}
		secretKey := client.ObjectKey{Name: string(*resource.UserData.SecretRef), Namespace: actuator.obj.Namespace}
		if err := k8sClient.Get(ctx, secretKey, secret); err != nil {
			if !apierrors.IsNotFound(err) {
				return nil, nil, fmt.Errorf("fetching secret %s: %w", secretKey.Name, err)
			}
			waitEvents = append(waitEvents, generic.WaitingOnORCExist("Secret", secretKey.Name))
		} else {
			var ok bool
			userData, ok = secret.Data["value"]
			if !ok {
				waitEvents = append(waitEvents, generic.WaitingOnORCReady("Secret", secret.Name))
			}
		}
	}

	if len(waitEvents) > 0 {
		return waitEvents, nil, nil
	}

	createOpts := servers.CreateOpts{
		Name:      string(getResourceName(actuator.obj)),
		ImageRef:  *image.Status.ID,
		FlavorRef: *flavor.Status.ID,
		Networks:  portList,
		UserData:  userData,
	}

	schedulerHints := servers.SchedulerHintOpts{}

	osResource, err := actuator.osClient.CreateServer(ctx, &createOpts, schedulerHints)

	// We should require the spec to be updated before retrying a create which returned a non-retryable error
	if err != nil && !orcerrors.IsRetryable(err) {
		return nil, nil, orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration creating resource: "+err.Error(), err)
	}

	return nil, osResource, err
}

func (actuator serverActuator) DeleteResource(ctx context.Context, osResource *servers.Server) ([]generic.WaitingOnEvent, error) {
	return nil, actuator.osClient.DeleteServer(ctx, osResource.ID)
}

var _ generic.UpdateResourceActuator[orcObjectPT, osResourcePT] = serverActuator{}

type resourceUpdater = generic.ResourceUpdater[orcObjectPT, osResourcePT]

func (actuator serverActuator) GetResourceUpdaters(ctx context.Context, orcObject orcObjectPT, osResource osResourcePT, controller generic.ResourceController) ([]resourceUpdater, error) {
	return []resourceUpdater{
		actuator.checkStatus,
	}, nil
}

func (serverActuator) checkStatus(ctx context.Context, orcObject orcObjectPT, osResource osResourcePT) ([]generic.WaitingOnEvent, orcObjectPT, osResourcePT, error) {
	log := ctrl.LoggerFrom(ctx)

	var waitEvents []generic.WaitingOnEvent
	var err error

	switch osResource.Status {
	case ServerStatusError:
		err = orcerrors.Terminal(orcv1alpha1.ConditionReasonUnrecoverableError, "Server is in ERROR state")
	case ServerStatusActive:
		// fall through
	default:
		log.V(3).Info("Waiting for OpenStack resource to be ACTIVE")
		waitEvents = append(waitEvents, generic.WaitingOnOpenStackReady(serverActivePollingPeriod))
	}

	return waitEvents, orcObject, osResource, err
}

type serverActuatorFactory struct{}

var _ generic.ActuatorFactory[orcObjectPT, osResourcePT] = serverActuatorFactory{}

func (serverActuatorFactory) NewCreateActuator(ctx context.Context, orcObject orcObjectPT, controller generic.ResourceController) ([]generic.WaitingOnEvent, generic.CreateResourceActuator[osResourcePT], error) {
	actuator, err := newActuator(ctx, controller, orcObject)
	return nil, actuator, err
}

func (serverActuatorFactory) NewDeleteActuator(ctx context.Context, orcObject orcObjectPT, controller generic.ResourceController) ([]generic.WaitingOnEvent, generic.DeleteResourceActuator[osResourcePT], error) {
	actuator, err := newActuator(ctx, controller, orcObject)
	return nil, actuator, err
}

func newActuator(ctx context.Context, controller generic.ResourceController, orcObject *orcv1alpha1.Server) (serverActuator, error) {
	if orcObject == nil {
		return serverActuator{}, fmt.Errorf("orcObject may not be nil")
	}

	log := ctrl.LoggerFrom(ctx)
	clientScope, err := controller.GetScopeFactory().NewClientScopeFromObject(ctx, controller.GetK8sClient(), log, orcObject)
	if err != nil {
		return serverActuator{}, err
	}
	osClient, err := clientScope.NewComputeClient()
	if err != nil {
		return serverActuator{}, err
	}

	return serverActuator{
		obj:        orcObject,
		osClient:   osClient,
		controller: controller,
	}, nil
}
