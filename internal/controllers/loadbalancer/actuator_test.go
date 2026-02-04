/*
Copyright 2026 The ORC Authors.

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

	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/loadbalancers"
	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	"k8s.io/utils/ptr"
)

func TestNeedsUpdate(t *testing.T) {
	testCases := []struct {
		name         string
		updateOpts   loadbalancers.UpdateOpts
		expectChange bool
	}{
		{
			name:         "Empty base opts",
			updateOpts:   loadbalancers.UpdateOpts{},
			expectChange: false,
		},
		{
			name:         "Updated opts",
			updateOpts:   loadbalancers.UpdateOpts{Name: ptr.To("updated")},
			expectChange: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := needsUpdate(tt.updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})
	}
}

func TestHandleNameUpdate(t *testing.T) {
	ptrToName := ptr.To[orcv1alpha1.OpenStackName]
	testCases := []struct {
		name          string
		newValue      *orcv1alpha1.OpenStackName
		existingValue string
		expectChange  bool
	}{
		{name: "Identical", newValue: ptrToName("name"), existingValue: "name", expectChange: false},
		{name: "Different", newValue: ptrToName("new-name"), existingValue: "name", expectChange: true},
		{name: "No value provided, existing is identical to object name", newValue: nil, existingValue: "object-name", expectChange: false},
		{name: "No value provided, existing is different from object name", newValue: nil, existingValue: "different-from-object-name", expectChange: true},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resource := &orcv1alpha1.LoadBalancer{}
			resource.Name = "object-name"
			resource.Spec = orcv1alpha1.LoadBalancerSpec{
				Resource: &orcv1alpha1.LoadBalancerResourceSpec{Name: tt.newValue},
			}
			osResource := &osResourceT{Name: tt.existingValue}

			updateOpts := loadbalancers.UpdateOpts{}
			handleNameUpdate(&updateOpts, resource, osResource)

			got, _ := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})

	}
}

func TestHandleDescriptionUpdate(t *testing.T) {
	ptrToDescription := ptr.To[string]
	testCases := []struct {
		name          string
		newValue      *string
		existingValue string
		expectChange  bool
	}{
		{name: "Identical", newValue: ptrToDescription("desc"), existingValue: "desc", expectChange: false},
		{name: "Different", newValue: ptrToDescription("new-desc"), existingValue: "desc", expectChange: true},
		{name: "No value provided, existing is set", newValue: nil, existingValue: "desc", expectChange: true},
		{name: "No value provided, existing is empty", newValue: nil, existingValue: "", expectChange: false},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resource := &orcv1alpha1.LoadBalancerResourceSpec{Description: tt.newValue}
			osResource := &osResourceT{Description: tt.existingValue}

			updateOpts := loadbalancers.UpdateOpts{}
			handleDescriptionUpdate(&updateOpts, resource, osResource)

			got, _ := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})

	}
}

func TestHandleAdminStateUpdate(t *testing.T) {
	ptrToBool := ptr.To[bool]
	testCases := []struct {
		name          string
		newValue      *bool
		existingValue bool
		expectChange  bool
	}{
		{name: "Identical true", newValue: ptrToBool(true), existingValue: true, expectChange: false},
		{name: "Identical false", newValue: ptrToBool(false), existingValue: false, expectChange: false},
		{name: "Different true to false", newValue: ptrToBool(false), existingValue: true, expectChange: true},
		{name: "Different false to true", newValue: ptrToBool(true), existingValue: false, expectChange: true},
		{name: "No value provided, existing is set to false", newValue: nil, existingValue: false, expectChange: true},
		{name: "No value provided, existing is default (true)", newValue: nil, existingValue: true, expectChange: false},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resource := &orcv1alpha1.LoadBalancerResourceSpec{AdminStateUp: tt.newValue}
			osResource := &osResourceT{AdminStateUp: tt.existingValue}

			updateOpts := loadbalancers.UpdateOpts{}
			handleAdminStateUpdate(&updateOpts, resource, osResource)

			got, _ := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})
	}
}

func TestHandleTagsUpdate(t *testing.T) {
	testCases := []struct {
		name          string
		newValue      []orcv1alpha1.LoadBalancerTag
		existingValue []string
		expectChange  bool
	}{
		{name: "Identical empty", newValue: nil, existingValue: nil, expectChange: false},
		{name: "Identical single", newValue: []orcv1alpha1.LoadBalancerTag{"tag1"}, existingValue: []string{"tag1"}, expectChange: false},
		{name: "Identical multiple", newValue: []orcv1alpha1.LoadBalancerTag{"tag1", "tag2"}, existingValue: []string{"tag1", "tag2"}, expectChange: false},
		{name: "Identical different order", newValue: []orcv1alpha1.LoadBalancerTag{"tag2", "tag1"}, existingValue: []string{"tag1", "tag2"}, expectChange: false},
		{name: "Different add tag", newValue: []orcv1alpha1.LoadBalancerTag{"tag1", "tag2"}, existingValue: []string{"tag1"}, expectChange: true},
		{name: "Different remove tag", newValue: []orcv1alpha1.LoadBalancerTag{"tag1"}, existingValue: []string{"tag1", "tag2"}, expectChange: true},
		{name: "Different replace tag", newValue: []orcv1alpha1.LoadBalancerTag{"tag1", "tag3"}, existingValue: []string{"tag1", "tag2"}, expectChange: true},
		{name: "Add tags to empty", newValue: []orcv1alpha1.LoadBalancerTag{"tag1"}, existingValue: nil, expectChange: true},
		{name: "Remove all tags", newValue: nil, existingValue: []string{"tag1"}, expectChange: true},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resource := &orcv1alpha1.LoadBalancerResourceSpec{Tags: tt.newValue}
			osResource := &osResourceT{Tags: tt.existingValue}

			updateOpts := loadbalancers.UpdateOpts{}
			handleTagsUpdate(&updateOpts, resource, osResource)

			got, _ := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})
	}
}
