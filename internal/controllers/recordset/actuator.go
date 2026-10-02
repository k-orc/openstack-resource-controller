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
	"context"
	"iter"

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
)

// OpenStack resource types
type (
	osResourceT = recordsets.RecordSet

	createResourceActuator = interfaces.CreateResourceActuator[orcObjectPT, orcObjectT, filterT, osResourceT]
	deleteResourceActuator = interfaces.DeleteResourceActuator[orcObjectPT, orcObjectT, osResourceT]
	resourceReconciler     = interfaces.ResourceReconciler[orcObjectPT, osResourceT]
	helperFactory          = interfaces.ResourceHelperFactory[orcObjectPT, orcObjectT, resourceSpecT, filterT, osResourceT]
)

type recordsetActuator struct {
	// RecordSet is a Designate resource - reuses the shared osclients.DNSClient, same as
	// DNSZone/DNSZoneShare, rather than a dedicated per-resource client.
	osClient  osclients.DNSClient
	k8sClient client.Client

	// zoneID is the owning DNSZone's OpenStack ID, resolved once when this actuator is
	// constructed (see newActuator) - every DNSClient recordset method is scoped under a zone in
	// Designate's own API shape, including GetOSResourceByID's single-ID signature, which the
	// generic interfaces.CreateResourceActuator contract fixes to one opaque ID with no room for
	// the zone alongside it. Same reasoning as DNSZoneShare's identical zoneID field.
	zoneID string
}

var _ createResourceActuator = recordsetActuator{}
var _ deleteResourceActuator = recordsetActuator{}

func (recordsetActuator) GetResourceID(osResource *osResourceT) string {
	return osResource.ID
}

func (actuator recordsetActuator) GetOSResourceByID(ctx context.Context, id string) (*osResourceT, progress.ReconcileStatus) {
	resource, err := actuator.osClient.GetRecordSet(ctx, actuator.zoneID, id)
	if err != nil {
		return nil, progress.WrapError(err)
	}
	return resource, nil
}

func (actuator recordsetActuator) ListOSResourcesForAdoption(ctx context.Context, orcObject orcObjectPT) (iter.Seq2[*osResourceT, error], bool) {
	resourceSpec := orcObject.Spec.Resource
	if resourceSpec == nil || actuator.zoneID == "" {
		return nil, false
	}

	// Matches the full declared spec, same reasoning as every other ListOSResourcesForAdoption
	// in this codebase: adoption must not match a recordset that only partially agrees with what
	// we'd otherwise create.
	listOpts := recordsets.ListOpts{
		Name: getResourceName(orcObject),
		Type: string(resourceSpec.Type),
	}

	return actuator.osClient.ListRecordSets(ctx, actuator.zoneID, listOpts), true
}

func (actuator recordsetActuator) ListOSResourcesForImport(ctx context.Context, obj orcObjectPT, filter filterT) (iter.Seq2[*osResourceT, error], progress.ReconcileStatus) {
	if actuator.zoneID == "" {
		return nil, progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "cannot import a RecordSet without a resolved zoneRef"))
	}

	listOpts := recordsets.ListOpts{
		Name: string(ptr.Deref(filter.Name, "")),
	}
	if filter.Type != nil {
		listOpts.Type = string(*filter.Type)
	}

	return actuator.osClient.ListRecordSets(ctx, actuator.zoneID, listOpts), nil
}

func (actuator recordsetActuator) CreateResource(ctx context.Context, obj orcObjectPT) (*osResourceT, progress.ReconcileStatus) {
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

	createOpts := recordsets.CreateOpts{
		Name:        getResourceName(obj),
		Description: ptr.Deref(resource.Description, ""),
		Type:        string(resource.Type),
		Records:     resource.Records,
	}
	if resource.TTL != nil {
		createOpts.TTL = int(*resource.TTL)
	}

	osResource, err := actuator.osClient.CreateRecordSet(ctx, actuator.zoneID, createOpts)
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

// updateResource handles description/ttl/records changes - name and type are immutable (see
// recordset_types.go), matching Designate's own UpdateOpts, which has no fields for either.
func (actuator recordsetActuator) updateResource(ctx context.Context, obj orcObjectPT, osResource *osResourceT) progress.ReconcileStatus {
	log := ctrl.LoggerFrom(ctx)
	resource := obj.Spec.Resource
	if resource == nil {
		// Should have been caught by API validation
		return progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Update requested, but spec.resource is not set"))
	}

	updateOpts := recordsets.UpdateOpts{}
	needsUpdate := false

	description := ptr.Deref(resource.Description, "")
	if osResource.Description != description {
		updateOpts.Description = &description
		needsUpdate = true
	}

	if resource.TTL != nil && osResource.TTL != int(*resource.TTL) {
		ttl := int(*resource.TTL)
		updateOpts.TTL = &ttl
		needsUpdate = true
	}

	if !stringSlicesEqualAsSets(osResource.Records, resource.Records) {
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

	// Resolve the owning zone once, here, rather than separately in every method below - see the
	// zoneID field's own doc comment on the actuator struct for why. The zone reference lives in
	// a different place depending on management policy: spec.resource.zoneRef when managed,
	// spec.import.filter.zoneRef when unmanaged (an unmanaged object never has spec.resource -
	// using zoneDependency, which only looks at spec.resource, unconditionally here was the actual
	// bug behind every import/import-error/dependency KUTTL scenario timing out in CI).
	var zone *orcv1alpha1.DNSZone
	var zoneRS progress.ReconcileStatus
	switch {
	case orcObject.Spec.Resource != nil:
		zone, zoneRS = zoneDependency.RequireDependency(ctx, controller.GetK8sClient(), orcObject, orcv1alpha1.IsAvailable)
	case orcObject.Spec.Import != nil && orcObject.Spec.Import.Filter != nil:
		zone, zoneRS = dependency.FetchDependency[*orcv1alpha1.DNSZone](ctx, controller.GetK8sClient(), orcObject.Namespace,
			&orcObject.Spec.Import.Filter.ZoneRef, "DNSZone", orcv1alpha1.IsAvailable)
	default:
		// import-by-bare-id has no zone reference anywhere in the spec, and every Designate
		// recordset operation (including Get) is zone-scoped - there's no way to resolve which
		// zone a bare recordset ID belongs to. Not supported; flagged in the PR description for
		// maintainer awareness rather than left to hang silently.
		zoneRS = progress.WrapError(orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration,
			"importing a RecordSet by bare id is not supported - use import.filter with zoneRef instead"))
	}
	if needsReschedule, _ := zoneRS.NeedsReschedule(); needsReschedule {
		return recordsetActuator{}, zoneRS
	}

	clientScope, err := controller.GetScopeFactory().NewClientScopeFromObject(ctx, controller.GetK8sClient(), log, orcObject)
	if err != nil {
		return recordsetActuator{}, progress.WrapError(err)
	}
	osClient, err := clientScope.NewDNSClient()
	if err != nil {
		return recordsetActuator{}, progress.WrapError(err)
	}

	return recordsetActuator{
		osClient:  osClient,
		k8sClient: controller.GetK8sClient(),
		zoneID:    ptr.Deref(zone.Status.ID, ""),
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
