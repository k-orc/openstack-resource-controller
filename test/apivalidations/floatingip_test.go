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

	It("should allow to create a minimal floatingip and managementPolicy should default to managed", func(ctx context.Context) {
		fip := floatingIPStub(namespace)
		patch := baseFloatingIPPatch(fip)
		patch.Spec.WithResource(testFloatingIPResource())
		Expect(applyObj(ctx, fip, patch)).To(Succeed())
		Expect(fip.Spec.ManagementPolicy).To(Equal(orcv1alpha1.ManagementPolicyManaged))
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

	It("should require import for unmanaged", func(ctx context.Context) {
		fip := floatingIPStub(namespace)
		patch := baseFloatingIPPatch(fip)
		patch.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged)
		Expect(applyObj(ctx, fip, patch)).To(MatchError(ContainSubstring("import must be specified when policy is unmanaged")))

		patch.Spec.WithImport(testFloatingIPImport())
		Expect(applyObj(ctx, fip, patch)).To(Succeed())
	})

	It("should not permit unmanaged with resource", func(ctx context.Context) {
		fip := floatingIPStub(namespace)
		patch := baseFloatingIPPatch(fip)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(testFloatingIPImport()).
			WithResource(testFloatingIPResource())
		Expect(applyObj(ctx, fip, patch)).To(MatchError(ContainSubstring("resource may not be specified when policy is unmanaged")))
	})

	It("should not permit empty import", func(ctx context.Context) {
		fip := floatingIPStub(namespace)
		patch := baseFloatingIPPatch(fip)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(applyconfigv1alpha1.FloatingIPImport())
		Expect(applyObj(ctx, fip, patch)).To(MatchError(ContainSubstring("spec.import in body should have at least 1 properties")))
	})

	It("should not permit empty import filter", func(ctx context.Context) {
		fip := floatingIPStub(namespace)
		patch := baseFloatingIPPatch(fip)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(applyconfigv1alpha1.FloatingIPImport().
				WithFilter(applyconfigv1alpha1.FloatingIPFilter()))
		Expect(applyObj(ctx, fip, patch)).To(MatchError(ContainSubstring("spec.import.filter in body should have at least 1 properties")))
	})

	It("should permit import filter with floatingNetworkRef", func(ctx context.Context) {
		fip := floatingIPStub(namespace)
		patch := baseFloatingIPPatch(fip)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(applyconfigv1alpha1.FloatingIPImport().
				WithFilter(applyconfigv1alpha1.FloatingIPFilter().WithFloatingNetworkRef("my-network")))
		Expect(applyObj(ctx, fip, patch)).To(Succeed())
	})

	It("should require resource for managed", func(ctx context.Context) {
		fip := floatingIPStub(namespace)
		patch := baseFloatingIPPatch(fip)
		patch.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged)
		Expect(applyObj(ctx, fip, patch)).To(MatchError(ContainSubstring("resource must be specified when policy is managed")))

		patch.Spec.WithResource(testFloatingIPResource())
		Expect(applyObj(ctx, fip, patch)).To(Succeed())
	})

	It("should not permit managed with import", func(ctx context.Context) {
		fip := floatingIPStub(namespace)
		patch := baseFloatingIPPatch(fip)
		patch.Spec.
			WithImport(testFloatingIPImport()).
			WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged).
			WithResource(testFloatingIPResource())
		Expect(applyObj(ctx, fip, patch)).To(MatchError(ContainSubstring("import may not be specified when policy is managed")))
	})

	It("should not permit managedOptions for unmanaged", func(ctx context.Context) {
		fip := floatingIPStub(namespace)
		patch := baseFloatingIPPatch(fip)
		patch.Spec.
			WithImport(testFloatingIPImport()).
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithManagedOptions(applyconfigv1alpha1.ManagedOptions().
				WithOnDelete(orcv1alpha1.OnDeleteDetach))
		Expect(applyObj(ctx, fip, patch)).To(MatchError(ContainSubstring("managedOptions may only be provided when policy is managed")))
	})

	It("should permit managedOptions for managed", func(ctx context.Context) {
		fip := floatingIPStub(namespace)
		patch := baseFloatingIPPatch(fip)
		patch.Spec.WithResource(testFloatingIPResource()).
			WithManagedOptions(applyconfigv1alpha1.ManagedOptions().
				WithOnDelete(orcv1alpha1.OnDeleteDetach))
		Expect(applyObj(ctx, fip, patch)).To(Succeed())
		Expect(fip.Spec.ManagedOptions.OnDelete).To(Equal(orcv1alpha1.OnDelete("detach")))
	})
})
