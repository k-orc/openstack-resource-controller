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
	"github.com/k-orc/openstack-resource-controller/v3/internal/util/dependency"
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
	osClient  osclients.DNSZoneShareClient
	k8sClient client.Client

	// For getting a share by ID, which Designate scopes under its zone: the zone from
	// status.resource, else the zoneRef of the spec or the import filter.
	namespace    string
	statusZoneID string
	zoneRef      *orcv1alpha1.KubernetesNameRef
}

var _ createResourceActuator = dnszoneshareActuator{}
var _ deleteResourceActuator = dnszoneshareActuator{}

func (dnszoneshareActuator) GetResourceID(osResource *osResourceT) string {
	return osResource.ID
}

func (actuator dnszoneshareActuator) GetOSResourceByID(ctx context.Context, id string) (*osResourceT, progress.ReconcileStatus) {
	zoneID := actuator.statusZoneID
	if zoneID == "" {
		if actuator.zoneRef == nil {
			return nil, progress.WrapError(orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration,
				"a DNSZoneShare can't be imported by ID, as Designate needs its zone: use import.filter"))
		}
		zone, rs := dependency.FetchDependency[*orcv1alpha1.DNSZone](
			ctx, actuator.k8sClient, actuator.namespace, actuator.zoneRef, "DNSZone", orcv1alpha1.IsAvailable,
		)
		if needsReschedule, _ := rs.NeedsReschedule(); needsReschedule {
			return nil, rs
		}
		zoneID = ptr.Deref(zone.Status.ID, "")
	}

	resource, err := actuator.osClient.GetZoneShare(ctx, zoneID, id)
	if err != nil {
		return nil, progress.WrapError(err)
	}
	return resource, nil
}

func (actuator dnszoneshareActuator) ListOSResourcesForAdoption(ctx context.Context, orcObject orcObjectPT) (iter.Seq2[*osResourceT, error], bool) {
	resource := orcObject.Spec.Resource
	if resource == nil {
		return nil, false
	}

	zone, rs := dependency.FetchDependency(
		ctx, actuator.k8sClient, orcObject.Namespace, &resource.ZoneRef, "DNSZone",
		func(dep *orcv1alpha1.DNSZone) bool {
			return orcv1alpha1.IsAvailable(dep) && dep.Status.ID != nil
		},
	)
	if needsReschedule, _ := rs.NeedsReschedule(); needsReschedule {
		return nil, false
	}

	shares := actuator.osClient.ListZoneShares(ctx, *zone.Status.ID)
	return osclients.Filter(shares, func(share *osResourceT) bool {
		return share.TargetProjectID == resource.TargetProjectID
	}), true
}

// Designate's share list takes no filters, so targetProjectID is matched client-side.
func (actuator dnszoneshareActuator) ListOSResourcesForImport(ctx context.Context, obj orcObjectPT, filter filterT) (iter.Seq2[*osResourceT, error], progress.ReconcileStatus) {
	zone, rs := dependency.FetchDependency[*orcv1alpha1.DNSZone](
		ctx, actuator.k8sClient, obj.Namespace, &filter.ZoneRef, "DNSZone", orcv1alpha1.IsAvailable,
	)
	if needsReschedule, _ := rs.NeedsReschedule(); needsReschedule {
		return nil, rs
	}

	shares := actuator.osClient.ListZoneShares(ctx, ptr.Deref(zone.Status.ID, ""))
	if filter.TargetProjectID == nil {
		return shares, nil
	}
	return osclients.Filter(shares, func(share *osResourceT) bool {
		return share.TargetProjectID == *filter.TargetProjectID
	}), nil
}

func (actuator dnszoneshareActuator) CreateResource(ctx context.Context, obj orcObjectPT) (*osResourceT, progress.ReconcileStatus) {
	resource := obj.Spec.Resource

	zone, reconcileStatus := zoneDependency.RequireDependency(ctx, actuator.k8sClient, obj, orcv1alpha1.IsAvailable)
	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return nil, reconcileStatus
	}

	createOpts := zones.ShareZoneOpts{
		TargetProjectID: resource.TargetProjectID,
	}

	osResource, err := actuator.osClient.CreateZoneShare(ctx, ptr.Deref(zone.Status.ID, ""), createOpts)
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

// Designate has no update for zone shares.
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

	statusZoneID := ""
	if orcObject.Status.Resource != nil {
		statusZoneID = orcObject.Status.Resource.ZoneID
	}
	var zoneRef *orcv1alpha1.KubernetesNameRef
	switch {
	case orcObject.Spec.Resource != nil:
		zoneRef = &orcObject.Spec.Resource.ZoneRef
	case orcObject.Spec.Import != nil && orcObject.Spec.Import.Filter != nil:
		zoneRef = &orcObject.Spec.Import.Filter.ZoneRef
	}

	clientScope, err := controller.GetScopeFactory().NewClientScopeFromObject(ctx, controller.GetK8sClient(), log, orcObject)
	if err != nil {
		return dnszoneshareActuator{}, progress.WrapError(err)
	}
	osClient, err := clientScope.NewDNSZoneShareClient()
	if err != nil {
		return dnszoneshareActuator{}, progress.WrapError(err)
	}

	return dnszoneshareActuator{
		osClient:     osClient,
		k8sClient:    controller.GetK8sClient(),
		namespace:    orcObject.Namespace,
		statusZoneID: statusZoneID,
		zoneRef:      zoneRef,
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
