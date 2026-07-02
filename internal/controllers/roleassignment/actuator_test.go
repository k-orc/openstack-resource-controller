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

package roleassignment

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"testing"

	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/roles"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v2/internal/osclients"
)

var (
	errNotImplemented = errors.New("not implemented")
	errTest           = errors.New("test error")
)

const testNamespace = "test-ns"

// mockRoleAssignmentClient is a simple mock that returns pre-configured assignments.
type mockRoleAssignmentClient struct {
	assignments []roles.RoleAssignment
}

var _ osclients.RoleAssignmentClient = mockRoleAssignmentClient{}

func (m mockRoleAssignmentClient) ListRoleAssignments(_ context.Context, _ roles.ListAssignmentsOpts) iter.Seq2[*roles.RoleAssignment, error] {
	return func(yield func(*roles.RoleAssignment, error) bool) {
		for i := range m.assignments {
			if !yield(&m.assignments[i], nil) {
				return
			}
		}
	}
}

func (m mockRoleAssignmentClient) AssignRole(_ context.Context, _ string, _ roles.AssignOpts) error {
	return errNotImplemented
}

func (m mockRoleAssignmentClient) UnassignRole(_ context.Context, _ string, _ roles.UnassignOpts) error {
	return errNotImplemented
}

// Test result type and check helpers

type raResult struct {
	assignment *roles.RoleAssignment
	err        error
}

type checkFunc func([]raResult) error

func checks(fns ...checkFunc) []checkFunc { return fns }

func noError(results []raResult) error {
	for _, result := range results {
		if result.err != nil {
			return fmt.Errorf("unexpected error: %w", result.err)
		}
	}
	return nil
}

func wantError(wantErr error) checkFunc {
	return func(results []raResult) error {
		for _, result := range results {
			if result.err != nil && errors.Is(result.err, wantErr) {
				return nil
			}
		}
		return fmt.Errorf("expected error %v not found in results", wantErr)
	}
}

func findsN(wantN int) checkFunc {
	return func(results []raResult) error {
		found := len(results)
		if found != wantN {
			return fmt.Errorf("expected %d results, got %d", wantN, found)
		}
		return nil
	}
}

// availableCondition returns an Available=True condition for test objects.
func availableCondition() metav1.Condition {
	return metav1.Condition{
		Type:               orcv1alpha1.ConditionAvailable,
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             "Available",
	}
}

// newFakeK8sClient creates a fake k8s client with the given objects and ORC scheme.
func newFakeK8sClient(objects ...client.Object) client.Client {
	scheme := runtime.NewScheme()
	_ = orcv1alpha1.AddToScheme(scheme)

	return fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objects...).
		Build()
}

// availableRole returns a Role object that is available with the given status ID.
func availableRole(name, statusID string) *orcv1alpha1.Role {
	return &orcv1alpha1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
		},
		Status: orcv1alpha1.RoleStatus{
			Conditions: []metav1.Condition{availableCondition()},
			ID:         ptr.To(statusID),
		},
	}
}

// availableUser returns a User object that is available with the given status ID.
func availableUser(name, statusID string) *orcv1alpha1.User {
	return &orcv1alpha1.User{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
		},
		Status: orcv1alpha1.UserStatus{
			Conditions: []metav1.Condition{availableCondition()},
			ID:         ptr.To(statusID),
		},
	}
}

// availableGroup returns a Group object that is available with the given status ID.
func availableGroup(name, statusID string) *orcv1alpha1.Group {
	return &orcv1alpha1.Group{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
		},
		Status: orcv1alpha1.GroupStatus{
			Conditions: []metav1.Condition{availableCondition()},
			ID:         ptr.To(statusID),
		},
	}
}

// availableProject returns a Project object that is available with the given status ID.
func availableProject(name, statusID string) *orcv1alpha1.Project {
	return &orcv1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
		},
		Status: orcv1alpha1.ProjectStatus{
			Conditions: []metav1.Condition{availableCondition()},
			ID:         ptr.To(statusID),
		},
	}
}

// availableDomain returns a Domain object that is available with the given status ID.
func availableDomain(name, statusID string) *orcv1alpha1.Domain {
	return &orcv1alpha1.Domain{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
		},
		Status: orcv1alpha1.DomainStatus{
			Conditions: []metav1.Condition{availableCondition()},
			ID:         ptr.To(statusID),
		},
	}
}

func TestListOSResourcesForAdoption(t *testing.T) {
	userProjectAssignment := roles.RoleAssignment{
		Role:  roles.AssignedRole{ID: "role-id-1"},
		User:  roles.User{ID: "user-id-1"},
		Scope: roles.Scope{Project: roles.Project{ID: "project-id-1"}},
	}

	groupDomainAssignment := roles.RoleAssignment{
		Role:  roles.AssignedRole{ID: "role-id-2"},
		Group: roles.Group{ID: "group-id-2"},
		Scope: roles.Scope{Domain: roles.Domain{ID: "domain-id-2"}},
	}

	for _, tc := range [...]struct {
		name       string
		orcObject  *orcv1alpha1.RoleAssignment
		k8sObjects []client.Object
		osClient   osclients.RoleAssignmentClient
		wantAdopt  bool
		checks     []checkFunc
	}{
		{
			name: "returns false when spec.resource is nil",
			orcObject: &orcv1alpha1.RoleAssignment{
				ObjectMeta: metav1.ObjectMeta{Name: "test-ra", Namespace: testNamespace},
				Spec:       orcv1alpha1.RoleAssignmentSpec{},
			},
			osClient:  mockRoleAssignmentClient{},
			wantAdopt: false,
		},
		{
			name: "user+project scope, all deps available, match found",
			orcObject: &orcv1alpha1.RoleAssignment{
				ObjectMeta: metav1.ObjectMeta{Name: "test-ra", Namespace: testNamespace},
				Spec: orcv1alpha1.RoleAssignmentSpec{
					Resource: &orcv1alpha1.RoleAssignmentResourceSpec{
						RoleRef:    "test-role",
						UserRef:    ptr.To[orcv1alpha1.KubernetesNameRef]("test-user"),
						ProjectRef: ptr.To[orcv1alpha1.KubernetesNameRef]("test-project"),
					},
				},
			},
			k8sObjects: []client.Object{
				availableRole("test-role", "role-id-1"),
				availableUser("test-user", "user-id-1"),
				availableProject("test-project", "project-id-1"),
			},
			osClient:  mockRoleAssignmentClient{assignments: []roles.RoleAssignment{userProjectAssignment}},
			wantAdopt: true,
			checks:    checks(noError, findsN(1)),
		},
		{
			name: "group+domain scope, all deps available, match found",
			orcObject: &orcv1alpha1.RoleAssignment{
				ObjectMeta: metav1.ObjectMeta{Name: "test-ra", Namespace: testNamespace},
				Spec: orcv1alpha1.RoleAssignmentSpec{
					Resource: &orcv1alpha1.RoleAssignmentResourceSpec{
						RoleRef:   "test-role",
						GroupRef:  ptr.To[orcv1alpha1.KubernetesNameRef]("test-group"),
						DomainRef: ptr.To[orcv1alpha1.KubernetesNameRef]("test-domain"),
					},
				},
			},
			k8sObjects: []client.Object{
				availableRole("test-role", "role-id-2"),
				availableGroup("test-group", "group-id-2"),
				availableDomain("test-domain", "domain-id-2"),
			},
			osClient:  mockRoleAssignmentClient{assignments: []roles.RoleAssignment{groupDomainAssignment}},
			wantAdopt: true,
			checks:    checks(noError, findsN(1)),
		},
		{
			name: "all deps available, no matches from OS",
			orcObject: &orcv1alpha1.RoleAssignment{
				ObjectMeta: metav1.ObjectMeta{Name: "test-ra", Namespace: testNamespace},
				Spec: orcv1alpha1.RoleAssignmentSpec{
					Resource: &orcv1alpha1.RoleAssignmentResourceSpec{
						RoleRef:    "test-role",
						UserRef:    ptr.To[orcv1alpha1.KubernetesNameRef]("test-user"),
						ProjectRef: ptr.To[orcv1alpha1.KubernetesNameRef]("test-project"),
					},
				},
			},
			k8sObjects: []client.Object{
				availableRole("test-role", "role-id-1"),
				availableUser("test-user", "user-id-1"),
				availableProject("test-project", "project-id-1"),
			},
			osClient:  mockRoleAssignmentClient{assignments: []roles.RoleAssignment{}},
			wantAdopt: true,
			checks:    checks(noError, findsN(0)),
		},
		{
			name: "OS client returns error",
			orcObject: &orcv1alpha1.RoleAssignment{
				ObjectMeta: metav1.ObjectMeta{Name: "test-ra", Namespace: testNamespace},
				Spec: orcv1alpha1.RoleAssignmentSpec{
					Resource: &orcv1alpha1.RoleAssignmentResourceSpec{
						RoleRef:    "test-role",
						UserRef:    ptr.To[orcv1alpha1.KubernetesNameRef]("test-user"),
						ProjectRef: ptr.To[orcv1alpha1.KubernetesNameRef]("test-project"),
					},
				},
			},
			k8sObjects: []client.Object{
				availableRole("test-role", "role-id-1"),
				availableUser("test-user", "user-id-1"),
				availableProject("test-project", "project-id-1"),
			},
			osClient:  osclients.NewRoleAssignmentErrorClient(errTest),
			wantAdopt: true,
			checks:    checks(wantError(errTest)),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			k8sClient := newFakeK8sClient(tc.k8sObjects...)

			actuator := roleassignmentActuator{
				osClient:  tc.osClient,
				k8sClient: k8sClient,
			}

			resourceIter, canAdopt := actuator.ListOSResourcesForAdoption(ctx, tc.orcObject)
			if canAdopt != tc.wantAdopt {
				t.Fatalf("canAdopt = %v, want %v", canAdopt, tc.wantAdopt)
			}

			if !canAdopt {
				return
			}

			var results []raResult
			for assignment, err := range resourceIter {
				results = append(results, raResult{assignment, err})
			}

			for _, check := range tc.checks {
				if e := check(results); e != nil {
					t.Error(e)
				}
			}
		})
	}
}
