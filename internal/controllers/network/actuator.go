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
	"fmt"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/dns"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/external"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/mtu"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/portsecurity"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks"
	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/generic"
	"github.com/k-orc/openstack-resource-controller/internal/osclients"
	"github.com/k-orc/openstack-resource-controller/internal/scope"
	orcerrors "github.com/k-orc/openstack-resource-controller/internal/util/errors"
	"github.com/k-orc/openstack-resource-controller/internal/util/neutrontags"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type networkActuator struct {
	*orcv1alpha1.Network
	osClient osclients.NetworkClient
}

func newActuator(ctx context.Context, k8sClient client.Client, scopeFactory scope.Factory, orcObject *orcv1alpha1.Network) (networkActuator, error) {
	log := ctrl.LoggerFrom(ctx)

	clientScope, err := scopeFactory.NewClientScopeFromObject(ctx, k8sClient, log, orcObject)
	if err != nil {
		return networkActuator{}, err
	}
	osClient, err := clientScope.NewNetworkClient()
	if err != nil {
		return networkActuator{}, err
	}

	return networkActuator{
		Network:  orcObject,
		osClient: osClient,
	}, nil
}

var _ generic.CreateResourceActuator[*networkExt] = networkActuator{}
var _ generic.DeleteResourceActuator[*networkExt] = networkActuator{}

func (obj networkActuator) GetObject() client.Object {
	return obj.Network
}

func (obj networkActuator) GetManagementPolicy() orcv1alpha1.ManagementPolicy {
	return obj.Spec.ManagementPolicy
}

func (obj networkActuator) GetManagedOptions() *orcv1alpha1.ManagedOptions {
	return obj.Spec.ManagedOptions
}

func (networkActuator) GetResourceID(osResource *networkExt) string {
	return osResource.ID
}

func getNetworkByID(ctx context.Context, osClient osclients.NetworkClient, id string) (*networkExt, error) {
	osResource := &networkExt{}
	getResult := osClient.GetNetwork(ctx, id)
	err := getResult.ExtractInto(osResource)
	if err != nil {
		return nil, err
	}
	return osResource, nil
}

func (obj networkActuator) GetOSResourceByStatusID(ctx context.Context) (bool, *networkExt, error) {
	if obj.Status.ID == nil {
		return false, nil, nil
	}
	network, err := getNetworkByID(ctx, obj.osClient, *obj.Status.ID)
	return true, network, err
}

func (obj networkActuator) GetOSResourceBySpec(ctx context.Context) (*networkExt, error) {
	if obj.Spec.Resource == nil {
		return nil, nil
	}

	listOpts := listOptsFromCreation(obj.Network)
	return getResourceFromList(ctx, listOpts, obj.osClient)
}

func (obj networkActuator) GetOSResourceByImportID(ctx context.Context) (bool, *networkExt, error) {
	if obj.Spec.Import == nil {
		return false, nil, nil
	}
	if obj.Spec.Import.ID == nil {
		return false, nil, nil
	}
	network, err := getNetworkByID(ctx, obj.osClient, *obj.Spec.Import.ID)
	return true, network, err
}

func (obj networkActuator) GetOSResourceByImportFilter(ctx context.Context) (bool, *networkExt, error) {
	if obj.Spec.Import == nil {
		return false, nil, nil
	}

	if obj.Spec.Import.Filter == nil {
		return false, nil, nil
	}

	listOpts := listOptsFromImportFilter(obj.Spec.Import.Filter)
	osResource, err := getResourceFromList(ctx, listOpts, obj.osClient)
	if err != nil {
		return true, nil, err
	}
	if osResource == nil {
		return true, nil, nil
	}
	return true, osResource, nil
}

func (obj networkActuator) CreateResource(ctx context.Context) ([]generic.WaitingOnEvent, *networkExt, error) {
	resource := obj.Spec.Resource
	if resource == nil {
		// Should have been caught by API validation
		return nil, nil, orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Creation requested, but spec.resource is not set")
	}

	var createOpts networks.CreateOptsBuilder
	{
		createOptsBase := networks.CreateOpts{
			Name:         string(getResourceName(obj.Network)),
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

	osResource := &networkExt{}
	createResult := obj.osClient.CreateNetwork(ctx, createOpts)
	if err := createResult.ExtractInto(osResource); err != nil {
		// We should require the spec to be updated before retrying a create which returned a conflict
		if orcerrors.IsConflict(err) {
			err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration creating resource: "+err.Error(), err)
		}
		return nil, nil, err
	}

	return nil, osResource, nil
}

func (obj networkActuator) DeleteResource(ctx context.Context, network *networkExt) ([]generic.WaitingOnEvent, error) {
	return nil, obj.osClient.DeleteNetwork(ctx, network.ID).ExtractErr()
}

// getResourceName returns the name of the OpenStack resource we should use.
func getResourceName(orcObject *orcv1alpha1.Network) orcv1alpha1.OpenStackName {
	if orcObject.Spec.Resource.Name != nil {
		return *orcObject.Spec.Resource.Name
	}
	return orcv1alpha1.OpenStackName(orcObject.Name)
}

func listOptsFromImportFilter(filter *orcv1alpha1.NetworkFilter) networks.ListOptsBuilder {
	listOpts := networks.ListOpts{
		Name:        string(ptr.Deref(filter.Name, "")),
		Description: string(ptr.Deref(filter.Description, "")),
		Tags:        neutrontags.Join(filter.FilterByNeutronTags.Tags),
		TagsAny:     neutrontags.Join(filter.FilterByNeutronTags.TagsAny),
		NotTags:     neutrontags.Join(filter.FilterByNeutronTags.NotTags),
		NotTagsAny:  neutrontags.Join(filter.FilterByNeutronTags.NotTagsAny),
	}

	return &listOpts
}

// listOptsFromCreation returns a listOpts which will return the OpenStack
// resource which would have been created from the current spec and hopefully no
// other. Its purpose is to automatically adopt a resource that we created but
// failed to write to status.id.
func listOptsFromCreation(osResource *orcv1alpha1.Network) networks.ListOptsBuilder {
	return networks.ListOpts{Name: string(getResourceName(osResource))}
}

func getResourceFromList(ctx context.Context, listOpts networks.ListOptsBuilder, networkClient osclients.NetworkClient) (*networkExt, error) {
	pages, err := networkClient.ListNetwork(listOpts).AllPages(ctx)
	if err != nil {
		return nil, err
	}

	var osResources []networkExt
	err = networks.ExtractNetworksInto(pages, &osResources)
	if err != nil {
		return nil, err
	}

	if len(osResources) == 1 {
		return &osResources[0], nil
	}

	// No resource found
	if len(osResources) == 0 {
		return nil, nil
	}

	// Multiple resources found
	return nil, orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, fmt.Sprintf("Expected to find exactly one OpenStack resource to import. Found %d", len(osResources)))
}
