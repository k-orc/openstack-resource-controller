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
		WithName("www.example.com.").
		WithZoneRef("dnszone").
		WithType(orcv1alpha1.RecordSetType("A")).
		WithRecords("192.0.2.1")
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
			p.Spec.WithImport(applyconfigv1alpha1.RecordSetImport().WithFilter(applyconfigv1alpha1.RecordSetFilter().WithZoneRef("dnszone").WithName("foo.")))
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

	It("should have immutable zoneRef", func(ctx context.Context) {
		obj := recordsetStub(namespace)
		patch := baseRecordSetPatch(obj)
		patch.Spec.WithResource(testRecordSetResource().
			WithZoneRef("dnszone-a"))
		Expect(applyObj(ctx, obj, patch)).To(Succeed())

		patch.Spec.WithResource(testRecordSetResource().
			WithZoneRef("dnszone-b"))
		Expect(applyObj(ctx, obj, patch)).To(MatchError(ContainSubstring("zoneRef is immutable")))
	})

	It("should have immutable type", func(ctx context.Context) {
		obj := recordsetStub(namespace)
		patch := baseRecordSetPatch(obj)
		patch.Spec.WithResource(testRecordSetResource().
			WithType(orcv1alpha1.RecordSetType("A")))
		Expect(applyObj(ctx, obj, patch)).To(Succeed())

		patch.Spec.WithResource(testRecordSetResource().
			WithType(orcv1alpha1.RecordSetType("TXT")))
		Expect(applyObj(ctx, obj, patch)).To(MatchError(ContainSubstring("type is immutable")))
	})

	It("should reject an invalid type value", func(ctx context.Context) {
		obj := recordsetStub(namespace)
		patch := baseRecordSetPatch(obj)
		patch.Spec.WithResource(testRecordSetResource().
			WithType(orcv1alpha1.RecordSetType("NOT_A_REAL_TYPE")))
		Expect(applyObj(ctx, obj, patch)).NotTo(Succeed())
	})

	It("should allow mutating records and ttl", func(ctx context.Context) {
		obj := recordsetStub(namespace)
		patch := baseRecordSetPatch(obj)
		patch.Spec.WithResource(applyconfigv1alpha1.RecordSetResourceSpec().
			WithName("www.example.com.").
			WithZoneRef("dnszone").
			WithType(orcv1alpha1.RecordSetType("A")).
			WithRecords("192.0.2.1"))
		Expect(applyObj(ctx, obj, patch)).To(Succeed())

		patch.Spec.WithResource(applyconfigv1alpha1.RecordSetResourceSpec().
			WithName("www.example.com.").
			WithZoneRef("dnszone").
			WithType(orcv1alpha1.RecordSetType("A")).
			WithRecords("192.0.2.2", "192.0.2.3").
			WithTTL(300))
		Expect(applyObj(ctx, obj, patch)).To(Succeed())
	})
})
