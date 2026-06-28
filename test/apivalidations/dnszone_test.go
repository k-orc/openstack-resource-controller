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
	dnszoneName = "dnszone"
	dnszoneID   = "265c9e4f-0f5a-46e4-9f3f-fb8de25ae120"
)

func dnszoneStub(namespace *corev1.Namespace) *orcv1alpha1.DNSZone {
	obj := &orcv1alpha1.DNSZone{}
	obj.Name = dnszoneName
	obj.Namespace = namespace.Name
	return obj
}

func testDNSZoneResource() *applyconfigv1alpha1.DNSZoneResourceSpecApplyConfiguration {
	return applyconfigv1alpha1.DNSZoneResourceSpec().WithEmail("admin@example.com")
}

func baseDNSZonePatch(obj client.Object) *applyconfigv1alpha1.DNSZoneApplyConfiguration {
	return applyconfigv1alpha1.DNSZone(obj.GetName(), obj.GetNamespace()).
		WithSpec(applyconfigv1alpha1.DNSZoneSpec().
			WithCloudCredentialsRef(testCredentials()))
}

func testDNSZoneImport() *applyconfigv1alpha1.DNSZoneImportApplyConfiguration {
	return applyconfigv1alpha1.DNSZoneImport().WithID(dnszoneID)
}

var _ = Describe("ORC DNSZone API validations", func() {
	var namespace *corev1.Namespace
	BeforeEach(func() {
		namespace = createNamespace()
	})

	runManagementPolicyTests(func() *corev1.Namespace { return namespace }, managementPolicyTestArgs[*applyconfigv1alpha1.DNSZoneApplyConfiguration]{
		createObject: func(ns *corev1.Namespace) client.Object { return dnszoneStub(ns) },
		basePatch: func(obj client.Object) *applyconfigv1alpha1.DNSZoneApplyConfiguration {
			return baseDNSZonePatch(obj)
		},
		applyResource: func(p *applyconfigv1alpha1.DNSZoneApplyConfiguration) {
			p.Spec.WithResource(testDNSZoneResource())
		},
		applyImport: func(p *applyconfigv1alpha1.DNSZoneApplyConfiguration) {
			p.Spec.WithImport(testDNSZoneImport())
		},
		applyEmptyImport: func(p *applyconfigv1alpha1.DNSZoneApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.DNSZoneImport())
		},
		applyEmptyFilter: func(p *applyconfigv1alpha1.DNSZoneApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.DNSZoneImport().WithFilter(applyconfigv1alpha1.DNSZoneFilter()))
		},
		applyValidFilter: func(p *applyconfigv1alpha1.DNSZoneApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.DNSZoneImport().WithFilter(applyconfigv1alpha1.DNSZoneFilter().WithName("foo.")))
		},
		applyManaged: func(p *applyconfigv1alpha1.DNSZoneApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged)
		},
		applyUnmanaged: func(p *applyconfigv1alpha1.DNSZoneApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged)
		},
		applyManagedOptions: func(p *applyconfigv1alpha1.DNSZoneApplyConfiguration) {
			p.Spec.WithManagedOptions(applyconfigv1alpha1.ManagedOptions().WithOnDelete(orcv1alpha1.OnDeleteDetach))
		},
		getManagementPolicy: func(obj client.Object) orcv1alpha1.ManagementPolicy {
			return obj.(*orcv1alpha1.DNSZone).Spec.ManagementPolicy
		},
		getOnDelete: func(obj client.Object) orcv1alpha1.OnDelete {
			return obj.(*orcv1alpha1.DNSZone).Spec.ManagedOptions.OnDelete
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
	It("should reject a dnszone without required fields (email)", func(ctx context.Context) {
		dnszone := dnszoneStub(namespace)
		patch := baseDNSZonePatch(dnszone)
		patch.Spec.WithResource(applyconfigv1alpha1.DNSZoneResourceSpec())
		Expect(applyObj(ctx, dnszone, patch)).To(MatchError(ContainSubstring("email is required for PRIMARY zones")))
	})

	It("should reject invalid type enum value", func(ctx context.Context) {
		dnszone := dnszoneStub(namespace)
		patch := baseDNSZonePatch(dnszone)
		patch.Spec.WithResource(applyconfigv1alpha1.DNSZoneResourceSpec().
			WithEmail("admin@example.com").
			WithType(orcv1alpha1.DNSZoneType("INVALID")))
		Expect(applyObj(ctx, dnszone, patch)).To(MatchError(ContainSubstring("Unsupported value")))
	})

	DescribeTable("should permit valid type enum values",
		func(ctx context.Context, ztype orcv1alpha1.DNSZoneType) {
			dnszone := dnszoneStub(namespace)
			patch := baseDNSZonePatch(dnszone)
			patch.Spec.WithResource(applyconfigv1alpha1.DNSZoneResourceSpec().
				WithEmail("admin@example.com").
				WithType(ztype))
			Expect(applyObj(ctx, dnszone, patch)).To(Succeed())
		},
		Entry("PRIMARY", orcv1alpha1.DNSZoneTypePrimary),
	)

	It("should reject SECONDARY type without masters", func(ctx context.Context) {
		dnszone := dnszoneStub(namespace)
		patch := baseDNSZonePatch(dnszone)
		patch.Spec.WithResource(applyconfigv1alpha1.DNSZoneResourceSpec().
			WithType(orcv1alpha1.DNSZoneTypeSecondary))
		Expect(applyObj(ctx, dnszone, patch)).To(MatchError(ContainSubstring("masters: required when type is SECONDARY")))
	})

	It("should reject SECONDARY type with email specified", func(ctx context.Context) {
		dnszone := dnszoneStub(namespace)
		patch := baseDNSZonePatch(dnszone)
		patch.Spec.WithResource(applyconfigv1alpha1.DNSZoneResourceSpec().
			WithType(orcv1alpha1.DNSZoneTypeSecondary).
			WithEmail("admin@example.com").
			WithMasters("1.2.3.4"))
		Expect(applyObj(ctx, dnszone, patch)).To(MatchError(ContainSubstring("email: must not be specified when type is SECONDARY")))
	})

	It("should permit SECONDARY type with masters", func(ctx context.Context) {
		dnszone := dnszoneStub(namespace)
		patch := baseDNSZonePatch(dnszone)
		patch.Spec.WithResource(applyconfigv1alpha1.DNSZoneResourceSpec().
			WithType(orcv1alpha1.DNSZoneTypeSecondary).
			WithMasters("1.2.3.4"))
		Expect(applyObj(ctx, dnszone, patch)).To(Succeed())
	})

	It("should reject PRIMARY type with masters specified", func(ctx context.Context) {
		dnszone := dnszoneStub(namespace)
		patch := baseDNSZonePatch(dnszone)
		patch.Spec.WithResource(applyconfigv1alpha1.DNSZoneResourceSpec().
			WithType(orcv1alpha1.DNSZoneTypePrimary).
			WithEmail("admin@example.com").
			WithMasters("1.2.3.4"))
		Expect(applyObj(ctx, dnszone, patch)).To(MatchError(ContainSubstring("masters: must not be specified when type is PRIMARY")))
	})

	It("should reject invalid email formats", func(ctx context.Context) {
		dnszone := dnszoneStub(namespace)
		patch := baseDNSZonePatch(dnszone)
		patch.Spec.WithResource(applyconfigv1alpha1.DNSZoneResourceSpec().
			WithEmail("invalid-email"))
		Expect(applyObj(ctx, dnszone, patch)).To(MatchError(ContainSubstring("spec.resource.email")))
	})

	It("should have immutable name", func(ctx context.Context) {
		dnszone := dnszoneStub(namespace)
		patch := baseDNSZonePatch(dnszone)
		patch.Spec.WithResource(applyconfigv1alpha1.DNSZoneResourceSpec().
			WithEmail("admin@example.com").
			WithName(orcv1alpha1.OpenStackName("example.com.")))
		Expect(applyObj(ctx, dnszone, patch)).To(Succeed())

		patch.Spec.WithResource(applyconfigv1alpha1.DNSZoneResourceSpec().
			WithEmail("admin@example.com").
			WithName(orcv1alpha1.OpenStackName("different.com.")))
		Expect(applyObj(ctx, dnszone, patch)).To(MatchError(ContainSubstring("name is immutable")))
	})

	It("should have immutable type", func(ctx context.Context) {
		dnszone := dnszoneStub(namespace)
		patch := baseDNSZonePatch(dnszone)
		patch.Spec.WithResource(applyconfigv1alpha1.DNSZoneResourceSpec().
			WithEmail("admin@example.com").
			WithType(orcv1alpha1.DNSZoneTypePrimary))
		Expect(applyObj(ctx, dnszone, patch)).To(Succeed())

		patch.Spec.WithResource(applyconfigv1alpha1.DNSZoneResourceSpec().
			WithType(orcv1alpha1.DNSZoneTypeSecondary).
			WithMasters("1.2.3.4"))
		Expect(applyObj(ctx, dnszone, patch)).To(MatchError(ContainSubstring("type is immutable")))
	})

	It("should accept a valid DNSZone manifest", func(ctx context.Context) {
		dnszone := dnszoneStub(namespace)
		patch := baseDNSZonePatch(dnszone)
		patch.Spec.WithResource(applyconfigv1alpha1.DNSZoneResourceSpec().
			WithName("example.com.").
			WithEmail("admin@example.com").
			WithTTL(3600).
			WithType(orcv1alpha1.DNSZoneTypePrimary))
		Expect(applyObj(ctx, dnszone, patch)).To(Succeed())
	})

	It("should reject invalid TTL values", func(ctx context.Context) {
		dnszone := dnszoneStub(namespace)
		patch := baseDNSZonePatch(dnszone)
		patch.Spec.WithResource(applyconfigv1alpha1.DNSZoneResourceSpec().
			WithEmail("admin@example.com").
			WithTTL(0))
		Expect(applyObj(ctx, dnszone, patch)).To(MatchError(ContainSubstring("should be greater than or equal to 1")))
	})

	It("should reject TTL values greater than 2147483647", func(ctx context.Context) {
		dnszone := dnszoneStub(namespace)
		patch := map[string]interface{}{
			"apiVersion": "openstack.k-orc.cloud/v1alpha1",
			"kind":       "DNSZone",
			"metadata": map[string]interface{}{
				"name":      dnszone.Name,
				"namespace": dnszone.Namespace,
			},
			"spec": map[string]interface{}{
				"cloudCredentialsRef": map[string]interface{}{
					"secretName": "openstack-credentials",
					"cloudName":  "openstack",
				},
				"resource": map[string]interface{}{
					"email": "admin@example.com",
					"ttl":   2147483648,
				},
			},
		}
		Expect(applyObj(ctx, dnszone, patch)).To(MatchError(ContainSubstring("should be less than or equal to 2147483647")))
	})

	It("should permit valid TTL values", func(ctx context.Context) {
		dnszone := dnszoneStub(namespace)
		patch := baseDNSZonePatch(dnszone)
		patch.Spec.WithResource(applyconfigv1alpha1.DNSZoneResourceSpec().
			WithEmail("admin@example.com").
			WithTTL(300))
		Expect(applyObj(ctx, dnszone, patch)).To(Succeed())
	})

	It("should reject Name if it does not end with a period", func(ctx context.Context) {
		dnszone := dnszoneStub(namespace)
		patch := baseDNSZonePatch(dnszone)
		patch.Spec.WithResource(applyconfigv1alpha1.DNSZoneResourceSpec().
			WithEmail("admin@example.com").
			WithName(orcv1alpha1.OpenStackName("example.com")))
		Expect(applyObj(ctx, dnszone, patch)).To(MatchError(ContainSubstring("name must end with a period")))
	})

	It("should permit Name ending with a period", func(ctx context.Context) {
		dnszone := dnszoneStub(namespace)
		patch := baseDNSZonePatch(dnszone)
		patch.Spec.WithResource(applyconfigv1alpha1.DNSZoneResourceSpec().
			WithEmail("admin@example.com").
			WithName(orcv1alpha1.OpenStackName("example.com.")))
		Expect(applyObj(ctx, dnszone, patch)).To(Succeed())
	})
})
