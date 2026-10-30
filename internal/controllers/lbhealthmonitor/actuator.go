/*
Copyright The ORC Authors.

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

package lbhealthmonitor

import (
	"context"
	"iter"
	"time"

	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/monitors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v3/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v3/internal/controllers/generic/progress"
	"github.com/k-orc/openstack-resource-controller/v3/internal/logging"
	"github.com/k-orc/openstack-resource-controller/v3/internal/osclients"
	"github.com/k-orc/openstack-resource-controller/v3/internal/util/dependency"
	orcerrors "github.com/k-orc/openstack-resource-controller/v3/internal/util/errors"
	orcslices "github.com/k-orc/openstack-resource-controller/v3/internal/util/slices"
)

// OpenStack resource types
type (
	osResourceT = monitors.Monitor

	createResourceActuator    = interfaces.CreateResourceActuator[orcObjectPT, orcObjectT, filterT, osResourceT]
	deleteResourceActuator    = interfaces.DeleteResourceActuator[orcObjectPT, orcObjectT, osResourceT]
	reconcileResourceActuator = interfaces.ReconcileResourceActuator[orcObjectPT, osResourceT]
	resourceReconciler        = interfaces.ResourceReconciler[orcObjectPT, osResourceT]
	helperFactory             = interfaces.ResourceHelperFactory[orcObjectPT, orcObjectT, resourceSpecT, filterT, osResourceT]
)

// The frequency to poll when waiting for the resource to become available
const lbhealthmonitorAvailablePollingPeriod = 15 * time.Second

// The frequency to poll when waiting for the resource to be deleted
const lbhealthmonitorDeletingPollingPeriod = 15 * time.Second

type lbhealthmonitorActuator struct {
	osClient  osclients.LBHealthMonitorClient
	k8sClient client.Client
}

var _ createResourceActuator = lbhealthmonitorActuator{}
var _ deleteResourceActuator = lbhealthmonitorActuator{}
var _ reconcileResourceActuator = lbhealthmonitorActuator{}

func (lbhealthmonitorActuator) GetResourceID(osResource *osResourceT) string {
	return osResource.ID
}

func (actuator lbhealthmonitorActuator) GetOSResourceByID(ctx context.Context, id string) (*osResourceT, progress.ReconcileStatus) {
	resource, err := actuator.osClient.GetLBHealthMonitor(ctx, id)
	if err != nil {
		return nil, progress.WrapError(err)
	}
	return resource, nil
}

func (actuator lbhealthmonitorActuator) ListOSResourcesForAdoption(ctx context.Context, orcObject orcObjectPT) (iter.Seq2[*osResourceT, error], bool) {
	resourceSpec := orcObject.Spec.Resource
	if resourceSpec == nil {
		return nil, false
	}

	pool, rs := dependency.FetchDependency[*orcv1alpha1.LBPool](
		ctx, actuator.k8sClient, orcObject.Namespace, &resourceSpec.PoolRef, "LBPool",
		orcv1alpha1.IsAvailable,
	)
	if needsReschedule, _ := rs.NeedsReschedule(); needsReschedule {
		return nil, false
	}

	listOpts := monitors.ListOpts{
		Name:   getResourceName(orcObject),
		PoolID: ptr.Deref(pool.Status.ID, ""),
		Type:   string(resourceSpec.Type),
	}

	return actuator.osClient.ListLBHealthMonitors(ctx, listOpts), true
}

func (actuator lbhealthmonitorActuator) ListOSResourcesForImport(ctx context.Context, obj orcObjectPT, filter filterT) (iter.Seq2[*osResourceT, error], progress.ReconcileStatus) {
	pool, reconcileStatus := dependency.FetchDependency[*orcv1alpha1.LBPool](
		ctx, actuator.k8sClient, obj.Namespace,
		filter.PoolRef, "LBPool",
		orcv1alpha1.IsAvailable,
	)
	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return nil, reconcileStatus
	}

	listOpts := monitors.ListOpts{
		Name:   string(ptr.Deref(filter.Name, "")),
		PoolID: ptr.Deref(pool.Status.ID, ""),
		Type:   string(ptr.Deref(filter.Type, "")),
		Tags:   tagsToStrings(filter.Tags),
	}

	return actuator.osClient.ListLBHealthMonitors(ctx, listOpts), reconcileStatus
}

func (actuator lbhealthmonitorActuator) CreateResource(ctx context.Context, obj orcObjectPT) (*osResourceT, progress.ReconcileStatus) {
	resource := obj.Spec.Resource

	pool, reconcileStatus := poolDependency.RequireDependency(
		ctx, actuator.k8sClient, obj, func(dep *orcv1alpha1.LBPool) bool {
			return orcv1alpha1.IsAvailable(dep) && dep.Status.ID != nil
		},
	)
	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return nil, reconcileStatus
	}

	createOpts := monitors.CreateOpts{
		Name:           getResourceName(obj),
		PoolID:         ptr.Deref(pool.Status.ID, ""),
		Type:           string(resource.Type),
		Delay:          int(resource.Delay),
		Timeout:        int(resource.Timeout),
		MaxRetries:     int(resource.MaxRetries),
		MaxRetriesDown: int(ptr.Deref(resource.MaxRetriesDown, 0)),
		HTTPMethod:     string(ptr.Deref(resource.HTTPMethod, "")),
		HTTPVersion:    string(ptr.Deref(resource.HTTPVersion, "")),
		ExpectedCodes:  ptr.Deref(resource.ExpectedCodes, ""),
		URLPath:        ptr.Deref(resource.URLPath, ""),
		DomainName:     ptr.Deref(resource.DomainName, ""),
		AdminStateUp:   resource.AdminStateUp,
		Tags:           tagsToStrings(resource.Tags),
	}

	osResource, err := actuator.osClient.CreateLBHealthMonitor(ctx, createOpts)
	if err != nil {
		// 409 Conflict means the pool or its load balancer is in a pending state.
		if orcerrors.IsConflict(err) {
			return nil, progress.WaitingOnOpenStack(progress.WaitingOnReady, lbhealthmonitorAvailablePollingPeriod)
		}
		if !orcerrors.IsRetryable(err) {
			err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration creating resource: "+err.Error(), err)
		}
		return nil, progress.WrapError(err)
	}

	return osResource, nil
}

func (actuator lbhealthmonitorActuator) DeleteResource(ctx context.Context, _ orcObjectPT, resource *osResourceT) progress.ReconcileStatus {
	switch resource.ProvisioningStatus {
	case MonitorProvisioningStatusPendingDelete:
		return progress.WaitingOnOpenStack(progress.WaitingOnDeletion, lbhealthmonitorDeletingPollingPeriod)
	case MonitorProvisioningStatusPendingCreate, MonitorProvisioningStatusPendingUpdate:
		// We can't delete a health monitor that's in a pending state, so we need to wait for it to become ACTIVE
		return progress.WaitingOnOpenStack(progress.WaitingOnReady, lbhealthmonitorDeletingPollingPeriod)
	}

	err := actuator.osClient.DeleteLBHealthMonitor(ctx, resource.ID)
	// 409 Conflict means the health monitor or its load balancer is in a pending state.
	// Let the controller poll for deletion completion.
	if orcerrors.IsConflict(err) {
		return progress.WaitingOnOpenStack(progress.WaitingOnDeletion, lbhealthmonitorDeletingPollingPeriod)
	}
	return progress.WrapError(err)
}

func (actuator lbhealthmonitorActuator) updateResource(ctx context.Context, obj orcObjectPT, osResource *osResourceT) progress.ReconcileStatus {
	log := ctrl.LoggerFrom(ctx)

	// Octavia rejects updates to a health monitor in a pending state
	switch osResource.ProvisioningStatus {
	case MonitorProvisioningStatusPendingCreate,
		MonitorProvisioningStatusPendingUpdate,
		MonitorProvisioningStatusPendingDelete:
		return progress.WaitingOnOpenStack(progress.WaitingOnReady, lbhealthmonitorAvailablePollingPeriod)
	}

	resource := obj.Spec.Resource

	updateOpts := monitors.UpdateOpts{}

	handleNameUpdate(&updateOpts, obj, osResource)
	handleUpdates(&updateOpts, resource, osResource)

	needsUpdate, err := needsUpdate(updateOpts)
	if err != nil {
		return progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration updating resource: "+err.Error(), err))
	}
	if !needsUpdate {
		log.V(logging.Debug).Info("No changes")
		return nil
	}

	_, err = actuator.osClient.UpdateLBHealthMonitor(ctx, osResource.ID, updateOpts)
	if err != nil {
		// 409 Conflict means the health monitor or its load balancer is in a pending state.
		if orcerrors.IsConflict(err) {
			return progress.WaitingOnOpenStack(progress.WaitingOnReady, lbhealthmonitorAvailablePollingPeriod)
		}
		if !orcerrors.IsRetryable(err) {
			err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration updating resource: "+err.Error(), err)
		}
		return progress.WrapError(err)
	}

	return progress.NeedsRefresh()
}

func needsUpdate(updateOpts monitors.UpdateOpts) (bool, error) {
	updateOptsMap, err := updateOpts.ToMonitorUpdateMap()
	if err != nil {
		return false, err
	}

	updateMap, ok := updateOptsMap["healthmonitor"].(map[string]any)
	if !ok {
		updateMap = make(map[string]any)
	}

	return len(updateMap) > 0, nil
}

func handleNameUpdate(updateOpts *monitors.UpdateOpts, obj orcObjectPT, osResource *osResourceT) {
	name := getResourceName(obj)
	if osResource.Name != name {
		updateOpts.Name = &name
	}
}

// handleUpdates updates the required fields and admin state, and the optional fields when they're
// set: Octavia fills in defaults for them, and gophercloud can't reset them or clear the tags.
func handleUpdates(updateOpts *monitors.UpdateOpts, resource *resourceSpecT, osResource *osResourceT) {
	if int(resource.Delay) != osResource.Delay {
		updateOpts.Delay = int(resource.Delay)
	}
	if int(resource.Timeout) != osResource.Timeout {
		updateOpts.Timeout = int(resource.Timeout)
	}
	if int(resource.MaxRetries) != osResource.MaxRetries {
		updateOpts.MaxRetries = int(resource.MaxRetries)
	}
	if resource.MaxRetriesDown != nil && int(*resource.MaxRetriesDown) != osResource.MaxRetriesDown {
		updateOpts.MaxRetriesDown = int(*resource.MaxRetriesDown)
	}
	if resource.HTTPMethod != nil && string(*resource.HTTPMethod) != osResource.HTTPMethod {
		updateOpts.HTTPMethod = string(*resource.HTTPMethod)
	}
	if resource.HTTPVersion != nil && string(*resource.HTTPVersion) != osResource.HTTPVersion {
		updateOpts.HTTPVersion = ptr.To(string(*resource.HTTPVersion))
	}
	if resource.ExpectedCodes != nil && *resource.ExpectedCodes != osResource.ExpectedCodes {
		updateOpts.ExpectedCodes = *resource.ExpectedCodes
	}
	if resource.URLPath != nil && *resource.URLPath != osResource.URLPath {
		updateOpts.URLPath = *resource.URLPath
	}
	if resource.DomainName != nil && *resource.DomainName != osResource.DomainName {
		updateOpts.DomainName = ptr.To(*resource.DomainName)
	}
	if adminStateUp := ptr.Deref(resource.AdminStateUp, true); adminStateUp != osResource.AdminStateUp {
		updateOpts.AdminStateUp = &adminStateUp
	}
	if tags := tagsToStrings(resource.Tags); len(tags) > 0 && !orcslices.EqualAsSets(tags, osResource.Tags) {
		updateOpts.Tags = tags
	}
}

func tagsToStrings(tags []orcv1alpha1.LBPoolTag) []string {
	if len(tags) == 0 {
		return nil
	}
	out := make([]string, len(tags))
	for i := range tags {
		out[i] = string(tags[i])
	}
	return out
}

func (actuator lbhealthmonitorActuator) GetResourceReconcilers(ctx context.Context, orcObject orcObjectPT, osResource *osResourceT, controller interfaces.ResourceController) ([]resourceReconciler, progress.ReconcileStatus) {
	return []resourceReconciler{
		actuator.reconcileDependencyGuards,
		actuator.updateResource,
	}, nil
}

// reconcileDependencyGuards adds the deletion guard to the health monitor's pool. CreateResource
// adds it before creating; this also covers an adopted health monitor.
func (actuator lbhealthmonitorActuator) reconcileDependencyGuards(ctx context.Context, obj orcObjectPT, _ *osResourceT) progress.ReconcileStatus {
	_, reconcileStatus := poolDependency.RequireDependencies(ctx, actuator.k8sClient, obj, orcv1alpha1.IsAvailable)
	return reconcileStatus
}

type lbhealthmonitorHelperFactory struct{}

var _ helperFactory = lbhealthmonitorHelperFactory{}

func newActuator(ctx context.Context, orcObject *orcv1alpha1.LBHealthMonitor, controller interfaces.ResourceController) (lbhealthmonitorActuator, progress.ReconcileStatus) {
	log := ctrl.LoggerFrom(ctx)

	// Ensure credential secrets exist and have our finalizer
	_, reconcileStatus := credentialsDependency.RequireDependencies(ctx, controller.GetK8sClient(), orcObject, func(*corev1.Secret) bool { return true })
	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return lbhealthmonitorActuator{}, reconcileStatus
	}

	clientScope, err := controller.GetScopeFactory().NewClientScopeFromObject(ctx, controller.GetK8sClient(), log, orcObject)
	if err != nil {
		return lbhealthmonitorActuator{}, progress.WrapError(err)
	}
	osClient, err := clientScope.NewLBHealthMonitorClient()
	if err != nil {
		return lbhealthmonitorActuator{}, progress.WrapError(err)
	}

	return lbhealthmonitorActuator{
		osClient:  osClient,
		k8sClient: controller.GetK8sClient(),
	}, nil
}

func (lbhealthmonitorHelperFactory) NewAPIObjectAdapter(obj orcObjectPT) adapterI {
	return lbhealthmonitorAdapter{obj}
}

func (lbhealthmonitorHelperFactory) NewCreateActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (createResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, orcObject, controller)
}

func (lbhealthmonitorHelperFactory) NewDeleteActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (deleteResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, orcObject, controller)
}
