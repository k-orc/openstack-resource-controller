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
	imagememberName = "imagemember"
	imagememberID   = "265c9e4f-0f5a-46e4-9f3f-fb8de25ae120"
)

func imagememberStub(namespace *corev1.Namespace) *orcv1alpha1.ImageMember {
	obj := &orcv1alpha1.ImageMember{}
	obj.Name = imagememberName
	obj.Namespace = namespace.Name
	return obj
}

func testImageMemberResource() *applyconfigv1alpha1.ImageMemberResourceSpecApplyConfiguration {
	return applyconfigv1alpha1.ImageMemberResourceSpec().
		WithImageRef("image")
}

func baseImageMemberPatch(obj client.Object) *applyconfigv1alpha1.ImageMemberApplyConfiguration {
	return applyconfigv1alpha1.ImageMember(obj.GetName(), obj.GetNamespace()).
		WithSpec(applyconfigv1alpha1.ImageMemberSpec().
			WithCloudCredentialsRef(testCredentials()))
}

func testImageMemberImport() *applyconfigv1alpha1.ImageMemberImportApplyConfiguration {
	return applyconfigv1alpha1.ImageMemberImport().WithID(imagememberID)
}

var _ = Describe("ORC ImageMember API validations", func() {
	var namespace *corev1.Namespace
	BeforeEach(func() {
		namespace = createNamespace()
	})

	runManagementPolicyTests(func() *corev1.Namespace { return namespace }, managementPolicyTestArgs[*applyconfigv1alpha1.ImageMemberApplyConfiguration]{
		createObject: func(ns *corev1.Namespace) client.Object { return imagememberStub(ns) },
		basePatch: func(obj client.Object) *applyconfigv1alpha1.ImageMemberApplyConfiguration {
			return baseImageMemberPatch(obj)
		},
		applyResource: func(p *applyconfigv1alpha1.ImageMemberApplyConfiguration) {
			p.Spec.WithResource(testImageMemberResource())
		},
		applyImport: func(p *applyconfigv1alpha1.ImageMemberApplyConfiguration) {
			p.Spec.WithImport(testImageMemberImport())
		},
		applyEmptyImport: func(p *applyconfigv1alpha1.ImageMemberApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.ImageMemberImport())
		},
		applyEmptyFilter: func(p *applyconfigv1alpha1.ImageMemberApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.ImageMemberImport().WithFilter(applyconfigv1alpha1.ImageMemberFilter()))
		},
		applyValidFilter: func(p *applyconfigv1alpha1.ImageMemberApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.ImageMemberImport().WithFilter(applyconfigv1alpha1.ImageMemberFilter().WithName("foo")))
		},
		applyManaged: func(p *applyconfigv1alpha1.ImageMemberApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged)
		},
		applyUnmanaged: func(p *applyconfigv1alpha1.ImageMemberApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged)
		},
		applyManagedOptions: func(p *applyconfigv1alpha1.ImageMemberApplyConfiguration) {
			p.Spec.WithManagedOptions(applyconfigv1alpha1.ManagedOptions().WithOnDelete(orcv1alpha1.OnDeleteDetach))
		},
		getManagementPolicy: func(obj client.Object) orcv1alpha1.ManagementPolicy {
			return obj.(*orcv1alpha1.ImageMember).Spec.ManagementPolicy
		},
		getOnDelete: func(obj client.Object) orcv1alpha1.OnDelete {
			return obj.(*orcv1alpha1.ImageMember).Spec.ManagedOptions.OnDelete
		},
	})

	It("should reject a imagemember without required fields", func(ctx context.Context) {
		obj := imagememberStub(namespace)
		patch := baseImageMemberPatch(obj)
		patch.Spec.WithResource(applyconfigv1alpha1.ImageMemberResourceSpec())
		Expect(applyObj(ctx, obj, patch)).NotTo(Succeed())
	})

	It("should have immutable imageRef", func(ctx context.Context) {
		obj := imagememberStub(namespace)
		patch := baseImageMemberPatch(obj)
		patch.Spec.WithResource(testImageMemberResource().
			WithImageRef("image-a"))
		Expect(applyObj(ctx, obj, patch)).To(Succeed())

		patch.Spec.WithResource(testImageMemberResource().
			WithImageRef("image-b"))
		Expect(applyObj(ctx, obj, patch)).To(MatchError(ContainSubstring("imageRef is immutable")))
	})

	// TODO(scaffolding): Add more resource-specific validation tests.
	// Some common things to test:
	// - Immutability of fields with `self == oldSelf` validation
	// - Enum validation (valid and invalid values)
	// - Numeric range validation (min/max bounds)
	// - Tag uniqueness (if the resource has tags with listType=set)
	// - Format validation (CIDR, UUID, etc.)
	// - Cross-field validation rules
})
