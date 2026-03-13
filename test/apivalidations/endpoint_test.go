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
	endpointName = "endpoint"
	endpointID   = "265c9e4f-0f5a-46e4-9f3f-fb8de25ae128"
)

func endpointStub(namespace *corev1.Namespace) *orcv1alpha1.Endpoint {
	obj := &orcv1alpha1.Endpoint{}
	obj.Name = endpointName
	obj.Namespace = namespace.Name
	return obj
}

func testEndpointResource() *applyconfigv1alpha1.EndpointResourceSpecApplyConfiguration {
	return applyconfigv1alpha1.EndpointResourceSpec().
		WithInterface("public").
		WithURL("https://example.com").
		WithServiceRef("my-service")
}

func baseEndpointPatch(endpoint client.Object) *applyconfigv1alpha1.EndpointApplyConfiguration {
	return applyconfigv1alpha1.Endpoint(endpoint.GetName(), endpoint.GetNamespace()).
		WithSpec(applyconfigv1alpha1.EndpointSpec().
			WithCloudCredentialsRef(testCredentials()))
}

func testEndpointImport() *applyconfigv1alpha1.EndpointImportApplyConfiguration {
	return applyconfigv1alpha1.EndpointImport().WithID(endpointID)
}

var _ = Describe("ORC Endpoint API validations", func() {
	var namespace *corev1.Namespace
	BeforeEach(func() {
		namespace = createNamespace()
	})

	runManagementPolicyTests(func() *corev1.Namespace { return namespace }, managementPolicyTestArgs[*applyconfigv1alpha1.EndpointApplyConfiguration]{
		createObject:  func(ns *corev1.Namespace) client.Object { return endpointStub(ns) },
		basePatch:     func(obj client.Object) *applyconfigv1alpha1.EndpointApplyConfiguration { return baseEndpointPatch(obj) },
		applyResource: func(p *applyconfigv1alpha1.EndpointApplyConfiguration) { p.Spec.WithResource(testEndpointResource()) },
		applyImport:   func(p *applyconfigv1alpha1.EndpointApplyConfiguration) { p.Spec.WithImport(testEndpointImport()) },
		applyEmptyImport: func(p *applyconfigv1alpha1.EndpointApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.EndpointImport())
		},
		applyEmptyFilter: func(p *applyconfigv1alpha1.EndpointApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.EndpointImport().WithFilter(applyconfigv1alpha1.EndpointFilter()))
		},
		applyValidFilter: func(p *applyconfigv1alpha1.EndpointApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.EndpointImport().WithFilter(applyconfigv1alpha1.EndpointFilter().WithInterface("public")))
		},
		applyManaged: func(p *applyconfigv1alpha1.EndpointApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged)
		},
		applyUnmanaged: func(p *applyconfigv1alpha1.EndpointApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged)
		},
		applyManagedOptions: func(p *applyconfigv1alpha1.EndpointApplyConfiguration) {
			p.Spec.WithManagedOptions(applyconfigv1alpha1.ManagedOptions().WithOnDelete(orcv1alpha1.OnDeleteDetach))
		},
		getManagementPolicy: func(obj client.Object) orcv1alpha1.ManagementPolicy {
			return obj.(*orcv1alpha1.Endpoint).Spec.ManagementPolicy
		},
		getOnDelete: func(obj client.Object) orcv1alpha1.OnDelete {
			return obj.(*orcv1alpha1.Endpoint).Spec.ManagedOptions.OnDelete
		},
	})

	It("should reject an endpoint without required fields", func(ctx context.Context) {
		endpoint := endpointStub(namespace)
		patch := baseEndpointPatch(endpoint)
		patch.Spec.WithResource(applyconfigv1alpha1.EndpointResourceSpec())
		Expect(applyObj(ctx, endpoint, patch)).NotTo(Succeed())

		patch.Spec.WithResource(applyconfigv1alpha1.EndpointResourceSpec().
			WithInterface("public").WithServiceRef("my-service"))
		Expect(applyObj(ctx, endpoint, patch)).To(MatchError(ContainSubstring("spec.resource.url")))

		patch.Spec.WithResource(applyconfigv1alpha1.EndpointResourceSpec().
			WithURL("https://example.com").WithServiceRef("my-service"))
		Expect(applyObj(ctx, endpoint, patch)).To(MatchError(ContainSubstring("spec.resource.interface")))

		patch.Spec.WithResource(applyconfigv1alpha1.EndpointResourceSpec().
			WithInterface("public").WithURL("https://example.com"))
		Expect(applyObj(ctx, endpoint, patch)).To(MatchError(ContainSubstring("spec.resource.serviceRef")))
	})

	It("should reject invalid interface enum value", func(ctx context.Context) {
		endpoint := endpointStub(namespace)
		patch := baseEndpointPatch(endpoint)
		patch.Spec.WithResource(applyconfigv1alpha1.EndpointResourceSpec().
			WithInterface("invalid").
			WithURL("https://example.com").
			WithServiceRef("my-service"))
		Expect(applyObj(ctx, endpoint, patch)).NotTo(Succeed())
	})

	DescribeTable("should permit valid interface enum values",
		func(ctx context.Context, iface string) {
			endpoint := endpointStub(namespace)
			patch := baseEndpointPatch(endpoint)
			patch.Spec.WithResource(applyconfigv1alpha1.EndpointResourceSpec().
				WithInterface(iface).
				WithURL("https://example.com").
				WithServiceRef("my-service"))
			Expect(applyObj(ctx, endpoint, patch)).To(Succeed())
		},
		Entry("admin", "admin"),
		Entry("internal", "internal"),
		Entry("public", "public"),
	)

	It("should have immutable serviceRef", func(ctx context.Context) {
		endpoint := endpointStub(namespace)
		patch := baseEndpointPatch(endpoint)
		patch.Spec.WithResource(applyconfigv1alpha1.EndpointResourceSpec().
			WithInterface("public").
			WithURL("https://example.com").
			WithServiceRef("service-a"))
		Expect(applyObj(ctx, endpoint, patch)).To(Succeed())

		patch.Spec.WithResource(applyconfigv1alpha1.EndpointResourceSpec().
			WithInterface("public").
			WithURL("https://example.com").
			WithServiceRef("service-b"))
		Expect(applyObj(ctx, endpoint, patch)).To(MatchError(ContainSubstring("serviceRef is immutable")))
	})

	It("should have immutable description", func(ctx context.Context) {
		endpoint := endpointStub(namespace)
		patch := baseEndpointPatch(endpoint)
		patch.Spec.WithResource(applyconfigv1alpha1.EndpointResourceSpec().
			WithInterface("public").
			WithURL("https://example.com").
			WithServiceRef("my-service").
			WithDescription("desc-a"))
		Expect(applyObj(ctx, endpoint, patch)).To(Succeed())

		patch.Spec.WithResource(applyconfigv1alpha1.EndpointResourceSpec().
			WithInterface("public").
			WithURL("https://example.com").
			WithServiceRef("my-service").
			WithDescription("desc-b"))
		Expect(applyObj(ctx, endpoint, patch)).To(MatchError(ContainSubstring("description is immutable")))
	})
})
