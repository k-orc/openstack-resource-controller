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
	serverName = "server"
	serverID   = "265c9e4f-0f5a-46e4-9f3f-fb8de25ae134"
)

func serverStub(namespace *corev1.Namespace) *orcv1alpha1.Server {
	obj := &orcv1alpha1.Server{}
	obj.Name = serverName
	obj.Namespace = namespace.Name
	return obj
}

func testServerResource() *applyconfigv1alpha1.ServerResourceSpecApplyConfiguration {
	return applyconfigv1alpha1.ServerResourceSpec().
		WithImageRef("my-image").
		WithFlavorRef("my-flavor").
		WithPorts(applyconfigv1alpha1.ServerPortSpec().WithPortRef("my-port"))
}

func baseServerPatch(server client.Object) *applyconfigv1alpha1.ServerApplyConfiguration {
	return applyconfigv1alpha1.Server(server.GetName(), server.GetNamespace()).
		WithSpec(applyconfigv1alpha1.ServerSpec().
			WithCloudCredentialsRef(testCredentials()))
}

func testServerImport() *applyconfigv1alpha1.ServerImportApplyConfiguration {
	return applyconfigv1alpha1.ServerImport().WithID(serverID)
}

var _ = Describe("ORC Server API validations", func() {
	var namespace *corev1.Namespace
	BeforeEach(func() {
		namespace = createNamespace()
	})

	runManagementPolicyTests(func() *corev1.Namespace { return namespace }, managementPolicyTestArgs[*applyconfigv1alpha1.ServerApplyConfiguration]{
		createObject: func(ns *corev1.Namespace) client.Object { return serverStub(ns) },
		basePatch: func(obj client.Object) *applyconfigv1alpha1.ServerApplyConfiguration {
			return baseServerPatch(obj)
		},
		applyResource: func(p *applyconfigv1alpha1.ServerApplyConfiguration) {
			p.Spec.WithResource(testServerResource())
		},
		applyImport: func(p *applyconfigv1alpha1.ServerApplyConfiguration) {
			p.Spec.WithImport(testServerImport())
		},
		applyEmptyImport: func(p *applyconfigv1alpha1.ServerApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.ServerImport())
		},
		applyEmptyFilter: func(p *applyconfigv1alpha1.ServerApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.ServerImport().WithFilter(applyconfigv1alpha1.ServerFilter()))
		},
		applyValidFilter: func(p *applyconfigv1alpha1.ServerApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.ServerImport().WithFilter(applyconfigv1alpha1.ServerFilter().WithName("foo")))
		},
		applyManaged: func(p *applyconfigv1alpha1.ServerApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged)
		},
		applyUnmanaged: func(p *applyconfigv1alpha1.ServerApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged)
		},
		applyManagedOptions: func(p *applyconfigv1alpha1.ServerApplyConfiguration) {
			p.Spec.WithManagedOptions(applyconfigv1alpha1.ManagedOptions().WithOnDelete(orcv1alpha1.OnDeleteDetach))
		},
		getManagementPolicy: func(obj client.Object) orcv1alpha1.ManagementPolicy {
			return obj.(*orcv1alpha1.Server).Spec.ManagementPolicy
		},
		getOnDelete: func(obj client.Object) orcv1alpha1.OnDelete {
			return obj.(*orcv1alpha1.Server).Spec.ManagedOptions.OnDelete
		},
	})

	It("should reject a server without required fields", func(ctx context.Context) {
		server := serverStub(namespace)
		patch := baseServerPatch(server)
		patch.Spec.WithResource(applyconfigv1alpha1.ServerResourceSpec())
		Expect(applyObj(ctx, server, patch)).NotTo(Succeed())

		// Missing flavorRef
		patch.Spec.WithResource(applyconfigv1alpha1.ServerResourceSpec().
			WithImageRef("my-image").
			WithPorts(applyconfigv1alpha1.ServerPortSpec().WithPortRef("my-port")))
		Expect(applyObj(ctx, server, patch)).To(MatchError(ContainSubstring("spec.resource.flavorRef")))

		// Missing imageRef or bootVolume
		patch.Spec.WithResource(applyconfigv1alpha1.ServerResourceSpec().
			WithFlavorRef("my-flavor").
			WithPorts(applyconfigv1alpha1.ServerPortSpec().WithPortRef("my-port")))
		Expect(applyObj(ctx, server, patch)).To(MatchError(ContainSubstring("either imageRef or bootVolume must be specified")))

		// Missing ports
		patch.Spec.WithResource(applyconfigv1alpha1.ServerResourceSpec().
			WithImageRef("my-image").
			WithFlavorRef("my-flavor"))
		Expect(applyObj(ctx, server, patch)).To(MatchError(ContainSubstring("spec.resource.ports")))
	})

	It("should have immutable imageRef", func(ctx context.Context) {
		server := serverStub(namespace)
		patch := baseServerPatch(server)
		patch.Spec.WithResource(applyconfigv1alpha1.ServerResourceSpec().
			WithImageRef("image-a").
			WithFlavorRef("my-flavor").
			WithPorts(applyconfigv1alpha1.ServerPortSpec().WithPortRef("my-port")))
		Expect(applyObj(ctx, server, patch)).To(Succeed())

		patch.Spec.WithResource(applyconfigv1alpha1.ServerResourceSpec().
			WithImageRef("image-b").
			WithFlavorRef("my-flavor").
			WithPorts(applyconfigv1alpha1.ServerPortSpec().WithPortRef("my-port")))
		Expect(applyObj(ctx, server, patch)).To(MatchError(ContainSubstring("imageRef is immutable")))
	})

	It("should have immutable flavorRef", func(ctx context.Context) {
		server := serverStub(namespace)
		patch := baseServerPatch(server)
		patch.Spec.WithResource(applyconfigv1alpha1.ServerResourceSpec().
			WithImageRef("my-image").
			WithFlavorRef("flavor-a").
			WithPorts(applyconfigv1alpha1.ServerPortSpec().WithPortRef("my-port")))
		Expect(applyObj(ctx, server, patch)).To(Succeed())

		patch.Spec.WithResource(applyconfigv1alpha1.ServerResourceSpec().
			WithImageRef("my-image").
			WithFlavorRef("flavor-b").
			WithPorts(applyconfigv1alpha1.ServerPortSpec().WithPortRef("my-port")))
		Expect(applyObj(ctx, server, patch)).To(MatchError(ContainSubstring("flavorRef is immutable")))
	})

	It("should have immutable schedulerHints", func(ctx context.Context) {
		server := serverStub(namespace)
		patch := baseServerPatch(server)
		patch.Spec.WithResource(applyconfigv1alpha1.ServerResourceSpec().
			WithImageRef("my-image").
			WithFlavorRef("my-flavor").
			WithPorts(applyconfigv1alpha1.ServerPortSpec().WithPortRef("my-port")).
			WithSchedulerHints(applyconfigv1alpha1.ServerSchedulerHints().WithServerGroupRef("sg-a")))
		Expect(applyObj(ctx, server, patch)).To(Succeed())

		patch.Spec.WithResource(applyconfigv1alpha1.ServerResourceSpec().
			WithImageRef("my-image").
			WithFlavorRef("my-flavor").
			WithPorts(applyconfigv1alpha1.ServerPortSpec().WithPortRef("my-port")).
			WithSchedulerHints(applyconfigv1alpha1.ServerSchedulerHints().WithServerGroupRef("sg-b")))
		Expect(applyObj(ctx, server, patch)).To(MatchError(ContainSubstring("schedulerHints is immutable")))
	})

	It("should have immutable keypairRef", func(ctx context.Context) {
		server := serverStub(namespace)
		patch := baseServerPatch(server)
		patch.Spec.WithResource(applyconfigv1alpha1.ServerResourceSpec().
			WithImageRef("my-image").
			WithFlavorRef("my-flavor").
			WithPorts(applyconfigv1alpha1.ServerPortSpec().WithPortRef("my-port")).
			WithKeypairRef("kp-a"))
		Expect(applyObj(ctx, server, patch)).To(Succeed())

		patch.Spec.WithResource(applyconfigv1alpha1.ServerResourceSpec().
			WithImageRef("my-image").
			WithFlavorRef("my-flavor").
			WithPorts(applyconfigv1alpha1.ServerPortSpec().WithPortRef("my-port")).
			WithKeypairRef("kp-b"))
		Expect(applyObj(ctx, server, patch)).To(MatchError(ContainSubstring("keypairRef is immutable")))
	})

	It("should have immutable configDrive", func(ctx context.Context) {
		server := serverStub(namespace)
		patch := baseServerPatch(server)
		patch.Spec.WithResource(applyconfigv1alpha1.ServerResourceSpec().
			WithImageRef("my-image").
			WithFlavorRef("my-flavor").
			WithPorts(applyconfigv1alpha1.ServerPortSpec().WithPortRef("my-port")).
			WithConfigDrive(true))
		Expect(applyObj(ctx, server, patch)).To(Succeed())

		patch.Spec.WithResource(applyconfigv1alpha1.ServerResourceSpec().
			WithImageRef("my-image").
			WithFlavorRef("my-flavor").
			WithPorts(applyconfigv1alpha1.ServerPortSpec().WithPortRef("my-port")).
			WithConfigDrive(false))
		Expect(applyObj(ctx, server, patch)).To(MatchError(ContainSubstring("configDrive is immutable")))
	})

	It("should reject duplicate tags", func(ctx context.Context) {
		server := serverStub(namespace)
		patch := baseServerPatch(server)
		patch.Spec.WithResource(applyconfigv1alpha1.ServerResourceSpec().
			WithImageRef("my-image").
			WithFlavorRef("my-flavor").
			WithPorts(applyconfigv1alpha1.ServerPortSpec().WithPortRef("my-port")).
			WithTags("foo", "bar", "foo"))
		Expect(applyObj(ctx, server, patch)).NotTo(Succeed())
	})
})
