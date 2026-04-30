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

package loadbalancer

import (
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/progress"
	orcapplyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/pkg/clients/applyconfiguration/api/v1alpha1"
)

const (
	lbProvisioningStatusActive = "ACTIVE"
)

type objectApplyPT = *orcapplyconfigv1alpha1.LoadBalancerApplyConfiguration
type statusApplyPT = *orcapplyconfigv1alpha1.LoadBalancerStatusApplyConfiguration

type loadbalancerStatusWriter struct{}

var _ interfaces.ResourceStatusWriter[orcObjectPT, *osResourceT, objectApplyPT, statusApplyPT] = loadbalancerStatusWriter{}

func (loadbalancerStatusWriter) GetApplyConfig(name, namespace string) objectApplyPT {
	return orcapplyconfigv1alpha1.LoadBalancer(name, namespace)
}

func (loadbalancerStatusWriter) ResourceAvailableStatus(orcObject orcObjectPT, osResource *osResourceT) (metav1.ConditionStatus, progress.ReconcileStatus) {
	if osResource == nil {
		if orcObject.Status.ID == nil {
			return metav1.ConditionFalse, nil
		}
		return metav1.ConditionUnknown, nil
	}

	if osResource.ProvisioningStatus == lbProvisioningStatusActive {
		return metav1.ConditionTrue, nil
	}

	return metav1.ConditionFalse, nil
}

func (loadbalancerStatusWriter) ApplyResourceStatus(log logr.Logger, osResource *osResourceT, statusApply statusApplyPT) {
	resourceStatus := orcapplyconfigv1alpha1.LoadBalancerResourceStatus().
		WithName(osResource.Name).
		WithDescription(osResource.Description).
		WithAdminStateUp(osResource.AdminStateUp).
		WithProvisioningStatus(osResource.ProvisioningStatus).
		WithOperatingStatus(osResource.OperatingStatus).
		WithVIPAddress(osResource.VipAddress).
		WithVIPSubnetID(osResource.VipSubnetID).
		WithVIPNetworkID(osResource.VipNetworkID).
		WithVIPPortID(osResource.VipPortID).
		WithProvider(osResource.Provider).
		WithFlavorID(osResource.FlavorID).
		WithProjectID(osResource.ProjectID).
		WithTags(osResource.Tags...)

	statusApply.WithResource(resourceStatus)
}
