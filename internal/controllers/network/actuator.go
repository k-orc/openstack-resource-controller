/*
Copyright 2024 The ORC Authors.

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

package network

import (
	"context"
	"iter"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/attributestags"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/dns"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/external"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/mtu"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/portsecurity"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks"
	"k8s.io/utils/ptr"
	"k8s.io/utils/set"
	ctrl "sigs.k8s.io/controller-runtime"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/generic"
	"github.com/k-orc/openstack-resource-controller/internal/osclients"
	orcerrors "github.com/k-orc/openstack-resource-controller/internal/util/errors"
	"github.com/k-orc/openstack-resource-controller/internal/util/neutrontags"
)

type (
	osResourceT = osclients.NetworkExt

	createResourceActuator    = generic.CreateResourceActuator[orcObjectPT, orcObjectT, filterT, osResourceT]
	deleteResourceActuator    = generic.DeleteResourceActuator[orcObjectPT, orcObjectT, osResourceT]
	reconcileResourceActuator = generic.ReconcileResourceActuator[orcObjectPT, osResourceT]
	resourceReconciler        = generic.ResourceReconciler[orcObjectPT, osResourceT]
	helperFactory             = generic.ResourceHelperFactory[orcObjectPT, orcObjectT, resourceSpecT, filterT, osResourceT]
)

type networkActuator struct {
	osClient osclients.NetworkClient
}

var _ createResourceActuator = networkActuator{}
var _ deleteResourceActuator = networkActuator{}
var _ reconcileResourceActuator = networkActuator{}

func (networkActuator) GetResourceID(osResource *osclients.NetworkExt) string {
	return osResource.ID
}

func (actuator networkActuator) GetOSResourceByID(ctx context.Context, id string) (*osclients.NetworkExt, error) {
	return actuator.osClient.GetNetwork(ctx, id)
}

func (actuator networkActuator) ListOSResourcesForAdoption(ctx context.Context, obj orcObjectPT) (iter.Seq2[*osclients.NetworkExt, error], bool) {
	if obj.Spec.Resource == nil {
		return nil, false
	}

	listOpts := networks.ListOpts{Name: string(getResourceName(obj))}
	return actuator.osClient.ListNetwork(ctx, listOpts), true
}

func (actuator networkActuator) ListOSResourcesForImport(ctx context.Context, filter filterT) iter.Seq2[*osclients.NetworkExt, error] {
	listOpts := networks.ListOpts{
		Name:        string(ptr.Deref(filter.Name, "")),
		Description: string(ptr.Deref(filter.Description, "")),
		Tags:        neutrontags.Join(filter.FilterByNeutronTags.Tags),
		TagsAny:     neutrontags.Join(filter.FilterByNeutronTags.TagsAny),
		NotTags:     neutrontags.Join(filter.FilterByNeutronTags.NotTags),
		NotTagsAny:  neutrontags.Join(filter.FilterByNeutronTags.NotTagsAny),
	}

	return actuator.osClient.ListNetwork(ctx, listOpts)
}

func (actuator networkActuator) CreateResource(ctx context.Context, obj orcObjectPT) ([]generic.WaitingOnEvent, *osclients.NetworkExt, error) {
	resource := obj.Spec.Resource
	if resource == nil {
		// Should have been caught by API validation
		return nil, nil, orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Creation requested, but spec.resource is not set")
	}

	var createOpts networks.CreateOptsBuilder
	{
		createOptsBase := networks.CreateOpts{
			Name:         string(getResourceName(obj)),
			Description:  string(*resource.Description),
			AdminStateUp: resource.AdminStateUp,
			Shared:       resource.Shared,
		}

		if len(resource.AvailabilityZoneHints) > 0 {
			createOptsBase.AvailabilityZoneHints = make([]string, len(resource.AvailabilityZoneHints))
			for i := range resource.AvailabilityZoneHints {
				createOptsBase.AvailabilityZoneHints[i] = string(resource.AvailabilityZoneHints[i])
			}
		}
		createOpts = createOptsBase
	}

	if resource.DNSDomain != nil {
		createOpts = &dns.NetworkCreateOptsExt{
			CreateOptsBuilder: createOpts,
			DNSDomain:         string(*resource.DNSDomain),
		}
	}

	if resource.MTU != nil {
		createOpts = &mtu.CreateOptsExt{
			CreateOptsBuilder: createOpts,
			MTU:               int(*resource.MTU),
		}
	}

	if resource.PortSecurityEnabled != nil {
		createOpts = &portsecurity.NetworkCreateOptsExt{
			CreateOptsBuilder:   createOpts,
			PortSecurityEnabled: resource.PortSecurityEnabled,
		}
	}

	if resource.External != nil {
		createOpts = &external.CreateOptsExt{
			CreateOptsBuilder: createOpts,
			External:          resource.External,
		}
	}

	osResource, err := actuator.osClient.CreateNetwork(ctx, createOpts)
	if err != nil {
		// We should require the spec to be updated before retrying a create which returned a conflict
		if orcerrors.IsConflict(err) {
			err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration creating resource: "+err.Error(), err)
		}
		return nil, nil, err
	}

	return nil, osResource, nil
}

func (actuator networkActuator) DeleteResource(ctx context.Context, _ orcObjectPT, network *osclients.NetworkExt) ([]generic.WaitingOnEvent, error) {
	return nil, actuator.osClient.DeleteNetwork(ctx, network.ID)
}

func (actuator networkActuator) GetResourceReconcilers(ctx context.Context, orcObject orcObjectPT, osResource *osclients.NetworkExt, controller generic.ResourceController) ([]resourceReconciler, error) {
	return []resourceReconciler{
		actuator.updateTags,
	}, nil
}

func (actuator networkActuator) updateTags(ctx context.Context, orcObject orcObjectPT, osResource *osclients.NetworkExt) ([]generic.WaitingOnEvent, error) {
	resourceTagSet := set.New[string](osResource.Tags...)
	objectTagSet := set.New[string]()
	for i := range orcObject.Spec.Resource.Tags {
		objectTagSet.Insert(string(orcObject.Spec.Resource.Tags[i]))
	}
	var err error
	if !objectTagSet.Equal(resourceTagSet) {
		opts := attributestags.ReplaceAllOpts{Tags: objectTagSet.SortedList()}
		_, err = actuator.osClient.ReplaceAllAttributesTags(ctx, "networks", osResource.ID, &opts)
	}
	return nil, err
}

type networkHelperFactory struct{}

var _ helperFactory = networkHelperFactory{}

func newActuator(ctx context.Context, orcObject *orcv1alpha1.Network, controller generic.ResourceController) (networkActuator, error) {
	log := ctrl.LoggerFrom(ctx)

	clientScope, err := controller.GetScopeFactory().NewClientScopeFromObject(ctx, controller.GetK8sClient(), log, orcObject)
	if err != nil {
		return networkActuator{}, err
	}
	osClient, err := clientScope.NewNetworkClient()
	if err != nil {
		return networkActuator{}, err
	}

	return networkActuator{
		osClient: osClient,
	}, nil
}

func (networkHelperFactory) NewAPIObjectAdapter(obj orcObjectPT) adapterI {
	return networkAdapter{obj}
}

func (networkHelperFactory) NewCreateActuator(ctx context.Context, orcObject orcObjectPT, controller generic.ResourceController) ([]generic.WaitingOnEvent, createResourceActuator, error) {
	actuator, err := newActuator(ctx, orcObject, controller)
	return nil, actuator, err
}

func (networkHelperFactory) NewDeleteActuator(ctx context.Context, orcObject orcObjectPT, controller generic.ResourceController) ([]generic.WaitingOnEvent, deleteResourceActuator, error) {
	actuator, err := newActuator(ctx, orcObject, controller)
	return nil, actuator, err
}
