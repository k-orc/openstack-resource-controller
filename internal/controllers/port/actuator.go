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

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/attributestags"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/utils/ptr"
	"k8s.io/utils/set"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/generic"
	osclients "github.com/k-orc/openstack-resource-controller/internal/osclients"
	orcerrors "github.com/k-orc/openstack-resource-controller/internal/util/errors"
	"github.com/k-orc/openstack-resource-controller/internal/util/neutrontags"
)

type osResourcePT = *ports.Port
type orcObjectPT = *orcv1alpha1.Port

type portActuator struct {
	obj        *orcv1alpha1.Port
	osClient   osclients.NetworkClient
	controller generic.ResourceController
}

type portCreateActuator struct {
	portActuator

	networkID orcv1alpha1.UUID
}

var _ generic.DeleteResourceActuator[*ports.Port] = portActuator{}
var _ generic.CreateResourceActuator[*ports.Port] = portCreateActuator{}

func (actuator portActuator) GetObject() client.Object {
	return actuator.obj
}

func (actuator portActuator) GetController() generic.ResourceController {
	return actuator.controller
}

func (actuator portActuator) GetManagementPolicy() orcv1alpha1.ManagementPolicy {
	return actuator.obj.Spec.ManagementPolicy
}

func (actuator portActuator) GetManagedOptions() *orcv1alpha1.ManagedOptions {
	return actuator.obj.Spec.ManagedOptions
}

func (portActuator) GetResourceID(osResource *ports.Port) string {
	return osResource.ID
}

func (actuator portActuator) GetStatusID() *string {
	return actuator.obj.Status.ID
}

func (actuator portActuator) GetOSResourceByStatusID(ctx context.Context) (bool, *ports.Port, error) {
	if actuator.obj.Status.ID == nil {
		return false, nil, nil
	}

	port, err := actuator.osClient.GetPort(ctx, *actuator.obj.Status.ID)
	return true, port, err
}

func (actuator portActuator) GetOSResourceBySpec(ctx context.Context) (*ports.Port, error) {
	if actuator.obj.Spec.Resource == nil {
		return nil, nil
	}

	listOpts := listOptsFromCreation(actuator.obj)
	return getResourceFromList(ctx, listOpts, actuator.osClient)
}

func (actuator portCreateActuator) GetOSResourceByImportID(ctx context.Context) (bool, *ports.Port, error) {
	if actuator.obj.Spec.Import == nil {
		return false, nil, nil
	}
	if actuator.obj.Spec.Import.ID == nil {
		return false, nil, nil
	}

	port, err := actuator.osClient.GetPort(ctx, *actuator.obj.Spec.Import.ID)
	return true, port, err
}

func (actuator portCreateActuator) GetOSResourceByImportFilter(ctx context.Context) (bool, *ports.Port, error) {
	if actuator.obj.Spec.Import == nil {
		return false, nil, nil
	}
	if actuator.obj.Spec.Import.Filter == nil {
		return false, nil, nil
	}

	listOpts := listOptsFromImportFilter(actuator.obj.Spec.Import.Filter, actuator.networkID)
	port, err := getResourceFromList(ctx, listOpts, actuator.osClient)
	return true, port, err
}

func (actuator portCreateActuator) CreateResource(ctx context.Context) ([]generic.WaitingOnEvent, *ports.Port, error) {
	resource := actuator.obj.Spec.Resource

	if resource == nil {
		// Should have been caught by API validation
		return nil, nil, orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Creation requested, but spec.resource is not set")
	}

	createOpts := ports.CreateOpts{
		NetworkID:   string(actuator.networkID),
		Name:        string(getResourceName(actuator.obj)),
		Description: string(ptr.Deref(resource.Description, "")),
	}

	if len(resource.AllowedAddressPairs) > 0 {
		createOpts.AllowedAddressPairs = make([]ports.AddressPair, len(resource.AllowedAddressPairs))
		for i := range resource.AllowedAddressPairs {
			createOpts.AllowedAddressPairs[i].IPAddress = string(resource.AllowedAddressPairs[i].IP)
			if resource.AllowedAddressPairs[i].MAC != nil {
				createOpts.AllowedAddressPairs[i].MACAddress = string(*resource.AllowedAddressPairs[i].MAC)
			}
		}
	}

	var waitEvents []generic.WaitingOnEvent
	k8sClient := actuator.controller.GetK8sClient()

	// We explicitly disable creation of IP addresses by passing an empty
	// value whenever the user does not specify addresses
	fixedIPs := make([]ports.IP, len(resource.Addresses))
	for i := range resource.Addresses {
		subnet := &orcv1alpha1.Subnet{}
		key := client.ObjectKey{Name: string(resource.Addresses[i].SubnetRef), Namespace: actuator.obj.Namespace}
		if err := k8sClient.Get(ctx, key, subnet); err != nil {
			if apierrors.IsNotFound(err) {
				waitEvents = append(waitEvents, generic.WaitingOnORCExist("Subnet", key.Name))
				continue
			}
			return nil, nil, fmt.Errorf("fetching subnet %s: %w", key.Name, err)
		}

		if !orcv1alpha1.IsAvailable(subnet) || subnet.Status.ID == nil {
			waitEvents = append(waitEvents, generic.WaitingOnORCReady("Subnet", key.Name))
			continue
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
		key := client.ObjectKey{Name: string(resource.SecurityGroupRefs[i]), Namespace: actuator.obj.Namespace}
		if err := k8sClient.Get(ctx, key, securityGroup); err != nil {
			if apierrors.IsNotFound(err) {
				waitEvents = append(waitEvents, generic.WaitingOnORCExist("Subnet", key.Name))
				continue
			}
			return nil, nil, fmt.Errorf("fetching securitygroup %s: %w", key.Name, err)
		}

		if !orcv1alpha1.IsAvailable(securityGroup) || securityGroup.Status.ID == nil {
			waitEvents = append(waitEvents, generic.WaitingOnORCReady("Subnet", key.Name))
			continue
		}
		securityGroups[i] = *securityGroup.Status.ID
	}
	createOpts.SecurityGroups = &securityGroups

	if len(waitEvents) > 0 {
		return waitEvents, nil, nil
	}

	osResource, err := actuator.osClient.CreatePort(ctx, &createOpts)
	if err != nil {
		// We should require the spec to be updated before retrying a create which returned a conflict
		if orcerrors.IsConflict(err) {
			err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration creating resource: "+err.Error(), err)
		}
		return nil, nil, err
	}

	return nil, osResource, nil
}

func (actuator portActuator) DeleteResource(ctx context.Context, flavor *ports.Port) ([]generic.WaitingOnEvent, error) {
	return nil, actuator.osClient.DeletePort(ctx, flavor.ID)
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
	return nil, orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, fmt.Sprintf("Expected to find exactly one OpenStack resource to import. Found %d", len(osResources)))
}

var _ generic.UpdateResourceActuator[orcObjectPT, osResourcePT] = portActuator{}

type resourceUpdater = generic.ResourceUpdater[orcObjectPT, osResourcePT]

func (actuator portActuator) GetResourceUpdaters(ctx context.Context, orcObject orcObjectPT, osResource osResourcePT, controller generic.ResourceController) ([]resourceUpdater, error) {
	return []resourceUpdater{
		actuator.updateTags,
	}, nil
}

func (actuator portActuator) updateTags(ctx context.Context, orcObject orcObjectPT, osResource osResourcePT) ([]generic.WaitingOnEvent, orcObjectPT, osResourcePT, error) {
	resourceTagSet := set.New[string](osResource.Tags...)
	objectTagSet := set.New[string]()
	for i := range orcObject.Spec.Resource.Tags {
		objectTagSet.Insert(string(orcObject.Spec.Resource.Tags[i]))
	}
	var err error
	if !objectTagSet.Equal(resourceTagSet) {
		opts := attributestags.ReplaceAllOpts{Tags: objectTagSet.SortedList()}
		_, err = actuator.osClient.ReplaceAllAttributesTags(ctx, "ports", osResource.ID, &opts)
	}
	return nil, orcObject, osResource, err
}

type portActuatorFactory struct{}

var _ generic.ActuatorFactory[orcObjectPT, osResourcePT] = portActuatorFactory{}

func (portActuatorFactory) NewCreateActuator(ctx context.Context, orcObject orcObjectPT, controller generic.ResourceController) ([]generic.WaitingOnEvent, generic.CreateResourceActuator[osResourcePT], error) {
	waitEvents, actuator, err := newCreateActuator(ctx, orcObject, controller)
	return waitEvents, actuator, err
}

func (portActuatorFactory) NewDeleteActuator(ctx context.Context, orcObject orcObjectPT, controller generic.ResourceController) ([]generic.WaitingOnEvent, generic.DeleteResourceActuator[osResourcePT], error) {
	actuator, err := newActuator(ctx, orcObject, controller)
	return nil, actuator, err
}

func newActuator(ctx context.Context, orcObject *orcv1alpha1.Port, controller generic.ResourceController) (portActuator, error) {
	log := ctrl.LoggerFrom(ctx)

	clientScope, err := controller.GetScopeFactory().NewClientScopeFromObject(ctx, controller.GetK8sClient(), log, orcObject)
	if err != nil {
		return portActuator{}, err
	}
	osClient, err := clientScope.NewNetworkClient()
	if err != nil {
		return portActuator{}, err
	}

	return portActuator{
		obj:        orcObject,
		osClient:   osClient,
		controller: controller,
	}, nil
}

func newCreateActuator(ctx context.Context, orcObject *orcv1alpha1.Port, controller generic.ResourceController) ([]generic.WaitingOnEvent, *portCreateActuator, error) {
	k8sClient := controller.GetK8sClient()

	orcNetwork := &orcv1alpha1.Network{}
	networkRef := string(orcObject.Spec.NetworkRef)
	if err := k8sClient.Get(ctx, client.ObjectKey{Name: networkRef, Namespace: orcObject.Namespace}, orcNetwork); err != nil {
		if apierrors.IsNotFound(err) {
			return []generic.WaitingOnEvent{generic.WaitingOnORCExist("network", networkRef)}, nil, nil
		}
		return nil, nil, err
	}

	if !orcv1alpha1.IsAvailable(orcNetwork) || orcNetwork.Status.ID == nil {
		return []generic.WaitingOnEvent{generic.WaitingOnORCReady("network", networkRef)}, nil, nil
	}
	networkID := orcv1alpha1.UUID(*orcNetwork.Status.ID)

	portActuator, err := newActuator(ctx, orcObject, controller)
	if err != nil {
		return nil, nil, err
	}

	return nil, &portCreateActuator{
		portActuator: portActuator,
		networkID:    networkID,
	}, nil
}
