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
	return applyconfigv1alpha1.UserResourceSpec().
		WithPasswordRef("user-password")
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

	runManagementPolicyTests(func() *corev1.Namespace { return namespace }, managementPolicyTestArgs[*applyconfigv1alpha1.UserApplyConfiguration]{
		createObject:  func(ns *corev1.Namespace) client.Object { return userStub(ns) },
		basePatch:     func(obj client.Object) *applyconfigv1alpha1.UserApplyConfiguration { return baseUserPatch(obj) },
		applyResource: func(p *applyconfigv1alpha1.UserApplyConfiguration) { p.Spec.WithResource(testUserResource()) },
		applyImport:   func(p *applyconfigv1alpha1.UserApplyConfiguration) { p.Spec.WithImport(testUserImport()) },
		applyEmptyImport: func(p *applyconfigv1alpha1.UserApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.UserImport())
		},
		applyEmptyFilter: func(p *applyconfigv1alpha1.UserApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.UserImport().WithFilter(applyconfigv1alpha1.UserFilter()))
		},
		applyValidFilter: func(p *applyconfigv1alpha1.UserApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.UserImport().WithFilter(applyconfigv1alpha1.UserFilter().WithName("foo")))
		},
		applyManaged: func(p *applyconfigv1alpha1.UserApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged)
		},
		applyUnmanaged: func(p *applyconfigv1alpha1.UserApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged)
		},
		applyManagedOptions: func(p *applyconfigv1alpha1.UserApplyConfiguration) {
			p.Spec.WithManagedOptions(applyconfigv1alpha1.ManagedOptions().WithOnDelete(orcv1alpha1.OnDeleteDetach))
		},
		getManagementPolicy: func(obj client.Object) orcv1alpha1.ManagementPolicy {
			return obj.(*orcv1alpha1.User).Spec.ManagementPolicy
		},
		getOnDelete: func(obj client.Object) orcv1alpha1.OnDelete {
			return obj.(*orcv1alpha1.User).Spec.ManagedOptions.OnDelete
		},
	})

	It("should have immutable domainRef", func(ctx context.Context) {
		user := userStub(namespace)
		patch := baseUserPatch(user)
		patch.Spec.WithResource(applyconfigv1alpha1.UserResourceSpec().
			WithPasswordRef("user-password").
			WithDomainRef("domain-a"))
		Expect(applyObj(ctx, user, patch)).To(Succeed())

		patch.Spec.WithResource(applyconfigv1alpha1.UserResourceSpec().
			WithPasswordRef("user-password").
			WithDomainRef("domain-b"))
		Expect(applyObj(ctx, user, patch)).To(MatchError(ContainSubstring("domainRef is immutable")))
	})

	It("should have immutable defaultProjectRef", func(ctx context.Context) {
		user := userStub(namespace)
		patch := baseUserPatch(user)
		patch.Spec.WithResource(applyconfigv1alpha1.UserResourceSpec().
			WithPasswordRef("user-password").
			WithDefaultProjectRef("project-a"))
		Expect(applyObj(ctx, user, patch)).To(Succeed())

		patch.Spec.WithResource(applyconfigv1alpha1.UserResourceSpec().
			WithPasswordRef("user-password").
			WithDefaultProjectRef("project-b"))
		Expect(applyObj(ctx, user, patch)).To(MatchError(ContainSubstring("defaultProjectRef is immutable")))
	})

	It("should have mutable passwordRef", func(ctx context.Context) {
		user := userStub(namespace)
		patch := baseUserPatch(user)
		patch.Spec.WithResource(applyconfigv1alpha1.UserResourceSpec().
			WithPasswordRef("password-a"))
		Expect(applyObj(ctx, user, patch)).To(Succeed())

		patch.Spec.WithResource(applyconfigv1alpha1.UserResourceSpec().
			WithPasswordRef("password-b"))
		Expect(applyObj(ctx, user, patch)).To(Succeed())
	})
})
