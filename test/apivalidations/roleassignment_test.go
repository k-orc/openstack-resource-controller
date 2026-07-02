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

package apivalidations

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	applyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/pkg/clients/applyconfiguration/api/v1alpha1"
)

const (
	roleassignmentName = "roleassignment"
)

func roleassignmentStub(namespace *corev1.Namespace) *orcv1alpha1.RoleAssignment {
	obj := &orcv1alpha1.RoleAssignment{}
	obj.Name = roleassignmentName
	obj.Namespace = namespace.Name
	return obj
}

func testRoleAssignmentResource() *applyconfigv1alpha1.RoleAssignmentResourceSpecApplyConfiguration {
	return applyconfigv1alpha1.RoleAssignmentResourceSpec().
		WithRoleRef("role").
		WithUserRef("user").
		WithProjectRef("project")
}

func baseRoleAssignmentPatch(obj client.Object) *applyconfigv1alpha1.RoleAssignmentApplyConfiguration {
	return applyconfigv1alpha1.RoleAssignment(obj.GetName(), obj.GetNamespace()).
		WithSpec(applyconfigv1alpha1.RoleAssignmentSpec().
			WithCloudCredentialsRef(testCredentials()))
}

func testRoleAssignmentImport() *applyconfigv1alpha1.RoleAssignmentImportApplyConfiguration {
	return applyconfigv1alpha1.RoleAssignmentImport().
		WithFilter(applyconfigv1alpha1.RoleAssignmentFilter().WithRoleRef("admin"))
}

var _ = Describe("ORC RoleAssignment API validations", func() {
	var namespace *corev1.Namespace
	BeforeEach(func() {
		namespace = createNamespace()
	})

	runManagementPolicyTests(func() *corev1.Namespace { return namespace }, managementPolicyTestArgs[*applyconfigv1alpha1.RoleAssignmentApplyConfiguration]{
		createObject: func(ns *corev1.Namespace) client.Object { return roleassignmentStub(ns) },
		basePatch: func(obj client.Object) *applyconfigv1alpha1.RoleAssignmentApplyConfiguration {
			return baseRoleAssignmentPatch(obj)
		},
		applyResource: func(p *applyconfigv1alpha1.RoleAssignmentApplyConfiguration) {
			p.Spec.WithResource(testRoleAssignmentResource())
		},
		applyImport: func(p *applyconfigv1alpha1.RoleAssignmentApplyConfiguration) {
			p.Spec.WithImport(testRoleAssignmentImport())
		},
		applyEmptyImport: func(p *applyconfigv1alpha1.RoleAssignmentApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.RoleAssignmentImport())
		},
		applyEmptyFilter: func(p *applyconfigv1alpha1.RoleAssignmentApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.RoleAssignmentImport().WithFilter(applyconfigv1alpha1.RoleAssignmentFilter()))
		},
		applyValidFilter: func(p *applyconfigv1alpha1.RoleAssignmentApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.RoleAssignmentImport().WithFilter(applyconfigv1alpha1.RoleAssignmentFilter().WithRoleRef("admin")))
		},
		applyManaged: func(p *applyconfigv1alpha1.RoleAssignmentApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged)
		},
		applyUnmanaged: func(p *applyconfigv1alpha1.RoleAssignmentApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged)
		},
		applyManagedOptions: func(p *applyconfigv1alpha1.RoleAssignmentApplyConfiguration) {
			p.Spec.WithManagedOptions(applyconfigv1alpha1.ManagedOptions().WithOnDelete(orcv1alpha1.OnDeleteDetach))
		},
		getManagementPolicy: func(obj client.Object) orcv1alpha1.ManagementPolicy {
			return obj.(*orcv1alpha1.RoleAssignment).Spec.ManagementPolicy
		},
		getOnDelete: func(obj client.Object) orcv1alpha1.OnDelete {
			return obj.(*orcv1alpha1.RoleAssignment).Spec.ManagedOptions.OnDelete
		},
	})

	It("should reject a roleassignment without required fields", func(ctx context.Context) {
		obj := roleassignmentStub(namespace)
		patch := baseRoleAssignmentPatch(obj)
		patch.Spec.WithResource(applyconfigv1alpha1.RoleAssignmentResourceSpec())
		Expect(applyObj(ctx, obj, patch)).NotTo(Succeed())
	})

	It("should have immutable RoleAssignmentResourceSpec", func(ctx context.Context) {
		obj := roleassignmentStub(namespace)
		patch := baseRoleAssignmentPatch(obj)
		patch.Spec.WithResource(applyconfigv1alpha1.RoleAssignmentResourceSpec().
			WithRoleRef("role").
			WithUserRef("user").
			WithProjectRef("project"))
		Expect(applyObj(ctx, obj, patch)).To(Succeed())

		// Try to change any field - should fail because entire spec is immutable
		patch.Spec.WithResource(applyconfigv1alpha1.RoleAssignmentResourceSpec().
			WithRoleRef("role").
			WithUserRef("user-changed").
			WithProjectRef("project"))
		Expect(applyObj(ctx, obj, patch)).To(MatchError(ContainSubstring("RoleAssignmentResourceSpec is immutable")))
	})

	// TODO(scaffolding): Add more resource-specific validation tests.
	// Some common things to test:
	// - Immutability of fields with `self == oldSelf` validation
	// - Enum validation (valid and invalid values)
	// - Numeric range validation (min/max bounds)
	// - Tag uniqueness (if the resource has tags with listType=set)
	// - Format validation (CIDR, UUID, etc.)
	// - Cross-field validation rules
})
