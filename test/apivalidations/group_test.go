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

	It("should allow to create a minimal group and managementPolicy should default to managed", func(ctx context.Context) {
		group := groupStub(namespace)
		patch := baseGroupPatch(group)
		patch.Spec.WithResource(testGroupResource())
		Expect(applyObj(ctx, group, patch)).To(Succeed())
		Expect(group.Spec.ManagementPolicy).To(Equal(orcv1alpha1.ManagementPolicyManaged))
	})

	It("should require import for unmanaged", func(ctx context.Context) {
		group := groupStub(namespace)
		patch := baseGroupPatch(group)
		patch.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged)
		Expect(applyObj(ctx, group, patch)).To(MatchError(ContainSubstring("import must be specified when policy is unmanaged")))

		patch.Spec.WithImport(testGroupImport())
		Expect(applyObj(ctx, group, patch)).To(Succeed())
	})

	It("should not permit unmanaged with resource", func(ctx context.Context) {
		group := groupStub(namespace)
		patch := baseGroupPatch(group)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(testGroupImport()).
			WithResource(testGroupResource())
		Expect(applyObj(ctx, group, patch)).To(MatchError(ContainSubstring("resource may not be specified when policy is unmanaged")))
	})

	It("should not permit empty import", func(ctx context.Context) {
		group := groupStub(namespace)
		patch := baseGroupPatch(group)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(applyconfigv1alpha1.GroupImport())
		Expect(applyObj(ctx, group, patch)).To(MatchError(ContainSubstring("spec.import in body should have at least 1 properties")))
	})

	It("should not permit empty import filter", func(ctx context.Context) {
		group := groupStub(namespace)
		patch := baseGroupPatch(group)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(applyconfigv1alpha1.GroupImport().
				WithFilter(applyconfigv1alpha1.GroupFilter()))
		Expect(applyObj(ctx, group, patch)).To(MatchError(ContainSubstring("spec.import.filter in body should have at least 1 properties")))
	})

	It("should permit import filter with name", func(ctx context.Context) {
		group := groupStub(namespace)
		patch := baseGroupPatch(group)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(applyconfigv1alpha1.GroupImport().
				WithFilter(applyconfigv1alpha1.GroupFilter().WithName("foo")))
		Expect(applyObj(ctx, group, patch)).To(Succeed())
	})

	It("should require resource for managed", func(ctx context.Context) {
		group := groupStub(namespace)
		patch := baseGroupPatch(group)
		patch.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged)
		Expect(applyObj(ctx, group, patch)).To(MatchError(ContainSubstring("resource must be specified when policy is managed")))

		patch.Spec.WithResource(testGroupResource())
		Expect(applyObj(ctx, group, patch)).To(Succeed())
	})

	It("should not permit managed with import", func(ctx context.Context) {
		group := groupStub(namespace)
		patch := baseGroupPatch(group)
		patch.Spec.
			WithImport(testGroupImport()).
			WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged).
			WithResource(testGroupResource())
		Expect(applyObj(ctx, group, patch)).To(MatchError(ContainSubstring("import may not be specified when policy is managed")))
	})

	It("should not permit managedOptions for unmanaged", func(ctx context.Context) {
		group := groupStub(namespace)
		patch := baseGroupPatch(group)
		patch.Spec.
			WithImport(testGroupImport()).
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithManagedOptions(applyconfigv1alpha1.ManagedOptions().
				WithOnDelete(orcv1alpha1.OnDeleteDetach))
		Expect(applyObj(ctx, group, patch)).To(MatchError(ContainSubstring("managedOptions may only be provided when policy is managed")))
	})

	It("should permit managedOptions for managed", func(ctx context.Context) {
		group := groupStub(namespace)
		patch := baseGroupPatch(group)
		patch.Spec.WithResource(testGroupResource()).
			WithManagedOptions(applyconfigv1alpha1.ManagedOptions().
				WithOnDelete(orcv1alpha1.OnDeleteDetach))
		Expect(applyObj(ctx, group, patch)).To(Succeed())
		Expect(group.Spec.ManagedOptions.OnDelete).To(Equal(orcv1alpha1.OnDelete("detach")))
	})

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
