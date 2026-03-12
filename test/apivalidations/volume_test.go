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

	It("should allow to create a minimal volume and managementPolicy should default to managed", func(ctx context.Context) {
		volume := volumeStub(namespace)
		patch := baseVolumePatch(volume)
		patch.Spec.WithResource(testVolumeResource())
		Expect(applyObj(ctx, volume, patch)).To(Succeed())
		Expect(volume.Spec.ManagementPolicy).To(Equal(orcv1alpha1.ManagementPolicyManaged))
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

	It("should require import for unmanaged", func(ctx context.Context) {
		volume := volumeStub(namespace)
		patch := baseVolumePatch(volume)
		patch.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged)
		Expect(applyObj(ctx, volume, patch)).To(MatchError(ContainSubstring("import must be specified when policy is unmanaged")))

		patch.Spec.WithImport(testVolumeImport())
		Expect(applyObj(ctx, volume, patch)).To(Succeed())
	})

	It("should not permit unmanaged with resource", func(ctx context.Context) {
		volume := volumeStub(namespace)
		patch := baseVolumePatch(volume)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(testVolumeImport()).
			WithResource(testVolumeResource())
		Expect(applyObj(ctx, volume, patch)).To(MatchError(ContainSubstring("resource may not be specified when policy is unmanaged")))
	})

	It("should not permit empty import", func(ctx context.Context) {
		volume := volumeStub(namespace)
		patch := baseVolumePatch(volume)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(applyconfigv1alpha1.VolumeImport())
		Expect(applyObj(ctx, volume, patch)).To(MatchError(ContainSubstring("spec.import in body should have at least 1 properties")))
	})

	It("should not permit empty import filter", func(ctx context.Context) {
		volume := volumeStub(namespace)
		patch := baseVolumePatch(volume)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(applyconfigv1alpha1.VolumeImport().
				WithFilter(applyconfigv1alpha1.VolumeFilter()))
		Expect(applyObj(ctx, volume, patch)).To(MatchError(ContainSubstring("spec.import.filter in body should have at least 1 properties")))
	})

	It("should permit import filter with name", func(ctx context.Context) {
		volume := volumeStub(namespace)
		patch := baseVolumePatch(volume)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(applyconfigv1alpha1.VolumeImport().
				WithFilter(applyconfigv1alpha1.VolumeFilter().WithName("foo")))
		Expect(applyObj(ctx, volume, patch)).To(Succeed())
	})

	It("should require resource for managed", func(ctx context.Context) {
		volume := volumeStub(namespace)
		patch := baseVolumePatch(volume)
		patch.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged)
		Expect(applyObj(ctx, volume, patch)).To(MatchError(ContainSubstring("resource must be specified when policy is managed")))

		patch.Spec.WithResource(testVolumeResource())
		Expect(applyObj(ctx, volume, patch)).To(Succeed())
	})

	It("should not permit managed with import", func(ctx context.Context) {
		volume := volumeStub(namespace)
		patch := baseVolumePatch(volume)
		patch.Spec.
			WithImport(testVolumeImport()).
			WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged).
			WithResource(testVolumeResource())
		Expect(applyObj(ctx, volume, patch)).To(MatchError(ContainSubstring("import may not be specified when policy is managed")))
	})

	It("should not permit managedOptions for unmanaged", func(ctx context.Context) {
		volume := volumeStub(namespace)
		patch := baseVolumePatch(volume)
		patch.Spec.
			WithImport(testVolumeImport()).
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithManagedOptions(applyconfigv1alpha1.ManagedOptions().
				WithOnDelete(orcv1alpha1.OnDeleteDetach))
		Expect(applyObj(ctx, volume, patch)).To(MatchError(ContainSubstring("managedOptions may only be provided when policy is managed")))
	})

	It("should permit managedOptions for managed", func(ctx context.Context) {
		volume := volumeStub(namespace)
		patch := baseVolumePatch(volume)
		patch.Spec.WithResource(testVolumeResource()).
			WithManagedOptions(applyconfigv1alpha1.ManagedOptions().
				WithOnDelete(orcv1alpha1.OnDeleteDetach))
		Expect(applyObj(ctx, volume, patch)).To(Succeed())
		Expect(volume.Spec.ManagedOptions.OnDelete).To(Equal(orcv1alpha1.OnDelete("detach")))
	})
})
