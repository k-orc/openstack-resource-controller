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
	routerInterfaceName = "routerinterface"
)

func routerInterfaceStub(namespace *corev1.Namespace) *orcv1alpha1.RouterInterface {
	obj := &orcv1alpha1.RouterInterface{}
	obj.Name = routerInterfaceName
	obj.Namespace = namespace.Name
	return obj
}

func baseRouterInterfacePatch(ri client.Object) *applyconfigv1alpha1.RouterInterfaceApplyConfiguration {
	return applyconfigv1alpha1.RouterInterface(ri.GetName(), ri.GetNamespace())
}

var _ = Describe("ORC RouterInterface API validations", func() {
	var namespace *corev1.Namespace
	BeforeEach(func() {
		namespace = createNamespace()
	})

	It("should allow to create a valid router interface", func(ctx context.Context) {
		ri := routerInterfaceStub(namespace)
		patch := baseRouterInterfacePatch(ri)
		patch.WithSpec(applyconfigv1alpha1.RouterInterfaceSpec().
			WithType(orcv1alpha1.RouterInterfaceTypeSubnet).
			WithRouterRef("my-router").
			WithSubnetRef("my-subnet"))
		Expect(applyObj(ctx, ri, patch)).To(Succeed())
	})

	It("should reject missing required field type", func(ctx context.Context) {
		ri := routerInterfaceStub(namespace)
		patch := baseRouterInterfacePatch(ri)
		patch.WithSpec(applyconfigv1alpha1.RouterInterfaceSpec().
			WithRouterRef("my-router").
			WithSubnetRef("my-subnet"))
		Expect(applyObj(ctx, ri, patch)).To(MatchError(ContainSubstring("spec.type")))
	})

	It("should reject missing required field routerRef", func(ctx context.Context) {
		ri := routerInterfaceStub(namespace)
		patch := baseRouterInterfacePatch(ri)
		patch.WithSpec(applyconfigv1alpha1.RouterInterfaceSpec().
			WithType(orcv1alpha1.RouterInterfaceTypeSubnet).
			WithSubnetRef("my-subnet"))
		Expect(applyObj(ctx, ri, patch)).To(MatchError(ContainSubstring("spec.routerRef")))
	})

	It("should reject invalid type enum value", func(ctx context.Context) {
		ri := routerInterfaceStub(namespace)
		patch := baseRouterInterfacePatch(ri)
		patch.WithSpec(applyconfigv1alpha1.RouterInterfaceSpec().
			WithType("Invalid").
			WithRouterRef("my-router").
			WithSubnetRef("my-subnet"))
		Expect(applyObj(ctx, ri, patch)).NotTo(Succeed())
	})

	It("should require subnetRef when type is Subnet", func(ctx context.Context) {
		ri := routerInterfaceStub(namespace)
		patch := baseRouterInterfacePatch(ri)
		patch.WithSpec(applyconfigv1alpha1.RouterInterfaceSpec().
			WithType(orcv1alpha1.RouterInterfaceTypeSubnet).
			WithRouterRef("my-router"))
		Expect(applyObj(ctx, ri, patch)).To(MatchError(ContainSubstring("subnetRef is required when type is 'Subnet'")))
	})

	It("should be immutable", func(ctx context.Context) {
		ri := routerInterfaceStub(namespace)
		patch := baseRouterInterfacePatch(ri)
		patch.WithSpec(applyconfigv1alpha1.RouterInterfaceSpec().
			WithType(orcv1alpha1.RouterInterfaceTypeSubnet).
			WithRouterRef("router-a").
			WithSubnetRef("subnet-a"))
		Expect(applyObj(ctx, ri, patch)).To(Succeed())

		patch.WithSpec(applyconfigv1alpha1.RouterInterfaceSpec().
			WithType(orcv1alpha1.RouterInterfaceTypeSubnet).
			WithRouterRef("router-b").
			WithSubnetRef("subnet-a"))
		Expect(applyObj(ctx, ri, patch)).To(MatchError(ContainSubstring("RouterInterfaceResourceSpec is immutable")))
	})
})
