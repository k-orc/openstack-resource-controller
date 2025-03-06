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
	"errors"
	"fmt"
	"iter"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/generic/progress"
	osclients "github.com/k-orc/openstack-resource-controller/internal/osclients"
	orcerrors "github.com/k-orc/openstack-resource-controller/internal/util/errors"
	"github.com/k-orc/openstack-resource-controller/internal/util/neutrontags"
)

type (
	osResourceT = ports.Port

	createResourceActuator    = interfaces.CreateResourceActuator[orcObjectPT, orcObjectT, filterT, osResourceT]
	deleteResourceActuator    = interfaces.DeleteResourceActuator[orcObjectPT, orcObjectT, osResourceT]
	reconcileResourceActuator = interfaces.ReconcileResourceActuator[orcObjectPT, osResourceT]
	resourceReconciler        = interfaces.ResourceReconciler[orcObjectPT, osResourceT]
	helperFactory             = interfaces.ResourceHelperFactory[orcObjectPT, orcObjectT, resourceSpecT, filterT, osResourceT]
	portIterator              = iter.Seq2[*osResourceT, error]
)

type portActuator struct {
	osClient osclients.NetworkClient
}

type portCreateActuator struct {
	portActuator
	k8sClient client.Client
	networkID string
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
		NetworkID:   actuator.networkID,
		Tags:        neutrontags.Join(filter.FilterByNeutronTags.Tags),
		TagsAny:     neutrontags.Join(filter.FilterByNeutronTags.TagsAny),
		NotTags:     neutrontags.Join(filter.FilterByNeutronTags.NotTags),
		NotTagsAny:  neutrontags.Join(filter.FilterByNeutronTags.NotTagsAny),
	}

	return actuator.osClient.ListPort(ctx, listOpts)
}

func (actuator portCreateActuator) CreateResource(ctx context.Context, obj *orcv1alpha1.Port) ([]progress.ProgressStatus, *ports.Port, error) {
	resource := obj.Spec.Resource
	if resource == nil {
		// Should have been caught by API validation
		return nil, nil, orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Creation requested, but spec.resource is not set")
	}

	var progressStatus []progress.ProgressStatus

	// Fetch all dependencies and ensure they have our finalizer
	subnetMap, subnetProgress, subnetErr := subnetDependency.GetDependencies(
		ctx, actuator.k8sClient, obj, func(dep *orcv1alpha1.Subnet) bool {
			return orcv1alpha1.IsAvailable(dep) && dep.Status.ID != nil
		},
	)
	secGroupMap, secGroupProgress, secGroupErr := securityGroupDependency.GetDependencies(
		ctx, actuator.k8sClient, obj, func(dep *orcv1alpha1.SecurityGroup) bool {
			return dep.Status.ID != nil
		},
	)

	progressStatus = append(progressStatus, subnetProgress...)
	progressStatus = append(progressStatus, secGroupProgress...)
	err := errors.Join(subnetErr, secGroupErr)

	if len(progressStatus) != 0 || err != nil {
		return progressStatus, nil, err
	}

	createOpts := ports.CreateOpts{
		NetworkID:   actuator.networkID,
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

	// We explicitly disable creation of IP addresses by passing an empty
	// value whenever the user does not specify addresses
	fixedIPs := make([]ports.IP, len(resource.Addresses))
	for i := range resource.Addresses {
		subnetName := string(resource.Addresses[i].SubnetRef)
		subnet, ok := subnetMap[subnetName]
		if !ok {
			// Programming error
			return nil, nil, fmt.Errorf("subnet %s was not returned by GetDependencies", subnetName)
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
		secGroupName := string(resource.SecurityGroupRefs[i])
		secGroup, ok := secGroupMap[secGroupName]
		if !ok {
			// Programming error
			return nil, nil, fmt.Errorf("security group %s was not returned by GetDependencies", secGroupName)
		}
		securityGroups[i] = *secGroup.Status.ID
	}
	createOpts.SecurityGroups = &securityGroups

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

func (actuator portActuator) DeleteResource(ctx context.Context, _ *orcv1alpha1.Port, flavor *ports.Port) ([]progress.ProgressStatus, error) {
	return nil, actuator.osClient.DeletePort(ctx, flavor.ID)
}

var _ reconcileResourceActuator = portActuator{}

func (actuator portActuator) GetResourceReconcilers(ctx context.Context, orcObject orcObjectPT, osResource *osResourceT, controller interfaces.ResourceController) ([]resourceReconciler, error) {
	return []resourceReconciler{
		neutrontags.ReconcileTags[orcObjectPT, osResourceT](actuator.osClient, "ports", osResource.ID, orcObject.Spec.Resource.Tags, osResource.Tags),
	}, nil
}

type portHelperFactory struct{}

var _ helperFactory = portHelperFactory{}

func (portHelperFactory) NewAPIObjectAdapter(obj orcObjectPT) adapterI {
	return portAdapter{obj}
}

func (portHelperFactory) NewCreateActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) ([]progress.ProgressStatus, createResourceActuator, error) {
	actuator, progressStatus, err := newCreateActuator(ctx, orcObject, controller)
	return progressStatus, actuator, err
}

func (portHelperFactory) NewDeleteActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) ([]progress.ProgressStatus, deleteResourceActuator, error) {
	actuator, err := newActuator(ctx, orcObject, controller)
	return nil, actuator, err
}

func newActuator(ctx context.Context, orcObject *orcv1alpha1.Port, controller interfaces.ResourceController) (portActuator, error) {
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

func newCreateActuator(ctx context.Context, orcObject *orcv1alpha1.Port, controller interfaces.ResourceController) (*portCreateActuator, []progress.ProgressStatus, error) {
	// Ensure credential secrets exist and have our finalizer
	_, progressStatus, err := credentialsDependency.GetDependencies(ctx, controller.GetK8sClient(), orcObject, func(*corev1.Secret) bool { return true })
	if len(progressStatus) > 0 || err != nil {
		return nil, progressStatus, err
	}

	orcNetwork, progressStatus, err := networkDependency.GetDependency(
		ctx, controller.GetK8sClient(), orcObject, func(dep *orcv1alpha1.Network) bool {
			return orcv1alpha1.IsAvailable(dep) && dep.Status.ID != nil
		},
	)
	if len(progressStatus) != 0 || err != nil {
		return nil, progressStatus, err
	}
	networkID := *orcNetwork.Status.ID

	portActuator, err := newActuator(ctx, orcObject, controller)
	if err != nil {
		return nil, nil, err
	}

	return &portCreateActuator{
		portActuator: portActuator,
		k8sClient:    controller.GetK8sClient(),
		networkID:    networkID,
	}, nil, nil
}
