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

package loadbalancer

import (
	"testing"

	"github.com/go-logr/logr"

	orcapplyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/pkg/clients/applyconfiguration/api/v1alpha1"
)

func TestApplyResourceStatus_omitsEmptyFields(t *testing.T) {
	osResource := &osResourceT{ID: "5b2a2a7e-6b8f-4c0e-9d3c-000000000001", Name: "lb"}

	statusApply := orcapplyconfigv1alpha1.LoadBalancerStatus()
	loadbalancerStatusWriter{}.ApplyResourceStatus(logr.Discard(), osResource, statusApply)

	resource := statusApply.Resource
	if resource == nil {
		t.Fatal("expected statusApply.Resource to be set")
	}
	if resource.Description != nil {
		t.Errorf("Description should be omitted when empty, got %q", *resource.Description)
	}
	if resource.FlavorID != nil {
		t.Errorf("FlavorID should be omitted when empty, got %q", *resource.FlavorID)
	}
	if resource.AvailabilityZone != nil {
		t.Errorf("AvailabilityZone should be omitted when empty, got %q", *resource.AvailabilityZone)
	}
}

func TestApplyResourceStatus_setsFields(t *testing.T) {
	osResource := &osResourceT{
		ID: "5b2a2a7e-6b8f-4c0e-9d3c-000000000002", Name: "lb",
		Description: "desc", FlavorID: "flavor", AvailabilityZone: "az1",
	}

	statusApply := orcapplyconfigv1alpha1.LoadBalancerStatus()
	loadbalancerStatusWriter{}.ApplyResourceStatus(logr.Discard(), osResource, statusApply)

	resource := statusApply.Resource
	if resource == nil || resource.Description == nil || *resource.Description != "desc" ||
		resource.FlavorID == nil || *resource.FlavorID != "flavor" ||
		resource.AvailabilityZone == nil || *resource.AvailabilityZone != "az1" {
		t.Errorf("expected description, flavorID and availabilityZone to be set, got %+v", resource)
	}
}
