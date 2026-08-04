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
	shareName = "share"
	shareID   = "265c9e4f-0f5a-46e4-9f3f-fb8de25ae120"
)

func shareStub(namespace *corev1.Namespace) *orcv1alpha1.Share {
	obj := &orcv1alpha1.Share{}
	obj.Name = shareName
	obj.Namespace = namespace.Name
	return obj
}

func testShareResource() *applyconfigv1alpha1.ShareResourceSpecApplyConfiguration {
	return applyconfigv1alpha1.ShareResourceSpec().
		WithShareProto("NFS").
		WithSize(1)
}

func baseSharePatch(obj client.Object) *applyconfigv1alpha1.ShareApplyConfiguration {
	return applyconfigv1alpha1.Share(obj.GetName(), obj.GetNamespace()).
		WithSpec(applyconfigv1alpha1.ShareSpec().
			WithCloudCredentialsRef(testCredentials()))
}

func testShareImport() *applyconfigv1alpha1.ShareImportApplyConfiguration {
	return applyconfigv1alpha1.ShareImport().WithID(shareID)
}

var _ = Describe("ORC Share API validations", func() {
	var namespace *corev1.Namespace
	BeforeEach(func() {
		namespace = createNamespace()
	})

	runManagementPolicyTests(func() *corev1.Namespace { return namespace }, managementPolicyTestArgs[*applyconfigv1alpha1.ShareApplyConfiguration]{
		createObject: func(ns *corev1.Namespace) client.Object { return shareStub(ns) },
		basePatch: func(obj client.Object) *applyconfigv1alpha1.ShareApplyConfiguration {
			return baseSharePatch(obj)
		},
		applyResource: func(p *applyconfigv1alpha1.ShareApplyConfiguration) {
			p.Spec.WithResource(testShareResource())
		},
		applyImport: func(p *applyconfigv1alpha1.ShareApplyConfiguration) {
			p.Spec.WithImport(testShareImport())
		},
		applyEmptyImport: func(p *applyconfigv1alpha1.ShareApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.ShareImport())
		},
		applyEmptyFilter: func(p *applyconfigv1alpha1.ShareApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.ShareImport().WithFilter(applyconfigv1alpha1.ShareFilter()))
		},
		applyValidFilter: func(p *applyconfigv1alpha1.ShareApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.ShareImport().WithFilter(applyconfigv1alpha1.ShareFilter().WithName("foo")))
		},
		applyManaged: func(p *applyconfigv1alpha1.ShareApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged)
		},
		applyUnmanaged: func(p *applyconfigv1alpha1.ShareApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged)
		},
		applyManagedOptions: func(p *applyconfigv1alpha1.ShareApplyConfiguration) {
			p.Spec.WithManagedOptions(applyconfigv1alpha1.ManagedOptions().WithOnDelete(orcv1alpha1.OnDeleteDetach))
		},
		getManagementPolicy: func(obj client.Object) orcv1alpha1.ManagementPolicy {
			return obj.(*orcv1alpha1.Share).Spec.ManagementPolicy
		},
		getOnDelete: func(obj client.Object) orcv1alpha1.OnDelete {
			return obj.(*orcv1alpha1.Share).Spec.ManagedOptions.OnDelete
		},
	})

	It("should have immutable shareNetworkRef", func(ctx context.Context) {
		obj := shareStub(namespace)
		patch := baseSharePatch(obj)
		patch.Spec.WithResource(testShareResource().
			WithShareNetworkRef("sharenetwork-a"))
		Expect(applyObj(ctx, obj, patch)).To(Succeed())

		patch.Spec.WithResource(testShareResource().
			WithShareNetworkRef("sharenetwork-b"))
		Expect(applyObj(ctx, obj, patch)).To(MatchError(ContainSubstring("shareNetworkRef is immutable")))
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
