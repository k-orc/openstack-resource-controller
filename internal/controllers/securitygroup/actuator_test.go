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
	"context"
	"errors"
	"iter"
	"slices"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/groups"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/rules"
	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v3/internal/osclients/mock"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestListOSResourcesForAdoption(t *testing.T) {
	listErr := errors.New("listing failed")
	candidates := []groups.SecGroup{
		{ID: "stateless", Stateful: false},
		{ID: "stateful", Stateful: true},
		{ID: "another-stateless", Stateful: false},
	}
	for _, tc := range []struct {
		name           string
		stateful       *bool
		candidates     []groups.SecGroup
		listErr        error
		resourceName   *orcv1alpha1.OpenStackName
		withProject    bool
		stopAfterFirst bool
		wantIDs        []string
	}{
		{
			name: "stateful candidates", stateful: ptr.To(true), candidates: candidates,
			wantIDs: []string{"stateful"},
		},
		{
			name: "stateless candidates", stateful: ptr.To(false), candidates: candidates,
			wantIDs: []string{"stateless", "another-stateless"},
		},
		{
			name: "unset statefulness", candidates: candidates,
			wantIDs: []string{"stateless", "stateful", "another-stateless"},
		},
		{
			name: "no matching candidates", stateful: ptr.To(true), candidates: candidates[:1],
		},
		{
			name: "empty listing", stateful: ptr.To(false),
		},
		{
			name: "initial list error", stateful: ptr.To(true), listErr: listErr,
		},
		{
			name: "error after filtered candidate", stateful: ptr.To(true),
			candidates: candidates[:1], listErr: listErr,
		},
		{
			name: "error after matching candidate", stateful: ptr.To(true),
			candidates: candidates, listErr: listErr, wantIDs: []string{"stateful"},
		},
		{
			name: "stop after first match", stateful: ptr.To(true),
			candidates: candidates, listErr: listErr, stopAfterFirst: true,
			wantIDs: []string{"stateful"},
		},
		{
			name: "preserve name and project filters", stateful: ptr.To(false),
			resourceName: ptr.To(orcv1alpha1.OpenStackName("overridden-name")), withProject: true,
			candidates: candidates, wantIDs: []string{"stateless", "another-stateless"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			networkClient := mock.NewMockNetworkClient(gomock.NewController(t))
			actuator := securityGroupActuator{osClient: networkClient}
			obj := &orcv1alpha1.SecurityGroup{
				ObjectMeta: metav1.ObjectMeta{Name: "object-name", Namespace: "namespace"},
				Spec: orcv1alpha1.SecurityGroupSpec{
					Resource: &orcv1alpha1.SecurityGroupResourceSpec{
						Name: tc.resourceName, Stateful: tc.stateful,
					},
				},
			}
			wantOpts := groups.ListOpts{Name: string(ptr.Deref(tc.resourceName, "object-name"))}
			if tc.withProject {
				scheme := runtime.NewScheme()
				if err := orcv1alpha1.AddToScheme(scheme); err != nil {
					t.Fatal(err)
				}
				project := &orcv1alpha1.Project{
					ObjectMeta: metav1.ObjectMeta{Name: "project", Namespace: obj.Namespace},
					Status: orcv1alpha1.ProjectStatus{
						ID: ptr.To("project-id"),
						Conditions: []metav1.Condition{{
							Type: orcv1alpha1.ConditionAvailable, Status: metav1.ConditionTrue,
						}},
					},
				}
				actuator.k8sClient = fake.NewClientBuilder().WithScheme(scheme).WithObjects(project).Build()
				obj.Spec.Resource.ProjectRef = ptr.To(orcv1alpha1.KubernetesNameRef(project.Name))
				wantOpts.ProjectID = *project.Status.ID
			}
			visited := 0
			var seq iter.Seq2[*groups.SecGroup, error] = func(yield func(*groups.SecGroup, error) bool) {
				for i := range tc.candidates {
					visited++
					if !yield(&tc.candidates[i], nil) {
						return
					}
				}
				if tc.listErr != nil {
					yield(nil, tc.listErr)
				}
			}
			// Stateful must be omitted from the server-side list options.
			networkClient.EXPECT().ListSecGroup(ctx, wantOpts).Return(seq)
			resourceIter, canAdopt := actuator.ListOSResourcesForAdoption(ctx, obj)
			if !canAdopt {
				t.Fatal("expected adoption to be supported")
			}
			var ids []string
			var gotErrors []error
			for group, err := range resourceIter {
				if err != nil {
					if group != nil {
						t.Errorf("error yielded with resource: %v", group)
					}
					gotErrors = append(gotErrors, err)
					continue
				}
				ids = append(ids, group.ID)
				if tc.stopAfterFirst {
					break
				}
			}
			if !slices.Equal(ids, tc.wantIDs) {
				t.Errorf("IDs = %v, want %v", ids, tc.wantIDs)
			}
			wantErrors := 0
			if tc.listErr != nil && !tc.stopAfterFirst {
				wantErrors = 1
			}
			if len(gotErrors) != wantErrors {
				t.Fatalf("errors = %v, want %d errors", gotErrors, wantErrors)
			}
			if wantErrors != 0 && !errors.Is(gotErrors[0], tc.listErr) {
				t.Errorf("error = %v, want %v", gotErrors[0], tc.listErr)
			}
			wantVisited := len(tc.candidates)
			if tc.stopAfterFirst {
				wantVisited = 2
			}
			if visited != wantVisited {
				t.Errorf("visited %d candidates, want %d", visited, wantVisited)
			}
			if tc.withProject {
				project := &orcv1alpha1.Project{}
				if err := actuator.k8sClient.Get(ctx, client.ObjectKey{Namespace: obj.Namespace, Name: "project"}, project); err != nil {
					t.Fatal(err)
				}
				if len(project.Finalizers) != 0 {
					t.Errorf("adoption added project finalizers: %v", project.Finalizers)
				}
			}
		})
	}
}

func TestNeedsUpdate(t *testing.T) {
	testCases := []struct {
		name         string
		updateOpts   groups.UpdateOpts
		expectChange bool
	}{
		{
			name:         "Empty base opts",
			updateOpts:   groups.UpdateOpts{},
			expectChange: false,
		},
		{
			name:         "Empty base opts with revision number",
			updateOpts:   groups.UpdateOpts{RevisionNumber: ptr.To(4)},
			expectChange: false,
		},
		{
			name:         "Updated opts",
			updateOpts:   groups.UpdateOpts{Name: "updated"},
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

func Test_securityGroupActuator_updateRules(t *testing.T) {
	const (
		groupID = "939da9ca-27c2-4fa6-881f-17f9038f8107"
		ruleID  = "cac8c6c9-6ab0-47d0-a472-978afe7300b9"
		ruleID2 = "617fee32-31fc-49fe-8e8c-ec4ab45ce508"
	)

	var (
		createError = errors.New("Test create error")
		deleteError = errors.New("Test delete error")
	)

	now := time.Now()

	osResourceWithRules := func(rules []rules.SecGroupRule) *groups.SecGroup {
		return &groups.SecGroup{
			ID:          groupID,
			Name:        "test-secgroup-name",
			Description: "test-secgroup-description",
			Rules:       rules,
			Stateful:    false,
			UpdatedAt:   now,
			CreatedAt:   now,
			ProjectID:   "40b8e1ec-2070-48c8-9ac2-ece92ad3d8b3",
			Tags:        []string{"tag1", "tag2"},
		}
	}

	orcObjectWithRules := func(rules []orcv1alpha1.SecurityGroupRule) *orcv1alpha1.SecurityGroup {
		return &orcv1alpha1.SecurityGroup{
			Spec: orcv1alpha1.SecurityGroupSpec{
				Resource: &orcv1alpha1.SecurityGroupResourceSpec{
					Rules: rules,
				},
			},
		}
	}

	tests := []struct {
		name           string
		orcObject      orcObjectPT
		osResource     *osResourceT
		expect         func(*mock.MockNetworkClientMockRecorder)
		wantReschedule bool
		wantErrs       []error
	}{
		{
			name:       "no rules exist, no rules defined",
			orcObject:  orcObjectWithRules(nil),
			osResource: osResourceWithRules(nil),
		},
		{
			name: "have no rules, want 1 rule",
			orcObject: orcObjectWithRules([]orcv1alpha1.SecurityGroupRule{
				{
					Description: ptr.To(orcv1alpha1.NeutronDescription("test-description")),
					Direction:   ptr.To(orcv1alpha1.RuleDirection("ingress")),
					Protocol:    ptr.To(orcv1alpha1.ProtocolTCP),
					Ethertype:   orcv1alpha1.EthertypeIPv4,
					PortRange: &orcv1alpha1.PortRangeSpec{
						Min: 1,
						Max: 2,
					},
				},
			}),
			osResource: osResourceWithRules(nil),
			expect: func(recorder *mock.MockNetworkClientMockRecorder) {
				createOpts := rules.CreateOpts{
					SecGroupID:   groupID,
					Direction:    "ingress",
					Description:  "test-description",
					EtherType:    "IPv4",
					PortRangeMin: 1,
					PortRangeMax: 2,
					Protocol:     "tcp",
				}
				recorder.CreateSecGroupRules(gomock.Any(), []rules.CreateOpts{createOpts}).Return(nil, nil)
			},
			wantReschedule: true,
		},
		{
			name: "1 rule is up to date",
			orcObject: orcObjectWithRules([]orcv1alpha1.SecurityGroupRule{
				{
					Description: ptr.To(orcv1alpha1.NeutronDescription("test-description")),
					Direction:   ptr.To(orcv1alpha1.RuleDirection("ingress")),
					Protocol:    ptr.To(orcv1alpha1.ProtocolTCP),
					Ethertype:   orcv1alpha1.EthertypeIPv4,
					PortRange: &orcv1alpha1.PortRangeSpec{
						Min: 1,
						Max: 2,
					},
				},
			}),
			osResource: osResourceWithRules([]rules.SecGroupRule{
				{
					ID:           ruleID,
					Direction:    "ingress",
					Description:  "test-description",
					EtherType:    "IPv4",
					SecGroupID:   groupID,
					PortRangeMin: 1,
					PortRangeMax: 2,
					Protocol:     "tcp",
				},
			}),
		},
		{
			name:      "have 1 rule, want none",
			orcObject: orcObjectWithRules(nil),
			osResource: osResourceWithRules([]rules.SecGroupRule{
				{
					ID:           ruleID,
					Direction:    "ingress",
					Description:  "test-description",
					EtherType:    "IPv4",
					SecGroupID:   groupID,
					PortRangeMin: 1,
					PortRangeMax: 2,
					Protocol:     "tcp",
				},
			}),
			expect: func(recorder *mock.MockNetworkClientMockRecorder) {
				recorder.DeleteSecGroupRule(gomock.Any(), ruleID).Return(nil)
			},
			wantReschedule: true,
		},
		{
			name: "have 1 rule, want different rule",
			orcObject: orcObjectWithRules([]orcv1alpha1.SecurityGroupRule{
				{
					Description: ptr.To(orcv1alpha1.NeutronDescription("egress rule")),
					Direction:   ptr.To(orcv1alpha1.RuleDirection("egress")),
					Protocol:    ptr.To(orcv1alpha1.ProtocolTCP),
					Ethertype:   orcv1alpha1.EthertypeIPv4,
					PortRange: &orcv1alpha1.PortRangeSpec{
						Min: 1,
						Max: 2,
					},
				},
			}),
			osResource: osResourceWithRules([]rules.SecGroupRule{
				{
					ID:           ruleID,
					Direction:    "ingress", // egress -> ingress
					Description:  "ingress rule",
					EtherType:    "IPv4",
					SecGroupID:   groupID,
					PortRangeMin: 1,
					PortRangeMax: 2,
					Protocol:     "tcp",
				},
			}),
			expect: func(recorder *mock.MockNetworkClientMockRecorder) {
				createOpts := rules.CreateOpts{
					SecGroupID:   groupID,
					Direction:    "egress",
					Description:  "egress rule",
					EtherType:    "IPv4",
					PortRangeMin: 1,
					PortRangeMax: 2,
					Protocol:     "tcp",
				}
				recorder.CreateSecGroupRules(gomock.Any(), []rules.CreateOpts{createOpts}).Return(nil, nil)
				recorder.DeleteSecGroupRule(gomock.Any(), ruleID).Return(nil)
			},
			wantReschedule: true,
		},
		{
			name: "have 1 rule, want 2 rules",
			orcObject: orcObjectWithRules([]orcv1alpha1.SecurityGroupRule{
				{
					Description: ptr.To(orcv1alpha1.NeutronDescription("ingress rule")),
					Direction:   ptr.To(orcv1alpha1.RuleDirection("ingress")),
					Protocol:    ptr.To(orcv1alpha1.ProtocolTCP),
					Ethertype:   orcv1alpha1.EthertypeIPv4,
					PortRange: &orcv1alpha1.PortRangeSpec{
						Min: 1,
						Max: 2,
					},
				},
				{
					Description: ptr.To(orcv1alpha1.NeutronDescription("egress rule")),
					Direction:   ptr.To(orcv1alpha1.RuleDirection("egress")),
					Protocol:    ptr.To(orcv1alpha1.ProtocolTCP),
					Ethertype:   orcv1alpha1.EthertypeIPv4,
					PortRange: &orcv1alpha1.PortRangeSpec{
						Min: 1,
						Max: 2,
					},
				},
			}),
			osResource: osResourceWithRules([]rules.SecGroupRule{
				{
					ID:           ruleID,
					Direction:    "ingress",
					Description:  "ingress rule",
					EtherType:    "IPv4",
					SecGroupID:   groupID,
					PortRangeMin: 1,
					PortRangeMax: 2,
					Protocol:     "tcp",
				},
			}),
			expect: func(recorder *mock.MockNetworkClientMockRecorder) {
				createOpts := rules.CreateOpts{
					SecGroupID:   groupID,
					Direction:    "egress",
					Description:  "egress rule",
					EtherType:    "IPv4",
					PortRangeMin: 1,
					PortRangeMax: 2,
					Protocol:     "tcp",
				}
				recorder.CreateSecGroupRules(gomock.Any(), []rules.CreateOpts{createOpts}).Return(nil, nil)
			},
			wantReschedule: true,
		},
		{
			// Regression test: reproduces a nil-pointer panic where updateRules indexed the
			// full resource.Rules slice with the loop variable from the filtered createRules
			// slice when copying PortRange. When an earlier-declared rule without a PortRange
			// (e.g. esp/icmp/vrrp) already matches live and is therefore excluded from
			// createRules, the same index into resource.Rules can land on that no-port-range
			// rule instead of the one actually being created, dereferencing a nil PortRange.
			name: "rule without PortRange already matches, later rule with PortRange needs creating",
			orcObject: orcObjectWithRules([]orcv1alpha1.SecurityGroupRule{
				{
					Description: ptr.To(orcv1alpha1.NeutronDescription("esp rule, no port range")),
					Direction:   ptr.To(orcv1alpha1.RuleDirection("ingress")),
					Protocol:    ptr.To(orcv1alpha1.ProtocolESP),
					Ethertype:   orcv1alpha1.EthertypeIPv4,
				},
				{
					Description: ptr.To(orcv1alpha1.NeutronDescription("ssh rule, needs creating")),
					Direction:   ptr.To(orcv1alpha1.RuleDirection("ingress")),
					Protocol:    ptr.To(orcv1alpha1.ProtocolTCP),
					Ethertype:   orcv1alpha1.EthertypeIPv4,
					PortRange: &orcv1alpha1.PortRangeSpec{
						Min: 22,
						Max: 22,
					},
				},
			}),
			osResource: osResourceWithRules([]rules.SecGroupRule{
				{
					ID:          ruleID,
					Direction:   "ingress",
					Description: "esp rule, no port range",
					EtherType:   "IPv4",
					SecGroupID:  groupID,
					Protocol:    "esp",
				},
			}),
			expect: func(recorder *mock.MockNetworkClientMockRecorder) {
				createOpts := rules.CreateOpts{
					SecGroupID:   groupID,
					Direction:    "ingress",
					Description:  "ssh rule, needs creating",
					EtherType:    "IPv4",
					PortRangeMin: 22,
					PortRangeMax: 22,
					Protocol:     "tcp",
				}
				recorder.CreateSecGroupRules(gomock.Any(), []rules.CreateOpts{createOpts}).Return(nil, nil)
			},
			wantReschedule: true,
		},
		{
			name: "delete should still be called if create fails",
			orcObject: orcObjectWithRules([]orcv1alpha1.SecurityGroupRule{
				{
					Description: ptr.To(orcv1alpha1.NeutronDescription("egress rule")),
					Direction:   ptr.To(orcv1alpha1.RuleDirection("egress")),
					Protocol:    ptr.To(orcv1alpha1.ProtocolTCP),
					Ethertype:   orcv1alpha1.EthertypeIPv4,
					PortRange: &orcv1alpha1.PortRangeSpec{
						Min: 1,
						Max: 2,
					},
				},
			}),
			osResource: osResourceWithRules([]rules.SecGroupRule{
				{
					ID:           ruleID,
					Direction:    "ingress", // egress -> ingress
					Description:  "ingress rule",
					EtherType:    "IPv4",
					SecGroupID:   groupID,
					PortRangeMin: 1,
					PortRangeMax: 2,
					Protocol:     "tcp",
				},
			}),
			expect: func(recorder *mock.MockNetworkClientMockRecorder) {
				createOpts := rules.CreateOpts{
					SecGroupID:   groupID,
					Direction:    "egress",
					Description:  "egress rule",
					EtherType:    "IPv4",
					PortRangeMin: 1,
					PortRangeMax: 2,
					Protocol:     "tcp",
				}
				recorder.CreateSecGroupRules(gomock.Any(), []rules.CreateOpts{createOpts}).Return(nil, createError)
				recorder.DeleteSecGroupRule(gomock.Any(), ruleID).Return(nil)
			},
			wantReschedule: true,
			wantErrs:       []error{createError},
		},
		{
			name: "all deletes should be called if one fails",
			orcObject: orcObjectWithRules([]orcv1alpha1.SecurityGroupRule{
				{
					Description: ptr.To(orcv1alpha1.NeutronDescription("egress rule")),
					Direction:   ptr.To(orcv1alpha1.RuleDirection("egress")),
					Protocol:    ptr.To(orcv1alpha1.ProtocolTCP),
					Ethertype:   orcv1alpha1.EthertypeIPv4,
					PortRange: &orcv1alpha1.PortRangeSpec{
						Min: 1,
						Max: 2,
					},
				},
			}),
			osResource: osResourceWithRules([]rules.SecGroupRule{
				{
					ID:           ruleID,
					Direction:    "ingress",
					Description:  "ingress rule 1",
					EtherType:    "IPv4",
					SecGroupID:   groupID,
					PortRangeMin: 1,
					PortRangeMax: 2,
					Protocol:     "tcp",
				},
				{
					ID:           ruleID2,
					Direction:    "ingress",
					Description:  "ingress rule 2",
					EtherType:    "IPv6",
					SecGroupID:   groupID,
					PortRangeMin: 1,
					PortRangeMax: 2,
					Protocol:     "tcp",
				},
			}),
			expect: func(recorder *mock.MockNetworkClientMockRecorder) {
				createOpts := rules.CreateOpts{
					SecGroupID:   groupID,
					Direction:    "egress",
					Description:  "egress rule",
					EtherType:    "IPv4",
					PortRangeMin: 1,
					PortRangeMax: 2,
					Protocol:     "tcp",
				}
				recorder.CreateSecGroupRules(gomock.Any(), []rules.CreateOpts{createOpts}).Return(nil, nil)
				recorder.DeleteSecGroupRule(gomock.Any(), ruleID).Return(deleteError)
				recorder.DeleteSecGroupRule(gomock.Any(), ruleID2).Return(nil)
			},
			wantReschedule: true,
			wantErrs:       []error{deleteError},
		},
		{
			name: "create and delete errors both reported",
			orcObject: orcObjectWithRules([]orcv1alpha1.SecurityGroupRule{
				{
					Description: ptr.To(orcv1alpha1.NeutronDescription("egress rule")),
					Direction:   ptr.To(orcv1alpha1.RuleDirection("egress")),
					Protocol:    ptr.To(orcv1alpha1.ProtocolTCP),
					Ethertype:   orcv1alpha1.EthertypeIPv4,
					PortRange: &orcv1alpha1.PortRangeSpec{
						Min: 1,
						Max: 2,
					},
				},
			}),
			osResource: osResourceWithRules([]rules.SecGroupRule{
				{
					ID:           ruleID,
					Direction:    "ingress",
					Description:  "ingress rule",
					EtherType:    "IPv4",
					SecGroupID:   groupID,
					PortRangeMin: 1,
					PortRangeMax: 2,
					Protocol:     "tcp",
				},
			}),
			expect: func(recorder *mock.MockNetworkClientMockRecorder) {
				createOpts := rules.CreateOpts{
					SecGroupID:   groupID,
					Direction:    "egress",
					Description:  "egress rule",
					EtherType:    "IPv4",
					PortRangeMin: 1,
					PortRangeMax: 2,
					Protocol:     "tcp",
				}
				recorder.CreateSecGroupRules(gomock.Any(), []rules.CreateOpts{createOpts}).Return(nil, createError)
				recorder.DeleteSecGroupRule(gomock.Any(), ruleID).Return(deleteError)
			},
			wantReschedule: true,
			wantErrs:       []error{createError, deleteError},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockctrl := gomock.NewController(t)
			networkClient := mock.NewMockNetworkClient(mockctrl)

			actuator := securityGroupActuator{
				osClient: networkClient,
			}

			recorder := networkClient.EXPECT()
			if tt.expect != nil {
				tt.expect(recorder)
			}

			reconcileStatus := actuator.updateRules(context.TODO(), tt.orcObject, tt.osResource)
			needsReschedule, err := reconcileStatus.NeedsReschedule()
			if len(tt.wantErrs) == 0 && err != nil {
				t.Errorf("securityGroupActuator.updateRules() error = %v, want 0 errors", err)
			}
			for _, wantErr := range tt.wantErrs {
				if !errors.Is(err, wantErr) {
					t.Errorf("securityGroupActuator.updateRules() error = %v, wantErr %v", err, wantErr)
				}
			}
			// How to assert that err doesn't contain any errors that we didn't want?

			if needsReschedule != tt.wantReschedule {
				t.Errorf("securityGroupActuator.updateRules() needsReschedule = %v, want %v", needsReschedule, tt.wantReschedule)
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
			resource := &orcv1alpha1.SecurityGroup{}
			resource.Name = "object-name"
			resource.Spec = orcv1alpha1.SecurityGroupSpec{
				Resource: &orcv1alpha1.SecurityGroupResourceSpec{Name: tt.newValue},
			}
			osResource := &groups.SecGroup{Name: tt.existingValue}

			updateOpts := groups.UpdateOpts{}
			handleNameUpdate(&updateOpts, resource, osResource)

			got, _ := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})

	}
}

func TestHandleDescriptionUpdate(t *testing.T) {
	ptrToDescription := ptr.To[orcv1alpha1.NeutronDescription]
	testCases := []struct {
		name          string
		newValue      *orcv1alpha1.NeutronDescription
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
			resource := &orcv1alpha1.SecurityGroupResourceSpec{Description: tt.newValue}
			osResource := &groups.SecGroup{Description: tt.existingValue}

			updateOpts := groups.UpdateOpts{}
			handleDescriptionUpdate(&updateOpts, resource, osResource)

			got, _ := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})

	}
}
