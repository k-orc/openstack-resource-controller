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
	"iter"
	"time"

	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/generic"
	"github.com/k-orc/openstack-resource-controller/internal/osclients"
	orcerrors "github.com/k-orc/openstack-resource-controller/internal/util/errors"
)

type (
	osResourceT = servers.Server

	createResourceActuator    = generic.CreateResourceActuator[orcObjectPT, orcObjectT, filterT, osResourceT]
	deleteResourceActuator    = generic.DeleteResourceActuator[orcObjectPT, orcObjectT, osResourceT]
	reconcileResourceActuator = generic.ReconcileResourceActuator[orcObjectPT, osResourceT]
	resourceReconciler        = generic.ResourceReconciler[orcObjectPT, osResourceT]
	helperFactory             = generic.ResourceHelperFactory[orcObjectPT, orcObjectT, resourceSpecT, filterT, osResourceT]
	serverIterator            = iter.Seq2[*osResourceT, error]
)

const (
	// The frequency to poll when waiting for a server to become active
	serverActivePollingPeriod = 15 * time.Second
)

type serverActuator struct {
	osClient  osclients.ComputeClient
	k8sClient client.Client
}

var _ createResourceActuator = serverActuator{}
var _ deleteResourceActuator = serverActuator{}

func (serverActuator) GetResourceID(osResource *servers.Server) string {
	return osResource.ID
}

func (actuator serverActuator) GetOSResourceByID(ctx context.Context, id string) (*servers.Server, error) {
	return actuator.osClient.GetServer(ctx, id)
}

func (actuator serverActuator) ListOSResourcesForAdoption(ctx context.Context, obj *orcv1alpha1.Server) (serverIterator, bool) {
	if obj.Spec.Resource == nil {
		return nil, false
	}

	listOpts := servers.ListOpts{
		Name: string(getResourceName(obj)),
	}
	return actuator.osClient.ListServers(ctx, listOpts), true
}

func (actuator serverActuator) ListOSResourcesForImport(ctx context.Context, filter filterT) serverIterator {
	listOpts := servers.ListOpts{
		Name: string(ptr.Deref(filter.Name, "")),
	}
	return actuator.osClient.ListServers(ctx, listOpts)
}

func (actuator serverActuator) CreateResource(ctx context.Context, obj *orcv1alpha1.Server) ([]generic.ProgressStatus, *servers.Server, error) {
	resource := obj.Spec.Resource
	if resource == nil {
		// Should have been caught by API validation
		return nil, nil, orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Creation requested, but spec.resource is not set")
	}

	var waitEvents []generic.ProgressStatus

	image := &orcv1alpha1.Image{}
	{
		imageKey := client.ObjectKey{Name: string(resource.ImageRef), Namespace: obj.Namespace}
		if err := actuator.k8sClient.Get(ctx, imageKey, image); err != nil {
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
		if err := actuator.k8sClient.Get(ctx, flavorKey, flavor); err != nil {
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
			if err := actuator.k8sClient.Get(ctx, portKey, portObject); err != nil {
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
		secretKey := client.ObjectKey{Name: string(*resource.UserData.SecretRef), Namespace: obj.Namespace}
		if err := actuator.k8sClient.Get(ctx, secretKey, secret); err != nil {
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
		Name:      string(getResourceName(obj)),
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

func (actuator serverActuator) DeleteResource(ctx context.Context, _ orcObjectPT, osResource *servers.Server) ([]generic.ProgressStatus, error) {
	return nil, actuator.osClient.DeleteServer(ctx, osResource.ID)
}

var _ reconcileResourceActuator = serverActuator{}

func (actuator serverActuator) GetResourceReconcilers(ctx context.Context, orcObject orcObjectPT, osResource *osResourceT, controller generic.ResourceController) ([]resourceReconciler, error) {
	return []resourceReconciler{
		actuator.checkStatus,
	}, nil
}

func (serverActuator) checkStatus(ctx context.Context, orcObject orcObjectPT, osResource *osResourceT) ([]generic.ProgressStatus, error) {
	log := ctrl.LoggerFrom(ctx)

	var waitEvents []generic.ProgressStatus
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

	return waitEvents, err
}

type serverHelperFactory struct{}

var _ helperFactory = serverHelperFactory{}

func (serverHelperFactory) NewAPIObjectAdapter(obj orcObjectPT) adapterI {
	return serverAdapter{obj}
}

func (serverHelperFactory) NewCreateActuator(ctx context.Context, orcObject orcObjectPT, controller generic.ResourceController) ([]generic.ProgressStatus, createResourceActuator, error) {
	actuator, err := newActuator(ctx, controller, orcObject)
	return nil, actuator, err
}

func (serverHelperFactory) NewDeleteActuator(ctx context.Context, orcObject orcObjectPT, controller generic.ResourceController) ([]generic.ProgressStatus, deleteResourceActuator, error) {
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
		osClient:  osClient,
		k8sClient: controller.GetK8sClient(),
	}, nil
}
