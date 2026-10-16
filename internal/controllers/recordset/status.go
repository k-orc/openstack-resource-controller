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

package recordset

import (
	"slices"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v3/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v3/internal/controllers/generic/progress"
	orcapplyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/pkg/clients/applyconfiguration/api/v1alpha1"
)

type recordsetStatusWriter struct{}

type objectApplyT = orcapplyconfigv1alpha1.RecordSetApplyConfiguration
type statusApplyT = orcapplyconfigv1alpha1.RecordSetStatusApplyConfiguration

var _ interfaces.ResourceStatusWriter[*orcv1alpha1.RecordSet, *osResourceT, *objectApplyT, *statusApplyT] = recordsetStatusWriter{}

func (recordsetStatusWriter) GetApplyConfig(name, namespace string) *objectApplyT {
	return orcapplyconfigv1alpha1.RecordSet(name, namespace)
}

func (recordsetStatusWriter) ResourceAvailableStatus(orcObject *orcv1alpha1.RecordSet, osResource *osResourceT) (metav1.ConditionStatus, progress.ReconcileStatus) {
	if osResource == nil {
		if orcObject.Status.ID == nil {
			return metav1.ConditionFalse, nil
		} else {
			return metav1.ConditionUnknown, nil
		}
	}
	return metav1.ConditionTrue, nil
}

func (recordsetStatusWriter) ApplyResourceStatus(log logr.Logger, osResource *osResourceT, statusApply *statusApplyT) {
	resourceStatus := orcapplyconfigv1alpha1.RecordSetResourceStatus().
		WithName(osResource.Name).
		WithZoneID(osResource.ZoneID).
		WithType(osResource.Type).
		WithProjectID(osResource.ProjectID).
		WithTTL(int32(osResource.TTL))

	if osResource.Description != "" {
		resourceStatus.WithDescription(osResource.Description)
	}
	if len(osResource.Records) > 0 {
		// Designate does not guarantee record order within a recordset; sort for a stable status.
		records := slices.Clone(osResource.Records)
		slices.Sort(records)
		resourceStatus.WithRecords(records...)
	}

	statusApply.WithResource(resourceStatus)
}
