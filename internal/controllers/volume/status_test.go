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

package volume

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes"

	orcapplyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/pkg/clients/applyconfiguration/api/v1alpha1"
)

// Regression test: see the equivalent test in internal/controllers/network/status_test.go for the
// full explanation. WithCreatedAt was called unconditionally here too (WithUpdatedAt already had
// a guard - CreatedAt did not), so a zero CreatedAt from OpenStack (e.g. for a resource this
// project doesn't fully own) produced a literal JSON null in the status patch, which the CRD
// schema (type: string) rejects.
func TestApplyResourceStatus_zeroCreatedAt(t *testing.T) {
	osResource := &volumes.Volume{
		ID:   "3fac9d0b-0e0e-4b0e-9b0e-000000000001",
		Name: "volume-owned-by-another-project",
	}

	statusApply := orcapplyconfigv1alpha1.VolumeStatus()
	volumeStatusWriter{}.ApplyResourceStatus(logr.Discard(), osResource, statusApply)

	if statusApply.Resource == nil {
		t.Fatal("expected statusApply.Resource to be set")
	}
	if statusApply.Resource.CreatedAt != nil {
		t.Errorf("CreatedAt should be omitted for a zero time, got %v", *statusApply.Resource.CreatedAt)
	}
}
