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

package roleassignment

import (
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/progress"
	orcapplyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/pkg/clients/applyconfiguration/api/v1alpha1"
)

type roleassignmentStatusWriter struct{}

type objectApplyT = orcapplyconfigv1alpha1.RoleAssignmentApplyConfiguration
type statusApplyT = orcapplyconfigv1alpha1.RoleAssignmentStatusApplyConfiguration

var _ interfaces.ResourceStatusWriter[*orcv1alpha1.RoleAssignment, *osResourceT, *objectApplyT, *statusApplyT] = roleassignmentStatusWriter{}

func (roleassignmentStatusWriter) GetApplyConfig(name, namespace string) *objectApplyT {
	return orcapplyconfigv1alpha1.RoleAssignment(name, namespace)
}

// ResourceAvailableStatus returns the availability status of the role assignment.
// Role assignments don't have Status.ID, so we just check if osResource exists.
func (roleassignmentStatusWriter) ResourceAvailableStatus(orcObject *orcv1alpha1.RoleAssignment, osResource *osResourceT) (metav1.ConditionStatus, progress.ReconcileStatus) {
	if osResource == nil {
		// Check if we have any status IDs set (indicates we may have created it but can't find it)
		if orcObject.Status.Resource != nil &&
			(orcObject.Status.Resource.RoleID != "" ||
				orcObject.Status.Resource.UserID != "" ||
				orcObject.Status.Resource.GroupID != "" ||
				orcObject.Status.Resource.ProjectID != "" ||
				orcObject.Status.Resource.DomainID != "") {
			return metav1.ConditionUnknown, nil
		}
		return metav1.ConditionFalse, nil
	}
	return metav1.ConditionTrue, nil
}

// ApplyResourceStatus extracts the role assignment details and applies them to status.
func (roleassignmentStatusWriter) ApplyResourceStatus(log logr.Logger, osResource *osResourceT, statusApply *statusApplyT) {
	resourceStatus := orcapplyconfigv1alpha1.RoleAssignmentResourceStatus()

	// Extract role ID
	if osResource.Role.ID != "" {
		resourceStatus.WithRoleID(osResource.Role.ID)
	}

	// Extract actor ID (user XOR group)
	if osResource.User.ID != "" {
		resourceStatus.WithUserID(osResource.User.ID)
	}
	if osResource.Group.ID != "" {
		resourceStatus.WithGroupID(osResource.Group.ID)
	}

	// Extract scope ID (project XOR domain)
	if osResource.Scope.Project.ID != "" {
		resourceStatus.WithProjectID(osResource.Scope.Project.ID)
	}
	if osResource.Scope.Domain.ID != "" {
		resourceStatus.WithDomainID(osResource.Scope.Domain.ID)
	}

	statusApply.WithResource(resourceStatus)
}
