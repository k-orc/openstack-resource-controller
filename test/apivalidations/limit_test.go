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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	applyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/pkg/clients/applyconfiguration/api/v1alpha1"
)

const (
	limitName = "limit"
	limitID   = "265c9e4f-0f5a-46e4-9f3f-fb8de25ae120"
)

func limitStub(namespace *corev1.Namespace) *orcv1alpha1.Limit {
	obj := &orcv1alpha1.Limit{}
	obj.Name = limitName
	obj.Namespace = namespace.Name
	return obj
}

func testLimitResource() *applyconfigv1alpha1.LimitResourceSpecApplyConfiguration {
	return applyconfigv1alpha1.LimitResourceSpec().
		WithServiceRef("nova").
		WithResourceName("servers").
		WithResourceLimit(10)
}

func testLimitResourceWithProject() *applyconfigv1alpha1.LimitResourceSpecApplyConfiguration {
	return testLimitResource().WithProjectRef("demo")
}

func baseLimitPatch(obj client.Object) *applyconfigv1alpha1.LimitApplyConfiguration {
	return applyconfigv1alpha1.Limit(obj.GetName(), obj.GetNamespace()).
		WithSpec(applyconfigv1alpha1.LimitSpec().
			WithCloudCredentialsRef(testCredentials()))
}

func testLimitImport() *applyconfigv1alpha1.LimitImportApplyConfiguration {
	return applyconfigv1alpha1.LimitImport().WithID(limitID)
}

var _ = Describe("ORC Limit API validations", func() {
	var namespace *corev1.Namespace
	BeforeEach(func() {
		namespace = createNamespace()
	})

	runManagementPolicyTests(func() *corev1.Namespace { return namespace }, managementPolicyTestArgs[*applyconfigv1alpha1.LimitApplyConfiguration]{
		createObject: func(ns *corev1.Namespace) client.Object { return limitStub(ns) },
		basePatch: func(obj client.Object) *applyconfigv1alpha1.LimitApplyConfiguration {
			return baseLimitPatch(obj)
		},
		applyResource: func(p *applyconfigv1alpha1.LimitApplyConfiguration) {
			p.Spec.WithResource(testLimitResourceWithProject())
		},
		applyImport: func(p *applyconfigv1alpha1.LimitApplyConfiguration) {
			p.Spec.WithImport(testLimitImport())
		},
		applyEmptyImport: func(p *applyconfigv1alpha1.LimitApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.LimitImport())
		},
		applyEmptyFilter: func(p *applyconfigv1alpha1.LimitApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.LimitImport().WithFilter(applyconfigv1alpha1.LimitFilter()))
		},
		applyValidFilter: func(p *applyconfigv1alpha1.LimitApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.LimitImport().WithFilter(applyconfigv1alpha1.LimitFilter().WithServiceRef("foo")))
		},
		applyManaged: func(p *applyconfigv1alpha1.LimitApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged)
		},
		applyUnmanaged: func(p *applyconfigv1alpha1.LimitApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged)
		},
		applyManagedOptions: func(p *applyconfigv1alpha1.LimitApplyConfiguration) {
			p.Spec.WithManagedOptions(applyconfigv1alpha1.ManagedOptions().WithOnDelete(orcv1alpha1.OnDeleteDetach))
		},
		getManagementPolicy: func(obj client.Object) orcv1alpha1.ManagementPolicy {
			return obj.(*orcv1alpha1.Limit).Spec.ManagementPolicy
		},
		getOnDelete: func(obj client.Object) orcv1alpha1.OnDelete {
			return obj.(*orcv1alpha1.Limit).Spec.ManagedOptions.OnDelete
		},
	})

	It("should reject a limit without required fields", func(ctx context.Context) {
		obj := limitStub(namespace)
		patch := baseLimitPatch(obj)
		patch.Spec.WithResource(applyconfigv1alpha1.LimitResourceSpec())
		Expect(applyObj(ctx, obj, patch)).NotTo(Succeed())
	})

	It("should have immutable serviceRef", func(ctx context.Context) {
		obj := limitStub(namespace)
		patch := baseLimitPatch(obj)
		res := testLimitResourceWithProject()
		patch.Spec.WithResource(res.WithServiceRef("service-a"))
		Expect(applyObj(ctx, obj, patch)).To(Succeed())

		patch.Spec.WithResource(res.WithServiceRef("service-b"))
		Expect(applyObj(ctx, obj, patch)).To(MatchError(ContainSubstring("serviceRef is immutable")))
	})

	It("should have immutable projectRef", func(ctx context.Context) {
		obj := limitStub(namespace)
		patch := baseLimitPatch(obj)
		patch.Spec.WithResource(testLimitResource().
			WithProjectRef("project-a"))
		Expect(applyObj(ctx, obj, patch)).To(Succeed())

		patch.Spec.WithResource(testLimitResource().
			WithProjectRef("project-b"))
		Expect(applyObj(ctx, obj, patch)).To(MatchError(ContainSubstring("projectRef is immutable")))
	})

	It("should have immutable domainRef", func(ctx context.Context) {
		obj := limitStub(namespace)
		patch := baseLimitPatch(obj)
		patch.Spec.WithResource(testLimitResource().
			WithDomainRef("domain-a"))
		Expect(applyObj(ctx, obj, patch)).To(Succeed())

		patch.Spec.WithResource(testLimitResource().
			WithDomainRef("domain-b"))
		Expect(applyObj(ctx, obj, patch)).To(MatchError(ContainSubstring("domainRef is immutable")))
	})

	It("should have immutable resourceName", func(ctx context.Context) {
		obj := limitStub(namespace)
		patch := baseLimitPatch(obj)
		res := testLimitResourceWithProject()

		patch.Spec.WithResource(res.WithResourceName("servers"))
		Expect(applyObj(ctx, obj, patch)).To(Succeed())

		patch.Spec.WithResource(res.WithResourceName("server_key_pairs"))
		Expect(applyObj(ctx, obj, patch)).To(MatchError(ContainSubstring("resourceName is immutable")))
	})

	It("should reject invalid resourceName", func(ctx context.Context) {
		obj := limitStub(namespace)
		patch := baseLimitPatch(obj)
		res := testLimitResourceWithProject()

		patch.Spec.WithResource(res.WithResourceName("invalid-name-\r\n"))
		Expect(applyObj(ctx, obj, patch)).To(MatchError(ContainSubstring(`spec.resource.resourceName in body should match '^[\S]+$'`)))

		patch = baseLimitPatch(obj)
		patch.Spec.WithResource(res.WithResourceName(strings.Repeat("a", 256)))
		Expect(applyObj(ctx, obj, patch)).To(MatchError(ContainSubstring(`spec.resource.resourceName: Too long: may not be longer than 255`)))
	})

	It("should reject invalid resourceLimit", func(ctx context.Context) {
		obj := limitStub(namespace)
		patch := baseLimitPatch(obj)
		patch.Spec.WithResource(testLimitResourceWithProject().WithResourceLimit(-2))
		Expect(applyObj(ctx, obj, patch)).To(MatchError(ContainSubstring(`spec.resource.resourceLimit in body should be greater than or equal to -1`)))
	})

	It("should include either projectRef or domainRef but not both", func(ctx context.Context) {
		obj := limitStub(namespace)
		patch := baseLimitPatch(obj)
		res := testLimitResource()

		patch.Spec.WithResource(res)
		Expect(applyObj(ctx, obj, patch)).To(MatchError(ContainSubstring(`either projectRef or domainRef must be specified`)))

		patch.Spec.WithResource(res.WithProjectRef("demo").WithDomainRef("default"))
		Expect(applyObj(ctx, obj, patch)).To(MatchError(ContainSubstring(`projectRef and domainRef are mutually exclusive`)))
	})
})
