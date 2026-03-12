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
	trunkName = "trunk"
	trunkID   = "265c9e4f-0f5a-46e4-9f3f-fb8de25ae133"
)

func trunkStub(namespace *corev1.Namespace) *orcv1alpha1.Trunk {
	obj := &orcv1alpha1.Trunk{}
	obj.Name = trunkName
	obj.Namespace = namespace.Name
	return obj
}

func testTrunkResource() *applyconfigv1alpha1.TrunkResourceSpecApplyConfiguration {
	return applyconfigv1alpha1.TrunkResourceSpec().
		WithPortRef("my-port")
}

func baseTrunkPatch(trunk client.Object) *applyconfigv1alpha1.TrunkApplyConfiguration {
	return applyconfigv1alpha1.Trunk(trunk.GetName(), trunk.GetNamespace()).
		WithSpec(applyconfigv1alpha1.TrunkSpec().
			WithCloudCredentialsRef(testCredentials()))
}

func testTrunkImport() *applyconfigv1alpha1.TrunkImportApplyConfiguration {
	return applyconfigv1alpha1.TrunkImport().WithID(trunkID)
}

var _ = Describe("ORC Trunk API validations", func() {
	var namespace *corev1.Namespace
	BeforeEach(func() {
		namespace = createNamespace()
	})

	It("should allow to create a minimal trunk and managementPolicy should default to managed", func(ctx context.Context) {
		trunk := trunkStub(namespace)
		patch := baseTrunkPatch(trunk)
		patch.Spec.WithResource(testTrunkResource())
		Expect(applyObj(ctx, trunk, patch)).To(Succeed())
		Expect(trunk.Spec.ManagementPolicy).To(Equal(orcv1alpha1.ManagementPolicyManaged))
	})

	It("should reject a trunk without required field portRef", func(ctx context.Context) {
		trunk := trunkStub(namespace)
		patch := baseTrunkPatch(trunk)
		patch.Spec.WithResource(applyconfigv1alpha1.TrunkResourceSpec())
		Expect(applyObj(ctx, trunk, patch)).To(MatchError(ContainSubstring("spec.resource.portRef")))
	})

	It("should have immutable portRef", func(ctx context.Context) {
		trunk := trunkStub(namespace)
		patch := baseTrunkPatch(trunk)
		patch.Spec.WithResource(applyconfigv1alpha1.TrunkResourceSpec().
			WithPortRef("port-a"))
		Expect(applyObj(ctx, trunk, patch)).To(Succeed())

		patch.Spec.WithResource(applyconfigv1alpha1.TrunkResourceSpec().
			WithPortRef("port-b"))
		Expect(applyObj(ctx, trunk, patch)).To(MatchError(ContainSubstring("portRef is immutable")))
	})

	It("should have immutable projectRef", func(ctx context.Context) {
		trunk := trunkStub(namespace)
		patch := baseTrunkPatch(trunk)
		patch.Spec.WithResource(applyconfigv1alpha1.TrunkResourceSpec().
			WithPortRef("my-port").
			WithProjectRef("project-a"))
		Expect(applyObj(ctx, trunk, patch)).To(Succeed())

		patch.Spec.WithResource(applyconfigv1alpha1.TrunkResourceSpec().
			WithPortRef("my-port").
			WithProjectRef("project-b"))
		Expect(applyObj(ctx, trunk, patch)).To(MatchError(ContainSubstring("projectRef is immutable")))
	})

	It("should reject invalid segmentationType enum value in subport", func(ctx context.Context) {
		trunk := trunkStub(namespace)
		patch := baseTrunkPatch(trunk)
		patch.Spec.WithResource(applyconfigv1alpha1.TrunkResourceSpec().
			WithPortRef("my-port").
			WithSubports(applyconfigv1alpha1.TrunkSubportSpec().
				WithPortRef("sub-port").
				WithSegmentationID(100).
				WithSegmentationType("invalid")))
		Expect(applyObj(ctx, trunk, patch)).NotTo(Succeed())
	})

	It("should permit valid segmentationType enum values in subport", func(ctx context.Context) {
		trunk := trunkStub(namespace)
		patch := baseTrunkPatch(trunk)
		patch.Spec.WithResource(applyconfigv1alpha1.TrunkResourceSpec().
			WithPortRef("my-port").
			WithSubports(applyconfigv1alpha1.TrunkSubportSpec().
				WithPortRef("sub-port").
				WithSegmentationID(100).
				WithSegmentationType("vlan")))
		Expect(applyObj(ctx, trunk, patch)).To(Succeed())
	})

	It("should reject segmentationID out of range in subport", func(ctx context.Context) {
		trunk := trunkStub(namespace)
		patch := baseTrunkPatch(trunk)

		// Below minimum
		patch.Spec.WithResource(applyconfigv1alpha1.TrunkResourceSpec().
			WithPortRef("my-port").
			WithSubports(applyconfigv1alpha1.TrunkSubportSpec().
				WithPortRef("sub-port").
				WithSegmentationID(0).
				WithSegmentationType("vlan")))
		Expect(applyObj(ctx, trunk, patch)).To(MatchError(ContainSubstring("spec.resource.subports[0].segmentationID")))

		// Above maximum
		patch.Spec.WithResource(applyconfigv1alpha1.TrunkResourceSpec().
			WithPortRef("my-port").
			WithSubports(applyconfigv1alpha1.TrunkSubportSpec().
				WithPortRef("sub-port").
				WithSegmentationID(4095).
				WithSegmentationType("vlan")))
		Expect(applyObj(ctx, trunk, patch)).To(MatchError(ContainSubstring("spec.resource.subports[0].segmentationID")))
	})

	It("should require import for unmanaged", func(ctx context.Context) {
		trunk := trunkStub(namespace)
		patch := baseTrunkPatch(trunk)
		patch.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged)
		Expect(applyObj(ctx, trunk, patch)).To(MatchError(ContainSubstring("import must be specified when policy is unmanaged")))

		patch.Spec.WithImport(testTrunkImport())
		Expect(applyObj(ctx, trunk, patch)).To(Succeed())
	})

	It("should not permit unmanaged with resource", func(ctx context.Context) {
		trunk := trunkStub(namespace)
		patch := baseTrunkPatch(trunk)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(testTrunkImport()).
			WithResource(testTrunkResource())
		Expect(applyObj(ctx, trunk, patch)).To(MatchError(ContainSubstring("resource may not be specified when policy is unmanaged")))
	})

	It("should not permit empty import", func(ctx context.Context) {
		trunk := trunkStub(namespace)
		patch := baseTrunkPatch(trunk)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(applyconfigv1alpha1.TrunkImport())
		Expect(applyObj(ctx, trunk, patch)).To(MatchError(ContainSubstring("spec.import in body should have at least 1 properties")))
	})

	It("should not permit empty import filter", func(ctx context.Context) {
		trunk := trunkStub(namespace)
		patch := baseTrunkPatch(trunk)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(applyconfigv1alpha1.TrunkImport().
				WithFilter(applyconfigv1alpha1.TrunkFilter()))
		Expect(applyObj(ctx, trunk, patch)).To(MatchError(ContainSubstring("spec.import.filter in body should have at least 1 properties")))
	})

	It("should permit import filter with name", func(ctx context.Context) {
		trunk := trunkStub(namespace)
		patch := baseTrunkPatch(trunk)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(applyconfigv1alpha1.TrunkImport().
				WithFilter(applyconfigv1alpha1.TrunkFilter().WithName("foo")))
		Expect(applyObj(ctx, trunk, patch)).To(Succeed())
	})

	It("should require resource for managed", func(ctx context.Context) {
		trunk := trunkStub(namespace)
		patch := baseTrunkPatch(trunk)
		patch.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged)
		Expect(applyObj(ctx, trunk, patch)).To(MatchError(ContainSubstring("resource must be specified when policy is managed")))

		patch.Spec.WithResource(testTrunkResource())
		Expect(applyObj(ctx, trunk, patch)).To(Succeed())
	})

	It("should not permit managed with import", func(ctx context.Context) {
		trunk := trunkStub(namespace)
		patch := baseTrunkPatch(trunk)
		patch.Spec.
			WithImport(testTrunkImport()).
			WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged).
			WithResource(testTrunkResource())
		Expect(applyObj(ctx, trunk, patch)).To(MatchError(ContainSubstring("import may not be specified when policy is managed")))
	})

	It("should not permit managedOptions for unmanaged", func(ctx context.Context) {
		trunk := trunkStub(namespace)
		patch := baseTrunkPatch(trunk)
		patch.Spec.
			WithImport(testTrunkImport()).
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithManagedOptions(applyconfigv1alpha1.ManagedOptions().
				WithOnDelete(orcv1alpha1.OnDeleteDetach))
		Expect(applyObj(ctx, trunk, patch)).To(MatchError(ContainSubstring("managedOptions may only be provided when policy is managed")))
	})

	It("should permit managedOptions for managed", func(ctx context.Context) {
		trunk := trunkStub(namespace)
		patch := baseTrunkPatch(trunk)
		patch.Spec.WithResource(testTrunkResource()).
			WithManagedOptions(applyconfigv1alpha1.ManagedOptions().
				WithOnDelete(orcv1alpha1.OnDeleteDetach))
		Expect(applyObj(ctx, trunk, patch)).To(Succeed())
		Expect(trunk.Spec.ManagedOptions.OnDelete).To(Equal(orcv1alpha1.OnDelete("detach")))
	})
})
