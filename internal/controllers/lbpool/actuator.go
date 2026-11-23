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

package lbpool

import (
	"context"
	"fmt"
	"iter"
	"slices"
	"time"

	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/pools"
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
	osResourceT = pools.Pool

	createResourceActuator    = interfaces.CreateResourceActuator[orcObjectPT, orcObjectT, filterT, osResourceT]
	deleteResourceActuator    = interfaces.DeleteResourceActuator[orcObjectPT, orcObjectT, osResourceT]
	reconcileResourceActuator = interfaces.ReconcileResourceActuator[orcObjectPT, osResourceT]
	resourceReconciler        = interfaces.ResourceReconciler[orcObjectPT, osResourceT]
	helperFactory             = interfaces.ResourceHelperFactory[orcObjectPT, orcObjectT, resourceSpecT, filterT, osResourceT]
)

// The frequency to poll when waiting for the resource to become available
const lbpoolAvailablePollingPeriod = 15 * time.Second

// The frequency to poll when waiting for the resource to be deleted
const lbpoolDeletingPollingPeriod = 15 * time.Second

// Provisioning status constants for LBPool
const (
	PoolProvisioningStatusActive        = "ACTIVE"
	PoolProvisioningStatusError         = "ERROR"
	PoolProvisioningStatusPendingCreate = "PENDING_CREATE"
	PoolProvisioningStatusPendingUpdate = "PENDING_UPDATE"
	PoolProvisioningStatusPendingDelete = "PENDING_DELETE"
)

type lbpoolActuator struct {
	osClient  osclients.LBPoolClient
	k8sClient client.Client
}

var _ createResourceActuator = lbpoolActuator{}
var _ deleteResourceActuator = lbpoolActuator{}
var _ reconcileResourceActuator = lbpoolActuator{}

func (lbpoolActuator) GetResourceID(osResource *osResourceT) string {
	return osResource.ID
}

func (actuator lbpoolActuator) GetOSResourceByID(ctx context.Context, id string) (*osResourceT, progress.ReconcileStatus) {
	resource, err := actuator.osClient.GetLBPool(ctx, id)
	if err != nil {
		return nil, progress.WrapError(err)
	}

	members := make([]pools.Member, 0, len(resource.Members))
	for member, err := range actuator.osClient.ListMembers(ctx, id, pools.ListMembersOpts{}) {
		if err != nil {
			return nil, progress.WrapError(err)
		}
		members = append(members, *member)
	}
	resource.Members = members

	return resource, nil
}

func (actuator lbpoolActuator) ListOSResourcesForAdoption(ctx context.Context, orcObject orcObjectPT) (iter.Seq2[*osResourceT, error], bool) {
	resourceSpec := orcObject.Spec.Resource
	if resourceSpec == nil {
		return nil, false
	}

	var rs progress.ReconcileStatus

	var loadBalancerID string
	if resourceSpec.LoadBalancerRef != nil {
		loadBalancer, rs1 := dependency.FetchDependency[*orcv1alpha1.LoadBalancer](
			ctx, actuator.k8sClient, orcObject.Namespace, resourceSpec.LoadBalancerRef, "LoadBalancer",
			orcv1alpha1.IsAvailable,
		)
		rs = rs.WithReconcileStatus(rs1)
		loadBalancerID = ptr.Deref(loadBalancer.Status.ID, "")
	}

	var listenerID string
	if resourceSpec.ListenerRef != nil {
		listener, rs1 := dependency.FetchDependency[*orcv1alpha1.Listener](
			ctx, actuator.k8sClient, orcObject.Namespace, resourceSpec.ListenerRef, "Listener",
			orcv1alpha1.IsAvailable,
		)
		rs = rs.WithReconcileStatus(rs1)
		listenerID = ptr.Deref(listener.Status.ID, "")
	}

	if needsReschedule, _ := rs.NeedsReschedule(); needsReschedule {
		return nil, false
	}

	listOpts := pools.ListOpts{
		Name:           getResourceName(orcObject),
		Protocol:       string(resourceSpec.Protocol),
		LoadbalancerID: loadBalancerID,
	}

	var filters []osclients.ResourceFilter[osResourceT]
	if listenerID != "" {
		filters = append(filters, func(p *osResourceT) bool { return hasListener(p, listenerID) })
	}

	return osclients.Filter(actuator.osClient.ListLBPools(ctx, listOpts), filters...), true
}

// The pool API has no listener filter.
func hasListener(pool *osResourceT, listenerID string) bool {
	return slices.ContainsFunc(pool.Listeners, func(l pools.ListenerID) bool { return l.ID == listenerID })
}

func (actuator lbpoolActuator) ListOSResourcesForImport(ctx context.Context, obj orcObjectPT, filter filterT) (iter.Seq2[*osResourceT, error], progress.ReconcileStatus) {
	var reconcileStatus progress.ReconcileStatus

	loadBalancer, rs := dependency.FetchDependency(
		ctx, actuator.k8sClient, obj.Namespace,
		filter.LoadBalancerRef, "LoadBalancer",
		func(dep *orcv1alpha1.LoadBalancer) bool { return orcv1alpha1.IsAvailable(dep) && dep.Status.ID != nil },
	)
	reconcileStatus = reconcileStatus.WithReconcileStatus(rs)

	listener, rs := dependency.FetchDependency(
		ctx, actuator.k8sClient, obj.Namespace,
		filter.ListenerRef, "Listener",
		func(dep *orcv1alpha1.Listener) bool { return orcv1alpha1.IsAvailable(dep) && dep.Status.ID != nil },
	)
	reconcileStatus = reconcileStatus.WithReconcileStatus(rs)

	project, rs := dependency.FetchDependency(
		ctx, actuator.k8sClient, obj.Namespace,
		filter.ProjectRef, "Project",
		func(dep *orcv1alpha1.Project) bool { return orcv1alpha1.IsAvailable(dep) && dep.Status.ID != nil },
	)
	reconcileStatus = reconcileStatus.WithReconcileStatus(rs)

	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return nil, reconcileStatus
	}

	listOpts := pools.ListOpts{
		Name:           string(ptr.Deref(filter.Name, "")),
		LoadbalancerID: ptr.Deref(loadBalancer.Status.ID, ""),
		ProjectID:      ptr.Deref(project.Status.ID, ""),
		LBMethod:       string(ptr.Deref(filter.LBAlgorithm, "")),
		Protocol:       string(ptr.Deref(filter.Protocol, "")),
		Tags:           tagsToStrings(filter.Tags),
	}

	// The pool API has no server-side filter for these.
	var filters []osclients.ResourceFilter[osResourceT]
	if listenerID := ptr.Deref(listener.Status.ID, ""); listenerID != "" {
		filters = append(filters, func(p *osResourceT) bool { return hasListener(p, listenerID) })
	}
	if filter.Description != nil {
		filters = append(filters, func(p *osResourceT) bool { return p.Description == *filter.Description })
	}
	if len(filter.TagsAny) > 0 {
		filters = append(filters, func(p *osResourceT) bool { return hasAnyTag(p.Tags, filter.TagsAny) })
	}
	if len(filter.NotTags) > 0 {
		filters = append(filters, func(p *osResourceT) bool { return !hasAllTags(p.Tags, filter.NotTags) })
	}
	if len(filter.NotTagsAny) > 0 {
		filters = append(filters, func(p *osResourceT) bool { return !hasAnyTag(p.Tags, filter.NotTagsAny) })
	}

	return osclients.Filter(actuator.osClient.ListLBPools(ctx, listOpts), filters...), reconcileStatus
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

func hasAllTags(have []string, want []orcv1alpha1.LBPoolTag) bool {
	for _, tag := range want {
		if !slices.Contains(have, string(tag)) {
			return false
		}
	}
	return true
}

func hasAnyTag(have []string, want []orcv1alpha1.LBPoolTag) bool {
	for _, tag := range want {
		if slices.Contains(have, string(tag)) {
			return true
		}
	}
	return false
}

func (actuator lbpoolActuator) CreateResource(ctx context.Context, obj orcObjectPT) (*osResourceT, progress.ReconcileStatus) {
	resource := obj.Spec.Resource
	var reconcileStatus progress.ReconcileStatus

	var loadBalancerID string
	if resource.LoadBalancerRef != nil {
		loadBalancer, loadBalancerDepRS := loadBalancerDependency.RequireDependency(
			ctx, actuator.k8sClient, obj, func(dep *orcv1alpha1.LoadBalancer) bool {
				return orcv1alpha1.IsAvailable(dep) && dep.Status.ID != nil
			},
		)
		reconcileStatus = reconcileStatus.WithReconcileStatus(loadBalancerDepRS)
		if loadBalancer != nil {
			loadBalancerID = ptr.Deref(loadBalancer.Status.ID, "")
		}
	}

	var listenerID string
	if resource.ListenerRef != nil {
		listener, listenerDepRS := listenerDependency.RequireDependency(
			ctx, actuator.k8sClient, obj, func(dep *orcv1alpha1.Listener) bool {
				return orcv1alpha1.IsAvailable(dep) && dep.Status.ID != nil
			},
		)
		reconcileStatus = reconcileStatus.WithReconcileStatus(listenerDepRS)
		if listener != nil {
			listenerID = ptr.Deref(listener.Status.ID, "")
		}
	}

	var projectID string
	if resource.ProjectRef != nil {
		project, projectDepRS := projectDependency.RequireDependency(
			ctx, actuator.k8sClient, obj, func(dep *orcv1alpha1.Project) bool {
				return orcv1alpha1.IsAvailable(dep) && dep.Status.ID != nil
			},
		)
		reconcileStatus = reconcileStatus.WithReconcileStatus(projectDepRS)
		if project != nil {
			projectID = ptr.Deref(project.Status.ID, "")
		}
	}

	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return nil, reconcileStatus
	}

	createOpts := pools.CreateOpts{
		Name:              getResourceName(obj),
		Description:       ptr.Deref(resource.Description, ""),
		LBMethod:          pools.LBMethod(resource.LBAlgorithm),
		Protocol:          pools.Protocol(resource.Protocol),
		LoadbalancerID:    loadBalancerID,
		ListenerID:        listenerID,
		ProjectID:         projectID,
		AdminStateUp:      resource.AdminStateUp,
		TLSEnabled:        ptr.Deref(resource.TLSEnabled, false),
		TLSContainerRef:   ptr.Deref(resource.TLSContainerURL, ""),
		CATLSContainerRef: ptr.Deref(resource.CATLSContainerURL, ""),
		CRLContainerRef:   ptr.Deref(resource.CRLContainerURL, ""),
		TLSCiphers:        ptr.Deref(resource.TLSCiphers, ""),
		TLSVersions:       tlsVersions(resource.TLSVersions),
		ALPNProtocols:     resource.ALPNProtocols,
		Tags:              tagsToStrings(resource.Tags),
	}

	if resource.SessionPersistence != nil {
		createOpts.Persistence = &pools.SessionPersistence{
			Type:       string(resource.SessionPersistence.Type),
			CookieName: ptr.Deref(resource.SessionPersistence.CookieName, ""),
		}
	}

	osResource, err := actuator.osClient.CreateLBPool(ctx, createOpts)
	if err != nil {
		// 409 Conflict typically means the LoadBalancer is in a pending state (immutable).
		// Wait for it to become available and retry.
		if orcerrors.IsConflict(err) {
			return nil, progress.WaitingOnOpenStack(progress.WaitingOnReady, lbpoolAvailablePollingPeriod)
		}
		// We should require the spec to be updated before retrying a create which returned a non-retryable error
		if !orcerrors.IsRetryable(err) {
			err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration creating resource: "+err.Error(), err)
		}
		return nil, progress.WrapError(err)
	}

	return osResource, nil
}

func (actuator lbpoolActuator) DeleteResource(ctx context.Context, _ orcObjectPT, resource *osResourceT) progress.ReconcileStatus {
	switch resource.ProvisioningStatus {
	case PoolProvisioningStatusPendingDelete:
		return progress.WaitingOnOpenStack(progress.WaitingOnDeletion, lbpoolDeletingPollingPeriod)
	case PoolProvisioningStatusPendingCreate, PoolProvisioningStatusPendingUpdate:
		// We can't delete a pool that's in a pending state, so we need to wait for it to become ACTIVE
		return progress.WaitingOnOpenStack(progress.WaitingOnReady, lbpoolDeletingPollingPeriod)
	}

	err := actuator.osClient.DeleteLBPool(ctx, resource.ID)
	// 409 Conflict means the pool or its load balancer is in a pending state.
	// Let the controller poll for deletion completion.
	if orcerrors.IsConflict(err) {
		return progress.WaitingOnOpenStack(progress.WaitingOnDeletion, lbpoolDeletingPollingPeriod)
	}

	return progress.WrapError(err)
}

func (actuator lbpoolActuator) updateResource(ctx context.Context, obj orcObjectPT, osResource *osResourceT) progress.ReconcileStatus {
	log := ctrl.LoggerFrom(ctx)

	// Octavia rejects updates to a pool in a pending state
	switch osResource.ProvisioningStatus {
	case PoolProvisioningStatusPendingCreate,
		PoolProvisioningStatusPendingUpdate,
		PoolProvisioningStatusPendingDelete:
		return progress.WaitingOnOpenStack(progress.WaitingOnReady, lbpoolAvailablePollingPeriod)
	}

	resource := obj.Spec.Resource

	updateOpts := pools.UpdateOpts{}

	handleNameUpdate(&updateOpts, obj, osResource)
	handleDescriptionUpdate(&updateOpts, resource, osResource)
	handleLBAlgorithmUpdate(&updateOpts, resource, osResource)
	handleAdminStateUpUpdate(&updateOpts, resource, osResource)
	handleSessionPersistenceUpdate(&updateOpts, resource, osResource)
	handleTLSUpdate(&updateOpts, resource, osResource)
	handleTagsUpdate(&updateOpts, resource, osResource)

	needsUpdate, err := needsUpdate(updateOpts)
	if err != nil {
		return progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration updating resource: "+err.Error(), err))
	}
	if !needsUpdate {
		log.V(logging.Debug).Info("No changes")
		return nil
	}

	_, err = actuator.osClient.UpdateLBPool(ctx, osResource.ID, updateOpts)
	if err != nil {
		// 409 Conflict means the pool or its load balancer is in a pending state.
		if orcerrors.IsConflict(err) {
			return progress.WaitingOnOpenStack(progress.WaitingOnReady, lbpoolAvailablePollingPeriod)
		}
		if !orcerrors.IsRetryable(err) {
			err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration updating resource: "+err.Error(), err)
		}
		return progress.WrapError(err)
	}

	return progress.NeedsRefresh()
}

func needsUpdate(updateOpts pools.UpdateOpts) (bool, error) {
	updateOptsMap, err := updateOpts.ToPoolUpdateMap()
	if err != nil {
		return false, err
	}

	updateMap, ok := updateOptsMap["pool"].(map[string]any)
	if !ok {
		updateMap = make(map[string]any)
	}

	return len(updateMap) > 0, nil
}

func handleNameUpdate(updateOpts *pools.UpdateOpts, obj orcObjectPT, osResource *osResourceT) {
	name := getResourceName(obj)
	if osResource.Name != name {
		updateOpts.Name = &name
	}
}

func handleDescriptionUpdate(updateOpts *pools.UpdateOpts, resource *resourceSpecT, osResource *osResourceT) {
	description := ptr.Deref(resource.Description, "")
	if osResource.Description != description {
		updateOpts.Description = &description
	}
}

func handleLBAlgorithmUpdate(updateOpts *pools.UpdateOpts, resource *resourceSpecT, osResource *osResourceT) {
	if string(resource.LBAlgorithm) != osResource.LBMethod {
		updateOpts.LBMethod = pools.LBMethod(resource.LBAlgorithm)
	}
}

func handleAdminStateUpUpdate(updateOpts *pools.UpdateOpts, resource *resourceSpecT, osResource *osResourceT) {
	// Default to true if not specified (OpenStack default)
	adminStateUp := ptr.Deref(resource.AdminStateUp, true)
	if osResource.AdminStateUp != adminStateUp {
		updateOpts.AdminStateUp = &adminStateUp
	}
}

func handleSessionPersistenceUpdate(updateOpts *pools.UpdateOpts, resource *resourceSpecT, osResource *osResourceT) {
	// Check if we need to clear session persistence
	if resource.SessionPersistence == nil {
		if osResource.Persistence.Type != "" {
			// Clear session persistence by setting an empty struct
			updateOpts.Persistence = &pools.SessionPersistence{}
		}
		return
	}

	// Check if session persistence needs to be updated
	specPersistence := resource.SessionPersistence
	osPersistence := osResource.Persistence

	if string(specPersistence.Type) != osPersistence.Type ||
		ptr.Deref(specPersistence.CookieName, "") != osPersistence.CookieName {
		updateOpts.Persistence = &pools.SessionPersistence{
			Type:       string(specPersistence.Type),
			CookieName: ptr.Deref(specPersistence.CookieName, ""),
		}
	}
}

// Octavia fills in deployment defaults for the TLS settings, so we only update the ones that are set.
func handleTLSUpdate(updateOpts *pools.UpdateOpts, resource *resourceSpecT, osResource *osResourceT) {
	if resource.TLSEnabled != nil && *resource.TLSEnabled != osResource.TLSEnabled {
		updateOpts.TLSEnabled = ptr.To(*resource.TLSEnabled)
	}

	updateIfSet := func(desired *string, current string, opt **string) {
		if desired != nil && *desired != current {
			*opt = ptr.To(*desired)
		}
	}
	updateIfSet(resource.TLSContainerURL, osResource.TLSContainerRef, &updateOpts.TLSContainerRef)
	updateIfSet(resource.CATLSContainerURL, osResource.CATLSContainerRef, &updateOpts.CATLSContainerRef)
	updateIfSet(resource.CRLContainerURL, osResource.CRLContainerRef, &updateOpts.CRLContainerRef)
	updateIfSet(resource.TLSCiphers, osResource.TLSCiphers, &updateOpts.TLSCiphers)

	if resource.TLSVersions != nil {
		desired := tlsVersions(resource.TLSVersions)
		current := make([]pools.TLSVersion, len(osResource.TLSVersions))
		for i := range osResource.TLSVersions {
			current[i] = pools.TLSVersion(osResource.TLSVersions[i])
		}
		if !orcslices.EqualAsSets(desired, current) {
			updateOpts.TLSVersions = &desired
		}
	}

	if resource.ALPNProtocols != nil && !orcslices.EqualAsSets(resource.ALPNProtocols, osResource.ALPNProtocols) {
		alpnProtocols := append([]string{}, resource.ALPNProtocols...)
		updateOpts.ALPNProtocols = &alpnProtocols
	}
}

func handleTagsUpdate(updateOpts *pools.UpdateOpts, resource *resourceSpecT, osResource *osResourceT) {
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

func tlsVersions(versions []orcv1alpha1.LBPoolTLSVersion) []pools.TLSVersion {
	if len(versions) == 0 {
		return nil
	}
	out := make([]pools.TLSVersion, len(versions))
	for i := range versions {
		out[i] = pools.TLSVersion(versions[i])
	}
	return out
}

// createMember creates a new member in the pool.
func (actuator lbpoolActuator) createMember(ctx context.Context, obj orcObjectPT, poolID string, memberSpec orcv1alpha1.LBPoolMemberSpec) progress.ReconcileStatus {
	log := ctrl.LoggerFrom(ctx)

	subnet, rs := dependency.FetchDependency(
		ctx, actuator.k8sClient, obj.Namespace,
		memberSpec.SubnetRef, "Subnet",
		func(dep *orcv1alpha1.Subnet) bool { return orcv1alpha1.IsAvailable(dep) && dep.Status.ID != nil },
	)
	if needsReschedule, _ := rs.NeedsReschedule(); needsReschedule {
		return rs
	}

	address := string(memberSpec.Address)
	log.V(logging.Debug).Info("Creating member", "address", address, "port", memberSpec.ProtocolPort)

	createOpts := pools.CreateMemberOpts{
		Address:        address,
		ProtocolPort:   int(memberSpec.ProtocolPort),
		Name:           ptr.Deref(memberSpec.Name, ""),
		Weight:         ptr.To(int(ptr.Deref(memberSpec.Weight, 1))),
		Backup:         memberSpec.Backup,
		AdminStateUp:   memberSpec.AdminStateUp,
		SubnetID:       ptr.Deref(subnet.Status.ID, ""),
		MonitorAddress: string(ptr.Deref(memberSpec.MonitorAddress, "")),
		MonitorPort:    int32PtrToInt(memberSpec.MonitorPort),
		Tags:           tagsToStrings(memberSpec.Tags),
	}

	_, err := actuator.osClient.CreateMember(ctx, poolID, createOpts)
	if err != nil {
		if orcerrors.IsConflict(err) {
			return progress.WaitingOnOpenStack(progress.WaitingOnReady, lbpoolAvailablePollingPeriod)
		}
		return progress.WrapError(err)
	}

	return nil
}

// updateMember updates an existing member if its spec differs from the current state.
// Returns true if the member was updated.
func (actuator lbpoolActuator) updateMember(ctx context.Context, poolID string, current *pools.Member, memberSpec orcv1alpha1.LBPoolMemberSpec) (bool, progress.ReconcileStatus) {
	log := ctrl.LoggerFrom(ctx)

	updateOpts := pools.UpdateMemberOpts{}
	needsUpdate := false

	desiredName := ptr.Deref(memberSpec.Name, "")
	if current.Name != desiredName {
		updateOpts.Name = &desiredName
		needsUpdate = true
	}

	desiredWeight := int(ptr.Deref(memberSpec.Weight, 1))
	if current.Weight != desiredWeight {
		updateOpts.Weight = &desiredWeight
		needsUpdate = true
	}

	desiredBackup := ptr.Deref(memberSpec.Backup, false)
	if current.Backup != desiredBackup {
		updateOpts.Backup = &desiredBackup
		needsUpdate = true
	}

	desiredAdminStateUp := ptr.Deref(memberSpec.AdminStateUp, true)
	if current.AdminStateUp != desiredAdminStateUp {
		updateOpts.AdminStateUp = &desiredAdminStateUp
		needsUpdate = true
	}

	// gophercloud can't reset these to null or clear the tags, so we only update them when set.
	if memberSpec.MonitorAddress != nil && string(*memberSpec.MonitorAddress) != current.MonitorAddress {
		updateOpts.MonitorAddress = ptr.To(string(*memberSpec.MonitorAddress))
		needsUpdate = true
	}
	if memberSpec.MonitorPort != nil && int(*memberSpec.MonitorPort) != current.MonitorPort {
		updateOpts.MonitorPort = ptr.To(int(*memberSpec.MonitorPort))
		needsUpdate = true
	}
	if desiredTags := tagsToStrings(memberSpec.Tags); len(desiredTags) > 0 && !orcslices.EqualAsSets(desiredTags, current.Tags) {
		updateOpts.Tags = desiredTags
		needsUpdate = true
	}

	if !needsUpdate {
		return false, nil
	}

	log.V(logging.Debug).Info("Updating member", "memberID", current.ID)
	_, err := actuator.osClient.UpdateMember(ctx, poolID, current.ID, updateOpts)
	if err != nil {
		if orcerrors.IsConflict(err) {
			return false, progress.WaitingOnOpenStack(progress.WaitingOnReady, lbpoolAvailablePollingPeriod)
		}
		return false, progress.WrapError(err)
	}

	return true, nil
}

// deleteMember deletes a member from the pool.
func (actuator lbpoolActuator) deleteMember(ctx context.Context, poolID string, member *pools.Member) progress.ReconcileStatus {
	log := ctrl.LoggerFrom(ctx)

	log.V(logging.Debug).Info("Deleting member", "memberID", member.ID)
	err := actuator.osClient.DeleteMember(ctx, poolID, member.ID)
	if err != nil {
		if orcerrors.IsConflict(err) {
			return progress.WaitingOnOpenStack(progress.WaitingOnReady, lbpoolAvailablePollingPeriod)
		}
		return progress.WrapError(err)
	}

	return nil
}

// reconcileMembers reconciles the pool members.
func (actuator lbpoolActuator) reconcileMembers(ctx context.Context, obj orcObjectPT, osResource *osResourceT) progress.ReconcileStatus {
	log := ctrl.LoggerFrom(ctx)

	// Check pool provisioning status - we can only modify members when pool is ACTIVE
	if osResource.ProvisioningStatus != PoolProvisioningStatusActive {
		log.V(logging.Debug).Info("Pool not in ACTIVE state, waiting before modifying members",
			"status", osResource.ProvisioningStatus)
		return progress.WaitingOnOpenStack(progress.WaitingOnReady, lbpoolAvailablePollingPeriod)
	}

	resource := obj.Spec.Resource

	// List existing members from OpenStack, keyed by address:port
	currentMembers := make(map[string]*pools.Member)
	for member, err := range actuator.osClient.ListMembers(ctx, osResource.ID, pools.ListMembersOpts{}) {
		if err != nil {
			return progress.WrapError(err)
		}
		key := fmt.Sprintf("%s:%d", member.Address, member.ProtocolPort)
		currentMembers[key] = member
	}

	// Build desired members map keyed by address:port
	desiredMemberMap := make(map[string]orcv1alpha1.LBPoolMemberSpec)
	for _, m := range resource.Members {
		key := fmt.Sprintf("%s:%d", m.Address, m.ProtocolPort)
		desiredMemberMap[key] = m
	}

	// Create missing members
	for _, memberSpec := range resource.Members {
		memberKey := fmt.Sprintf("%s:%d", memberSpec.Address, memberSpec.ProtocolPort)
		if _, exists := currentMembers[memberKey]; !exists {
			if rs := actuator.createMember(ctx, obj, osResource.ID, memberSpec); rs != nil {
				return rs
			}
			return progress.NeedsRefresh()
		}
	}

	// Update existing members if changed
	for key, current := range currentMembers {
		memberSpec, exists := desiredMemberMap[key]
		// Skip members marked for deletion
		if !exists {
			continue
		}

		updated, rs := actuator.updateMember(ctx, osResource.ID, current, memberSpec)
		if rs != nil {
			return rs
		}
		if updated {
			return progress.NeedsRefresh()
		}
	}

	// Delete extra members
	for key, current := range currentMembers {
		if _, exists := desiredMemberMap[key]; !exists {
			if rs := actuator.deleteMember(ctx, osResource.ID, current); rs != nil {
				return rs
			}
			return progress.NeedsRefresh()
		}
	}

	return nil
}

func (actuator lbpoolActuator) GetResourceReconcilers(ctx context.Context, orcObject orcObjectPT, osResource *osResourceT, controller interfaces.ResourceController) ([]resourceReconciler, progress.ReconcileStatus) {
	return []resourceReconciler{
		actuator.reconcileDependencyGuards,
		actuator.updateResource,
		actuator.reconcileMembers,
	}, nil
}

// reconcileDependencyGuards adds the deletion guards to the pool's dependencies, including the
// members' subnets. CreateResource adds the others before creating; this also covers an adopted
// pool.
func (actuator lbpoolActuator) reconcileDependencyGuards(ctx context.Context, obj orcObjectPT, _ *osResourceT) progress.ReconcileStatus {
	var reconcileStatus progress.ReconcileStatus
	_, rs := loadBalancerDependency.RequireDependencies(ctx, actuator.k8sClient, obj, orcv1alpha1.IsAvailable)
	reconcileStatus = reconcileStatus.WithReconcileStatus(rs)
	_, rs = listenerDependency.RequireDependencies(ctx, actuator.k8sClient, obj, orcv1alpha1.IsAvailable)
	reconcileStatus = reconcileStatus.WithReconcileStatus(rs)
	_, rs = projectDependency.RequireDependencies(ctx, actuator.k8sClient, obj, orcv1alpha1.IsAvailable)
	reconcileStatus = reconcileStatus.WithReconcileStatus(rs)
	_, rs = subnetMemberDependency.RequireDependencies(ctx, actuator.k8sClient, obj, orcv1alpha1.IsAvailable)
	return reconcileStatus.WithReconcileStatus(rs)
}

type lbpoolHelperFactory struct{}

var _ helperFactory = lbpoolHelperFactory{}

func newActuator(ctx context.Context, orcObject *orcv1alpha1.LBPool, controller interfaces.ResourceController) (lbpoolActuator, progress.ReconcileStatus) {
	log := ctrl.LoggerFrom(ctx)

	// Ensure credential secrets exist and have our finalizer
	_, reconcileStatus := credentialsDependency.RequireDependencies(ctx, controller.GetK8sClient(), orcObject, func(*corev1.Secret) bool { return true })
	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return lbpoolActuator{}, reconcileStatus
	}

	clientScope, err := controller.GetScopeFactory().NewClientScopeFromObject(ctx, controller.GetK8sClient(), log, orcObject)
	if err != nil {
		return lbpoolActuator{}, progress.WrapError(err)
	}
	osClient, err := clientScope.NewLBPoolClient()
	if err != nil {
		return lbpoolActuator{}, progress.WrapError(err)
	}

	return lbpoolActuator{
		osClient:  osClient,
		k8sClient: controller.GetK8sClient(),
	}, nil
}

func (lbpoolHelperFactory) NewAPIObjectAdapter(obj orcObjectPT) adapterI {
	return lbpoolAdapter{obj}
}

func (lbpoolHelperFactory) NewCreateActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (createResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, orcObject, controller)
}

func (lbpoolHelperFactory) NewDeleteActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (deleteResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, orcObject, controller)
}
