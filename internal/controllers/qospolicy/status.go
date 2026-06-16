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

package qospolicy

import (
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/progress"
	orcapplyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/pkg/clients/applyconfiguration/api/v1alpha1"
)

type qospolicyStatusWriter struct{}

type objectApplyT = orcapplyconfigv1alpha1.QosPolicyApplyConfiguration
type statusApplyT = orcapplyconfigv1alpha1.QosPolicyStatusApplyConfiguration

var _ interfaces.ResourceStatusWriter[*orcv1alpha1.QosPolicy, *osResourceT, *objectApplyT, *statusApplyT] = qospolicyStatusWriter{}

func (qospolicyStatusWriter) GetApplyConfig(name, namespace string) *objectApplyT {
	return orcapplyconfigv1alpha1.QosPolicy(name, namespace)
}

func (qospolicyStatusWriter) ResourceAvailableStatus(orcObject *orcv1alpha1.QosPolicy, osResource *osResourceT) (metav1.ConditionStatus, progress.ReconcileStatus) {
	if osResource == nil {
		if orcObject.Status.ID == nil {
			return metav1.ConditionFalse, nil
		} else {
			return metav1.ConditionUnknown, nil
		}
	}
	return metav1.ConditionTrue, nil
}

func (qospolicyStatusWriter) ApplyResourceStatus(log logr.Logger, osResource *osResourceT, statusApply *statusApplyT) {
	resourceStatus := orcapplyconfigv1alpha1.QosPolicyResourceStatus().
		WithName(osResource.Name).
		WithProjectID(osResource.ProjectID).
		WithTags(osResource.Tags...).
		WithShared(osResource.Shared).
		WithRevisionNumber(int64(osResource.RevisionNumber)).
		WithCreatedAt(metav1.NewTime(osResource.CreatedAt)).
		WithUpdatedAt(metav1.NewTime(osResource.UpdatedAt))

	if osResource.Description != "" {
		resourceStatus.WithDescription(osResource.Description)
	}

	// TODO(follow-up): map QoS rules (osResource.Rules) into the typed
	// *Rules status fields. Rules are returned as untyped []map[string]any
	// and are better fetched via the dedicated qos/rules APIs. This is
	// planned together with rule reconciliation in the actuator; note that
	// gophercloud lacks bindings for the two packet-rate rule types.

	statusApply.WithResource(resourceStatus)
}
