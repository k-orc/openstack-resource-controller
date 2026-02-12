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
	applicationcredentialName = "applicationcredential"
	applicationcredentialID   = "265c9e4f-0f5a-46e4-9f3f-fb8de25ae120"
)

func applicationcredentialStub(namespace *corev1.Namespace) *orcv1alpha1.ApplicationCredential {
	obj := &orcv1alpha1.ApplicationCredential{}
	obj.Name = applicationcredentialName
	obj.Namespace = namespace.Name
	return obj
}

func testApplicationCredentialResource() *applyconfigv1alpha1.ApplicationCredentialResourceSpecApplyConfiguration {
	return applyconfigv1alpha1.ApplicationCredentialResourceSpec().
		WithUserRef("user").
		WithSecretRef("applicationcredential-secret")
}

func baseApplicationCredentialPatch(obj client.Object) *applyconfigv1alpha1.ApplicationCredentialApplyConfiguration {
	return applyconfigv1alpha1.ApplicationCredential(obj.GetName(), obj.GetNamespace()).
		WithSpec(applyconfigv1alpha1.ApplicationCredentialSpec().
			WithCloudCredentialsRef(testCredentials()))
}

func testApplicationCredentialImport() *applyconfigv1alpha1.ApplicationCredentialImportApplyConfiguration {
	return applyconfigv1alpha1.ApplicationCredentialImport().WithID(applicationcredentialID)
}

var _ = Describe("ORC ApplicationCredential API validations", func() {
	var namespace *corev1.Namespace
	BeforeEach(func() {
		namespace = createNamespace()
	})

	runManagementPolicyTests(func() *corev1.Namespace { return namespace }, managementPolicyTestArgs[*applyconfigv1alpha1.ApplicationCredentialApplyConfiguration]{
		createObject: func(ns *corev1.Namespace) client.Object { return applicationcredentialStub(ns) },
		basePatch: func(obj client.Object) *applyconfigv1alpha1.ApplicationCredentialApplyConfiguration {
			return baseApplicationCredentialPatch(obj)
		},
		applyResource: func(p *applyconfigv1alpha1.ApplicationCredentialApplyConfiguration) {
			p.Spec.WithResource(testApplicationCredentialResource())
		},
		applyImport: func(p *applyconfigv1alpha1.ApplicationCredentialApplyConfiguration) {
			p.Spec.WithImport(testApplicationCredentialImport())
		},
		applyEmptyImport: func(p *applyconfigv1alpha1.ApplicationCredentialApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.ApplicationCredentialImport())
		},
		applyEmptyFilter: func(p *applyconfigv1alpha1.ApplicationCredentialApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.ApplicationCredentialImport().WithFilter(applyconfigv1alpha1.ApplicationCredentialFilter()))
		},
		applyValidFilter: func(p *applyconfigv1alpha1.ApplicationCredentialApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.ApplicationCredentialImport().WithFilter(applyconfigv1alpha1.ApplicationCredentialFilter().WithName("foo").WithUserRef("user")))
		},
		applyManaged: func(p *applyconfigv1alpha1.ApplicationCredentialApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged)
		},
		applyUnmanaged: func(p *applyconfigv1alpha1.ApplicationCredentialApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged)
		},
		applyManagedOptions: func(p *applyconfigv1alpha1.ApplicationCredentialApplyConfiguration) {
			p.Spec.WithManagedOptions(applyconfigv1alpha1.ManagedOptions().WithOnDelete(orcv1alpha1.OnDeleteDetach))
		},
		getManagementPolicy: func(obj client.Object) orcv1alpha1.ManagementPolicy {
			return obj.(*orcv1alpha1.ApplicationCredential).Spec.ManagementPolicy
		},
		getOnDelete: func(obj client.Object) orcv1alpha1.OnDelete {
			return obj.(*orcv1alpha1.ApplicationCredential).Spec.ManagedOptions.OnDelete
		},
	})

	It("should reject a applicationcredential without required fields", func(ctx context.Context) {
		obj := applicationcredentialStub(namespace)
		patch := baseApplicationCredentialPatch(obj)
		patch.Spec.WithResource(applyconfigv1alpha1.ApplicationCredentialResourceSpec())
		Expect(applyObj(ctx, obj, patch)).NotTo(Succeed())
	})

	It("should be immutable", func(ctx context.Context) {
		obj := applicationcredentialStub(namespace)
		patch := baseApplicationCredentialPatch(obj)
		patch.Spec.WithResource(testApplicationCredentialResource().
			WithUserRef("user-a"))
		Expect(applyObj(ctx, obj, patch)).To(Succeed())

		patch.Spec.WithResource(testApplicationCredentialResource().
			WithUserRef("user-b"))
		Expect(applyObj(ctx, obj, patch)).To(MatchError(ContainSubstring("ApplicationCredentialResourceSpec is immutable")))
	})

	DescribeTable("should permit valid http method",
		func(ctx context.Context, httpmethod orcv1alpha1.HTTPMethod) {
			obj := applicationcredentialStub(namespace)
			patch := baseApplicationCredentialPatch(obj)
			specPatch := applyconfigv1alpha1.ApplicationCredentialAccessRule().WithMethod(httpmethod)
			patch.Spec.WithResource(testApplicationCredentialResource().WithAccessRules(specPatch))
			Expect(applyObj(ctx, obj, patch)).To(Succeed(), "create application credential")
		},
		Entry(string(orcv1alpha1.HTTPMethodCONNECT), orcv1alpha1.HTTPMethodCONNECT),
		Entry(string(orcv1alpha1.HTTPMethodDELETE), orcv1alpha1.HTTPMethodDELETE),
		Entry(string(orcv1alpha1.HTTPMethodGET), orcv1alpha1.HTTPMethodGET),
		Entry(string(orcv1alpha1.HTTPMethodHEAD), orcv1alpha1.HTTPMethodHEAD),
		Entry(string(orcv1alpha1.HTTPMethodOPTIONS), orcv1alpha1.HTTPMethodOPTIONS),
		Entry(string(orcv1alpha1.HTTPMethodPATCH), orcv1alpha1.HTTPMethodPATCH),
		Entry(string(orcv1alpha1.HTTPMethodPOST), orcv1alpha1.HTTPMethodPOST),
		Entry(string(orcv1alpha1.HTTPMethodPUT), orcv1alpha1.HTTPMethodPUT),
		Entry(string(orcv1alpha1.HTTPMethodTRACE), orcv1alpha1.HTTPMethodTRACE),
	)

	It("should not permit invalid http method", func(ctx context.Context) {
		obj := applicationcredentialStub(namespace)
		patch := baseApplicationCredentialPatch(obj)
		patch.Spec.WithResource(testApplicationCredentialResource().WithAccessRules(applyconfigv1alpha1.ApplicationCredentialAccessRule().WithMethod("foo")))
		Expect(applyObj(ctx, obj, patch)).NotTo(Succeed(), "create application credential")
	})
})
