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
	"time"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/progress"
	orcapplyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/pkg/clients/applyconfiguration/api/v1alpha1"
)

// Octavia LoadBalancer provisioning_status constants.
const (
	ProvisioningStatusActive        = "ACTIVE"
	ProvisioningStatusPendingCreate = "PENDING_CREATE"
	ProvisioningStatusPendingUpdate = "PENDING_UPDATE"
	ProvisioningStatusPendingDelete = "PENDING_DELETE"
	ProvisioningStatusError         = "ERROR"
)

// provisioningPollingPeriod is the frequency to poll when waiting for a
// load balancer to finish provisioning. LoadBalancers can take tens of
// seconds to provision, so 15 seconds is a reasonable polling interval.
const provisioningPollingPeriod = 15 * time.Second

type objectApplyPT = *orcapplyconfigv1alpha1.LoadBalancerApplyConfiguration
type statusApplyPT = *orcapplyconfigv1alpha1.LoadBalancerStatusApplyConfiguration

type loadbalancerStatusWriter struct{}

var _ interfaces.ResourceStatusWriter[orcObjectPT, *osResourceT, objectApplyPT, statusApplyPT] = loadbalancerStatusWriter{}

func (loadbalancerStatusWriter) GetApplyConfig(name, namespace string) objectApplyPT {
	return orcapplyconfigv1alpha1.LoadBalancer(name, namespace)
}

// ResourceAvailableStatus maps Octavia provisioning_status to Kubernetes
// Available condition status.
//
//   - ACTIVE → ConditionTrue (resource is fully provisioned and ready)
//   - PENDING_CREATE, PENDING_UPDATE, PENDING_DELETE → ConditionUnknown + requeue (transient)
//   - ERROR → ConditionFalse (terminal, manual intervention required)
//   - nil osResource, no ID → ConditionFalse (not yet created)
//   - nil osResource, has ID → ConditionUnknown (ID known but resource not yet fetched)
func (loadbalancerStatusWriter) ResourceAvailableStatus(orcObject orcObjectPT, osResource *osResourceT) (metav1.ConditionStatus, progress.ReconcileStatus) {
	if osResource == nil {
		if orcObject.Status.ID == nil {
			return metav1.ConditionFalse, nil
		}
		return metav1.ConditionUnknown, nil
	}

	switch osResource.ProvisioningStatus {
	case ProvisioningStatusActive:
		return metav1.ConditionTrue, nil

	case ProvisioningStatusPendingCreate, ProvisioningStatusPendingUpdate, ProvisioningStatusPendingDelete:
		// Transient states: wait for OpenStack to complete the operation.
		return metav1.ConditionUnknown, progress.WaitingOnOpenStack(progress.WaitingOnReady, provisioningPollingPeriod)

	case ProvisioningStatusError:
		// Terminal state: the load balancer is in an error state that
		// requires manual intervention in OpenStack to resolve.
		return metav1.ConditionFalse, nil

	default:
		// Unknown provisioning status: poll until we see a known state.
		return metav1.ConditionUnknown, progress.WaitingOnOpenStack(progress.WaitingOnReady, provisioningPollingPeriod)
	}
}

// ApplyResourceStatus populates the Kubernetes status.resource fields from
// the OpenStack LoadBalancer response.
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

	if !osResource.CreatedAt.IsZero() {
		resourceStatus.WithCreatedAt(metav1.NewTime(osResource.CreatedAt))
	}
	if !osResource.UpdatedAt.IsZero() {
		resourceStatus.WithUpdatedAt(metav1.NewTime(osResource.UpdatedAt))
	}

	statusApply.WithResource(resourceStatus)
}
