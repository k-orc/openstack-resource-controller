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
	volumeTypeName = "volumetype"
	volumeTypeID   = "265c9e4f-0f5a-46e4-9f3f-fb8de25ae126"
)

func volumeTypeStub(namespace *corev1.Namespace) *orcv1alpha1.VolumeType {
	obj := &orcv1alpha1.VolumeType{}
	obj.Name = volumeTypeName
	obj.Namespace = namespace.Name
	return obj
}

func testVolumeTypeResource() *applyconfigv1alpha1.VolumeTypeResourceSpecApplyConfiguration {
	return applyconfigv1alpha1.VolumeTypeResourceSpec()
}

func baseVolumeTypePatch(volumeType client.Object) *applyconfigv1alpha1.VolumeTypeApplyConfiguration {
	return applyconfigv1alpha1.VolumeType(volumeType.GetName(), volumeType.GetNamespace()).
		WithSpec(applyconfigv1alpha1.VolumeTypeSpec().
			WithCloudCredentialsRef(testCredentials()))
}

func testVolumeTypeImport() *applyconfigv1alpha1.VolumeTypeImportApplyConfiguration {
	return applyconfigv1alpha1.VolumeTypeImport().WithID(volumeTypeID)
}

var _ = Describe("ORC VolumeType API validations", func() {
	var namespace *corev1.Namespace
	BeforeEach(func() {
		namespace = createNamespace()
	})

	It("should allow to create a minimal volumetype and managementPolicy should default to managed", func(ctx context.Context) {
		volumeType := volumeTypeStub(namespace)
		patch := baseVolumeTypePatch(volumeType)
		patch.Spec.WithResource(testVolumeTypeResource())
		Expect(applyObj(ctx, volumeType, patch)).To(Succeed())
		Expect(volumeType.Spec.ManagementPolicy).To(Equal(orcv1alpha1.ManagementPolicyManaged))
	})

	It("should permit extraSpecs with required fields", func(ctx context.Context) {
		volumeType := volumeTypeStub(namespace)
		patch := baseVolumeTypePatch(volumeType)
		patch.Spec.WithResource(applyconfigv1alpha1.VolumeTypeResourceSpec().
			WithExtraSpecs(applyconfigv1alpha1.VolumeTypeExtraSpec().
				WithName("key").WithValue("value")))
		Expect(applyObj(ctx, volumeType, patch)).To(Succeed())
	})

	It("should require import for unmanaged", func(ctx context.Context) {
		volumeType := volumeTypeStub(namespace)
		patch := baseVolumeTypePatch(volumeType)
		patch.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged)
		Expect(applyObj(ctx, volumeType, patch)).To(MatchError(ContainSubstring("import must be specified when policy is unmanaged")))

		patch.Spec.WithImport(testVolumeTypeImport())
		Expect(applyObj(ctx, volumeType, patch)).To(Succeed())
	})

	It("should not permit unmanaged with resource", func(ctx context.Context) {
		volumeType := volumeTypeStub(namespace)
		patch := baseVolumeTypePatch(volumeType)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(testVolumeTypeImport()).
			WithResource(testVolumeTypeResource())
		Expect(applyObj(ctx, volumeType, patch)).To(MatchError(ContainSubstring("resource may not be specified when policy is unmanaged")))
	})

	It("should not permit empty import", func(ctx context.Context) {
		volumeType := volumeTypeStub(namespace)
		patch := baseVolumeTypePatch(volumeType)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(applyconfigv1alpha1.VolumeTypeImport())
		Expect(applyObj(ctx, volumeType, patch)).To(MatchError(ContainSubstring("spec.import in body should have at least 1 properties")))
	})

	It("should not permit empty import filter", func(ctx context.Context) {
		volumeType := volumeTypeStub(namespace)
		patch := baseVolumeTypePatch(volumeType)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(applyconfigv1alpha1.VolumeTypeImport().
				WithFilter(applyconfigv1alpha1.VolumeTypeFilter()))
		Expect(applyObj(ctx, volumeType, patch)).To(MatchError(ContainSubstring("spec.import.filter in body should have at least 1 properties")))
	})

	It("should permit import filter with name", func(ctx context.Context) {
		volumeType := volumeTypeStub(namespace)
		patch := baseVolumeTypePatch(volumeType)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(applyconfigv1alpha1.VolumeTypeImport().
				WithFilter(applyconfigv1alpha1.VolumeTypeFilter().WithName("foo")))
		Expect(applyObj(ctx, volumeType, patch)).To(Succeed())
	})

	It("should require resource for managed", func(ctx context.Context) {
		volumeType := volumeTypeStub(namespace)
		patch := baseVolumeTypePatch(volumeType)
		patch.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged)
		Expect(applyObj(ctx, volumeType, patch)).To(MatchError(ContainSubstring("resource must be specified when policy is managed")))

		patch.Spec.WithResource(testVolumeTypeResource())
		Expect(applyObj(ctx, volumeType, patch)).To(Succeed())
	})

	It("should not permit managed with import", func(ctx context.Context) {
		volumeType := volumeTypeStub(namespace)
		patch := baseVolumeTypePatch(volumeType)
		patch.Spec.
			WithImport(testVolumeTypeImport()).
			WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged).
			WithResource(testVolumeTypeResource())
		Expect(applyObj(ctx, volumeType, patch)).To(MatchError(ContainSubstring("import may not be specified when policy is managed")))
	})

	It("should not permit managedOptions for unmanaged", func(ctx context.Context) {
		volumeType := volumeTypeStub(namespace)
		patch := baseVolumeTypePatch(volumeType)
		patch.Spec.
			WithImport(testVolumeTypeImport()).
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithManagedOptions(applyconfigv1alpha1.ManagedOptions().
				WithOnDelete(orcv1alpha1.OnDeleteDetach))
		Expect(applyObj(ctx, volumeType, patch)).To(MatchError(ContainSubstring("managedOptions may only be provided when policy is managed")))
	})

	It("should permit managedOptions for managed", func(ctx context.Context) {
		volumeType := volumeTypeStub(namespace)
		patch := baseVolumeTypePatch(volumeType)
		patch.Spec.WithResource(testVolumeTypeResource()).
			WithManagedOptions(applyconfigv1alpha1.ManagedOptions().
				WithOnDelete(orcv1alpha1.OnDeleteDetach))
		Expect(applyObj(ctx, volumeType, patch)).To(Succeed())
		Expect(volumeType.Spec.ManagedOptions.OnDelete).To(Equal(orcv1alpha1.OnDelete("detach")))
	})
})
