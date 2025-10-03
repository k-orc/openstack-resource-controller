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
	"maps"
	"slices"
	"time"

	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/volumeattach"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/progress"
	"github.com/k-orc/openstack-resource-controller/v2/internal/logging"
	"github.com/k-orc/openstack-resource-controller/v2/internal/osclients"
	"github.com/k-orc/openstack-resource-controller/v2/internal/util/attachments"
	orcerrors "github.com/k-orc/openstack-resource-controller/v2/internal/util/errors"
	"github.com/k-orc/openstack-resource-controller/v2/internal/util/tags"
	orcapplyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/pkg/clients/applyconfiguration/api/v1alpha1"
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

	// The frequency to poll when waiting for an attachment or detachment to be reflected
	serverAttachmentPollingPeriod = 5 * time.Second
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
		Name: fmt.Sprintf("^%s$", getResourceName(obj)),
		Tags: tags.Join(obj.Spec.Resource.Tags),
	}

	return actuator.osClient.ListServers(ctx, listOpts), true
}

func (actuator serverActuator) ListOSResourcesForImport(ctx context.Context, obj orcObjectPT, filter filterT) (iter.Seq2[*osResourceT, error], progress.ReconcileStatus) {
	listOpts := servers.ListOpts{
		Tags:       tags.Join(filter.Tags),
		TagsAny:    tags.Join(filter.TagsAny),
		NotTags:    tags.Join(filter.NotTags),
		NotTagsAny: tags.Join(filter.NotTagsAny),
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
	serverGroup := &orcv1alpha1.ServerGroup{}
	if resource.ServerGroupRef != nil {
		serverGroupKey := client.ObjectKey{Name: string(*resource.ServerGroupRef), Namespace: obj.Namespace}
		if err := actuator.k8sClient.Get(ctx, serverGroupKey, serverGroup); err != nil {
			if apierrors.IsNotFound(err) {
				reconcileStatus = reconcileStatus.WaitingOnObject("ServerGroup", serverGroupKey.Name, progress.WaitingOnCreation)
			} else {
				return nil, reconcileStatus.WithError(fmt.Errorf("fetching server group %s: %w", serverGroupKey.Name, err))
			}
		} else if !orcv1alpha1.IsAvailable(serverGroup) || serverGroup.Status.ID == nil {
			reconcileStatus = reconcileStatus.WaitingOnObject("ServerGroup", serverGroupKey.Name, progress.WaitingOnReady)
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
		Name:      getResourceName(obj),
		ImageRef:  *image.Status.ID,
		FlavorRef: *flavor.Status.ID,
		Networks:  portList,
		UserData:  userData,
		Tags:      tags,
	}

	schedulerHints := servers.SchedulerHintOpts{
		Group: ptr.Deref(serverGroup.Status.ID, ""),
	}

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

func (actuator serverActuator) DeleteResource(ctx context.Context, obj orcObjectPT, osResource *servers.Server) progress.ReconcileStatus {
	log := ctrl.LoggerFrom(ctx)

	if osResource.Status != ServerStatusDeleted && osResource.Status != ServerStatusSoftDeleted {
		err := actuator.osClient.DeleteServer(ctx, osResource.ID)
		if err != nil {
			return progress.WrapError(err)
		}
		return progress.NewReconcileStatus()
	}

	// Delete attachments references
	for _, attachment := range osResource.AttachedVolumes {
		volume, err := actuator.findVolumeByID(ctx, obj.Namespace, attachment.ID)
		if err != nil {
			// Do nothing, there's nothing to refresh
			log.V(logging.Verbose).Info("Failed to get volume", "volume", attachment.ID, "err", err)
		} else {
			// Remove the reference to the server on the volume object, to trigger a refresh of it
			volumeAttachment := attachments.NewAttachableResource(
				volume, orcapplyconfigv1alpha1.Volume, orcapplyconfigv1alpha1.VolumeSpec)
			volumeAttachment.DetachFrom(ctx, actuator.k8sClient, controllerName, obj.Name)
		}
	}

	return progress.NewReconcileStatus()
}

var _ reconcileResourceActuator = serverActuator{}

func (actuator serverActuator) GetResourceReconcilers(ctx context.Context, orcObject orcObjectPT, osResource *osResourceT, controller interfaces.ResourceController) ([]resourceReconciler, progress.ReconcileStatus) {
	return []resourceReconciler{
		tags.ReconcileTags[orcObjectPT, osResourceT](orcObject.Spec.Resource.Tags, *osResource.Tags, tags.NewServerTagReplacer(actuator.osClient, osResource.ID)),
		actuator.checkStatus,
		actuator.updateResource,
		actuator.reconcileVolumeAttachments,
	}, nil
}

func (actuator serverActuator) updateResource(ctx context.Context, obj orcObjectPT, osResource *osResourceT) progress.ReconcileStatus {
	log := ctrl.LoggerFrom(ctx)
	resource := obj.Spec.Resource
	if resource == nil {
		return progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Update requested, but spec.resource is not set"))
	}

	updateOpts := &servers.UpdateOpts{}

	handleNameUpdate(updateOpts, obj, osResource)

	needsUpdate, err := needsUpdate(updateOpts)
	if err != nil {
		return progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration updating resource: "+err.Error(), err))
	}
	if !needsUpdate {
		log.V(logging.Debug).Info("No changes")
		return nil
	}

	_, err = actuator.osClient.UpdateServer(ctx, osResource.ID, updateOpts)

	if orcerrors.IsConflict(err) {
		err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration updating resource: "+err.Error(), err)
	}
	if err != nil {
		return progress.WrapError(err)
	}

	return progress.NeedsRefresh()
}

func needsUpdate(updateOpts servers.UpdateOptsBuilder) (bool, error) {
	updateOptsMap, err := updateOpts.ToServerUpdateMap()
	if err != nil {
		return false, err
	}

	serverUpdateMap, ok := updateOptsMap["server"].(map[string]any)
	if !ok {
		serverUpdateMap = make(map[string]any)
	}

	return len(serverUpdateMap) > 0, nil
}

func handleNameUpdate(updateOpts *servers.UpdateOpts, obj orcObjectPT, osResource *osResourceT) {
	name := getResourceName(obj)
	if osResource.Name != name {
		updateOpts.Name = name
	}
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

func (actuator serverActuator) reconcileVolumeAttachments(ctx context.Context, obj orcObjectPT, osResource *osResourceT) progress.ReconcileStatus {
	log := ctrl.LoggerFrom(ctx)
	resource := obj.Spec.Resource
	if resource == nil {
		// Should have been caught by API validation
		return progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Update requested, but spec.resource is not set"))
	}

	volumeDepsMap, reconcileStatus := volumeDependency.GetDependencies(
		ctx, actuator.k8sClient, obj, func(volume *orcv1alpha1.Volume) bool {
			return orcv1alpha1.IsAvailable(volume) && volume.Status.ID != nil
		},
	)

	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return reconcileStatus
	}

	// Only operate on servers in available status
	if osResource.Status != ServerStatusActive {
		return progress.NewReconcileStatus().WaitingOnOpenStack(progress.WaitingOnReady, serverActivePollingPeriod)
	}

	// Create missing attachments
	for i := range resource.Volumes {
		volumeRef := resource.Volumes[i].VolumeRef
		if volume, ok := volumeDepsMap[string(volumeRef)]; !ok {
			// Programming error
			return reconcileStatus.WithError(fmt.Errorf("volume %s not present in dependencies", volumeRef))
		} else {
			// The volume is not yet attached to the server
			if !slices.ContainsFunc(osResource.AttachedVolumes, func(attachment servers.AttachedVolume) bool {
				return attachment.ID == *volume.Status.ID
			}) {
				createOpts := volumeattach.CreateOpts{
					VolumeID: *volume.Status.ID,
				}
				log.V(logging.Verbose).Info("Attaching volume to server", "volume", *volume.Status.ID, "server", *obj.Status.ID)
				_, err := actuator.osClient.CreateVolumeAttachment(ctx, *obj.Status.ID, createOpts)
				if err != nil {
					return reconcileStatus.WithReconcileStatus(progress.WrapError(err))
				}

				// Give time for the change to be reflected on the server resource before next reconcile
				reconcileStatus = reconcileStatus.WithReconcileStatus(
					progress.NewReconcileStatus().WaitingOnOpenStack(progress.WaitingOnReady, serverAttachmentPollingPeriod))

				// Add a reference to the server on the volume object, to trigger a refresh of it
				volumeAttachment := attachments.NewAttachableResource(
					volume, orcapplyconfigv1alpha1.Volume, orcapplyconfigv1alpha1.VolumeSpec)
				volumeAttachment.AttachTo(ctx, actuator.k8sClient, controllerName, obj.Name)
			}
		}
	}

	volumeDeps := slices.Collect(maps.Values(volumeDepsMap))

	// Delete extra attachments
	for _, attachment := range osResource.AttachedVolumes {
		// There's a attachment that is not marked as a dependency
		if !slices.ContainsFunc(volumeDeps, func(v *orcv1alpha1.Volume) bool {
			return attachment.ID == *v.Status.ID
		}) {
			log.V(logging.Verbose).Info("Detaching volume from server", "volume", attachment.ID, "server", *obj.Status.ID)
			err := actuator.osClient.DeleteVolumeAttachment(ctx, *obj.Status.ID, attachment.ID)
			if err != nil {
				return reconcileStatus.WithReconcileStatus(progress.WrapError(err))
			}

			// Give time for the change to be reflected on the server resource before next reconcile
			reconcileStatus = reconcileStatus.WithReconcileStatus(
				progress.NewReconcileStatus().WaitingOnOpenStack(progress.WaitingOnReady, serverAttachmentPollingPeriod))

			volume, err := actuator.findVolumeByID(ctx, obj.Namespace, attachment.ID)
			if err != nil {
				// Do nothing, there's nothing to refresh
				log.V(logging.Verbose).Info("Failed to get volume", "volume", attachment.ID, "err", err)
			} else {
				// Remove the reference to the server on the volume object, to trigger a refresh of it
				volumeAttachment := attachments.NewAttachableResource(
					volume, orcapplyconfigv1alpha1.Volume, orcapplyconfigv1alpha1.VolumeSpec)
				volumeAttachment.DetachFrom(ctx, actuator.k8sClient, controllerName, obj.Name)
			}
		}
	}

	return reconcileStatus
}

func (actuator serverActuator) findVolumeByID(ctx context.Context, namespace string, volumeID string) (*orcv1alpha1.Volume, error) {
	var volumeList = orcv1alpha1.VolumeList{}
	if err := actuator.k8sClient.List(ctx, &volumeList, client.InNamespace(namespace)); err != nil {
		return nil, err
	}

	for _, v := range volumeList.GetItems() {
		if v.Status.ID != nil && *v.Status.ID == volumeID {
			return &v, nil
		}
	}
	return nil, fmt.Errorf("no volume found with ID %s", volumeID)
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
