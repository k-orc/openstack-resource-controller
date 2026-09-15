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

package limit

import (
	"testing"

	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/limits"
	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	"k8s.io/utils/ptr"
)

func TestNeedsUpdate(t *testing.T) {
	testCases := []struct {
		name         string
		updateOpts   limits.UpdateOpts
		expectChange bool
	}{
		{
			name:         "Empty base opts",
			updateOpts:   limits.UpdateOpts{},
			expectChange: false,
		},
		{
			name:         "Updated opts with description",
			updateOpts:   limits.UpdateOpts{Description: ptr.To("updated")},
			expectChange: true,
		},
		{
			name:         "Updated opts with resourceLimit",
			updateOpts:   limits.UpdateOpts{ResourceLimit: ptr.To(-1)},
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
			resource := &orcv1alpha1.LimitResourceSpec{Description: tt.newValue}
			osResource := &osResourceT{Description: tt.existingValue}

			updateOpts := limits.UpdateOpts{}
			handleDescriptionUpdate(&updateOpts, resource, osResource)

			got, _ := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})

	}
}

func TestHandleResourceLimitUpdate(t *testing.T) {
	testCases := []struct {
		name          string
		newValue      int32
		existingValue int
		expectChange  bool
	}{
		{name: "Identical", newValue: -1, existingValue: -1, expectChange: false},
		{name: "Different", newValue: -1, existingValue: 10, expectChange: true},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resource := &orcv1alpha1.LimitResourceSpec{ResourceLimit: tt.newValue}
			osResource := &osResourceT{ResourceLimit: tt.existingValue}

			updateOpts := limits.UpdateOpts{}
			handleResourceLimitUpdate(&updateOpts, resource, osResource)

			got, _ := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})
	}
}

func TestValidateUpdate(t *testing.T) {
	testCases := []struct {
		name        string
		resource    *orcv1alpha1.LimitResourceSpec
		osResource  *osResourceT
		expectError error
	}{
		{
			name: "Update domainRef",
			resource: &orcv1alpha1.LimitResourceSpec{
				DomainRef: ptr.To(orcv1alpha1.KubernetesNameRef("domain-ref")),
			},
			osResource: &osResourceT{
				ProjectID: "12312312312",
			},
			expectError: errInvalidDomainRefUpdate,
		},
		{
			name: "Update projectRef",
			resource: &orcv1alpha1.LimitResourceSpec{
				ProjectRef: ptr.To(orcv1alpha1.KubernetesNameRef("project-ref")),
			},
			osResource: &osResourceT{
				DomainID: "default",
			},
			expectError: errInvalidProjectRefUpdate,
		},
		{
			name: "Normal update",
			resource: &orcv1alpha1.LimitResourceSpec{
				ProjectRef: ptr.To(orcv1alpha1.KubernetesNameRef("project-ref")),
			},
			osResource: &osResourceT{
				ProjectID: "12312312312",
			},
			expectError: nil,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			got := validateUpdate(tt.resource, tt.osResource)

			if got != tt.expectError {
				t.Errorf("Expected error: %v, got: %v", tt.expectError, got)
			}
		})
	}
}
