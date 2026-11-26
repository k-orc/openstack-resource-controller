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

package swiftcontainer

import (
	"testing"

	"github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/containers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/api/v1alpha1"
)

func TestResourceAvailableStatus(t *testing.T) {
	testCases := []struct {
		name          string
		statusID      *string
		osResource    *osResourceT
		expected      metav1.ConditionStatus
		expectRefresh bool
	}{
		{name: "Not created", expected: metav1.ConditionFalse},
		{name: "Not found", statusID: ptr.To("container"), expected: metav1.ConditionUnknown},
		{name: "Created but not read", osResource: &osResourceT{Name: "container"}, expected: metav1.ConditionFalse, expectRefresh: true},
		{name: "Read", osResource: &osResourceT{Name: "container", Header: &containers.GetHeader{}}, expected: metav1.ConditionTrue},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			obj := &orcv1alpha1.SwiftContainer{}
			obj.Status.ID = tt.statusID

			got, rs := swiftcontainerStatusWriter{}.ResourceAvailableStatus(obj, tt.osResource)
			if got != tt.expected {
				t.Errorf("Expected: %s, got: %s", tt.expected, got)
			}
			if refresh, _ := rs.NeedsReschedule(); refresh != tt.expectRefresh {
				t.Errorf("Expected refresh: %v, got: %v", tt.expectRefresh, refresh)
			}
		})
	}
}
