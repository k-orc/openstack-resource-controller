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

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/progress"
	"github.com/k-orc/openstack-resource-controller/v2/internal/logging"
	"github.com/k-orc/openstack-resource-controller/v2/internal/osclients"
	orcerrors "github.com/k-orc/openstack-resource-controller/v2/internal/util/errors"
)

// OpenStack resource types
type (
	osResourceT = zones.Zone

	createResourceActuator = interfaces.CreateResourceActuator[orcObjectPT, orcObjectT, filterT, osResourceT]
	deleteResourceActuator = interfaces.DeleteResourceActuator[orcObjectPT, orcObjectT, osResourceT]
	resourceReconciler     = interfaces.ResourceReconciler[orcObjectPT, osResourceT]
	helperFactory          = interfaces.ResourceHelperFactory[orcObjectPT, orcObjectT, resourceSpecT, filterT, osResourceT]
)

type dnsZoneActuator struct {
	osClient  osclients.DNSZoneClient
	k8sClient client.Client
}

var _ createResourceActuator = dnsZoneActuator{}
var _ deleteResourceActuator = dnsZoneActuator{}

func (dnsZoneActuator) GetResourceID(osResource *osResourceT) string {
	return osResource.ID
}

func (actuator dnsZoneActuator) GetOSResourceByID(ctx context.Context, id string) (*osResourceT, progress.ReconcileStatus) {
	resource, err := actuator.osClient.GetZone(ctx, id)
	if err != nil {
		return nil, progress.WrapError(err)
	}
	return resource, nil
}

func (actuator dnsZoneActuator) ListOSResourcesForAdoption(ctx context.Context, orcObject orcObjectPT) (iter.Seq2[*osResourceT, error], bool) {
	resourceSpec := orcObject.Spec.Resource
	if resourceSpec == nil {
		return nil, false
	}

	var filters []osclients.ResourceFilter[osResourceT]

	if resourceSpec.Description != nil {
		filters = append(filters, func(f *zones.Zone) bool {
			return f.Description == *resourceSpec.Description
		})
	} else {
		filters = append(filters, func(f *zones.Zone) bool {
			return f.Description == ""
		})
	}
	if resourceSpec.Email != nil {
		filters = append(filters, func(f *zones.Zone) bool {
			return f.Email == *resourceSpec.Email
		})
	} else {
		filters = append(filters, func(f *zones.Zone) bool {
			return f.Email == ""
		})
	}
	if resourceSpec.TTL != nil {
		filters = append(filters, func(f *zones.Zone) bool {
			return f.TTL == int(*resourceSpec.TTL)
		})
	}
	filters = append(filters, func(f *zones.Zone) bool {
		return f.Type == string(resourceSpec.Type)
	})
	if len(resourceSpec.Masters) > 0 {
		filters = append(filters, func(f *zones.Zone) bool {
			if len(f.Masters) != len(resourceSpec.Masters) {
				return false
			}
			for i, m := range f.Masters {
				if m != resourceSpec.Masters[i] {
					return false
				}
			}
			return true
		})
	} else {
		filters = append(filters, func(f *zones.Zone) bool {
			return len(f.Masters) == 0
		})
	}

	listOpts := zones.ListOpts{
		Name: getDNSZoneName(orcObject),
	}

	return actuator.listOSResources(ctx, filters, listOpts), true
}

func (actuator dnsZoneActuator) ListOSResourcesForImport(ctx context.Context, obj orcObjectPT, filter filterT) (iter.Seq2[*osResourceT, error], progress.ReconcileStatus) {
	var filters []osclients.ResourceFilter[osResourceT]

	if filter.Name != nil {
		filters = append(filters, func(f *zones.Zone) bool { return f.Name == string(*filter.Name) })
	}
	if filter.Email != nil {
		filters = append(filters, func(f *zones.Zone) bool { return f.Email == *filter.Email })
	}
	if filter.Description != nil {
		filters = append(filters, func(f *zones.Zone) bool { return f.Description == *filter.Description })
	}
	if filter.TTL != nil {
		filters = append(filters, func(f *zones.Zone) bool { return f.TTL == int(*filter.TTL) })
	}
	if filter.Type != nil {
		filters = append(filters, func(f *zones.Zone) bool { return f.Type == string(*filter.Type) })
	}
	if len(filter.Masters) > 0 {
		filters = append(filters, func(f *zones.Zone) bool {
			if len(f.Masters) != len(filter.Masters) {
				return false
			}
			for i, m := range f.Masters {
				if m != filter.Masters[i] {
					return false
				}
			}
			return true
		})
	}

	listOpts := zones.ListOpts{}
	if filter.Name != nil {
		listOpts.Name = string(*filter.Name)
	}

	return actuator.listOSResources(ctx, filters, listOpts), nil
}

func (actuator dnsZoneActuator) listOSResources(ctx context.Context, filters []osclients.ResourceFilter[osResourceT], listOpts zones.ListOptsBuilder) iter.Seq2[*zones.Zone, error] {
	zones := actuator.osClient.ListZones(ctx, listOpts)
	return osclients.Filter(zones, filters...)
}

func (actuator dnsZoneActuator) CreateResource(ctx context.Context, obj orcObjectPT) (*osResourceT, progress.ReconcileStatus) {
	resource := obj.Spec.Resource

	if resource == nil {
		// Should have been caught by API validation
		return nil, progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Creation requested, but spec.resource is not set"))
	}
	createOpts := zones.CreateOpts{
		Name:        getDNSZoneName(obj),
		Email:       ptr.Deref(resource.Email, ""),
		Description: ptr.Deref(resource.Description, ""),
		Type:        string(resource.Type),
		Masters:     resource.Masters,
	}
	if resource.TTL != nil {
		createOpts.TTL = int(*resource.TTL)
	}

	osResource, err := actuator.osClient.CreateZone(ctx, createOpts)
	if err != nil {
		if !orcerrors.IsRetryable(err) {
			reason := orcv1alpha1.ConditionReasonInvalidConfiguration
			if orcerrors.IsConflict(err) {
				reason = orcv1alpha1.ConditionReasonUnrecoverableError
			}
			err = orcerrors.Terminal(reason, "invalid configuration creating resource: "+err.Error(), err)
		}
		return nil, progress.WrapError(err)
	}

	return osResource, nil
}

func (actuator dnsZoneActuator) DeleteResource(ctx context.Context, _ orcObjectPT, resource *osResourceT) progress.ReconcileStatus {
	return progress.WrapError(actuator.osClient.DeleteZone(ctx, resource.ID))
}

func (actuator dnsZoneActuator) updateResource(ctx context.Context, obj orcObjectPT, osResource *osResourceT) progress.ReconcileStatus {
	log := ctrl.LoggerFrom(ctx)
	resource := obj.Spec.Resource
	if resource == nil {
		// Should have been caught by API validation
		return progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Update requested, but spec.resource is not set"))
	}

	updateOpts := zones.UpdateOpts{}

	handleDescriptionUpdate(&updateOpts, resource, osResource)
	handleEmailUpdate(&updateOpts, resource, osResource)
	handleTTLUpdate(&updateOpts, resource, osResource)
	handleMastersUpdate(&updateOpts, resource, osResource)

	needsUpdate, err := needsUpdate(updateOpts)
	if err != nil {
		return progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration updating resource: "+err.Error(), err))
	}
	if !needsUpdate {
		log.V(logging.Debug).Info("No changes")
		return nil
	}

	_, err = actuator.osClient.UpdateZone(ctx, osResource.ID, updateOpts)

	if err != nil {
		if !orcerrors.IsRetryable(err) {
			err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration updating resource: "+err.Error(), err)
		}
		return progress.WrapError(err)
	}

	return progress.NeedsRefresh()
}

func needsUpdate(updateOpts zones.UpdateOpts) (bool, error) {
	updateOptsMap, err := updateOpts.ToZoneUpdateMap()
	if err != nil {
		return false, err
	}

	return len(updateOptsMap) > 0, nil
}

func handleDescriptionUpdate(updateOpts *zones.UpdateOpts, resource *resourceSpecT, osResource *osResourceT) {
	description := ptr.Deref(resource.Description, "")
	if osResource.Description != description {
		updateOpts.Description = &description
	}
}

func handleEmailUpdate(updateOpts *zones.UpdateOpts, resource *resourceSpecT, osResource *osResourceT) {
	email := ptr.Deref(resource.Email, "")
	if osResource.Email != email {
		updateOpts.Email = email
	}
}

func handleMastersUpdate(updateOpts *zones.UpdateOpts, resource *resourceSpecT, osResource *osResourceT) {
	mastersMatch := true
	if len(osResource.Masters) != len(resource.Masters) {
		mastersMatch = false
	} else {
		for i, m := range osResource.Masters {
			if m != resource.Masters[i] {
				mastersMatch = false
				break
			}
		}
	}
	if !mastersMatch {
		updateOpts.Masters = resource.Masters
	}
}

func handleTTLUpdate(updateOpts *zones.UpdateOpts, resource *resourceSpecT, osResource *osResourceT) {
	if resource.TTL != nil {
		ttl := int(*resource.TTL)
		if osResource.TTL != ttl {
			updateOpts.TTL = ttl
		}
	}
}

func (actuator dnsZoneActuator) GetResourceReconcilers(ctx context.Context, orcObject orcObjectPT, osResource *osResourceT, controller interfaces.ResourceController) ([]resourceReconciler, progress.ReconcileStatus) {
	return []resourceReconciler{
		actuator.updateResource,
	}, nil
}

type dnszoneHelperFactory struct{}

var _ helperFactory = dnszoneHelperFactory{}

func newActuator(ctx context.Context, orcObject *orcv1alpha1.DNSZone, controller interfaces.ResourceController) (dnsZoneActuator, progress.ReconcileStatus) {
	log := ctrl.LoggerFrom(ctx)

	// Ensure credential secrets exist and have our finalizer
	_, reconcileStatus := credentialsDependency.GetDependencies(ctx, controller.GetK8sClient(), orcObject, func(*corev1.Secret) bool { return true })
	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return dnsZoneActuator{}, reconcileStatus
	}

	clientScope, err := controller.GetScopeFactory().NewClientScopeFromObject(ctx, controller.GetK8sClient(), log, orcObject)
	if err != nil {
		return dnsZoneActuator{}, progress.WrapError(err)
	}
	osClient, err := clientScope.NewDNSZoneClient()
	if err != nil {
		return dnsZoneActuator{}, progress.WrapError(err)
	}

	return dnsZoneActuator{
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

func getDNSZoneName(orcObject orcObjectPT) string {
	name := getResourceName(orcObject)
	if name != "" && name[len(name)-1] != '.' {
		return name + "."
	}
	return name
}
