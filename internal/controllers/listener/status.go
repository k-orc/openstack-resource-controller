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

package listener

import (
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/progress"
	orcapplyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/pkg/clients/applyconfiguration/api/v1alpha1"
)

// Octavia provisioning status values
const (
	ListenerProvisioningStatusActive        = "ACTIVE"
	ListenerProvisioningStatusError         = "ERROR"
	ListenerProvisioningStatusPendingCreate = "PENDING_CREATE"
	ListenerProvisioningStatusPendingUpdate = "PENDING_UPDATE"
	ListenerProvisioningStatusPendingDelete = "PENDING_DELETE"
)

type listenerStatusWriter struct{}

type objectApplyT = orcapplyconfigv1alpha1.ListenerApplyConfiguration
type statusApplyT = orcapplyconfigv1alpha1.ListenerStatusApplyConfiguration

var _ interfaces.ResourceStatusWriter[*orcv1alpha1.Listener, *osResourceT, *objectApplyT, *statusApplyT] = listenerStatusWriter{}

func (listenerStatusWriter) GetApplyConfig(name, namespace string) *objectApplyT {
	return orcapplyconfigv1alpha1.Listener(name, namespace)
}

func (listenerStatusWriter) ResourceAvailableStatus(orcObject *orcv1alpha1.Listener, osResource *osResourceT) (metav1.ConditionStatus, progress.ReconcileStatus) {
	if osResource == nil {
		if orcObject.Status.ID == nil {
			return metav1.ConditionFalse, nil
		}
		return metav1.ConditionUnknown, nil
	}

	switch osResource.ProvisioningStatus {
	case ListenerProvisioningStatusActive:
		return metav1.ConditionTrue, nil
	case ListenerProvisioningStatusError:
		return metav1.ConditionFalse, nil
	default:
		// PENDING_CREATE, PENDING_UPDATE, PENDING_DELETE
		return metav1.ConditionFalse, progress.WaitingOnOpenStack(progress.WaitingOnReady, listenerAvailablePollingPeriod)
	}
}

func (listenerStatusWriter) ApplyResourceStatus(log logr.Logger, osResource *osResourceT, statusApply *statusApplyT) {
	resourceStatus := orcapplyconfigv1alpha1.ListenerResourceStatus().
		WithName(osResource.Name).
		WithProtocol(osResource.Protocol).
		WithProtocolPort(int32(osResource.ProtocolPort)).
		WithAdminStateUp(osResource.AdminStateUp).
		WithConnectionLimit(int32(osResource.ConnLimit)).
		WithProvisioningStatus(osResource.ProvisioningStatus).
		WithOperatingStatus(osResource.OperatingStatus).
		WithTimeoutClientData(int32(osResource.TimeoutClientData)).
		WithTimeoutMemberConnect(int32(osResource.TimeoutMemberConnect)).
		WithTimeoutMemberData(int32(osResource.TimeoutMemberData)).
		WithTimeoutTCPInspect(int32(osResource.TimeoutTCPInspect))

	if osResource.Description != "" {
		resourceStatus.WithDescription(osResource.Description)
	}

	if osResource.DefaultPoolID != "" {
		resourceStatus.WithDefaultPoolID(osResource.DefaultPoolID)
	}

	// Get the first loadbalancer ID if available
	if len(osResource.Loadbalancers) > 0 {
		resourceStatus.WithLoadBalancerID(osResource.Loadbalancers[0].ID)
	}

	if len(osResource.AllowedCIDRs) > 0 {
		resourceStatus.WithAllowedCIDRs(osResource.AllowedCIDRs...)
	}

	if osResource.InsertHeaders != nil {
		resourceStatus.WithInsertHeaders(osResource.InsertHeaders)
	}

	if len(osResource.Tags) > 0 {
		resourceStatus.WithTags(osResource.Tags...)
	}

	statusApply.WithResource(resourceStatus)
}
