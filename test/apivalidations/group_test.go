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
	groupName = "group"
	groupID   = "265c9e4f-0f5a-46e4-9f3f-fb8de25ae122"
)

func groupStub(namespace *corev1.Namespace) *orcv1alpha1.Group {
	obj := &orcv1alpha1.Group{}
	obj.Name = groupName
	obj.Namespace = namespace.Name
	return obj
}

func testGroupResource() *applyconfigv1alpha1.GroupResourceSpecApplyConfiguration {
	return applyconfigv1alpha1.GroupResourceSpec()
}

func baseGroupPatch(group client.Object) *applyconfigv1alpha1.GroupApplyConfiguration {
	return applyconfigv1alpha1.Group(group.GetName(), group.GetNamespace()).
		WithSpec(applyconfigv1alpha1.GroupSpec().
			WithCloudCredentialsRef(testCredentials()))
}

func testGroupImport() *applyconfigv1alpha1.GroupImportApplyConfiguration {
	return applyconfigv1alpha1.GroupImport().WithID(groupID)
}

var _ = Describe("ORC Group API validations", func() {
	var namespace *corev1.Namespace
	BeforeEach(func() {
		namespace = createNamespace()
	})

	runManagementPolicyTests(func() *corev1.Namespace { return namespace },
		managementPolicyTestArgs[*applyconfigv1alpha1.GroupApplyConfiguration]{
			createObject: func(ns *corev1.Namespace) client.Object { return groupStub(ns) },
			basePatch:    func(obj client.Object) *applyconfigv1alpha1.GroupApplyConfiguration { return baseGroupPatch(obj) },
			applyResource: func(p *applyconfigv1alpha1.GroupApplyConfiguration) {
				p.Spec.WithResource(testGroupResource())
			},
			applyImport: func(p *applyconfigv1alpha1.GroupApplyConfiguration) {
				p.Spec.WithImport(testGroupImport())
			},
			applyEmptyImport: func(p *applyconfigv1alpha1.GroupApplyConfiguration) {
				p.Spec.WithImport(applyconfigv1alpha1.GroupImport())
			},
			applyEmptyFilter: func(p *applyconfigv1alpha1.GroupApplyConfiguration) {
				p.Spec.WithImport(applyconfigv1alpha1.GroupImport().WithFilter(applyconfigv1alpha1.GroupFilter()))
			},
			applyValidFilter: func(p *applyconfigv1alpha1.GroupApplyConfiguration) {
				p.Spec.WithImport(applyconfigv1alpha1.GroupImport().WithFilter(applyconfigv1alpha1.GroupFilter().WithName("foo")))
			},
			applyManaged: func(p *applyconfigv1alpha1.GroupApplyConfiguration) {
				p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged)
			},
			applyUnmanaged: func(p *applyconfigv1alpha1.GroupApplyConfiguration) {
				p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged)
			},
			applyManagedOptions: func(p *applyconfigv1alpha1.GroupApplyConfiguration) {
				p.Spec.WithManagedOptions(applyconfigv1alpha1.ManagedOptions().WithOnDelete(orcv1alpha1.OnDeleteDetach))
			},
			getManagementPolicy: func(obj client.Object) orcv1alpha1.ManagementPolicy {
				return obj.(*orcv1alpha1.Group).Spec.ManagementPolicy
			},
			getOnDelete: func(obj client.Object) orcv1alpha1.OnDelete {
				return obj.(*orcv1alpha1.Group).Spec.ManagedOptions.OnDelete
			},
		},
	)

	It("should have immutable domainRef", func(ctx context.Context) {
		group := groupStub(namespace)
		patch := baseGroupPatch(group)
		patch.Spec.WithResource(applyconfigv1alpha1.GroupResourceSpec().
			WithDomainRef("domain-a"))
		Expect(applyObj(ctx, group, patch)).To(Succeed())

		patch.Spec.WithResource(applyconfigv1alpha1.GroupResourceSpec().
			WithDomainRef("domain-b"))
		Expect(applyObj(ctx, group, patch)).To(MatchError(ContainSubstring("domainRef is immutable")))
	})
})
