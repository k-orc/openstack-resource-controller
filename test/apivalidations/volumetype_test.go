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

	runManagementPolicyTests(func() *corev1.Namespace { return namespace }, managementPolicyTestArgs[*applyconfigv1alpha1.VolumeTypeApplyConfiguration]{
		createObject: func(ns *corev1.Namespace) client.Object { return volumeTypeStub(ns) },
		basePatch: func(obj client.Object) *applyconfigv1alpha1.VolumeTypeApplyConfiguration {
			return baseVolumeTypePatch(obj)
		},
		applyResource: func(p *applyconfigv1alpha1.VolumeTypeApplyConfiguration) {
			p.Spec.WithResource(testVolumeTypeResource())
		},
		applyImport: func(p *applyconfigv1alpha1.VolumeTypeApplyConfiguration) { p.Spec.WithImport(testVolumeTypeImport()) },
		applyEmptyImport: func(p *applyconfigv1alpha1.VolumeTypeApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.VolumeTypeImport())
		},
		applyEmptyFilter: func(p *applyconfigv1alpha1.VolumeTypeApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.VolumeTypeImport().WithFilter(applyconfigv1alpha1.VolumeTypeFilter()))
		},
		applyValidFilter: func(p *applyconfigv1alpha1.VolumeTypeApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.VolumeTypeImport().WithFilter(applyconfigv1alpha1.VolumeTypeFilter().WithName("foo")))
		},
		applyManaged: func(p *applyconfigv1alpha1.VolumeTypeApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged)
		},
		applyUnmanaged: func(p *applyconfigv1alpha1.VolumeTypeApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged)
		},
		applyManagedOptions: func(p *applyconfigv1alpha1.VolumeTypeApplyConfiguration) {
			p.Spec.WithManagedOptions(applyconfigv1alpha1.ManagedOptions().WithOnDelete(orcv1alpha1.OnDeleteDetach))
		},
		getManagementPolicy: func(obj client.Object) orcv1alpha1.ManagementPolicy {
			return obj.(*orcv1alpha1.VolumeType).Spec.ManagementPolicy
		},
		getOnDelete: func(obj client.Object) orcv1alpha1.OnDelete {
			return obj.(*orcv1alpha1.VolumeType).Spec.ManagedOptions.OnDelete
		},
	})

	It("should permit extraSpecs with required fields", func(ctx context.Context) {
		volumeType := volumeTypeStub(namespace)
		patch := baseVolumeTypePatch(volumeType)
		patch.Spec.WithResource(applyconfigv1alpha1.VolumeTypeResourceSpec().
			WithExtraSpecs(applyconfigv1alpha1.VolumeTypeExtraSpec().
				WithName("key").WithValue("value")))
		Expect(applyObj(ctx, volumeType, patch)).To(Succeed())
	})
})
