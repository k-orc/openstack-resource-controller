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
	swiftcontainerName = "swiftcontainer"
	swiftcontainerID   = "app-assets-bucket"
)

func swiftcontainerStub(namespace *corev1.Namespace) *orcv1alpha1.SwiftContainer {
	obj := &orcv1alpha1.SwiftContainer{}
	obj.Name = swiftcontainerName
	obj.Namespace = namespace.Name
	return obj
}

func testSwiftContainerResource() *applyconfigv1alpha1.SwiftContainerResourceSpecApplyConfiguration {
	return applyconfigv1alpha1.SwiftContainerResourceSpec().
		WithName("app-assets-bucket").
		WithStoragePolicy("GoldPolicy").
		WithReadACL(".r:*").
		WithWriteACL("project_admin")
}

func baseSwiftContainerPatch(obj client.Object) *applyconfigv1alpha1.SwiftContainerApplyConfiguration {
	return applyconfigv1alpha1.SwiftContainer(obj.GetName(), obj.GetNamespace()).
		WithSpec(applyconfigv1alpha1.SwiftContainerSpec().
			WithCloudCredentialsRef(testCredentials()))
}

func testSwiftContainerImport() *applyconfigv1alpha1.SwiftContainerImportApplyConfiguration {
	return applyconfigv1alpha1.SwiftContainerImport().WithID(swiftcontainerID)
}

var _ = Describe("ORC SwiftContainer API validations", func() {
	var namespace *corev1.Namespace
	BeforeEach(func() {
		namespace = createNamespace()
	})

	runManagementPolicyTests(func() *corev1.Namespace { return namespace }, managementPolicyTestArgs[*applyconfigv1alpha1.SwiftContainerApplyConfiguration]{
		createObject: func(ns *corev1.Namespace) client.Object { return swiftcontainerStub(ns) },
		basePatch: func(obj client.Object) *applyconfigv1alpha1.SwiftContainerApplyConfiguration {
			return baseSwiftContainerPatch(obj)
		},
		applyResource: func(p *applyconfigv1alpha1.SwiftContainerApplyConfiguration) {
			p.Spec.WithResource(testSwiftContainerResource())
		},
		applyImport: func(p *applyconfigv1alpha1.SwiftContainerApplyConfiguration) {
			p.Spec.WithImport(testSwiftContainerImport())
		},
		applyEmptyImport: func(p *applyconfigv1alpha1.SwiftContainerApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.SwiftContainerImport())
		},
		applyEmptyFilter: func(p *applyconfigv1alpha1.SwiftContainerApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.SwiftContainerImport().WithFilter(applyconfigv1alpha1.SwiftContainerFilter()))
		},
		applyValidFilter: func(p *applyconfigv1alpha1.SwiftContainerApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.SwiftContainerImport().WithFilter(applyconfigv1alpha1.SwiftContainerFilter().WithName("foo")))
		},
		applyManaged: func(p *applyconfigv1alpha1.SwiftContainerApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged)
		},
		applyUnmanaged: func(p *applyconfigv1alpha1.SwiftContainerApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged)
		},
		applyManagedOptions: func(p *applyconfigv1alpha1.SwiftContainerApplyConfiguration) {
			p.Spec.WithManagedOptions(applyconfigv1alpha1.ManagedOptions().WithOnDelete(orcv1alpha1.OnDeleteDetach))
		},
		getManagementPolicy: func(obj client.Object) orcv1alpha1.ManagementPolicy {
			return obj.(*orcv1alpha1.SwiftContainer).Spec.ManagementPolicy
		},
		getOnDelete: func(obj client.Object) orcv1alpha1.OnDelete {
			return obj.(*orcv1alpha1.SwiftContainer).Spec.ManagedOptions.OnDelete
		},
	})

	It("should reject slashes in container names", func(ctx context.Context) {
		obj := swiftcontainerStub(namespace)
		patch := baseSwiftContainerPatch(obj)
		patch.Spec.WithResource(testSwiftContainerResource().WithName("invalid/name"))
		Expect(applyObj(ctx, obj, patch)).To(MatchError(ContainSubstring("Swift container names cannot contain '/'")))

		patch = baseSwiftContainerPatch(obj)
		patch.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(applyconfigv1alpha1.SwiftContainerImport().
				WithFilter(applyconfigv1alpha1.SwiftContainerFilter().WithName("invalid/name")))
		Expect(applyObj(ctx, obj, patch)).To(MatchError(ContainSubstring("Swift container names cannot contain '/'")))
	})

	It("should reject duplicate metadata names", func(ctx context.Context) {
		obj := swiftcontainerStub(namespace)
		patch := baseSwiftContainerPatch(obj)
		patch.Spec.WithResource(testSwiftContainerResource().WithMetadata(
			applyconfigv1alpha1.SwiftContainerMetadata().WithName("owner").WithValue("first"),
			applyconfigv1alpha1.SwiftContainerMetadata().WithName("owner").WithValue("second"),
		))
		Expect(applyObj(ctx, obj, patch)).To(MatchError(ContainSubstring("duplicate entries")))
	})

	It("should have immutable managementPolicy", func(ctx context.Context) {
		obj := swiftcontainerStub(namespace)
		patch := baseSwiftContainerPatch(obj)
		patch.Spec.WithResource(testSwiftContainerResource()).
			WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged)
		Expect(applyObj(ctx, obj, patch)).To(Succeed())

		patch.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged)
		Expect(applyObj(ctx, obj, patch)).To(MatchError(ContainSubstring("managementPolicy is immutable")))
	})

	It("should have immutable resource spec", func(ctx context.Context) {
		obj := swiftcontainerStub(namespace)
		patch := baseSwiftContainerPatch(obj)
		patch.Spec.WithResource(testSwiftContainerResource())
		Expect(applyObj(ctx, obj, patch)).To(Succeed())

		patch.Spec.WithResource(testSwiftContainerResource().WithReadACL(".r:-example.invalid"))
		Expect(applyObj(ctx, obj, patch)).To(MatchError(ContainSubstring("SwiftContainerResourceSpec is immutable")))
	})
})
