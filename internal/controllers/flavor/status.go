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

package flavor

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/flavors"
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
func (r *orcFlavorReconciler) setStatusID(ctx context.Context, obj client.Object, id string) error {
	applyConfig := orcapplyconfigv1alpha1.Flavor(obj.GetName(), obj.GetNamespace()).
		WithUID(obj.GetUID()).
		WithStatus(orcapplyconfigv1alpha1.FlavorStatus().
			WithID(id))

	return r.client.Status().Patch(ctx, obj, applyconfigs.Patch(types.ApplyPatchType, applyConfig), client.ForceOwnership, ssaFieldOwner(SSAIDTxn))
}

type updateStatusOpts struct {
	resource        *flavors.Flavor
	progressMessage *string
	err             error
}

type updateStatusOpt func(*updateStatusOpts)

func withResource(resource *flavors.Flavor) updateStatusOpt {
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

func getOSResourceStatus(_ logr.Logger, osResource *flavors.Flavor) *orcapplyconfigv1alpha1.FlavorResourceStatusApplyConfiguration {
	return (&orcapplyconfigv1alpha1.FlavorResourceStatusApplyConfiguration{}).
		WithName(osResource.Name).
		WithDescription(osResource.Description).
		WithRAM(int32(osResource.RAM)).
		WithDisk(int32(osResource.Disk)).
		WithVcpus(int32(osResource.VCPUs)).
		WithSwap(int32(osResource.Swap)).
		WithIsPublic(osResource.IsPublic).
		WithEphemeral(int32(osResource.Ephemeral))
}

// createStatusUpdate computes a complete status update based on the given
// observed state. This is separated from updateStatus to facilitate unit
// testing, as the version of k8s we currently import does not support patch
// apply in the fake client.
// Needs: https://github.com/kubernetes/kubernetes/pull/125560
func createStatusUpdate(ctx context.Context, orcObject *orcv1alpha1.Flavor, now metav1.Time, opts ...updateStatusOpt) *orcapplyconfigv1alpha1.FlavorApplyConfiguration {
	log := ctrl.LoggerFrom(ctx)

	statusOpts := updateStatusOpts{}
	for i := range opts {
		opts[i](&statusOpts)
	}

	osResource := statusOpts.resource

	applyConfigStatus := orcapplyconfigv1alpha1.FlavorStatus()
	applyConfig := orcapplyconfigv1alpha1.Flavor(orcObject.Name, orcObject.Namespace).WithStatus(applyConfigStatus)

	if osResource != nil {
		resourceStatus := getOSResourceStatus(log, osResource)
		applyConfigStatus.WithResource(resourceStatus)
	}

	available := osResource != nil
	common.SetCommonConditions(orcObject, applyConfigStatus, available, available, statusOpts.progressMessage, statusOpts.err, now)

	return applyConfig
}

// updateStatus computes a complete status based on the given observed state and writes it to status.
func (r *orcFlavorReconciler) updateStatus(ctx context.Context, orcObject *orcv1alpha1.Flavor, opts ...updateStatusOpt) error {
	now := metav1.NewTime(time.Now())

	statusUpdate := createStatusUpdate(ctx, orcObject, now, opts...)

	return r.client.Status().Patch(ctx, orcObject, applyconfigs.Patch(types.ApplyPatchType, statusUpdate), client.ForceOwnership, ssaFieldOwner(SSAStatusTxn))
}
