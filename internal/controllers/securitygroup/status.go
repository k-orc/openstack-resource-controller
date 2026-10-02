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

package securitygroup

import (
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/k-orc/openstack-resource-controller/v3/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v3/internal/controllers/generic/progress"
	orcapplyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/pkg/clients/applyconfiguration/api/v1alpha1"
)

type objectApplyPT = *orcapplyconfigv1alpha1.SecurityGroupApplyConfiguration
type statusApplyPT = *orcapplyconfigv1alpha1.SecurityGroupStatusApplyConfiguration

type securityGroupStatusWriter struct{}

var _ interfaces.ResourceStatusWriter[orcObjectPT, *osResourceT, objectApplyPT, statusApplyPT] = securityGroupStatusWriter{}

func (securityGroupStatusWriter) GetApplyConfig(name, namespace string) objectApplyPT {
	return orcapplyconfigv1alpha1.SecurityGroup(name, namespace)
}

func (securityGroupStatusWriter) ResourceAvailableStatus(orcObject orcObjectPT, osResource *osResourceT) (metav1.ConditionStatus, progress.ReconcileStatus) {
	if osResource == nil {
		if orcObject.Status.ID == nil {
			return metav1.ConditionFalse, nil
		} else {
			return metav1.ConditionUnknown, nil
		}
	}

	resourceSpec := orcObject.Spec.Resource
	if resourceSpec != nil && len(resourceSpec.Rules) != len(osResource.Rules) {
		// The freshly-fetched OpenStack resource doesn't yet have as many rules as declared - a
		// create/update reconciler presumably just fired off the necessary Neutron calls, but
		// this reconcile's own read of the resource predates them taking effect. Wait and
		// recheck, rather than report Available prematurely.
		//
		// Deliberately compares against osResource (this reconcile's own fresh read), not
		// orcObject.Status.Resource.Rules (last reconcile's status write): a previous version of
		// this check compared against the status instead, and nil-checked it to decide whether to
		// wait - but an empty []SecurityGroupRule{} in spec.resource.rules is a non-nil slice in
		// Go, and ApplyResourceStatus below never calls WithRules() at all when osResource.Rules
		// is empty (its loop runs zero times), leaving status.resource.rules permanently nil. A
		// SecurityGroup with a deliberately empty rule list (e.g. a placeholder profile with no
		// openings added yet) could therefore never pass that nil check and would stay stuck
		// "Waiting for OpenStack resource to be ready" forever, 15s poll after 15s poll, despite
		// genuinely matching its spec. Comparing lengths against osResource directly has no such
		// nil/empty ambiguity and fixes this for the zero-rules case without changing behavior for
		// the non-zero case the original two checks were also covering.
		return metav1.ConditionFalse, progress.WaitingOnOpenStack(progress.WaitingOnReady, securityGroupAvailablePollingPeriod)
	}

	return metav1.ConditionTrue, nil
}

func (securityGroupStatusWriter) ApplyResourceStatus(log logr.Logger, osResource *osResourceT, statusApply statusApplyPT) {
	securitygroupResourceStatus := orcapplyconfigv1alpha1.SecurityGroupResourceStatus().
		WithName(osResource.Name).
		WithProjectID(osResource.ProjectID).
		WithTags(osResource.Tags...).
		WithStateful(osResource.Stateful)

	if !osResource.CreatedAt.IsZero() {
		securitygroupResourceStatus.WithCreatedAt(metav1.NewTime(osResource.CreatedAt))
	}
	if !osResource.UpdatedAt.IsZero() {
		securitygroupResourceStatus.WithUpdatedAt(metav1.NewTime(osResource.UpdatedAt))
	}

	if osResource.Description != "" {
		securitygroupResourceStatus.WithDescription(osResource.Description)
	}
	for i := range osResource.Rules {
		rule := &osResource.Rules[i]

		ruleStatus := orcapplyconfigv1alpha1.SecurityGroupRuleStatus().
			WithID(osResource.Rules[i].ID).
			WithDescription(osResource.Rules[i].Description).
			WithDirection(osResource.Rules[i].Direction).
			WithRemoteGroupID(osResource.Rules[i].RemoteGroupID).
			WithRemoteIPPrefix(osResource.Rules[i].RemoteIPPrefix).
			WithProtocol(osResource.Rules[i].Protocol).
			WithEthertype(osResource.Rules[i].EtherType)

		if rule.PortRangeMin != 0 || rule.PortRangeMax != 0 {
			ruleStatus.WithPortRange(orcapplyconfigv1alpha1.PortRangeStatus().
				WithMin(int32(rule.PortRangeMin)).
				WithMax(int32(rule.PortRangeMax)))
		}

		securitygroupResourceStatus.WithRules(ruleStatus)
	}

	statusApply.WithResource(securitygroupResourceStatus)
}
