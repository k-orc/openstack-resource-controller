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

package dnszone

import (
	"time"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/progress"
	orcerrors "github.com/k-orc/openstack-resource-controller/v2/internal/util/errors"
	orcapplyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/pkg/clients/applyconfiguration/api/v1alpha1"
)

const (
	ZoneStatusActive  = "ACTIVE"
	ZoneStatusPending = "PENDING"
	ZoneStatusError   = "ERROR"

	// The time to wait before reconciling again when we are expecting OpenStack to finish some task and update status.
	externalUpdatePollingPeriod = 15 * time.Second
)

type dnsZoneStatusWriter struct{}

type objectApplyT = orcapplyconfigv1alpha1.DNSZoneApplyConfiguration
type statusApplyT = orcapplyconfigv1alpha1.DNSZoneStatusApplyConfiguration

var _ interfaces.ResourceStatusWriter[*orcv1alpha1.DNSZone, *osResourceT, *objectApplyT, *statusApplyT] = dnsZoneStatusWriter{}

func (dnsZoneStatusWriter) GetApplyConfig(name, namespace string) *objectApplyT {
	return orcapplyconfigv1alpha1.DNSZone(name, namespace)
}

func (dnsZoneStatusWriter) ResourceAvailableStatus(orcObject *orcv1alpha1.DNSZone, osResource *osResourceT) (metav1.ConditionStatus, progress.ReconcileStatus) {
	if osResource == nil {
		if orcObject.Status.ID == nil {
			return metav1.ConditionFalse, nil
		} else {
			return metav1.ConditionUnknown, nil
		}
	}

	switch osResource.Status {
	case ZoneStatusActive:
		return metav1.ConditionTrue, nil
	case ZoneStatusPending:
		return metav1.ConditionFalse, progress.WaitingOnOpenStack(progress.WaitingOnReady, externalUpdatePollingPeriod)
	case ZoneStatusError:
		return metav1.ConditionFalse, progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonUnrecoverableError, "OpenStack zone is in ERROR status"))
	default:
		// Fallback for any other/unexpected status
		return metav1.ConditionFalse, progress.WaitingOnOpenStack(progress.WaitingOnReady, externalUpdatePollingPeriod)
	}
}

func (dnsZoneStatusWriter) ApplyResourceStatus(log logr.Logger, osResource *osResourceT, statusApply *statusApplyT) {
	resourceStatus := orcapplyconfigv1alpha1.DNSZoneResourceStatus().
		WithName(osResource.Name)

	if osResource.Email != "" {
		resourceStatus.WithEmail(osResource.Email)
	}

	if osResource.Description != "" {
		resourceStatus.WithDescription(osResource.Description)
	}

	if osResource.TTL > 0 {
		resourceStatus.WithTTL(int32(osResource.TTL))
	}

	if osResource.Type != "" {
		resourceStatus.WithType(osResource.Type)
	}

	if len(osResource.Masters) > 0 {
		resourceStatus.WithMasters(osResource.Masters...)
	}

	if !osResource.TransferredAt.IsZero() {
		resourceStatus.WithTransferredAt(metav1.NewTime(osResource.TransferredAt))
	}

	if osResource.Status != "" {
		resourceStatus.WithStatus(osResource.Status)
	}

	statusApply.WithResource(resourceStatus)
}
