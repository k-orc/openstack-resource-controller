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
	serviceName = "service"
	serviceID   = "265c9e4f-0f5a-46e4-9f3f-fb8de25ae123"
)

func serviceStub(namespace *corev1.Namespace) *orcv1alpha1.Service {
	obj := &orcv1alpha1.Service{}
	obj.Name = serviceName
	obj.Namespace = namespace.Name
	return obj
}

func testServiceResource() *applyconfigv1alpha1.ServiceResourceSpecApplyConfiguration {
	return applyconfigv1alpha1.ServiceResourceSpec().WithType("compute")
}

func baseServicePatch(service client.Object) *applyconfigv1alpha1.ServiceApplyConfiguration {
	return applyconfigv1alpha1.Service(service.GetName(), service.GetNamespace()).
		WithSpec(applyconfigv1alpha1.ServiceSpec().
			WithCloudCredentialsRef(testCredentials()))
}

func testServiceImport() *applyconfigv1alpha1.ServiceImportApplyConfiguration {
	return applyconfigv1alpha1.ServiceImport().WithID(serviceID)
}

var _ = Describe("ORC Service API validations", func() {
	var namespace *corev1.Namespace
	BeforeEach(func() {
		namespace = createNamespace()
	})

	It("should allow to create a minimal service and managementPolicy should default to managed", func(ctx context.Context) {
		service := serviceStub(namespace)
		patch := baseServicePatch(service)
		patch.Spec.WithResource(testServiceResource())
		Expect(applyObj(ctx, service, patch)).To(Succeed())
		Expect(service.Spec.ManagementPolicy).To(Equal(orcv1alpha1.ManagementPolicyManaged))
	})

	It("should reject a service without required field type", func(ctx context.Context) {
		service := serviceStub(namespace)
		patch := baseServicePatch(service)
		patch.Spec.WithResource(applyconfigv1alpha1.ServiceResourceSpec())
		Expect(applyObj(ctx, service, patch)).To(MatchError(ContainSubstring("spec.resource.type")))
	})

	It("should require import for unmanaged", func(ctx context.Context) {
		service := serviceStub(namespace)
		patch := baseServicePatch(service)
		patch.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged)
		Expect(applyObj(ctx, service, patch)).To(MatchError(ContainSubstring("import must be specified when policy is unmanaged")))

		patch.Spec.WithImport(testServiceImport())
		Expect(applyObj(ctx, service, patch)).To(Succeed())
	})

	It("should not permit unmanaged with resource", func(ctx context.Context) {
		service := serviceStub(namespace)
		patch := baseServicePatch(service)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(testServiceImport()).
			WithResource(testServiceResource())
		Expect(applyObj(ctx, service, patch)).To(MatchError(ContainSubstring("resource may not be specified when policy is unmanaged")))
	})

	It("should not permit empty import", func(ctx context.Context) {
		service := serviceStub(namespace)
		patch := baseServicePatch(service)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(applyconfigv1alpha1.ServiceImport())
		Expect(applyObj(ctx, service, patch)).To(MatchError(ContainSubstring("spec.import in body should have at least 1 properties")))
	})

	It("should not permit empty import filter", func(ctx context.Context) {
		service := serviceStub(namespace)
		patch := baseServicePatch(service)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(applyconfigv1alpha1.ServiceImport().
				WithFilter(applyconfigv1alpha1.ServiceFilter()))
		Expect(applyObj(ctx, service, patch)).To(MatchError(ContainSubstring("spec.import.filter in body should have at least 1 properties")))
	})

	It("should permit import filter with name", func(ctx context.Context) {
		service := serviceStub(namespace)
		patch := baseServicePatch(service)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(applyconfigv1alpha1.ServiceImport().
				WithFilter(applyconfigv1alpha1.ServiceFilter().WithName("foo")))
		Expect(applyObj(ctx, service, patch)).To(Succeed())
	})

	It("should require resource for managed", func(ctx context.Context) {
		service := serviceStub(namespace)
		patch := baseServicePatch(service)
		patch.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged)
		Expect(applyObj(ctx, service, patch)).To(MatchError(ContainSubstring("resource must be specified when policy is managed")))

		patch.Spec.WithResource(testServiceResource())
		Expect(applyObj(ctx, service, patch)).To(Succeed())
	})

	It("should not permit managed with import", func(ctx context.Context) {
		service := serviceStub(namespace)
		patch := baseServicePatch(service)
		patch.Spec.
			WithImport(testServiceImport()).
			WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged).
			WithResource(testServiceResource())
		Expect(applyObj(ctx, service, patch)).To(MatchError(ContainSubstring("import may not be specified when policy is managed")))
	})

	It("should not permit managedOptions for unmanaged", func(ctx context.Context) {
		service := serviceStub(namespace)
		patch := baseServicePatch(service)
		patch.Spec.
			WithImport(testServiceImport()).
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithManagedOptions(applyconfigv1alpha1.ManagedOptions().
				WithOnDelete(orcv1alpha1.OnDeleteDetach))
		Expect(applyObj(ctx, service, patch)).To(MatchError(ContainSubstring("managedOptions may only be provided when policy is managed")))
	})

	It("should permit managedOptions for managed", func(ctx context.Context) {
		service := serviceStub(namespace)
		patch := baseServicePatch(service)
		patch.Spec.WithResource(testServiceResource()).
			WithManagedOptions(applyconfigv1alpha1.ManagedOptions().
				WithOnDelete(orcv1alpha1.OnDeleteDetach))
		Expect(applyObj(ctx, service, patch)).To(Succeed())
		Expect(service.Spec.ManagedOptions.OnDelete).To(Equal(orcv1alpha1.OnDelete("detach")))
	})
})
