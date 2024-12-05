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

package routerinterface

import (
	"context"
	"time"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/common"
	"github.com/k-orc/openstack-resource-controller/internal/util/applyconfigs"
	orcapplyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/pkg/clients/applyconfiguration/api/v1alpha1"
)

const portStatusActive = "ACTIVE"

type updateStatusOpts struct {
	subnet *orcv1alpha1.Subnet
	port   *ports.Port
	err    error
}

func getStatusSummary(routerInterface *orcv1alpha1.RouterInterface, opts *updateStatusOpts) (bool, bool, *string) {
	if routerInterface == nil {
		return false, false, nil
	}

	if routerInterface.Spec.Type == orcv1alpha1.RouterInterfaceTypeSubnet {
		if opts.subnet == nil {
			return false, false, ptr.To("Waiting for subnet object to be created")
		}
		if opts.subnet.Status.ID == nil {
			return false, false, ptr.To("Waiting for subnet resource to be created")
		}
	}

	if opts.port == nil {
		return false, false, ptr.To("Waiting for interface to be created")
	}

	if opts.port.Status != portStatusActive {
		return false, false, ptr.To("Waiting for interface to be ACTIVE, currently " + opts.port.Status)
	}

	return true, true, nil
}

// createStatusUpdate computes a complete status update based on the given
// observed state. This is separated from updateStatus to facilitate unit
// testing, as the version of k8s we currently import does not support patch
// apply in the fake client.
// Needs: https://github.com/kubernetes/kubernetes/pull/125560
func createStatusUpdate(orcObject *orcv1alpha1.RouterInterface, now metav1.Time, statusOpts *updateStatusOpts) *orcapplyconfigv1alpha1.RouterInterfaceApplyConfiguration {
	applyConfigStatus := orcapplyconfigv1alpha1.RouterInterfaceStatus()
	applyConfig := orcapplyconfigv1alpha1.RouterInterface(orcObject.Name, orcObject.Namespace).WithStatus(applyConfigStatus)

	// Note that unlike other resources we don't rely on this value to be immutable, so it's not in a separate transaction.
	if statusOpts.port != nil {
		applyConfigStatus.WithID(statusOpts.port.ID)
	}

	isAvailable, isUpToDate, progressMessage := getStatusSummary(orcObject, statusOpts)
	common.SetCommonConditions(orcObject, applyConfigStatus, isAvailable, isUpToDate, progressMessage, statusOpts.err, now)

	return applyConfig
}

// updateStatus computes a complete status based on the given observed state and writes it to status.
func (r *orcRouterInterfaceReconciler) updateStatus(ctx context.Context, orcObject *orcv1alpha1.RouterInterface, opts *updateStatusOpts) error {
	now := metav1.NewTime(time.Now())

	statusUpdate := createStatusUpdate(orcObject, now, opts)
	return r.client.Status().Patch(ctx, orcObject, applyconfigs.Patch(types.ApplyPatchType, statusUpdate), client.ForceOwnership, ssaFieldOwner(SSAStatusTxn))
}
