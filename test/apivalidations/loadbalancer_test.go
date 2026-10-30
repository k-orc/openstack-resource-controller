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
	loadbalancerName = "loadbalancer"
	loadbalancerID   = "265c9e4f-0f5a-46e4-9f3f-fb8de25ae120"
)

func loadbalancerStub(namespace *corev1.Namespace) *orcv1alpha1.LoadBalancer {
	obj := &orcv1alpha1.LoadBalancer{}
	obj.Name = loadbalancerName
	obj.Namespace = namespace.Name
	return obj
}

func testLoadBalancerResource() *applyconfigv1alpha1.LoadBalancerResourceSpecApplyConfiguration {
	return applyconfigv1alpha1.LoadBalancerResourceSpec().WithVIPSubnetRef("subnet")
}

func baseLoadBalancerPatch(obj client.Object) *applyconfigv1alpha1.LoadBalancerApplyConfiguration {
	return applyconfigv1alpha1.LoadBalancer(obj.GetName(), obj.GetNamespace()).
		WithSpec(applyconfigv1alpha1.LoadBalancerSpec().
			WithCloudCredentialsRef(testCredentials()))
}

func testLoadBalancerImport() *applyconfigv1alpha1.LoadBalancerImportApplyConfiguration {
	return applyconfigv1alpha1.LoadBalancerImport().WithID(loadbalancerID)
}

var _ = Describe("ORC LoadBalancer API validations", func() {
	var namespace *corev1.Namespace
	BeforeEach(func() {
		namespace = createNamespace()
	})

	runManagementPolicyTests(func() *corev1.Namespace { return namespace }, managementPolicyTestArgs[*applyconfigv1alpha1.LoadBalancerApplyConfiguration]{
		createObject: func(ns *corev1.Namespace) client.Object { return loadbalancerStub(ns) },
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
			p.Spec.WithImport(applyconfigv1alpha1.LoadBalancerImport().WithFilter(applyconfigv1alpha1.LoadBalancerFilter().WithName("foo")))
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

	It("should require one of vipSubnetRef, vipNetworkRef or vipPortRef", func(ctx context.Context) {
		obj := loadbalancerStub(namespace)
		patch := baseLoadBalancerPatch(obj)
		patch.Spec.WithResource(applyconfigv1alpha1.LoadBalancerResourceSpec())
		Expect(applyObj(ctx, obj, patch)).To(MatchError(ContainSubstring("at least one of vipSubnetRef, vipNetworkRef, or vipPortRef must be specified")))

		patch.Spec.WithResource(applyconfigv1alpha1.LoadBalancerResourceSpec().WithVIPPortRef("port"))
		Expect(applyObj(ctx, obj, patch)).To(Succeed())
	})

	DescribeTable("should have immutable fields",
		func(ctx context.Context, field string, set func(*applyconfigv1alpha1.LoadBalancerResourceSpecApplyConfiguration, string)) {
			obj := loadbalancerStub(namespace)
			patch := baseLoadBalancerPatch(obj)
			resource := testLoadBalancerResource()
			set(resource, "a")
			patch.Spec.WithResource(resource)
			Expect(applyObj(ctx, obj, patch)).To(Succeed())

			resource = testLoadBalancerResource()
			set(resource, "b")
			patch.Spec.WithResource(resource)
			Expect(applyObj(ctx, obj, patch)).To(MatchError(ContainSubstring(field + " is immutable")))
		},
		Entry("vipSubnetRef", "vipSubnetRef", func(r *applyconfigv1alpha1.LoadBalancerResourceSpecApplyConfiguration, v string) {
			r.WithVIPSubnetRef(orcv1alpha1.KubernetesNameRef("subnet-" + v))
		}),
		Entry("vipNetworkRef", "vipNetworkRef", func(r *applyconfigv1alpha1.LoadBalancerResourceSpecApplyConfiguration, v string) {
			r.WithVIPNetworkRef(orcv1alpha1.KubernetesNameRef("network-" + v))
		}),
		Entry("vipPortRef", "vipPortRef", func(r *applyconfigv1alpha1.LoadBalancerResourceSpecApplyConfiguration, v string) {
			r.WithVIPPortRef(orcv1alpha1.KubernetesNameRef("port-" + v))
		}),
		Entry("projectRef", "projectRef", func(r *applyconfigv1alpha1.LoadBalancerResourceSpecApplyConfiguration, v string) {
			r.WithProjectRef(orcv1alpha1.KubernetesNameRef("project-" + v))
		}),
		Entry("availabilityZone", "availabilityZone", func(r *applyconfigv1alpha1.LoadBalancerResourceSpecApplyConfiguration, v string) {
			r.WithAvailabilityZone("az-" + v)
		}),
		Entry("provider", "provider", func(r *applyconfigv1alpha1.LoadBalancerResourceSpecApplyConfiguration, v string) {
			r.WithProvider("provider-" + v)
		}),
		Entry("vipAddress", "vipAddress", func(r *applyconfigv1alpha1.LoadBalancerResourceSpecApplyConfiguration, v string) {
			if v == "a" {
				r.WithVIPAddress("192.0.2.10")
			} else {
				r.WithVIPAddress("192.0.2.11")
			}
		}),
	)
})
