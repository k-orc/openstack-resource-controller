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
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/groups"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/rules"
	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/generic/progress"
	"github.com/k-orc/openstack-resource-controller/internal/osclients/mock"
	"go.uber.org/mock/gomock"
	"k8s.io/utils/ptr"
)

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
		name       string
		orcObject  orcObjectPT
		osResource *osResourceT
		expect     func(*mock.MockNetworkClientMockRecorder)
		wantEvents []progress.ProgressStatus
		wantErrs   []error
	}{
		{
			name:       "ignore nil resource spec",
			orcObject:  &orcv1alpha1.SecurityGroup{},
			osResource: osResourceWithRules(nil),
		},
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
			wantEvents: []progress.ProgressStatus{
				progress.WaitingOnOpenStackUpdate(time.Second),
			},
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
			wantEvents: []progress.ProgressStatus{
				progress.WaitingOnOpenStackUpdate(time.Second),
			},
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
			wantEvents: []progress.ProgressStatus{
				progress.WaitingOnOpenStackUpdate(time.Second),
			},
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
			wantEvents: []progress.ProgressStatus{
				progress.WaitingOnOpenStackUpdate(time.Second),
			},
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
			wantEvents: []progress.ProgressStatus{
				progress.WaitingOnOpenStackUpdate(time.Second),
			},
			wantErrs: []error{createError},
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
			wantEvents: []progress.ProgressStatus{
				progress.WaitingOnOpenStackUpdate(time.Second),
			},
			wantErrs: []error{deleteError},
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
			wantEvents: []progress.ProgressStatus{
				progress.WaitingOnOpenStackUpdate(time.Second),
			},
			wantErrs: []error{createError, deleteError},
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

			gotWaitEvents, err := actuator.updateRules(context.TODO(), tt.orcObject, tt.osResource)
			if len(tt.wantErrs) == 0 && err != nil {
				t.Errorf("securityGroupActuator.updateRules() error = %v, want 0 errors", err)
			}
			for _, wantErr := range tt.wantErrs {
				if !errors.Is(err, wantErr) {
					t.Errorf("securityGroupActuator.updateRules() error = %v, wantErr %v", err, wantErr)
				}
			}
			// How to assert that err doesn't contain any errors that we didn't want?

			if !reflect.DeepEqual(gotWaitEvents, tt.wantEvents) {
				t.Errorf("securityGroupActuator.updateRules() waitEvents = %v, want %v", gotWaitEvents, tt.wantEvents)
			}
		})
	}
}
