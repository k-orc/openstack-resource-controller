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
	"slices"
	"time"

	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/progress"
	"github.com/k-orc/openstack-resource-controller/v2/internal/logging"
	"github.com/k-orc/openstack-resource-controller/v2/internal/osclients"
	orcerrors "github.com/k-orc/openstack-resource-controller/v2/internal/util/errors"
	"github.com/k-orc/openstack-resource-controller/v2/internal/util/neutrontags"
)

type (
	osResourceT = servers.Server

	createResourceActuator    = interfaces.CreateResourceActuator[orcObjectPT, orcObjectT, filterT, osResourceT]
	deleteResourceActuator    = interfaces.DeleteResourceActuator[orcObjectPT, orcObjectT, osResourceT]
	reconcileResourceActuator = interfaces.ReconcileResourceActuator[orcObjectPT, osResourceT]
	resourceReconciler        = interfaces.ResourceReconciler[orcObjectPT, osResourceT]
	helperFactory             = interfaces.ResourceHelperFactory[orcObjectPT, orcObjectT, resourceSpecT, filterT, osResourceT]
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

func (actuator serverActuator) GetOSResourceByID(ctx context.Context, id string) (*servers.Server, progress.ReconcileStatus) {
	server, err := actuator.osClient.GetServer(ctx, id)
	if err != nil {
		return nil, progress.WrapError(err)
	}
	return server, nil
}

func (actuator serverActuator) ListOSResourcesForAdoption(ctx context.Context, obj *orcv1alpha1.Server) (serverIterator, bool) {
	if obj.Spec.Resource == nil {
		return nil, false
	}

	listOpts := servers.ListOpts{
		Name: fmt.Sprintf("^%s$", string(getResourceName(obj))),
		Tags: neutrontags.Join(obj.Spec.Resource.Tags),
	}

	return actuator.osClient.ListServers(ctx, listOpts), true
}

func (actuator serverActuator) ListOSResourcesForImport(ctx context.Context, obj orcObjectPT, filter filterT) (iter.Seq2[*osResourceT, error], progress.ReconcileStatus) {
	listOpts := servers.ListOpts{
		Tags:       neutrontags.Join(filter.Tags),
		TagsAny:    neutrontags.Join(filter.TagsAny),
		NotTags:    neutrontags.Join(filter.NotTags),
		NotTagsAny: neutrontags.Join(filter.NotTagsAny),
	}

	if filter.Name != nil {
		listOpts.Name = fmt.Sprintf("^%s$", string(*filter.Name))
	}

	return actuator.osClient.ListServers(ctx, listOpts), nil
}

func (actuator serverActuator) CreateResource(ctx context.Context, obj *orcv1alpha1.Server) (*servers.Server, progress.ReconcileStatus) {
	resource := obj.Spec.Resource
	if resource == nil {
		// Should have been caught by API validation
		return nil, progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Creation requested, but spec.resource is not set"))
	}

	reconcileStatus := progress.NewReconcileStatus()

	var image *orcv1alpha1.Image
	{
		dep, imageReconcileStatus := imageDependency.GetDependency(
			ctx, actuator.k8sClient, obj, func(image *orcv1alpha1.Image) bool {
				return orcv1alpha1.IsAvailable(image) && image.Status.ID != nil
			},
		)
		reconcileStatus = reconcileStatus.WithReconcileStatus(imageReconcileStatus)
		image = dep
	}

	flavor := &orcv1alpha1.Flavor{}
	{
		flavorKey := client.ObjectKey{Name: string(resource.FlavorRef), Namespace: obj.Namespace}
		if err := actuator.k8sClient.Get(ctx, flavorKey, flavor); err != nil {
			if apierrors.IsNotFound(err) {
				reconcileStatus = reconcileStatus.WaitingOnObject("Flavor", flavorKey.Name, progress.WaitingOnCreation)
			} else {
				return nil, reconcileStatus.WithError(fmt.Errorf("fetching flavor %s: %w", flavorKey.Name, err))
			}
		} else if !orcv1alpha1.IsAvailable(flavor) || flavor.Status.ID == nil {
			reconcileStatus = reconcileStatus.WaitingOnObject("Flavor", flavorKey.Name, progress.WaitingOnReady)
		}
	}

	portList := make([]servers.Network, len(resource.Ports))
	{
		portsMap, portsReconcileStatus := portDependency.GetDependencies(
			ctx, actuator.k8sClient, obj, func(port *orcv1alpha1.Port) bool {
				return port.Status.ID != nil
			},
		)
		reconcileStatus = reconcileStatus.WithReconcileStatus(portsReconcileStatus)
		if needsReschedule, _ := portsReconcileStatus.NeedsReschedule(); !needsReschedule {
			for i := range resource.Ports {
				portSpec := &resource.Ports[i]
				serverNetwork := &portList[i]

				if portSpec.PortRef == nil {
					// Should have been caught by API validation
					return nil, reconcileStatus.WithError(orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "empty port spec"))
				}

				if port, ok := portsMap[string(*portSpec.PortRef)]; !ok {
					// Programming error
					return nil, reconcileStatus.WithError(fmt.Errorf("port %s not present in portsMap", *portSpec.PortRef))
				} else {
					serverNetwork.Port = *port.Status.ID
				}
			}
		}
	}

	var userData []byte
	if resource.UserData != nil && resource.UserData.SecretRef != nil {
		secret := &corev1.Secret{}
		secretKey := client.ObjectKey{Name: string(*resource.UserData.SecretRef), Namespace: obj.Namespace}
		if err := actuator.k8sClient.Get(ctx, secretKey, secret); err != nil {
			if !apierrors.IsNotFound(err) {
				reconcileStatus = reconcileStatus.WithError(fmt.Errorf("fetching secret %s: %w", secretKey.Name, err))
			} else {
				reconcileStatus = reconcileStatus.WaitingOnObject("Secret", secretKey.Name, progress.WaitingOnCreation)
			}
		} else {
			var ok bool
			userData, ok = secret.Data["value"]
			if !ok {
				reconcileStatus.WithProgressMessage("User data secret does not contain \"value\" key")
			}
		}
	}

	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return nil, reconcileStatus
	}

	tags := make([]string, len(resource.Tags))
	for i := range resource.Tags {
		tags[i] = string(resource.Tags[i])
	}
	// Sort tags before creation to simplify comparisons
	slices.Sort(tags)

	createOpts := servers.CreateOpts{
		Name:      string(getResourceName(obj)),
		ImageRef:  *image.Status.ID,
		FlavorRef: *flavor.Status.ID,
		Networks:  portList,
		UserData:  userData,
		Tags:      tags,
	}

	schedulerHints := servers.SchedulerHintOpts{}

	osResource, err := actuator.osClient.CreateServer(ctx, &createOpts, schedulerHints)

	// We should require the spec to be updated before retrying a create which returned a non-retryable error
	if err != nil {
		if !orcerrors.IsRetryable(err) {
			err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration creating resource: "+err.Error(), err)
		}
		return nil, progress.WrapError(err)
	}

	return osResource, nil
}

func (actuator serverActuator) DeleteResource(ctx context.Context, _ orcObjectPT, osResource *servers.Server) progress.ReconcileStatus {
	return progress.WrapError(actuator.osClient.DeleteServer(ctx, osResource.ID))
}

var _ reconcileResourceActuator = serverActuator{}

func (actuator serverActuator) GetResourceReconcilers(ctx context.Context, orcObject orcObjectPT, osResource *osResourceT, controller interfaces.ResourceController) ([]resourceReconciler, progress.ReconcileStatus) {
	return []resourceReconciler{
		actuator.checkStatus,
	}, nil
}

func (serverActuator) checkStatus(ctx context.Context, orcObject orcObjectPT, osResource *osResourceT) progress.ReconcileStatus {
	log := ctrl.LoggerFrom(ctx)

	switch osResource.Status {
	case ServerStatusError:
		return progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonUnrecoverableError, "Server is in ERROR state"))

	case ServerStatusActive:
		return nil

	default:
		log.V(logging.Verbose).Info("Waiting for OpenStack resource to be ACTIVE")
		return progress.NewReconcileStatus().WaitingOnOpenStack(progress.WaitingOnReady, serverActivePollingPeriod)
	}
}

type serverHelperFactory struct{}

var _ helperFactory = serverHelperFactory{}

func (serverHelperFactory) NewAPIObjectAdapter(obj orcObjectPT) adapterI {
	return serverAdapter{obj}
}

func (serverHelperFactory) NewCreateActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (createResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, controller, orcObject)
}

func (serverHelperFactory) NewDeleteActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (deleteResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, controller, orcObject)
}

func newActuator(ctx context.Context, controller interfaces.ResourceController, orcObject *orcv1alpha1.Server) (serverActuator, progress.ReconcileStatus) {
	if orcObject == nil {
		return serverActuator{}, progress.WrapError(fmt.Errorf("orcObject may not be nil"))
	}

	log := ctrl.LoggerFrom(ctx)

	// Ensure credential secrets exist and have our finalizer
	_, reconcileStatus := credentialsDependency.GetDependencies(ctx, controller.GetK8sClient(), orcObject, func(*corev1.Secret) bool { return true })
	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return serverActuator{}, reconcileStatus
	}

	clientScope, err := controller.GetScopeFactory().NewClientScopeFromObject(ctx, controller.GetK8sClient(), log, orcObject)
	if err != nil {
		return serverActuator{}, progress.WrapError(err)
	}
	osClient, err := clientScope.NewComputeClient()
	if err != nil {
		return serverActuator{}, progress.WrapError(err)
	}

	return serverActuator{
		osClient:  osClient,
		k8sClient: controller.GetK8sClient(),
	}, nil
}
