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
	addressScopeObjName = "addressscope"
)

func addressScopeStub(namespace *corev1.Namespace) *orcv1alpha1.AddressScope {
	obj := &orcv1alpha1.AddressScope{}
	obj.Name = addressScopeObjName
	obj.Namespace = namespace.Name
	return obj
}

func baseAddressScopePatch(addressScope client.Object) *applyconfigv1alpha1.AddressScopeApplyConfiguration {
	return applyconfigv1alpha1.AddressScope(addressScope.GetName(), addressScope.GetNamespace()).
		WithSpec(applyconfigv1alpha1.AddressScopeSpec().
			WithCloudCredentialsRef(testCredentials()))
}

var _ = Describe("ORC AddressScope API validations", func() {
	var namespace *corev1.Namespace
	BeforeEach(func() {
		namespace = createNamespace()
	})

	When("updating the shared field", func() {
		It("should permit share a unshared address scope", func(ctx context.Context) {
			addressScope := addressScopeStub(namespace)
			patch := baseAddressScopePatch(addressScope)
			patch.Spec.WithResource(applyconfigv1alpha1.AddressScopeResourceSpec().
				WithIPVersion(orcv1alpha1.IPVersion(4)).
				WithShared(false))
			Expect(applyObj(ctx, addressScope, patch)).To(Succeed())
			patch.Spec.WithResource(patch.Spec.Resource).Resource.WithShared(true)
			Expect(applyObj(ctx, addressScope, patch)).To(Succeed())
		})

		It("should not permit unshare a shared address scope", func(ctx context.Context) {
			addressScope := addressScopeStub(namespace)
			patch := baseAddressScopePatch(addressScope)
			patch.Spec.WithResource(applyconfigv1alpha1.AddressScopeResourceSpec().
				WithIPVersion(orcv1alpha1.IPVersion(4)).
				WithShared(true))
			Expect(applyObj(ctx, addressScope, patch)).To(Succeed())
			patch.Spec.WithResource(patch.Spec.Resource).Resource.WithShared(false)
			Expect(applyObj(ctx, addressScope, patch)).To(MatchError(ContainSubstring("shared address scope can't be unshared")))
		})
	})
})
