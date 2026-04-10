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
	projectObjName = "project"
	projectID      = "265c9e4f-0f5a-46e4-9f3f-fb8de25ae125"
)

func projectStub(namespace *corev1.Namespace) *orcv1alpha1.Project {
	obj := &orcv1alpha1.Project{}
	obj.Name = projectObjName
	obj.Namespace = namespace.Name
	return obj
}

func testProjectResource() *applyconfigv1alpha1.ProjectResourceSpecApplyConfiguration {
	return applyconfigv1alpha1.ProjectResourceSpec()
}

func baseProjectPatch(project client.Object) *applyconfigv1alpha1.ProjectApplyConfiguration {
	return applyconfigv1alpha1.Project(project.GetName(), project.GetNamespace()).
		WithSpec(applyconfigv1alpha1.ProjectSpec().
			WithCloudCredentialsRef(testCredentials()))
}

func testProjectImport() *applyconfigv1alpha1.ProjectImportApplyConfiguration {
	return applyconfigv1alpha1.ProjectImport().WithID(projectID)
}

var _ = Describe("ORC Project API validations", func() {
	var namespace *corev1.Namespace
	BeforeEach(func() {
		namespace = createNamespace()
	})

	runManagementPolicyTests(func() *corev1.Namespace { return namespace }, managementPolicyTestArgs[*applyconfigv1alpha1.ProjectApplyConfiguration]{
		createObject:  func(ns *corev1.Namespace) client.Object { return projectStub(ns) },
		basePatch:     func(obj client.Object) *applyconfigv1alpha1.ProjectApplyConfiguration { return baseProjectPatch(obj) },
		applyResource: func(p *applyconfigv1alpha1.ProjectApplyConfiguration) { p.Spec.WithResource(testProjectResource()) },
		applyImport:   func(p *applyconfigv1alpha1.ProjectApplyConfiguration) { p.Spec.WithImport(testProjectImport()) },
		applyEmptyImport: func(p *applyconfigv1alpha1.ProjectApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.ProjectImport())
		},
		applyEmptyFilter: func(p *applyconfigv1alpha1.ProjectApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.ProjectImport().WithFilter(applyconfigv1alpha1.ProjectFilter()))
		},
		applyValidFilter: func(p *applyconfigv1alpha1.ProjectApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.ProjectImport().WithFilter(applyconfigv1alpha1.ProjectFilter().WithName("foo")))
		},
		applyManaged: func(p *applyconfigv1alpha1.ProjectApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged)
		},
		applyUnmanaged: func(p *applyconfigv1alpha1.ProjectApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged)
		},
		applyManagedOptions: func(p *applyconfigv1alpha1.ProjectApplyConfiguration) {
			p.Spec.WithManagedOptions(applyconfigv1alpha1.ManagedOptions().WithOnDelete(orcv1alpha1.OnDeleteDetach))
		},
		getManagementPolicy: func(obj client.Object) orcv1alpha1.ManagementPolicy {
			return obj.(*orcv1alpha1.Project).Spec.ManagementPolicy
		},
		getOnDelete: func(obj client.Object) orcv1alpha1.OnDelete {
			return obj.(*orcv1alpha1.Project).Spec.ManagedOptions.OnDelete
		},
	})

	It("should reject duplicate tags", func(ctx context.Context) {
		project := projectStub(namespace)
		patch := baseProjectPatch(project)
		patch.Spec.WithResource(applyconfigv1alpha1.ProjectResourceSpec().
			WithTags("foo", "bar", "foo"))
		Expect(applyObj(ctx, project, patch)).NotTo(Succeed())
	})

	It("should permit unique tags", func(ctx context.Context) {
		project := projectStub(namespace)
		patch := baseProjectPatch(project)
		patch.Spec.WithResource(applyconfigv1alpha1.ProjectResourceSpec().
			WithTags("foo", "bar"))
		Expect(applyObj(ctx, project, patch)).To(Succeed())
	})

	It("should have immutable domainRef", func(ctx context.Context) {
		project := projectStub(namespace)
		patch := baseProjectPatch(project)
		patch.Spec.WithResource(applyconfigv1alpha1.ProjectResourceSpec().
			WithDomainRef("domain-a"))
		Expect(applyObj(ctx, project, patch)).To(Succeed())

		patch.Spec.WithResource(applyconfigv1alpha1.ProjectResourceSpec().
			WithDomainRef("domain-b"))
		Expect(applyObj(ctx, project, patch)).To(MatchError(ContainSubstring("domainRef is immutable")))
	})
})
