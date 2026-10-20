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
	"context"
	"errors"
	"iter"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/gophercloud/gophercloud/v2/openstack/reservation/v1/leases"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v3/internal/osclients/mock"
	orcerrors "github.com/k-orc/openstack-resource-controller/v3/internal/util/errors"
	orcapplyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/pkg/clients/applyconfiguration/api/v1alpha1"
)

func TestReservationOpts(t *testing.T) {
	testCases := []struct {
		name        string
		reservation orcv1alpha1.LeaseReservation
		expected    leases.ReservationOptsBuilder
	}{
		{
			name: "Host",
			reservation: orcv1alpha1.LeaseReservation{
				Host: &orcv1alpha1.LeaseHostReservation{
					Min:                  1,
					Max:                  2,
					HypervisorProperties: ptr.To(`[">=", "$vcpus", "4"]`),
				},
			},
			expected: leases.HostReservationOpts{
				Min:                  1,
				Max:                  2,
				HypervisorProperties: `[">=", "$vcpus", "4"]`,
			},
		},
		{
			name: "Instance",
			reservation: orcv1alpha1.LeaseReservation{
				Instance: &orcv1alpha1.LeaseInstanceReservation{
					Amount:             3,
					Vcpus:              2,
					MemoryMB:           4096,
					DiskGB:             ptr.To[int32](0),
					Affinity:           ptr.To(false),
					ResourceProperties: ptr.To(`["==", "$gpu", "a100"]`),
				},
			},
			expected: leases.InstanceReservationOpts{
				Amount:             3,
				VCPUs:              2,
				MemoryMB:           4096,
				DiskGB:             0,
				Affinity:           ptr.To(false),
				ResourceProperties: `["==", "$gpu", "a100"]`,
			},
		},
		{
			name: "Flavor instance",
			reservation: orcv1alpha1.LeaseReservation{
				FlavorInstance: &orcv1alpha1.LeaseFlavorInstanceReservation{
					Amount:    2,
					FlavorRef: "my-flavor",
				},
			},
			// The flavor ID comes from the resolved ORC Flavor, not from the
			// name of the reference
			expected: leases.FlavorInstanceReservationOpts{
				Amount:   2,
				FlavorID: "flavor-id",
			},
		},
	}

	flavorIDs := map[orcv1alpha1.KubernetesNameRef]string{"my-flavor": "flavor-id"}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			got, err := reservationOpts(&tt.reservation, flavorIDs).ToReservationMap()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			expected, err := tt.expected.ToReservationMap()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			for k, v := range expected {
				if ptrVal, ok := v.(*bool); ok {
					if gotVal, _ := got[k].(*bool); ptr.Deref(gotVal, true) != ptr.Deref(ptrVal, true) {
						t.Errorf("%s: expected %v, got %v", k, ptr.Deref(ptrVal, true), ptr.Deref(gotVal, true))
					}
					continue
				}
				if got[k] != v {
					t.Errorf("%s: expected %v, got %v", k, v, got[k])
				}
			}
		})
	}
}

func TestPendingPollingPeriod(t *testing.T) {
	testCases := []struct {
		name       string
		untilStart time.Duration
		expected   time.Duration
	}{
		{name: "Start date passed", untilStart: -time.Minute, expected: leaseAvailablePollingPeriod},
		{name: "Starting soon", untilStart: time.Second, expected: leaseAvailablePollingPeriod},
		{name: "Starting within the maximum", untilStart: 5 * time.Minute, expected: 5 * time.Minute},
		{name: "Starting in days", untilStart: 72 * time.Hour, expected: leasePendingMaxPollingPeriod},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			if got := pendingPollingPeriod(tt.untilStart); got != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}

func TestCheckStatus(t *testing.T) {
	testCases := []struct {
		status       string
		wantTerminal bool
	}{
		{status: LeaseStatusPending},
		{status: LeaseStatusActive},
		{status: LeaseStatusTerminated, wantTerminal: true},
		{status: LeaseStatusError, wantTerminal: true},
	}

	for _, tt := range testCases {
		t.Run(tt.status, func(t *testing.T) {
			reconcileStatus := leaseActuator{}.checkStatus(context.TODO(), nil, &osResourceT{Status: tt.status})
			if isTerminal(reconcileStatus.GetError()) != tt.wantTerminal {
				t.Errorf("expected terminal: %v, got: %v", tt.wantTerminal, reconcileStatus.GetError())
			}
		})
	}
}

func TestCreateResourceNameTooLong(t *testing.T) {
	obj := &orcv1alpha1.Lease{}
	obj.Name = strings.Repeat("a", leaseNameMaxLength+1)
	obj.Spec.Resource = &orcv1alpha1.LeaseResourceSpec{}

	_, reconcileStatus := leaseActuator{}.CreateResource(context.TODO(), obj)
	if !isTerminal(reconcileStatus.GetError()) {
		t.Errorf("expected a terminal error, got: %v", reconcileStatus.GetError())
	}
}

func isTerminal(err error) bool {
	var terminalErr *orcerrors.TerminalError
	return errors.As(err, &terminalErr)
}

func TestListOSResourcesForAdoption(t *testing.T) {
	startDate := time.Date(2030, 1, 1, 10, 0, 0, 0, time.UTC)
	endDate := time.Date(2030, 1, 1, 11, 0, 0, 0, time.UTC)

	osLease := func(id, name string, start, end time.Time, resourceTypes ...string) *osResourceT {
		lease := &osResourceT{ID: id, Name: name, StartDate: start, EndDate: end}
		for _, resourceType := range resourceTypes {
			lease.Reservations = append(lease.Reservations, leases.Reservation{ResourceType: resourceType})
		}
		return lease
	}

	osLeases := []*osResourceT{
		osLease("match", "lease", startDate, endDate, leases.ResourceTypeInstance, leases.ResourceTypeHost),
		osLease("other-name", "other", startDate, endDate, leases.ResourceTypeHost, leases.ResourceTypeInstance),
		osLease("other-start", "lease", startDate.Add(time.Hour), endDate, leases.ResourceTypeHost, leases.ResourceTypeInstance),
		osLease("other-end", "lease", startDate, endDate.Add(time.Hour), leases.ResourceTypeHost, leases.ResourceTypeInstance),
		osLease("other-types", "lease", startDate, endDate, leases.ResourceTypeHost, leases.ResourceTypeHost),
		osLease("fewer-reservations", "lease", startDate, endDate, leases.ResourceTypeHost),
	}

	obj := &orcv1alpha1.Lease{}
	obj.Name = "lease"
	obj.Spec.Resource = &orcv1alpha1.LeaseResourceSpec{
		// Seconds are truncated by Blazar
		StartDate: ptr.To(metav1.NewTime(startDate.Add(30 * time.Second))),
		EndDate:   metav1.NewTime(endDate),
		Reservations: []orcv1alpha1.LeaseReservation{
			{Host: &orcv1alpha1.LeaseHostReservation{Min: 1, Max: 1}},
			{Instance: &orcv1alpha1.LeaseInstanceReservation{Amount: 1, Vcpus: 1, MemoryMB: 128, DiskGB: ptr.To[int32](1)}},
		},
	}

	mockctrl := gomock.NewController(t)
	leaseClient := mock.NewMockLeaseClient(mockctrl)
	leaseClient.EXPECT().ListLeases(gomock.Any(), gomock.Any()).Return(
		iter.Seq2[*osResourceT, error](func(yield func(*osResourceT, error) bool) {
			for _, lease := range osLeases {
				if !yield(lease, nil) {
					return
				}
			}
		}))

	results, ok := leaseActuator{osClient: leaseClient}.ListOSResourcesForAdoption(context.TODO(), obj)
	if !ok {
		t.Fatal("expected adoption to be possible")
	}

	var got []string
	for lease, err := range results {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got = append(got, lease.ID)
	}
	if len(got) != 1 || got[0] != "match" {
		t.Errorf("expected only the matching lease, got %v", got)
	}
}

func TestApplyResourceStatusReservations(t *testing.T) {
	osResource := &osResourceT{
		Reservations: []leases.Reservation{
			{
				ID:                   "host-reservation",
				ResourceID:           "host-resource",
				ResourceType:         leases.ResourceTypeHost,
				Min:                  ptr.To(1),
				Max:                  ptr.To(2),
				HypervisorProperties: ptr.To(`[">=", "$vcpus", "4"]`),
				ResourceProperties:   ptr.To(""),
				BeforeEnd:            ptr.To("default"),
			},
			{
				ID:                 "instance-reservation",
				ResourceID:         "instance-resource",
				ResourceType:       leases.ResourceTypeInstance,
				AggregateID:        ptr.To(7),
				MissingResources:   true,
				FlavorID:           ptr.To("instance-reservation"),
				ServerGroupID:      ptr.To("server-group"),
				Amount:             ptr.To(3),
				VCPUs:              ptr.To(2),
				MemoryMB:           ptr.To(4096),
				DiskGB:             ptr.To(0),
				Affinity:           ptr.To(false),
				ResourceProperties: ptr.To(`["==", "$gpu", "a100"]`),
			},
		},
	}

	statusApply := orcapplyconfigv1alpha1.LeaseStatus()
	leaseStatusWriter{}.ApplyResourceStatus(logr.Discard(), osResource, statusApply)

	reservations := statusApply.Resource.Reservations
	if len(reservations) != 2 {
		t.Fatalf("expected 2 reservations, got %d", len(reservations))
	}
	// A host reservation has no flavor or server group, and must not report
	// empty IDs for them
	if reservations[0].FlavorID != nil || reservations[0].ServerGroupID != nil {
		t.Errorf("expected no flavor or server group for a host reservation, got %v and %v",
			reservations[0].FlavorID, reservations[0].ServerGroupID)
	}
	if ptr.Deref(reservations[1].FlavorID, "") != "instance-reservation" ||
		ptr.Deref(reservations[1].ServerGroupID, "") != "server-group" {
		t.Errorf("expected flavor and server group for an instance reservation, got %v and %v",
			reservations[1].FlavorID, reservations[1].ServerGroupID)
	}

	host, instance := reservations[0], reservations[1]
	for name, check := range map[string]bool{
		"host min":                    ptr.Deref(host.Min, 0) == 1,
		"host max":                    ptr.Deref(host.Max, 0) == 2,
		"host hypervisorProperties":   ptr.Deref(host.HypervisorProperties, "") == `[">=", "$vcpus", "4"]`,
		"host has no resourceProps":   host.ResourceProperties == nil,
		"host resourceID":             ptr.Deref(host.ResourceID, "") == "host-resource",
		"host beforeEnd":              ptr.Deref(host.BeforeEnd, "") == "default",
		"host has no aggregateID":     host.AggregateID == nil,
		"host has no amount":          host.Amount == nil,
		"host missingResources":       !ptr.Deref(host.MissingResources, true),
		"instance amount":             ptr.Deref(instance.Amount, 0) == 3,
		"instance vcpus":              ptr.Deref(instance.Vcpus, 0) == 2,
		"instance memoryMB":           ptr.Deref(instance.MemoryMB, 0) == 4096,
		"instance diskGB of 0":        instance.DiskGB != nil && *instance.DiskGB == 0,
		"instance affinity":           instance.Affinity != nil && !*instance.Affinity,
		"instance resourceProperties": ptr.Deref(instance.ResourceProperties, "") == `["==", "$gpu", "a100"]`,
		"instance missingResources":   ptr.Deref(instance.MissingResources, false),
		"instance has no min":         instance.Min == nil,
		"instance resourceID":         ptr.Deref(instance.ResourceID, "") == "instance-resource",
		"instance aggregateID":        ptr.Deref(instance.AggregateID, 0) == 7,
		"instance has no beforeEnd":   instance.BeforeEnd == nil,
		"instance has no hypervisor":  instance.HypervisorProperties == nil,
	} {
		if !check {
			t.Errorf("unexpected status for %s", name)
		}
	}
}

func TestApplyResourceStatusDates(t *testing.T) {
	createdAt := time.Date(2030, 1, 1, 9, 0, 0, 0, time.UTC)

	testCases := []struct {
		name          string
		updatedAt     *time.Time
		wantUpdatedAt bool
	}{
		{
			name: "Never updated",
		},
		{
			name:          "Updated",
			updatedAt:     &createdAt,
			wantUpdatedAt: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			osResource := &osResourceT{
				StartDate: createdAt.Add(time.Hour),
				EndDate:   createdAt.Add(2 * time.Hour),
				CreatedAt: createdAt,
				UpdatedAt: tt.updatedAt,
			}
			statusApply := orcapplyconfigv1alpha1.LeaseStatus()
			leaseStatusWriter{}.ApplyResourceStatus(logr.Discard(), osResource, statusApply)

			resource := statusApply.Resource
			for field, want := range map[string]struct {
				got  *metav1.Time
				want time.Time
			}{
				"startDate": {resource.StartDate, osResource.StartDate},
				"endDate":   {resource.EndDate, osResource.EndDate},
				"createdAt": {resource.CreatedAt, osResource.CreatedAt},
			} {
				if want.got == nil || !want.got.Equal(&metav1.Time{Time: want.want}) {
					t.Errorf("%s: expected %v, got %v", field, want.want, want.got)
				}
			}
			if (resource.UpdatedAt != nil) != tt.wantUpdatedAt {
				t.Errorf("updatedAt: expected set: %v, got %v", tt.wantUpdatedAt, resource.UpdatedAt)
			}
		})
	}
}

func TestApplyResourceStatusEvents(t *testing.T) {
	start := time.Date(2030, 1, 1, 10, 0, 0, 0, time.UTC)
	osResource := &osResourceT{
		TrustID: "trust",
		Events: []leases.Event{
			{ID: "start", EventType: "start_lease", Status: "DONE", Time: start},
			{ID: "end", EventType: "end_lease", Status: "UNDONE", Time: start.Add(time.Hour)},
		},
	}

	statusApply := orcapplyconfigv1alpha1.LeaseStatus()
	leaseStatusWriter{}.ApplyResourceStatus(logr.Discard(), osResource, statusApply)

	resource := statusApply.Resource
	if ptr.Deref(resource.TrustID, "") != "trust" {
		t.Errorf("trustID: expected %q, got %v", "trust", resource.TrustID)
	}
	if len(resource.Events) != len(osResource.Events) {
		t.Fatalf("expected %d events, got %d", len(osResource.Events), len(resource.Events))
	}
	for i, want := range osResource.Events {
		got := resource.Events[i]
		if ptr.Deref(got.ID, "") != want.ID ||
			ptr.Deref(got.EventType, "") != want.EventType ||
			ptr.Deref(got.Status, "") != want.Status ||
			got.Time == nil || !got.Time.Equal(&metav1.Time{Time: want.Time}) {
			t.Errorf("event %d: expected %+v, got id=%v type=%v status=%v time=%v",
				i, want, got.ID, got.EventType, got.Status, got.Time)
		}
	}
}

func TestSpecReservationTypes(t *testing.T) {
	resourceSpec := &orcv1alpha1.LeaseResourceSpec{
		Reservations: []orcv1alpha1.LeaseReservation{
			{FlavorInstance: &orcv1alpha1.LeaseFlavorInstanceReservation{Amount: 1, FlavorRef: "flavor"}},
			{Instance: &orcv1alpha1.LeaseInstanceReservation{Amount: 1, Vcpus: 1, MemoryMB: 128, DiskGB: ptr.To[int32](1)}},
			{Host: &orcv1alpha1.LeaseHostReservation{Min: 1, Max: 1}},
		},
	}

	got := specReservationTypes(resourceSpec)
	expected := []string{leases.ResourceTypeFlavorInstance, leases.ResourceTypeHost, leases.ResourceTypeInstance}
	if !slices.Equal(got, expected) {
		t.Errorf("expected %v, got %v", expected, got)
	}
}
