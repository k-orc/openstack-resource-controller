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
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v3/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v3/internal/controllers/generic/progress"
	orcapplyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/pkg/clients/applyconfiguration/api/v1alpha1"
)

type dnszoneStatusWriter struct{}

type objectApplyT = orcapplyconfigv1alpha1.DNSZoneApplyConfiguration
type statusApplyT = orcapplyconfigv1alpha1.DNSZoneStatusApplyConfiguration

var _ interfaces.ResourceStatusWriter[*orcv1alpha1.DNSZone, *osResourceT, *objectApplyT, *statusApplyT] = dnszoneStatusWriter{}

func (dnszoneStatusWriter) GetApplyConfig(name, namespace string) *objectApplyT {
	return orcapplyconfigv1alpha1.DNSZone(name, namespace)
}

func (dnszoneStatusWriter) ResourceAvailableStatus(orcObject *orcv1alpha1.DNSZone, osResource *osResourceT) (metav1.ConditionStatus, progress.ReconcileStatus) {
	if osResource == nil {
		if orcObject.Status.ID == nil {
			return metav1.ConditionFalse, nil
		} else {
			return metav1.ConditionUnknown, nil
		}
	}
	return metav1.ConditionTrue, nil
}

func (dnszoneStatusWriter) ApplyResourceStatus(log logr.Logger, osResource *osResourceT, statusApply *statusApplyT) {
	resourceStatus := orcapplyconfigv1alpha1.DNSZoneResourceStatus().
		WithName(osResource.Name).
		WithType(osResource.Type).
		WithProjectID(osResource.ProjectID).
		WithSerial(int64(osResource.Serial))

	if osResource.Email != "" {
		resourceStatus.WithEmail(osResource.Email)
	}
	if osResource.Description != "" {
		resourceStatus.WithDescription(osResource.Description)
	}
	if osResource.TTL != 0 {
		resourceStatus.WithTTL(int32(osResource.TTL))
	}
	if len(osResource.Masters) > 0 {
		resourceStatus.WithMasters(osResource.Masters...)
	}
	// TransferredAt only has meaning for a SECONDARY zone that has actually synced from its
	// masters at least once - a PRIMARY zone (or a SECONDARY that hasn't transferred yet) reports
	// a zero time here, which must not be rendered as a literal JSON null (the exact bug already
	// found and fixed upstream for six other controllers' CreatedAt/UpdatedAt fields - see PR
	// #947 - applied proactively here rather than reintroducing the same class of bug).
	if !osResource.TransferredAt.IsZero() {
		resourceStatus.WithTransferredAt(metav1.NewTime(osResource.TransferredAt))
	}

	statusApply.WithResource(resourceStatus)
}
