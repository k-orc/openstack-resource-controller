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
	. "github.com/onsi/ginkgo/v2"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	applyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/pkg/clients/applyconfiguration/api/v1alpha1"
)

const (
	domainName = "domain"
	domainID   = "265c9e4f-0f5a-46e4-9f3f-fb8de25ae120"
)

func domainStub(namespace *corev1.Namespace) *orcv1alpha1.Domain {
	obj := &orcv1alpha1.Domain{}
	obj.Name = domainName
	obj.Namespace = namespace.Name
	return obj
}

func testDomainResource() *applyconfigv1alpha1.DomainResourceSpecApplyConfiguration {
	return applyconfigv1alpha1.DomainResourceSpec()
}

func baseDomainPatch(domain client.Object) *applyconfigv1alpha1.DomainApplyConfiguration {
	return applyconfigv1alpha1.Domain(domain.GetName(), domain.GetNamespace()).
		WithSpec(applyconfigv1alpha1.DomainSpec().
			WithCloudCredentialsRef(testCredentials()))
}

func testDomainImport() *applyconfigv1alpha1.DomainImportApplyConfiguration {
	return applyconfigv1alpha1.DomainImport().WithID(domainID)
}

var _ = Describe("ORC Domain API validations", func() {
	var namespace *corev1.Namespace
	BeforeEach(func() {
		namespace = createNamespace()
	})

	runManagementPolicyTests(func() *corev1.Namespace { return namespace }, managementPolicyTestArgs[*applyconfigv1alpha1.DomainApplyConfiguration]{
		createObject:  func(ns *corev1.Namespace) client.Object { return domainStub(ns) },
		basePatch:     func(obj client.Object) *applyconfigv1alpha1.DomainApplyConfiguration { return baseDomainPatch(obj) },
		applyResource: func(p *applyconfigv1alpha1.DomainApplyConfiguration) { p.Spec.WithResource(testDomainResource()) },
		applyImport:   func(p *applyconfigv1alpha1.DomainApplyConfiguration) { p.Spec.WithImport(testDomainImport()) },
		applyEmptyImport: func(p *applyconfigv1alpha1.DomainApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.DomainImport())
		},
		applyEmptyFilter: func(p *applyconfigv1alpha1.DomainApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.DomainImport().WithFilter(applyconfigv1alpha1.DomainFilter()))
		},
		applyValidFilter: func(p *applyconfigv1alpha1.DomainApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.DomainImport().WithFilter(applyconfigv1alpha1.DomainFilter().WithName("foo")))
		},
		applyManaged: func(p *applyconfigv1alpha1.DomainApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged)
		},
		applyUnmanaged: func(p *applyconfigv1alpha1.DomainApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged)
		},
		applyManagedOptions: func(p *applyconfigv1alpha1.DomainApplyConfiguration) {
			p.Spec.WithManagedOptions(applyconfigv1alpha1.ManagedOptions().WithOnDelete(orcv1alpha1.OnDeleteDetach))
		},
		getManagementPolicy: func(obj client.Object) orcv1alpha1.ManagementPolicy {
			return obj.(*orcv1alpha1.Domain).Spec.ManagementPolicy
		},
		getOnDelete: func(obj client.Object) orcv1alpha1.OnDelete {
			return obj.(*orcv1alpha1.Domain).Spec.ManagedOptions.OnDelete
		},
	})
})
