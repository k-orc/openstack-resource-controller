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

package rbacpolicy

import (
	"context"
	"errors"
	"testing"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/rbacpolicies"
	"go.uber.org/mock/gomock"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v3/internal/osclients/mock"
)

// updateResource only has one mutable field to detect changes on (targetProjectID - Neutron's
// UpdateOpts has no other field, see rbacpolicy_types.go's own comment on why networkRef/action
// are immutable instead). These tests cover that comparison directly, the same scope as every
// other controller's update-detection unit tests in this codebase.
func Test_rbacpolicyActuator_updateResource(t *testing.T) {
	const (
		policyID = "939da9ca-27c2-4fa6-881f-17f9038f8107"
	)

	updateError := errors.New("test update error")

	orcObjectWithTarget := func(targetProjectID string) orcObjectPT {
		return &orcv1alpha1.RBACPolicy{
			Spec: orcv1alpha1.RBACPolicySpec{
				Resource: &orcv1alpha1.RBACPolicyResourceSpec{
					Action:          orcv1alpha1.RBACPolicyActionAccessShared,
					TargetProjectID: targetProjectID,
				},
			},
		}
	}

	osResourceWithTarget := func(targetProjectID string) *osResourceT {
		return &rbacpolicies.RBACPolicy{
			ID:           policyID,
			Action:       rbacpolicies.ActionAccessShared,
			TargetTenant: targetProjectID,
		}
	}

	tests := []struct {
		name           string
		orcObject      orcObjectPT
		osResource     *osResourceT
		expect         func(*mock.MockNetworkClientMockRecorder)
		wantReschedule bool
		wantErr        error
	}{
		{
			name:       "target project unchanged, no update call",
			orcObject:  orcObjectWithTarget("project-a"),
			osResource: osResourceWithTarget("project-a"),
		},
		{
			name:       "target project changed, calls UpdateRBACPolicy",
			orcObject:  orcObjectWithTarget("project-b"),
			osResource: osResourceWithTarget("project-a"),
			expect: func(recorder *mock.MockNetworkClientMockRecorder) {
				recorder.UpdateRBACPolicy(gomock.Any(), policyID, rbacpolicies.UpdateOpts{TargetTenant: "project-b"}).
					Return(nil, nil)
			},
			wantReschedule: true,
		},
		{
			name:       "update error is propagated",
			orcObject:  orcObjectWithTarget("project-b"),
			osResource: osResourceWithTarget("project-a"),
			expect: func(recorder *mock.MockNetworkClientMockRecorder) {
				recorder.UpdateRBACPolicy(gomock.Any(), policyID, rbacpolicies.UpdateOpts{TargetTenant: "project-b"}).
					Return(nil, updateError)
			},
			// A plain error (not classified non-retryable by orcerrors.IsRetryable)
			// defaults to retryable, so the generic reconciler still reschedules.
			wantReschedule: true,
			wantErr:        updateError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockctrl := gomock.NewController(t)
			networkClient := mock.NewMockNetworkClient(mockctrl)

			actuator := rbacpolicyActuator{osClient: networkClient}

			recorder := networkClient.EXPECT()
			if tt.expect != nil {
				tt.expect(recorder)
			}

			reconcileStatus := actuator.updateResource(context.TODO(), tt.orcObject, tt.osResource)
			needsReschedule, err := reconcileStatus.NeedsReschedule()

			if tt.wantErr == nil && err != nil {
				t.Errorf("updateResource() error = %v, want no error", err)
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Errorf("updateResource() error = %v, want %v", err, tt.wantErr)
			}
			if needsReschedule != tt.wantReschedule {
				t.Errorf("updateResource() needsReschedule = %v, want %v", needsReschedule, tt.wantReschedule)
			}
		})
	}
}
