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
	floatingIPName = "floatingip"
	floatingIPID   = "265c9e4f-0f5a-46e4-9f3f-fb8de25ae130"
)

func floatingIPStub(namespace *corev1.Namespace) *orcv1alpha1.FloatingIP {
	obj := &orcv1alpha1.FloatingIP{}
	obj.Name = floatingIPName
	obj.Namespace = namespace.Name
	return obj
}

func testFloatingIPResource() *applyconfigv1alpha1.FloatingIPResourceSpecApplyConfiguration {
	return applyconfigv1alpha1.FloatingIPResourceSpec().
		WithFloatingNetworkRef("my-network")
}

func baseFloatingIPPatch(fip client.Object) *applyconfigv1alpha1.FloatingIPApplyConfiguration {
	return applyconfigv1alpha1.FloatingIP(fip.GetName(), fip.GetNamespace()).
		WithSpec(applyconfigv1alpha1.FloatingIPSpec().
			WithCloudCredentialsRef(testCredentials()))
}

func testFloatingIPImport() *applyconfigv1alpha1.FloatingIPImportApplyConfiguration {
	return applyconfigv1alpha1.FloatingIPImport().WithID(floatingIPID)
}

var _ = Describe("ORC FloatingIP API validations", func() {
	var namespace *corev1.Namespace
	BeforeEach(func() {
		namespace = createNamespace()
	})

	runManagementPolicyTests(func() *corev1.Namespace { return namespace }, managementPolicyTestArgs[*applyconfigv1alpha1.FloatingIPApplyConfiguration]{
		createObject: func(ns *corev1.Namespace) client.Object { return floatingIPStub(ns) },
		basePatch: func(obj client.Object) *applyconfigv1alpha1.FloatingIPApplyConfiguration {
			return baseFloatingIPPatch(obj)
		},
		applyResource: func(p *applyconfigv1alpha1.FloatingIPApplyConfiguration) {
			p.Spec.WithResource(testFloatingIPResource())
		},
		applyImport: func(p *applyconfigv1alpha1.FloatingIPApplyConfiguration) {
			p.Spec.WithImport(testFloatingIPImport())
		},
		applyEmptyImport: func(p *applyconfigv1alpha1.FloatingIPApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.FloatingIPImport())
		},
		applyEmptyFilter: func(p *applyconfigv1alpha1.FloatingIPApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.FloatingIPImport().WithFilter(applyconfigv1alpha1.FloatingIPFilter()))
		},
		applyValidFilter: func(p *applyconfigv1alpha1.FloatingIPApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.FloatingIPImport().WithFilter(applyconfigv1alpha1.FloatingIPFilter().WithFloatingNetworkRef("my-network")))
		},
		applyManaged: func(p *applyconfigv1alpha1.FloatingIPApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged)
		},
		applyUnmanaged: func(p *applyconfigv1alpha1.FloatingIPApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged)
		},
		applyManagedOptions: func(p *applyconfigv1alpha1.FloatingIPApplyConfiguration) {
			p.Spec.WithManagedOptions(applyconfigv1alpha1.ManagedOptions().WithOnDelete(orcv1alpha1.OnDeleteDetach))
		},
		getManagementPolicy: func(obj client.Object) orcv1alpha1.ManagementPolicy {
			return obj.(*orcv1alpha1.FloatingIP).Spec.ManagementPolicy
		},
		getOnDelete: func(obj client.Object) orcv1alpha1.OnDelete {
			return obj.(*orcv1alpha1.FloatingIP).Spec.ManagedOptions.OnDelete
		},
	})

	It("should require exactly one of floatingNetworkRef or floatingSubnetRef", func(ctx context.Context) {
		fip := floatingIPStub(namespace)
		patch := baseFloatingIPPatch(fip)

		// Neither set
		patch.Spec.WithResource(applyconfigv1alpha1.FloatingIPResourceSpec())
		Expect(applyObj(ctx, fip, patch)).To(MatchError(ContainSubstring("Exactly one of 'floatingNetworkRef' or 'floatingSubnetRef' must be set")))

		// Both set
		patch.Spec.WithResource(applyconfigv1alpha1.FloatingIPResourceSpec().
			WithFloatingNetworkRef("net-a").
			WithFloatingSubnetRef("subnet-a"))
		Expect(applyObj(ctx, fip, patch)).To(MatchError(ContainSubstring("Exactly one of 'floatingNetworkRef' or 'floatingSubnetRef' must be set")))

		// Only floatingSubnetRef set - should succeed
		patch.Spec.WithResource(applyconfigv1alpha1.FloatingIPResourceSpec().
			WithFloatingSubnetRef("subnet-a"))
		Expect(applyObj(ctx, fip, patch)).To(Succeed())
	})

	It("should have immutable floatingNetworkRef", func(ctx context.Context) {
		fip := floatingIPStub(namespace)
		patch := baseFloatingIPPatch(fip)
		patch.Spec.WithResource(applyconfigv1alpha1.FloatingIPResourceSpec().
			WithFloatingNetworkRef("net-a"))
		Expect(applyObj(ctx, fip, patch)).To(Succeed())

		patch.Spec.WithResource(applyconfigv1alpha1.FloatingIPResourceSpec().
			WithFloatingNetworkRef("net-b"))
		Expect(applyObj(ctx, fip, patch)).To(MatchError(ContainSubstring("floatingNetworkRef is immutable")))
	})

	It("should have immutable floatingSubnetRef", func(ctx context.Context) {
		fip := floatingIPStub(namespace)
		patch := baseFloatingIPPatch(fip)
		patch.Spec.WithResource(applyconfigv1alpha1.FloatingIPResourceSpec().
			WithFloatingSubnetRef("subnet-a"))
		Expect(applyObj(ctx, fip, patch)).To(Succeed())

		patch.Spec.WithResource(applyconfigv1alpha1.FloatingIPResourceSpec().
			WithFloatingSubnetRef("subnet-b"))
		Expect(applyObj(ctx, fip, patch)).To(MatchError(ContainSubstring("floatingSubnetRef is immutable")))
	})

	It("should have immutable portRef", func(ctx context.Context) {
		fip := floatingIPStub(namespace)
		patch := baseFloatingIPPatch(fip)
		patch.Spec.WithResource(applyconfigv1alpha1.FloatingIPResourceSpec().
			WithFloatingNetworkRef("my-network").
			WithPortRef("port-a"))
		Expect(applyObj(ctx, fip, patch)).To(Succeed())

		patch.Spec.WithResource(applyconfigv1alpha1.FloatingIPResourceSpec().
			WithFloatingNetworkRef("my-network").
			WithPortRef("port-b"))
		Expect(applyObj(ctx, fip, patch)).To(MatchError(ContainSubstring("portRef is immutable")))
	})

	It("should have immutable projectRef", func(ctx context.Context) {
		fip := floatingIPStub(namespace)
		patch := baseFloatingIPPatch(fip)
		patch.Spec.WithResource(applyconfigv1alpha1.FloatingIPResourceSpec().
			WithFloatingNetworkRef("my-network").
			WithProjectRef("project-a"))
		Expect(applyObj(ctx, fip, patch)).To(Succeed())

		patch.Spec.WithResource(applyconfigv1alpha1.FloatingIPResourceSpec().
			WithFloatingNetworkRef("my-network").
			WithProjectRef("project-b"))
		Expect(applyObj(ctx, fip, patch)).To(MatchError(ContainSubstring("projectRef is immutable")))
	})
})
