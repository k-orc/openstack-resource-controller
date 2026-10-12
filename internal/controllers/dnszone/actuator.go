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

package dnszone

import (
	"context"
	"iter"

	"github.com/gophercloud/gophercloud/v2/openstack/dns/v2/zones"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v3/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v3/internal/controllers/generic/progress"
	"github.com/k-orc/openstack-resource-controller/v3/internal/logging"
	"github.com/k-orc/openstack-resource-controller/v3/internal/osclients"
	orcerrors "github.com/k-orc/openstack-resource-controller/v3/internal/util/errors"
)

// OpenStack resource types
type (
	osResourceT = zones.Zone

	createResourceActuator = interfaces.CreateResourceActuator[orcObjectPT, orcObjectT, filterT, osResourceT]
	deleteResourceActuator = interfaces.DeleteResourceActuator[orcObjectPT, orcObjectT, osResourceT]
	resourceReconciler     = interfaces.ResourceReconciler[orcObjectPT, osResourceT]
	helperFactory          = interfaces.ResourceHelperFactory[orcObjectPT, orcObjectT, resourceSpecT, filterT, osResourceT]
)

type dnszoneActuator struct {
	osClient  osclients.DNSZoneClient
	k8sClient client.Client
}

var _ createResourceActuator = dnszoneActuator{}
var _ deleteResourceActuator = dnszoneActuator{}

func (dnszoneActuator) GetResourceID(osResource *osResourceT) string {
	return osResource.ID
}

func (actuator dnszoneActuator) GetOSResourceByID(ctx context.Context, id string) (*osResourceT, progress.ReconcileStatus) {
	resource, err := actuator.osClient.GetDNSZone(ctx, id)
	if err != nil {
		return nil, progress.WrapError(err)
	}
	return resource, nil
}

func (actuator dnszoneActuator) ListOSResourcesForAdoption(ctx context.Context, orcObject orcObjectPT) (iter.Seq2[*osResourceT, error], bool) {
	resourceSpec := orcObject.Spec.Resource
	if resourceSpec == nil {
		return nil, false
	}

	// Matches the full declared spec, same reasoning as every other ListOSResourcesForAdoption
	// in this codebase: adoption must not match a zone that only partially agrees with what we'd
	// otherwise create.
	listOpts := zones.ListOpts{
		Name:        getResourceName(orcObject),
		Description: ptr.Deref(resourceSpec.Description, ""),
		Email:       ptr.Deref(resourceSpec.Email, ""),
		Type:        string(resourceSpec.Type),
	}

	return actuator.osClient.ListDNSZones(ctx, listOpts), true
}

func (actuator dnszoneActuator) ListOSResourcesForImport(ctx context.Context, obj orcObjectPT, filter filterT) (iter.Seq2[*osResourceT, error], progress.ReconcileStatus) {
	listOpts := zones.ListOpts{
		Name:        string(ptr.Deref(filter.Name, "")),
		Description: ptr.Deref(filter.Description, ""),
		Email:       ptr.Deref(filter.Email, ""),
	}
	if filter.Type != nil {
		listOpts.Type = string(*filter.Type)
	}

	return actuator.osClient.ListDNSZones(ctx, listOpts), nil
}

func (actuator dnszoneActuator) CreateResource(ctx context.Context, obj orcObjectPT) (*osResourceT, progress.ReconcileStatus) {
	resource := obj.Spec.Resource

	if resource == nil {
		// Should have been caught by API validation
		return nil, progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Creation requested, but spec.resource is not set"))
	}

	createOpts := zones.CreateOpts{
		Name:        getResourceName(obj),
		Description: ptr.Deref(resource.Description, ""),
		Email:       ptr.Deref(resource.Email, ""),
		Type:        string(resource.Type),
	}
	if resource.TTL != nil {
		createOpts.TTL = int(*resource.TTL)
	}
	for _, master := range resource.Masters {
		createOpts.Masters = append(createOpts.Masters, string(master))
	}

	osResource, err := actuator.osClient.CreateDNSZone(ctx, createOpts)
	if err != nil {
		if !orcerrors.IsRetryable(err) {
			err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration creating resource: "+err.Error(), err)
		}
		return nil, progress.WrapError(err)
	}

	return osResource, nil
}

func (actuator dnszoneActuator) DeleteResource(ctx context.Context, _ orcObjectPT, resource *osResourceT) progress.ReconcileStatus {
	return progress.WrapError(actuator.osClient.DeleteDNSZone(ctx, resource.ID))
}

// updateResource handles description/ttl/masters changes - name and type are immutable (see
// dnszone_types.go), matching Designate's own UpdateOpts, which has no fields for either.
func (actuator dnszoneActuator) updateResource(ctx context.Context, obj orcObjectPT, osResource *osResourceT) progress.ReconcileStatus {
	log := ctrl.LoggerFrom(ctx)
	resource := obj.Spec.Resource
	if resource == nil {
		// Should have been caught by API validation
		return progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Update requested, but spec.resource is not set"))
	}

	updateOpts := zones.UpdateOpts{}
	needsUpdate := false

	description := ptr.Deref(resource.Description, "")
	if osResource.Description != description {
		updateOpts.Description = &description
		needsUpdate = true
	}

	if resource.Email != nil && osResource.Email != *resource.Email {
		// Email isn't in UpdateOpts for SECONDARY zones, but the CEL validation on
		// DNSZoneResourceSpec already ensures email is only ever set for PRIMARY zones, where
		// Designate does accept updating it.
		updateOpts.Email = *resource.Email
		needsUpdate = true
	}

	if resource.TTL != nil && osResource.TTL != int(*resource.TTL) {
		updateOpts.TTL = int(*resource.TTL)
		needsUpdate = true
	}

	desiredMasters := make([]string, 0, len(resource.Masters))
	for _, master := range resource.Masters {
		desiredMasters = append(desiredMasters, string(master))
	}
	if !stringSlicesEqualAsSets(osResource.Masters, desiredMasters) {
		updateOpts.Masters = desiredMasters
		needsUpdate = true
	}

	if !needsUpdate {
		log.V(logging.Debug).Info("No changes")
		return nil
	}

	_, err := actuator.osClient.UpdateDNSZone(ctx, osResource.ID, updateOpts)
	if err != nil {
		if !orcerrors.IsRetryable(err) {
			err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration updating resource: "+err.Error(), err)
		}
		return progress.WrapError(err)
	}

	return progress.NeedsRefresh()
}

func stringSlicesEqualAsSets(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	seen := make(map[string]int, len(a))
	for _, v := range a {
		seen[v]++
	}
	for _, v := range b {
		seen[v]--
	}
	for _, count := range seen {
		if count != 0 {
			return false
		}
	}
	return true
}

func (actuator dnszoneActuator) GetResourceReconcilers(ctx context.Context, orcObject orcObjectPT, osResource *osResourceT, controller interfaces.ResourceController) ([]resourceReconciler, progress.ReconcileStatus) {
	return []resourceReconciler{
		actuator.updateResource,
	}, nil
}

type dnszoneHelperFactory struct{}

var _ helperFactory = dnszoneHelperFactory{}

func newActuator(ctx context.Context, orcObject *orcv1alpha1.DNSZone, controller interfaces.ResourceController) (dnszoneActuator, progress.ReconcileStatus) {
	log := ctrl.LoggerFrom(ctx)

	// Ensure credential secrets exist and have our finalizer
	_, reconcileStatus := credentialsDependency.RequireDependencies(ctx, controller.GetK8sClient(), orcObject, func(*corev1.Secret) bool { return true })
	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return dnszoneActuator{}, reconcileStatus
	}

	clientScope, err := controller.GetScopeFactory().NewClientScopeFromObject(ctx, controller.GetK8sClient(), log, orcObject)
	if err != nil {
		return dnszoneActuator{}, progress.WrapError(err)
	}
	osClient, err := clientScope.NewDNSZoneClient()
	if err != nil {
		return dnszoneActuator{}, progress.WrapError(err)
	}

	return dnszoneActuator{
		osClient:  osClient,
		k8sClient: controller.GetK8sClient(),
	}, nil
}

func (dnszoneHelperFactory) NewAPIObjectAdapter(obj orcObjectPT) adapterI {
	return dnszoneAdapter{obj}
}

func (dnszoneHelperFactory) NewCreateActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (createResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, orcObject, controller)
}

func (dnszoneHelperFactory) NewDeleteActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (deleteResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, orcObject, controller)
}
