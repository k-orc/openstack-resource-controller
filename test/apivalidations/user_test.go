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
	userName = "user"
	userID   = "265c9e4f-0f5a-46e4-9f3f-fb8de25ae127"
)

func userStub(namespace *corev1.Namespace) *orcv1alpha1.User {
	obj := &orcv1alpha1.User{}
	obj.Name = userName
	obj.Namespace = namespace.Name
	return obj
}

func testUserResource() *applyconfigv1alpha1.UserResourceSpecApplyConfiguration {
	return applyconfigv1alpha1.UserResourceSpec()
}

func baseUserPatch(user client.Object) *applyconfigv1alpha1.UserApplyConfiguration {
	return applyconfigv1alpha1.User(user.GetName(), user.GetNamespace()).
		WithSpec(applyconfigv1alpha1.UserSpec().
			WithCloudCredentialsRef(testCredentials()))
}

func testUserImport() *applyconfigv1alpha1.UserImportApplyConfiguration {
	return applyconfigv1alpha1.UserImport().WithID(userID)
}

var _ = Describe("ORC User API validations", func() {
	var namespace *corev1.Namespace
	BeforeEach(func() {
		namespace = createNamespace()
	})

	It("should allow to create a minimal user and managementPolicy should default to managed", func(ctx context.Context) {
		user := userStub(namespace)
		patch := baseUserPatch(user)
		patch.Spec.WithResource(testUserResource())
		Expect(applyObj(ctx, user, patch)).To(Succeed())
		Expect(user.Spec.ManagementPolicy).To(Equal(orcv1alpha1.ManagementPolicyManaged))
	})

	It("should have immutable domainRef", func(ctx context.Context) {
		user := userStub(namespace)
		patch := baseUserPatch(user)
		patch.Spec.WithResource(applyconfigv1alpha1.UserResourceSpec().
			WithDomainRef("domain-a"))
		Expect(applyObj(ctx, user, patch)).To(Succeed())

		patch.Spec.WithResource(applyconfigv1alpha1.UserResourceSpec().
			WithDomainRef("domain-b"))
		Expect(applyObj(ctx, user, patch)).To(MatchError(ContainSubstring("domainRef is immutable")))
	})

	It("should have immutable defaultProjectRef", func(ctx context.Context) {
		user := userStub(namespace)
		patch := baseUserPatch(user)
		patch.Spec.WithResource(applyconfigv1alpha1.UserResourceSpec().
			WithDefaultProjectRef("project-a"))
		Expect(applyObj(ctx, user, patch)).To(Succeed())

		patch.Spec.WithResource(applyconfigv1alpha1.UserResourceSpec().
			WithDefaultProjectRef("project-b"))
		Expect(applyObj(ctx, user, patch)).To(MatchError(ContainSubstring("defaultProjectRef is immutable")))
	})

	It("should require import for unmanaged", func(ctx context.Context) {
		user := userStub(namespace)
		patch := baseUserPatch(user)
		patch.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged)
		Expect(applyObj(ctx, user, patch)).To(MatchError(ContainSubstring("import must be specified when policy is unmanaged")))

		patch.Spec.WithImport(testUserImport())
		Expect(applyObj(ctx, user, patch)).To(Succeed())
	})

	It("should not permit unmanaged with resource", func(ctx context.Context) {
		user := userStub(namespace)
		patch := baseUserPatch(user)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(testUserImport()).
			WithResource(testUserResource())
		Expect(applyObj(ctx, user, patch)).To(MatchError(ContainSubstring("resource may not be specified when policy is unmanaged")))
	})

	It("should not permit empty import", func(ctx context.Context) {
		user := userStub(namespace)
		patch := baseUserPatch(user)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(applyconfigv1alpha1.UserImport())
		Expect(applyObj(ctx, user, patch)).To(MatchError(ContainSubstring("spec.import in body should have at least 1 properties")))
	})

	It("should not permit empty import filter", func(ctx context.Context) {
		user := userStub(namespace)
		patch := baseUserPatch(user)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(applyconfigv1alpha1.UserImport().
				WithFilter(applyconfigv1alpha1.UserFilter()))
		Expect(applyObj(ctx, user, patch)).To(MatchError(ContainSubstring("spec.import.filter in body should have at least 1 properties")))
	})

	It("should permit import filter with name", func(ctx context.Context) {
		user := userStub(namespace)
		patch := baseUserPatch(user)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(applyconfigv1alpha1.UserImport().
				WithFilter(applyconfigv1alpha1.UserFilter().WithName("foo")))
		Expect(applyObj(ctx, user, patch)).To(Succeed())
	})

	It("should require resource for managed", func(ctx context.Context) {
		user := userStub(namespace)
		patch := baseUserPatch(user)
		patch.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged)
		Expect(applyObj(ctx, user, patch)).To(MatchError(ContainSubstring("resource must be specified when policy is managed")))

		patch.Spec.WithResource(testUserResource())
		Expect(applyObj(ctx, user, patch)).To(Succeed())
	})

	It("should not permit managed with import", func(ctx context.Context) {
		user := userStub(namespace)
		patch := baseUserPatch(user)
		patch.Spec.
			WithImport(testUserImport()).
			WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged).
			WithResource(testUserResource())
		Expect(applyObj(ctx, user, patch)).To(MatchError(ContainSubstring("import may not be specified when policy is managed")))
	})

	It("should not permit managedOptions for unmanaged", func(ctx context.Context) {
		user := userStub(namespace)
		patch := baseUserPatch(user)
		patch.Spec.
			WithImport(testUserImport()).
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithManagedOptions(applyconfigv1alpha1.ManagedOptions().
				WithOnDelete(orcv1alpha1.OnDeleteDetach))
		Expect(applyObj(ctx, user, patch)).To(MatchError(ContainSubstring("managedOptions may only be provided when policy is managed")))
	})

	It("should permit managedOptions for managed", func(ctx context.Context) {
		user := userStub(namespace)
		patch := baseUserPatch(user)
		patch.Spec.WithResource(testUserResource()).
			WithManagedOptions(applyconfigv1alpha1.ManagedOptions().
				WithOnDelete(orcv1alpha1.OnDeleteDetach))
		Expect(applyObj(ctx, user, patch)).To(Succeed())
		Expect(user.Spec.ManagedOptions.OnDelete).To(Equal(orcv1alpha1.OnDelete("detach")))
	})
})
