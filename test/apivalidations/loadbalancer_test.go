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
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	applyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/pkg/clients/applyconfiguration/api/v1alpha1"
)

const (
	loadBalancerName = "loadbalancer"
	loadBalancerID   = "265c9e4f-0f5a-46e4-9f3f-fb8de25ae135"
)

func loadBalancerStub(namespace *corev1.Namespace) *orcv1alpha1.LoadBalancer {
	obj := &orcv1alpha1.LoadBalancer{}
	obj.Name = loadBalancerName
	obj.Namespace = namespace.Name
	return obj
}

func testLoadBalancerResource() *applyconfigv1alpha1.LoadBalancerResourceSpecApplyConfiguration {
	return applyconfigv1alpha1.LoadBalancerResourceSpec().
		WithSubnetRef("my-subnet")
}

func baseLoadBalancerPatch(lb client.Object) *applyconfigv1alpha1.LoadBalancerApplyConfiguration {
	return applyconfigv1alpha1.LoadBalancer(lb.GetName(), lb.GetNamespace()).
		WithSpec(applyconfigv1alpha1.LoadBalancerSpec().
			WithCloudCredentialsRef(testCredentials()))
}

func testLoadBalancerImport() *applyconfigv1alpha1.LoadBalancerImportApplyConfiguration {
	return applyconfigv1alpha1.LoadBalancerImport().WithID(loadBalancerID)
}

var _ = Describe("ORC LoadBalancer API validations", func() {
	var namespace *corev1.Namespace
	BeforeEach(func() {
		namespace = createNamespace()
	})

	runManagementPolicyTests(func() *corev1.Namespace { return namespace }, managementPolicyTestArgs[*applyconfigv1alpha1.LoadBalancerApplyConfiguration]{
		createObject: func(ns *corev1.Namespace) client.Object { return loadBalancerStub(ns) },
		basePatch: func(obj client.Object) *applyconfigv1alpha1.LoadBalancerApplyConfiguration {
			return baseLoadBalancerPatch(obj)
		},
		applyResource: func(p *applyconfigv1alpha1.LoadBalancerApplyConfiguration) {
			p.Spec.WithResource(testLoadBalancerResource())
		},
		applyImport: func(p *applyconfigv1alpha1.LoadBalancerApplyConfiguration) {
			p.Spec.WithImport(testLoadBalancerImport())
		},
		applyEmptyImport: func(p *applyconfigv1alpha1.LoadBalancerApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.LoadBalancerImport())
		},
		applyEmptyFilter: func(p *applyconfigv1alpha1.LoadBalancerApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.LoadBalancerImport().WithFilter(applyconfigv1alpha1.LoadBalancerFilter()))
		},
		applyValidFilter: func(p *applyconfigv1alpha1.LoadBalancerApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.LoadBalancerImport().WithFilter(applyconfigv1alpha1.LoadBalancerFilter().WithName("my-lb")))
		},
		applyManaged: func(p *applyconfigv1alpha1.LoadBalancerApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged)
		},
		applyUnmanaged: func(p *applyconfigv1alpha1.LoadBalancerApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged)
		},
		applyManagedOptions: func(p *applyconfigv1alpha1.LoadBalancerApplyConfiguration) {
			p.Spec.WithManagedOptions(applyconfigv1alpha1.ManagedOptions().WithOnDelete(orcv1alpha1.OnDeleteDetach))
		},
		getManagementPolicy: func(obj client.Object) orcv1alpha1.ManagementPolicy {
			return obj.(*orcv1alpha1.LoadBalancer).Spec.ManagementPolicy
		},
		getOnDelete: func(obj client.Object) orcv1alpha1.OnDelete {
			return obj.(*orcv1alpha1.LoadBalancer).Spec.ManagedOptions.OnDelete
		},
	})

	It("should require exactly one of subnetRef, networkRef, or vipPortRef", func(ctx context.Context) {
		lb := loadBalancerStub(namespace)
		patch := baseLoadBalancerPatch(lb)

		// None set
		patch.Spec.WithResource(applyconfigv1alpha1.LoadBalancerResourceSpec())
		Expect(applyObj(ctx, lb, patch)).To(MatchError(ContainSubstring("exactly one of subnetRef, networkRef, or vipPortRef must be set")))

		// Two set: subnetRef and networkRef
		patch.Spec.WithResource(applyconfigv1alpha1.LoadBalancerResourceSpec().
			WithSubnetRef("my-subnet").
			WithNetworkRef("my-network"))
		Expect(applyObj(ctx, lb, patch)).To(MatchError(ContainSubstring("exactly one of subnetRef, networkRef, or vipPortRef must be set")))

		// Two set: subnetRef and vipPortRef
		patch.Spec.WithResource(applyconfigv1alpha1.LoadBalancerResourceSpec().
			WithSubnetRef("my-subnet").
			WithVIPPortRef("my-port"))
		Expect(applyObj(ctx, lb, patch)).To(MatchError(ContainSubstring("exactly one of subnetRef, networkRef, or vipPortRef must be set")))

		// Two set: networkRef and vipPortRef
		patch.Spec.WithResource(applyconfigv1alpha1.LoadBalancerResourceSpec().
			WithNetworkRef("my-network").
			WithVIPPortRef("my-port"))
		Expect(applyObj(ctx, lb, patch)).To(MatchError(ContainSubstring("exactly one of subnetRef, networkRef, or vipPortRef must be set")))

		// All three set
		patch.Spec.WithResource(applyconfigv1alpha1.LoadBalancerResourceSpec().
			WithSubnetRef("my-subnet").
			WithNetworkRef("my-network").
			WithVIPPortRef("my-port"))
		Expect(applyObj(ctx, lb, patch)).To(MatchError(ContainSubstring("exactly one of subnetRef, networkRef, or vipPortRef must be set")))

		// Only subnetRef set - should succeed
		patch.Spec.WithResource(applyconfigv1alpha1.LoadBalancerResourceSpec().
			WithSubnetRef("my-subnet"))
		Expect(applyObj(ctx, lb, patch)).To(Succeed())
	})

	It("should succeed when only networkRef is set", func(ctx context.Context) {
		lb := loadBalancerStub(namespace)
		patch := baseLoadBalancerPatch(lb)
		patch.Spec.WithResource(applyconfigv1alpha1.LoadBalancerResourceSpec().
			WithNetworkRef("my-network"))
		Expect(applyObj(ctx, lb, patch)).To(Succeed())
	})

	It("should succeed when only vipPortRef is set", func(ctx context.Context) {
		lb := loadBalancerStub(namespace)
		patch := baseLoadBalancerPatch(lb)
		patch.Spec.WithResource(applyconfigv1alpha1.LoadBalancerResourceSpec().
			WithVIPPortRef("my-port"))
		Expect(applyObj(ctx, lb, patch)).To(Succeed())
	})

	It("should have immutable subnetRef", func(ctx context.Context) {
		lb := loadBalancerStub(namespace)
		patch := baseLoadBalancerPatch(lb)
		patch.Spec.WithResource(applyconfigv1alpha1.LoadBalancerResourceSpec().
			WithSubnetRef("subnet-a"))
		Expect(applyObj(ctx, lb, patch)).To(Succeed())

		patch.Spec.WithResource(applyconfigv1alpha1.LoadBalancerResourceSpec().
			WithSubnetRef("subnet-b"))
		Expect(applyObj(ctx, lb, patch)).To(MatchError(ContainSubstring("subnetRef is immutable")))
	})

	It("should have immutable networkRef", func(ctx context.Context) {
		lb := loadBalancerStub(namespace)
		patch := baseLoadBalancerPatch(lb)
		patch.Spec.WithResource(applyconfigv1alpha1.LoadBalancerResourceSpec().
			WithNetworkRef("network-a"))
		Expect(applyObj(ctx, lb, patch)).To(Succeed())

		patch.Spec.WithResource(applyconfigv1alpha1.LoadBalancerResourceSpec().
			WithNetworkRef("network-b"))
		Expect(applyObj(ctx, lb, patch)).To(MatchError(ContainSubstring("networkRef is immutable")))
	})

	It("should have immutable vipPortRef", func(ctx context.Context) {
		lb := loadBalancerStub(namespace)
		patch := baseLoadBalancerPatch(lb)
		patch.Spec.WithResource(applyconfigv1alpha1.LoadBalancerResourceSpec().
			WithVIPPortRef("port-a"))
		Expect(applyObj(ctx, lb, patch)).To(Succeed())

		patch.Spec.WithResource(applyconfigv1alpha1.LoadBalancerResourceSpec().
			WithVIPPortRef("port-b"))
		Expect(applyObj(ctx, lb, patch)).To(MatchError(ContainSubstring("vipPortRef is immutable")))
	})

	It("should have immutable flavor", func(ctx context.Context) {
		lb := loadBalancerStub(namespace)
		patch := baseLoadBalancerPatch(lb)
		patch.Spec.WithResource(applyconfigv1alpha1.LoadBalancerResourceSpec().
			WithSubnetRef("my-subnet").
			WithFlavor("flavor-a"))
		Expect(applyObj(ctx, lb, patch)).To(Succeed())

		patch.Spec.WithResource(applyconfigv1alpha1.LoadBalancerResourceSpec().
			WithSubnetRef("my-subnet").
			WithFlavor("flavor-b"))
		Expect(applyObj(ctx, lb, patch)).To(MatchError(ContainSubstring("flavor is immutable")))
	})

	It("should have immutable projectRef", func(ctx context.Context) {
		lb := loadBalancerStub(namespace)
		patch := baseLoadBalancerPatch(lb)
		patch.Spec.WithResource(applyconfigv1alpha1.LoadBalancerResourceSpec().
			WithSubnetRef("my-subnet").
			WithProjectRef("project-a"))
		Expect(applyObj(ctx, lb, patch)).To(Succeed())

		patch.Spec.WithResource(applyconfigv1alpha1.LoadBalancerResourceSpec().
			WithSubnetRef("my-subnet").
			WithProjectRef("project-b"))
		Expect(applyObj(ctx, lb, patch)).To(MatchError(ContainSubstring("projectRef is immutable")))
	})

	It("should allow up to 64 tags", func(ctx context.Context) {
		lb := loadBalancerStub(namespace)
		patch := baseLoadBalancerPatch(lb)

		// Build 64 unique tags
		tags := make([]orcv1alpha1.NeutronTag, 64)
		for i := range tags {
			tags[i] = orcv1alpha1.NeutronTag(fmt.Sprintf("tag-%02d", i))
		}
		patch.Spec.WithResource(applyconfigv1alpha1.LoadBalancerResourceSpec().
			WithSubnetRef("my-subnet").
			WithTags(tags...))
		Expect(applyObj(ctx, lb, patch)).To(Succeed())
	})

	It("should reject more than 64 tags", func(ctx context.Context) {
		lb := loadBalancerStub(namespace)
		patch := baseLoadBalancerPatch(lb)

		// Build 65 unique tags
		tags := make([]orcv1alpha1.NeutronTag, 65)
		for i := range tags {
			tags[i] = orcv1alpha1.NeutronTag(fmt.Sprintf("tag-%02d", i))
		}
		patch.Spec.WithResource(applyconfigv1alpha1.LoadBalancerResourceSpec().
			WithSubnetRef("my-subnet").
			WithTags(tags...))
		Expect(applyObj(ctx, lb, patch)).To(MatchError(ContainSubstring("spec.resource.tags: Too many: 65: must have at most 64 items")))
	})

	It("should reject duplicate tags", func(ctx context.Context) {
		lb := loadBalancerStub(namespace)
		patch := baseLoadBalancerPatch(lb)
		patch.Spec.WithResource(applyconfigv1alpha1.LoadBalancerResourceSpec().
			WithSubnetRef("my-subnet").
			WithTags("foo", "bar", "foo"))
		Expect(applyObj(ctx, lb, patch)).NotTo(Succeed())
	})
})
