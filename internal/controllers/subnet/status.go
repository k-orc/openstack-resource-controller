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
	"time"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/subnets"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/common"
	"github.com/k-orc/openstack-resource-controller/internal/util/applyconfigs"
	orcapplyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/pkg/clients/applyconfiguration/api/v1alpha1"
)

// setStatusID sets status.ID in its own SSA transaction.
func (r *orcSubnetReconciler) setStatusID(ctx context.Context, obj client.Object, id string) error {
	applyConfig := orcapplyconfigv1alpha1.Subnet(obj.GetName(), obj.GetNamespace()).
		WithUID(obj.GetUID()).
		WithStatus(orcapplyconfigv1alpha1.SubnetStatus().
			WithID(id))

	return r.client.Status().Patch(ctx, obj, applyconfigs.Patch(types.ApplyPatchType, applyConfig), client.ForceOwnership, ssaFieldOwner(SSAIDTxn))
}

type updateStatusOpts struct {
	resource        *subnets.Subnet
	routerInterface *orcv1alpha1.RouterInterface
	progressMessage *string
	err             error
}

type updateStatusOpt func(*updateStatusOpts)

func withResource(resource *subnets.Subnet) updateStatusOpt {
	return func(opts *updateStatusOpts) {
		opts.resource = resource
	}
}

func withRouterInterface(routerInterface *orcv1alpha1.RouterInterface) updateStatusOpt {
	return func(opts *updateStatusOpts) {
		opts.routerInterface = routerInterface
	}
}

func withError(err error) updateStatusOpt {
	return func(opts *updateStatusOpts) {
		opts.err = err
	}
}

// withProgressMessage sets a custom progressing message if and only if the reconcile is progressing.
func withProgressMessage(message string) updateStatusOpt {
	return func(opts *updateStatusOpts) {
		opts.progressMessage = &message
	}
}

func getOSResourceStatus(osResource *subnets.Subnet) *orcapplyconfigv1alpha1.SubnetResourceStatusApplyConfiguration {
	status := orcapplyconfigv1alpha1.SubnetResourceStatus().
		WithName(osResource.Name).
		WithDescription(osResource.Description).
		WithIPVersion(osResource.IPVersion).
		WithCIDR(osResource.CIDR).
		WithGatewayIP(osResource.GatewayIP).
		WithDNSPublishFixedIP(osResource.DNSPublishFixedIP).
		WithEnableDHCP(osResource.EnableDHCP).
		WithProjectID(osResource.ProjectID).
		WithRevisionNumber(int64(osResource.RevisionNumber)).
		WithIPv6AddressMode(osResource.IPv6AddressMode).
		WithIPv6RAMode(osResource.IPv6RAMode).
		WithTags(osResource.Tags...).
		WithDNSNameservers(osResource.DNSNameservers...)

	if len(osResource.AllocationPools) > 0 {
		allocationPools := make([]*orcapplyconfigv1alpha1.AllocationPoolStatusApplyConfiguration, len(osResource.AllocationPools))
		for i := range osResource.AllocationPools {
			allocationPools[i] = orcapplyconfigv1alpha1.AllocationPoolStatus().
				WithStart(osResource.AllocationPools[i].Start).
				WithEnd(osResource.AllocationPools[i].End)
		}
		status.WithAllocationPools(allocationPools...)
	}

	if len(osResource.HostRoutes) > 0 {
		hostRoutes := make([]*orcapplyconfigv1alpha1.HostRouteStatusApplyConfiguration, len(osResource.HostRoutes))
		for i := range osResource.HostRoutes {
			hostRoutes[i] = orcapplyconfigv1alpha1.HostRouteStatus().
				WithDestination(osResource.HostRoutes[i].DestinationCIDR).
				WithNextHop(osResource.HostRoutes[i].NextHop)
		}
		status.WithHostRoutes(hostRoutes...)
	}

	return status
}

func isAvailable(orcObject *orcv1alpha1.Subnet, opts *updateStatusOpts) bool {
	if orcObject == nil {
		return false
	}

	if opts.resource == nil {
		return false
	}

	if orcObject.Spec.ManagementPolicy == orcv1alpha1.ManagementPolicyManaged {
		resource := orcObject.Spec.Resource
		if resource == nil {
			// Should have been caught by validation
			return false
		}

		// We should have a matching routerinterface if the spec requires one
		if !routerInterfaceMatchesSpec(opts.routerInterface, orcObject.Name, resource) {
			return false
		}

		// If we have a routerinterface it should be available
		if opts.routerInterface != nil && !orcv1alpha1.IsAvailable(opts.routerInterface) {
			return false
		}
	}

	return true
}

// createStatusUpdate computes a complete status update based on the given
// observed state. This is separated from updateStatus to facilitate unit
// testing, as the version of k8s we currently import does not support patch
// apply in the fake client.
// Needs: https://github.com/kubernetes/kubernetes/pull/125560
func createStatusUpdate(orcObject *orcv1alpha1.Subnet, now metav1.Time, opts ...updateStatusOpt) *orcapplyconfigv1alpha1.SubnetApplyConfiguration {
	statusOpts := updateStatusOpts{}
	for i := range opts {
		opts[i](&statusOpts)
	}

	osResource := statusOpts.resource

	applyConfigStatus := orcapplyconfigv1alpha1.SubnetStatus()
	applyConfig := orcapplyconfigv1alpha1.Subnet(orcObject.Name, orcObject.Namespace).WithStatus(applyConfigStatus)

	if osResource != nil {
		resourceStatus := getOSResourceStatus(osResource)
		applyConfigStatus.WithResource(resourceStatus)
	}

	available := isAvailable(orcObject, &statusOpts)
	common.SetCommonConditions(orcObject, applyConfigStatus, available, available, statusOpts.progressMessage, statusOpts.err, now)

	return applyConfig
}

// updateStatus computes a complete status based on the given observed state and writes it to status.
func (r *orcSubnetReconciler) updateStatus(ctx context.Context, orcObject *orcv1alpha1.Subnet, opts ...updateStatusOpt) error {
	now := metav1.NewTime(time.Now())

	statusUpdate := createStatusUpdate(orcObject, now, opts...)

	return r.client.Status().Patch(ctx, orcObject, applyconfigs.Patch(types.ApplyPatchType, statusUpdate), client.ForceOwnership, ssaFieldOwner(SSAStatusTxn))
}
