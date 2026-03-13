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
	volumeName = "volume"
	volumeID   = "265c9e4f-0f5a-46e4-9f3f-fb8de25ae131"
)

func volumeStub(namespace *corev1.Namespace) *orcv1alpha1.Volume {
	obj := &orcv1alpha1.Volume{}
	obj.Name = volumeName
	obj.Namespace = namespace.Name
	return obj
}

func testVolumeResource() *applyconfigv1alpha1.VolumeResourceSpecApplyConfiguration {
	return applyconfigv1alpha1.VolumeResourceSpec().WithSize(1)
}

func baseVolumePatch(volume client.Object) *applyconfigv1alpha1.VolumeApplyConfiguration {
	return applyconfigv1alpha1.Volume(volume.GetName(), volume.GetNamespace()).
		WithSpec(applyconfigv1alpha1.VolumeSpec().
			WithCloudCredentialsRef(testCredentials()))
}

func testVolumeImport() *applyconfigv1alpha1.VolumeImportApplyConfiguration {
	return applyconfigv1alpha1.VolumeImport().WithID(volumeID)
}

var _ = Describe("ORC Volume API validations", func() {
	var namespace *corev1.Namespace
	BeforeEach(func() {
		namespace = createNamespace()
	})

	runManagementPolicyTests(func() *corev1.Namespace { return namespace }, managementPolicyTestArgs[*applyconfigv1alpha1.VolumeApplyConfiguration]{
		createObject: func(ns *corev1.Namespace) client.Object { return volumeStub(ns) },
		basePatch: func(obj client.Object) *applyconfigv1alpha1.VolumeApplyConfiguration {
			return baseVolumePatch(obj)
		},
		applyResource: func(p *applyconfigv1alpha1.VolumeApplyConfiguration) {
			p.Spec.WithResource(testVolumeResource())
		},
		applyImport: func(p *applyconfigv1alpha1.VolumeApplyConfiguration) {
			p.Spec.WithImport(testVolumeImport())
		},
		applyEmptyImport: func(p *applyconfigv1alpha1.VolumeApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.VolumeImport())
		},
		applyEmptyFilter: func(p *applyconfigv1alpha1.VolumeApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.VolumeImport().WithFilter(applyconfigv1alpha1.VolumeFilter()))
		},
		applyValidFilter: func(p *applyconfigv1alpha1.VolumeApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.VolumeImport().WithFilter(applyconfigv1alpha1.VolumeFilter().WithName("foo")))
		},
		applyManaged: func(p *applyconfigv1alpha1.VolumeApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged)
		},
		applyUnmanaged: func(p *applyconfigv1alpha1.VolumeApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged)
		},
		applyManagedOptions: func(p *applyconfigv1alpha1.VolumeApplyConfiguration) {
			p.Spec.WithManagedOptions(applyconfigv1alpha1.ManagedOptions().WithOnDelete(orcv1alpha1.OnDeleteDetach))
		},
		getManagementPolicy: func(obj client.Object) orcv1alpha1.ManagementPolicy {
			return obj.(*orcv1alpha1.Volume).Spec.ManagementPolicy
		},
		getOnDelete: func(obj client.Object) orcv1alpha1.OnDelete {
			return obj.(*orcv1alpha1.Volume).Spec.ManagedOptions.OnDelete
		},
	})

	It("should reject a volume without required field size", func(ctx context.Context) {
		volume := volumeStub(namespace)
		patch := baseVolumePatch(volume)
		patch.Spec.WithResource(applyconfigv1alpha1.VolumeResourceSpec())
		Expect(applyObj(ctx, volume, patch)).To(MatchError(ContainSubstring("spec.resource.size")))
	})

	It("should reject size less than minimum", func(ctx context.Context) {
		volume := volumeStub(namespace)
		patch := baseVolumePatch(volume)
		patch.Spec.WithResource(applyconfigv1alpha1.VolumeResourceSpec().WithSize(0))
		Expect(applyObj(ctx, volume, patch)).To(MatchError(ContainSubstring("spec.resource.size in body should be greater than or equal to 1")))
	})

	It("should have immutable size", func(ctx context.Context) {
		volume := volumeStub(namespace)
		patch := baseVolumePatch(volume)
		patch.Spec.WithResource(applyconfigv1alpha1.VolumeResourceSpec().WithSize(1))
		Expect(applyObj(ctx, volume, patch)).To(Succeed())

		patch.Spec.WithResource(applyconfigv1alpha1.VolumeResourceSpec().WithSize(2))
		Expect(applyObj(ctx, volume, patch)).To(MatchError(ContainSubstring("size is immutable")))
	})

	It("should have immutable volumeTypeRef", func(ctx context.Context) {
		volume := volumeStub(namespace)
		patch := baseVolumePatch(volume)
		patch.Spec.WithResource(applyconfigv1alpha1.VolumeResourceSpec().
			WithSize(1).WithVolumeTypeRef("type-a"))
		Expect(applyObj(ctx, volume, patch)).To(Succeed())

		patch.Spec.WithResource(applyconfigv1alpha1.VolumeResourceSpec().
			WithSize(1).WithVolumeTypeRef("type-b"))
		Expect(applyObj(ctx, volume, patch)).To(MatchError(ContainSubstring("volumeTypeRef is immutable")))
	})

	It("should have immutable availabilityZone", func(ctx context.Context) {
		volume := volumeStub(namespace)
		patch := baseVolumePatch(volume)
		patch.Spec.WithResource(applyconfigv1alpha1.VolumeResourceSpec().
			WithSize(1).WithAvailabilityZone("az-a"))
		Expect(applyObj(ctx, volume, patch)).To(Succeed())

		patch.Spec.WithResource(applyconfigv1alpha1.VolumeResourceSpec().
			WithSize(1).WithAvailabilityZone("az-b"))
		Expect(applyObj(ctx, volume, patch)).To(MatchError(ContainSubstring("availabilityZone is immutable")))
	})

	It("should have immutable imageRef", func(ctx context.Context) {
		volume := volumeStub(namespace)
		patch := baseVolumePatch(volume)
		patch.Spec.WithResource(applyconfigv1alpha1.VolumeResourceSpec().
			WithSize(1).WithImageRef("image-a"))
		Expect(applyObj(ctx, volume, patch)).To(Succeed())

		patch.Spec.WithResource(applyconfigv1alpha1.VolumeResourceSpec().
			WithSize(1).WithImageRef("image-b"))
		Expect(applyObj(ctx, volume, patch)).To(MatchError(ContainSubstring("imageRef is immutable")))
	})
})
