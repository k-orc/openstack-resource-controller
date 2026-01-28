/*
Copyright 2025 The ORC Authors.

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

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/progress"
	"github.com/k-orc/openstack-resource-controller/v2/internal/logging"
	"github.com/k-orc/openstack-resource-controller/v2/internal/osclients"
	"github.com/k-orc/openstack-resource-controller/v2/internal/util/dependency"
	orcerrors "github.com/k-orc/openstack-resource-controller/v2/internal/util/errors"
)

// OpenStack resource types
type (
	osResourceT = pools.Pool

	createResourceActuator = interfaces.CreateResourceActuator[orcObjectPT, orcObjectT, filterT, osResourceT]
	deleteResourceActuator = interfaces.DeleteResourceActuator[orcObjectPT, orcObjectT, osResourceT]
	resourceReconciler     = interfaces.ResourceReconciler[orcObjectPT, osResourceT]
	helperFactory          = interfaces.ResourceHelperFactory[orcObjectPT, orcObjectT, resourceSpecT, filterT, osResourceT]
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

func (lbpoolActuator) GetResourceID(osResource *osResourceT) string {
	return osResource.ID
}

func (actuator lbpoolActuator) GetOSResourceByID(ctx context.Context, id string) (*osResourceT, progress.ReconcileStatus) {
	resource, err := actuator.osClient.GetLBPool(ctx, id)
	if err != nil {
		return nil, progress.WrapError(err)
	}

	members := make([]pools.Member, 0, len(resource.Members))
	for _, memberId := range resource.Members {
		member, err := actuator.osClient.GetMember(ctx, id, memberId.ID)
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

	listOpts := pools.ListOpts{
		Name:     getResourceName(orcObject),
		Protocol: string(resourceSpec.Protocol),
		LBMethod: string(resourceSpec.LBAlgorithm),
	}

	return actuator.osClient.ListLBPools(ctx, listOpts), true
}

func (actuator lbpoolActuator) ListOSResourcesForImport(ctx context.Context, obj orcObjectPT, filter filterT) (iter.Seq2[*osResourceT, error], progress.ReconcileStatus) {
	var reconcileStatus progress.ReconcileStatus

	loadBalancer, rs := dependency.FetchDependency(
		ctx, actuator.k8sClient, obj.Namespace,
		filter.LoadBalancerRef, "LoadBalancer",
		func(dep *orcv1alpha1.LoadBalancer) bool { return orcv1alpha1.IsAvailable(dep) && dep.Status.ID != nil },
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
	}

	if len(filter.Tags) > 0 {
		tags := make([]string, len(filter.Tags))
		for i := range filter.Tags {
			tags[i] = string(filter.Tags[i])
		}
		listOpts.Tags = tags
	}

	return actuator.osClient.ListLBPools(ctx, listOpts), reconcileStatus
}

func (actuator lbpoolActuator) CreateResource(ctx context.Context, obj orcObjectPT) (*osResourceT, progress.ReconcileStatus) {
	resource := obj.Spec.Resource

	if resource == nil {
		// Should have been caught by API validation
		return nil, progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Creation requested, but spec.resource is not set"))
	}
	var reconcileStatus progress.ReconcileStatus

	var loadBalancerID string
	if resource.LoadBalancerRef != nil {
		loadBalancer, loadBalancerDepRS := loadBalancerDependency.GetDependency(
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
		listener, listenerDepRS := listenerDependency.GetDependency(
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
		project, projectDepRS := projectDependency.GetDependency(
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
		TLSContainerRef:   ptr.Deref(resource.TLSContainerRef, ""),
		CATLSContainerRef: ptr.Deref(resource.CATLSContainerRef, ""),
		CRLContainerRef:   ptr.Deref(resource.CRLContainerRef, ""),
		TLSCiphers:        ptr.Deref(resource.TLSCiphers, ""),
	}

	if resource.SessionPersistence != nil {
		createOpts.Persistence = &pools.SessionPersistence{
			Type:       string(resource.SessionPersistence.Type),
			CookieName: ptr.Deref(resource.SessionPersistence.CookieName, ""),
		}
	}

	if len(resource.TLSVersions) > 0 {
		tlsVersions := make([]pools.TLSVersion, len(resource.TLSVersions))
		for i := range resource.TLSVersions {
			tlsVersions[i] = pools.TLSVersion(resource.TLSVersions[i])
		}
		createOpts.TLSVersions = tlsVersions
	}

	if len(resource.ALPNProtocols) > 0 {
		createOpts.ALPNProtocols = resource.ALPNProtocols
	}

	if len(resource.Tags) > 0 {
		tags := make([]string, len(resource.Tags))
		for i := range resource.Tags {
			tags[i] = string(resource.Tags[i])
		}
		createOpts.Tags = tags
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
		return progress.WaitingOnOpenStack(progress.WaitingOnReady, lbpoolDeletingPollingPeriod)
	case PoolProvisioningStatusPendingCreate, PoolProvisioningStatusPendingUpdate:
		// We can't delete a pool that's in a pending state, so we need to wait for it to become ACTIVE
		return progress.WaitingOnOpenStack(progress.WaitingOnReady, lbpoolDeletingPollingPeriod)
	}

	err := actuator.osClient.DeleteLBPool(ctx, resource.ID)
	// 409 Conflict means the loadbalancer is already in PENDING_DELETE state.
	// Treat this as success and let the controller poll for deletion completion.
	if orcerrors.IsConflict(err) {
		return progress.WaitingOnOpenStack(progress.WaitingOnReady, lbpoolDeletingPollingPeriod)
	}

	return progress.WrapError(err)
}

func (actuator lbpoolActuator) updateResource(ctx context.Context, obj orcObjectPT, osResource *osResourceT) progress.ReconcileStatus {
	log := ctrl.LoggerFrom(ctx)
	resource := obj.Spec.Resource
	if resource == nil {
		// Should have been caught by API validation
		return progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Update requested, but spec.resource is not set"))
	}

	updateOpts := pools.UpdateOpts{}

	handleNameUpdate(&updateOpts, obj, osResource)
	handleDescriptionUpdate(&updateOpts, resource, osResource)
	handleAdminStateUpUpdate(&updateOpts, resource, osResource)
	handleSessionPersistenceUpdate(&updateOpts, resource, osResource)
	handleTLSContainerRefUpdate(&updateOpts, resource, osResource)
	handleTLSCiphersUpdate(&updateOpts, resource, osResource)
	handleTLSVersionsUpdate(&updateOpts, resource, osResource)
	handleALPNProtocolsUpdate(&updateOpts, resource, osResource)
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

	// 409 Conflict typically means the LoadBalancer is in a pending state (immutable).
	// Wait for it to become available and retry.
	if orcerrors.IsConflict(err) {
		return progress.WaitingOnOpenStack(progress.WaitingOnReady, lbpoolAvailablePollingPeriod)
	}

	if err != nil {
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

func handleAdminStateUpUpdate(updateOpts *pools.UpdateOpts, resource *resourceSpecT, osResource *osResourceT) {
	if resource.AdminStateUp != nil && *resource.AdminStateUp != osResource.AdminStateUp {
		updateOpts.AdminStateUp = resource.AdminStateUp
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

func handleTLSContainerRefUpdate(updateOpts *pools.UpdateOpts, resource *resourceSpecT, osResource *osResourceT) {
	tlsContainerRef := ptr.Deref(resource.TLSContainerRef, "")
	if osResource.TLSContainerRef != tlsContainerRef {
		updateOpts.TLSContainerRef = &tlsContainerRef
	}
}

func handleTLSCiphersUpdate(updateOpts *pools.UpdateOpts, resource *resourceSpecT, osResource *osResourceT) {
	tlsCiphers := ptr.Deref(resource.TLSCiphers, "")
	if osResource.TLSCiphers != tlsCiphers {
		updateOpts.TLSCiphers = &tlsCiphers
	}
}

func handleTLSVersionsUpdate(updateOpts *pools.UpdateOpts, resource *resourceSpecT, osResource *osResourceT) {
	specVersions := resource.TLSVersions
	osVersions := osResource.TLSVersions

	if len(specVersions) == 0 && len(osVersions) == 0 {
		return
	}

	// Compare slices
	if !slices.Equal(specVersions, osVersions) {
		tlsVersions := make([]pools.TLSVersion, len(specVersions))
		for i := range specVersions {
			tlsVersions[i] = pools.TLSVersion(specVersions[i])
		}
		updateOpts.TLSVersions = &tlsVersions
	}
}

func handleALPNProtocolsUpdate(updateOpts *pools.UpdateOpts, resource *resourceSpecT, osResource *osResourceT) {
	specProtocols := resource.ALPNProtocols
	osProtocols := osResource.ALPNProtocols

	if len(specProtocols) == 0 && len(osProtocols) == 0 {
		return
	}

	if !slices.Equal(specProtocols, osProtocols) {
		updateOpts.ALPNProtocols = &specProtocols
	}
}

func handleTagsUpdate(updateOpts *pools.UpdateOpts, resource *resourceSpecT, osResource *osResourceT) {
	desiredTags := make([]string, len(resource.Tags))
	for i, tag := range resource.Tags {
		desiredTags[i] = string(tag)
	}

	slices.Sort(desiredTags)
	slices.Sort(osResource.Tags)

	if !slices.Equal(desiredTags, osResource.Tags) {
		updateOpts.Tags = &desiredTags
	}
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
		Address:      address,
		ProtocolPort: int(memberSpec.ProtocolPort),
		Name:         ptr.Deref(memberSpec.Name, ""),
		Weight:       ptr.To(int(ptr.Deref(memberSpec.Weight, 1))),
		Backup:       memberSpec.Backup,
		AdminStateUp: memberSpec.AdminStateUp,
		SubnetID:     ptr.Deref(subnet.Status.ID, ""),
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
	if resource == nil {
		resource = &orcv1alpha1.LBPoolResourceSpec{}
	}

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
		actuator.updateResource,
		actuator.reconcileMembers,
	}, nil
}

type lbpoolHelperFactory struct{}

var _ helperFactory = lbpoolHelperFactory{}

func newActuator(ctx context.Context, orcObject *orcv1alpha1.LBPool, controller interfaces.ResourceController) (lbpoolActuator, progress.ReconcileStatus) {
	log := ctrl.LoggerFrom(ctx)

	// Ensure credential secrets exist and have our finalizer
	_, reconcileStatus := credentialsDependency.GetDependencies(ctx, controller.GetK8sClient(), orcObject, func(*corev1.Secret) bool { return true })
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
