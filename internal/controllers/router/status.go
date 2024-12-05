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

package router

import (
	"context"
	"time"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/routers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/common"
	"github.com/k-orc/openstack-resource-controller/internal/util/applyconfigs"
	orcapplyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/pkg/clients/applyconfiguration/api/v1alpha1"
)

const (
	RouterStatusActive = "ACTIVE"
)

// setStatusID sets status.ID in its own SSA transaction.
func (r *orcRouterReconciler) setStatusID(ctx context.Context, obj client.Object, id string) error {
	applyConfig := orcapplyconfigv1alpha1.Router(obj.GetName(), obj.GetNamespace()).
		WithUID(obj.GetUID()).
		WithStatus(orcapplyconfigv1alpha1.RouterStatus().
			WithID(id))

	return r.client.Status().Patch(ctx, obj, applyconfigs.Patch(types.ApplyPatchType, applyConfig), client.ForceOwnership, ssaFieldOwner(SSAIDTxn))
}

type updateStatusOpts struct {
	resource        *routers.Router
	progressMessage *string
	err             error
}

type updateStatusOpt func(*updateStatusOpts)

func withResource(resource *routers.Router) updateStatusOpt {
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

func getOSResourceStatus(osResource *routers.Router) *orcapplyconfigv1alpha1.RouterResourceStatusApplyConfiguration {
	status := orcapplyconfigv1alpha1.RouterResourceStatus().
		WithName(osResource.Name).
		WithDescription(osResource.Description).
		WithProjectID(osResource.ProjectID).
		WithStatus(osResource.Status).
		WithTags(osResource.Tags...).
		WithAdminStateUp(osResource.AdminStateUp).
		WithAvailabilityZoneHints(osResource.AvailabilityZoneHints...)

	if osResource.GatewayInfo.NetworkID != "" {
		status.WithExternalGateways(orcapplyconfigv1alpha1.ExternalGatewayStatus().
			WithNetworkID(osResource.GatewayInfo.NetworkID))
	}

	return status
}

func isAvailable(orcObject *orcv1alpha1.Router, osResource *routers.Router) bool {
	return orcObject.Status.ID != nil && osResource != nil && osResource.Status == RouterStatusActive
}

// createStatusUpdate computes a complete status update based on the given
// observed state. This is separated from updateStatus to facilitate unit
// testing, as the version of k8s we currently import does not support patch
// apply in the fake client.
// Needs: https://github.com/kubernetes/kubernetes/pull/125560
func createStatusUpdate(orcObject *orcv1alpha1.Router, now metav1.Time, opts ...updateStatusOpt) *orcapplyconfigv1alpha1.RouterApplyConfiguration {
	statusOpts := updateStatusOpts{}
	for i := range opts {
		opts[i](&statusOpts)
	}

	osResource := statusOpts.resource

	applyConfigStatus := orcapplyconfigv1alpha1.RouterStatus()
	applyConfig := orcapplyconfigv1alpha1.Router(orcObject.Name, orcObject.Namespace).WithStatus(applyConfigStatus)

	if osResource != nil {
		resourceStatus := getOSResourceStatus(osResource)
		applyConfigStatus.WithResource(resourceStatus)
	}

	available := isAvailable(orcObject, osResource)
	common.SetCommonConditions(orcObject, applyConfigStatus, available, available, statusOpts.progressMessage, statusOpts.err, now)

	return applyConfig
}

// updateStatus computes a complete status based on the given observed state and writes it to status.
func (r *orcRouterReconciler) updateStatus(ctx context.Context, orcObject *orcv1alpha1.Router, opts ...updateStatusOpt) error {
	now := metav1.NewTime(time.Now())

	statusUpdate := createStatusUpdate(orcObject, now, opts...)

	return r.client.Status().Patch(ctx, orcObject, applyconfigs.Patch(types.ApplyPatchType, statusUpdate), client.ForceOwnership, ssaFieldOwner(SSAStatusTxn))
}
