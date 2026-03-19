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
	serverGroupName = "servergroup"
	serverGroupID   = "265c9e4f-0f5a-46e4-9f3f-fb8de25ae129"
)

func serverGroupStub(namespace *corev1.Namespace) *orcv1alpha1.ServerGroup {
	obj := &orcv1alpha1.ServerGroup{}
	obj.Name = serverGroupName
	obj.Namespace = namespace.Name
	return obj
}

func testServerGroupResource() *applyconfigv1alpha1.ServerGroupResourceSpecApplyConfiguration {
	return applyconfigv1alpha1.ServerGroupResourceSpec().
		WithPolicy(orcv1alpha1.ServerGroupPolicyAffinity)
}

func baseServerGroupPatch(serverGroup client.Object) *applyconfigv1alpha1.ServerGroupApplyConfiguration {
	return applyconfigv1alpha1.ServerGroup(serverGroup.GetName(), serverGroup.GetNamespace()).
		WithSpec(applyconfigv1alpha1.ServerGroupSpec().
			WithCloudCredentialsRef(testCredentials()))
}

func testServerGroupImport() *applyconfigv1alpha1.ServerGroupImportApplyConfiguration {
	return applyconfigv1alpha1.ServerGroupImport().WithID(serverGroupID)
}

var _ = Describe("ORC ServerGroup API validations", func() {
	var namespace *corev1.Namespace
	BeforeEach(func() {
		namespace = createNamespace()
	})

	runManagementPolicyTests(func() *corev1.Namespace { return namespace }, managementPolicyTestArgs[*applyconfigv1alpha1.ServerGroupApplyConfiguration]{
		createObject: func(ns *corev1.Namespace) client.Object { return serverGroupStub(ns) },
		basePatch: func(obj client.Object) *applyconfigv1alpha1.ServerGroupApplyConfiguration {
			return baseServerGroupPatch(obj)
		},
		applyResource: func(p *applyconfigv1alpha1.ServerGroupApplyConfiguration) {
			p.Spec.WithResource(testServerGroupResource())
		},
		applyImport: func(p *applyconfigv1alpha1.ServerGroupApplyConfiguration) { p.Spec.WithImport(testServerGroupImport()) },
		applyEmptyImport: func(p *applyconfigv1alpha1.ServerGroupApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.ServerGroupImport())
		},
		applyEmptyFilter: func(p *applyconfigv1alpha1.ServerGroupApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.ServerGroupImport().WithFilter(applyconfigv1alpha1.ServerGroupFilter()))
		},
		applyValidFilter: func(p *applyconfigv1alpha1.ServerGroupApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.ServerGroupImport().WithFilter(applyconfigv1alpha1.ServerGroupFilter().WithName("foo")))
		},
		applyManaged: func(p *applyconfigv1alpha1.ServerGroupApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged)
		},
		applyUnmanaged: func(p *applyconfigv1alpha1.ServerGroupApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged)
		},
		applyManagedOptions: func(p *applyconfigv1alpha1.ServerGroupApplyConfiguration) {
			p.Spec.WithManagedOptions(applyconfigv1alpha1.ManagedOptions().WithOnDelete(orcv1alpha1.OnDeleteDetach))
		},
		getManagementPolicy: func(obj client.Object) orcv1alpha1.ManagementPolicy {
			return obj.(*orcv1alpha1.ServerGroup).Spec.ManagementPolicy
		},
		getOnDelete: func(obj client.Object) orcv1alpha1.OnDelete {
			return obj.(*orcv1alpha1.ServerGroup).Spec.ManagedOptions.OnDelete
		},
	})

	It("should reject a servergroup without required field policy", func(ctx context.Context) {
		serverGroup := serverGroupStub(namespace)
		patch := baseServerGroupPatch(serverGroup)
		patch.Spec.WithResource(applyconfigv1alpha1.ServerGroupResourceSpec())
		Expect(applyObj(ctx, serverGroup, patch)).To(MatchError(ContainSubstring("spec.resource.policy")))
	})

	It("should be immutable", func(ctx context.Context) {
		serverGroup := serverGroupStub(namespace)
		patch := baseServerGroupPatch(serverGroup)
		patch.Spec.WithResource(applyconfigv1alpha1.ServerGroupResourceSpec().
			WithPolicy(orcv1alpha1.ServerGroupPolicyAffinity))
		Expect(applyObj(ctx, serverGroup, patch)).To(Succeed())

		patch.Spec.WithResource(applyconfigv1alpha1.ServerGroupResourceSpec().
			WithPolicy(orcv1alpha1.ServerGroupPolicyAntiAffinity))
		Expect(applyObj(ctx, serverGroup, patch)).To(MatchError(ContainSubstring("ServerGroupResourceSpec is immutable")))
	})

	It("should reject invalid policy enum value", func(ctx context.Context) {
		serverGroup := serverGroupStub(namespace)
		patch := baseServerGroupPatch(serverGroup)
		patch.Spec.WithResource(applyconfigv1alpha1.ServerGroupResourceSpec().
			WithPolicy("invalid"))
		Expect(applyObj(ctx, serverGroup, patch)).NotTo(Succeed())
	})

	DescribeTable("should permit valid policy enum values",
		func(ctx context.Context, policy orcv1alpha1.ServerGroupPolicy) {
			serverGroup := serverGroupStub(namespace)
			patch := baseServerGroupPatch(serverGroup)
			patch.Spec.WithResource(applyconfigv1alpha1.ServerGroupResourceSpec().
				WithPolicy(policy))
			Expect(applyObj(ctx, serverGroup, patch)).To(Succeed())
		},
		Entry(string(orcv1alpha1.ServerGroupPolicyAffinity), orcv1alpha1.ServerGroupPolicyAffinity),
		Entry(string(orcv1alpha1.ServerGroupPolicyAntiAffinity), orcv1alpha1.ServerGroupPolicyAntiAffinity),
		Entry(string(orcv1alpha1.ServerGroupPolicySoftAffinity), orcv1alpha1.ServerGroupPolicySoftAffinity),
		Entry(string(orcv1alpha1.ServerGroupPolicySoftAntiAffinity), orcv1alpha1.ServerGroupPolicySoftAntiAffinity),
	)

	It("should permit maxServerPerHost with anti-affinity policy", func(ctx context.Context) {
		serverGroup := serverGroupStub(namespace)
		patch := baseServerGroupPatch(serverGroup)
		patch.Spec.WithResource(applyconfigv1alpha1.ServerGroupResourceSpec().
			WithPolicy(orcv1alpha1.ServerGroupPolicyAntiAffinity).
			WithRules(applyconfigv1alpha1.ServerGroupRules().WithMaxServerPerHost(2)))
		Expect(applyObj(ctx, serverGroup, patch)).To(Succeed())
	})

	It("should reject maxServerPerHost with non-anti-affinity policy", func(ctx context.Context) {
		serverGroup := serverGroupStub(namespace)
		patch := baseServerGroupPatch(serverGroup)
		patch.Spec.WithResource(applyconfigv1alpha1.ServerGroupResourceSpec().
			WithPolicy(orcv1alpha1.ServerGroupPolicyAffinity).
			WithRules(applyconfigv1alpha1.ServerGroupRules().WithMaxServerPerHost(2)))
		Expect(applyObj(ctx, serverGroup, patch)).To(MatchError(ContainSubstring("maxServerPerHost can only be used with the anti-affinity policy")))
	})
})
