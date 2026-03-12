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
	roleName = "role"
	roleID   = "265c9e4f-0f5a-46e4-9f3f-fb8de25ae121"
)

func roleStub(namespace *corev1.Namespace) *orcv1alpha1.Role {
	obj := &orcv1alpha1.Role{}
	obj.Name = roleName
	obj.Namespace = namespace.Name
	return obj
}

func testRoleResource() *applyconfigv1alpha1.RoleResourceSpecApplyConfiguration {
	return applyconfigv1alpha1.RoleResourceSpec()
}

func baseRolePatch(role client.Object) *applyconfigv1alpha1.RoleApplyConfiguration {
	return applyconfigv1alpha1.Role(role.GetName(), role.GetNamespace()).
		WithSpec(applyconfigv1alpha1.RoleSpec().
			WithCloudCredentialsRef(testCredentials()))
}

func testRoleImport() *applyconfigv1alpha1.RoleImportApplyConfiguration {
	return applyconfigv1alpha1.RoleImport().WithID(roleID)
}

var _ = Describe("ORC Role API validations", func() {
	var namespace *corev1.Namespace
	BeforeEach(func() {
		namespace = createNamespace()
	})

	It("should allow to create a minimal role and managementPolicy should default to managed", func(ctx context.Context) {
		role := roleStub(namespace)
		patch := baseRolePatch(role)
		patch.Spec.WithResource(testRoleResource())
		Expect(applyObj(ctx, role, patch)).To(Succeed())
		Expect(role.Spec.ManagementPolicy).To(Equal(orcv1alpha1.ManagementPolicyManaged))
	})

	It("should require import for unmanaged", func(ctx context.Context) {
		role := roleStub(namespace)
		patch := baseRolePatch(role)
		patch.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged)
		Expect(applyObj(ctx, role, patch)).To(MatchError(ContainSubstring("import must be specified when policy is unmanaged")))

		patch.Spec.WithImport(testRoleImport())
		Expect(applyObj(ctx, role, patch)).To(Succeed())
	})

	It("should not permit unmanaged with resource", func(ctx context.Context) {
		role := roleStub(namespace)
		patch := baseRolePatch(role)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(testRoleImport()).
			WithResource(testRoleResource())
		Expect(applyObj(ctx, role, patch)).To(MatchError(ContainSubstring("resource may not be specified when policy is unmanaged")))
	})

	It("should not permit empty import", func(ctx context.Context) {
		role := roleStub(namespace)
		patch := baseRolePatch(role)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(applyconfigv1alpha1.RoleImport())
		Expect(applyObj(ctx, role, patch)).To(MatchError(ContainSubstring("spec.import in body should have at least 1 properties")))
	})

	It("should not permit empty import filter", func(ctx context.Context) {
		role := roleStub(namespace)
		patch := baseRolePatch(role)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(applyconfigv1alpha1.RoleImport().
				WithFilter(applyconfigv1alpha1.RoleFilter()))
		Expect(applyObj(ctx, role, patch)).To(MatchError(ContainSubstring("spec.import.filter in body should have at least 1 properties")))
	})

	It("should permit import filter with name", func(ctx context.Context) {
		role := roleStub(namespace)
		patch := baseRolePatch(role)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(applyconfigv1alpha1.RoleImport().
				WithFilter(applyconfigv1alpha1.RoleFilter().WithName("foo")))
		Expect(applyObj(ctx, role, patch)).To(Succeed())
	})

	It("should require resource for managed", func(ctx context.Context) {
		role := roleStub(namespace)
		patch := baseRolePatch(role)
		patch.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged)
		Expect(applyObj(ctx, role, patch)).To(MatchError(ContainSubstring("resource must be specified when policy is managed")))

		patch.Spec.WithResource(testRoleResource())
		Expect(applyObj(ctx, role, patch)).To(Succeed())
	})

	It("should not permit managed with import", func(ctx context.Context) {
		role := roleStub(namespace)
		patch := baseRolePatch(role)
		patch.Spec.
			WithImport(testRoleImport()).
			WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged).
			WithResource(testRoleResource())
		Expect(applyObj(ctx, role, patch)).To(MatchError(ContainSubstring("import may not be specified when policy is managed")))
	})

	It("should not permit managedOptions for unmanaged", func(ctx context.Context) {
		role := roleStub(namespace)
		patch := baseRolePatch(role)
		patch.Spec.
			WithImport(testRoleImport()).
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithManagedOptions(applyconfigv1alpha1.ManagedOptions().
				WithOnDelete(orcv1alpha1.OnDeleteDetach))
		Expect(applyObj(ctx, role, patch)).To(MatchError(ContainSubstring("managedOptions may only be provided when policy is managed")))
	})

	It("should permit managedOptions for managed", func(ctx context.Context) {
		role := roleStub(namespace)
		patch := baseRolePatch(role)
		patch.Spec.WithResource(testRoleResource()).
			WithManagedOptions(applyconfigv1alpha1.ManagedOptions().
				WithOnDelete(orcv1alpha1.OnDeleteDetach))
		Expect(applyObj(ctx, role, patch)).To(Succeed())
		Expect(role.Spec.ManagedOptions.OnDelete).To(Equal(orcv1alpha1.OnDelete("detach")))
	})

	It("should have immutable domainRef", func(ctx context.Context) {
		role := roleStub(namespace)
		patch := baseRolePatch(role)
		patch.Spec.WithResource(applyconfigv1alpha1.RoleResourceSpec().
			WithDomainRef("domain-a"))
		Expect(applyObj(ctx, role, patch)).To(Succeed())

		patch.Spec.WithResource(applyconfigv1alpha1.RoleResourceSpec().
			WithDomainRef("domain-b"))
		Expect(applyObj(ctx, role, patch)).To(MatchError(ContainSubstring("domainRef is immutable")))
	})
})
