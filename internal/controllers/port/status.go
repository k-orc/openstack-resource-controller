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
	"time"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/common"
	"github.com/k-orc/openstack-resource-controller/internal/util/applyconfigs"
	orcapplyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/pkg/clients/applyconfiguration/api/v1alpha1"
)

// setStatusID sets status.ID in its own SSA transaction.
func (r *orcPortReconciler) setStatusID(ctx context.Context, obj client.Object, id string) error {
	applyConfig := orcapplyconfigv1alpha1.Port(obj.GetName(), obj.GetNamespace()).
		WithUID(obj.GetUID()).
		WithStatus(orcapplyconfigv1alpha1.PortStatus().
			WithID(id))

	return r.client.Status().Patch(ctx, obj, applyconfigs.Patch(types.MergePatchType, applyConfig))
}

type updateStatusOpts struct {
	resource        *ports.Port
	progressMessage *string
	err             error
}

type updateStatusOpt func(*updateStatusOpts)

func withResource(resource *ports.Port) updateStatusOpt {
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

func getOSResourceStatus(osResource *ports.Port) *orcapplyconfigv1alpha1.PortResourceStatusApplyConfiguration {
	status := orcapplyconfigv1alpha1.PortResourceStatus().
		WithName(osResource.Name).
		WithDescription(osResource.Description).
		WithAdminStateUp(osResource.AdminStateUp).
		WithMACAddress(osResource.MACAddress).
		WithDeviceID(osResource.DeviceID).
		WithDeviceOwner(osResource.DeviceOwner).
		WithStatus(osResource.Status).
		WithProjectID(osResource.ProjectID).
		WithTags(osResource.Tags...).
		WithSecurityGroups(osResource.SecurityGroups...).
		WithPropagateUplinkStatus(osResource.PropagateUplinkStatus).
		WithRevisionNumber(int64(osResource.RevisionNumber)).
		WithCreatedAt(metav1.NewTime(osResource.CreatedAt)).
		WithUpdatedAt(metav1.NewTime(osResource.UpdatedAt))

	if len(osResource.AllowedAddressPairs) > 0 {
		allowedAddressPairs := make([]*orcapplyconfigv1alpha1.AllowedAddressPairStatusApplyConfiguration, len(osResource.AllowedAddressPairs))
		for i := range osResource.AllowedAddressPairs {
			allowedAddressPairs[i] = orcapplyconfigv1alpha1.AllowedAddressPairStatus().
				WithIP(osResource.AllowedAddressPairs[i].IPAddress).
				WithMAC(osResource.AllowedAddressPairs[i].MACAddress)
		}
		status.WithAllowedAddressPairs(allowedAddressPairs...)
	}

	if len(osResource.FixedIPs) > 0 {
		fixedIPs := make([]*orcapplyconfigv1alpha1.FixedIPStatusApplyConfiguration, len(osResource.FixedIPs))
		for i := range osResource.FixedIPs {
			fixedIPs[i] = orcapplyconfigv1alpha1.FixedIPStatus().
				WithIP(osResource.FixedIPs[i].IPAddress).
				WithSubnetID(osResource.FixedIPs[i].SubnetID)
		}
		status.WithFixedIPs(fixedIPs...)
	}

	return status
}

// createStatusUpdate computes a complete status update based on the given
// observed state. This is separated from updateStatus to facilitate unit
// testing, as the version of k8s we currently import does not support patch
// apply in the fake client.
// Needs: https://github.com/kubernetes/kubernetes/pull/125560
func createStatusUpdate(_ context.Context, orcPort *orcv1alpha1.Port, now metav1.Time, opts ...updateStatusOpt) *orcapplyconfigv1alpha1.PortApplyConfiguration {
	statusOpts := updateStatusOpts{}
	for i := range opts {
		opts[i](&statusOpts)
	}

	osResource := statusOpts.resource

	applyConfigStatus := orcapplyconfigv1alpha1.PortStatus()
	applyConfig := orcapplyconfigv1alpha1.Port(orcPort.Name, orcPort.Namespace).WithStatus(applyConfigStatus)

	if osResource != nil {
		resourceStatus := getOSResourceStatus(osResource)
		applyConfigStatus.WithResource(resourceStatus)
	}

	// A port is available as soon as it exists
	available := osResource != nil

	common.SetCommonConditions(orcPort, applyConfigStatus, available, available, statusOpts.progressMessage, statusOpts.err, now)

	return applyConfig
}

// updateStatus computes a complete status based on the given observed state and writes it to status.
func (r *orcPortReconciler) updateStatus(ctx context.Context, orcObject *orcv1alpha1.Port, opts ...updateStatusOpt) error {
	now := metav1.NewTime(time.Now())

	statusUpdate := createStatusUpdate(ctx, orcObject, now, opts...)

	return r.client.Status().Patch(ctx, orcObject, applyconfigs.Patch(types.ApplyPatchType, statusUpdate), client.ForceOwnership, ssaFieldOwner(SSAStatusTxn))
}
