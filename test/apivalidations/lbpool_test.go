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

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/api/v1alpha1"
	applyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/pkg/clients/applyconfiguration/api/v1alpha1"
)

const (
	lbpoolName = "lbpool"
	lbpoolID   = "265c9e4f-0f5a-46e4-9f3f-fb8de25ae120"
)

func lbpoolStub(namespace *corev1.Namespace) *orcv1alpha1.LBPool {
	obj := &orcv1alpha1.LBPool{}
	obj.Name = lbpoolName
	obj.Namespace = namespace.Name
	return obj
}

func testLBPoolResource() *applyconfigv1alpha1.LBPoolResourceSpecApplyConfiguration {
	return applyconfigv1alpha1.LBPoolResourceSpec().
		WithLoadBalancerRef("loadbalancer").
		WithLBAlgorithm(orcv1alpha1.LBPoolLBAlgorithmRoundRobin).
		WithProtocol(orcv1alpha1.LBPoolProtocolHTTP)
}

func baseLBPoolPatch(obj client.Object) *applyconfigv1alpha1.LBPoolApplyConfiguration {
	return applyconfigv1alpha1.LBPool(obj.GetName(), obj.GetNamespace()).
		WithSpec(applyconfigv1alpha1.LBPoolSpec().
			WithCloudCredentialsRef(testCredentials()))
}

func testLBPoolImport() *applyconfigv1alpha1.LBPoolImportApplyConfiguration {
	return applyconfigv1alpha1.LBPoolImport().WithID(lbpoolID)
}

var _ = Describe("ORC LBPool API validations", func() {
	var namespace *corev1.Namespace
	BeforeEach(func() {
		namespace = createNamespace()
	})

	runManagementPolicyTests(func() *corev1.Namespace { return namespace }, managementPolicyTestArgs[*applyconfigv1alpha1.LBPoolApplyConfiguration]{
		createObject: func(ns *corev1.Namespace) client.Object { return lbpoolStub(ns) },
		basePatch: func(obj client.Object) *applyconfigv1alpha1.LBPoolApplyConfiguration {
			return baseLBPoolPatch(obj)
		},
		applyResource: func(p *applyconfigv1alpha1.LBPoolApplyConfiguration) {
			p.Spec.WithResource(testLBPoolResource())
		},
		applyImport: func(p *applyconfigv1alpha1.LBPoolApplyConfiguration) {
			p.Spec.WithImport(testLBPoolImport())
		},
		applyEmptyImport: func(p *applyconfigv1alpha1.LBPoolApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.LBPoolImport())
		},
		applyEmptyFilter: func(p *applyconfigv1alpha1.LBPoolApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.LBPoolImport().WithFilter(applyconfigv1alpha1.LBPoolFilter()))
		},
		applyValidFilter: func(p *applyconfigv1alpha1.LBPoolApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.LBPoolImport().WithFilter(applyconfigv1alpha1.LBPoolFilter().WithName("foo")))
		},
		applyManaged: func(p *applyconfigv1alpha1.LBPoolApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged)
		},
		applyUnmanaged: func(p *applyconfigv1alpha1.LBPoolApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged)
		},
		applyManagedOptions: func(p *applyconfigv1alpha1.LBPoolApplyConfiguration) {
			p.Spec.WithManagedOptions(applyconfigv1alpha1.ManagedOptions().WithOnDelete(orcv1alpha1.OnDeleteDetach))
		},
		getManagementPolicy: func(obj client.Object) orcv1alpha1.ManagementPolicy {
			return obj.(*orcv1alpha1.LBPool).Spec.ManagementPolicy
		},
		getOnDelete: func(obj client.Object) orcv1alpha1.OnDelete {
			return obj.(*orcv1alpha1.LBPool).Spec.ManagedOptions.OnDelete
		},
	})

	It("should require loadBalancerRef or listenerRef", func(ctx context.Context) {
		obj := lbpoolStub(namespace)
		patch := baseLBPoolPatch(obj)
		patch.Spec.WithResource(applyconfigv1alpha1.LBPoolResourceSpec().
			WithLBAlgorithm(orcv1alpha1.LBPoolLBAlgorithmRoundRobin).
			WithProtocol(orcv1alpha1.LBPoolProtocolHTTP))
		Expect(applyObj(ctx, obj, patch)).To(MatchError(ContainSubstring("either loadBalancerRef or listenerRef must be specified")))
	})

	It("should permit changing lbAlgorithm", func(ctx context.Context) {
		obj := lbpoolStub(namespace)
		patch := baseLBPoolPatch(obj)
		patch.Spec.WithResource(testLBPoolResource())
		Expect(applyObj(ctx, obj, patch)).To(Succeed())

		patch.Spec.WithResource(testLBPoolResource().WithLBAlgorithm(orcv1alpha1.LBPoolLBAlgorithmLeastConnections))
		Expect(applyObj(ctx, obj, patch)).To(Succeed())
	})

	DescribeTable("should have immutable fields",
		func(ctx context.Context, field string, set func(*applyconfigv1alpha1.LBPoolResourceSpecApplyConfiguration, string)) {
			obj := lbpoolStub(namespace)
			patch := baseLBPoolPatch(obj)
			resource := testLBPoolResource()
			set(resource, "a")
			patch.Spec.WithResource(resource)
			Expect(applyObj(ctx, obj, patch)).To(Succeed())

			resource = testLBPoolResource()
			set(resource, "b")
			patch.Spec.WithResource(resource)
			Expect(applyObj(ctx, obj, patch)).To(MatchError(ContainSubstring(field + " is immutable")))
		},
		Entry("protocol", "protocol", func(r *applyconfigv1alpha1.LBPoolResourceSpecApplyConfiguration, v string) {
			if v == "b" {
				r.WithProtocol(orcv1alpha1.LBPoolProtocolTCP)
			}
		}),
		Entry("loadBalancerRef", "loadBalancerRef", func(r *applyconfigv1alpha1.LBPoolResourceSpecApplyConfiguration, v string) {
			r.WithLoadBalancerRef(orcv1alpha1.KubernetesNameRef("loadbalancer-" + v))
		}),
		Entry("listenerRef", "listenerRef", func(r *applyconfigv1alpha1.LBPoolResourceSpecApplyConfiguration, v string) {
			r.WithListenerRef(orcv1alpha1.KubernetesNameRef("listener-" + v))
		}),
		Entry("projectRef", "projectRef", func(r *applyconfigv1alpha1.LBPoolResourceSpecApplyConfiguration, v string) {
			r.WithProjectRef(orcv1alpha1.KubernetesNameRef("project-" + v))
		}),
	)
})
