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

package server

import (
	"context"
	"time"

	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/common"
	"github.com/k-orc/openstack-resource-controller/internal/util/applyconfigs"
	applyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/pkg/clients/applyconfiguration/api/v1alpha1"
)

const (
	ServerStatusActive = "ACTIVE"
)

// setStatusID sets status.ID in its own SSA transaction.
func (r *orcServerReconciler) setStatusID(ctx context.Context, obj client.Object, id string) error {
	applyConfig := applyconfigv1alpha1.Server(obj.GetName(), obj.GetNamespace()).
		WithUID(obj.GetUID()).
		WithStatus(applyconfigv1alpha1.ServerStatus().
			WithID(id))

	return r.client.Status().Patch(ctx, obj, applyconfigs.Patch(types.ApplyPatchType, applyConfig), client.ForceOwnership, ssaFieldOwner(SSAIDTxn))
}

type updateStatusOpts struct {
	resource        *servers.Server
	progressMessage *string
	err             error
}

type updateStatusOpt func(*updateStatusOpts)

func withResource(resource *servers.Server) updateStatusOpt {
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

func getOSResourceStatus(osResource *servers.Server) *applyconfigv1alpha1.ServerResourceStatusApplyConfiguration {
	// TODO: Add the rest of the OpenStack data to Status
	status := applyconfigv1alpha1.ServerResourceStatus().
		WithName(osResource.Name).
		WithStatus(osResource.Status).
		WithHostID(osResource.HostID).
		WithAccessIPv4(osResource.AccessIPv4).
		WithAccessIPv6(osResource.AccessIPv6).
		WithFault(osResource.Fault.Message)
	return status
}

func isAvailable(orcObject *v1alpha1.Server, osResource *servers.Server) bool {
	return orcObject.Status.ID != nil && osResource != nil && osResource.Status == ServerStatusActive
}

// createStatusUpdate computes a complete status update based on the given
// observed state. This is separated from updateStatus to facilitate unit
// testing, as the version of k8s we currently import does not support patch
// apply in the fake client.
// Needs: https://github.com/kubernetes/kubernetes/pull/125560
func createStatusUpdate(orcObject *v1alpha1.Server, now metav1.Time, opts ...updateStatusOpt) *applyconfigv1alpha1.ServerApplyConfiguration {
	statusOpts := updateStatusOpts{}
	for i := range opts {
		opts[i](&statusOpts)
	}

	osResource := statusOpts.resource

	applyConfigStatus := applyconfigv1alpha1.ServerStatus()
	applyConfig := applyconfigv1alpha1.Server(orcObject.Name, orcObject.Namespace).WithStatus(applyConfigStatus)

	if osResource != nil {
		resourceStatus := getOSResourceStatus(osResource)
		applyConfigStatus.WithResource(resourceStatus)
	}

	available := isAvailable(orcObject, osResource)
	common.SetCommonConditions(orcObject, applyConfigStatus, available, available, statusOpts.progressMessage, statusOpts.err, now)

	return applyConfig
}

// updateStatus computes a complete status based on the given observed state and writes it to status.
func (r *orcServerReconciler) updateStatus(ctx context.Context, orcObject *v1alpha1.Server, opts ...updateStatusOpt) error {
	now := metav1.NewTime(time.Now())

	statusUpdate := createStatusUpdate(orcObject, now, opts...)

	return r.client.Status().Patch(ctx, orcObject, applyconfigs.Patch(types.ApplyPatchType, statusUpdate), client.ForceOwnership, ssaFieldOwner(SSAStatusTxn))
}
