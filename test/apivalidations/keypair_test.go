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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	applyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/pkg/clients/applyconfiguration/api/v1alpha1"
)

const (
	keypairName = "keypair"
	keypairID   = "265c9e4f-0f5a-46e4-9f3f-fb8de25ae124"
)

func keypairStub(namespace *corev1.Namespace) *orcv1alpha1.KeyPair {
	obj := &orcv1alpha1.KeyPair{}
	obj.Name = keypairName
	obj.Namespace = namespace.Name
	return obj
}

func testKeypairResource() *applyconfigv1alpha1.KeyPairResourceSpecApplyConfiguration {
	return applyconfigv1alpha1.KeyPairResourceSpec().WithPublicKey("ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQ")
}

func baseKeypairPatch(keypair client.Object) *applyconfigv1alpha1.KeyPairApplyConfiguration {
	return applyconfigv1alpha1.KeyPair(keypair.GetName(), keypair.GetNamespace()).
		WithSpec(applyconfigv1alpha1.KeyPairSpec().
			WithCloudCredentialsRef(testCredentials()))
}

func testKeypairImport() *applyconfigv1alpha1.KeyPairImportApplyConfiguration {
	return applyconfigv1alpha1.KeyPairImport().WithID(keypairID)
}

var _ = Describe("ORC KeyPair API validations", func() {
	var namespace *corev1.Namespace
	BeforeEach(func() {
		namespace = createNamespace()
	})

	runManagementPolicyTests(func() *corev1.Namespace { return namespace }, managementPolicyTestArgs[*applyconfigv1alpha1.KeyPairApplyConfiguration]{
		createObject:  func(ns *corev1.Namespace) client.Object { return keypairStub(ns) },
		basePatch:     func(obj client.Object) *applyconfigv1alpha1.KeyPairApplyConfiguration { return baseKeypairPatch(obj) },
		applyResource: func(p *applyconfigv1alpha1.KeyPairApplyConfiguration) { p.Spec.WithResource(testKeypairResource()) },
		applyImport:   func(p *applyconfigv1alpha1.KeyPairApplyConfiguration) { p.Spec.WithImport(testKeypairImport()) },
		applyEmptyImport: func(p *applyconfigv1alpha1.KeyPairApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.KeyPairImport())
		},
		applyEmptyFilter: func(p *applyconfigv1alpha1.KeyPairApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.KeyPairImport().WithFilter(applyconfigv1alpha1.KeyPairFilter()))
		},
		applyValidFilter: func(p *applyconfigv1alpha1.KeyPairApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.KeyPairImport().WithFilter(applyconfigv1alpha1.KeyPairFilter().WithName("foo")))
		},
		applyManaged: func(p *applyconfigv1alpha1.KeyPairApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged)
		},
		applyUnmanaged: func(p *applyconfigv1alpha1.KeyPairApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged)
		},
		applyManagedOptions: func(p *applyconfigv1alpha1.KeyPairApplyConfiguration) {
			p.Spec.WithManagedOptions(applyconfigv1alpha1.ManagedOptions().WithOnDelete(orcv1alpha1.OnDeleteDetach))
		},
		getManagementPolicy: func(obj client.Object) orcv1alpha1.ManagementPolicy {
			return obj.(*orcv1alpha1.KeyPair).Spec.ManagementPolicy
		},
		getOnDelete: func(obj client.Object) orcv1alpha1.OnDelete {
			return obj.(*orcv1alpha1.KeyPair).Spec.ManagedOptions.OnDelete
		},
	})

	It("should reject a keypair without required field publicKey", func(ctx context.Context) {
		keypair := keypairStub(namespace)
		patch := baseKeypairPatch(keypair)
		patch.Spec.WithResource(applyconfigv1alpha1.KeyPairResourceSpec())
		Expect(applyObj(ctx, keypair, patch)).To(MatchError(ContainSubstring("spec.resource.publicKey")))
	})

	It("should reject invalid type enum value", func(ctx context.Context) {
		keypair := keypairStub(namespace)
		patch := baseKeypairPatch(keypair)
		patch.Spec.WithResource(applyconfigv1alpha1.KeyPairResourceSpec().
			WithPublicKey("ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQ").
			WithType("invalid"))
		Expect(applyObj(ctx, keypair, patch)).NotTo(Succeed())
	})

	It("should permit valid type enum values", func(ctx context.Context) {
		keypair := keypairStub(namespace)
		patch := baseKeypairPatch(keypair)
		patch.Spec.WithResource(applyconfigv1alpha1.KeyPairResourceSpec().
			WithPublicKey("ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQ").
			WithType("ssh"))
		Expect(applyObj(ctx, keypair, patch)).To(Succeed())
	})

	It("should reject publicKey exceeding max length", func(ctx context.Context) {
		keypair := keypairStub(namespace)
		patch := baseKeypairPatch(keypair)
		patch.Spec.WithResource(applyconfigv1alpha1.KeyPairResourceSpec().
			WithPublicKey(strings.Repeat("a", 16385)))
		Expect(applyObj(ctx, keypair, patch)).To(MatchError(ContainSubstring("spec.resource.publicKey")))
	})
})
