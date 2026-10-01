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

package rbacpolicy

import (
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v3/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v3/internal/controllers/generic/progress"
	orcapplyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/pkg/clients/applyconfiguration/api/v1alpha1"
)

type rbacpolicyStatusWriter struct{}

type objectApplyT = orcapplyconfigv1alpha1.RBACPolicyApplyConfiguration
type statusApplyT = orcapplyconfigv1alpha1.RBACPolicyStatusApplyConfiguration

var _ interfaces.ResourceStatusWriter[*orcv1alpha1.RBACPolicy, *osResourceT, *objectApplyT, *statusApplyT] = rbacpolicyStatusWriter{}

func (rbacpolicyStatusWriter) GetApplyConfig(name, namespace string) *objectApplyT {
	return orcapplyconfigv1alpha1.RBACPolicy(name, namespace)
}

// Neutron's RBAC policy API has no intermediate/provisioning state at all (confirmed via
// gophercloud's RBACPolicy struct - no status field) - a policy exists fully formed the moment
// Create returns, so Available tracks existence alone, same as the generic nil-osResource cases
// every other ResourceAvailableStatus in this codebase already handles identically.
func (rbacpolicyStatusWriter) ResourceAvailableStatus(orcObject *orcv1alpha1.RBACPolicy, osResource *osResourceT) (metav1.ConditionStatus, progress.ReconcileStatus) {
	if osResource == nil {
		if orcObject.Status.ID == nil {
			return metav1.ConditionFalse, nil
		} else {
			return metav1.ConditionUnknown, nil
		}
	}
	return metav1.ConditionTrue, nil
}

func (rbacpolicyStatusWriter) ApplyResourceStatus(log logr.Logger, osResource *osResourceT, statusApply *statusApplyT) {
	resourceStatus := orcapplyconfigv1alpha1.RBACPolicyResourceStatus().
		WithNetworkID(osResource.ObjectID).
		WithAction(string(osResource.Action)).
		WithTargetProjectID(osResource.TargetTenant)

	if osResource.ProjectID != "" {
		resourceStatus.WithProjectID(osResource.ProjectID)
	}

	statusApply.WithResource(resourceStatus)
}
