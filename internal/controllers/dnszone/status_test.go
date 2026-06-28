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

package dnszone

import (
	"errors"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/gophercloud/gophercloud/v2/openstack/dns/v2/zones"
	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	orcerrors "github.com/k-orc/openstack-resource-controller/v2/internal/util/errors"
	orcapplyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/pkg/clients/applyconfiguration/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestResourceAvailableStatus(t *testing.T) {
	writer := dnsZoneStatusWriter{}

	tests := []struct {
		name           string
		orcObject      *orcv1alpha1.DNSZone
		osResource     *zones.Zone
		expectedStatus metav1.ConditionStatus
		expectRequeue  time.Duration
		expectTerminal bool
	}{
		{
			name: "osResource is nil, Status.ID is nil",
			orcObject: &orcv1alpha1.DNSZone{
				Status: orcv1alpha1.DNSZoneStatus{
					ID: nil,
				},
			},
			osResource:     nil,
			expectedStatus: metav1.ConditionFalse,
			expectRequeue:  0,
			expectTerminal: false,
		},
		{
			name: "osResource is nil, Status.ID is set",
			orcObject: &orcv1alpha1.DNSZone{
				Status: orcv1alpha1.DNSZoneStatus{
					ID: ptr.To("some-id"),
				},
			},
			osResource:     nil,
			expectedStatus: metav1.ConditionUnknown,
			expectRequeue:  0,
			expectTerminal: false,
		},
		{
			name:           "zone is ACTIVE",
			orcObject:      &orcv1alpha1.DNSZone{},
			osResource:     &zones.Zone{Status: "ACTIVE"},
			expectedStatus: metav1.ConditionTrue,
			expectRequeue:  0,
			expectTerminal: false,
		},
		{
			name:           "zone is PENDING",
			orcObject:      &orcv1alpha1.DNSZone{},
			osResource:     &zones.Zone{Status: "PENDING"},
			expectedStatus: metav1.ConditionFalse,
			expectRequeue:  15 * time.Second,
			expectTerminal: false,
		},
		{
			name:           "zone is ERROR",
			orcObject:      &orcv1alpha1.DNSZone{},
			osResource:     &zones.Zone{Status: "ERROR"},
			expectedStatus: metav1.ConditionFalse,
			expectRequeue:  0,
			expectTerminal: true,
		},
		{
			name:           "zone has unknown status",
			orcObject:      &orcv1alpha1.DNSZone{},
			osResource:     &zones.Zone{Status: "UNKNOWN_STATUS"},
			expectedStatus: metav1.ConditionFalse,
			expectRequeue:  15 * time.Second,
			expectTerminal: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, rs := writer.ResourceAvailableStatus(tt.orcObject, tt.osResource)
			if status != tt.expectedStatus {
				t.Errorf("expected status %v, got %v", tt.expectedStatus, status)
			}

			if rs == nil {
				if tt.expectRequeue != 0 || tt.expectTerminal {
					t.Errorf("expected non-nil ReconcileStatus")
				}
				return
			}

			if rs.GetRequeue() != tt.expectRequeue {
				t.Errorf("expected requeue %v, got %v", tt.expectRequeue, rs.GetRequeue())
			}

			err := rs.GetError()
			var terminalError *orcerrors.TerminalError
			hasTerminal := errors.As(err, &terminalError)
			if hasTerminal != tt.expectTerminal {
				t.Errorf("expected terminal error %v, got %v (err: %v)", tt.expectTerminal, hasTerminal, err)
			}
		})
	}
}

func TestApplyResourceStatus(t *testing.T) {
	writer := dnsZoneStatusWriter{}

	now := time.Now().UTC()
	osResource := &zones.Zone{
		Name:          testZoneName,
		Email:         "admin@example.com",
		Description:   "A test DNS zone",
		TTL:           3600,
		Type:          "SECONDARY",
		Status:        "ACTIVE",
		Masters:       []string{"192.0.2.1", "192.0.2.2"},
		TransferredAt: now,
	}

	statusApply := orcapplyconfigv1alpha1.DNSZoneStatus()
	writer.ApplyResourceStatus(logr.Discard(), osResource, statusApply)

	if statusApply.Resource == nil {
		t.Fatal("expected Resource in apply configuration to be non-nil")
	}

	res := statusApply.Resource
	if res.Name == nil || *res.Name != testZoneName {
		t.Errorf("expected name 'example.com.', got %v", res.Name)
	}
	if res.Email == nil || *res.Email != "admin@example.com" {
		t.Errorf("expected email 'admin@example.com', got %v", res.Email)
	}
	if res.Description == nil || *res.Description != "A test DNS zone" {
		t.Errorf("expected description 'A test DNS zone', got %v", res.Description)
	}
	if res.TTL == nil || *res.TTL != 3600 {
		t.Errorf("expected TTL 3600, got %v", res.TTL)
	}
	if res.Type == nil || *res.Type != "SECONDARY" {
		t.Errorf("expected type 'SECONDARY', got %v", res.Type)
	}
	if len(res.Masters) != 2 || res.Masters[0] != "192.0.2.1" || res.Masters[1] != "192.0.2.2" {
		t.Errorf("expected masters ['192.0.2.1', '192.0.2.2'], got %v", res.Masters)
	}
	if res.TransferredAt == nil || !res.TransferredAt.Time.Equal(now) {
		t.Errorf("expected transferredAt %v, got %v", now, res.TransferredAt)
	}
	if res.Status == nil || *res.Status != "ACTIVE" {
		t.Errorf("expected status 'ACTIVE', got %v", res.Status)
	}
}

func TestApplyResourceStatus_EmptyFields(t *testing.T) {
	writer := dnsZoneStatusWriter{}

	osResource := &zones.Zone{
		Name: testZoneName,
	}

	statusApply := orcapplyconfigv1alpha1.DNSZoneStatus()
	writer.ApplyResourceStatus(logr.Discard(), osResource, statusApply)

	if statusApply.Resource == nil {
		t.Fatal("expected Resource in apply configuration to be non-nil")
	}

	res := statusApply.Resource
	if res.Name == nil || *res.Name != testZoneName {
		t.Errorf("expected name 'example.com.', got %v", res.Name)
	}
	if res.Email != nil {
		t.Errorf("expected Email to be nil, got %v", res.Email)
	}
	if res.Description != nil {
		t.Errorf("expected Description to be nil, got %v", res.Description)
	}
	if res.TTL != nil {
		t.Errorf("expected TTL to be nil, got %v", res.TTL)
	}
	if res.Type != nil {
		t.Errorf("expected Type to be nil, got %v", res.Type)
	}
	if res.Status != nil {
		t.Errorf("expected Status to be nil, got %v", res.Status)
	}
}

func TestGetApplyConfig(t *testing.T) {
	writer := dnsZoneStatusWriter{}
	config := writer.GetApplyConfig("test-name", "test-namespace")
	if config == nil {
		t.Fatal("expected GetApplyConfig to return non-nil config")
	}
	if config.Name == nil || *config.Name != "test-name" {
		t.Errorf("expected Name to be 'test-name', got %v", config.Name)
	}
	if config.Namespace == nil || *config.Namespace != "test-namespace" {
		t.Errorf("expected Namespace to be 'test-namespace', got %v", config.Namespace)
	}
}
