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
	qospolicyName = "qospolicy"
	qospolicyID   = "265c9e4f-0f5a-46e4-9f3f-fb8de25ae120"
)

func qospolicyStub(namespace *corev1.Namespace) *orcv1alpha1.QosPolicy {
	obj := &orcv1alpha1.QosPolicy{}
	obj.Name = qospolicyName
	obj.Namespace = namespace.Name
	return obj
}

func testQosPolicyResource() *applyconfigv1alpha1.QosPolicyResourceSpecApplyConfiguration {
	return applyconfigv1alpha1.QosPolicyResourceSpec()
}

func baseQosPolicyPatch(obj client.Object) *applyconfigv1alpha1.QosPolicyApplyConfiguration {
	return applyconfigv1alpha1.QosPolicy(obj.GetName(), obj.GetNamespace()).
		WithSpec(applyconfigv1alpha1.QosPolicySpec().
			WithCloudCredentialsRef(testCredentials()))
}

func testQosPolicyImport() *applyconfigv1alpha1.QosPolicyImportApplyConfiguration {
	return applyconfigv1alpha1.QosPolicyImport().WithID(qospolicyID)
}

var _ = Describe("ORC QosPolicy API validations", func() {
	var namespace *corev1.Namespace
	BeforeEach(func() {
		namespace = createNamespace()
	})

	runManagementPolicyTests(func() *corev1.Namespace { return namespace }, managementPolicyTestArgs[*applyconfigv1alpha1.QosPolicyApplyConfiguration]{
		createObject: func(ns *corev1.Namespace) client.Object { return qospolicyStub(ns) },
		basePatch: func(obj client.Object) *applyconfigv1alpha1.QosPolicyApplyConfiguration {
			return baseQosPolicyPatch(obj)
		},
		applyResource: func(p *applyconfigv1alpha1.QosPolicyApplyConfiguration) {
			p.Spec.WithResource(testQosPolicyResource())
		},
		applyImport: func(p *applyconfigv1alpha1.QosPolicyApplyConfiguration) {
			p.Spec.WithImport(testQosPolicyImport())
		},
		applyEmptyImport: func(p *applyconfigv1alpha1.QosPolicyApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.QosPolicyImport())
		},
		applyEmptyFilter: func(p *applyconfigv1alpha1.QosPolicyApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.QosPolicyImport().WithFilter(applyconfigv1alpha1.QosPolicyFilter()))
		},
		applyValidFilter: func(p *applyconfigv1alpha1.QosPolicyApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.QosPolicyImport().WithFilter(applyconfigv1alpha1.QosPolicyFilter().WithName("foo")))
		},
		applyManaged: func(p *applyconfigv1alpha1.QosPolicyApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged)
		},
		applyUnmanaged: func(p *applyconfigv1alpha1.QosPolicyApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged)
		},
		applyManagedOptions: func(p *applyconfigv1alpha1.QosPolicyApplyConfiguration) {
			p.Spec.WithManagedOptions(applyconfigv1alpha1.ManagedOptions().WithOnDelete(orcv1alpha1.OnDeleteDetach))
		},
		getManagementPolicy: func(obj client.Object) orcv1alpha1.ManagementPolicy {
			return obj.(*orcv1alpha1.QosPolicy).Spec.ManagementPolicy
		},
		getOnDelete: func(obj client.Object) orcv1alpha1.OnDelete {
			return obj.(*orcv1alpha1.QosPolicy).Spec.ManagedOptions.OnDelete
		},
	})

	It("should have immutable projectRef", func(ctx context.Context) {
		obj := qospolicyStub(namespace)
		patch := baseQosPolicyPatch(obj)
		patch.Spec.WithResource(testQosPolicyResource().
			WithProjectRef("project-a"))
		Expect(applyObj(ctx, obj, patch)).To(Succeed())

		patch.Spec.WithResource(testQosPolicyResource().
			WithProjectRef("project-b"))
		Expect(applyObj(ctx, obj, patch)).To(MatchError(ContainSubstring("projectRef is immutable")))
	})

	It("should reject duplicate tags", func(ctx context.Context) {
		obj := qospolicyStub(namespace)
		patch := baseQosPolicyPatch(obj)
		patch.Spec.WithResource(testQosPolicyResource().
			WithTags("foo", "bar", "foo"))
		Expect(applyObj(ctx, obj, patch)).NotTo(Succeed())
	})

	It("should permit unique tags", func(ctx context.Context) {
		obj := qospolicyStub(namespace)
		patch := baseQosPolicyPatch(obj)
		patch.Spec.WithResource(testQosPolicyResource().
			WithTags("foo", "bar"))
		Expect(applyObj(ctx, obj, patch)).To(Succeed())
	})
})
