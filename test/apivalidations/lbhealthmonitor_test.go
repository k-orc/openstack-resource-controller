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
	lbhealthmonitorName = "lbhealthmonitor"
	lbhealthmonitorID   = "265c9e4f-0f5a-46e4-9f3f-fb8de25ae120"
)

func lbhealthmonitorStub(namespace *corev1.Namespace) *orcv1alpha1.LBHealthMonitor {
	obj := &orcv1alpha1.LBHealthMonitor{}
	obj.Name = lbhealthmonitorName
	obj.Namespace = namespace.Name
	return obj
}

func testLBHealthMonitorResource() *applyconfigv1alpha1.LBHealthMonitorResourceSpecApplyConfiguration {
	return applyconfigv1alpha1.LBHealthMonitorResourceSpec().
		WithPoolRef("lbpool").
		WithType("HTTP").
		WithDelay(10).
		WithTimeout(5).
		WithMaxRetries(3)
}

func baseLBHealthMonitorPatch(obj client.Object) *applyconfigv1alpha1.LBHealthMonitorApplyConfiguration {
	return applyconfigv1alpha1.LBHealthMonitor(obj.GetName(), obj.GetNamespace()).
		WithSpec(applyconfigv1alpha1.LBHealthMonitorSpec().
			WithCloudCredentialsRef(testCredentials()))
}

func testLBHealthMonitorImport() *applyconfigv1alpha1.LBHealthMonitorImportApplyConfiguration {
	return applyconfigv1alpha1.LBHealthMonitorImport().WithID(lbhealthmonitorID)
}

var _ = Describe("ORC LBHealthMonitor API validations", func() {
	var namespace *corev1.Namespace
	BeforeEach(func() {
		namespace = createNamespace()
	})

	runManagementPolicyTests(func() *corev1.Namespace { return namespace }, managementPolicyTestArgs[*applyconfigv1alpha1.LBHealthMonitorApplyConfiguration]{
		createObject: func(ns *corev1.Namespace) client.Object { return lbhealthmonitorStub(ns) },
		basePatch: func(obj client.Object) *applyconfigv1alpha1.LBHealthMonitorApplyConfiguration {
			return baseLBHealthMonitorPatch(obj)
		},
		applyResource: func(p *applyconfigv1alpha1.LBHealthMonitorApplyConfiguration) {
			p.Spec.WithResource(testLBHealthMonitorResource())
		},
		applyImport: func(p *applyconfigv1alpha1.LBHealthMonitorApplyConfiguration) {
			p.Spec.WithImport(testLBHealthMonitorImport())
		},
		applyEmptyImport: func(p *applyconfigv1alpha1.LBHealthMonitorApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.LBHealthMonitorImport())
		},
		applyEmptyFilter: func(p *applyconfigv1alpha1.LBHealthMonitorApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.LBHealthMonitorImport().WithFilter(applyconfigv1alpha1.LBHealthMonitorFilter()))
		},
		applyValidFilter: func(p *applyconfigv1alpha1.LBHealthMonitorApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.LBHealthMonitorImport().WithFilter(applyconfigv1alpha1.LBHealthMonitorFilter().WithName("foo")))
		},
		applyManaged: func(p *applyconfigv1alpha1.LBHealthMonitorApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged)
		},
		applyUnmanaged: func(p *applyconfigv1alpha1.LBHealthMonitorApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged)
		},
		applyManagedOptions: func(p *applyconfigv1alpha1.LBHealthMonitorApplyConfiguration) {
			p.Spec.WithManagedOptions(applyconfigv1alpha1.ManagedOptions().WithOnDelete(orcv1alpha1.OnDeleteDetach))
		},
		getManagementPolicy: func(obj client.Object) orcv1alpha1.ManagementPolicy {
			return obj.(*orcv1alpha1.LBHealthMonitor).Spec.ManagementPolicy
		},
		getOnDelete: func(obj client.Object) orcv1alpha1.OnDelete {
			return obj.(*orcv1alpha1.LBHealthMonitor).Spec.ManagedOptions.OnDelete
		},
	})

	It("should reject a lbhealthmonitor without required fields", func(ctx context.Context) {
		obj := lbhealthmonitorStub(namespace)
		patch := baseLBHealthMonitorPatch(obj)
		patch.Spec.WithResource(applyconfigv1alpha1.LBHealthMonitorResourceSpec())
		Expect(applyObj(ctx, obj, patch)).NotTo(Succeed())
	})

	DescribeTable("should have immutable fields",
		func(ctx context.Context, field string, set func(*applyconfigv1alpha1.LBHealthMonitorResourceSpecApplyConfiguration)) {
			obj := lbhealthmonitorStub(namespace)
			patch := baseLBHealthMonitorPatch(obj)
			patch.Spec.WithResource(testLBHealthMonitorResource())
			Expect(applyObj(ctx, obj, patch)).To(Succeed())

			resource := testLBHealthMonitorResource()
			set(resource)
			patch.Spec.WithResource(resource)
			Expect(applyObj(ctx, obj, patch)).To(MatchError(ContainSubstring(field + " is immutable")))
		},
		Entry("poolRef", "poolRef", func(r *applyconfigv1alpha1.LBHealthMonitorResourceSpecApplyConfiguration) {
			r.WithPoolRef("other")
		}),
		Entry("type", "type", func(r *applyconfigv1alpha1.LBHealthMonitorResourceSpecApplyConfiguration) {
			r.WithType("TCP")
		}),
	)

	It("should reject a timeout that isn't less than the delay", func(ctx context.Context) {
		obj := lbhealthmonitorStub(namespace)
		patch := baseLBHealthMonitorPatch(obj)
		patch.Spec.WithResource(testLBHealthMonitorResource().WithTimeout(10))
		Expect(applyObj(ctx, obj, patch)).To(MatchError(ContainSubstring("timeout must be less than delay")))
	})

	DescribeTable("should reject invalid values",
		func(ctx context.Context, set func(*applyconfigv1alpha1.LBHealthMonitorResourceSpecApplyConfiguration)) {
			obj := lbhealthmonitorStub(namespace)
			patch := baseLBHealthMonitorPatch(obj)
			resource := testLBHealthMonitorResource()
			set(resource)
			patch.Spec.WithResource(resource)
			Expect(applyObj(ctx, obj, patch)).NotTo(Succeed())
		},
		Entry("unknown type", func(r *applyconfigv1alpha1.LBHealthMonitorResourceSpecApplyConfiguration) { r.WithType("ICMP") }),
		Entry("maxRetries above 10", func(r *applyconfigv1alpha1.LBHealthMonitorResourceSpecApplyConfiguration) { r.WithMaxRetries(11) }),
		Entry("maxRetriesDown below 1", func(r *applyconfigv1alpha1.LBHealthMonitorResourceSpecApplyConfiguration) { r.WithMaxRetriesDown(0) }),
		Entry("urlPath without a leading slash", func(r *applyconfigv1alpha1.LBHealthMonitorResourceSpecApplyConfiguration) { r.WithURLPath("healthz") }),
		Entry("malformed expectedCodes", func(r *applyconfigv1alpha1.LBHealthMonitorResourceSpecApplyConfiguration) { r.WithExpectedCodes("2xx") }),
		Entry("unknown httpVersion", func(r *applyconfigv1alpha1.LBHealthMonitorResourceSpecApplyConfiguration) { r.WithHTTPVersion("2.0") }),
	)

	DescribeTable("should permit valid expectedCodes",
		func(ctx context.Context, codes string) {
			obj := lbhealthmonitorStub(namespace)
			patch := baseLBHealthMonitorPatch(obj)
			patch.Spec.WithResource(testLBHealthMonitorResource().WithExpectedCodes(codes))
			Expect(applyObj(ctx, obj, patch)).To(Succeed())
		},
		Entry("single", "200"),
		Entry("list", "200,202"),
		Entry("range", "200-204"),
	)
})
