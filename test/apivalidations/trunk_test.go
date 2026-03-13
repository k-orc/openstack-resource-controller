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

	runManagementPolicyTests(func() *corev1.Namespace { return namespace }, managementPolicyTestArgs[*applyconfigv1alpha1.TrunkApplyConfiguration]{
		createObject: func(ns *corev1.Namespace) client.Object { return trunkStub(ns) },
		basePatch: func(obj client.Object) *applyconfigv1alpha1.TrunkApplyConfiguration {
			return baseTrunkPatch(obj)
		},
		applyResource: func(p *applyconfigv1alpha1.TrunkApplyConfiguration) {
			p.Spec.WithResource(testTrunkResource())
		},
		applyImport: func(p *applyconfigv1alpha1.TrunkApplyConfiguration) {
			p.Spec.WithImport(testTrunkImport())
		},
		applyEmptyImport: func(p *applyconfigv1alpha1.TrunkApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.TrunkImport())
		},
		applyEmptyFilter: func(p *applyconfigv1alpha1.TrunkApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.TrunkImport().WithFilter(applyconfigv1alpha1.TrunkFilter()))
		},
		applyValidFilter: func(p *applyconfigv1alpha1.TrunkApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.TrunkImport().WithFilter(applyconfigv1alpha1.TrunkFilter().WithName("foo")))
		},
		applyManaged: func(p *applyconfigv1alpha1.TrunkApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged)
		},
		applyUnmanaged: func(p *applyconfigv1alpha1.TrunkApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged)
		},
		applyManagedOptions: func(p *applyconfigv1alpha1.TrunkApplyConfiguration) {
			p.Spec.WithManagedOptions(applyconfigv1alpha1.ManagedOptions().WithOnDelete(orcv1alpha1.OnDeleteDetach))
		},
		getManagementPolicy: func(obj client.Object) orcv1alpha1.ManagementPolicy {
			return obj.(*orcv1alpha1.Trunk).Spec.ManagementPolicy
		},
		getOnDelete: func(obj client.Object) orcv1alpha1.OnDelete {
			return obj.(*orcv1alpha1.Trunk).Spec.ManagedOptions.OnDelete
		},
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
})
