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

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	applyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/pkg/clients/applyconfiguration/api/v1alpha1"
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
	return applyconfigv1alpha1.FlavorResourceSpec().WithVcpus(1).WithRAM(1).WithDisk(1)
}

func baseFlavorPatch(flavor client.Object) *applyconfigv1alpha1.FlavorApplyConfiguration {
	return applyconfigv1alpha1.Flavor(flavor.GetName(), flavor.GetNamespace()).
		WithSpec(applyconfigv1alpha1.FlavorSpec().
			WithCloudCredentialsRef(testCredentials()))
}

func testFlavorImport() *applyconfigv1alpha1.FlavorImportApplyConfiguration {
	return applyconfigv1alpha1.FlavorImport().WithID(flavorID)
}

var _ = Describe("ORC Flavor API validations", func() {
	var namespace *corev1.Namespace
	BeforeEach(func() {
		namespace = createNamespace()
	})

	runManagementPolicyTests(func() *corev1.Namespace { return namespace }, managementPolicyTestArgs[*applyconfigv1alpha1.FlavorApplyConfiguration]{
		createObject: func(ns *corev1.Namespace) client.Object { return flavorStub(ns) },
		basePatch: func(obj client.Object) *applyconfigv1alpha1.FlavorApplyConfiguration {
			return baseFlavorPatch(obj)
		},
		applyResource: func(p *applyconfigv1alpha1.FlavorApplyConfiguration) {
			p.Spec.WithResource(testFlavorResource())
		},
		applyImport: func(p *applyconfigv1alpha1.FlavorApplyConfiguration) {
			p.Spec.WithImport(testFlavorImport())
		},
		applyEmptyImport: func(p *applyconfigv1alpha1.FlavorApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.FlavorImport())
		},
		applyEmptyFilter: func(p *applyconfigv1alpha1.FlavorApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.FlavorImport().WithFilter(applyconfigv1alpha1.FlavorFilter()))
		},
		applyValidFilter: func(p *applyconfigv1alpha1.FlavorApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.FlavorImport().WithFilter(applyconfigv1alpha1.FlavorFilter().WithName("foo")))
		},
		applyManaged: func(p *applyconfigv1alpha1.FlavorApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged)
		},
		applyUnmanaged: func(p *applyconfigv1alpha1.FlavorApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged)
		},
		applyManagedOptions: func(p *applyconfigv1alpha1.FlavorApplyConfiguration) {
			p.Spec.WithManagedOptions(applyconfigv1alpha1.ManagedOptions().WithOnDelete(orcv1alpha1.OnDeleteDetach))
		},
		getManagementPolicy: func(obj client.Object) orcv1alpha1.ManagementPolicy {
			return obj.(*orcv1alpha1.Flavor).Spec.ManagementPolicy
		},
		getOnDelete: func(obj client.Object) orcv1alpha1.OnDelete {
			return obj.(*orcv1alpha1.Flavor).Spec.ManagedOptions.OnDelete
		},
	})

	It("should be immutable", func(ctx context.Context) {
		flavor := flavorStub(namespace)
		patch := baseFlavorPatch(flavor)
		patch.Spec.WithResource(applyconfigv1alpha1.FlavorResourceSpec().WithRAM(1).WithVcpus(1).WithDisk(1))
		Expect(applyObj(ctx, flavor, patch)).To(Succeed())
		patch.Spec.WithResource(applyconfigv1alpha1.FlavorResourceSpec().WithRAM(2).WithVcpus(1).WithDisk(1))
		Expect(applyObj(ctx, flavor, patch)).To(MatchError(ContainSubstring("FlavorResourceSpec is immutable")))
	})

	It("should reject a flavor without required fields", func(ctx context.Context) {
		flavor := flavorStub(namespace)
		patch := baseFlavorPatch(flavor)
		patch.Spec.WithResource(applyconfigv1alpha1.FlavorResourceSpec())
		Expect(applyObj(ctx, flavor, patch)).To(MatchError(ContainSubstring("Required value")))
		patch.Spec.WithResource(applyconfigv1alpha1.FlavorResourceSpec().WithRAM(1).WithDisk(1))
		Expect(applyObj(ctx, flavor, patch)).To(MatchError(ContainSubstring("spec.resource.vcpus: Required value")))
		patch.Spec.WithResource(applyconfigv1alpha1.FlavorResourceSpec().WithVcpus(1).WithDisk(1))
		Expect(applyObj(ctx, flavor, patch)).To(MatchError(ContainSubstring("spec.resource.ram: Required value")))
		patch.Spec.WithResource(applyconfigv1alpha1.FlavorResourceSpec().WithRAM(1).WithVcpus(1))
		Expect(applyObj(ctx, flavor, patch)).To(MatchError(ContainSubstring("spec.resource.disk: Required value")))
	})

	It("should reject a flavor with values less than minimal", func(ctx context.Context) {
		flavor := flavorStub(namespace)
		patch := baseFlavorPatch(flavor)
		patch.Spec.WithResource(applyconfigv1alpha1.FlavorResourceSpec().WithRAM(1).WithVcpus(0).WithDisk(1))
		Expect(applyObj(ctx, flavor, patch)).To(MatchError(ContainSubstring("spec.resource.vcpus in body should be greater than or equal to 1")))
		patch.Spec.WithResource(applyconfigv1alpha1.FlavorResourceSpec().WithRAM(0).WithVcpus(1).WithDisk(1))
		Expect(applyObj(ctx, flavor, patch)).To(MatchError(ContainSubstring("spec.resource.ram in body should be greater than or equal to 1")))
		patch.Spec.WithResource(applyconfigv1alpha1.FlavorResourceSpec().WithRAM(1).WithVcpus(1).WithDisk(-1))
		Expect(applyObj(ctx, flavor, patch)).To(MatchError(ContainSubstring("spec.resource.disk in body should be greater than or equal to 0")))
	})

	It("should reject a flavor with values greater than max", func(ctx context.Context) {
		flavor := flavorStub(namespace)
		patch := baseFlavorPatch(flavor)
		maxString := strings.Repeat("a", 65536)
		patch.Spec.WithResource(applyconfigv1alpha1.FlavorResourceSpec().WithRAM(1).WithVcpus(1).WithDescription(maxString))
		Expect(applyObj(ctx, flavor, patch)).To(MatchError(ContainSubstring("spec.resource.description: Too long")))

	})

	It("should reject import filter with value less than minimal", func(ctx context.Context) {
		flavor := flavorStub(namespace)
		patch := baseFlavorPatch(flavor)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(applyconfigv1alpha1.FlavorImport().
				WithFilter(applyconfigv1alpha1.FlavorFilter().WithRAM(0)))
		Expect(applyObj(ctx, flavor, patch)).To(MatchError(ContainSubstring("spec.import.filter.ram in body should be greater than or equal to 1")))
	})
})
