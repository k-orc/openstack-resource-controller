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
	dnszoneshareName = "dnszoneshare"
	dnszoneshareID   = "265c9e4f-0f5a-46e4-9f3f-fb8de25ae120"
)

func dnszoneshareStub(namespace *corev1.Namespace) *orcv1alpha1.DNSZoneShare {
	obj := &orcv1alpha1.DNSZoneShare{}
	obj.Name = dnszoneshareName
	obj.Namespace = namespace.Name
	return obj
}

func testDNSZoneShareResource() *applyconfigv1alpha1.DNSZoneShareResourceSpecApplyConfiguration {
	return applyconfigv1alpha1.DNSZoneShareResourceSpec().
		WithZoneRef("dnszone").
		WithTargetProjectID("3fac9d0b0e0e4b0e9b0e000000000001")
}

func baseDNSZoneSharePatch(obj client.Object) *applyconfigv1alpha1.DNSZoneShareApplyConfiguration {
	return applyconfigv1alpha1.DNSZoneShare(obj.GetName(), obj.GetNamespace()).
		WithSpec(applyconfigv1alpha1.DNSZoneShareSpec().
			WithCloudCredentialsRef(testCredentials()))
}

func testDNSZoneShareImport() *applyconfigv1alpha1.DNSZoneShareImportApplyConfiguration {
	return applyconfigv1alpha1.DNSZoneShareImport().WithID(dnszoneshareID)
}

var _ = Describe("ORC DNSZoneShare API validations", func() {
	var namespace *corev1.Namespace
	BeforeEach(func() {
		namespace = createNamespace()
	})

	runManagementPolicyTests(func() *corev1.Namespace { return namespace }, managementPolicyTestArgs[*applyconfigv1alpha1.DNSZoneShareApplyConfiguration]{
		createObject: func(ns *corev1.Namespace) client.Object { return dnszoneshareStub(ns) },
		basePatch: func(obj client.Object) *applyconfigv1alpha1.DNSZoneShareApplyConfiguration {
			return baseDNSZoneSharePatch(obj)
		},
		applyResource: func(p *applyconfigv1alpha1.DNSZoneShareApplyConfiguration) {
			p.Spec.WithResource(testDNSZoneShareResource())
		},
		applyImport: func(p *applyconfigv1alpha1.DNSZoneShareApplyConfiguration) {
			p.Spec.WithImport(testDNSZoneShareImport())
		},
		applyEmptyImport: func(p *applyconfigv1alpha1.DNSZoneShareApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.DNSZoneShareImport())
		},
		applyEmptyFilter: func(p *applyconfigv1alpha1.DNSZoneShareApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.DNSZoneShareImport().WithFilter(applyconfigv1alpha1.DNSZoneShareFilter()))
		},
		applyValidFilter: func(p *applyconfigv1alpha1.DNSZoneShareApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.DNSZoneShareImport().WithFilter(applyconfigv1alpha1.DNSZoneShareFilter().WithTargetProjectID("foo")))
		},
		applyManaged: func(p *applyconfigv1alpha1.DNSZoneShareApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged)
		},
		applyUnmanaged: func(p *applyconfigv1alpha1.DNSZoneShareApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged)
		},
		applyManagedOptions: func(p *applyconfigv1alpha1.DNSZoneShareApplyConfiguration) {
			p.Spec.WithManagedOptions(applyconfigv1alpha1.ManagedOptions().WithOnDelete(orcv1alpha1.OnDeleteDetach))
		},
		getManagementPolicy: func(obj client.Object) orcv1alpha1.ManagementPolicy {
			return obj.(*orcv1alpha1.DNSZoneShare).Spec.ManagementPolicy
		},
		getOnDelete: func(obj client.Object) orcv1alpha1.OnDelete {
			return obj.(*orcv1alpha1.DNSZoneShare).Spec.ManagedOptions.OnDelete
		},
	})

	It("should reject a dnszoneshare without required fields", func(ctx context.Context) {
		obj := dnszoneshareStub(namespace)
		patch := baseDNSZoneSharePatch(obj)
		patch.Spec.WithResource(applyconfigv1alpha1.DNSZoneShareResourceSpec())
		Expect(applyObj(ctx, obj, patch)).NotTo(Succeed())
	})

	It("should have immutable zoneRef", func(ctx context.Context) {
		obj := dnszoneshareStub(namespace)
		patch := baseDNSZoneSharePatch(obj)
		patch.Spec.WithResource(testDNSZoneShareResource().
			WithZoneRef("dnszone-a"))
		Expect(applyObj(ctx, obj, patch)).To(Succeed())

		patch.Spec.WithResource(testDNSZoneShareResource().
			WithZoneRef("dnszone-b"))
		Expect(applyObj(ctx, obj, patch)).To(MatchError(ContainSubstring("zoneRef is immutable")))
	})

	It("should have immutable targetProjectID", func(ctx context.Context) {
		obj := dnszoneshareStub(namespace)
		patch := baseDNSZoneSharePatch(obj)
		patch.Spec.WithResource(testDNSZoneShareResource().
			WithTargetProjectID("3fac9d0b0e0e4b0e9b0e000000000001"))
		Expect(applyObj(ctx, obj, patch)).To(Succeed())

		// Unlike RBACPolicy's analogous field, Designate's zone-share API has no update
		// operation at all - confirmed via gophercloud (only List/Get/Share/Unshare) - so
		// targetProjectID is immutable here, not mutable like RBACPolicy's.
		patch.Spec.WithResource(testDNSZoneShareResource().
			WithTargetProjectID("4fac9d0b0e0e4b0e9b0e000000000002"))
		Expect(applyObj(ctx, obj, patch)).To(MatchError(ContainSubstring("targetProjectID is immutable")))
	})
})
