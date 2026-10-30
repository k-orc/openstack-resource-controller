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

package listener

import (
	"context"
	"iter"
	"maps"
	"slices"
	"time"

	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/listeners"
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
	osResourceT = listeners.Listener

	createResourceActuator    = interfaces.CreateResourceActuator[orcObjectPT, orcObjectT, filterT, osResourceT]
	deleteResourceActuator    = interfaces.DeleteResourceActuator[orcObjectPT, orcObjectT, osResourceT]
	reconcileResourceActuator = interfaces.ReconcileResourceActuator[orcObjectPT, osResourceT]
	resourceReconciler        = interfaces.ResourceReconciler[orcObjectPT, osResourceT]
	helperFactory             = interfaces.ResourceHelperFactory[orcObjectPT, orcObjectT, resourceSpecT, filterT, osResourceT]
)

// The frequency to poll when waiting for the resource to become available
const listenerAvailablePollingPeriod = 15 * time.Second

// The frequency to poll when waiting for the resource to be deleted
const listenerDeletingPollingPeriod = 15 * time.Second

type listenerActuator struct {
	osClient  osclients.ListenerClient
	k8sClient client.Client
}

var _ createResourceActuator = listenerActuator{}
var _ deleteResourceActuator = listenerActuator{}
var _ reconcileResourceActuator = listenerActuator{}

func (listenerActuator) GetResourceID(osResource *osResourceT) string {
	return osResource.ID
}

func (actuator listenerActuator) GetOSResourceByID(ctx context.Context, id string) (*osResourceT, progress.ReconcileStatus) {
	resource, err := actuator.osClient.GetListener(ctx, id)
	if err != nil {
		return nil, progress.WrapError(err)
	}
	return resource, nil
}

func (actuator listenerActuator) ListOSResourcesForAdoption(ctx context.Context, orcObject orcObjectPT) (iter.Seq2[*osResourceT, error], bool) {
	resourceSpec := orcObject.Spec.Resource
	if resourceSpec == nil {
		return nil, false
	}

	loadBalancer, rs := dependency.FetchDependency[*orcv1alpha1.LoadBalancer](
		ctx, actuator.k8sClient, orcObject.Namespace, &resourceSpec.LoadBalancerRef, "LoadBalancer",
		orcv1alpha1.IsAvailable,
	)
	if needsReschedule, _ := rs.NeedsReschedule(); needsReschedule {
		return nil, false
	}

	listOpts := listeners.ListOpts{
		Name:           getResourceName(orcObject),
		LoadbalancerID: ptr.Deref(loadBalancer.Status.ID, ""),
		Protocol:       string(resourceSpec.Protocol),
		ProtocolPort:   int(resourceSpec.ProtocolPort),
	}

	return actuator.osClient.ListListeners(ctx, listOpts), true
}

func (actuator listenerActuator) ListOSResourcesForImport(ctx context.Context, obj orcObjectPT, filter filterT) (iter.Seq2[*osResourceT, error], progress.ReconcileStatus) {
	var reconcileStatus progress.ReconcileStatus

	loadBalancer, rs := dependency.FetchDependency[*orcv1alpha1.LoadBalancer, orcv1alpha1.LoadBalancer](
		ctx, actuator.k8sClient, obj.Namespace,
		filter.LoadBalancerRef, "LoadBalancer",
		func(dep *orcv1alpha1.LoadBalancer) bool { return orcv1alpha1.IsAvailable(dep) && dep.Status.ID != nil },
	)
	reconcileStatus = reconcileStatus.WithReconcileStatus(rs)

	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return nil, reconcileStatus
	}

	listOpts := listeners.ListOpts{
		Name:           string(ptr.Deref(filter.Name, "")),
		LoadbalancerID: ptr.Deref(loadBalancer.Status.ID, ""),
		Protocol:       string(ptr.Deref(filter.Protocol, "")),
		ProtocolPort:   int(ptr.Deref(filter.ProtocolPort, 0)),
		Tags:           tagsToStrings(filter.Tags),
	}

	// The listener API has no server-side filter for these.
	var filters []osclients.ResourceFilter[osResourceT]
	if filter.Description != nil {
		filters = append(filters, func(l *osResourceT) bool { return l.Description == *filter.Description })
	}
	if len(filter.TagsAny) > 0 {
		filters = append(filters, func(l *osResourceT) bool { return hasAnyTag(l.Tags, filter.TagsAny) })
	}
	if len(filter.NotTags) > 0 {
		filters = append(filters, func(l *osResourceT) bool { return !hasAllTags(l.Tags, filter.NotTags) })
	}
	if len(filter.NotTagsAny) > 0 {
		filters = append(filters, func(l *osResourceT) bool { return !hasAnyTag(l.Tags, filter.NotTagsAny) })
	}

	return osclients.Filter(actuator.osClient.ListListeners(ctx, listOpts), filters...), reconcileStatus
}

func tagsToStrings(tags []orcv1alpha1.ListenerTag) []string {
	if len(tags) == 0 {
		return nil
	}
	out := make([]string, len(tags))
	for i := range tags {
		out[i] = string(tags[i])
	}
	return out
}

func hasAllTags(have []string, want []orcv1alpha1.ListenerTag) bool {
	for _, tag := range want {
		if !slices.Contains(have, string(tag)) {
			return false
		}
	}
	return true
}

func hasAnyTag(have []string, want []orcv1alpha1.ListenerTag) bool {
	for _, tag := range want {
		if slices.Contains(have, string(tag)) {
			return true
		}
	}
	return false
}

func (actuator listenerActuator) CreateResource(ctx context.Context, obj orcObjectPT) (*osResourceT, progress.ReconcileStatus) {
	resource := obj.Spec.Resource
	var reconcileStatus progress.ReconcileStatus

	var loadBalancerID string
	loadBalancer, loadBalancerDepRS := loadBalancerDependency.RequireDependency(
		ctx, actuator.k8sClient, obj, func(dep *orcv1alpha1.LoadBalancer) bool {
			return orcv1alpha1.IsAvailable(dep) && dep.Status.ID != nil
		},
	)
	reconcileStatus = reconcileStatus.WithReconcileStatus(loadBalancerDepRS)
	if loadBalancer != nil {
		loadBalancerID = ptr.Deref(loadBalancer.Status.ID, "")
	}

	var defaultPoolID string
	if resource.DefaultPoolRef != nil {
		defaultPool, defaultPoolDepRS := defaultPoolDependency.RequireDependency(
			ctx, actuator.k8sClient, obj, func(dep *orcv1alpha1.LBPool) bool {
				return orcv1alpha1.IsAvailable(dep) && dep.Status.ID != nil
			},
		)
		reconcileStatus = reconcileStatus.WithReconcileStatus(defaultPoolDepRS)
		if defaultPool != nil {
			defaultPoolID = ptr.Deref(defaultPool.Status.ID, "")
		}
	}

	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return nil, reconcileStatus
	}

	createOpts := listeners.CreateOpts{
		Name:                    getResourceName(obj),
		Description:             ptr.Deref(resource.Description, ""),
		LoadbalancerID:          loadBalancerID,
		DefaultPoolID:           defaultPoolID,
		Protocol:                listeners.Protocol(resource.Protocol),
		ProtocolPort:            int(resource.ProtocolPort),
		AdminStateUp:            resource.AdminStateUp,
		ConnLimit:               int32PtrToInt(resource.ConnectionLimit),
		TimeoutClientData:       int32PtrToInt(resource.TimeoutClientData),
		TimeoutMemberConnect:    int32PtrToInt(resource.TimeoutMemberConnect),
		TimeoutMemberData:       int32PtrToInt(resource.TimeoutMemberData),
		TimeoutTCPInspect:       int32PtrToInt(resource.TimeoutTCPInspect),
		InsertHeaders:           resource.InsertHeaders,
		AllowedCIDRs:            resource.AllowedCIDRs,
		DefaultTlsContainerRef:  ptr.Deref(resource.DefaultTLSContainerURL, ""),
		SniContainerRefs:        resource.SNIContainerURLs,
		TLSCiphers:              ptr.Deref(resource.TLSCiphers, ""),
		TLSVersions:             tlsVersions(resource.TLSVersions),
		ALPNProtocols:           resource.ALPNProtocols,
		ClientAuthentication:    listeners.ClientAuthentication(ptr.Deref(resource.ClientAuthentication, "")),
		ClientCATLSContainerRef: ptr.Deref(resource.ClientCATLSContainerURL, ""),
		ClientCRLContainerRef:   ptr.Deref(resource.ClientCRLContainerURL, ""),
		Tags:                    tagsToStrings(resource.Tags),
	}
	if hsts := resource.HSTS; hsts != nil {
		createOpts.HSTSMaxAge = int(ptr.Deref(hsts.MaxAge, 0))
		createOpts.HSTSIncludeSubdomains = ptr.Deref(hsts.IncludeSubDomains, false)
		createOpts.HSTSPreload = ptr.Deref(hsts.Preload, false)
	}

	osResource, err := actuator.osClient.CreateListener(ctx, createOpts)
	if err != nil {
		// 409 Conflict typically means the LoadBalancer is in a pending state (immutable).
		// Wait for it to become available and retry.
		if orcerrors.IsConflict(err) {
			return nil, progress.WaitingOnOpenStack(progress.WaitingOnReady, listenerAvailablePollingPeriod)
		}
		if !orcerrors.IsRetryable(err) {
			err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration creating resource: "+err.Error(), err)
		}
		return nil, progress.WrapError(err)
	}

	return osResource, nil
}

func (actuator listenerActuator) DeleteResource(ctx context.Context, _ orcObjectPT, resource *osResourceT) progress.ReconcileStatus {
	switch resource.ProvisioningStatus {
	case ListenerProvisioningStatusPendingDelete:
		return progress.WaitingOnOpenStack(progress.WaitingOnDeletion, listenerDeletingPollingPeriod)
	case ListenerProvisioningStatusPendingCreate, ListenerProvisioningStatusPendingUpdate:
		// We can't delete a listener that's in a pending state, so we need to wait for it to become ACTIVE
		return progress.WaitingOnOpenStack(progress.WaitingOnReady, listenerDeletingPollingPeriod)
	}

	err := actuator.osClient.DeleteListener(ctx, resource.ID)
	// 409 Conflict means the listener or its load balancer is in a pending state.
	// Let the controller poll for deletion completion.
	if orcerrors.IsConflict(err) {
		return progress.WaitingOnOpenStack(progress.WaitingOnDeletion, listenerDeletingPollingPeriod)
	}

	return progress.WrapError(err)
}

func (actuator listenerActuator) updateResource(ctx context.Context, obj orcObjectPT, osResource *osResourceT) progress.ReconcileStatus {
	log := ctrl.LoggerFrom(ctx)

	// Octavia rejects updates to a listener in a pending state
	switch osResource.ProvisioningStatus {
	case ListenerProvisioningStatusPendingCreate,
		ListenerProvisioningStatusPendingUpdate,
		ListenerProvisioningStatusPendingDelete:
		return progress.WaitingOnOpenStack(progress.WaitingOnReady, listenerAvailablePollingPeriod)
	}

	resource := obj.Spec.Resource

	updateOpts := listeners.UpdateOpts{}

	handleNameUpdate(&updateOpts, obj, osResource)
	handleDescriptionUpdate(&updateOpts, resource, osResource)
	handleAdminStateUpdate(&updateOpts, resource, osResource)
	handleConnectionLimitUpdate(&updateOpts, resource, osResource)
	handleTimeoutsUpdate(&updateOpts, resource, osResource)
	handleInsertHeadersUpdate(&updateOpts, resource, osResource)
	handleAllowedCIDRsUpdate(&updateOpts, resource, osResource)
	handleTLSUpdate(&updateOpts, resource, osResource)
	handleHSTSUpdate(&updateOpts, resource, osResource)
	handleTagsUpdate(&updateOpts, resource, osResource)

	// A pool created with listenerRef sets the default pool itself, so only update it when it's set.
	if resource.DefaultPoolRef != nil {
		defaultPool, rs := dependency.FetchDependency(
			ctx, actuator.k8sClient, obj.Namespace, resource.DefaultPoolRef, "LBPool",
			func(dep *orcv1alpha1.LBPool) bool { return orcv1alpha1.IsAvailable(dep) && dep.Status.ID != nil },
		)
		if needsReschedule, _ := rs.NeedsReschedule(); needsReschedule {
			return rs
		}
		if defaultPoolID := ptr.Deref(defaultPool.Status.ID, ""); defaultPoolID != osResource.DefaultPoolID {
			updateOpts.DefaultPoolID = &defaultPoolID
		}
	}

	needsUpdate, err := needsUpdate(updateOpts)
	if err != nil {
		return progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration updating resource: "+err.Error(), err))
	}
	if !needsUpdate {
		log.V(logging.Debug).Info("No changes")
		return nil
	}

	_, err = actuator.osClient.UpdateListener(ctx, osResource.ID, updateOpts)
	if err != nil {
		// 409 Conflict means the listener or its load balancer is in a pending state.
		if orcerrors.IsConflict(err) {
			return progress.WaitingOnOpenStack(progress.WaitingOnReady, listenerAvailablePollingPeriod)
		}
		if !orcerrors.IsRetryable(err) {
			err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration updating resource: "+err.Error(), err)
		}
		return progress.WrapError(err)
	}

	return progress.NeedsRefresh()
}

func needsUpdate(updateOpts listeners.UpdateOpts) (bool, error) {
	updateOptsMap, err := updateOpts.ToListenerUpdateMap()
	if err != nil {
		return false, err
	}

	updateMap, ok := updateOptsMap["listener"].(map[string]any)
	if !ok {
		updateMap = make(map[string]any)
	}

	return len(updateMap) > 0, nil
}

func handleNameUpdate(updateOpts *listeners.UpdateOpts, obj orcObjectPT, osResource *osResourceT) {
	name := getResourceName(obj)
	if osResource.Name != name {
		updateOpts.Name = &name
	}
}

func handleDescriptionUpdate(updateOpts *listeners.UpdateOpts, resource *resourceSpecT, osResource *osResourceT) {
	description := ptr.Deref(resource.Description, "")
	if osResource.Description != description {
		updateOpts.Description = &description
	}
}

func handleAdminStateUpdate(updateOpts *listeners.UpdateOpts, resource *resourceSpecT, osResource *osResourceT) {
	// Default to true if not specified (OpenStack default)
	adminStateUp := ptr.Deref(resource.AdminStateUp, true)
	if osResource.AdminStateUp != adminStateUp {
		updateOpts.AdminStateUp = &adminStateUp
	}
}

func handleConnectionLimitUpdate(updateOpts *listeners.UpdateOpts, resource *resourceSpecT, osResource *osResourceT) {
	// Default to -1 (unlimited) if not specified (OpenStack default)
	connLimit := int(ptr.Deref(resource.ConnectionLimit, -1))
	if osResource.ConnLimit != connLimit {
		updateOpts.ConnLimit = &connLimit
	}
}

// Timeout defaults are set by the Octavia deployment, so we only update the ones that are set.
func handleTimeoutsUpdate(updateOpts *listeners.UpdateOpts, resource *resourceSpecT, osResource *osResourceT) {
	updateIfSet := func(desired *int32, current int, opt **int) {
		if desired != nil && int(*desired) != current {
			*opt = ptr.To(int(*desired))
		}
	}
	updateIfSet(resource.TimeoutClientData, osResource.TimeoutClientData, &updateOpts.TimeoutClientData)
	updateIfSet(resource.TimeoutMemberConnect, osResource.TimeoutMemberConnect, &updateOpts.TimeoutMemberConnect)
	updateIfSet(resource.TimeoutMemberData, osResource.TimeoutMemberData, &updateOpts.TimeoutMemberData)
	updateIfSet(resource.TimeoutTCPInspect, osResource.TimeoutTCPInspect, &updateOpts.TimeoutTCPInspect)
}

func handleInsertHeadersUpdate(updateOpts *listeners.UpdateOpts, resource *resourceSpecT, osResource *osResourceT) {
	if !maps.Equal(resource.InsertHeaders, osResource.InsertHeaders) {
		insertHeaders := resource.InsertHeaders
		if insertHeaders == nil {
			insertHeaders = map[string]string{}
		}
		updateOpts.InsertHeaders = &insertHeaders
	}
}

func handleAllowedCIDRsUpdate(updateOpts *listeners.UpdateOpts, resource *resourceSpecT, osResource *osResourceT) {
	if !orcslices.EqualAsSets(resource.AllowedCIDRs, osResource.AllowedCIDRs) {
		allowedCIDRs := append([]string{}, resource.AllowedCIDRs...)
		updateOpts.AllowedCIDRs = &allowedCIDRs
	}
}

// The TLS settings only apply to TERMINATED_HTTPS listeners, and some have deployment defaults,
// so apart from the SNI list we only update the ones that are set.
func handleTLSUpdate(updateOpts *listeners.UpdateOpts, resource *resourceSpecT, osResource *osResourceT) {
	updateIfSet := func(desired *string, current string, opt **string) {
		if desired != nil && *desired != current {
			*opt = ptr.To(*desired)
		}
	}
	updateIfSet(resource.DefaultTLSContainerURL, osResource.DefaultTlsContainerRef, &updateOpts.DefaultTlsContainerRef)
	updateIfSet(resource.TLSCiphers, osResource.TLSCiphers, &updateOpts.TLSCiphers)
	updateIfSet(resource.ClientCATLSContainerURL, osResource.ClientCATLSContainerRef, &updateOpts.ClientCATLSContainerRef)
	updateIfSet(resource.ClientCRLContainerURL, osResource.ClientCRLContainerRef, &updateOpts.ClientCRLContainerRef)

	if !orcslices.EqualAsSets(resource.SNIContainerURLs, osResource.SniContainerRefs) {
		sniContainerRefs := append([]string{}, resource.SNIContainerURLs...)
		updateOpts.SniContainerRefs = &sniContainerRefs
	}

	if resource.TLSVersions != nil {
		desired := tlsVersions(resource.TLSVersions)
		current := make([]listeners.TLSVersion, len(osResource.TLSVersions))
		for i := range osResource.TLSVersions {
			current[i] = listeners.TLSVersion(osResource.TLSVersions[i])
		}
		if !orcslices.EqualAsSets(desired, current) {
			updateOpts.TLSVersions = &desired
		}
	}

	if resource.ALPNProtocols != nil && !orcslices.EqualAsSets(resource.ALPNProtocols, osResource.ALPNProtocols) {
		alpnProtocols := append([]string{}, resource.ALPNProtocols...)
		updateOpts.ALPNProtocols = &alpnProtocols
	}

	if resource.ClientAuthentication != nil && string(*resource.ClientAuthentication) != osResource.ClientAuthentication {
		updateOpts.ClientAuthentication = ptr.To(listeners.ClientAuthentication(*resource.ClientAuthentication))
	}
}

func handleHSTSUpdate(updateOpts *listeners.UpdateOpts, resource *resourceSpecT, osResource *osResourceT) {
	hsts := resource.HSTS
	if hsts == nil {
		return
	}
	if hsts.MaxAge != nil && int(*hsts.MaxAge) != osResource.HSTSMaxAge {
		updateOpts.HSTSMaxAge = ptr.To(int(*hsts.MaxAge))
	}
	if hsts.IncludeSubDomains != nil && *hsts.IncludeSubDomains != osResource.HSTSIncludeSubdomains {
		updateOpts.HSTSIncludeSubdomains = ptr.To(*hsts.IncludeSubDomains)
	}
	if hsts.Preload != nil && *hsts.Preload != osResource.HSTSPreload {
		updateOpts.HSTSPreload = ptr.To(*hsts.Preload)
	}
}

func handleTagsUpdate(updateOpts *listeners.UpdateOpts, resource *resourceSpecT, osResource *osResourceT) {
	desiredTags := tagsToStrings(resource.Tags)
	if !orcslices.EqualAsSets(desiredTags, osResource.Tags) {
		if desiredTags == nil {
			desiredTags = []string{}
		}
		updateOpts.Tags = &desiredTags
	}
}

func int32PtrToInt(v *int32) *int {
	if v == nil {
		return nil
	}
	return ptr.To(int(*v))
}

func tlsVersions(versions []orcv1alpha1.ListenerTLSVersion) []listeners.TLSVersion {
	if len(versions) == 0 {
		return nil
	}
	out := make([]listeners.TLSVersion, len(versions))
	for i := range versions {
		out[i] = listeners.TLSVersion(versions[i])
	}
	return out
}

func (actuator listenerActuator) GetResourceReconcilers(ctx context.Context, orcObject orcObjectPT, osResource *osResourceT, controller interfaces.ResourceController) ([]resourceReconciler, progress.ReconcileStatus) {
	return []resourceReconciler{
		actuator.reconcileDependencyGuards,
		actuator.updateResource,
	}, nil
}

// reconcileDependencyGuards adds the deletion guards to the listener's dependencies. CreateResource
// adds them before creating; this also covers an adopted listener.
func (actuator listenerActuator) reconcileDependencyGuards(ctx context.Context, obj orcObjectPT, _ *osResourceT) progress.ReconcileStatus {
	var reconcileStatus progress.ReconcileStatus
	_, rs := loadBalancerDependency.RequireDependencies(ctx, actuator.k8sClient, obj, orcv1alpha1.IsAvailable)
	reconcileStatus = reconcileStatus.WithReconcileStatus(rs)
	_, rs = defaultPoolDependency.RequireDependencies(ctx, actuator.k8sClient, obj, orcv1alpha1.IsAvailable)
	return reconcileStatus.WithReconcileStatus(rs)
}

type listenerHelperFactory struct{}

var _ helperFactory = listenerHelperFactory{}

func newActuator(ctx context.Context, orcObject *orcv1alpha1.Listener, controller interfaces.ResourceController) (listenerActuator, progress.ReconcileStatus) {
	log := ctrl.LoggerFrom(ctx)

	// Ensure credential secrets exist and have our finalizer
	_, reconcileStatus := credentialsDependency.RequireDependencies(ctx, controller.GetK8sClient(), orcObject, func(*corev1.Secret) bool { return true })
	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return listenerActuator{}, reconcileStatus
	}

	clientScope, err := controller.GetScopeFactory().NewClientScopeFromObject(ctx, controller.GetK8sClient(), log, orcObject)
	if err != nil {
		return listenerActuator{}, progress.WrapError(err)
	}
	osClient, err := clientScope.NewListenerClient()
	if err != nil {
		return listenerActuator{}, progress.WrapError(err)
	}

	return listenerActuator{
		osClient:  osClient,
		k8sClient: controller.GetK8sClient(),
	}, nil
}

func (listenerHelperFactory) NewAPIObjectAdapter(obj orcObjectPT) adapterI {
	return listenerAdapter{obj}
}

func (listenerHelperFactory) NewCreateActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (createResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, orcObject, controller)
}

func (listenerHelperFactory) NewDeleteActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (deleteResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, orcObject, controller)
}
