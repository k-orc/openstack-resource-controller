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

package trunk

import (
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/progress"
	orcapplyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/pkg/clients/applyconfiguration/api/v1alpha1"
)

const (
	TrunkStatusActive = "ACTIVE"
	TrunkStatusDown   = "DOWN"
)

type objectApplyPT = *orcapplyconfigv1alpha1.TrunkApplyConfiguration
type statusApplyPT = *orcapplyconfigv1alpha1.TrunkStatusApplyConfiguration

type trunkStatusWriter struct{}

var _ interfaces.ResourceStatusWriter[orcObjectPT, *osResourceT, objectApplyPT, statusApplyPT] = trunkStatusWriter{}

func (trunkStatusWriter) GetApplyConfig(name, namespace string) objectApplyPT {
	return orcapplyconfigv1alpha1.Trunk(name, namespace)
}

func (trunkStatusWriter) ResourceAvailableStatus(orcObject orcObjectPT, osResource *osResourceT) (metav1.ConditionStatus, progress.ReconcileStatus) {
	if osResource == nil {
		if orcObject.Status.ID == nil {
			return metav1.ConditionFalse, nil
		} else {
			return metav1.ConditionUnknown, nil
		}
	}

	// Both active and down trunks are Available
	if osResource.Status == TrunkStatusActive || osResource.Status == TrunkStatusDown {
		return metav1.ConditionTrue, nil
	}
	return metav1.ConditionFalse, nil
}

func (trunkStatusWriter) ApplyResourceStatus(log logr.Logger, osResource *osResourceT, statusApply statusApplyPT) {
	resourceStatus := orcapplyconfigv1alpha1.TrunkResourceStatus().
		WithName(osResource.Name).
		WithAdminStateUp(osResource.AdminStateUp).
		WithStatus(osResource.Status).
		WithProjectID(osResource.ProjectID).
		WithPortID(osResource.PortID).
		WithTags(osResource.Tags...).
		WithRevisionNumber(int64(osResource.RevisionNumber)).
		WithCreatedAt(metav1.NewTime(osResource.CreatedAt)).
		WithUpdatedAt(metav1.NewTime(osResource.UpdatedAt))

	if osResource.Description != "" {
		resourceStatus = resourceStatus.WithDescription(osResource.Description)
	}

	if len(osResource.Subports) > 0 {
		subports := make([]*orcapplyconfigv1alpha1.SubportStatusApplyConfiguration, len(osResource.Subports))
		for i := range osResource.Subports {
			subportStatus := orcapplyconfigv1alpha1.SubportStatus().
				WithPortID(osResource.Subports[i].PortID).
				WithSegmentationType(osResource.Subports[i].SegmentationType).
				WithSegmentationID(int32(osResource.Subports[i].SegmentationID))
			subports[i] = subportStatus
		}
		resourceStatus.WithSubports(subports...)
	}

	statusApply.WithResource(resourceStatus)
}

