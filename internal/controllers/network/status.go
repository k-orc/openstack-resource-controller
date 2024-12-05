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
	"strconv"
	"time"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/common"
	"github.com/k-orc/openstack-resource-controller/internal/util/applyconfigs"
	orcapplyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/pkg/clients/applyconfiguration/api/v1alpha1"
)

// setStatusID sets status.ID in its own SSA transaction.
func (r *orcNetworkReconciler) setStatusID(ctx context.Context, obj client.Object, id string) error {
	applyConfig := orcapplyconfigv1alpha1.Network(obj.GetName(), obj.GetNamespace()).
		WithUID(obj.GetUID()).
		WithStatus(orcapplyconfigv1alpha1.NetworkStatus().
			WithID(id))

	return r.client.Status().Patch(ctx, obj, applyconfigs.Patch(types.ApplyPatchType, applyConfig), client.ForceOwnership, ssaFieldOwner(SSAIDTxn))
}

type updateStatusOpts struct {
	resource        *networkExt
	progressMessage *string
	err             error
}

type updateStatusOpt func(*updateStatusOpts)

func withResource(resource *networkExt) updateStatusOpt {
	return func(opts *updateStatusOpts) {
		opts.resource = resource
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

func getOSResourceStatus(log logr.Logger, osResource *networkExt) *orcapplyconfigv1alpha1.NetworkResourceStatusApplyConfiguration {
	networkResourceStatus := (&orcapplyconfigv1alpha1.NetworkResourceStatusApplyConfiguration{}).
		WithName(osResource.Name).
		WithDescription(osResource.Description).
		WithAdminStateUp(osResource.AdminStateUp).
		WithAvailabilityZoneHints(osResource.AvailabilityZoneHints...).
		WithStatus(osResource.Status).
		WithProjectID(osResource.ProjectID).
		WithTags(osResource.Tags...).
		WithDNSDomain(osResource.DNSDomain).
		WithRevisionNumber(int64(osResource.RevisionNumber)).
		WithExternal(osResource.External).
		WithSubnets(osResource.Subnets...).
		WithMTU(int32(osResource.MTU)).
		WithPortSecurityEnabled(osResource.PortSecurityEnabled).
		WithShared(osResource.Shared).
		WithCreatedAt(metav1.NewTime(osResource.CreatedAt)).
		WithUpdatedAt(metav1.NewTime(osResource.UpdatedAt))

	if osResource.NetworkType != "" {
		providerProperties := orcapplyconfigv1alpha1.ProviderProperties().
			WithNetworkType(orcv1alpha1.ProviderNetworkType(osResource.NetworkType)).
			WithPhysicalNetwork(orcv1alpha1.PhysicalNetwork(osResource.PhysicalNetwork))

		if osResource.SegmentationID != "" {
			segmentationID, err := strconv.ParseInt(osResource.SegmentationID, 10, 32)
			if err != nil {
				log.V(3).Error(err, "Invalid segmentation ID", "segmentationID", osResource.SegmentationID)
			} else {
				providerProperties.WithSegmentationID(int32(segmentationID))
			}
		}
		networkResourceStatus.WithProvider(providerProperties)
	}

	return networkResourceStatus
}

const NetworkStatusActive = "ACTIVE"

// createStatusUpdate computes a complete status update based on the given
// observed state. This is separated from updateStatus to facilitate unit
// testing, as the version of k8s we currently import does not support patch
// apply in the fake client.
// Needs: https://github.com/kubernetes/kubernetes/pull/125560
func createStatusUpdate(ctx context.Context, orcNetwork *orcv1alpha1.Network, now metav1.Time, opts ...updateStatusOpt) *orcapplyconfigv1alpha1.NetworkApplyConfiguration {
	log := ctrl.LoggerFrom(ctx)

	statusOpts := updateStatusOpts{}
	for i := range opts {
		opts[i](&statusOpts)
	}

	osResource := statusOpts.resource

	applyConfigStatus := orcapplyconfigv1alpha1.NetworkStatus()
	applyConfig := orcapplyconfigv1alpha1.Network(orcNetwork.Name, orcNetwork.Namespace).WithStatus(applyConfigStatus)

	if osResource != nil {
		resourceStatus := getOSResourceStatus(log, osResource)
		applyConfigStatus.WithResource(resourceStatus)
	}

	available := osResource != nil && osResource.Status == NetworkStatusActive
	common.SetCommonConditions(orcNetwork, applyConfigStatus, available, available, statusOpts.progressMessage, statusOpts.err, now)

	return applyConfig
}

// updateStatus computes a complete status based on the given observed state and writes it to status.
func (r *orcNetworkReconciler) updateStatus(ctx context.Context, orcObject *orcv1alpha1.Network, opts ...updateStatusOpt) error {
	now := metav1.NewTime(time.Now())

	statusUpdate := createStatusUpdate(ctx, orcObject, now, opts...)

	return r.client.Status().Patch(ctx, orcObject, applyconfigs.Patch(types.ApplyPatchType, statusUpdate), client.ForceOwnership, ssaFieldOwner(SSAStatusTxn))
}
