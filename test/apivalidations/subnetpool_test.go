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

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/api/v1alpha1"
	applyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/pkg/clients/applyconfiguration/api/v1alpha1"
)

const (
	subnetpoolName = "subnetpool"
	subnetpoolID   = "265c9e4f-0f5a-46e4-9f3f-fb8de25ae120"
)

func subnetpoolStub(namespace *corev1.Namespace) *orcv1alpha1.SubnetPool {
	obj := &orcv1alpha1.SubnetPool{}
	obj.Name = subnetpoolName
	obj.Namespace = namespace.Name
	return obj
}

func testSubnetPoolResource() *applyconfigv1alpha1.SubnetPoolResourceSpecApplyConfiguration {
	return applyconfigv1alpha1.SubnetPoolResourceSpec().
		WithPrefixes("10.0.0.0/16").
		WithMinPrefixLength(24).
		WithMaxPrefixLength(28)
}

func baseSubnetPoolPatch(obj client.Object) *applyconfigv1alpha1.SubnetPoolApplyConfiguration {
	return applyconfigv1alpha1.SubnetPool(obj.GetName(), obj.GetNamespace()).
		WithSpec(applyconfigv1alpha1.SubnetPoolSpec().
			WithCloudCredentialsRef(testCredentials()))
}

func testSubnetPoolImport() *applyconfigv1alpha1.SubnetPoolImportApplyConfiguration {
	return applyconfigv1alpha1.SubnetPoolImport().WithID(subnetpoolID)
}

var _ = Describe("ORC SubnetPool API validations", func() {
	var namespace *corev1.Namespace
	BeforeEach(func() {
		namespace = createNamespace()
	})

	runManagementPolicyTests(func() *corev1.Namespace { return namespace }, managementPolicyTestArgs[*applyconfigv1alpha1.SubnetPoolApplyConfiguration]{
		createObject: func(ns *corev1.Namespace) client.Object { return subnetpoolStub(ns) },
		basePatch: func(obj client.Object) *applyconfigv1alpha1.SubnetPoolApplyConfiguration {
			return baseSubnetPoolPatch(obj)
		},
		applyResource: func(p *applyconfigv1alpha1.SubnetPoolApplyConfiguration) {
			p.Spec.WithResource(testSubnetPoolResource())
		},
		applyImport: func(p *applyconfigv1alpha1.SubnetPoolApplyConfiguration) {
			p.Spec.WithImport(testSubnetPoolImport())
		},
		applyEmptyImport: func(p *applyconfigv1alpha1.SubnetPoolApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.SubnetPoolImport())
		},
		applyEmptyFilter: func(p *applyconfigv1alpha1.SubnetPoolApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.SubnetPoolImport().WithFilter(applyconfigv1alpha1.SubnetPoolFilter()))
		},
		applyValidFilter: func(p *applyconfigv1alpha1.SubnetPoolApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.SubnetPoolImport().WithFilter(applyconfigv1alpha1.SubnetPoolFilter().WithName("foo")))
		},
		applyManaged: func(p *applyconfigv1alpha1.SubnetPoolApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged)
		},
		applyUnmanaged: func(p *applyconfigv1alpha1.SubnetPoolApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged)
		},
		applyManagedOptions: func(p *applyconfigv1alpha1.SubnetPoolApplyConfiguration) {
			p.Spec.WithManagedOptions(applyconfigv1alpha1.ManagedOptions().WithOnDelete(orcv1alpha1.OnDeleteDetach))
		},
		getManagementPolicy: func(obj client.Object) orcv1alpha1.ManagementPolicy {
			return obj.(*orcv1alpha1.SubnetPool).Spec.ManagementPolicy
		},
		getOnDelete: func(obj client.Object) orcv1alpha1.OnDelete {
			return obj.(*orcv1alpha1.SubnetPool).Spec.ManagedOptions.OnDelete
		},
	})

	It("should have immutable projectRef", func(ctx context.Context) {
		obj := subnetpoolStub(namespace)
		patch := baseSubnetPoolPatch(obj)
		patch.Spec.WithResource(testSubnetPoolResource().
			WithProjectRef("project-a"))
		Expect(applyObj(ctx, obj, patch)).To(Succeed())

		patch.Spec.WithResource(testSubnetPoolResource().
			WithProjectRef("project-b"))
		Expect(applyObj(ctx, obj, patch)).To(MatchError(ContainSubstring("projectRef is immutable")))
	})

	It("should have immutable addressScopeRef", func(ctx context.Context) {
		obj := subnetpoolStub(namespace)
		patch := baseSubnetPoolPatch(obj)
		patch.Spec.WithResource(testSubnetPoolResource().
			WithAddressScopeRef("addressscope-a"))
		Expect(applyObj(ctx, obj, patch)).To(Succeed())

		patch.Spec.WithResource(testSubnetPoolResource().
			WithAddressScopeRef("addressscope-b"))
		Expect(applyObj(ctx, obj, patch)).To(MatchError(ContainSubstring("addressScopeRef is immutable")))
	})

	It("should have immutable prefixes", func(ctx context.Context) {
		obj := subnetpoolStub(namespace)
		patch := baseSubnetPoolPatch(obj)
		patch.Spec.WithResource(testSubnetPoolResource())
		Expect(applyObj(ctx, obj, patch)).To(Succeed())

		patch.Spec.WithResource(testSubnetPoolResource().WithPrefixes("10.1.0.0/16"))
		Expect(applyObj(ctx, obj, patch)).To(MatchError(ContainSubstring("prefixes is immutable")))
	})

	It("should have immutable minPrefixLength", func(ctx context.Context) {
		obj := subnetpoolStub(namespace)
		patch := baseSubnetPoolPatch(obj)
		patch.Spec.WithResource(testSubnetPoolResource())
		Expect(applyObj(ctx, obj, patch)).To(Succeed())

		patch.Spec.WithResource(testSubnetPoolResource().WithMinPrefixLength(25))
		Expect(applyObj(ctx, obj, patch)).To(MatchError(ContainSubstring("minPrefixLength is immutable")))
	})

	It("should have immutable maxPrefixLength", func(ctx context.Context) {
		obj := subnetpoolStub(namespace)
		patch := baseSubnetPoolPatch(obj)
		patch.Spec.WithResource(testSubnetPoolResource())
		Expect(applyObj(ctx, obj, patch)).To(Succeed())

		patch.Spec.WithResource(testSubnetPoolResource().WithMaxPrefixLength(29))
		Expect(applyObj(ctx, obj, patch)).To(MatchError(ContainSubstring("maxPrefixLength is immutable")))
	})

	It("should have immutable shared", func(ctx context.Context) {
		obj := subnetpoolStub(namespace)
		patch := baseSubnetPoolPatch(obj)
		patch.Spec.WithResource(testSubnetPoolResource().WithShared(true))
		Expect(applyObj(ctx, obj, patch)).To(Succeed())

		patch.Spec.WithResource(testSubnetPoolResource().WithShared(false))
		Expect(applyObj(ctx, obj, patch)).To(MatchError(ContainSubstring("shared is immutable")))
	})

	It("should have immutable defaultPrefixLength", func(ctx context.Context) {
		obj := subnetpoolStub(namespace)
		patch := baseSubnetPoolPatch(obj)
		patch.Spec.WithResource(testSubnetPoolResource().WithDefaultPrefixLength(24))
		Expect(applyObj(ctx, obj, patch)).To(Succeed())

		patch.Spec.WithResource(testSubnetPoolResource().WithDefaultPrefixLength(25))
		Expect(applyObj(ctx, obj, patch)).To(MatchError(ContainSubstring("defaultPrefixLength is immutable")))
	})

	It("should have immutable isDefault", func(ctx context.Context) {
		obj := subnetpoolStub(namespace)
		patch := baseSubnetPoolPatch(obj)
		patch.Spec.WithResource(testSubnetPoolResource().WithIsDefault(true))
		Expect(applyObj(ctx, obj, patch)).To(Succeed())

		patch.Spec.WithResource(testSubnetPoolResource().WithIsDefault(false))
		Expect(applyObj(ctx, obj, patch)).To(MatchError(ContainSubstring("isDefault is immutable")))
	})

	It("should reject minPrefixLength greater than maxPrefixLength", func(ctx context.Context) {
		obj := subnetpoolStub(namespace)
		patch := baseSubnetPoolPatch(obj)
		patch.Spec.WithResource(testSubnetPoolResource().
			WithMinPrefixLength(29).
			WithMaxPrefixLength(28))
		Err := applyObj(ctx, obj, patch)
		Expect(Err).To(MatchError(ContainSubstring("minPrefixLength must be less than or equal to maxPrefixLength")))
	})

	It("should permit minPrefixLength equal to maxPrefixLength", func(ctx context.Context) {
		obj := subnetpoolStub(namespace)
		patch := baseSubnetPoolPatch(obj)
		patch.Spec.WithResource(testSubnetPoolResource().
			WithMinPrefixLength(28).
			WithMaxPrefixLength(28))
		Expect(applyObj(ctx, obj, patch)).To(Succeed())
	})

	DescribeTable("should reject maxPrefixLength greater than 128",
		func(ctx context.Context, maxPrefixLength int32) {
			obj := subnetpoolStub(namespace)
			patch := baseSubnetPoolPatch(obj)
			patch.Spec.WithResource(testSubnetPoolResource().
				WithMinPrefixLength(24).
				WithMaxPrefixLength(maxPrefixLength))
			Expect(applyObj(ctx, obj, patch)).NotTo(Succeed())
		},
		Entry("129", int32(129)),
		Entry("255", int32(255)),
	)

	It("should permit maxPrefixLength equal to 128", func(ctx context.Context) {
		obj := subnetpoolStub(namespace)
		patch := baseSubnetPoolPatch(obj)
		patch.Spec.WithResource(testSubnetPoolResource().
			WithMinPrefixLength(1).
			WithMaxPrefixLength(128))
		Expect(applyObj(ctx, obj, patch)).To(Succeed())
	})

	It("should reject defaultPrefixLength outside of [minPrefixLength, maxPrefixLength]", func(ctx context.Context) {
		obj := subnetpoolStub(namespace)
		patch := baseSubnetPoolPatch(obj)
		patch.Spec.WithResource(testSubnetPoolResource().
			WithMinPrefixLength(24).
			WithMaxPrefixLength(28).
			WithDefaultPrefixLength(29))
		Expect(applyObj(ctx, obj, patch)).To(MatchError(ContainSubstring("defaultPrefixLength must be between minPrefixLength and maxPrefixLength")))

		patch.Spec.WithResource(testSubnetPoolResource().
			WithMinPrefixLength(24).
			WithMaxPrefixLength(28).
			WithDefaultPrefixLength(23))
		Expect(applyObj(ctx, obj, patch)).To(MatchError(ContainSubstring("defaultPrefixLength must be between minPrefixLength and maxPrefixLength")))
	})

	It("should permit defaultPrefixLength within [minPrefixLength, maxPrefixLength]", func(ctx context.Context) {
		obj := subnetpoolStub(namespace)
		patch := baseSubnetPoolPatch(obj)
		patch.Spec.WithResource(testSubnetPoolResource().
			WithMinPrefixLength(24).
			WithMaxPrefixLength(28).
			WithDefaultPrefixLength(26))
		Expect(applyObj(ctx, obj, patch)).To(Succeed())
	})

	It("should permit omitting defaultPrefixLength", func(ctx context.Context) {
		obj := subnetpoolStub(namespace)
		patch := baseSubnetPoolPatch(obj)
		patch.Spec.WithResource(testSubnetPoolResource().
			WithMinPrefixLength(24).
			WithMaxPrefixLength(28))
		Expect(applyObj(ctx, obj, patch)).To(Succeed())
	})
})
