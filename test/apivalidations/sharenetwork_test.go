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
	shareNetworkName = "sharenetwork-foo"
	shareNetworkID   = "7b7a8e4c-1c2d-4e5f-9a8b-3c4d5e6f7a8b"
)

func shareNetworkStub(namespace *corev1.Namespace) *orcv1alpha1.ShareNetwork {
	obj := &orcv1alpha1.ShareNetwork{}
	obj.Name = shareNetworkName
	obj.Namespace = namespace.Name
	return obj
}

func testShareNetworkResource() *applyconfigv1alpha1.ShareNetworkResourceSpecApplyConfiguration {
	return applyconfigv1alpha1.ShareNetworkResourceSpec()
}

func baseShareNetworkPatch(shareNetwork client.Object) *applyconfigv1alpha1.ShareNetworkApplyConfiguration {
	return applyconfigv1alpha1.ShareNetwork(shareNetwork.GetName(), shareNetwork.GetNamespace()).
		WithSpec(applyconfigv1alpha1.ShareNetworkSpec().
			WithCloudCredentialsRef(testCredentials()))
}

var _ = Describe("ORC ShareNetwork API validations", func() {
	var namespace *corev1.Namespace
	BeforeEach(func() {
		namespace = createNamespace()
	})

	It("should allow to create a minimal share network and managementPolicy should default to managed", func(ctx context.Context) {
		shareNetwork := shareNetworkStub(namespace)
		patch := baseShareNetworkPatch(shareNetwork)
		patch.Spec.WithResource(testShareNetworkResource())
		Expect(applyObj(ctx, shareNetwork, patch)).To(Succeed())
		Expect(shareNetwork.Spec.ManagementPolicy).To(Equal(orcv1alpha1.ManagementPolicyManaged))
	})

	It("should not permit empty import filter", func(ctx context.Context) {
		shareNetwork := shareNetworkStub(namespace)
		patch := baseShareNetworkPatch(shareNetwork)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(applyconfigv1alpha1.ShareNetworkImport().
				WithFilter(applyconfigv1alpha1.ShareNetworkFilter()))
		Expect(applyObj(ctx, shareNetwork, patch)).NotTo(Succeed())
	})

	It("should permit valid import filter", func(ctx context.Context) {
		shareNetwork := shareNetworkStub(namespace)
		patch := baseShareNetworkPatch(shareNetwork)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(applyconfigv1alpha1.ShareNetworkImport().
				WithFilter(applyconfigv1alpha1.ShareNetworkFilter().WithName("foo").WithDescription("bar")))
		Expect(applyObj(ctx, shareNetwork, patch)).To(Succeed())
	})

	// NetworkRef and SubnetRef co-dependency validation tests
	Describe("networkRef and subnetRef validation", func() {
		It("should allow both networkRef and subnetRef together", func(ctx context.Context) {
			shareNetwork := shareNetworkStub(namespace)
			patch := baseShareNetworkPatch(shareNetwork)
			patch.Spec.WithResource(
				applyconfigv1alpha1.ShareNetworkResourceSpec().
					WithNetworkRef("foo").
					WithSubnetRef("bar"))
			Expect(applyObj(ctx, shareNetwork, patch)).To(Succeed(), "should accept both networkRef and subnetRef")
		})

		It("should allow neither networkRef nor subnetRef", func(ctx context.Context) {
			shareNetwork := shareNetworkStub(namespace)
			patch := baseShareNetworkPatch(shareNetwork)
			patch.Spec.WithResource(testShareNetworkResource())
			Expect(applyObj(ctx, shareNetwork, patch)).To(Succeed(), "should accept when both are absent")
		})

		It("should reject networkRef without subnetRef", func(ctx context.Context) {
			shareNetwork := shareNetworkStub(namespace)
			patch := baseShareNetworkPatch(shareNetwork)
			patch.Spec.WithResource(
				applyconfigv1alpha1.ShareNetworkResourceSpec().
					WithNetworkRef("foo"))
			Expect(applyObj(ctx, shareNetwork, patch)).NotTo(Succeed(), "should reject networkRef without subnetRef")
		})

		It("should reject subnetRef without networkRef", func(ctx context.Context) {
			shareNetwork := shareNetworkStub(namespace)
			patch := baseShareNetworkPatch(shareNetwork)
			patch.Spec.WithResource(
				applyconfigv1alpha1.ShareNetworkResourceSpec().
					WithSubnetRef("bar"))
			Expect(applyObj(ctx, shareNetwork, patch)).NotTo(Succeed(), "should reject subnetRef without networkRef")
		})
	})
})
