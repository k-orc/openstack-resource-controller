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
	portName = "port-foo"
	portID   = "87e14a4c-5f16-4e45-8a2b-7c34b5b9d59f"
)

var longString = "adrmUWSxYcp5FPTxjE01uRjg6NP3BqCIHxd6spdrIWTiV6XtLxmM4AHAiIzhZ7bqlv"

func portStub(namespace *corev1.Namespace) *orcv1alpha1.Port {
	obj := &orcv1alpha1.Port{}
	obj.Name = portName
	obj.Namespace = namespace.Name
	return obj
}

func basePortPatch(port client.Object) *applyconfigv1alpha1.PortApplyConfiguration {
	return applyconfigv1alpha1.Port(port.GetName(), port.GetNamespace()).
		WithSpec(applyconfigv1alpha1.PortSpec().
			WithCloudCredentialsRef(testCredentials()))
}

var _ = Describe("ORC Port API validations", func() {
	var namespace *corev1.Namespace
	BeforeEach(func() {
		namespace = createNamespace()
	})

	It("should allow to create a minimal port and managementPolicy should default to managed", func(ctx context.Context) {
		port := portStub(namespace)
		patch := basePortPatch(port)
		patch.Spec.WithResource(applyconfigv1alpha1.PortResourceSpec().WithNetworkRef(networkName))
		Expect(applyObj(ctx, port, patch)).To(Succeed())
		Expect(port.Spec.ManagementPolicy).To(Equal(orcv1alpha1.ManagementPolicyManaged))
	})

	It("should allow to create a port with securityGroupRefs when portSecurity is enabled", func(ctx context.Context) {
		port := portStub(namespace)
		patch := basePortPatch(port)
		patch.Spec.WithResource(applyconfigv1alpha1.PortResourceSpec().
			WithNetworkRef(networkName).
			WithSecurityGroupRefs("sg-foo").
			WithPortSecurity(orcv1alpha1.PortSecurityEnabled))
		Expect(applyObj(ctx, port, patch)).To(Succeed())
		Expect(port.Spec.Resource.SecurityGroupRefs).To(Equal([]orcv1alpha1.OpenStackName{"sg-foo"}))
		Expect(port.Spec.Resource.PortSecurity).To(Equal(orcv1alpha1.PortSecurityEnabled))
	})

	It("should allow to create a port with portSecurity set to Inherit", func(ctx context.Context) {
		port := portStub(namespace)
		patch := basePortPatch(port)
		patch.Spec.WithResource(applyconfigv1alpha1.PortResourceSpec().
			WithNetworkRef(networkName).
			WithPortSecurity(orcv1alpha1.PortSecurityInherit))
		Expect(applyObj(ctx, port, patch)).To(Succeed())
		Expect(port.Spec.Resource.PortSecurity).To(Equal(orcv1alpha1.PortSecurityInherit))
	})

	It("should allow to create a port with portSecurity set to Disabled", func(ctx context.Context) {
		port := portStub(namespace)
		patch := basePortPatch(port)
		patch.Spec.WithResource(applyconfigv1alpha1.PortResourceSpec().
			WithNetworkRef(networkName).
			WithPortSecurity(orcv1alpha1.PortSecurityDisabled))
		Expect(applyObj(ctx, port, patch)).To(Succeed())
		Expect(port.Spec.Resource.PortSecurity).To(Equal(orcv1alpha1.PortSecurityDisabled))
	})

	It("should not allow to create a port with securityGroupRefs when portSecurity is explicitly set to Disabled", func(ctx context.Context) {
		port := portStub(namespace)
		patch := basePortPatch(port)
		patch.Spec.WithResource(applyconfigv1alpha1.PortResourceSpec().
			WithNetworkRef(networkName).
			WithSecurityGroupRefs("sg-foo").
			WithPortSecurity(orcv1alpha1.PortSecurityDisabled))
		Expect(applyObj(ctx, port, patch)).To(MatchError(ContainSubstring("Invalid value: \"object\": securityGroupRefs must be empty when portSecurity is set to Disabled")))
	})

	It("should not allow to create a port with allowedAddressPairs when portSecurity is explicitly set to Disabled", func(ctx context.Context) {
		port := portStub(namespace)
		patch := basePortPatch(port)
		var ip orcv1alpha1.IPvAny = "192.168.11.11"
		pairs := applyconfigv1alpha1.AllowedAddressPairApplyConfiguration{IP: &ip}
		patch.Spec.WithResource(applyconfigv1alpha1.PortResourceSpec().
			WithNetworkRef(networkName).
			WithAllowedAddressPairs(&pairs).
			WithPortSecurity(orcv1alpha1.PortSecurityDisabled))
		Expect(applyObj(ctx, port, patch)).To(MatchError(ContainSubstring("Invalid value: \"object\": allowedAddressPairs must be empty when portSecurity is set to Disabled")))
	})

	It("should reject to create a port with an invalid vnicType", func(ctx context.Context) {
		port := portStub(namespace)
		patch := basePortPatch(port)
		patch.Spec.WithResource(applyconfigv1alpha1.PortResourceSpec().WithVNICType(longString))
		Expect(applyObj(ctx, port, patch)).To(MatchError(ContainSubstring("spec.resource.vnicType: Too long: may not be longer than 64")))
	})

	It("should not allow hostID to be modified", func(ctx context.Context) {
		port := portStub(namespace)
		patch := basePortPatch(port)
		patch.Spec.WithResource(applyconfigv1alpha1.PortResourceSpec().
			WithNetworkRef(networkName).
			WithHostID(applyconfigv1alpha1.HostID().WithID("host-a")))
		Expect(applyObj(ctx, port, patch)).To(Succeed())

		patch.Spec.WithResource(applyconfigv1alpha1.PortResourceSpec().
			WithNetworkRef(networkName).
			WithHostID(applyconfigv1alpha1.HostID().WithID("host-b")))
		Expect(applyObj(ctx, port, patch)).To(MatchError(ContainSubstring("hostID is immutable")))
	})

	// Note: we can't create a test for when the portSecurity is set to Inherit and the securityGroupRefs are set, because
	// the validation is done in the OpenStack API and not in the ORC API. The OpenStack API will return an error if
	// the network has port security disabled and the port has security group references.
})
