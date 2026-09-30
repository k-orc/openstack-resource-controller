/*
Copyright 2024 The ORC Authors.

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
	"encoding/json"
	"testing"

	"github.com/go-logr/logr"

	orcapplyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/pkg/clients/applyconfiguration/api/v1alpha1"
)

// Regression test: see the equivalent test in internal/controllers/network/status_test.go for the
// full explanation. WithCreatedAt/WithUpdatedAt were called unconditionally here too, so a zero
// CreatedAt/UpdatedAt from OpenStack (e.g. for a resource this project doesn't fully own) produced
// a literal JSON null in the status patch, which the CRD schema (type: string) rejects.
func TestApplyResourceStatus_zeroCreatedAtUpdatedAt(t *testing.T) {
	osResource := &osResourceT{
		ID:   "3fac9d0b-0e0e-4b0e-9b0e-000000000001",
		Name: "sg-owned-by-another-project",
	}

	statusApply := orcapplyconfigv1alpha1.SecurityGroupStatus()
	securityGroupStatusWriter{}.ApplyResourceStatus(logr.Discard(), osResource, statusApply)

	if statusApply.Resource == nil {
		t.Fatal("expected statusApply.Resource to be set")
	}
	if statusApply.Resource.CreatedAt != nil {
		t.Errorf("CreatedAt should be omitted for a zero time, got %v", *statusApply.Resource.CreatedAt)
	}
	if statusApply.Resource.UpdatedAt != nil {
		t.Errorf("UpdatedAt should be omitted for a zero time, got %v", *statusApply.Resource.UpdatedAt)
	}

	b, err := json.Marshal(statusApply.Resource)
	if err != nil {
		t.Fatalf("failed to marshal resource status: %v", err)
	}
	var asMap map[string]any
	if err := json.Unmarshal(b, &asMap); err != nil {
		t.Fatalf("failed to unmarshal resource status: %v", err)
	}
	if v, ok := asMap["createdAt"]; ok {
		t.Errorf(`expected "createdAt" to be absent, got present with value %v (json: %s)`, v, b)
	}
	if v, ok := asMap["updatedAt"]; ok {
		t.Errorf(`expected "updatedAt" to be absent, got present with value %v (json: %s)`, v, b)
	}
}
