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

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/api/v1alpha1"
	applyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/pkg/clients/applyconfiguration/api/v1alpha1"
)

const (
	rbacpolicyName = "rbacpolicy"
	rbacpolicyID   = "265c9e4f-0f5a-46e4-9f3f-fb8de25ae120"
)

func rbacpolicyStub(namespace *corev1.Namespace) *orcv1alpha1.RBACPolicy {
	obj := &orcv1alpha1.RBACPolicy{}
	obj.Name = rbacpolicyName
	obj.Namespace = namespace.Name
	return obj
}

func testRBACPolicyResource() *applyconfigv1alpha1.RBACPolicyResourceSpecApplyConfiguration {
	return applyconfigv1alpha1.RBACPolicyResourceSpec().
		WithNetworkRef("network").
		WithAction(orcv1alpha1.RBACPolicyActionAccessShared).
		WithTargetProjectID("3fac9d0b0e0e4b0e9b0e000000000001")
}

func baseRBACPolicyPatch(obj client.Object) *applyconfigv1alpha1.RBACPolicyApplyConfiguration {
	return applyconfigv1alpha1.RBACPolicy(obj.GetName(), obj.GetNamespace()).
		WithSpec(applyconfigv1alpha1.RBACPolicySpec().
			WithCloudCredentialsRef(testCredentials()))
}

func testRBACPolicyImport() *applyconfigv1alpha1.RBACPolicyImportApplyConfiguration {
	return applyconfigv1alpha1.RBACPolicyImport().WithID(rbacpolicyID)
}

var _ = Describe("ORC RBACPolicy API validations", func() {
	var namespace *corev1.Namespace
	BeforeEach(func() {
		namespace = createNamespace()
	})

	runManagementPolicyTests(func() *corev1.Namespace { return namespace }, managementPolicyTestArgs[*applyconfigv1alpha1.RBACPolicyApplyConfiguration]{
		createObject: func(ns *corev1.Namespace) client.Object { return rbacpolicyStub(ns) },
		basePatch: func(obj client.Object) *applyconfigv1alpha1.RBACPolicyApplyConfiguration {
			return baseRBACPolicyPatch(obj)
		},
		applyResource: func(p *applyconfigv1alpha1.RBACPolicyApplyConfiguration) {
			p.Spec.WithResource(testRBACPolicyResource())
		},
		applyImport: func(p *applyconfigv1alpha1.RBACPolicyApplyConfiguration) {
			p.Spec.WithImport(testRBACPolicyImport())
		},
		applyEmptyImport: func(p *applyconfigv1alpha1.RBACPolicyApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.RBACPolicyImport())
		},
		applyEmptyFilter: func(p *applyconfigv1alpha1.RBACPolicyApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.RBACPolicyImport().WithFilter(applyconfigv1alpha1.RBACPolicyFilter()))
		},
		applyValidFilter: func(p *applyconfigv1alpha1.RBACPolicyApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.RBACPolicyImport().WithFilter(applyconfigv1alpha1.RBACPolicyFilter().WithTargetProjectID("foo")))
		},
		applyManaged: func(p *applyconfigv1alpha1.RBACPolicyApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged)
		},
		applyUnmanaged: func(p *applyconfigv1alpha1.RBACPolicyApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged)
		},
		applyManagedOptions: func(p *applyconfigv1alpha1.RBACPolicyApplyConfiguration) {
			p.Spec.WithManagedOptions(applyconfigv1alpha1.ManagedOptions().WithOnDelete(orcv1alpha1.OnDeleteDetach))
		},
		getManagementPolicy: func(obj client.Object) orcv1alpha1.ManagementPolicy {
			return obj.(*orcv1alpha1.RBACPolicy).Spec.ManagementPolicy
		},
		getOnDelete: func(obj client.Object) orcv1alpha1.OnDelete {
			return obj.(*orcv1alpha1.RBACPolicy).Spec.ManagedOptions.OnDelete
		},
	})

	It("should reject a rbacpolicy without required fields", func(ctx context.Context) {
		obj := rbacpolicyStub(namespace)
		patch := baseRBACPolicyPatch(obj)
		patch.Spec.WithResource(applyconfigv1alpha1.RBACPolicyResourceSpec())
		Expect(applyObj(ctx, obj, patch)).NotTo(Succeed())
	})

	It("should have immutable networkRef", func(ctx context.Context) {
		obj := rbacpolicyStub(namespace)
		patch := baseRBACPolicyPatch(obj)
		patch.Spec.WithResource(testRBACPolicyResource().
			WithNetworkRef("network-a"))
		Expect(applyObj(ctx, obj, patch)).To(Succeed())

		patch.Spec.WithResource(testRBACPolicyResource().
			WithNetworkRef("network-b"))
		Expect(applyObj(ctx, obj, patch)).To(MatchError(ContainSubstring("networkRef is immutable")))
	})

	It("should have immutable action", func(ctx context.Context) {
		obj := rbacpolicyStub(namespace)
		patch := baseRBACPolicyPatch(obj)
		patch.Spec.WithResource(testRBACPolicyResource().
			WithAction(orcv1alpha1.RBACPolicyActionAccessShared))
		Expect(applyObj(ctx, obj, patch)).To(Succeed())

		patch.Spec.WithResource(testRBACPolicyResource().
			WithAction(orcv1alpha1.RBACPolicyActionAccessExternal))
		Expect(applyObj(ctx, obj, patch)).To(MatchError(ContainSubstring("action is immutable")))
	})

	It("should reject an invalid action value", func(ctx context.Context) {
		obj := rbacpolicyStub(namespace)
		patch := baseRBACPolicyPatch(obj)
		patch.Spec.WithResource(testRBACPolicyResource().
			WithAction("not_a_real_action"))
		Expect(applyObj(ctx, obj, patch)).NotTo(Succeed())
	})

	It("should allow mutating targetProjectID", func(ctx context.Context) {
		obj := rbacpolicyStub(namespace)
		patch := baseRBACPolicyPatch(obj)
		patch.Spec.WithResource(testRBACPolicyResource().
			WithTargetProjectID("3fac9d0b0e0e4b0e9b0e000000000001"))
		Expect(applyObj(ctx, obj, patch)).To(Succeed())

		// Neutron's own UpdateOpts only supports changing target_tenant - confirmed via
		// gophercloud - so this must stay mutable, unlike networkRef/action above.
		patch.Spec.WithResource(testRBACPolicyResource().
			WithTargetProjectID("4fac9d0b0e0e4b0e9b0e000000000002"))
		Expect(applyObj(ctx, obj, patch)).To(Succeed())
	})
})
