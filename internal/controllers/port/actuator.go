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

package port

import (
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/generic"
	osclients "github.com/k-orc/openstack-resource-controller/internal/osclients"
	"github.com/k-orc/openstack-resource-controller/internal/scope"
	orcerrors "github.com/k-orc/openstack-resource-controller/internal/util/errors"
	"github.com/k-orc/openstack-resource-controller/internal/util/neutrontags"
)

type portActuator struct {
	*orcv1alpha1.Port
	osClient osclients.NetworkClient
}

type portCreateActuator struct {
	portActuator

	k8sClient client.Client
	networkID orcv1alpha1.UUID
}

func newActuator(ctx context.Context, k8sClient client.Client, scopeFactory scope.Factory, orcObject *orcv1alpha1.Port) (portActuator, error) {
	log := ctrl.LoggerFrom(ctx)

	clientScope, err := scopeFactory.NewClientScopeFromObject(ctx, k8sClient, log, orcObject)
	if err != nil {
		return portActuator{}, err
	}
	osClient, err := clientScope.NewNetworkClient()
	if err != nil {
		return portActuator{}, err
	}

	return portActuator{
		Port:     orcObject,
		osClient: osClient,
	}, nil
}

func newCreateActuator(ctx context.Context, k8sClient client.Client, scopeFactory scope.Factory, orcObject *orcv1alpha1.Port, networkID orcv1alpha1.UUID) (portCreateActuator, error) {
	portActuator, err := newActuator(ctx, k8sClient, scopeFactory, orcObject)
	if err != nil {
		return portCreateActuator{}, err
	}

	return portCreateActuator{
		portActuator: portActuator,
		k8sClient:    k8sClient,
		networkID:    networkID,
	}, nil
}

var _ generic.DeleteResourceActuator[*ports.Port] = portActuator{}
var _ generic.CreateResourceActuator[*ports.Port] = portCreateActuator{}

func (obj portActuator) GetManagementPolicy() orcv1alpha1.ManagementPolicy {
	return obj.Spec.ManagementPolicy
}

func (obj portActuator) GetManagedOptions() *orcv1alpha1.ManagedOptions {
	return obj.Spec.ManagedOptions
}

func (portActuator) GetResourceID(osResource *ports.Port) string {
	return osResource.ID
}

func (obj portActuator) GetOSResourceByStatusID(ctx context.Context) (bool, *ports.Port, error) {
	if obj.Status.ID == nil {
		return false, nil, nil
	}

	port, err := obj.osClient.GetPort(ctx, *obj.Status.ID)
	return true, port, err
}

func (obj portActuator) GetOSResourceBySpec(ctx context.Context) (*ports.Port, error) {
	if obj.Spec.Resource == nil {
		return nil, nil
	}

	listOpts := listOptsFromCreation(obj.Port)
	return getResourceFromList(ctx, listOpts, obj.osClient)
}

func (obj portCreateActuator) GetOSResourceByImportID(ctx context.Context) (bool, *ports.Port, error) {
	if obj.Spec.Import == nil {
		return false, nil, nil
	}
	if obj.Spec.Import.ID == nil {
		return false, nil, nil
	}

	port, err := obj.osClient.GetPort(ctx, *obj.Spec.Import.ID)
	return true, port, err
}

func (obj portCreateActuator) GetOSResourceByImportFilter(ctx context.Context) (bool, *ports.Port, error) {
	if obj.Spec.Import == nil {
		return false, nil, nil
	}
	if obj.Spec.Import.Filter == nil {
		return false, nil, nil
	}

	listOpts := listOptsFromImportFilter(obj.Spec.Import.Filter, obj.networkID)
	port, err := getResourceFromList(ctx, listOpts, obj.osClient)
	return true, port, err
}

func (obj portCreateActuator) CreateResource(ctx context.Context) ([]string, *ports.Port, error) {
	resource := obj.Spec.Resource

	if resource == nil {
		// Should have been caught by API validation
		return nil, nil, orcerrors.Terminal(orcv1alpha1.OpenStackConditionReasonInvalidConfiguration, "Creation requested, but spec.resource is not set")
	}

	createOpts := ports.CreateOpts{
		NetworkID:   string(obj.networkID),
		Name:        string(getResourceName(obj.Port)),
		Description: string(ptr.Deref(resource.Description, "")),
	}

	if len(resource.AllowedAddressPairs) > 0 {
		createOpts.AllowedAddressPairs = make([]ports.AddressPair, len(resource.AllowedAddressPairs))
		for i := range resource.AllowedAddressPairs {
			createOpts.AllowedAddressPairs[i].IPAddress = string(*resource.AllowedAddressPairs[i].IP)
			if resource.AllowedAddressPairs[i].MAC != nil {
				createOpts.AllowedAddressPairs[i].MACAddress = string(*resource.AllowedAddressPairs[i].MAC)
			}
		}
	}

	// We explicitly disable creation of IP addresses by passing an empty
	// value whenever the user does not specify addresses
	fixedIPs := make([]ports.IP, len(resource.Addresses))
	for i := range resource.Addresses {
		subnet := &orcv1alpha1.Subnet{}
		key := client.ObjectKey{Name: string(*resource.Addresses[i].SubnetRef), Namespace: obj.Namespace}
		if err := obj.k8sClient.Get(ctx, key, subnet); err != nil {
			if orcerrors.IsNotFound(err) {
				return []string{generic.WaitingOnCreationMsg("Subnet", key.Name)}, nil, nil
			}
			return nil, nil, fmt.Errorf("fetching subnet %s: %w", key.Name, err)
		}

		if !orcv1alpha1.IsAvailable(subnet) || subnet.Status.ID == nil {
			return []string{generic.WaitingOnAvailableMsg("Subnet", key.Name)}, nil, nil
		}
		fixedIPs[i].SubnetID = *subnet.Status.ID

		if resource.Addresses[i].IP != nil {
			fixedIPs[i].IPAddress = string(*resource.Addresses[i].IP)
		}
	}
	createOpts.FixedIPs = fixedIPs

	// We explicitly disable default security groups by passing an empty
	// value whenever the user does not specifies security groups
	securityGroups := make([]string, len(resource.SecurityGroupRefs))
	for i := range resource.SecurityGroupRefs {
		securityGroup := &orcv1alpha1.SecurityGroup{}
		key := client.ObjectKey{Name: string(resource.SecurityGroupRefs[i]), Namespace: obj.Namespace}
		if err := obj.k8sClient.Get(ctx, key, securityGroup); err != nil {
			if orcerrors.IsNotFound(err) {
				return []string{generic.WaitingOnCreationMsg("Subnet", key.Name)}, nil, nil
			}
			return nil, nil, fmt.Errorf("fetching securitygroup %s: %w", key.Name, err)
		}

		if !orcv1alpha1.IsAvailable(securityGroup) || securityGroup.Status.ID == nil {
			return []string{generic.WaitingOnAvailableMsg("Subnet", key.Name)}, nil, nil
		}
		securityGroups[i] = *securityGroup.Status.ID
	}
	createOpts.SecurityGroups = &securityGroups

	osResource, err := obj.osClient.CreatePort(ctx, &createOpts)
	if err != nil {
		// We should require the spec to be updated before retrying a create which returned a conflict
		if orcerrors.IsConflict(err) {
			err = orcerrors.Terminal(orcv1alpha1.OpenStackConditionReasonInvalidConfiguration, "invalid configuration creating resource: "+err.Error(), err)
		}
		return nil, nil, err
	}

	return nil, osResource, nil
}

func (obj portActuator) DeleteResource(ctx context.Context, flavor *ports.Port) error {
	return obj.osClient.DeletePort(ctx, flavor.ID)
}

// getResourceName returns the name of the OpenStack resource we should use.
func getResourceName(orcObject *orcv1alpha1.Port) orcv1alpha1.OpenStackName {
	if orcObject.Spec.Resource.Name != nil {
		return *orcObject.Spec.Resource.Name
	}
	return orcv1alpha1.OpenStackName(orcObject.Name)
}

func listOptsFromImportFilter(filter *orcv1alpha1.PortFilter, networkID orcv1alpha1.UUID) ports.ListOptsBuilder {
	listOpts := ports.ListOpts{
		Name:        string(ptr.Deref(filter.Name, "")),
		Description: string(ptr.Deref(filter.Description, "")),
		NetworkID:   string(networkID),
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
func listOptsFromCreation(osResource *orcv1alpha1.Port) ports.ListOptsBuilder {
	return ports.ListOpts{Name: string(getResourceName(osResource))}
}

func getResourceFromList(ctx context.Context, listOpts ports.ListOptsBuilder, networkClient osclients.NetworkClient) (*ports.Port, error) {
	osResources, err := networkClient.ListPort(ctx, listOpts)
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
	return nil, orcerrors.Terminal(orcv1alpha1.OpenStackConditionReasonInvalidConfiguration, fmt.Sprintf("Expected to find exactly one OpenStack resource to import. Found %d", len(osResources)))
}
