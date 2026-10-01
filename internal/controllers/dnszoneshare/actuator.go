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

package dnszoneshare

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
	"github.com/k-orc/openstack-resource-controller/v3/internal/osclients"
	orcerrors "github.com/k-orc/openstack-resource-controller/v3/internal/util/errors"
)

// OpenStack resource types
type (
	osResourceT = zones.ZoneShare

	createResourceActuator = interfaces.CreateResourceActuator[orcObjectPT, orcObjectT, filterT, osResourceT]
	deleteResourceActuator = interfaces.DeleteResourceActuator[orcObjectPT, orcObjectT, osResourceT]
	resourceReconciler     = interfaces.ResourceReconciler[orcObjectPT, osResourceT]
	helperFactory          = interfaces.ResourceHelperFactory[orcObjectPT, orcObjectT, resourceSpecT, filterT, osResourceT]
)

type dnszoneshareActuator struct {
	// DNSZoneShare is a Designate resource - reuses the shared osclients.DNSClient, same as
	// DNSZone/RecordSet, rather than a dedicated per-resource client.
	osClient  osclients.DNSClient
	k8sClient client.Client

	// zoneID is the owning DNSZone's OpenStack ID, resolved once when this actuator is
	// constructed (see newActuator). Every DNSClient zone-share method is scoped under a zone
	// (Designate's own API shape: shares are a sub-resource of a zone, not top-level), including
	// GetOSResourceByID's single-ID lookup - which the generic interfaces.CreateResourceActuator
	// contract requires to take just the share's own ID, with no way to pass the zone alongside
	// it. Resolving the zone once here, against the specific ORC object this actuator instance
	// was constructed for, and reusing it across every method call on this instance is what makes
	// that single-ID signature work for a resource that Designate itself always scopes by zone.
	zoneID string
}

var _ createResourceActuator = dnszoneshareActuator{}
var _ deleteResourceActuator = dnszoneshareActuator{}

func (dnszoneshareActuator) GetResourceID(osResource *osResourceT) string {
	return osResource.ID
}

func (actuator dnszoneshareActuator) GetOSResourceByID(ctx context.Context, id string) (*osResourceT, progress.ReconcileStatus) {
	resource, err := actuator.osClient.GetZoneShare(ctx, actuator.zoneID, id)
	if err != nil {
		return nil, progress.WrapError(err)
	}
	return resource, nil
}

func (actuator dnszoneshareActuator) ListOSResourcesForAdoption(ctx context.Context, orcObject orcObjectPT) (iter.Seq2[*osResourceT, error], bool) {
	if orcObject.Spec.Resource == nil || actuator.zoneID == "" {
		return nil, false
	}

	return actuator.osClient.ListZoneShares(ctx, actuator.zoneID), true
}

// ListOSResourcesForImport can't filter server-side by targetProjectID (Designate's zone-share
// list endpoint takes no query parameters at all - see gophercloud's ListSharesOpts, which only
// carries the AllProjects header), so this lists everything under the zone and lets the generic
// reconciler's own filter-matching narrow it down client-side.
func (actuator dnszoneshareActuator) ListOSResourcesForImport(ctx context.Context, obj orcObjectPT, filter filterT) (iter.Seq2[*osResourceT, error], progress.ReconcileStatus) {
	if actuator.zoneID == "" {
		return nil, progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "cannot import a DNSZoneShare without a resolved zoneRef"))
	}

	return actuator.osClient.ListZoneShares(ctx, actuator.zoneID), nil
}

func (actuator dnszoneshareActuator) CreateResource(ctx context.Context, obj orcObjectPT) (*osResourceT, progress.ReconcileStatus) {
	resource := obj.Spec.Resource
	if resource == nil {
		// Should have been caught by API validation
		return nil, progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Creation requested, but spec.resource is not set"))
	}
	if actuator.zoneID == "" {
		// Should have been caught in newActuator, which requires the zone dependency before
		// returning an actuator at all - defensive only.
		return nil, progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "zoneRef did not resolve to an OpenStack zone ID"))
	}

	createOpts := zones.ShareZoneOpts{
		TargetProjectID: resource.TargetProjectID,
	}

	osResource, err := actuator.osClient.CreateZoneShare(ctx, actuator.zoneID, createOpts)
	if err != nil {
		if !orcerrors.IsRetryable(err) {
			err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration creating resource: "+err.Error(), err)
		}
		return nil, progress.WrapError(err)
	}

	return osResource, nil
}

func (actuator dnszoneshareActuator) DeleteResource(ctx context.Context, obj orcObjectPT, resource *osResourceT) progress.ReconcileStatus {
	return progress.WrapError(actuator.osClient.DeleteZoneShare(ctx, resource.ZoneID, resource.ID))
}

// GetResourceReconcilers: Designate's zone-share API has no update operation at all (confirmed
// via gophercloud - only List/Get/Share/Unshare) - both zoneRef and targetProjectID are
// immutable (see dnszoneshare_types.go), so there's nothing to reconcile after creation.
func (actuator dnszoneshareActuator) GetResourceReconcilers(ctx context.Context, orcObject orcObjectPT, osResource *osResourceT, controller interfaces.ResourceController) ([]resourceReconciler, progress.ReconcileStatus) {
	return []resourceReconciler{}, nil
}

type dnszoneshareHelperFactory struct{}

var _ helperFactory = dnszoneshareHelperFactory{}

func newActuator(ctx context.Context, orcObject *orcv1alpha1.DNSZoneShare, controller interfaces.ResourceController) (dnszoneshareActuator, progress.ReconcileStatus) {
	log := ctrl.LoggerFrom(ctx)

	// Ensure credential secrets exist and have our finalizer
	_, reconcileStatus := credentialsDependency.RequireDependencies(ctx, controller.GetK8sClient(), orcObject, func(*corev1.Secret) bool { return true })
	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return dnszoneshareActuator{}, reconcileStatus
	}

	// Resolve the owning zone once, here, rather than separately in every method below - see the
	// zoneID field's own doc comment on the actuator struct for why.
	zone, zoneRS := zoneDependency.RequireDependency(ctx, controller.GetK8sClient(), orcObject, orcv1alpha1.IsAvailable)
	if needsReschedule, _ := zoneRS.NeedsReschedule(); needsReschedule {
		return dnszoneshareActuator{}, zoneRS
	}

	clientScope, err := controller.GetScopeFactory().NewClientScopeFromObject(ctx, controller.GetK8sClient(), log, orcObject)
	if err != nil {
		return dnszoneshareActuator{}, progress.WrapError(err)
	}
	osClient, err := clientScope.NewDNSClient()
	if err != nil {
		return dnszoneshareActuator{}, progress.WrapError(err)
	}

	return dnszoneshareActuator{
		osClient:  osClient,
		k8sClient: controller.GetK8sClient(),
		zoneID:    ptr.Deref(zone.Status.ID, ""),
	}, nil
}

func (dnszoneshareHelperFactory) NewAPIObjectAdapter(obj orcObjectPT) adapterI {
	return dnszoneshareAdapter{obj}
}

func (dnszoneshareHelperFactory) NewCreateActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (createResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, orcObject, controller)
}

func (dnszoneshareHelperFactory) NewDeleteActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (deleteResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, orcObject, controller)
}
