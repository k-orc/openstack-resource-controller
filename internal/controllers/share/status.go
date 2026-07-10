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

package share

import (
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/progress"
	orcapplyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/pkg/clients/applyconfiguration/api/v1alpha1"
)

// Manila share status constants
// Based on https://docs.openstack.org/manila/latest/contributor/share_status.html
const (
	ShareStatusCreating            = "creating"
	ShareStatusAvailable           = "available"
	ShareStatusDeleting            = "deleting"
	ShareStatusError               = "error"
	ShareStatusErrorDeleting       = "error_deleting"
	ShareStatusManageStarting      = "manage_starting"
	ShareStatusManageError         = "manage_error"
	ShareStatusUnmanageStarting    = "unmanage_starting"
	ShareStatusUnmanageError       = "unmanage_error"
	ShareStatusExtending           = "extending"
	ShareStatusExtendingError      = "extending_error"
	ShareStatusShrinking           = "shrinking"
	ShareStatusShrinkingError      = "shrinking_error"
	ShareStatusMigrating           = "migrating"
	ShareStatusMigratingTo         = "migrating_to"
	ShareStatusRevertingToSnapshot = "reverting_to_snapshot"
	ShareStatusRevertingError      = "reverting_error"
)

type shareStatusWriter struct{}

type objectApplyT = orcapplyconfigv1alpha1.ShareApplyConfiguration
type statusApplyT = orcapplyconfigv1alpha1.ShareStatusApplyConfiguration

var _ interfaces.ResourceStatusWriter[*orcv1alpha1.Share, *osResourceT, *objectApplyT, *statusApplyT] = shareStatusWriter{}

func (shareStatusWriter) GetApplyConfig(name, namespace string) *objectApplyT {
	return orcapplyconfigv1alpha1.Share(name, namespace)
}

func (shareStatusWriter) ResourceAvailableStatus(orcObject *orcv1alpha1.Share, osResource *osResourceT) (metav1.ConditionStatus, progress.ReconcileStatus) {
	if osResource == nil {
		if orcObject.Status.ID == nil {
			return metav1.ConditionFalse, nil
		} else {
			return metav1.ConditionUnknown, nil
		}
	}

	// Share is available when it's in available status
	if osResource.Status == ShareStatusAvailable {
		return metav1.ConditionTrue, nil
	}

	// Error states are terminal - won't become available without intervention
	switch osResource.Status {
	case ShareStatusError, ShareStatusErrorDeleting, ShareStatusManageError,
		ShareStatusUnmanageError, ShareStatusExtendingError, ShareStatusShrinkingError,
		ShareStatusRevertingError:
		return metav1.ConditionFalse, nil
	}

	// Otherwise we're in a transitional state, continue polling
	return metav1.ConditionFalse, progress.WaitingOnOpenStack(progress.WaitingOnReady, shareAvailablePollingPeriod)
}

func (shareStatusWriter) ApplyResourceStatus(log logr.Logger, osResource *osResourceT, statusApply *statusApplyT) {
	resourceStatus := orcapplyconfigv1alpha1.ShareResourceStatus().
		WithName(osResource.Name).
		WithStatus(osResource.Status).
		WithShareProto(osResource.ShareProto).
		WithSize(int32(osResource.Size))

	if osResource.Description != "" {
		resourceStatus.WithDescription(osResource.Description)
	}

	if osResource.AvailabilityZone != "" {
		resourceStatus.WithAvailabilityZone(osResource.AvailabilityZone)
	}

	if osResource.ShareType != "" {
		resourceStatus.WithShareType(osResource.ShareType)
	}

	if osResource.ShareTypeName != "" {
		resourceStatus.WithShareTypeName(osResource.ShareTypeName)
	}

	if osResource.ShareNetworkID != "" {
		resourceStatus.WithShareNetworkID(osResource.ShareNetworkID)
	}

	resourceStatus.WithIsPublic(osResource.IsPublic)

	if len(osResource.Metadata) > 0 {
		resourceStatus.WithMetadata(osResource.Metadata)
	}

	// Note: Export locations need to be fetched separately via ListExportLocations API
	// They are not included in the basic Share struct from Get/List operations
	// We'll skip them for now and can add them later as a separate reconciler if needed

	statusApply.WithResource(resourceStatus)
}
