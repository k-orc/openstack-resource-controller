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

package lease

import (
	"time"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v3/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v3/internal/controllers/generic/progress"
	orcapplyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/pkg/clients/applyconfiguration/api/v1alpha1"
)

// Lease statuses reported by Blazar
const (
	LeaseStatusPending    = "PENDING"
	LeaseStatusActive     = "ACTIVE"
	LeaseStatusTerminated = "TERMINATED"
	LeaseStatusError      = "ERROR"
	LeaseStatusDeleting   = "DELETING"
)

type leaseStatusWriter struct{}

type objectApplyT = orcapplyconfigv1alpha1.LeaseApplyConfiguration
type statusApplyT = orcapplyconfigv1alpha1.LeaseStatusApplyConfiguration

var _ interfaces.ResourceStatusWriter[*orcv1alpha1.Lease, *osResourceT, *objectApplyT, *statusApplyT] = leaseStatusWriter{}

func (leaseStatusWriter) GetApplyConfig(name, namespace string) *objectApplyT {
	return orcapplyconfigv1alpha1.Lease(name, namespace)
}

func (leaseStatusWriter) ResourceAvailableStatus(orcObject *orcv1alpha1.Lease, osResource *osResourceT) (metav1.ConditionStatus, progress.ReconcileStatus) {
	if osResource == nil {
		if orcObject.Status.ID == nil {
			return metav1.ConditionFalse, nil
		} else {
			return metav1.ConditionUnknown, nil
		}
	}

	switch osResource.Status {
	case LeaseStatusActive:
		return metav1.ConditionTrue, nil
	case LeaseStatusTerminated, LeaseStatusError:
		// checkStatus reports these as terminal
		return metav1.ConditionFalse, nil
	case LeaseStatusPending:
		// A lease may start days after it was created. Wake up when it is
		// due rather than polling the whole time.
		return metav1.ConditionFalse, progress.WaitingOnOpenStack(progress.WaitingOnReady, pendingPollingPeriod(time.Until(osResource.StartDate)))
	default:
		return metav1.ConditionFalse, progress.WaitingOnOpenStack(progress.WaitingOnReady, leaseAvailablePollingPeriod)
	}
}

func pendingPollingPeriod(untilStart time.Duration) time.Duration {
	return min(max(untilStart, leaseAvailablePollingPeriod), leasePendingMaxPollingPeriod)
}

func (leaseStatusWriter) ApplyResourceStatus(log logr.Logger, osResource *osResourceT, statusApply *statusApplyT) {
	resourceStatus := orcapplyconfigv1alpha1.LeaseResourceStatus().
		WithName(osResource.Name).
		WithStatus(osResource.Status).
		WithDegraded(osResource.Degraded).
		WithProjectID(osResource.ProjectID).
		WithUserID(osResource.UserID).
		WithTrustID(osResource.TrustID).
		WithStartDate(metav1.NewTime(osResource.StartDate)).
		WithEndDate(metav1.NewTime(osResource.EndDate)).
		WithCreatedAt(metav1.NewTime(osResource.CreatedAt))

	if osResource.UpdatedAt != nil && !osResource.UpdatedAt.IsZero() {
		resourceStatus.WithUpdatedAt(metav1.NewTime(*osResource.UpdatedAt))
	}

	for i := range osResource.Reservations {
		reservation := &osResource.Reservations[i]
		reservationStatus := orcapplyconfigv1alpha1.LeaseReservationStatus().
			WithID(reservation.ID).
			WithResourceType(reservation.ResourceType).
			WithStatus(reservation.Status).
			WithResourceID(reservation.ResourceID).
			WithMissingResources(reservation.MissingResources).
			WithResourcesChanged(reservation.ResourcesChanged)

		// Blazar only returns the fields of the reservation's resource type
		if reservation.FlavorID != nil {
			reservationStatus.WithFlavorID(*reservation.FlavorID)
		}
		if reservation.ServerGroupID != nil {
			reservationStatus.WithServerGroupID(*reservation.ServerGroupID)
		}
		if reservation.HypervisorProperties != nil {
			reservationStatus.WithHypervisorProperties(*reservation.HypervisorProperties)
		}
		if reservation.ResourceProperties != nil {
			reservationStatus.WithResourceProperties(*reservation.ResourceProperties)
		}
		if reservation.BeforeEnd != nil {
			reservationStatus.WithBeforeEnd(*reservation.BeforeEnd)
		}
		if reservation.AggregateID != nil {
			reservationStatus.WithAggregateID(int32(*reservation.AggregateID))
		}
		if reservation.Affinity != nil {
			reservationStatus.WithAffinity(*reservation.Affinity)
		}
		if reservation.Min != nil {
			reservationStatus.WithMin(int32(*reservation.Min))
		}
		if reservation.Max != nil {
			reservationStatus.WithMax(int32(*reservation.Max))
		}
		if reservation.Amount != nil {
			reservationStatus.WithAmount(int32(*reservation.Amount))
		}
		if reservation.VCPUs != nil {
			reservationStatus.WithVcpus(int32(*reservation.VCPUs))
		}
		if reservation.MemoryMB != nil {
			reservationStatus.WithMemoryMB(int32(*reservation.MemoryMB))
		}
		if reservation.DiskGB != nil {
			reservationStatus.WithDiskGB(int32(*reservation.DiskGB))
		}
		resourceStatus.WithReservations(reservationStatus)
	}

	for i := range osResource.Events {
		event := &osResource.Events[i]
		resourceStatus.WithEvents(orcapplyconfigv1alpha1.LeaseEventStatus().
			WithID(event.ID).
			WithEventType(event.EventType).
			WithStatus(event.Status).
			WithTime(metav1.NewTime(event.Time)))
	}

	statusApply.WithResource(resourceStatus)
}
