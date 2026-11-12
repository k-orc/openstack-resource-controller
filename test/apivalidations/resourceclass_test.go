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

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/api/v1alpha1"
	applyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/pkg/clients/applyconfiguration/api/v1alpha1"
)

const (
	resourceclassName = "resourceclass"
	resourceclassID   = "265c9e4f-0f5a-46e4-9f3f-fb8de25ae120"
)

func resourceclassStub(namespace *corev1.Namespace) *orcv1alpha1.ResourceClass {
	obj := &orcv1alpha1.ResourceClass{}
	obj.Name = resourceclassName
	obj.Namespace = namespace.Name
	return obj
}

func testResourceClassResource() *applyconfigv1alpha1.ResourceClassResourceSpecApplyConfiguration {
	return applyconfigv1alpha1.ResourceClassResourceSpec()
}

func baseResourceClassPatch(obj client.Object) *applyconfigv1alpha1.ResourceClassApplyConfiguration {
	return applyconfigv1alpha1.ResourceClass(obj.GetName(), obj.GetNamespace()).
		WithSpec(applyconfigv1alpha1.ResourceClassSpec().
			WithCloudCredentialsRef(testCredentials()))
}

func testResourceClassImport() *applyconfigv1alpha1.ResourceClassImportApplyConfiguration {
	return applyconfigv1alpha1.ResourceClassImport().WithID(resourceclassID)
}

var _ = Describe("ORC ResourceClass API validations", func() {
	var namespace *corev1.Namespace
	BeforeEach(func() {
		namespace = createNamespace()
	})

	runManagementPolicyTests(func() *corev1.Namespace { return namespace }, managementPolicyTestArgs[*applyconfigv1alpha1.ResourceClassApplyConfiguration]{
		createObject: func(ns *corev1.Namespace) client.Object { return resourceclassStub(ns) },
		basePatch: func(obj client.Object) *applyconfigv1alpha1.ResourceClassApplyConfiguration {
			return baseResourceClassPatch(obj)
		},
		applyResource: func(p *applyconfigv1alpha1.ResourceClassApplyConfiguration) {
			p.Spec.WithResource(testResourceClassResource())
		},
		applyImport: func(p *applyconfigv1alpha1.ResourceClassApplyConfiguration) {
			p.Spec.WithImport(testResourceClassImport())
		},
		applyEmptyImport: func(p *applyconfigv1alpha1.ResourceClassApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.ResourceClassImport())
		},
		applyEmptyFilter: func(p *applyconfigv1alpha1.ResourceClassApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.ResourceClassImport().WithFilter(applyconfigv1alpha1.ResourceClassFilter()))
		},
		applyValidFilter: func(p *applyconfigv1alpha1.ResourceClassApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.ResourceClassImport().WithFilter(applyconfigv1alpha1.ResourceClassFilter().WithName("foo")))
		},
		applyManaged: func(p *applyconfigv1alpha1.ResourceClassApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged)
		},
		applyUnmanaged: func(p *applyconfigv1alpha1.ResourceClassApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged)
		},
		applyManagedOptions: func(p *applyconfigv1alpha1.ResourceClassApplyConfiguration) {
			p.Spec.WithManagedOptions(applyconfigv1alpha1.ManagedOptions().WithOnDelete(orcv1alpha1.OnDeleteDetach))
		},
		getManagementPolicy: func(obj client.Object) orcv1alpha1.ManagementPolicy {
			return obj.(*orcv1alpha1.ResourceClass).Spec.ManagementPolicy
		},
		getOnDelete: func(obj client.Object) orcv1alpha1.OnDelete {
			return obj.(*orcv1alpha1.ResourceClass).Spec.ManagedOptions.OnDelete
		},
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
