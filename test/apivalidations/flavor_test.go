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
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
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
	return applyconfigv1alpha1.FlavorResourceSpec().WithVcpus(1).WithRAM(1).WithDisk(1)
}

func baseFlavorPatch(flavor client.Object) *applyconfigv1alpha1.FlavorApplyConfiguration {
	return applyconfigv1alpha1.Flavor(flavor.GetName(), flavor.GetNamespace()).
		WithSpec(applyconfigv1alpha1.FlavorSpec().
			WithCloudCredentialsRef(testCredentials()))
}

func baseWorkingFlavorPatch(flavor client.Object) *applyconfigv1alpha1.FlavorApplyConfiguration {
	patch := baseFlavorPatch(flavor)
	patch.Spec.WithResource(applyconfigv1alpha1.FlavorResourceSpec().WithRAM(1).WithVcpus(1).WithDisk(1))
	return patch
}

func testFlavorImport() *applyconfigv1alpha1.FlavorImportApplyConfiguration {
	return applyconfigv1alpha1.FlavorImport().WithID(flavorID)
}

type getWithFlavorFn[argType, returnType any] func(*applyconfigv1alpha1.FlavorApplyConfiguration) func(argType) returnType

func testFlavorMutability[argType, returnType any](ctx context.Context, namespace *corev1.Namespace, getFn getWithFlavorFn[argType, returnType], valueA, valueB argType, allowsUnset bool, initFns ...func(*applyconfigv1alpha1.FlavorApplyConfiguration)) {
	setup := func() (client.Object, *applyconfigv1alpha1.FlavorApplyConfiguration, func(argType) returnType) {
		obj := flavorStub(namespace)
		patch := baseWorkingFlavorPatch(obj)
		for _, initFn := range initFns {
			initFn(patch)
		}
		withFn := getFn(patch)

		return obj, patch, withFn
	}

	if allowsUnset {
		obj, patch, withFn := setup()

		Expect(applyObj(ctx, obj, patch)).To(Succeed(), fmt.Sprintf("create with value unset: %s", format.Object(patch, 2)))

		withFn(valueA)
		Expect(applyObj(ctx, obj, patch)).NotTo(Succeed(), fmt.Sprintf("update with value set: %s", format.Object(patch, 2)))
	}

	obj, patch, withFn := setup()

	withFn(valueA)
	Expect(applyObj(ctx, obj, patch)).To(Succeed(), fmt.Sprintf("create with value '%v': %s", valueA, format.Object(patch, 2)))

	withFn(valueB)
	Expect(applyObj(ctx, obj, patch)).NotTo(Succeed(), fmt.Sprintf("update with value '%v': %s", valueB, format.Object(patch, 2)))
}

var _ = Describe("ORC Flavor API validations", func() {
	var namespace *corev1.Namespace
	BeforeEach(func() {
		namespace = createNamespace()
	})

	It("should allow to create a minimal flavor and managementPolicy should default to managed", func(ctx context.Context) {
		flavor := flavorStub(namespace)
		patch := baseFlavorPatch(flavor)
		patch.Spec.WithResource(applyconfigv1alpha1.FlavorResourceSpec().WithRAM(1).WithVcpus(1).WithDisk(1))
		Expect(applyObj(ctx, flavor, patch)).To(Succeed())
		Expect(flavor.Spec.ManagementPolicy).To(Equal(orcv1alpha1.ManagementPolicyManaged))
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
		Expect(applyObj(ctx, flavor, patch)).To(MatchError(ContainSubstring("import must be specified when policy is unmanaged")))

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
		Expect(applyObj(ctx, flavor, patch)).To(MatchError(ContainSubstring("resource may not be specified when policy is unmanaged")))
	})

	It("should not permit empty import", func(ctx context.Context) {
		flavor := flavorStub(namespace)
		patch := baseFlavorPatch(flavor)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(applyconfigv1alpha1.FlavorImport())
		Expect(applyObj(ctx, flavor, patch)).To(MatchError(ContainSubstring("spec.import in body should have at least 1 properties")))
	})

	It("should not permit empty import filter", func(ctx context.Context) {
		flavor := flavorStub(namespace)
		patch := baseFlavorPatch(flavor)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(applyconfigv1alpha1.FlavorImport().
				WithFilter(applyconfigv1alpha1.FlavorFilter()))
		Expect(applyObj(ctx, flavor, patch)).To(MatchError(ContainSubstring("spec.import.filter in body should have at least 1 properties")))
	})

	It("should permit import filter with values within bound", func(ctx context.Context) {
		flavor := flavorStub(namespace)
		patch := baseFlavorPatch(flavor)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(applyconfigv1alpha1.FlavorImport().
				WithFilter(applyconfigv1alpha1.FlavorFilter().
					WithName("foo").WithRAM(1)))
		Expect(applyObj(ctx, flavor, patch)).To(Succeed())
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

	It("should require resource for managed", func(ctx context.Context) {
		flavor := flavorStub(namespace)
		patch := baseFlavorPatch(flavor)
		patch.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged)
		Expect(applyObj(ctx, flavor, patch)).To(MatchError(ContainSubstring("resource must be specified when policy is managed")))

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
		Expect(applyObj(ctx, flavor, patch)).To(MatchError(ContainSubstring("import may not be specified when policy is managed")))
	})

	It("should not permit managedOptions for unmanaged", func(ctx context.Context) {
		flavor := flavorStub(namespace)
		patch := baseFlavorPatch(flavor)
		patch.Spec.
			WithImport(testFlavorImport()).
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithManagedOptions(applyconfigv1alpha1.ManagedOptions().
				WithOnDelete(orcv1alpha1.OnDeleteDetach))
		Expect(applyObj(ctx, flavor, patch)).To(MatchError(ContainSubstring("managedOptions may only be provided when policy is managed")))
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

	It("should not permit modifying flavor resource.name", func(ctx context.Context) {
		testFlavorMutability(ctx, namespace,
			func(applyConfig *applyconfigv1alpha1.FlavorApplyConfiguration) func(orcv1alpha1.OpenStackName) *applyconfigv1alpha1.FlavorResourceSpecApplyConfiguration {
				return applyConfig.Spec.Resource.WithName
			},
			"foo", "bar", false,
		)
	})

	It("should permit modifying flavor resource.vcpus", func(ctx context.Context) {
		testFlavorMutability(ctx, namespace,
			func(applyConfig *applyconfigv1alpha1.FlavorApplyConfiguration) func(int32) *applyconfigv1alpha1.FlavorResourceSpecApplyConfiguration {
				return applyConfig.Spec.Resource.WithVcpus
			},
			1, 2, false,
		)
	})

	It("should permit modifying flavor resource.ram", func(ctx context.Context) {
		testFlavorMutability(ctx, namespace,
			func(applyConfig *applyconfigv1alpha1.FlavorApplyConfiguration) func(int32) *applyconfigv1alpha1.FlavorResourceSpecApplyConfiguration {
				return applyConfig.Spec.Resource.WithRAM
			},
			1, 2, false,
		)
	})

	It("should permit modifying flavor resource.disk", func(ctx context.Context) {
		testFlavorMutability(ctx, namespace,
			func(applyConfig *applyconfigv1alpha1.FlavorApplyConfiguration) func(int32) *applyconfigv1alpha1.FlavorResourceSpecApplyConfiguration {
				return applyConfig.Spec.Resource.WithDisk
			},
			1, 2, false,
		)
	})

	It("should permit modifying flavor resource.swap", func(ctx context.Context) {
		testFlavorMutability(ctx, namespace,
			func(applyConfig *applyconfigv1alpha1.FlavorApplyConfiguration) func(int32) *applyconfigv1alpha1.FlavorResourceSpecApplyConfiguration {
				return applyConfig.Spec.Resource.WithSwap
			},
			1, 2, false,
		)
	})

	It("should permit modifying flavor resource.isPublic", func(ctx context.Context) {
		testFlavorMutability(ctx, namespace,
			func(applyConfig *applyconfigv1alpha1.FlavorApplyConfiguration) func(bool) *applyconfigv1alpha1.FlavorResourceSpecApplyConfiguration {
				return applyConfig.Spec.Resource.WithIsPublic
			},
			true, false, false,
		)
	})

	It("should permit modifying flavor resource.ephermal", func(ctx context.Context) {
		testFlavorMutability(ctx, namespace,
			func(applyConfig *applyconfigv1alpha1.FlavorApplyConfiguration) func(int32) *applyconfigv1alpha1.FlavorResourceSpecApplyConfiguration {
				return applyConfig.Spec.Resource.WithEphemeral
			},
			0, 1, false,
		)
	})

})
