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

package network

import (
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks"

	"github.com/k-orc/openstack-resource-controller/v3/internal/osclients"
	orcapplyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/pkg/clients/applyconfiguration/api/v1alpha1"
)

// Regression test for a status-write failure seen with a real, externally-owned network: when
// Neutron's response for a network omits created_at/updated_at (observed for a network not fully
// owned by the querying project), gophercloud leaves those fields as the zero time.Time. Every
// "With*" setter on the generated apply-configuration takes its argument by value and always
// stores a non-nil pointer, so calling WithCreatedAt/WithUpdatedAt unconditionally - as this file
// used to - produces a non-nil *metav1.Time wrapping a zero time.Time. metav1.Time's own
// MarshalJSON renders a zero time as the literal JSON null, and because the field is a non-nil
// pointer, encoding/json's `omitempty` does not suppress it. The result is a status patch
// containing "createdAt": null, which the CRD's structural schema (type: string) rejects on every
// reconcile - the resource can never report a status.resource.createdAt and therefore never
// becomes Available. sharenetwork's status writer already guards against this
// (`if !osResource.CreatedAt.IsZero()`); network's did not.
func TestApplyResourceStatus_zeroCreatedAtUpdatedAt(t *testing.T) {
	osResource := &osclients.NetworkExt{
		Network: networks.Network{
			ID:   "3fac9d0b-0e0e-4b0e-9b0e-000000000001",
			Name: "external-network-owned-by-another-project",
			// CreatedAt/UpdatedAt deliberately left zero - reproduces what this cloud's
			// Neutron actually returns for a network outside the querying project.
		},
	}

	statusApply := orcapplyconfigv1alpha1.NetworkStatus()
	networkStatusWriter{}.ApplyResourceStatus(logr.Discard(), osResource, statusApply)

	if statusApply.Resource == nil {
		t.Fatal("expected statusApply.Resource to be set")
	}

	if statusApply.Resource.CreatedAt != nil {
		t.Errorf("CreatedAt should be omitted for a zero time, got %v", *statusApply.Resource.CreatedAt)
	}
	if statusApply.Resource.UpdatedAt != nil {
		t.Errorf("UpdatedAt should be omitted for a zero time, got %v", *statusApply.Resource.UpdatedAt)
	}
}

// Non-zero CreatedAt/UpdatedAt should still be reported normally - guard against a fix that
// swallows real values along with zero ones.
func TestApplyResourceStatus_nonZeroCreatedAtUpdatedAt(t *testing.T) {
	created := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	updated := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	osResource := &osclients.NetworkExt{
		Network: networks.Network{
			ID:        "3fac9d0b-0e0e-4b0e-9b0e-000000000002",
			Name:      "normal-network",
			CreatedAt: created,
			UpdatedAt: updated,
		},
	}

	statusApply := orcapplyconfigv1alpha1.NetworkStatus()
	networkStatusWriter{}.ApplyResourceStatus(logr.Discard(), osResource, statusApply)

	if statusApply.Resource == nil {
		t.Fatal("expected statusApply.Resource to be set")
	}
	if statusApply.Resource.CreatedAt == nil || !statusApply.Resource.CreatedAt.Time.Equal(created) {
		t.Errorf("expected CreatedAt %v, got %v", created, statusApply.Resource.CreatedAt)
	}
	if statusApply.Resource.UpdatedAt == nil || !statusApply.Resource.UpdatedAt.Time.Equal(updated) {
		t.Errorf("expected UpdatedAt %v, got %v", updated, statusApply.Resource.UpdatedAt)
	}
}
