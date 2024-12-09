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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	applyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/pkg/clients/applyconfiguration/api/v1alpha1"
)

const flavorName = "flavor"

func flavorStub(namespace *corev1.Namespace) *orcv1alpha1.Flavor {
	obj := &orcv1alpha1.Flavor{}
	obj.Name = flavorName
	obj.Namespace = namespace.Name
	return obj
}

func baseFlavorPatch(flavor client.Object) *applyconfigv1alpha1.FlavorApplyConfiguration {
	return applyconfigv1alpha1.Flavor(flavor.GetName(), flavor.GetNamespace()).
		WithSpec(applyconfigv1alpha1.FlavorSpec().
			WithCloudCredentialsRef(testCredentials()))
}

var _ = Describe("ORC Flavor API validations", func() {
	var namespace *corev1.Namespace
	BeforeEach(func() {
		namespace = createNamespace()
	})

	It("should allow to create a minimal flavor and managementPolicy should default to managed", func(ctx context.Context) {
		flavor := flavorStub(namespace)
		patch := baseFlavorPatch(flavor)
		patch.Spec.WithResource(applyconfigv1alpha1.FlavorResourceSpec().WithRAM(1).WithVcpus(1))
		Expect(applyObj(ctx, flavor, patch)).To(Succeed())
		Expect(flavor.Spec.ManagementPolicy).To(Equal(orcv1alpha1.ManagementPolicyManaged))
	})

	It("should reject a flavor without required fields", func(ctx context.Context) {
		flavor := flavorStub(namespace)
		patch := baseFlavorPatch(flavor)
		patch.Spec.WithResource(applyconfigv1alpha1.FlavorResourceSpec())
		Expect(applyObj(ctx, flavor, patch)).NotTo(Succeed())
		patch.Spec.WithResource(applyconfigv1alpha1.FlavorResourceSpec().WithRAM(1))
		Expect(applyObj(ctx, flavor, patch)).NotTo(Succeed())
		patch.Spec.WithResource(applyconfigv1alpha1.FlavorResourceSpec().WithVcpus(1))
		Expect(applyObj(ctx, flavor, patch)).NotTo(Succeed())
	})

	It("should reject a flavor with values less than minimal", func(ctx context.Context) {
		flavor := flavorStub(namespace)
		patch := baseFlavorPatch(flavor)
		patch.Spec.WithResource(applyconfigv1alpha1.FlavorResourceSpec().WithRAM(1).WithVcpus(0))
		Expect(applyObj(ctx, flavor, patch)).NotTo(Succeed())
		patch.Spec.WithResource(applyconfigv1alpha1.FlavorResourceSpec().WithRAM(0).WithVcpus(1))
		Expect(applyObj(ctx, flavor, patch)).NotTo(Succeed())
	})

	It("should reject a flavor with values greater than max", func(ctx context.Context) {
		flavor := flavorStub(namespace)
		patch := baseFlavorPatch(flavor)
		maxString := orcv1alpha1.OpenStackDescription(strings.Repeat("a", 1025))
		patch.Spec.WithResource(applyconfigv1alpha1.FlavorResourceSpec().WithRAM(1).WithVcpus(1).WithDescription(maxString))
		Expect(applyObj(ctx, flavor, patch)).NotTo(Succeed())

	})
})
