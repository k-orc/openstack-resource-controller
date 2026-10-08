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
	recordsetName = "recordset"
	recordsetID   = "265c9e4f-0f5a-46e4-9f3f-fb8de25ae120"
)

func recordsetStub(namespace *corev1.Namespace) *orcv1alpha1.RecordSet {
	obj := &orcv1alpha1.RecordSet{}
	obj.Name = recordsetName
	obj.Namespace = namespace.Name
	return obj
}

func testRecordSetResource() *applyconfigv1alpha1.RecordSetResourceSpecApplyConfiguration {
	return applyconfigv1alpha1.RecordSetResourceSpec().
		WithDNSZoneRef("dnszone")
}

func baseRecordSetPatch(obj client.Object) *applyconfigv1alpha1.RecordSetApplyConfiguration {
	return applyconfigv1alpha1.RecordSet(obj.GetName(), obj.GetNamespace()).
		WithSpec(applyconfigv1alpha1.RecordSetSpec().
			WithCloudCredentialsRef(testCredentials()))
}

func testRecordSetImport() *applyconfigv1alpha1.RecordSetImportApplyConfiguration {
	return applyconfigv1alpha1.RecordSetImport().WithID(recordsetID)
}

var _ = Describe("ORC RecordSet API validations", func() {
	var namespace *corev1.Namespace
	BeforeEach(func() {
		namespace = createNamespace()
	})

	runManagementPolicyTests(func() *corev1.Namespace { return namespace }, managementPolicyTestArgs[*applyconfigv1alpha1.RecordSetApplyConfiguration]{
		createObject: func(ns *corev1.Namespace) client.Object { return recordsetStub(ns) },
		basePatch: func(obj client.Object) *applyconfigv1alpha1.RecordSetApplyConfiguration {
			return baseRecordSetPatch(obj)
		},
		applyResource: func(p *applyconfigv1alpha1.RecordSetApplyConfiguration) {
			p.Spec.WithResource(testRecordSetResource())
		},
		applyImport: func(p *applyconfigv1alpha1.RecordSetApplyConfiguration) {
			p.Spec.WithImport(testRecordSetImport())
		},
		applyEmptyImport: func(p *applyconfigv1alpha1.RecordSetApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.RecordSetImport())
		},
		applyEmptyFilter: func(p *applyconfigv1alpha1.RecordSetApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.RecordSetImport().WithFilter(applyconfigv1alpha1.RecordSetFilter()))
		},
		applyValidFilter: func(p *applyconfigv1alpha1.RecordSetApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.RecordSetImport().WithFilter(applyconfigv1alpha1.RecordSetFilter().WithName("foo")))
		},
		applyManaged: func(p *applyconfigv1alpha1.RecordSetApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged)
		},
		applyUnmanaged: func(p *applyconfigv1alpha1.RecordSetApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged)
		},
		applyManagedOptions: func(p *applyconfigv1alpha1.RecordSetApplyConfiguration) {
			p.Spec.WithManagedOptions(applyconfigv1alpha1.ManagedOptions().WithOnDelete(orcv1alpha1.OnDeleteDetach))
		},
		getManagementPolicy: func(obj client.Object) orcv1alpha1.ManagementPolicy {
			return obj.(*orcv1alpha1.RecordSet).Spec.ManagementPolicy
		},
		getOnDelete: func(obj client.Object) orcv1alpha1.OnDelete {
			return obj.(*orcv1alpha1.RecordSet).Spec.ManagedOptions.OnDelete
		},
	})

	It("should reject a recordset without required fields", func(ctx context.Context) {
		obj := recordsetStub(namespace)
		patch := baseRecordSetPatch(obj)
		patch.Spec.WithResource(applyconfigv1alpha1.RecordSetResourceSpec())
		Expect(applyObj(ctx, obj, patch)).NotTo(Succeed())
	})

	It("should have immutable dNSZoneRef", func(ctx context.Context) {
		obj := recordsetStub(namespace)
		patch := baseRecordSetPatch(obj)
		patch.Spec.WithResource(testRecordSetResource().
			WithDNSZoneRef("dnszone-a"))
		Expect(applyObj(ctx, obj, patch)).To(Succeed())

		patch.Spec.WithResource(testRecordSetResource().
			WithDNSZoneRef("dnszone-b"))
		Expect(applyObj(ctx, obj, patch)).To(MatchError(ContainSubstring("dNSZoneRef is immutable")))
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
