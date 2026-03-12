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

	It("should allow to create a minimal keypair and managementPolicy should default to managed", func(ctx context.Context) {
		keypair := keypairStub(namespace)
		patch := baseKeypairPatch(keypair)
		patch.Spec.WithResource(testKeypairResource())
		Expect(applyObj(ctx, keypair, patch)).To(Succeed())
		Expect(keypair.Spec.ManagementPolicy).To(Equal(orcv1alpha1.ManagementPolicyManaged))
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

	It("should require import for unmanaged", func(ctx context.Context) {
		keypair := keypairStub(namespace)
		patch := baseKeypairPatch(keypair)
		patch.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged)
		Expect(applyObj(ctx, keypair, patch)).To(MatchError(ContainSubstring("import must be specified when policy is unmanaged")))

		patch.Spec.WithImport(testKeypairImport())
		Expect(applyObj(ctx, keypair, patch)).To(Succeed())
	})

	It("should not permit unmanaged with resource", func(ctx context.Context) {
		keypair := keypairStub(namespace)
		patch := baseKeypairPatch(keypair)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(testKeypairImport()).
			WithResource(testKeypairResource())
		Expect(applyObj(ctx, keypair, patch)).To(MatchError(ContainSubstring("resource may not be specified when policy is unmanaged")))
	})

	It("should not permit empty import", func(ctx context.Context) {
		keypair := keypairStub(namespace)
		patch := baseKeypairPatch(keypair)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(applyconfigv1alpha1.KeyPairImport())
		Expect(applyObj(ctx, keypair, patch)).To(MatchError(ContainSubstring("spec.import in body should have at least 1 properties")))
	})

	It("should not permit empty import filter", func(ctx context.Context) {
		keypair := keypairStub(namespace)
		patch := baseKeypairPatch(keypair)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(applyconfigv1alpha1.KeyPairImport().
				WithFilter(applyconfigv1alpha1.KeyPairFilter()))
		Expect(applyObj(ctx, keypair, patch)).To(MatchError(ContainSubstring("spec.import.filter in body should have at least 1 properties")))
	})

	It("should permit import filter with name", func(ctx context.Context) {
		keypair := keypairStub(namespace)
		patch := baseKeypairPatch(keypair)
		patch.Spec.
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithImport(applyconfigv1alpha1.KeyPairImport().
				WithFilter(applyconfigv1alpha1.KeyPairFilter().WithName("foo")))
		Expect(applyObj(ctx, keypair, patch)).To(Succeed())
	})

	It("should require resource for managed", func(ctx context.Context) {
		keypair := keypairStub(namespace)
		patch := baseKeypairPatch(keypair)
		patch.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged)
		Expect(applyObj(ctx, keypair, patch)).To(MatchError(ContainSubstring("resource must be specified when policy is managed")))

		patch.Spec.WithResource(testKeypairResource())
		Expect(applyObj(ctx, keypair, patch)).To(Succeed())
	})

	It("should not permit managed with import", func(ctx context.Context) {
		keypair := keypairStub(namespace)
		patch := baseKeypairPatch(keypair)
		patch.Spec.
			WithImport(testKeypairImport()).
			WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged).
			WithResource(testKeypairResource())
		Expect(applyObj(ctx, keypair, patch)).To(MatchError(ContainSubstring("import may not be specified when policy is managed")))
	})

	It("should not permit managedOptions for unmanaged", func(ctx context.Context) {
		keypair := keypairStub(namespace)
		patch := baseKeypairPatch(keypair)
		patch.Spec.
			WithImport(testKeypairImport()).
			WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged).
			WithManagedOptions(applyconfigv1alpha1.ManagedOptions().
				WithOnDelete(orcv1alpha1.OnDeleteDetach))
		Expect(applyObj(ctx, keypair, patch)).To(MatchError(ContainSubstring("managedOptions may only be provided when policy is managed")))
	})

	It("should permit managedOptions for managed", func(ctx context.Context) {
		keypair := keypairStub(namespace)
		patch := baseKeypairPatch(keypair)
		patch.Spec.WithResource(testKeypairResource()).
			WithManagedOptions(applyconfigv1alpha1.ManagedOptions().
				WithOnDelete(orcv1alpha1.OnDeleteDetach))
		Expect(applyObj(ctx, keypair, patch)).To(Succeed())
		Expect(keypair.Spec.ManagedOptions.OnDelete).To(Equal(orcv1alpha1.OnDelete("detach")))
	})
})
