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

const (
	ShareStatusCreating         = "creating"
	ShareStatusAvailable        = "available"
	ShareStatusDeleting         = "deleting"
	ShareStatusError            = "error"
	ShareStatusErrorDeleting    = "error_deleting"
	ShareStatusManageStarting   = "manage_starting"
	ShareStatusManageError      = "manage_error"
	ShareStatusUnmanageStarting = "unmanage_starting"
	ShareStatusUnmanageError    = "unmanage_error"
	ShareStatusExtending        = "extending"
	ShareStatusExtendingError   = "extending_error"
	ShareStatusShrinking        = "shrinking"
	ShareStatusShrinkingError   = "shrinking_error"
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

	if osResource.Status == ShareStatusAvailable {
		return metav1.ConditionTrue, nil
	}

	// Terminal error states - don't poll
	if osResource.Status == ShareStatusError ||
		osResource.Status == ShareStatusErrorDeleting ||
		osResource.Status == ShareStatusManageError ||
		osResource.Status == ShareStatusUnmanageError ||
		osResource.Status == ShareStatusExtendingError ||
		osResource.Status == ShareStatusShrinkingError {
		return metav1.ConditionFalse, nil
	}

	// Otherwise we should continue to poll
	return metav1.ConditionFalse, progress.WaitingOnOpenStack(progress.WaitingOnReady, shareAvailablePollingPeriod)
}

func (shareStatusWriter) ApplyResourceStatus(log logr.Logger, osResource *osResourceT, statusApply *statusApplyT) {
	resourceStatus := orcapplyconfigv1alpha1.ShareResourceStatus().
		WithName(osResource.Name)

	if osResource.Description != "" {
		resourceStatus.WithDescription(osResource.Description)
	}

	// TODO: Add more fields after running make generate to create apply configurations
	// Fields to add: ShareProto, Status, Size, AvailabilityZone, IsPublic,
	// ExportLocations, Metadata, CreatedAt, ProjectID

	statusApply.WithResource(resourceStatus)
}
