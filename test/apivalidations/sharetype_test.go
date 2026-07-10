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
	sharetypeName = "sharetype"
	sharetypeID   = "265c9e4f-0f5a-46e4-9f3f-fb8de25ae120"
)

func sharetypeStub(namespace *corev1.Namespace) *orcv1alpha1.ShareType {
	obj := &orcv1alpha1.ShareType{}
	obj.Name = sharetypeName
	obj.Namespace = namespace.Name
	return obj
}

func testShareTypeResource() *applyconfigv1alpha1.ShareTypeResourceSpecApplyConfiguration {
	return applyconfigv1alpha1.ShareTypeResourceSpec()
}

func baseShareTypePatch(obj client.Object) *applyconfigv1alpha1.ShareTypeApplyConfiguration {
	return applyconfigv1alpha1.ShareType(obj.GetName(), obj.GetNamespace()).
		WithSpec(applyconfigv1alpha1.ShareTypeSpec().
			WithCloudCredentialsRef(testCredentials()))
}

func testShareTypeImport() *applyconfigv1alpha1.ShareTypeImportApplyConfiguration {
	return applyconfigv1alpha1.ShareTypeImport().WithID(sharetypeID)
}

var _ = Describe("ORC ShareType API validations", func() {
	var namespace *corev1.Namespace
	BeforeEach(func() {
		namespace = createNamespace()
	})

	runManagementPolicyTests(func() *corev1.Namespace { return namespace }, managementPolicyTestArgs[*applyconfigv1alpha1.ShareTypeApplyConfiguration]{
		createObject: func(ns *corev1.Namespace) client.Object { return sharetypeStub(ns) },
		basePatch: func(obj client.Object) *applyconfigv1alpha1.ShareTypeApplyConfiguration {
			return baseShareTypePatch(obj)
		},
		applyResource: func(p *applyconfigv1alpha1.ShareTypeApplyConfiguration) {
			p.Spec.WithResource(testShareTypeResource())
		},
		applyImport: func(p *applyconfigv1alpha1.ShareTypeApplyConfiguration) {
			p.Spec.WithImport(testShareTypeImport())
		},
		applyEmptyImport: func(p *applyconfigv1alpha1.ShareTypeApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.ShareTypeImport())
		},
		applyEmptyFilter: func(p *applyconfigv1alpha1.ShareTypeApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.ShareTypeImport().WithFilter(applyconfigv1alpha1.ShareTypeFilter()))
		},
		applyValidFilter: func(p *applyconfigv1alpha1.ShareTypeApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.ShareTypeImport().WithFilter(applyconfigv1alpha1.ShareTypeFilter().WithName("foo")))
		},
		applyManaged: func(p *applyconfigv1alpha1.ShareTypeApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged)
		},
		applyUnmanaged: func(p *applyconfigv1alpha1.ShareTypeApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged)
		},
		applyManagedOptions: func(p *applyconfigv1alpha1.ShareTypeApplyConfiguration) {
			p.Spec.WithManagedOptions(applyconfigv1alpha1.ManagedOptions().WithOnDelete(orcv1alpha1.OnDeleteDetach))
		},
		getManagementPolicy: func(obj client.Object) orcv1alpha1.ManagementPolicy {
			return obj.(*orcv1alpha1.ShareType).Spec.ManagementPolicy
		},
		getOnDelete: func(obj client.Object) orcv1alpha1.OnDelete {
			return obj.(*orcv1alpha1.ShareType).Spec.ManagedOptions.OnDelete
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
