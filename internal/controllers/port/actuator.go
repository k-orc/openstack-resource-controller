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
	"iter"

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

type (
	osResourceT = ports.Port

	createResourceActuator    = generic.CreateResourceActuator[orcObjectPT, orcObjectT, filterT, osResourceT]
	deleteResourceActuator    = generic.DeleteResourceActuator[orcObjectPT, orcObjectT, osResourceT]
	reconcileResourceActuator = generic.ReconcileResourceActuator[orcObjectPT, osResourceT]
	resourceReconciler        = generic.ResourceReconciler[orcObjectPT, osResourceT]
	helperFactory             = generic.ResourceHelperFactory[orcObjectPT, orcObjectT, resourceSpecT, filterT, osResourceT]
	portIterator              = iter.Seq2[*osResourceT, error]
)

type portActuator struct {
	osClient osclients.NetworkClient
}

type portCreateActuator struct {
	portActuator
	k8sClient client.Client
	networkID orcv1alpha1.UUID
}

var _ createResourceActuator = portCreateActuator{}
var _ deleteResourceActuator = portActuator{}

func (portActuator) GetResourceID(osResource *ports.Port) string {
	return osResource.ID
}

func (actuator portActuator) GetOSResourceByID(ctx context.Context, id string) (*ports.Port, error) {
	return actuator.osClient.GetPort(ctx, id)
}

func (actuator portActuator) ListOSResourcesForAdoption(ctx context.Context, obj *orcv1alpha1.Port) (portIterator, bool) {
	if obj.Spec.Resource == nil {
		return nil, false
	}

	listOpts := ports.ListOpts{Name: string(getResourceName(obj))}
	return actuator.osClient.ListPort(ctx, listOpts), true
}

func (actuator portCreateActuator) ListOSResourcesForImport(ctx context.Context, filter orcv1alpha1.PortFilter) portIterator {
	listOpts := ports.ListOpts{
		Name:        string(ptr.Deref(filter.Name, "")),
		Description: string(ptr.Deref(filter.Description, "")),
		NetworkID:   string(actuator.networkID),
		Tags:        neutrontags.Join(filter.FilterByNeutronTags.Tags),
		TagsAny:     neutrontags.Join(filter.FilterByNeutronTags.TagsAny),
		NotTags:     neutrontags.Join(filter.FilterByNeutronTags.NotTags),
		NotTagsAny:  neutrontags.Join(filter.FilterByNeutronTags.NotTagsAny),
	}

	return actuator.osClient.ListPort(ctx, listOpts)
}

func (actuator portCreateActuator) CreateResource(ctx context.Context, obj *orcv1alpha1.Port) ([]generic.ProgressStatus, *ports.Port, error) {
	resource := obj.Spec.Resource

	if resource == nil {
		// Should have been caught by API validation
		return nil, nil, orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Creation requested, but spec.resource is not set")
	}

	createOpts := ports.CreateOpts{
		NetworkID:   string(actuator.networkID),
		Name:        string(getResourceName(obj)),
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

	var waitEvents []generic.ProgressStatus

	// We explicitly disable creation of IP addresses by passing an empty
	// value whenever the user does not specify addresses
	fixedIPs := make([]ports.IP, len(resource.Addresses))
	for i := range resource.Addresses {
		subnet := &orcv1alpha1.Subnet{}
		key := client.ObjectKey{Name: string(resource.Addresses[i].SubnetRef), Namespace: obj.Namespace}
		if err := actuator.k8sClient.Get(ctx, key, subnet); err != nil {
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
		key := client.ObjectKey{Name: string(resource.SecurityGroupRefs[i]), Namespace: obj.Namespace}
		if err := actuator.k8sClient.Get(ctx, key, securityGroup); err != nil {
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

func (actuator portActuator) DeleteResource(ctx context.Context, _ *orcv1alpha1.Port, flavor *ports.Port) ([]generic.ProgressStatus, error) {
	return nil, actuator.osClient.DeletePort(ctx, flavor.ID)
}

var _ reconcileResourceActuator = portActuator{}

func (actuator portActuator) GetResourceReconcilers(ctx context.Context, orcObject orcObjectPT, osResource *osResourceT, controller generic.ResourceController) ([]resourceReconciler, error) {
	return []resourceReconciler{
		actuator.updateTags,
	}, nil
}

func (actuator portActuator) updateTags(ctx context.Context, orcObject orcObjectPT, osResource *osResourceT) ([]generic.ProgressStatus, error) {
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
	return nil, err
}

type portHelperFactory struct{}

var _ helperFactory = portHelperFactory{}

func (portHelperFactory) NewAPIObjectAdapter(obj orcObjectPT) adapterI {
	return portAdapter{obj}
}

func (portHelperFactory) NewCreateActuator(ctx context.Context, orcObject orcObjectPT, controller generic.ResourceController) ([]generic.ProgressStatus, createResourceActuator, error) {
	waitEvents, actuator, err := newCreateActuator(ctx, orcObject, controller)
	return waitEvents, actuator, err
}

func (portHelperFactory) NewDeleteActuator(ctx context.Context, orcObject orcObjectPT, controller generic.ResourceController) ([]generic.ProgressStatus, deleteResourceActuator, error) {
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
		osClient: osClient,
	}, nil
}

func newCreateActuator(ctx context.Context, orcObject *orcv1alpha1.Port, controller generic.ResourceController) ([]generic.ProgressStatus, *portCreateActuator, error) {
	k8sClient := controller.GetK8sClient()

	orcNetwork := &orcv1alpha1.Network{}
	networkRef := string(orcObject.Spec.NetworkRef)
	if err := k8sClient.Get(ctx, client.ObjectKey{Name: networkRef, Namespace: orcObject.Namespace}, orcNetwork); err != nil {
		if apierrors.IsNotFound(err) {
			return []generic.ProgressStatus{generic.WaitingOnORCExist("network", networkRef)}, nil, nil
		}
		return nil, nil, err
	}

	if !orcv1alpha1.IsAvailable(orcNetwork) || orcNetwork.Status.ID == nil {
		return []generic.ProgressStatus{generic.WaitingOnORCReady("network", networkRef)}, nil, nil
	}
	networkID := orcv1alpha1.UUID(*orcNetwork.Status.ID)

	portActuator, err := newActuator(ctx, orcObject, controller)
	if err != nil {
		return nil, nil, err
	}

	return nil, &portCreateActuator{
		portActuator: portActuator,
		k8sClient:    controller.GetK8sClient(),
		networkID:    networkID,
	}, nil
}
