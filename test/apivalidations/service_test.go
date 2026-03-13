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

	runManagementPolicyTests(func() *corev1.Namespace { return namespace }, managementPolicyTestArgs[*applyconfigv1alpha1.ServiceApplyConfiguration]{
		createObject:  func(ns *corev1.Namespace) client.Object { return serviceStub(ns) },
		basePatch:     func(obj client.Object) *applyconfigv1alpha1.ServiceApplyConfiguration { return baseServicePatch(obj) },
		applyResource: func(p *applyconfigv1alpha1.ServiceApplyConfiguration) { p.Spec.WithResource(testServiceResource()) },
		applyImport:   func(p *applyconfigv1alpha1.ServiceApplyConfiguration) { p.Spec.WithImport(testServiceImport()) },
		applyEmptyImport: func(p *applyconfigv1alpha1.ServiceApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.ServiceImport())
		},
		applyEmptyFilter: func(p *applyconfigv1alpha1.ServiceApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.ServiceImport().WithFilter(applyconfigv1alpha1.ServiceFilter()))
		},
		applyValidFilter: func(p *applyconfigv1alpha1.ServiceApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.ServiceImport().WithFilter(applyconfigv1alpha1.ServiceFilter().WithName("foo")))
		},
		applyManaged: func(p *applyconfigv1alpha1.ServiceApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged)
		},
		applyUnmanaged: func(p *applyconfigv1alpha1.ServiceApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged)
		},
		applyManagedOptions: func(p *applyconfigv1alpha1.ServiceApplyConfiguration) {
			p.Spec.WithManagedOptions(applyconfigv1alpha1.ManagedOptions().WithOnDelete(orcv1alpha1.OnDeleteDetach))
		},
		getManagementPolicy: func(obj client.Object) orcv1alpha1.ManagementPolicy {
			return obj.(*orcv1alpha1.Service).Spec.ManagementPolicy
		},
		getOnDelete: func(obj client.Object) orcv1alpha1.OnDelete {
			return obj.(*orcv1alpha1.Service).Spec.ManagedOptions.OnDelete
		},
	})

	It("should reject a service without required field type", func(ctx context.Context) {
		service := serviceStub(namespace)
		patch := baseServicePatch(service)
		patch.Spec.WithResource(applyconfigv1alpha1.ServiceResourceSpec())
		Expect(applyObj(ctx, service, patch)).To(MatchError(ContainSubstring("spec.resource.type")))
	})
})
