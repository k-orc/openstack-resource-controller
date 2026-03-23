/*
Copyright 2024 The ORC Authors.

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
	networkName = "sg-foo"
	networkID   = "365c9e4f-0f5a-46e4-9f3f-fb8de25ae12f"
)

func networkStub(namespace *corev1.Namespace) *orcv1alpha1.Network {
	obj := &orcv1alpha1.Network{}
	obj.Name = networkName
	obj.Namespace = namespace.Name
	return obj
}

func testNetworkResource() *applyconfigv1alpha1.NetworkResourceSpecApplyConfiguration {
	return applyconfigv1alpha1.NetworkResourceSpec()
}

func testNetworkImport() *applyconfigv1alpha1.NetworkImportApplyConfiguration {
	return applyconfigv1alpha1.NetworkImport().WithID(networkID)
}

func baseNetworkPatch(network client.Object) *applyconfigv1alpha1.NetworkApplyConfiguration {
	return applyconfigv1alpha1.Network(network.GetName(), network.GetNamespace()).
		WithSpec(applyconfigv1alpha1.NetworkSpec().
			WithCloudCredentialsRef(testCredentials()))
}

var _ = Describe("ORC Network API validations", func() {
	var namespace *corev1.Namespace
	BeforeEach(func() {
		namespace = createNamespace()
	})

	runManagementPolicyTests(func() *corev1.Namespace { return namespace }, managementPolicyTestArgs[*applyconfigv1alpha1.NetworkApplyConfiguration]{
		createObject: func(ns *corev1.Namespace) client.Object { return networkStub(ns) },
		basePatch: func(obj client.Object) *applyconfigv1alpha1.NetworkApplyConfiguration {
			return baseNetworkPatch(obj)
		},
		applyResource: func(p *applyconfigv1alpha1.NetworkApplyConfiguration) {
			p.Spec.WithResource(testNetworkResource())
		},
		applyImport: func(p *applyconfigv1alpha1.NetworkApplyConfiguration) {
			p.Spec.WithImport(testNetworkImport())
		},
		applyEmptyImport: func(p *applyconfigv1alpha1.NetworkApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.NetworkImport())
		},
		applyEmptyFilter: func(p *applyconfigv1alpha1.NetworkApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.NetworkImport().WithFilter(applyconfigv1alpha1.NetworkFilter()))
		},
		applyValidFilter: func(p *applyconfigv1alpha1.NetworkApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.NetworkImport().WithFilter(applyconfigv1alpha1.NetworkFilter().WithName("foo")))
		},
		applyManaged: func(p *applyconfigv1alpha1.NetworkApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged)
		},
		applyUnmanaged: func(p *applyconfigv1alpha1.NetworkApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged)
		},
		applyManagedOptions: func(p *applyconfigv1alpha1.NetworkApplyConfiguration) {
			p.Spec.WithManagedOptions(applyconfigv1alpha1.ManagedOptions().WithOnDelete(orcv1alpha1.OnDeleteDetach))
		},
		getManagementPolicy: func(obj client.Object) orcv1alpha1.ManagementPolicy {
			return obj.(*orcv1alpha1.Network).Spec.ManagementPolicy
		},
		getOnDelete: func(obj client.Object) orcv1alpha1.OnDelete {
			return obj.(*orcv1alpha1.Network).Spec.ManagedOptions.OnDelete
		},
	})

	DescribeTable("should permit valid DNS domain",
		func(ctx context.Context, domain orcv1alpha1.DNSDomain) {
			network := networkStub(namespace)
			patch := baseNetworkPatch(network)
			patch.Spec.WithResource(applyconfigv1alpha1.NetworkResourceSpec().WithDNSDomain(domain))
			Expect(applyObj(ctx, network, patch)).To(Succeed(), "create network")
		},
		Entry("example", orcv1alpha1.DNSDomain("example")),
		Entry("example.com", orcv1alpha1.DNSDomain("example.com")),
		Entry("foo.bar.example.com.", orcv1alpha1.DNSDomain("foo.bar.example.com.")),
	)

	DescribeTable("should reject invalid DNS domain",
		func(ctx context.Context, domain orcv1alpha1.DNSDomain) {
			network := networkStub(namespace)
			patch := baseNetworkPatch(network)
			patch.Spec.WithResource(applyconfigv1alpha1.NetworkResourceSpec().WithDNSDomain(domain))
			Expect(applyObj(ctx, network, patch)).NotTo(Succeed(), "create network")
		},
		Entry("-example.com", orcv1alpha1.DNSDomain("-example.com")),
		Entry("empty", orcv1alpha1.DNSDomain("")),
		Entry("foo..bar", orcv1alpha1.DNSDomain("foo..bar")),
	)

	DescribeTable("should permit valid MTU",
		func(ctx context.Context, mtu orcv1alpha1.MTU) {
			network := networkStub(namespace)
			patch := baseNetworkPatch(network)
			patch.Spec.WithResource(applyconfigv1alpha1.NetworkResourceSpec().WithMTU(mtu))
			Expect(applyObj(ctx, network, patch)).To(Succeed(), "create network")
		},
		Entry("68", orcv1alpha1.MTU(68)),
		Entry("9000", orcv1alpha1.MTU(9000)),
	)

	DescribeTable("should reject invalid MTU",
		func(ctx context.Context, mtu orcv1alpha1.MTU) {
			network := networkStub(namespace)
			patch := baseNetworkPatch(network)
			patch.Spec.WithResource(applyconfigv1alpha1.NetworkResourceSpec().WithMTU(mtu))
			Expect(applyObj(ctx, network, patch)).NotTo(Succeed(), "create network")
		},
		Entry("9800", orcv1alpha1.MTU(9800)),
		Entry("30", orcv1alpha1.MTU(30)),
	)

	It("should allow valid tags", func(ctx context.Context) {
		network := networkStub(namespace)
		patch := baseNetworkPatch(network)
		patch.Spec.WithResource(applyconfigv1alpha1.NetworkResourceSpec().WithTags(
			orcv1alpha1.NeutronTag("foo"),
			orcv1alpha1.NeutronTag("bar")))
		Expect(applyObj(ctx, network, patch)).To(Succeed())
	})

	It("should allow valid tags", func(ctx context.Context) {
		network := networkStub(namespace)
		patch := baseNetworkPatch(network)
		patch.Spec.WithResource(applyconfigv1alpha1.NetworkResourceSpec().WithTags(
			orcv1alpha1.NeutronTag("foo"),
			orcv1alpha1.NeutronTag("bar"),
			orcv1alpha1.NeutronTag("kozo")))
		Expect(applyObj(ctx, network, patch)).To(Succeed())
		patch.Spec.WithResource(applyconfigv1alpha1.NetworkResourceSpec().WithTags(
			orcv1alpha1.NeutronTag("foo"),
			orcv1alpha1.NeutronTag("bar"),
			orcv1alpha1.NeutronTag("foo")))
		Expect(applyObj(ctx, network, patch)).NotTo(Succeed())
	})

	It("should permit valid import filter", func(ctx context.Context) {
		network := networkStub(namespace)
		patch := baseNetworkPatch(network)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(applyconfigv1alpha1.NetworkImport().
				WithFilter(applyconfigv1alpha1.NetworkFilter().
					WithName("foo").
					WithDescription("bar").
					WithTags("kozo").WithNotTags()))
		Expect(applyObj(ctx, network, patch)).To(Succeed())

		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(applyconfigv1alpha1.NetworkImport().
				WithFilter(applyconfigv1alpha1.NetworkFilter().
					WithName("foo").
					WithDescription("bar").
					WithTags("kozo").WithTagsAny("anytag")))
		Expect(applyObj(ctx, network, patch)).To(Succeed())
	})

	It("should not permit invalid import filter", func(ctx context.Context) {
		network := networkStub(namespace)
		patch := baseNetworkPatch(network)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(applyconfigv1alpha1.NetworkImport().
				WithFilter(applyconfigv1alpha1.NetworkFilter().WithName("foo,bar")))
		Expect(applyObj(ctx, network, patch)).NotTo(Succeed())
	})
})
