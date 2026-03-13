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

	runManagementPolicyTests(func() *corev1.Namespace { return namespace },
		managementPolicyTestArgs[*applyconfigv1alpha1.RoleApplyConfiguration]{
			createObject: func(ns *corev1.Namespace) client.Object { return roleStub(ns) },
			basePatch:    func(obj client.Object) *applyconfigv1alpha1.RoleApplyConfiguration { return baseRolePatch(obj) },
			applyResource: func(p *applyconfigv1alpha1.RoleApplyConfiguration) {
				p.Spec.WithResource(testRoleResource())
			},
			applyImport: func(p *applyconfigv1alpha1.RoleApplyConfiguration) {
				p.Spec.WithImport(testRoleImport())
			},
			applyEmptyImport: func(p *applyconfigv1alpha1.RoleApplyConfiguration) {
				p.Spec.WithImport(applyconfigv1alpha1.RoleImport())
			},
			applyEmptyFilter: func(p *applyconfigv1alpha1.RoleApplyConfiguration) {
				p.Spec.WithImport(applyconfigv1alpha1.RoleImport().WithFilter(applyconfigv1alpha1.RoleFilter()))
			},
			applyValidFilter: func(p *applyconfigv1alpha1.RoleApplyConfiguration) {
				p.Spec.WithImport(applyconfigv1alpha1.RoleImport().WithFilter(applyconfigv1alpha1.RoleFilter().WithName("foo")))
			},
			applyManaged: func(p *applyconfigv1alpha1.RoleApplyConfiguration) {
				p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged)
			},
			applyUnmanaged: func(p *applyconfigv1alpha1.RoleApplyConfiguration) {
				p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged)
			},
			applyManagedOptions: func(p *applyconfigv1alpha1.RoleApplyConfiguration) {
				p.Spec.WithManagedOptions(applyconfigv1alpha1.ManagedOptions().WithOnDelete(orcv1alpha1.OnDeleteDetach))
			},
			getManagementPolicy: func(obj client.Object) orcv1alpha1.ManagementPolicy {
				return obj.(*orcv1alpha1.Role).Spec.ManagementPolicy
			},
			getOnDelete: func(obj client.Object) orcv1alpha1.OnDelete {
				return obj.(*orcv1alpha1.Role).Spec.ManagedOptions.OnDelete
			},
		},
	)

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
