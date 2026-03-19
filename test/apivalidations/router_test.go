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
	routerObjName = "router"
	routerID      = "265c9e4f-0f5a-46e4-9f3f-fb8de25ae132"
)

func routerStub(namespace *corev1.Namespace) *orcv1alpha1.Router {
	obj := &orcv1alpha1.Router{}
	obj.Name = routerObjName
	obj.Namespace = namespace.Name
	return obj
}

func testRouterResource() *applyconfigv1alpha1.RouterResourceSpecApplyConfiguration {
	return applyconfigv1alpha1.RouterResourceSpec()
}

func baseRouterPatch(router client.Object) *applyconfigv1alpha1.RouterApplyConfiguration {
	return applyconfigv1alpha1.Router(router.GetName(), router.GetNamespace()).
		WithSpec(applyconfigv1alpha1.RouterSpec().
			WithCloudCredentialsRef(testCredentials()))
}

func testRouterImport() *applyconfigv1alpha1.RouterImportApplyConfiguration {
	return applyconfigv1alpha1.RouterImport().WithID(routerID)
}

var _ = Describe("ORC Router API validations", func() {
	var namespace *corev1.Namespace
	BeforeEach(func() {
		namespace = createNamespace()
	})

	runManagementPolicyTests(func() *corev1.Namespace { return namespace }, managementPolicyTestArgs[*applyconfigv1alpha1.RouterApplyConfiguration]{
		createObject: func(ns *corev1.Namespace) client.Object { return routerStub(ns) },
		basePatch: func(obj client.Object) *applyconfigv1alpha1.RouterApplyConfiguration {
			return baseRouterPatch(obj)
		},
		applyResource: func(p *applyconfigv1alpha1.RouterApplyConfiguration) {
			p.Spec.WithResource(testRouterResource())
		},
		applyImport: func(p *applyconfigv1alpha1.RouterApplyConfiguration) {
			p.Spec.WithImport(testRouterImport())
		},
		applyEmptyImport: func(p *applyconfigv1alpha1.RouterApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.RouterImport())
		},
		applyEmptyFilter: func(p *applyconfigv1alpha1.RouterApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.RouterImport().WithFilter(applyconfigv1alpha1.RouterFilter()))
		},
		applyValidFilter: func(p *applyconfigv1alpha1.RouterApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.RouterImport().WithFilter(applyconfigv1alpha1.RouterFilter().WithName("foo")))
		},
		applyManaged: func(p *applyconfigv1alpha1.RouterApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged)
		},
		applyUnmanaged: func(p *applyconfigv1alpha1.RouterApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged)
		},
		applyManagedOptions: func(p *applyconfigv1alpha1.RouterApplyConfiguration) {
			p.Spec.WithManagedOptions(applyconfigv1alpha1.ManagedOptions().WithOnDelete(orcv1alpha1.OnDeleteDetach))
		},
		getManagementPolicy: func(obj client.Object) orcv1alpha1.ManagementPolicy {
			return obj.(*orcv1alpha1.Router).Spec.ManagementPolicy
		},
		getOnDelete: func(obj client.Object) orcv1alpha1.OnDelete {
			return obj.(*orcv1alpha1.Router).Spec.ManagedOptions.OnDelete
		},
	})

	It("should have immutable externalGateways", func(ctx context.Context) {
		router := routerStub(namespace)
		patch := baseRouterPatch(router)
		patch.Spec.WithResource(applyconfigv1alpha1.RouterResourceSpec().
			WithExternalGateways(applyconfigv1alpha1.ExternalGateway().WithNetworkRef("net-a")))
		Expect(applyObj(ctx, router, patch)).To(Succeed())

		patch.Spec.WithResource(applyconfigv1alpha1.RouterResourceSpec().
			WithExternalGateways(applyconfigv1alpha1.ExternalGateway().WithNetworkRef("net-b")))
		Expect(applyObj(ctx, router, patch)).To(MatchError(ContainSubstring("externalGateways is immutable")))
	})

	It("should have immutable distributed", func(ctx context.Context) {
		router := routerStub(namespace)
		patch := baseRouterPatch(router)
		patch.Spec.WithResource(applyconfigv1alpha1.RouterResourceSpec().
			WithDistributed(true))
		Expect(applyObj(ctx, router, patch)).To(Succeed())

		patch.Spec.WithResource(applyconfigv1alpha1.RouterResourceSpec().
			WithDistributed(false))
		Expect(applyObj(ctx, router, patch)).To(MatchError(ContainSubstring("distributed is immutable")))
	})

	It("should have immutable projectRef", func(ctx context.Context) {
		router := routerStub(namespace)
		patch := baseRouterPatch(router)
		patch.Spec.WithResource(applyconfigv1alpha1.RouterResourceSpec().
			WithProjectRef("project-a"))
		Expect(applyObj(ctx, router, patch)).To(Succeed())

		patch.Spec.WithResource(applyconfigv1alpha1.RouterResourceSpec().
			WithProjectRef("project-b"))
		Expect(applyObj(ctx, router, patch)).To(MatchError(ContainSubstring("projectRef is immutable")))
	})

	It("should reject duplicate tags", func(ctx context.Context) {
		router := routerStub(namespace)
		patch := baseRouterPatch(router)
		patch.Spec.WithResource(applyconfigv1alpha1.RouterResourceSpec().
			WithTags("foo", "bar", "foo"))
		Expect(applyObj(ctx, router, patch)).NotTo(Succeed())
	})
})
