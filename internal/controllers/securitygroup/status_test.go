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

package securitygroup

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/rules"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/api/v1alpha1"
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
}

// Regression test: a SecurityGroup with a deliberately empty rule list (e.g. a placeholder
// profile with no openings added yet - a real, supported pattern, not a hypothetical) could never
// reach Available. spec.resource.rules: [] is a non-nil, zero-length slice in Go, and
// ApplyResourceStatus never calls WithRules() when osResource.Rules is empty (its loop runs zero
// times), leaving status.resource.rules permanently nil - which the old check treated as "not
// ready yet" forever, regardless of how many reconciles passed. See status.go's own comment on
// ResourceAvailableStatus for the full explanation.
func Test_securityGroupStatusWriter_ResourceAvailableStatus_emptyRules(t *testing.T) {
	emptyRulesSpec := &orcv1alpha1.SecurityGroupResourceSpec{
		Rules: []orcv1alpha1.SecurityGroupRule{},
	}

	testCases := []struct {
		name          string
		orcObject     orcObjectPT
		osResource    *osResourceT
		wantAvailable metav1.ConditionStatus
		wantWaiting   bool
	}{
		{
			name: "empty spec.resource.rules, empty osResource.Rules - should be immediately Available",
			orcObject: &orcv1alpha1.SecurityGroup{
				Spec: orcv1alpha1.SecurityGroupSpec{Resource: emptyRulesSpec},
				// status.resource.rules has never been written - exactly the first-reconcile
				// state that triggered the bug, since nothing has called ApplyResourceStatus yet.
				Status: orcv1alpha1.SecurityGroupStatus{},
			},
			osResource:    &osResourceT{ID: "sg-empty", Rules: nil},
			wantAvailable: metav1.ConditionTrue,
			wantWaiting:   false,
		},
		{
			name: "non-empty spec.resource.rules, osResource still missing rules - should still wait (unchanged behavior)",
			orcObject: &orcv1alpha1.SecurityGroup{
				Spec: orcv1alpha1.SecurityGroupSpec{
					Resource: &orcv1alpha1.SecurityGroupResourceSpec{
						Rules: []orcv1alpha1.SecurityGroupRule{{}},
					},
				},
				Status: orcv1alpha1.SecurityGroupStatus{},
			},
			osResource:    &osResourceT{ID: "sg-pending", Rules: nil},
			wantAvailable: metav1.ConditionFalse,
			wantWaiting:   true,
		},
		{
			name: "non-empty spec.resource.rules matching osResource.Rules - should be Available",
			orcObject: &orcv1alpha1.SecurityGroup{
				Spec: orcv1alpha1.SecurityGroupSpec{
					Resource: &orcv1alpha1.SecurityGroupResourceSpec{
						Rules: []orcv1alpha1.SecurityGroupRule{{}},
					},
				},
				Status: orcv1alpha1.SecurityGroupStatus{},
			},
			osResource:    &osResourceT{ID: "sg-matched", Rules: []rules.SecGroupRule{{}}},
			wantAvailable: metav1.ConditionTrue,
			wantWaiting:   false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			gotAvailable, gotStatus := securityGroupStatusWriter{}.ResourceAvailableStatus(tt.orcObject, tt.osResource)
			if gotAvailable != tt.wantAvailable {
				t.Errorf("ResourceAvailableStatus() available = %v, want %v", gotAvailable, tt.wantAvailable)
			}
			needsReschedule, _ := gotStatus.NeedsReschedule()
			if needsReschedule != tt.wantWaiting {
				t.Errorf("ResourceAvailableStatus() needsReschedule = %v, want %v", needsReschedule, tt.wantWaiting)
			}
		})
	}
}
