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

	"github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	applyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/pkg/clients/applyconfiguration/api/v1alpha1"
	"k8s.io/utils/ptr"
)

const (
	securityGroupName = "sg-foo"
	securityGroupID   = "265c9e4f-0f5a-46e4-9f3f-fb8de25ae12f"
)

func securityGroupStub(namespace *corev1.Namespace) *orcv1alpha1.SecurityGroup {
	obj := &orcv1alpha1.SecurityGroup{}
	obj.Name = securityGroupName
	obj.Namespace = namespace.Name
	return obj
}

func testSecurityGroupResource() *applyconfigv1alpha1.SecurityGroupResourceSpecApplyConfiguration {
	return applyconfigv1alpha1.SecurityGroupResourceSpec().WithRules()
}

func baseSecurityGroupPatch(securityGroup client.Object) *applyconfigv1alpha1.SecurityGroupApplyConfiguration {
	return applyconfigv1alpha1.SecurityGroup(securityGroup.GetName(), securityGroup.GetNamespace()).
		WithSpec(applyconfigv1alpha1.SecurityGroupSpec().
			WithCloudCredentialsRef(testCredentials()))
}

func baseSGRulePatchSpec() *applyconfigv1alpha1.SecurityGroupRuleApplyConfiguration {
	return applyconfigv1alpha1.SecurityGroupRule().WithEthertype(orcv1alpha1.EthertypeIPv4).WithProtocol(orcv1alpha1.ProtocolTCP)
}

func testSecurityGroupImport() *applyconfigv1alpha1.SecurityGroupImportApplyConfiguration {
	return applyconfigv1alpha1.SecurityGroupImport().WithID(securityGroupID)
}

var _ = Describe("ORC SecurityGroup API validations", func() {
	var namespace *corev1.Namespace
	BeforeEach(func() {
		namespace = createNamespace()
	})

	It("should allow to create a minimal security group and managementPolicy should default to managed", func(ctx context.Context) {
		securityGroup := securityGroupStub(namespace)
		patch := baseSecurityGroupPatch(securityGroup)
		patch.Spec.WithResource(applyconfigv1alpha1.SecurityGroupResourceSpec())
		Expect(applyObj(ctx, securityGroup, patch)).To(Succeed())
		Expect(securityGroup.Spec.ManagementPolicy).To(Equal(orcv1alpha1.ManagementPolicyManaged))
	})

	It("should require import for unmanaged", func(ctx context.Context) {
		securityGroup := securityGroupStub(namespace)
		patch := baseSecurityGroupPatch(securityGroup)
		patch.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged)
		Expect(applyObj(ctx, securityGroup, patch)).NotTo(Succeed())

		patch.Spec.WithImport(testSecurityGroupImport())
		Expect(applyObj(ctx, securityGroup, patch)).To(Succeed())
	})

	It("should not permit unmanaged with resource", func(ctx context.Context) {
		securityGroup := securityGroupStub(namespace)
		patch := baseSecurityGroupPatch(securityGroup)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(testSecurityGroupImport()).
			WithResource(testSecurityGroupResource())
		Expect(applyObj(ctx, securityGroup, patch)).NotTo(Succeed())
	})

	It("should not permit empty import", func(ctx context.Context) {
		securityGroup := securityGroupStub(namespace)
		patch := baseSecurityGroupPatch(securityGroup)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(applyconfigv1alpha1.SecurityGroupImport())
		Expect(applyObj(ctx, securityGroup, patch)).NotTo(Succeed())
	})

	It("should not permit empty import filter", func(ctx context.Context) {
		securityGroup := securityGroupStub(namespace)
		patch := baseSecurityGroupPatch(securityGroup)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(applyconfigv1alpha1.SecurityGroupImport().
				WithFilter(applyconfigv1alpha1.SecurityGroupFilter()))
		Expect(applyObj(ctx, securityGroup, patch)).NotTo(Succeed())
	})

	It("should permit import filter with name", func(ctx context.Context) {
		securityGroup := securityGroupStub(namespace)
		patch := baseSecurityGroupPatch(securityGroup)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(applyconfigv1alpha1.SecurityGroupImport().
				WithFilter(applyconfigv1alpha1.SecurityGroupFilter().WithName("foo")))
		Expect(applyObj(ctx, securityGroup, patch)).To(Succeed())
	})

	It("should require resource for managed", func(ctx context.Context) {
		securityGroup := securityGroupStub(namespace)
		patch := baseSecurityGroupPatch(securityGroup)
		patch.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged)
		Expect(applyObj(ctx, securityGroup, patch)).NotTo(Succeed())

		patch.Spec.WithResource(testSecurityGroupResource())
		Expect(applyObj(ctx, securityGroup, patch)).To(Succeed())
	})

	It("should not permit managed with import", func(ctx context.Context) {
		securityGroup := securityGroupStub(namespace)
		patch := baseSecurityGroupPatch(securityGroup)
		patch.Spec.
			WithImport(testSecurityGroupImport()).
			WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged).
			WithResource(testSecurityGroupResource())
		Expect(applyObj(ctx, securityGroup, patch)).NotTo(Succeed())
	})

	It("should not permit managedOptions for unmanaged", func(ctx context.Context) {
		securityGroup := securityGroupStub(namespace)
		patch := baseSecurityGroupPatch(securityGroup)
		patch.Spec.
			WithImport(testSecurityGroupImport()).
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithManagedOptions(applyconfigv1alpha1.ManagedOptions().
				WithOnDelete(orcv1alpha1.OnDeleteDetach))
		Expect(applyObj(ctx, securityGroup, patch)).NotTo(Succeed())
	})

	It("should permit managedOptions for managed", func(ctx context.Context) {
		securityGroup := securityGroupStub(namespace)
		patch := baseSecurityGroupPatch(securityGroup)
		patch.Spec.WithResource(applyconfigv1alpha1.SecurityGroupResourceSpec())
		patch.Spec.
			WithManagedOptions(applyconfigv1alpha1.ManagedOptions().
				WithOnDelete(orcv1alpha1.OnDeleteDetach)).WithResource(
			applyconfigv1alpha1.SecurityGroupResourceSpec())
		Expect(applyObj(ctx, securityGroup, patch)).To(Succeed())
		Expect(securityGroup.Spec.ManagedOptions.OnDelete).To(Equal(orcv1alpha1.OnDelete("detach")))
	})

	It("should not permit invalid direction", func(ctx context.Context) {
		securityGroup := securityGroupStub(namespace)
		patch := baseSecurityGroupPatch(securityGroup)
		patch.Spec.WithResource(applyconfigv1alpha1.SecurityGroupResourceSpec().WithRules(baseSGRulePatchSpec().WithDirection("foo")))
		Expect(applyObj(ctx, securityGroup, patch)).NotTo(Succeed(), "create security group")
	})

	It("should permit valid direction", func(ctx context.Context) {
		securityGroup := securityGroupStub(namespace)
		patch := baseSecurityGroupPatch(securityGroup)
		patch.Spec.WithResource(applyconfigv1alpha1.SecurityGroupResourceSpec().WithRules(baseSGRulePatchSpec().WithDirection("ingress")))
		Expect(applyObj(ctx, securityGroup, patch)).To(Succeed(), "create security group")
		patch.Spec.WithResource(applyconfigv1alpha1.SecurityGroupResourceSpec().WithRules(baseSGRulePatchSpec().WithDirection("egress")))
		Expect(applyObj(ctx, securityGroup, patch)).To(Succeed(), "create security group")
	})

	It("should not permit invalid ethertype", func(ctx context.Context) {
		securityGroup := securityGroupStub(namespace)
		patch := baseSecurityGroupPatch(securityGroup)
		patch.Spec.WithResource(applyconfigv1alpha1.SecurityGroupResourceSpec().WithRules(baseSGRulePatchSpec().WithEthertype("foo")))
		Expect(applyObj(ctx, securityGroup, patch)).NotTo(Succeed(), "create security group")
	})

	It("should not permit no ethertype", func(ctx context.Context) {
		securityGroup := securityGroupStub(namespace)
		patch := baseSecurityGroupPatch(securityGroup)
		sgSpecRule := applyconfigv1alpha1.SecurityGroupRule().WithProtocol(orcv1alpha1.ProtocolTCP)
		patch.Spec.WithResource(applyconfigv1alpha1.SecurityGroupResourceSpec().WithRules(sgSpecRule))
		Expect(applyObj(ctx, securityGroup, patch)).NotTo(Succeed(), "create security group")
	})

	It("should permit valid ethertype", func(ctx context.Context) {
		securityGroup := securityGroupStub(namespace)
		patch := baseSecurityGroupPatch(securityGroup)
		patch.Spec.WithResource(applyconfigv1alpha1.SecurityGroupResourceSpec().WithRules(baseSGRulePatchSpec().WithEthertype(orcv1alpha1.EthertypeIPv6)))
		Expect(applyObj(ctx, securityGroup, patch)).To(Succeed(), "create security group")
		patch.Spec.WithResource(applyconfigv1alpha1.SecurityGroupResourceSpec().WithRules(baseSGRulePatchSpec().WithEthertype(orcv1alpha1.EthertypeIPv6)))
		Expect(applyObj(ctx, securityGroup, patch)).To(Succeed(), "create security group")
	})

	DescribeTable("should permit valid protocol",
		func(ctx context.Context, protocol orcv1alpha1.Protocol, ethertype orcv1alpha1.Ethertype) {
			securityGroup := securityGroupStub(namespace)
			patch := baseSecurityGroupPatch(securityGroup)
			sgRuleSpecPatch := applyconfigv1alpha1.SecurityGroupRule().WithEthertype(ethertype).WithProtocol(protocol)
			patch.Spec.WithResource(applyconfigv1alpha1.SecurityGroupResourceSpec().WithRules(sgRuleSpecPatch))
			Expect(applyObj(ctx, securityGroup, patch)).To(Succeed(), "create security group")
		},
		Entry(string(orcv1alpha1.ProtocolAH), orcv1alpha1.ProtocolAH, orcv1alpha1.EthertypeIPv4),
		Entry(string(orcv1alpha1.ProtocolDCCP), orcv1alpha1.ProtocolDCCP, orcv1alpha1.EthertypeIPv4),
		Entry(string(orcv1alpha1.ProtocolEGP), orcv1alpha1.ProtocolEGP, orcv1alpha1.EthertypeIPv4),
		Entry(string(orcv1alpha1.ProtocolESP), orcv1alpha1.ProtocolESP, orcv1alpha1.EthertypeIPv4),
		Entry(string(orcv1alpha1.ProtocolGRE), orcv1alpha1.ProtocolGRE, orcv1alpha1.EthertypeIPv4),
		Entry(string(orcv1alpha1.ProtocolICMP), orcv1alpha1.ProtocolICMP, orcv1alpha1.EthertypeIPv4),
		Entry(string(orcv1alpha1.ProtocolICMPV6), orcv1alpha1.ProtocolICMPV6, orcv1alpha1.EthertypeIPv4),
		Entry(string(orcv1alpha1.ProtocolIGMP), orcv1alpha1.ProtocolIGMP, orcv1alpha1.EthertypeIPv4),
		Entry(string(orcv1alpha1.ProtocolIPIP), orcv1alpha1.ProtocolIPIP, orcv1alpha1.EthertypeIPv4),
		Entry(string(orcv1alpha1.ProtocolOSPF), orcv1alpha1.ProtocolOSPF, orcv1alpha1.EthertypeIPv4),
		Entry(string(orcv1alpha1.ProtocolPGM), orcv1alpha1.ProtocolPGM, orcv1alpha1.EthertypeIPv4),
		Entry(string(orcv1alpha1.ProtocolRSVP), orcv1alpha1.ProtocolRSVP, orcv1alpha1.EthertypeIPv4),
		Entry(string(orcv1alpha1.ProtocolSCTP), orcv1alpha1.ProtocolSCTP, orcv1alpha1.EthertypeIPv4),
		Entry(string(orcv1alpha1.ProtocolTCP), orcv1alpha1.ProtocolTCP, orcv1alpha1.EthertypeIPv4),
		Entry(string(orcv1alpha1.ProtocolUDP), orcv1alpha1.ProtocolUDP, orcv1alpha1.EthertypeIPv4),
		Entry(string(orcv1alpha1.ProtocolUDPLITE), orcv1alpha1.ProtocolUDPLITE, orcv1alpha1.EthertypeIPv4),
		Entry(string(orcv1alpha1.ProtocolVRRP), orcv1alpha1.ProtocolVRRP, orcv1alpha1.EthertypeIPv4),
		Entry(string(orcv1alpha1.ProtocolIPV6ENCAP), orcv1alpha1.ProtocolIPV6ENCAP, orcv1alpha1.EthertypeIPv6),
		Entry(string(orcv1alpha1.ProtocolIPV6FRAG), orcv1alpha1.ProtocolIPV6FRAG, orcv1alpha1.EthertypeIPv6),
		Entry(string(orcv1alpha1.ProtocolIPV6ICMP), orcv1alpha1.ProtocolIPV6ICMP, orcv1alpha1.EthertypeIPv6),
		Entry(string(orcv1alpha1.ProtocolIPV6NONXT), orcv1alpha1.ProtocolIPV6NONXT, orcv1alpha1.EthertypeIPv6),
		Entry(string(orcv1alpha1.ProtocolIPV6OPTS), orcv1alpha1.ProtocolIPV6OPTS, orcv1alpha1.EthertypeIPv6),
		Entry(string(orcv1alpha1.ProtocolIPV6ROUTE), orcv1alpha1.ProtocolIPV6ROUTE, orcv1alpha1.EthertypeIPv6),
	)

	DescribeTable("should only allow valid port range for ICMP and ICMPv6 valid protocol",
		func(ctx context.Context, protocol orcv1alpha1.Protocol, ethertype orcv1alpha1.Ethertype) {
			securityGroup := securityGroupStub(namespace)
			patch := baseSecurityGroupPatch(securityGroup)
			sgRulePatchSpec := applyconfigv1alpha1.SecurityGroupRule().WithEthertype(ethertype).WithProtocol(protocol)
			patch.Spec.WithResource(applyconfigv1alpha1.SecurityGroupResourceSpec().WithRules(sgRulePatchSpec.WithPortRange(
				&applyconfigv1alpha1.PortRangeSpecApplyConfiguration{
					Min: ptr.To(v1alpha1.PortNumber(256)), Max: ptr.To(v1alpha1.PortNumber(0))})))
			Expect(applyObj(ctx, securityGroup, patch)).NotTo(Succeed(), "create security group")

			patch.Spec.WithResource(applyconfigv1alpha1.SecurityGroupResourceSpec().WithRules(sgRulePatchSpec.WithPortRange(
				&applyconfigv1alpha1.PortRangeSpecApplyConfiguration{
					Min: ptr.To(v1alpha1.PortNumber(0)), Max: ptr.To(v1alpha1.PortNumber(256))})))
			Expect(applyObj(ctx, securityGroup, patch)).NotTo(Succeed(), "create security group")

			patch.Spec.WithResource(applyconfigv1alpha1.SecurityGroupResourceSpec().WithRules(sgRulePatchSpec.WithPortRange(
				&applyconfigv1alpha1.PortRangeSpecApplyConfiguration{
					Min: ptr.To(v1alpha1.PortNumber(255)), Max: ptr.To(v1alpha1.PortNumber(255))})))
			Expect(applyObj(ctx, securityGroup, patch)).To(Succeed(), "create security group")
		},
		Entry(string(orcv1alpha1.ProtocolICMP), orcv1alpha1.ProtocolICMP, orcv1alpha1.EthertypeIPv4),
		Entry(string(orcv1alpha1.ProtocolICMPV6), orcv1alpha1.ProtocolICMPV6, orcv1alpha1.EthertypeIPv4),
	)

	It("should not permit invalid protocol", func(ctx context.Context) {
		securityGroup := securityGroupStub(namespace)
		patch := baseSecurityGroupPatch(securityGroup)
		patch.Spec.WithResource(applyconfigv1alpha1.SecurityGroupResourceSpec().WithRules(baseSGRulePatchSpec().WithProtocol("foo")))
		Expect(applyObj(ctx, securityGroup, patch)).NotTo(Succeed(), "create security group")
	})

	It("should permit without protocol", func(ctx context.Context) {
		securityGroup := securityGroupStub(namespace)
		patch := baseSecurityGroupPatch(securityGroup)
		sgRulePatchSpec := applyconfigv1alpha1.SecurityGroupRule().WithEthertype(orcv1alpha1.EthertypeIPv4)
		patch.Spec.WithResource(applyconfigv1alpha1.SecurityGroupResourceSpec().WithRules(sgRulePatchSpec))
		Expect(applyObj(ctx, securityGroup, patch)).To(Succeed(), "create security group")
	})

	It("should permit valid port range min and max", func(ctx context.Context) {
		securityGroup := securityGroupStub(namespace)
		patch := baseSecurityGroupPatch(securityGroup)
		sgRulePatchSpec := baseSGRulePatchSpec().WithProtocol(orcv1alpha1.ProtocolTCP)
		patch.Spec.WithResource(applyconfigv1alpha1.SecurityGroupResourceSpec().WithRules(sgRulePatchSpec.WithPortRange(
			&applyconfigv1alpha1.PortRangeSpecApplyConfiguration{
				Min: ptr.To(v1alpha1.PortNumber(22)), Max: ptr.To(v1alpha1.PortNumber(23))})))
		Expect(applyObj(ctx, securityGroup, patch)).To(Succeed(), "create security group")
		patch.Spec.WithResource(applyconfigv1alpha1.SecurityGroupResourceSpec().WithRules(sgRulePatchSpec.WithPortRange(
			&applyconfigv1alpha1.PortRangeSpecApplyConfiguration{
				Min: ptr.To(v1alpha1.PortNumber(22)), Max: ptr.To(v1alpha1.PortNumber(22))})))
		Expect(applyObj(ctx, securityGroup, patch)).To(Succeed(), "create security group")
	})

	It("should reject invalid port range min or max", func(ctx context.Context) {
		securityGroup := securityGroupStub(namespace)
		patch := baseSecurityGroupPatch(securityGroup)
		sgRulePatchSpec := baseSGRulePatchSpec().WithProtocol(orcv1alpha1.ProtocolTCP)
		patch.Spec.WithResource(applyconfigv1alpha1.SecurityGroupResourceSpec().WithRules(sgRulePatchSpec.WithPortRange(
			&applyconfigv1alpha1.PortRangeSpecApplyConfiguration{
				Min: ptr.To(v1alpha1.PortNumber(51)), Max: ptr.To(v1alpha1.PortNumber(50))})))
		Expect(applyObj(ctx, securityGroup, patch)).NotTo(Succeed(), "create security group")

		patch.Spec.WithResource(applyconfigv1alpha1.SecurityGroupResourceSpec().WithRules(sgRulePatchSpec.WithPortRange(
			&applyconfigv1alpha1.PortRangeSpecApplyConfiguration{
				Min: ptr.To(v1alpha1.PortNumber(-1))})))
		Expect(applyObj(ctx, securityGroup, patch)).NotTo(Succeed(), "create security group")

		patch.Spec.WithResource(applyconfigv1alpha1.SecurityGroupResourceSpec().WithRules(sgRulePatchSpec.WithPortRange(
			&applyconfigv1alpha1.PortRangeSpecApplyConfiguration{
				Min: ptr.To(v1alpha1.PortNumber(50))})))
		Expect(applyObj(ctx, securityGroup, patch)).NotTo(Succeed(), "create security group")

		patch.Spec.WithResource(applyconfigv1alpha1.SecurityGroupResourceSpec().WithRules(sgRulePatchSpec.WithPortRange(
			&applyconfigv1alpha1.PortRangeSpecApplyConfiguration{
				Max: ptr.To(v1alpha1.PortNumber(50))})))
		Expect(applyObj(ctx, securityGroup, patch)).NotTo(Succeed(), "create security group")
	})

	It("should reject invalid CIDR for RemoteIPPrefix", func(ctx context.Context) {
		securityGroup := securityGroupStub(namespace)
		patch := baseSecurityGroupPatch(securityGroup)
		sgRulePatchSpec := baseSGRulePatchSpec().WithProtocol(orcv1alpha1.ProtocolTCP).WithEthertype(orcv1alpha1.EthertypeIPv4)
		patch.Spec.WithResource(applyconfigv1alpha1.SecurityGroupResourceSpec().WithRules(sgRulePatchSpec.WithPortRange(
			&applyconfigv1alpha1.PortRangeSpecApplyConfiguration{
				Min: ptr.To(v1alpha1.PortNumber(22)), Max: ptr.To(v1alpha1.PortNumber(22))}).WithRemoteIPPrefix("foo")))
		Expect(applyObj(ctx, securityGroup, patch)).NotTo(Succeed(), "create security group")
	})

	It("should reject CIDR for RemoteIPPrefix that doesn't match the ethertype", func(ctx context.Context) {
		var sgRulePatchSpec *applyconfigv1alpha1.SecurityGroupRuleApplyConfiguration
		securityGroup := securityGroupStub(namespace)
		patch := baseSecurityGroupPatch(securityGroup)
		sgRulePatchSpec = applyconfigv1alpha1.SecurityGroupRule().WithEthertype(orcv1alpha1.EthertypeIPv6)
		patch.Spec.WithResource(applyconfigv1alpha1.SecurityGroupResourceSpec().WithRules(sgRulePatchSpec.WithPortRange(
			&applyconfigv1alpha1.PortRangeSpecApplyConfiguration{
				Min: ptr.To(v1alpha1.PortNumber(22)), Max: ptr.To(v1alpha1.PortNumber(22))}).WithRemoteIPPrefix("192.168.0.1/24")))
		Expect(applyObj(ctx, securityGroup, patch)).NotTo(Succeed(), "create security group")

		sgRulePatchSpec = applyconfigv1alpha1.SecurityGroupRule().WithEthertype(orcv1alpha1.EthertypeIPv4)
		patch.Spec.WithResource(applyconfigv1alpha1.SecurityGroupResourceSpec().WithRules(sgRulePatchSpec.WithPortRange(
			&applyconfigv1alpha1.PortRangeSpecApplyConfiguration{
				Min: ptr.To(v1alpha1.PortNumber(22)), Max: ptr.To(v1alpha1.PortNumber(22))}).WithRemoteIPPrefix("2001:db8::/47")))
		Expect(applyObj(ctx, securityGroup, patch)).NotTo(Succeed(), "create security group")
	})

	It("should permit valid CIDR that matches the ethertype", func(ctx context.Context) {
		securityGroup := securityGroupStub(namespace)
		patch := baseSecurityGroupPatch(securityGroup)
		patch.Spec.WithResource(applyconfigv1alpha1.SecurityGroupResourceSpec().WithRules(baseSGRulePatchSpec().WithRemoteIPPrefix("192.168.0.1/24")))
		Expect(applyObj(ctx, securityGroup, patch)).To(Succeed(), "create security group")
		sgRulePatchSpec := applyconfigv1alpha1.SecurityGroupRule().WithEthertype(orcv1alpha1.EthertypeIPv6).WithProtocol(orcv1alpha1.ProtocolTCP)
		patch.Spec.WithResource(applyconfigv1alpha1.SecurityGroupResourceSpec().WithRules(sgRulePatchSpec.WithRemoteIPPrefix("2001:db8::/47")))
		Expect(applyObj(ctx, securityGroup, patch)).To(Succeed(), "create security group")
	})
})
