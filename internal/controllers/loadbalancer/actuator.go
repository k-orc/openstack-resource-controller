/*
Copyright 2025 The ORC Authors.

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

package loadbalancer

import (
	"context"
	"iter"
	"time"

	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/loadbalancers"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/progress"
	osclients "github.com/k-orc/openstack-resource-controller/v2/internal/osclients"
	"github.com/k-orc/openstack-resource-controller/v2/internal/util/dependency"
	orcerrors "github.com/k-orc/openstack-resource-controller/v2/internal/util/errors"
)

const controllerName = "loadbalancer"

// Type aliases for the generic controller framework.
type (
	osResourceT = loadbalancers.LoadBalancer

	createResourceActuator    = interfaces.CreateResourceActuator[orcObjectPT, orcObjectT, filterT, osResourceT]
	deleteResourceActuator    = interfaces.DeleteResourceActuator[orcObjectPT, orcObjectT, osResourceT]
	reconcileResourceActuator = interfaces.ReconcileResourceActuator[orcObjectPT, osResourceT]
	resourceReconciler        = interfaces.ResourceReconciler[orcObjectPT, osResourceT]
	helperFactory             = interfaces.ResourceHelperFactory[orcObjectPT, orcObjectT, resourceSpecT, filterT, osResourceT]
)

const (
	// lbDeletingPollingPeriod is the frequency to poll when waiting for a
	// load balancer to finish deleting.
	lbDeletingPollingPeriod = 15 * time.Second
)

// Dependency declarations for use in CreateResource (with finalizer management)
// and in controller.go (for watchers and index registration).
var (
	subnetDependency = dependency.NewDeletionGuardDependency[*orcv1alpha1.LoadBalancerList, *orcv1alpha1.Subnet](
		"spec.resource.subnetRef",
		func(lb *orcv1alpha1.LoadBalancer) []string {
			if lb.Spec.Resource == nil || lb.Spec.Resource.SubnetRef == nil {
				return nil
			}
			return []string{string(*lb.Spec.Resource.SubnetRef)}
		},
		finalizer, externalObjectFieldOwner,
	)

	networkDependency = dependency.NewDeletionGuardDependency[*orcv1alpha1.LoadBalancerList, *orcv1alpha1.Network](
		"spec.resource.networkRef",
		func(lb *orcv1alpha1.LoadBalancer) []string {
			if lb.Spec.Resource == nil || lb.Spec.Resource.NetworkRef == nil {
				return nil
			}
			return []string{string(*lb.Spec.Resource.NetworkRef)}
		},
		finalizer, externalObjectFieldOwner,
	)

	portDependency = dependency.NewDeletionGuardDependency[*orcv1alpha1.LoadBalancerList, *orcv1alpha1.Port](
		"spec.resource.vipPortRef",
		func(lb *orcv1alpha1.LoadBalancer) []string {
			if lb.Spec.Resource == nil || lb.Spec.Resource.VIPPortRef == nil {
				return nil
			}
			return []string{string(*lb.Spec.Resource.VIPPortRef)}
		},
		finalizer, externalObjectFieldOwner,
	)

	projectDependency = dependency.NewDeletionGuardDependency[*orcv1alpha1.LoadBalancerList, *orcv1alpha1.Project](
		"spec.resource.projectRef",
		func(lb *orcv1alpha1.LoadBalancer) []string {
			if lb.Spec.Resource == nil || lb.Spec.Resource.ProjectRef == nil {
				return nil
			}
			return []string{string(*lb.Spec.Resource.ProjectRef)}
		},
		finalizer, externalObjectFieldOwner,
	)

	// Import dependencies (no finalizer management - just for watching).
	subnetImportDependency = dependency.NewDependency[*orcv1alpha1.LoadBalancerList, *orcv1alpha1.Subnet](
		"spec.import.filter.vipSubnetRef",
		func(lb *orcv1alpha1.LoadBalancer) []string {
			if lb.Spec.Import == nil || lb.Spec.Import.Filter == nil || lb.Spec.Import.Filter.VIPSubnetRef == nil {
				return nil
			}
			return []string{string(*lb.Spec.Import.Filter.VIPSubnetRef)}
		},
	)

	networkImportDependency = dependency.NewDependency[*orcv1alpha1.LoadBalancerList, *orcv1alpha1.Network](
		"spec.import.filter.vipNetworkRef",
		func(lb *orcv1alpha1.LoadBalancer) []string {
			if lb.Spec.Import == nil || lb.Spec.Import.Filter == nil || lb.Spec.Import.Filter.VIPNetworkRef == nil {
				return nil
			}
			return []string{string(*lb.Spec.Import.Filter.VIPNetworkRef)}
		},
	)

	projectImportDependency = dependency.NewDependency[*orcv1alpha1.LoadBalancerList, *orcv1alpha1.Project](
		"spec.import.filter.projectRef",
		func(lb *orcv1alpha1.LoadBalancer) []string {
			if lb.Spec.Import == nil || lb.Spec.Import.Filter == nil || lb.Spec.Import.Filter.ProjectRef == nil {
				return nil
			}
			return []string{string(*lb.Spec.Import.Filter.ProjectRef)}
		},
	)
)

type loadbalancerActuator struct {
	osClient  osclients.LoadBalancerClient
	k8sClient client.Client
}

var _ createResourceActuator = loadbalancerActuator{}
var _ deleteResourceActuator = loadbalancerActuator{}

func (loadbalancerActuator) GetResourceID(osResource *osResourceT) string {
	return osResource.ID
}

func (actuator loadbalancerActuator) GetOSResourceByID(ctx context.Context, id string) (*osResourceT, progress.ReconcileStatus) {
	lb, err := actuator.osClient.GetLoadBalancer(ctx, id)
	if err != nil {
		return nil, progress.WrapError(err)
	}
	return lb, nil
}

func (actuator loadbalancerActuator) ListOSResourcesForAdoption(ctx context.Context, obj orcObjectPT) (iter.Seq2[*osResourceT, error], bool) {
	resource := obj.Spec.Resource
	if resource == nil {
		return nil, false
	}

	// Resolve the project ID from ProjectRef if set. Without the project ID,
	// adoption with admin-scoped credentials could match a load balancer in
	// the wrong project.
	var projectID string
	if resource.ProjectRef != nil {
		project, rs := dependency.FetchDependency(
			ctx, actuator.k8sClient, obj.Namespace, resource.ProjectRef, "Project",
			func(dep *orcv1alpha1.Project) bool {
				return orcv1alpha1.IsAvailable(dep) && dep.Status.ID != nil
			},
		)
		if needsReschedule, _ := rs.NeedsReschedule(); needsReschedule {
			return nil, false
		}
		projectID = ptr.Deref(project.Status.ID, "")
	}

	listOpts := loadbalancers.ListOpts{
		Name:      getResourceName(obj),
		ProjectID: projectID,
	}
	return actuator.osClient.ListLoadBalancer(ctx, listOpts), true
}

func (actuator loadbalancerActuator) ListOSResourcesForImport(ctx context.Context, obj orcObjectPT, filter filterT) (iter.Seq2[*osResourceT, error], progress.ReconcileStatus) {
	var reconcileStatus progress.ReconcileStatus

	var vipSubnetID string
	var vipNetworkID string

	if filter.VIPSubnetRef != nil {
		subnet, rs := dependency.FetchDependency[*orcv1alpha1.Subnet](
			ctx, actuator.k8sClient, obj.Namespace, filter.VIPSubnetRef, "Subnet",
			orcv1alpha1.IsAvailable,
		)
		reconcileStatus = reconcileStatus.WithReconcileStatus(rs)
		if subnet != nil {
			vipSubnetID = ptr.Deref(subnet.Status.ID, "")
		}
	}

	if filter.VIPNetworkRef != nil {
		network, rs := dependency.FetchDependency[*orcv1alpha1.Network](
			ctx, actuator.k8sClient, obj.Namespace, filter.VIPNetworkRef, "Network",
			orcv1alpha1.IsAvailable,
		)
		reconcileStatus = reconcileStatus.WithReconcileStatus(rs)
		if network != nil {
			vipNetworkID = ptr.Deref(network.Status.ID, "")
		}
	}

	project, rs := dependency.FetchDependency[*orcv1alpha1.Project](
		ctx, actuator.k8sClient, obj.Namespace, filter.ProjectRef, "Project",
		orcv1alpha1.IsAvailable,
	)
	reconcileStatus = reconcileStatus.WithReconcileStatus(rs)

	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return nil, reconcileStatus
	}

	listOpts := loadbalancers.ListOpts{
		Name:         string(ptr.Deref(filter.Name, "")),
		Description:  string(ptr.Deref(filter.Description, "")),
		VipSubnetID:  vipSubnetID,
		VipNetworkID: vipNetworkID,
		ProjectID:    ptr.Deref(project.Status.ID, ""),
		Tags:         neutronTagsToStrings(filter.Tags),
		TagsAny:      neutronTagsToStrings(filter.TagsAny),
		TagsNot:      neutronTagsToStrings(filter.NotTags),
		TagsNotAny:   neutronTagsToStrings(filter.NotTagsAny),
	}

	return actuator.osClient.ListLoadBalancer(ctx, listOpts), nil
}

// neutronTagsToStrings converts a slice of NeutronTag to a plain []string.
func neutronTagsToStrings(neutronTags []orcv1alpha1.NeutronTag) []string {
	if len(neutronTags) == 0 {
		return nil
	}
	result := make([]string, len(neutronTags))
	for i := range neutronTags {
		result[i] = string(neutronTags[i])
	}
	return result
}

func (actuator loadbalancerActuator) CreateResource(ctx context.Context, obj orcObjectPT) (*osResourceT, progress.ReconcileStatus) {
	resource := obj.Spec.Resource
	if resource == nil {
		// Should have been caught by API validation
		return nil, progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Creation requested, but spec.resource is not set"))
	}

	var reconcileStatus progress.ReconcileStatus

	// Resolve exactly one of SubnetRef, NetworkRef, or VIPPortRef.
	var vipSubnetID string
	if resource.SubnetRef != nil {
		subnet, subnetDepRS := subnetDependency.GetDependency(
			ctx, actuator.k8sClient, obj, orcv1alpha1.IsAvailable,
		)
		reconcileStatus = reconcileStatus.WithReconcileStatus(subnetDepRS)
		if subnet != nil {
			vipSubnetID = ptr.Deref(subnet.Status.ID, "")
		}
	}

	var vipNetworkID string
	if resource.NetworkRef != nil {
		network, networkDepRS := networkDependency.GetDependency(
			ctx, actuator.k8sClient, obj, orcv1alpha1.IsAvailable,
		)
		reconcileStatus = reconcileStatus.WithReconcileStatus(networkDepRS)
		if network != nil {
			vipNetworkID = ptr.Deref(network.Status.ID, "")
		}
	}

	var vipPortID string
	if resource.VIPPortRef != nil {
		port, portDepRS := portDependency.GetDependency(
			ctx, actuator.k8sClient, obj, orcv1alpha1.IsAvailable,
		)
		reconcileStatus = reconcileStatus.WithReconcileStatus(portDepRS)
		if port != nil {
			vipPortID = ptr.Deref(port.Status.ID, "")
		}
	}

	var projectID string
	if resource.ProjectRef != nil {
		project, projectDepRS := projectDependency.GetDependency(
			ctx, actuator.k8sClient, obj, orcv1alpha1.IsAvailable,
		)
		reconcileStatus = reconcileStatus.WithReconcileStatus(projectDepRS)
		if project != nil {
			projectID = ptr.Deref(project.Status.ID, "")
		}
	}

	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return nil, reconcileStatus
	}

	createOpts := loadbalancers.CreateOpts{
		Name:         getResourceName(obj),
		Description:  string(ptr.Deref(resource.Description, "")),
		VipSubnetID:  vipSubnetID,
		VipNetworkID: vipNetworkID,
		VipPortID:    vipPortID,
		ProjectID:    projectID,
		AdminStateUp: resource.AdminStateUp,
		Provider:     ptr.Deref(resource.Provider, ""),
	}

	if resource.VIPAddress != nil {
		createOpts.VipAddress = string(*resource.VIPAddress)
	}

	if len(resource.Tags) > 0 {
		createOpts.Tags = make([]string, len(resource.Tags))
		for i := range resource.Tags {
			createOpts.Tags[i] = string(resource.Tags[i])
		}
	}

	osResource, err := actuator.osClient.CreateLoadBalancer(ctx, &createOpts)
	if err != nil {
		// We should require the spec to be updated before retrying a create which returned a conflict
		if orcerrors.IsConflict(err) {
			err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration creating resource: "+err.Error(), err)
		}
		return nil, progress.WrapError(err)
	}

	return osResource, nil
}

func (actuator loadbalancerActuator) DeleteResource(ctx context.Context, _ orcObjectPT, osResource *osResourceT) progress.ReconcileStatus {
	// If the load balancer is in a PENDING_* state, we cannot delete it yet.
	// Wait for it to stabilize before attempting deletion.
	switch osResource.ProvisioningStatus {
	case ProvisioningStatusPendingCreate, ProvisioningStatusPendingUpdate:
		return progress.WaitingOnOpenStack(progress.WaitingOnReady, lbDeletingPollingPeriod)
	case ProvisioningStatusPendingDelete:
		// Already being deleted, wait for it to complete.
		return progress.WaitingOnOpenStack(progress.WaitingOnDeletion, lbDeletingPollingPeriod)
	}

	return progress.WrapError(actuator.osClient.DeleteLoadBalancer(ctx, osResource.ID, nil))
}

type loadbalancerHelperFactory struct{}

var _ helperFactory = loadbalancerHelperFactory{}

func (loadbalancerHelperFactory) NewAPIObjectAdapter(obj orcObjectPT) adapterI {
	return loadbalancerAdapter{obj}
}

func (loadbalancerHelperFactory) NewCreateActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (createResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, orcObject, controller)
}

func (loadbalancerHelperFactory) NewDeleteActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (deleteResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, orcObject, controller)
}

func newActuator(ctx context.Context, orcObject *orcv1alpha1.LoadBalancer, controller interfaces.ResourceController) (loadbalancerActuator, progress.ReconcileStatus) {
	log := ctrl.LoggerFrom(ctx)

	// Ensure credential secrets exist and have our finalizer
	_, reconcileStatus := credentialsDependency.GetDependencies(ctx, controller.GetK8sClient(), orcObject, func(*corev1.Secret) bool { return true })
	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return loadbalancerActuator{}, reconcileStatus
	}

	clientScope, err := controller.GetScopeFactory().NewClientScopeFromObject(ctx, controller.GetK8sClient(), log, orcObject)
	if err != nil {
		return loadbalancerActuator{}, progress.WrapError(err)
	}
	osClient, err := clientScope.NewLoadBalancerClient()
	if err != nil {
		return loadbalancerActuator{}, progress.WrapError(err)
	}

	return loadbalancerActuator{
		osClient:  osClient,
		k8sClient: controller.GetK8sClient(),
	}, nil
}
