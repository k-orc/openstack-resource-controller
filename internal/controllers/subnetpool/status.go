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

package subnetpool

import (
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v3/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v3/internal/controllers/generic/progress"
	orcapplyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/pkg/clients/applyconfiguration/api/v1alpha1"
)

type subnetpoolStatusWriter struct{}

type (
	objectApplyT = orcapplyconfigv1alpha1.SubnetPoolApplyConfiguration
	statusApplyT = orcapplyconfigv1alpha1.SubnetPoolStatusApplyConfiguration
)

var _ interfaces.ResourceStatusWriter[*orcv1alpha1.SubnetPool, *osResourceT, *objectApplyT, *statusApplyT] = subnetpoolStatusWriter{}

func (subnetpoolStatusWriter) GetApplyConfig(name, namespace string) *objectApplyT {
	return orcapplyconfigv1alpha1.SubnetPool(name, namespace)
}

func (subnetpoolStatusWriter) ResourceAvailableStatus(orcObject *orcv1alpha1.SubnetPool, osResource *osResourceT) (metav1.ConditionStatus, progress.ReconcileStatus) {
	if osResource == nil {
		if orcObject.Status.ID == nil {
			return metav1.ConditionFalse, nil
		} else {
			return metav1.ConditionUnknown, nil
		}
	}
	return metav1.ConditionTrue, nil
}

func (subnetpoolStatusWriter) ApplyResourceStatus(log logr.Logger, osResource *osResourceT, statusApply *statusApplyT) {
	resourceStatus := orcapplyconfigv1alpha1.SubnetPoolResourceStatus().
		WithName(osResource.Name).
		WithPrefixes(osResource.Prefixes...).
		WithMinPrefixLength(int32(osResource.MinPrefixLen)).
		WithMaxPrefixLength(int32(osResource.MaxPrefixLen)).
		WithDefaultPrefixLength(int32(osResource.DefaultPrefixLen)).
		WithIsDefault(osResource.IsDefault).
		WithShared(osResource.Shared).
		WithIPVersion(int32(osResource.IPversion)).
		WithRevisionNumber(int64(osResource.RevisionNumber)).
		WithProjectID(osResource.ProjectID).
		WithTags(osResource.Tags...)

	if !osResource.CreatedAt.IsZero() {
		resourceStatus.WithCreatedAt(metav1.NewTime(osResource.CreatedAt))
	}

	if !osResource.UpdatedAt.IsZero() {
		resourceStatus.WithUpdatedAt(metav1.NewTime(osResource.UpdatedAt))
	}

	if osResource.Description != "" {
		resourceStatus.WithDescription(osResource.Description)
	}

	if osResource.AddressScopeID != "" {
		resourceStatus.WithAddressScopeID(osResource.AddressScopeID)
	}

	statusApply.WithResource(resourceStatus)
}
