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
	"iter"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/subnets"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/generic/progress"
	"github.com/k-orc/openstack-resource-controller/internal/osclients"
	orcerrors "github.com/k-orc/openstack-resource-controller/internal/util/errors"
	"github.com/k-orc/openstack-resource-controller/internal/util/neutrontags"
)

// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=subnets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=subnets/status,verbs=get;update;patch

// OpenStack resource types
type (
	osResourceT = subnets.Subnet

	createResourceActuator    = interfaces.CreateResourceActuator[orcObjectPT, orcObjectT, filterT, osResourceT]
	deleteResourceActuator    = interfaces.DeleteResourceActuator[orcObjectPT, orcObjectT, osResourceT]
	reconcileResourceActuator = interfaces.ReconcileResourceActuator[orcObjectPT, osResourceT]
	resourceReconciler        = interfaces.ResourceReconciler[orcObjectPT, osResourceT]
	helperFactory             = interfaces.ResourceHelperFactory[orcObjectPT, orcObjectT, resourceSpecT, filterT, osResourceT]
)

type subnetActuator struct {
	osClient  osclients.NetworkClient
	k8sClient client.Client
}

var _ createResourceActuator = subnetActuator{}
var _ deleteResourceActuator = subnetActuator{}

func (subnetActuator) GetResourceID(osResource *subnets.Subnet) string {
	return osResource.ID
}

func (actuator subnetActuator) GetOSResourceByID(ctx context.Context, id string) (*subnets.Subnet, error) {
	return actuator.osClient.GetSubnet(ctx, id)
}

func (actuator subnetActuator) ListOSResourcesForAdoption(ctx context.Context, obj orcObjectPT) (iter.Seq2[*osResourceT, error], bool) {
	if obj.Spec.Resource == nil {
		return nil, false
	}
	listOpts := subnets.ListOpts{Name: string(getResourceName(obj))}
	return actuator.osClient.ListSubnet(ctx, listOpts), true
}

func (actuator subnetActuator) ListOSResourcesForImport(ctx context.Context, obj orcObjectPT, filter filterT) ([]progress.ProgressStatus, iter.Seq2[*osResourceT, error], error) {
	var networkID string
	var progressStatus []progress.ProgressStatus

	if filter.NetworkRef != "" {
		network := &orcv1alpha1.Network{}
		{
			networkKey := client.ObjectKey{Name: string(filter.NetworkRef), Namespace: obj.Namespace}
			if err := actuator.k8sClient.Get(ctx, networkKey, network); err != nil {
				if apierrors.IsNotFound(err) {
					progressStatus = append(progressStatus, progress.WaitingOnORCExist("Network", networkKey.Name))
				} else {
					return nil, nil, fmt.Errorf("fetching network %s: %w", networkKey.Name, err)
				}
			} else {
				if !orcv1alpha1.IsAvailable(network) || network.Status.ID == nil {
					progressStatus = append(progressStatus, progress.WaitingOnORCReady("Network", networkKey.Name))
				}
			}
			if len(progressStatus) > 0 {
				return progressStatus, nil, nil
			}
			networkID = *network.Status.ID
		}
	}

	listOpts := subnets.ListOpts{
		Name:        string(ptr.Deref(filter.Name, "")),
		Description: string(ptr.Deref(filter.Description, "")),
		NetworkID:   networkID,
		IPVersion:   int(ptr.Deref(filter.IPVersion, 0)),
		GatewayIP:   string(ptr.Deref(filter.GatewayIP, "")),
		CIDR:        string(ptr.Deref(filter.CIDR, "")),
		Tags:        neutrontags.Join(filter.Tags),
		TagsAny:     neutrontags.Join(filter.TagsAny),
		NotTags:     neutrontags.Join(filter.NotTags),
		NotTagsAny:  neutrontags.Join(filter.NotTagsAny),
	}
	if filter.IPv6 != nil {
		listOpts.IPv6AddressMode = string(ptr.Deref(filter.IPv6.AddressMode, ""))
		listOpts.IPv6RAMode = string(ptr.Deref(filter.IPv6.RAMode, ""))
	}

	return nil, actuator.osClient.ListSubnet(ctx, listOpts), nil
}

func (actuator subnetActuator) CreateResource(ctx context.Context, obj orcObjectPT) ([]progress.ProgressStatus, *subnets.Subnet, error) {
	resource := obj.Spec.Resource
	if resource == nil {
		// Should have been caught by API validation
		return nil, nil, orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Creation requested, but spec.resource is not set")
	}

	network, progressStatus, err := networkDependency.GetDependency(
		ctx, actuator.k8sClient, obj, func(dep *orcv1alpha1.Network) bool {
			return orcv1alpha1.IsAvailable(dep) && dep.Status.ID != nil
		},
	)

	if len(progressStatus) != 0 || err != nil {
		return progressStatus, nil, err
	}

	createOpts := subnets.CreateOpts{
		NetworkID:         *network.Status.ID,
		CIDR:              string(resource.CIDR),
		Name:              string(getResourceName(obj)),
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

func (actuator subnetActuator) DeleteResource(ctx context.Context, obj orcObjectPT, osResource *subnets.Subnet) ([]progress.ProgressStatus, error) {
	// Delete any RouterInterface first, as this would prevent deletion of the subnet
	routerInterface, err := getRouterInterface(ctx, actuator.k8sClient, obj)
	if err != nil {
		return nil, err
	}

	if routerInterface != nil {
		// We will be reconciled again when it's gone
		if routerInterface.GetDeletionTimestamp().IsZero() {
			if err := actuator.k8sClient.Delete(ctx, routerInterface); err != nil {
				return nil, err
			}
		}
		return []progress.ProgressStatus{progress.WaitingOnORCDeleted("RouterInterface", routerInterface.GetName())}, nil
	}

	return nil, actuator.osClient.DeleteSubnet(ctx, osResource.ID)
}

var _ reconcileResourceActuator = subnetActuator{}

func (actuator subnetActuator) GetResourceReconcilers(ctx context.Context, orcObject orcObjectPT, osResource *osResourceT, controller interfaces.ResourceController) (reconcilers []resourceReconciler, err error) {
	return []resourceReconciler{
		neutrontags.ReconcileTags[orcObjectPT, osResourceT](actuator.osClient, "subnets", osResource.ID, orcObject.Spec.Resource.Tags, osResource.Tags),
		actuator.ensureRouterInterface,
	}, nil
}

func (actuator subnetActuator) ensureRouterInterface(ctx context.Context, orcObject orcObjectPT, osResource *osResourceT) ([]progress.ProgressStatus, error) {
	var waitEvents []progress.ProgressStatus
	var err error

	routerInterface, err := getRouterInterface(ctx, actuator.k8sClient, orcObject)
	if routerInterfaceMatchesSpec(routerInterface, orcObject.Name, orcObject.Spec.Resource) {
		// Nothing to do
		return waitEvents, err
	}

	// If it doesn't match we should delete any existing interface
	if routerInterface != nil {
		if routerInterface.GetDeletionTimestamp().IsZero() {
			if err := actuator.k8sClient.Delete(ctx, routerInterface); err != nil {
				return waitEvents, fmt.Errorf("deleting RouterInterface %s: %w", client.ObjectKeyFromObject(routerInterface), err)
			}
		}
		waitEvents = append(waitEvents, progress.WaitingOnORCDeleted("routerinterface", routerInterface.Name))
		return waitEvents, err
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

	if err := actuator.k8sClient.Create(ctx, routerInterface); err != nil {
		return waitEvents, fmt.Errorf("creating RouterInterface %s: %w", client.ObjectKeyFromObject(orcObject), err)
	}
	waitEvents = append(waitEvents, progress.WaitingOnORCReady("routerinterface", routerInterface.Name))

	return waitEvents, err
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

type subnetHelperFactory struct{}

var _ helperFactory = subnetHelperFactory{}

func (subnetHelperFactory) NewAPIObjectAdapter(obj orcObjectPT) adapterI {
	return subnetAdapter{obj}
}

func (subnetHelperFactory) NewCreateActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) ([]progress.ProgressStatus, createResourceActuator, error) {
	actuator, progressStatus, err := newActuator(ctx, controller, orcObject)
	return progressStatus, actuator, err
}

func (subnetHelperFactory) NewDeleteActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) ([]progress.ProgressStatus, deleteResourceActuator, error) {
	actuator, progressStatus, err := newActuator(ctx, controller, orcObject)
	return progressStatus, actuator, err
}

func newActuator(ctx context.Context, controller interfaces.ResourceController, orcObject *orcv1alpha1.Subnet) (subnetActuator, []progress.ProgressStatus, error) {
	if orcObject == nil {
		return subnetActuator{}, nil, fmt.Errorf("orcObject may not be nil")
	}

	// Ensure credential secrets exist and have our finalizer
	_, progressStatus, err := credentialsDependency.GetDependencies(ctx, controller.GetK8sClient(), orcObject, func(*corev1.Secret) bool { return true })
	if len(progressStatus) > 0 || err != nil {
		return subnetActuator{}, progressStatus, err
	}

	log := ctrl.LoggerFrom(ctx)
	clientScope, err := controller.GetScopeFactory().NewClientScopeFromObject(ctx, controller.GetK8sClient(), log, orcObject)
	if err != nil {
		return subnetActuator{}, nil, err
	}
	osClient, err := clientScope.NewNetworkClient()
	if err != nil {
		return subnetActuator{}, nil, err
	}

	return subnetActuator{
		osClient:  osClient,
		k8sClient: controller.GetK8sClient(),
	}, nil, nil
}
