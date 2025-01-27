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
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/attributestags"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/subnets"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"k8s.io/utils/set"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/generic"
	"github.com/k-orc/openstack-resource-controller/internal/osclients"
	orcerrors "github.com/k-orc/openstack-resource-controller/internal/util/errors"
	"github.com/k-orc/openstack-resource-controller/internal/util/neutrontags"
)

// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=subnets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=subnets/status,verbs=get;update;patch

type osResourcePT = *subnets.Subnet

type subnetActuator struct {
	obj        *orcv1alpha1.Subnet
	controller generic.ResourceController
	osClient   osclients.NetworkClient
}

type subnetCreateActuator struct {
	subnetActuator
	networkID string
}

type subnetDeleteActuator struct {
	subnetActuator
}

var _ generic.DeleteResourceActuator[*subnets.Subnet] = subnetDeleteActuator{}
var _ generic.CreateResourceActuator[*subnets.Subnet] = subnetCreateActuator{}

func (actuator subnetActuator) GetObject() client.Object {
	return actuator.obj
}

func (actuator subnetActuator) GetController() generic.ResourceController {
	return actuator.controller
}

func (actuator subnetActuator) GetManagementPolicy() orcv1alpha1.ManagementPolicy {
	return actuator.obj.Spec.ManagementPolicy
}

func (actuator subnetActuator) GetManagedOptions() *orcv1alpha1.ManagedOptions {
	return actuator.obj.Spec.ManagedOptions
}

func (subnetActuator) GetResourceID(osResource *subnets.Subnet) string {
	return osResource.ID
}

func (actuator subnetActuator) GetStatusID() *string {
	return actuator.obj.Status.ID
}

func (actuator subnetActuator) GetOSResourceByStatusID(ctx context.Context) (bool, *subnets.Subnet, error) {
	if actuator.obj.Status.ID == nil {
		return false, nil, nil
	}
	osResource, err := actuator.osClient.GetSubnet(ctx, *actuator.obj.Status.ID)
	return true, osResource, err
}

func (actuator subnetActuator) GetOSResourceBySpec(ctx context.Context) (*subnets.Subnet, error) {
	listOpts := listOptsFromCreation(actuator.obj)
	return getResourceFromList(ctx, listOpts, actuator.osClient)
}

func (actuator subnetActuator) GetOSResourceByImportID(ctx context.Context) (bool, *subnets.Subnet, error) {
	if actuator.obj.Spec.Import == nil || actuator.obj.Spec.Import.ID == nil {
		return false, nil, nil
	}
	osResource, err := actuator.osClient.GetSubnet(ctx, *actuator.obj.Spec.Import.ID)
	return true, osResource, err
}

func (actuator subnetCreateActuator) GetOSResourceByImportFilter(ctx context.Context) (bool, *subnets.Subnet, error) {
	if actuator.obj.Spec.Import == nil || actuator.obj.Spec.Import.Filter == nil {
		return false, nil, nil
	}
	listOpts := listOptsFromImportFilter(actuator.obj.Spec.Import.Filter, actuator.networkID)
	osResource, err := getResourceFromList(ctx, listOpts, actuator.osClient)
	return true, osResource, err
}

func (actuator subnetCreateActuator) CreateResource(ctx context.Context) ([]generic.WaitingOnEvent, *subnets.Subnet, error) {
	resource := actuator.obj.Spec.Resource
	if resource == nil {
		// Should have been caught by API validation
		return nil, nil, orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Creation requested, but spec.resource is not set")
	}

	createOpts := subnets.CreateOpts{
		NetworkID:         actuator.networkID,
		CIDR:              string(resource.CIDR),
		Name:              string(getResourceName(actuator.obj)),
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

	osResource, err := actuator.osClient.CreateSubnet(ctx, &createOpts)

	// We should require the spec to be updated before retrying a create which returned a conflict
	if orcerrors.IsConflict(err) {
		err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration creating resource: "+err.Error(), err)
	}

	return nil, osResource, err
}

func (actuator subnetDeleteActuator) DeleteResource(ctx context.Context, osResource *subnets.Subnet) ([]generic.WaitingOnEvent, error) {
	k8sClient := actuator.controller.GetK8sClient()

	// Delete any RouterInterface first, as this would prevent deletion of the subnet
	routerInterface, err := getRouterInterface(ctx, k8sClient, actuator.obj)
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

	return nil, actuator.osClient.DeleteSubnet(ctx, *actuator.obj.Status.ID)
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

var _ generic.UpdateResourceActuator[orcObjectPT, osResourcePT] = subnetActuator{}

type resourceUpdater = generic.ResourceUpdater[orcObjectPT, osResourcePT]

func (actuator subnetActuator) GetResourceUpdaters(ctx context.Context, orcObject orcObjectPT, osResource osResourcePT, controller generic.ResourceController) ([]resourceUpdater, error) {
	return []resourceUpdater{
		actuator.updateTags,
		actuator.ensureRouterInterface,
	}, nil
}

func (actuator subnetActuator) ensureRouterInterface(ctx context.Context, orcObject orcObjectPT, osResource osResourcePT) ([]generic.WaitingOnEvent, orcObjectPT, osResourcePT, error) {
	k8sClient := actuator.controller.GetK8sClient()

	var waitEvents []generic.WaitingOnEvent
	var err error

	routerInterface, err := getRouterInterface(ctx, k8sClient, orcObject)
	if routerInterfaceMatchesSpec(routerInterface, orcObject.Name, orcObject.Spec.Resource) {
		// Nothing to do
		return waitEvents, orcObject, osResource, err
	}

	// If it doesn't match we should delete any existing interface
	if routerInterface != nil {
		if routerInterface.GetDeletionTimestamp().IsZero() {
			if err := k8sClient.Delete(ctx, routerInterface); err != nil {
				return waitEvents, orcObject, osResource, fmt.Errorf("deleting RouterInterface %s: %w", client.ObjectKeyFromObject(routerInterface), err)
			}
		}
		waitEvents = append(waitEvents, generic.WaitingOnORCDeleted("routerinterface", routerInterface.Name))
		return waitEvents, orcObject, osResource, err
	}

	// Otherwise create it
	routerInterface = &orcv1alpha1.RouterInterface{}
	routerInterface.Name = getRouterInterfaceName(orcObject)
	routerInterface.Namespace = orcObject.Namespace
	routerInterface.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion:         orcObject.APIVersion,
			Kind:               orcObject.Kind,
			Name:               orcObject.Name,
			UID:                orcObject.UID,
			BlockOwnerDeletion: ptr.To(true),
		},
	}
	routerInterface.Spec = orcv1alpha1.RouterInterfaceSpec{
		Type:      orcv1alpha1.RouterInterfaceTypeSubnet,
		RouterRef: *orcObject.Spec.Resource.RouterRef,
		SubnetRef: ptr.To(orcv1alpha1.KubernetesNameRef(orcObject.Name)),
	}

	if err := k8sClient.Create(ctx, routerInterface); err != nil {
		return waitEvents, orcObject, osResource, fmt.Errorf("creating RouterInterface %s: %w", client.ObjectKeyFromObject(orcObject), err)
	}
	waitEvents = append(waitEvents, generic.WaitingOnORCReady("routerinterface", routerInterface.Name))

	return waitEvents, orcObject, osResource, err
}

func (actuator subnetActuator) updateTags(ctx context.Context, orcObject orcObjectPT, osResource osResourcePT) ([]generic.WaitingOnEvent, orcObjectPT, osResourcePT, error) {
	resourceTagSet := set.New[string](osResource.Tags...)
	objectTagSet := set.New[string]()
	for i := range orcObject.Spec.Resource.Tags {
		objectTagSet.Insert(string(orcObject.Spec.Resource.Tags[i]))
	}
	var err error
	if !objectTagSet.Equal(resourceTagSet) {
		opts := attributestags.ReplaceAllOpts{Tags: objectTagSet.SortedList()}
		_, err = actuator.osClient.ReplaceAllAttributesTags(ctx, "subnets", osResource.ID, &opts)
	}
	return nil, orcObject, osResource, err
}

func getRouterInterfaceName(orcObject *orcv1alpha1.Subnet) string {
	return orcObject.Name + "-subnet"
}

func routerInterfaceMatchesSpec(routerInterface *orcv1alpha1.RouterInterface, objectName string, resource *orcv1alpha1.SubnetResourceSpec) bool {
	// No routerRef -> there should be no routerInterface
	if resource.RouterRef == nil {
		return routerInterface == nil
	}

	// The router interface should:
	// * Exist
	// * Be of Subnet type
	// * Reference this subnet
	// * Reference the router in our spec

	if routerInterface == nil {
		return false
	}

	if routerInterface.Spec.Type != orcv1alpha1.RouterInterfaceTypeSubnet {
		return false
	}

	if string(ptr.Deref(routerInterface.Spec.SubnetRef, "")) != objectName {
		return false
	}

	return routerInterface.Spec.RouterRef == *resource.RouterRef
}

// getRouterInterface returns the router interface for this subnet, identified by its name
// returns nil for routerinterface without returning an error if the routerinterface does not exist
func getRouterInterface(ctx context.Context, k8sClient client.Client, orcObject *orcv1alpha1.Subnet) (*orcv1alpha1.RouterInterface, error) {
	routerInterface := &orcv1alpha1.RouterInterface{}
	err := k8sClient.Get(ctx, types.NamespacedName{Name: getRouterInterfaceName(orcObject), Namespace: orcObject.GetNamespace()}, routerInterface)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("fetching RouterInterface: %w", err)
	}

	return routerInterface, nil
}

type subnetActuatorFactory struct{}

var _ generic.ActuatorFactory[orcObjectPT, osResourcePT] = subnetActuatorFactory{}

func (subnetActuatorFactory) NewCreateActuator(ctx context.Context, orcObject orcObjectPT, controller generic.ResourceController) ([]generic.WaitingOnEvent, generic.CreateResourceActuator[osResourcePT], error) {
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
	return nil, subnetCreateActuator{
		subnetActuator: actuator,
		networkID:      *orcNetwork.Status.ID,
	}, nil
}

func (subnetActuatorFactory) NewDeleteActuator(ctx context.Context, orcObject orcObjectPT, controller generic.ResourceController) ([]generic.WaitingOnEvent, generic.DeleteResourceActuator[osResourcePT], error) {
	actuator, err := newActuator(ctx, controller, orcObject)
	if err != nil {
		return nil, nil, err
	}
	return nil, subnetDeleteActuator{
		subnetActuator: actuator,
	}, nil
}

func newActuator(ctx context.Context, controller generic.ResourceController, orcObject *orcv1alpha1.Subnet) (subnetActuator, error) {
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
		obj:        orcObject,
		osClient:   osClient,
		controller: controller,
	}, nil
}
