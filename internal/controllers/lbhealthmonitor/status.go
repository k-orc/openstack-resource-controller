/*
Copyright The ORC Authors.

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

package lbhealthmonitor

import (
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v3/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v3/internal/controllers/generic/progress"
	orcapplyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/pkg/clients/applyconfiguration/api/v1alpha1"
)

// Octavia provisioning status values
const (
	MonitorProvisioningStatusActive        = "ACTIVE"
	MonitorProvisioningStatusError         = "ERROR"
	MonitorProvisioningStatusPendingCreate = "PENDING_CREATE"
	MonitorProvisioningStatusPendingUpdate = "PENDING_UPDATE"
	MonitorProvisioningStatusPendingDelete = "PENDING_DELETE"
)

type lbhealthmonitorStatusWriter struct{}

type objectApplyT = orcapplyconfigv1alpha1.LBHealthMonitorApplyConfiguration
type statusApplyT = orcapplyconfigv1alpha1.LBHealthMonitorStatusApplyConfiguration

var _ interfaces.ResourceStatusWriter[*orcv1alpha1.LBHealthMonitor, *osResourceT, *objectApplyT, *statusApplyT] = lbhealthmonitorStatusWriter{}

func (lbhealthmonitorStatusWriter) GetApplyConfig(name, namespace string) *objectApplyT {
	return orcapplyconfigv1alpha1.LBHealthMonitor(name, namespace)
}

func (lbhealthmonitorStatusWriter) ResourceAvailableStatus(orcObject *orcv1alpha1.LBHealthMonitor, osResource *osResourceT) (metav1.ConditionStatus, progress.ReconcileStatus) {
	if osResource == nil {
		if orcObject.Status.ID == nil {
			return metav1.ConditionFalse, nil
		} else {
			return metav1.ConditionUnknown, nil
		}
	}
	switch osResource.ProvisioningStatus {
	case MonitorProvisioningStatusActive:
		return metav1.ConditionTrue, nil
	case MonitorProvisioningStatusError:
		return metav1.ConditionFalse, nil
	default:
		// PENDING_CREATE, PENDING_UPDATE, PENDING_DELETE
		return metav1.ConditionFalse, progress.WaitingOnOpenStack(progress.WaitingOnReady, lbhealthmonitorAvailablePollingPeriod)
	}
}

func (lbhealthmonitorStatusWriter) ApplyResourceStatus(log logr.Logger, osResource *osResourceT, statusApply *statusApplyT) {
	resourceStatus := orcapplyconfigv1alpha1.LBHealthMonitorResourceStatus().
		WithType(osResource.Type).
		WithDelay(int32(osResource.Delay)).
		WithTimeout(int32(osResource.Timeout)).
		WithMaxRetries(int32(osResource.MaxRetries)).
		WithMaxRetriesDown(int32(osResource.MaxRetriesDown)).
		WithAdminStateUp(osResource.AdminStateUp).
		WithProvisioningStatus(osResource.ProvisioningStatus).
		WithOperatingStatus(osResource.OperatingStatus)

	if osResource.Name != "" {
		resourceStatus.WithName(osResource.Name)
	}
	if len(osResource.Pools) > 0 {
		resourceStatus.WithPoolID(osResource.Pools[0].ID)
	}
	if osResource.HTTPMethod != "" {
		resourceStatus.WithHTTPMethod(osResource.HTTPMethod)
	}
	if osResource.HTTPVersion != "" {
		resourceStatus.WithHTTPVersion(osResource.HTTPVersion)
	}
	if osResource.ExpectedCodes != "" {
		resourceStatus.WithExpectedCodes(osResource.ExpectedCodes)
	}
	if osResource.URLPath != "" {
		resourceStatus.WithURLPath(osResource.URLPath)
	}
	if osResource.DomainName != "" {
		resourceStatus.WithDomainName(osResource.DomainName)
	}
	if len(osResource.Tags) > 0 {
		resourceStatus.WithTags(osResource.Tags...)
	}

	statusApply.WithResource(resourceStatus)
}
