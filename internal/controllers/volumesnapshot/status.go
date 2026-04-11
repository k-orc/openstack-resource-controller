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

package volumesnapshot

import (
	"sort"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/progress"
	orcerrors "github.com/k-orc/openstack-resource-controller/v2/internal/util/errors"
	orcapplyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/pkg/clients/applyconfiguration/api/v1alpha1"
)

const (
	SnapshotStatusAvailable = "available"
	SnapshotStatusDeleting  = "deleting"
	SnapshotStatusError     = "error"
)

type volumesnapshotStatusWriter struct{}

type objectApplyT = orcapplyconfigv1alpha1.VolumeSnapshotApplyConfiguration
type statusApplyT = orcapplyconfigv1alpha1.VolumeSnapshotStatusApplyConfiguration

var _ interfaces.ResourceStatusWriter[*orcv1alpha1.VolumeSnapshot, *osResourceT, *objectApplyT, *statusApplyT] = volumesnapshotStatusWriter{}

func (volumesnapshotStatusWriter) GetApplyConfig(name, namespace string) *objectApplyT {
	return orcapplyconfigv1alpha1.VolumeSnapshot(name, namespace)
}

func (volumesnapshotStatusWriter) ResourceAvailableStatus(orcObject *orcv1alpha1.VolumeSnapshot, osResource *osResourceT) (metav1.ConditionStatus, progress.ReconcileStatus) {
	if osResource == nil {
		if orcObject.Status.ID == nil {
			return metav1.ConditionFalse, nil
		} else {
			return metav1.ConditionUnknown, nil
		}
	}
	if osResource.Status == SnapshotStatusAvailable {
		return metav1.ConditionTrue, nil
	}
	if osResource.Status == SnapshotStatusError {
		return metav1.ConditionFalse, progress.WrapError(
			orcerrors.Terminal(
				orcv1alpha1.ConditionReasonUnrecoverableError,
				"OpenStack volume snapshot is in error state",
			),
		)
	}

	// Otherwise we should continue to poll
	return metav1.ConditionFalse, progress.WaitingOnOpenStack(progress.WaitingOnReady, volumesnapshotAvailablePollingPeriod)
}

func (volumesnapshotStatusWriter) ApplyResourceStatus(log logr.Logger, osResource *osResourceT, statusApply *statusApplyT) {
	resourceStatus := orcapplyconfigv1alpha1.VolumeSnapshotResourceStatus().
		WithName(osResource.Name).
		WithVolumeID(osResource.VolumeID).
		WithStatus(osResource.Status).
		WithSize(int32(osResource.Size)).
		WithConsumesQuota(osResource.ConsumesQuota)

	if !osResource.CreatedAt.IsZero() {
		resourceStatus.WithCreatedAt(metav1.NewTime(osResource.CreatedAt))
	}

	if !osResource.UpdatedAt.IsZero() {
		resourceStatus.WithUpdatedAt(metav1.NewTime(osResource.UpdatedAt))
	}

	if osResource.Description != "" {
		resourceStatus.WithDescription(osResource.Description)
	}

	if osResource.Progress != "" {
		resourceStatus.WithProgress(osResource.Progress)
	}

	if osResource.ProjectID != "" {
		resourceStatus.WithProjectID(osResource.ProjectID)
	}

	if osResource.UserID != "" {
		resourceStatus.WithUserID(osResource.UserID)
	}

	if osResource.GroupSnapshotID != "" {
		resourceStatus.WithGroupSnapshotID(osResource.GroupSnapshotID)
	}

	metadataKeys := make([]string, 0, len(osResource.Metadata))
	for k := range osResource.Metadata {
		metadataKeys = append(metadataKeys, k)
	}
	sort.Strings(metadataKeys)
	for _, k := range metadataKeys {
		resourceStatus.WithMetadata(orcapplyconfigv1alpha1.VolumeSnapshotMetadataStatus().
			WithName(k).
			WithValue(osResource.Metadata[k]))
	}

	statusApply.WithResource(resourceStatus)
}
