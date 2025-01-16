/*
Copyright 2024 The ORC Authors.

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

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	applyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/pkg/clients/applyconfiguration/api/v1alpha1"
)

const (
	flavorName = "flavor"
	flavorID   = "265c9e4f-0f5a-46e4-9f3f-fb8de25ae12e"
)

func flavorStub(namespace *corev1.Namespace) *orcv1alpha1.Flavor {
	obj := &orcv1alpha1.Flavor{}
	obj.Name = flavorName
	obj.Namespace = namespace.Name
	return obj
}

func testFlavorResource() *applyconfigv1alpha1.FlavorResourceSpecApplyConfiguration {
	return applyconfigv1alpha1.FlavorResourceSpec().WithVcpus(1).WithRAM(1)
}

func baseFlavorPatch(flavor client.Object) *applyconfigv1alpha1.FlavorApplyConfiguration {
	return applyconfigv1alpha1.Flavor(flavor.GetName(), flavor.GetNamespace()).
		WithSpec(applyconfigv1alpha1.FlavorSpec().
			WithCloudCredentialsRef(testCredentials()))
}

func baseWorkingFlavorPatch(flavor client.Object) *applyconfigv1alpha1.FlavorApplyConfiguration {
	patch := baseFlavorPatch(flavor)
	patch.Spec.WithResource(applyconfigv1alpha1.FlavorResourceSpec().WithRAM(1).WithVcpus(1))
	return patch
}

func testFlavorImport() *applyconfigv1alpha1.FlavorImportApplyConfiguration {
	return applyconfigv1alpha1.FlavorImport().WithID(flavorID)
}

var _ = Describe("ORC Flavor API validations", func() {
	var namespace *corev1.Namespace
	BeforeEach(func() {
		namespace = createNamespace()
	})

	It("should allow to create a minimal flavor and managementPolicy should default to managed", func(ctx context.Context) {
		flavor := flavorStub(namespace)
		patch := baseFlavorPatch(flavor)
		patch.Spec.WithResource(applyconfigv1alpha1.FlavorResourceSpec().WithRAM(1).WithVcpus(1))
		Expect(applyObj(ctx, flavor, patch)).To(Succeed())
		Expect(flavor.Spec.ManagementPolicy).To(Equal(orcv1alpha1.ManagementPolicyManaged))
	})

	It("should be immutable", func(ctx context.Context) {
		flavor := flavorStub(namespace)
		patch := baseFlavorPatch(flavor)
		patch.Spec.WithResource(applyconfigv1alpha1.FlavorResourceSpec().WithRAM(1).WithVcpus(1))
		Expect(applyObj(ctx, flavor, patch)).To(Succeed())
		patch.Spec.WithResource(applyconfigv1alpha1.FlavorResourceSpec().WithRAM(2))
		Expect(applyObj(ctx, flavor, patch)).NotTo(Succeed())
	})

	It("should reject a flavor without required fields", func(ctx context.Context) {
		flavor := flavorStub(namespace)
		patch := baseFlavorPatch(flavor)
		patch.Spec.WithResource(applyconfigv1alpha1.FlavorResourceSpec())
		Expect(applyObj(ctx, flavor, patch)).NotTo(Succeed())
		patch.Spec.WithResource(applyconfigv1alpha1.FlavorResourceSpec().WithRAM(1))
		Expect(applyObj(ctx, flavor, patch)).NotTo(Succeed())
		patch.Spec.WithResource(applyconfigv1alpha1.FlavorResourceSpec().WithVcpus(1))
		Expect(applyObj(ctx, flavor, patch)).NotTo(Succeed())
	})

	It("should reject a flavor with values less than minimal", func(ctx context.Context) {
		flavor := flavorStub(namespace)
		patch := baseFlavorPatch(flavor)
		patch.Spec.WithResource(applyconfigv1alpha1.FlavorResourceSpec().WithRAM(1).WithVcpus(0))
		Expect(applyObj(ctx, flavor, patch)).NotTo(Succeed())
		patch.Spec.WithResource(applyconfigv1alpha1.FlavorResourceSpec().WithRAM(0).WithVcpus(1))
		Expect(applyObj(ctx, flavor, patch)).NotTo(Succeed())
	})

	It("should reject a flavor with values greater than max", func(ctx context.Context) {
		flavor := flavorStub(namespace)
		patch := baseFlavorPatch(flavor)
		maxString := strings.Repeat("a", 65536)
		patch.Spec.WithResource(applyconfigv1alpha1.FlavorResourceSpec().WithRAM(1).WithVcpus(1).WithDescription(maxString))
		Expect(applyObj(ctx, flavor, patch)).NotTo(Succeed())

	})
	It("should default to managementPolicy managed", func(ctx context.Context) {
		flavor := flavorStub(namespace)
		flavor.Spec.Resource = &orcv1alpha1.FlavorResourceSpec{
			RAM:   1,
			Vcpus: 1,
		}
		flavor.Spec.CloudCredentialsRef = orcv1alpha1.CloudCredentialsReference{
			SecretName: "my-secret",
			CloudName:  "my-cloud",
		}

		Expect(k8sClient.Create(ctx, flavor)).To(Succeed())
		Expect(flavor.Spec.ManagementPolicy).To(Equal(orcv1alpha1.ManagementPolicyManaged))
	})

	It("should require import for unmanaged", func(ctx context.Context) {
		flavor := flavorStub(namespace)
		patch := baseFlavorPatch(flavor)
		patch.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged)
		Expect(applyObj(ctx, flavor, patch)).NotTo(Succeed())

		patch.Spec.WithImport(testFlavorImport())
		Expect(applyObj(ctx, flavor, patch)).To(Succeed())
	})

	It("should not permit unmanaged with resource", func(ctx context.Context) {
		flavor := flavorStub(namespace)
		patch := baseFlavorPatch(flavor)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(testFlavorImport()).
			WithResource(testFlavorResource())
		Expect(applyObj(ctx, flavor, patch)).NotTo(Succeed())
	})

	It("should not permit empty import", func(ctx context.Context) {
		flavor := flavorStub(namespace)
		patch := baseFlavorPatch(flavor)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(applyconfigv1alpha1.FlavorImport())
		Expect(applyObj(ctx, flavor, patch)).NotTo(Succeed())
	})

	It("should not permit empty import filter", func(ctx context.Context) {
		flavor := flavorStub(namespace)
		patch := baseFlavorPatch(flavor)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(applyconfigv1alpha1.FlavorImport().
				WithFilter(applyconfigv1alpha1.FlavorFilter()))
		Expect(applyObj(ctx, flavor, patch)).NotTo(Succeed())
	})

	It("should permit import filter with name", func(ctx context.Context) {
		flavor := flavorStub(namespace)
		patch := baseFlavorPatch(flavor)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(applyconfigv1alpha1.FlavorImport().
				WithFilter(applyconfigv1alpha1.FlavorFilter().WithName("foo")))
		Expect(applyObj(ctx, flavor, patch)).To(Succeed())
	})

	It("should require resource for managed", func(ctx context.Context) {
		flavor := flavorStub(namespace)
		patch := baseFlavorPatch(flavor)
		patch.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged)
		Expect(applyObj(ctx, flavor, patch)).NotTo(Succeed())

		patch.Spec.WithResource(testFlavorResource())
		Expect(applyObj(ctx, flavor, patch)).To(Succeed())
	})

	It("should not permit managed with import", func(ctx context.Context) {
		flavor := flavorStub(namespace)
		patch := baseFlavorPatch(flavor)
		patch.Spec.
			WithImport(testFlavorImport()).
			WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged).
			WithResource(testFlavorResource())
		Expect(applyObj(ctx, flavor, patch)).NotTo(Succeed())
	})

	It("should not permit managedOptions for unmanaged", func(ctx context.Context) {
		flavor := flavorStub(namespace)
		patch := baseFlavorPatch(flavor)
		patch.Spec.
			WithImport(testFlavorImport()).
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithManagedOptions(applyconfigv1alpha1.ManagedOptions().
				WithOnDelete(orcv1alpha1.OnDeleteDetach))
		Expect(applyObj(ctx, flavor, patch)).NotTo(Succeed())
	})

	It("should permit managedOptions for managed", func(ctx context.Context) {
		flavor := flavorStub(namespace)
		patch := baseWorkingFlavorPatch(flavor)
		patch.Spec.
			WithManagedOptions(applyconfigv1alpha1.ManagedOptions().
				WithOnDelete(orcv1alpha1.OnDeleteDetach))
		Expect(applyObj(ctx, flavor, patch)).To(Succeed())
		Expect(flavor.Spec.ManagedOptions.OnDelete).To(Equal(orcv1alpha1.OnDelete("detach")))
	})
})
