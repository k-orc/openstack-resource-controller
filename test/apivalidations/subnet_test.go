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
	subnetName  = "subnet"
	subnetID    = "265c9e4f-0f5a-46e4-9f3f-fb8de25ae19e"
	routerName  = "router"
	projectName = "project"
)

func subnetStub(namespace *corev1.Namespace) *orcv1alpha1.Subnet {
	obj := &orcv1alpha1.Subnet{}
	obj.Name = subnetName
	obj.Namespace = namespace.Name
	return obj
}

func baseSubnetPatch(subnet client.Object) *applyconfigv1alpha1.SubnetApplyConfiguration {
	return applyconfigv1alpha1.Subnet(subnet.GetName(), subnet.GetNamespace()).
		WithSpec(applyconfigv1alpha1.SubnetSpec().
			WithResource(applyconfigv1alpha1.SubnetResourceSpec().
				WithNetworkRef(networkName).
				WithIPVersion(4).
				WithCIDR("192.168.100.0/24")).
			WithCloudCredentialsRef(testCredentials()))
}

var _ = Describe("ORC Subnet API validations", func() {
	var namespace *corev1.Namespace
	BeforeEach(func() {
		namespace = createNamespace()
	})

	It("should allow to create a minimal subnet and managementPolicy should default to managed", func(ctx context.Context) {
		subnet := subnetStub(namespace)
		patch := baseSubnetPatch(subnet)
		Expect(applyObj(ctx, subnet, patch)).To(Succeed())
		Expect(subnet.Spec.ManagementPolicy).To(HaveValue(Equal(orcv1alpha1.ManagementPolicyManaged)))
	})
	It("should allow valid tags", func(ctx context.Context) {
		subnet := subnetStub(namespace)
		patch := baseSubnetPatch(subnet)
		patch.Spec.Resource.WithTags(orcv1alpha1.NeutronTag("foo"), orcv1alpha1.NeutronTag("bar"))
		Expect(applyObj(ctx, subnet, patch)).To(Succeed())
	})

	It("should allow valid tags", func(ctx context.Context) {
		subnet := subnetStub(namespace)
		patch := baseSubnetPatch(subnet)
		patch.Spec.Resource.WithTags(
			orcv1alpha1.NeutronTag("foo"),
			orcv1alpha1.NeutronTag("bar"),
			orcv1alpha1.NeutronTag("kozo"))
		Expect(applyObj(ctx, subnet, patch)).To(Succeed())
		patch.Spec.Resource.WithTags(
			orcv1alpha1.NeutronTag("foo"),
			orcv1alpha1.NeutronTag("bar"),
			orcv1alpha1.NeutronTag("foo"))
		Expect(applyObj(ctx, subnet, patch)).NotTo(Succeed())
	})

	It("should allow valid allocation pools", func(ctx context.Context) {
		subnet := subnetStub(namespace)
		patch := baseSubnetPatch(subnet)
		var startPool orcv1alpha1.IPvAny = "192.168.100.100"
		var endPool orcv1alpha1.IPvAny = "192.168.100.110"
		pools := applyconfigv1alpha1.AllocationPoolApplyConfiguration{Start: &startPool, End: &endPool}
		patch.Spec.Resource.WithAllocationPools(&pools)
		Expect(applyObj(ctx, subnet, patch)).To(Succeed())
	})

	It("should allow valid enable dhcp", func(ctx context.Context) {
		subnet := subnetStub(namespace)
		patch := baseSubnetPatch(subnet)
		patch.Spec.Resource.
			WithEnableDHCP(true)
		Expect(applyObj(ctx, subnet, patch)).To(Succeed())
		patch.Spec.Resource.
			WithEnableDHCP(false)
		Expect(applyObj(ctx, subnet, patch)).To(Succeed())
	})

	It("should allow valid dns servers", func(ctx context.Context) {
		subnet := subnetStub(namespace)
		patch := baseSubnetPatch(subnet)
		dnsServers := []orcv1alpha1.IPvAny{"1.1.1.1", "1.1.1.2"}
		patch.Spec.WithResource(applyconfigv1alpha1.SubnetResourceSpec().
			WithNetworkRef(networkName).
			WithIPVersion(4).
			WithCIDR("192.168.100.0/24").
			WithDNSNameservers(dnsServers...))
		Expect(applyObj(ctx, subnet, patch)).To(Succeed())
	})

	It("should allow valid host routes", func(ctx context.Context) {
		subnet := subnetStub(namespace)
		patch := baseSubnetPatch(subnet)

		routes := []*applyconfigv1alpha1.HostRouteApplyConfiguration{
			applyconfigv1alpha1.HostRoute().WithDestination("192.168.150.0/24").WithNextHop("192.168.100.1"),
			applyconfigv1alpha1.HostRoute().WithDestination("192.168.150.1/24").WithNextHop("192.168.100.2"),
		}

		patch.Spec.Resource.
			WithHostRoutes(routes...)
		Expect(applyObj(ctx, subnet, patch)).To(Succeed())
	})

	It("should allow valid router ref", func(ctx context.Context) {
		subnet := subnetStub(namespace)
		patch := baseSubnetPatch(subnet)

		patch.Spec.Resource.
			WithRouterRef(routerName)
		Expect(applyObj(ctx, subnet, patch)).To(Succeed())
	})

	It("should allow valid project ref", func(ctx context.Context) {
		subnet := subnetStub(namespace)
		patch := baseSubnetPatch(subnet)

		patch.Spec.WithResource(applyconfigv1alpha1.SubnetResourceSpec().WithNetworkRef(networkName).WithIPVersion(4).WithCIDR("192.168.100.0/24").
			WithProjectRef(projectName))
		Expect(applyObj(ctx, subnet, patch)).To(Succeed())
	})

	DescribeTable("should allow valid gateway", func(gateway *applyconfigv1alpha1.SubnetGatewayApplyConfiguration) {
		subnet := subnetStub(namespace)
		patch := baseSubnetPatch(subnet)
		patch.Spec.Resource.WithGateway(gateway)
		Expect(applyObj(ctx, subnet, patch)).To(Succeed())
	},
		Entry("with None type", applyconfigv1alpha1.SubnetGateway().WithType("None")),
		Entry("with automatic type", applyconfigv1alpha1.SubnetGateway().WithType("Automatic")),
		Entry("with IP", applyconfigv1alpha1.SubnetGateway().WithType("IP").WithIP("192.168.100.1")),
	)

	DescribeTable("should allow valid IPv6 option",
		func(addressMode string, raMode string) {
			subnet := subnetStub(namespace)
			patch := baseSubnetPatch(subnet)
			ipv6Options := applyconfigv1alpha1.IPv6Options().
				WithAddressMode(orcv1alpha1.IPv6AddressMode(addressMode)).
				WithRAMode(orcv1alpha1.IPv6RAMode(raMode))

			patch.Spec.WithResource(applyconfigv1alpha1.SubnetResourceSpec().
				WithNetworkRef(networkName).
				WithCIDR("192.168.100.0/24").
				WithIPVersion(6).WithIPv6(ipv6Options))
			Expect(applyObj(ctx, subnet, patch)).To(Succeed(), "create subnet")
		},
		Entry("when using SLAAC", orcv1alpha1.IPv6RAModeSLAAC, orcv1alpha1.IPv6RAModeSLAAC),
		Entry("when using DHCPv6 stateful", orcv1alpha1.IPv6AddressModeDHCPv6Stateful, orcv1alpha1.IPv6RAModeDHCPv6Stateful),
		Entry("when using DHCPv6 stateless", orcv1alpha1.IPv6AddressModeDHCPv6Stateless, orcv1alpha1.IPv6RAModeDHCPv6Stateless),
	)

	It("should permit valid import filter", func(ctx context.Context) {
		subnet := subnetStub(namespace)
		patch := applyconfigv1alpha1.Subnet(subnet.GetName(), subnet.GetNamespace()).
			WithSpec(applyconfigv1alpha1.SubnetSpec().
				WithCloudCredentialsRef(testCredentials()))

		patch.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(applyconfigv1alpha1.SubnetImport().
				WithFilter(applyconfigv1alpha1.SubnetFilter().
					WithName("foo").
					WithDescription("bar").
					WithTags("kozo").WithNotTags()))
		Expect(applyObj(ctx, subnet, patch)).To(Succeed())

		patch.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(applyconfigv1alpha1.SubnetImport().
				WithFilter(applyconfigv1alpha1.SubnetFilter().
					WithName("foo").
					WithDescription("bar").
					WithTags("kozo").WithTagsAny("anytag")))
		Expect(applyObj(ctx, subnet, patch)).To(Succeed())
	})

	It("should not permit empty import filter", func(ctx context.Context) {
		subnet := subnetStub(namespace)
		patch := baseSubnetPatch(subnet)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(applyconfigv1alpha1.SubnetImport().
				WithFilter(applyconfigv1alpha1.SubnetFilter()))
		Expect(applyObj(ctx, subnet, patch)).NotTo(Succeed())
	})

	It("should not permit invalid import filter", func(ctx context.Context) {
		network := subnetStub(namespace)
		patch := baseSubnetPatch(network)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(applyconfigv1alpha1.SubnetImport().
				WithFilter(applyconfigv1alpha1.SubnetFilter().WithName("foo,bar")))
		Expect(applyObj(ctx, network, patch)).NotTo(Succeed())
	})
})
