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

package subnet

import (
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/subnets"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/generic"
	"github.com/k-orc/openstack-resource-controller/internal/osclients"
	orcerrors "github.com/k-orc/openstack-resource-controller/internal/util/errors"
	"github.com/k-orc/openstack-resource-controller/internal/util/neutrontags"
)

type subnetActuator struct {
	*orcv1alpha1.Subnet
	controller generic.ResourceControllerCommon
	osClient   osclients.NetworkClient
}

type subnetCreateActuator struct {
	subnetActuator
	networkID string
}

type subnetDeleteActuator struct {
	subnetActuator
}

func newActuator(ctx context.Context, controller generic.ResourceControllerCommon, orcObject *orcv1alpha1.Subnet) (subnetActuator, error) {
	if orcObject == nil {
		return subnetActuator{}, fmt.Errorf("orcObject may not be nil")
	}

	log := ctrl.LoggerFrom(ctx)
	clientScope, err := controller.GetScopeFactory().NewClientScopeFromObject(ctx, controller.GetK8sClient(), log, orcObject)
	if err != nil {
		return subnetActuator{}, err
	}
	osClient, err := clientScope.NewNetworkClient()
	if err != nil {
		return subnetActuator{}, err
	}

	return subnetActuator{
		Subnet:     orcObject,
		osClient:   osClient,
		controller: controller,
	}, nil
}

func newCreateActuator(ctx context.Context, controller generic.ResourceControllerCommon, orcObject *orcv1alpha1.Subnet) ([]generic.WaitingOnEvent, *subnetCreateActuator, error) {
	orcNetwork := &orcv1alpha1.Network{}
	if err := controller.GetK8sClient().Get(ctx, client.ObjectKey{Name: string(orcObject.Spec.NetworkRef), Namespace: orcObject.Namespace}, orcNetwork); err != nil {
		if apierrors.IsNotFound(err) {
			return []generic.WaitingOnEvent{generic.WaitingOnORCExist("Network", string(orcObject.Spec.NetworkRef))}, nil, nil
		}
		return nil, nil, err
	}

	if !orcv1alpha1.IsAvailable(orcNetwork) || orcNetwork.Status.ID == nil {
		return []generic.WaitingOnEvent{generic.WaitingOnORCReady("Network", string(orcObject.Spec.NetworkRef))}, nil, nil
	}

	actuator, err := newActuator(ctx, controller, orcObject)
	if err != nil {
		return nil, nil, err
	}
	return nil, &subnetCreateActuator{
		subnetActuator: actuator,
		networkID:      *orcNetwork.Status.ID,
	}, nil
}

func newDeleteActuator(ctx context.Context, controller generic.ResourceControllerCommon, orcObject *orcv1alpha1.Subnet) (*subnetDeleteActuator, error) {
	actuator, err := newActuator(ctx, controller, orcObject)
	if err != nil {
		return nil, err
	}
	return &subnetDeleteActuator{
		subnetActuator: actuator,
	}, nil
}

var _ generic.DeleteResourceActuator[*subnets.Subnet] = subnetDeleteActuator{}
var _ generic.CreateResourceActuator[*subnets.Subnet] = subnetCreateActuator{}

func (obj subnetActuator) GetObject() client.Object {
	return obj.Subnet
}

func (obj subnetActuator) GetController() generic.ResourceControllerCommon {
	return obj.controller
}

func (obj subnetActuator) GetManagementPolicy() orcv1alpha1.ManagementPolicy {
	return obj.Spec.ManagementPolicy
}

func (obj subnetActuator) GetManagedOptions() *orcv1alpha1.ManagedOptions {
	return obj.Spec.ManagedOptions
}

func (subnetActuator) GetResourceID(osResource *subnets.Subnet) string {
	return osResource.ID
}

func (obj subnetActuator) GetOSResourceByStatusID(ctx context.Context) (bool, *subnets.Subnet, error) {
	if obj.Status.ID == nil {
		return false, nil, nil
	}
	osResource, err := obj.osClient.GetSubnet(ctx, *obj.Status.ID)
	return true, osResource, err
}

func (obj subnetActuator) GetOSResourceBySpec(ctx context.Context) (*subnets.Subnet, error) {
	listOpts := listOptsFromCreation(obj.Subnet)
	return getResourceFromList(ctx, listOpts, obj.osClient)
}

func (obj subnetActuator) GetOSResourceByImportID(ctx context.Context) (bool, *subnets.Subnet, error) {
	if obj.Spec.Import == nil || obj.Spec.Import.ID == nil {
		return false, nil, nil
	}
	osResource, err := obj.osClient.GetSubnet(ctx, *obj.Spec.Import.ID)
	return true, osResource, err
}

func (obj subnetCreateActuator) GetOSResourceByImportFilter(ctx context.Context) (bool, *subnets.Subnet, error) {
	if obj.Spec.Import == nil || obj.Spec.Import.Filter == nil {
		return false, nil, nil
	}
	listOpts := listOptsFromImportFilter(obj.Spec.Import.Filter, obj.networkID)
	osResource, err := getResourceFromList(ctx, listOpts, obj.osClient)
	return true, osResource, err
}

func (orcObject subnetCreateActuator) CreateResource(ctx context.Context) ([]generic.WaitingOnEvent, *subnets.Subnet, error) {
	resource := orcObject.Spec.Resource
	if resource == nil {
		// Should have been caught by API validation
		return nil, nil, orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Creation requested, but spec.resource is not set")
	}

	createOpts := subnets.CreateOpts{
		NetworkID:         orcObject.networkID,
		CIDR:              string(resource.CIDR),
		Name:              string(getResourceName(orcObject.Subnet)),
		Description:       string(ptr.Deref(resource.Description, "")),
		IPVersion:         gophercloud.IPVersion(resource.IPVersion),
		EnableDHCP:        resource.EnableDHCP,
		DNSPublishFixedIP: resource.DNSPublishFixedIP,
	}

	if len(resource.AllocationPools) > 0 {
		createOpts.AllocationPools = make([]subnets.AllocationPool, len(resource.AllocationPools))
		for i := range resource.AllocationPools {
			createOpts.AllocationPools[i].Start = string(resource.AllocationPools[i].Start)
			createOpts.AllocationPools[i].End = string(resource.AllocationPools[i].End)
		}
	}

	if resource.Gateway != nil {
		switch resource.Gateway.Type {
		case orcv1alpha1.SubnetGatewayTypeAutomatic:
			// Nothing to do
		case orcv1alpha1.SubnetGatewayTypeNone:
			createOpts.GatewayIP = ptr.To("")
		case orcv1alpha1.SubnetGatewayTypeIP:
			fallthrough
		default:
			createOpts.GatewayIP = (*string)(resource.Gateway.IP)
		}
	}

	if len(resource.DNSNameservers) > 0 {
		createOpts.DNSNameservers = make([]string, len(resource.DNSNameservers))
		for i := range resource.DNSNameservers {
			createOpts.DNSNameservers[i] = string(resource.DNSNameservers[i])
		}
	}

	if len(resource.HostRoutes) > 0 {
		createOpts.HostRoutes = make([]subnets.HostRoute, len(resource.HostRoutes))
		for i := range resource.HostRoutes {
			createOpts.HostRoutes[i].DestinationCIDR = string(resource.HostRoutes[i].Destination)
			createOpts.HostRoutes[i].NextHop = string(resource.HostRoutes[i].NextHop)
		}
	}

	if resource.IPv6 != nil {
		createOpts.IPv6AddressMode = string(ptr.Deref(resource.IPv6.AddressMode, ""))
		createOpts.IPv6RAMode = string(ptr.Deref(resource.IPv6.RAMode, ""))
	}

	osResource, err := orcObject.osClient.CreateSubnet(ctx, &createOpts)

	// We should require the spec to be updated before retrying a create which returned a conflict
	if orcerrors.IsConflict(err) {
		err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration creating resource: "+err.Error(), err)
	}

	return nil, osResource, err
}

func (obj subnetDeleteActuator) DeleteResource(ctx context.Context, osResource *subnets.Subnet) ([]generic.WaitingOnEvent, error) {
	k8sClient := obj.controller.GetK8sClient()

	// Delete any RouterInterface first, as this would prevent deletion of the subnet
	routerInterface, err := getRouterInterface(ctx, k8sClient, obj.Subnet)
	if err != nil {
		return nil, err
	}

	if routerInterface != nil {
		// We will be reconciled again when it's gone
		if routerInterface.GetDeletionTimestamp().IsZero() {
			if err := k8sClient.Delete(ctx, routerInterface); err != nil {
				return nil, err
			}
		}
		return []generic.WaitingOnEvent{generic.WaitingOnORCDeleted("RouterInterface", routerInterface.GetName())}, nil
	}

	return nil, obj.osClient.DeleteSubnet(ctx, *obj.Status.ID)
}

// getResourceName returns the name of the OpenStack resource we should use.
func getResourceName(orcObject *orcv1alpha1.Subnet) orcv1alpha1.OpenStackName {
	if orcObject.Spec.Resource.Name != nil {
		return *orcObject.Spec.Resource.Name
	}
	return orcv1alpha1.OpenStackName(orcObject.Name)
}

func listOptsFromImportFilter(filter *orcv1alpha1.SubnetFilter, networkID string) subnets.ListOptsBuilder {
	listOpts := subnets.ListOpts{
		Name:        string(ptr.Deref(filter.Name, "")),
		Description: string(ptr.Deref(filter.Description, "")),
		NetworkID:   networkID,
		IPVersion:   int(ptr.Deref(filter.IPVersion, 0)),
		GatewayIP:   string(ptr.Deref(filter.GatewayIP, "")),
		CIDR:        string(ptr.Deref(filter.CIDR, "")),
		Tags:        neutrontags.Join(filter.FilterByNeutronTags.Tags),
		TagsAny:     neutrontags.Join(filter.FilterByNeutronTags.TagsAny),
		NotTags:     neutrontags.Join(filter.FilterByNeutronTags.NotTags),
		NotTagsAny:  neutrontags.Join(filter.FilterByNeutronTags.NotTagsAny),
	}
	if filter.IPv6 != nil {
		listOpts.IPv6AddressMode = string(ptr.Deref(filter.IPv6.AddressMode, ""))
		listOpts.IPv6RAMode = string(ptr.Deref(filter.IPv6.RAMode, ""))
	}

	return &listOpts
}

// listOptsFromCreation returns a listOpts which will return the OpenStack
// resource which would have been created from the current spec and hopefully no
// other. Its purpose is to automatically adopt a resource that we created but
// failed to write to status.id.
func listOptsFromCreation(osResource *orcv1alpha1.Subnet) subnets.ListOptsBuilder {
	return subnets.ListOpts{Name: string(getResourceName(osResource))}
}

func getResourceFromList(ctx context.Context, listOpts subnets.ListOptsBuilder, networkClient osclients.NetworkClient) (*subnets.Subnet, error) {
	osResources, err := networkClient.ListSubnet(ctx, listOpts)
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
