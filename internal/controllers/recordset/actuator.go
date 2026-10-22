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

package recordset

import (
	"time"

	"context"
	"iter"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/dns/v2/recordsets"
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
	osResourceT = recordsets.RecordSet

	createResourceActuator = interfaces.CreateResourceActuator[orcObjectPT, orcObjectT, filterT, osResourceT]
	deleteResourceActuator = interfaces.DeleteResourceActuator[orcObjectPT, orcObjectT, osResourceT]
	resourceReconciler     = interfaces.ResourceReconciler[orcObjectPT, osResourceT]
	helperFactory          = interfaces.ResourceHelperFactory[orcObjectPT, orcObjectT, resourceSpecT, filterT, osResourceT]
)

// The frequency to poll when waiting for the resource to become available
const recordsetAvailablePollingPeriod = 15 * time.Second

type recordsetActuator struct {
	osClient  osclients.RecordSetClient
	k8sClient client.Client

	// statusZoneID is the zone from status.resource, for getting the recordset by ID.
	statusZoneID string
}

var _ createResourceActuator = recordsetActuator{}
var _ deleteResourceActuator = recordsetActuator{}

func (recordsetActuator) GetResourceID(osResource *osResourceT) string {
	return osResource.ID
}

// Designate's /v2/recordsets/{id} redirect can't be used to get a recordset without its zone: its
// location is built from the API's own host URL.
func (actuator recordsetActuator) GetOSResourceByID(ctx context.Context, id string) (*osResourceT, progress.ReconcileStatus) {
	if actuator.statusZoneID == "" {
		return actuator.findRecordSetByID(ctx, id)
	}
	resource, err := actuator.osClient.GetRecordSet(ctx, actuator.statusZoneID, id)
	if err != nil {
		return nil, progress.WrapError(err)
	}
	return resource, nil
}

// findRecordSetByID searches all zones, as /v2/recordsets has no id filter.
func (actuator recordsetActuator) findRecordSetByID(ctx context.Context, id string) (*osResourceT, progress.ReconcileStatus) {
	for resource, err := range actuator.osClient.ListRecordSets(ctx, "", recordsets.ListOpts{}) {
		if err != nil {
			return nil, progress.WrapError(err)
		}
		if resource.ID == id {
			return resource, nil
		}
	}
	return nil, progress.WrapError(gophercloud.ErrResourceNotFound{Name: id, ResourceType: "recordset"})
}

func (actuator recordsetActuator) ListOSResourcesForAdoption(ctx context.Context, orcObject orcObjectPT) (iter.Seq2[*osResourceT, error], bool) {
	resourceSpec := orcObject.Spec.Resource
	if resourceSpec == nil {
		return nil, false
	}

	zone, rs := dependency.FetchDependency(
		ctx, actuator.k8sClient, orcObject.Namespace, &resourceSpec.ZoneRef, "DNSZone",
		func(dep *orcv1alpha1.DNSZone) bool {
			return orcv1alpha1.IsAvailable(dep) && dep.Status.ID != nil
		},
	)
	if needsReschedule, _ := rs.NeedsReschedule(); needsReschedule {
		return nil, false
	}

	listOpts := recordsets.ListOpts{
		Name:        getResourceName(orcObject),
		Type:        string(resourceSpec.Type),
		Description: ptr.Deref(resourceSpec.Description, ""),
		TTL:         int(ptr.Deref(resourceSpec.TTL, 0)),
	}
	records := func(r *osResourceT) bool {
		return orcslices.EqualAsSets(r.Records, resourceSpec.Records)
	}

	return osclients.Filter(actuator.osClient.ListRecordSets(ctx, *zone.Status.ID, listOpts), records), true
}

func (actuator recordsetActuator) ListOSResourcesForImport(ctx context.Context, obj orcObjectPT, filter filterT) (iter.Seq2[*osResourceT, error], progress.ReconcileStatus) {
	// Without a zoneRef, all zones in the project are searched.
	zone, rs := dependency.FetchDependency[*orcv1alpha1.DNSZone](
		ctx, actuator.k8sClient, obj.Namespace, filter.ZoneRef, "DNSZone",
		orcv1alpha1.IsAvailable,
	)
	if needsReschedule, _ := rs.NeedsReschedule(); needsReschedule {
		return nil, rs
	}

	listOpts := recordsets.ListOpts{
		Name:        string(ptr.Deref(filter.Name, "")),
		Type:        string(ptr.Deref(filter.Type, "")),
		Description: ptr.Deref(filter.Description, ""),
		TTL:         int(ptr.Deref(filter.TTL, 0)),
	}

	return actuator.osClient.ListRecordSets(ctx, ptr.Deref(zone.Status.ID, ""), listOpts), nil
}

func (actuator recordsetActuator) CreateResource(ctx context.Context, obj orcObjectPT) (*osResourceT, progress.ReconcileStatus) {
	resource := obj.Spec.Resource

	createOpts := recordsets.CreateOpts{
		Name:        getResourceName(obj),
		Description: ptr.Deref(resource.Description, ""),
		Type:        string(resource.Type),
		Records:     resource.Records,
	}
	if resource.TTL != nil {
		createOpts.TTL = int(*resource.TTL)
	}

	zone, reconcileStatus := zoneDependency.RequireDependency(ctx, actuator.k8sClient, obj, orcv1alpha1.IsAvailable)
	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return nil, reconcileStatus
	}

	osResource, err := actuator.osClient.CreateRecordSet(ctx, ptr.Deref(zone.Status.ID, ""), createOpts)
	if err != nil {
		if !orcerrors.IsRetryable(err) {
			err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration creating resource: "+err.Error(), err)
		}
		return nil, progress.WrapError(err)
	}

	return osResource, nil
}

func (actuator recordsetActuator) DeleteResource(ctx context.Context, _ orcObjectPT, resource *osResourceT) progress.ReconcileStatus {
	return progress.WrapError(actuator.osClient.DeleteRecordSet(ctx, resource.ZoneID, resource.ID))
}

func (actuator recordsetActuator) updateResource(ctx context.Context, obj orcObjectPT, osResource *osResourceT) progress.ReconcileStatus {
	log := ctrl.LoggerFrom(ctx)
	resource := obj.Spec.Resource

	updateOpts := recordsets.UpdateOpts{}
	needsUpdate := false

	description := ptr.Deref(resource.Description, "")
	if osResource.Description != description {
		updateOpts.Description = &description
		needsUpdate = true
	}

	// Designate reports an unset TTL as 0; setting 0 sends null, back to the zone's TTL.
	ttl := int(ptr.Deref(resource.TTL, 0))
	if osResource.TTL != ttl {
		updateOpts.TTL = &ttl
		needsUpdate = true
	}

	if !orcslices.EqualAsSets(osResource.Records, resource.Records) {
		updateOpts.Records = resource.Records
		needsUpdate = true
	}

	if !needsUpdate {
		log.V(logging.Debug).Info("No changes")
		return nil
	}

	_, err := actuator.osClient.UpdateRecordSet(ctx, osResource.ZoneID, osResource.ID, updateOpts)
	if err != nil {
		if !orcerrors.IsRetryable(err) {
			err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration updating resource: "+err.Error(), err)
		}
		return progress.WrapError(err)
	}

	return progress.NeedsRefresh()
}

func (actuator recordsetActuator) GetResourceReconcilers(ctx context.Context, orcObject orcObjectPT, osResource *osResourceT, controller interfaces.ResourceController) ([]resourceReconciler, progress.ReconcileStatus) {
	return []resourceReconciler{
		actuator.updateResource,
	}, nil
}

type recordsetHelperFactory struct{}

var _ helperFactory = recordsetHelperFactory{}

func newActuator(ctx context.Context, orcObject *orcv1alpha1.RecordSet, controller interfaces.ResourceController) (recordsetActuator, progress.ReconcileStatus) {
	log := ctrl.LoggerFrom(ctx)

	// Ensure credential secrets exist and have our finalizer
	_, reconcileStatus := credentialsDependency.RequireDependencies(ctx, controller.GetK8sClient(), orcObject, func(*corev1.Secret) bool { return true })
	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return recordsetActuator{}, reconcileStatus
	}

	statusZoneID := ""
	if orcObject.Status.Resource != nil {
		statusZoneID = orcObject.Status.Resource.ZoneID
	}

	clientScope, err := controller.GetScopeFactory().NewClientScopeFromObject(ctx, controller.GetK8sClient(), log, orcObject)
	if err != nil {
		return recordsetActuator{}, progress.WrapError(err)
	}
	osClient, err := clientScope.NewRecordSetClient()
	if err != nil {
		return recordsetActuator{}, progress.WrapError(err)
	}

	return recordsetActuator{
		osClient:     osClient,
		k8sClient:    controller.GetK8sClient(),
		statusZoneID: statusZoneID,
	}, nil
}

func (recordsetHelperFactory) NewAPIObjectAdapter(obj orcObjectPT) adapterI {
	return recordsetAdapter{obj}
}

func (recordsetHelperFactory) NewCreateActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (createResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, orcObject, controller)
}

func (recordsetHelperFactory) NewDeleteActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (deleteResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, orcObject, controller)
}
