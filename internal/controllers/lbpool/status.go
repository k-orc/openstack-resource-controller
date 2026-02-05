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

package lbpool

import (
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/progress"
	orcapplyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/pkg/clients/applyconfiguration/api/v1alpha1"
)

type lbpoolStatusWriter struct{}

type objectApplyT = orcapplyconfigv1alpha1.LBPoolApplyConfiguration
type statusApplyT = orcapplyconfigv1alpha1.LBPoolStatusApplyConfiguration

var _ interfaces.ResourceStatusWriter[*orcv1alpha1.LBPool, *osResourceT, *objectApplyT, *statusApplyT] = lbpoolStatusWriter{}

func (lbpoolStatusWriter) GetApplyConfig(name, namespace string) *objectApplyT {
	return orcapplyconfigv1alpha1.LBPool(name, namespace)
}

func (lbpoolStatusWriter) ResourceAvailableStatus(orcObject *orcv1alpha1.LBPool, osResource *osResourceT) (metav1.ConditionStatus, progress.ReconcileStatus) {
	if osResource == nil {
		if orcObject.Status.ID == nil {
			return metav1.ConditionFalse, nil
		} else {
			return metav1.ConditionUnknown, nil
		}
	}

	switch osResource.ProvisioningStatus {
	case PoolProvisioningStatusActive:
		return metav1.ConditionTrue, nil
	case PoolProvisioningStatusError:
		return metav1.ConditionFalse, nil
	}

	// Otherwise we should continue to poll
	return metav1.ConditionFalse, progress.WaitingOnOpenStack(progress.WaitingOnReady, lbpoolAvailablePollingPeriod)
}

func (lbpoolStatusWriter) ApplyResourceStatus(log logr.Logger, osResource *osResourceT, statusApply *statusApplyT) {
	resourceStatus := orcapplyconfigv1alpha1.LBPoolResourceStatus().
		WithName(osResource.Name).
		WithLBAlgorithm(osResource.LBMethod).
		WithProtocol(osResource.Protocol).
		WithProjectID(osResource.ProjectID).
		WithProvisioningStatus(osResource.ProvisioningStatus).
		WithOperatingStatus(osResource.OperatingStatus).
		WithAdminStateUp(osResource.AdminStateUp).
		WithTLSEnabled(osResource.TLSEnabled)

	if osResource.Description != "" {
		resourceStatus.WithDescription(osResource.Description)
	}

	if osResource.MonitorID != "" {
		resourceStatus.WithHealthMonitorID(osResource.MonitorID)
	}

	if osResource.TLSContainerRef != "" {
		resourceStatus.WithTLSContainerRef(osResource.TLSContainerRef)
	}

	if osResource.CATLSContainerRef != "" {
		resourceStatus.WithCATLSContainerRef(osResource.CATLSContainerRef)
	}

	if osResource.CRLContainerRef != "" {
		resourceStatus.WithCRLContainerRef(osResource.CRLContainerRef)
	}

	if osResource.TLSCiphers != "" {
		resourceStatus.WithTLSCiphers(osResource.TLSCiphers)
	}

	if len(osResource.TLSVersions) > 0 {
		resourceStatus.WithTLSVersions(osResource.TLSVersions...)
	}

	if len(osResource.ALPNProtocols) > 0 {
		resourceStatus.WithALPNProtocols(osResource.ALPNProtocols...)
	}

	if len(osResource.Tags) > 0 {
		resourceStatus.WithTags(osResource.Tags...)
	}

	// Extract LoadBalancer IDs
	if len(osResource.Loadbalancers) > 0 {
		lbIDs := make([]string, len(osResource.Loadbalancers))
		for i, lb := range osResource.Loadbalancers {
			lbIDs[i] = lb.ID
		}
		resourceStatus.WithLoadBalancerIDs(lbIDs...)
	}

	// Extract Listener IDs
	if len(osResource.Listeners) > 0 {
		listenerIDs := make([]string, len(osResource.Listeners))
		for i, listener := range osResource.Listeners {
			listenerIDs[i] = listener.ID
		}
		resourceStatus.WithListenerIDs(listenerIDs...)
	}

	// Extract Member details
	for i := range osResource.Members {
		member := &osResource.Members[i]

		memberStatus := orcapplyconfigv1alpha1.LBPoolMemberStatus().
			WithID(member.ID).
			WithAddress(member.Address).
			WithProtocolPort(int32(member.ProtocolPort)).
			WithWeight(int32(member.Weight)).
			WithAdminStateUp(member.AdminStateUp).
			WithBackup(member.Backup).
			WithProvisioningStatus(member.ProvisioningStatus).
			WithOperatingStatus(member.OperatingStatus)
		if member.Name != "" {
			memberStatus.WithName(member.Name)
		}
		if member.SubnetID != "" {
			memberStatus.WithSubnetID(member.SubnetID)
		}
		resourceStatus.WithMembers(memberStatus)
	}

	// Session persistence
	if osResource.Persistence.Type != "" {
		sessionPersistence := orcapplyconfigv1alpha1.LBPoolSessionPersistence().
			WithType(orcv1alpha1.LBPoolSessionPersistenceType(osResource.Persistence.Type))
		if osResource.Persistence.CookieName != "" {
			sessionPersistence.WithCookieName(osResource.Persistence.CookieName)
		}
		resourceStatus.WithSessionPersistence(sessionPersistence)
	}

	statusApply.WithResource(resourceStatus)
}
