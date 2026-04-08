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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	applyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/pkg/clients/applyconfiguration/api/v1alpha1"
)

func testCredentials() *applyconfigv1alpha1.CloudCredentialsReferenceApplyConfiguration {
	return applyconfigv1alpha1.CloudCredentialsReference().
		WithSecretName("openstack-credentials").
		WithCloudName("openstack")
}

// managementPolicyTestArgs provides resource-specific callbacks for the shared
// management policy validation tests. PatchT is the concrete apply
// configuration type for the resource (e.g. *applyconfigv1alpha1.FlavorApplyConfiguration).
type managementPolicyTestArgs[PatchT any] struct {
	// createObject returns a new stub object in the given namespace.
	createObject func(*corev1.Namespace) client.Object
	// basePatch returns a patch with only cloudCredentialsRef set.
	basePatch func(client.Object) PatchT
	// applyResource adds a valid resource spec to the patch.
	applyResource func(PatchT)
	// applyImport adds a valid import (by ID) to the patch.
	applyImport func(PatchT)
	// applyEmptyImport adds an empty import to the patch.
	applyEmptyImport func(PatchT)
	// applyEmptyFilter adds an import with an empty filter to the patch.
	applyEmptyFilter func(PatchT)
	// applyValidFilter adds an import with a valid filter to the patch.
	applyValidFilter func(PatchT)
	// applyManaged sets the management policy to managed.
	applyManaged func(PatchT)
	// applyUnmanaged sets the management policy to unmanaged.
	applyUnmanaged func(PatchT)
	// applyManagedOptions adds managedOptions to the patch.
	applyManagedOptions func(PatchT)
	// getManagementPolicy reads the management policy from the object.
	getManagementPolicy func(client.Object) orcv1alpha1.ManagementPolicy
	// getOnDelete reads the onDelete value from the object's managedOptions.
	getOnDelete func(client.Object) orcv1alpha1.OnDelete
}

// runManagementPolicyTests registers shared Ginkgo test cases for the standard
// management policy validations that apply to all ORC resources with a
// managementPolicy field.
func runManagementPolicyTests[PatchT any](getNamespace func() *corev1.Namespace, args managementPolicyTestArgs[PatchT]) {
	It("should allow to create a minimal resource and managementPolicy should default to managed", func(ctx context.Context) {
		obj := args.createObject(getNamespace())
		patch := args.basePatch(obj)
		args.applyResource(patch)
		Expect(applyObj(ctx, obj, patch)).To(Succeed())
		Expect(args.getManagementPolicy(obj)).To(Equal(orcv1alpha1.ManagementPolicyManaged))
	})

	It("should require import for unmanaged", func(ctx context.Context) {
		obj := args.createObject(getNamespace())
		patch := args.basePatch(obj)
		args.applyUnmanaged(patch)
		Expect(applyObj(ctx, obj, patch)).To(MatchError(ContainSubstring("import must be specified when policy is unmanaged")))

		args.applyImport(patch)
		Expect(applyObj(ctx, obj, patch)).To(Succeed())
	})

	It("should not permit unmanaged with resource", func(ctx context.Context) {
		obj := args.createObject(getNamespace())
		patch := args.basePatch(obj)
		args.applyUnmanaged(patch)
		args.applyImport(patch)
		args.applyResource(patch)
		Expect(applyObj(ctx, obj, patch)).To(MatchError(ContainSubstring("resource may not be specified when policy is unmanaged")))
	})

	It("should not permit empty import", func(ctx context.Context) {
		obj := args.createObject(getNamespace())
		patch := args.basePatch(obj)
		args.applyUnmanaged(patch)
		args.applyEmptyImport(patch)
		Expect(applyObj(ctx, obj, patch)).To(MatchError(ContainSubstring("spec.import in body should have at least 1 properties")))
	})

	It("should not permit empty import filter", func(ctx context.Context) {
		obj := args.createObject(getNamespace())
		patch := args.basePatch(obj)
		args.applyUnmanaged(patch)
		args.applyEmptyFilter(patch)
		// Do not force the maximum number of filter properties to be 1 by not hard-coding that string
		Expect(applyObj(ctx, obj, patch)).To(MatchError(ContainSubstring("spec.import.filter in body should have at least")))
	})

	It("should permit valid import filter", func(ctx context.Context) {
		obj := args.createObject(getNamespace())
		patch := args.basePatch(obj)
		args.applyUnmanaged(patch)
		args.applyValidFilter(patch)
		Expect(applyObj(ctx, obj, patch)).To(Succeed())
	})

	It("should require resource for managed", func(ctx context.Context) {
		obj := args.createObject(getNamespace())
		patch := args.basePatch(obj)
		args.applyManaged(patch)
		Expect(applyObj(ctx, obj, patch)).To(MatchError(ContainSubstring("resource must be specified when policy is managed")))

		args.applyResource(patch)
		Expect(applyObj(ctx, obj, patch)).To(Succeed())
	})

	It("should not permit managed with import", func(ctx context.Context) {
		obj := args.createObject(getNamespace())
		patch := args.basePatch(obj)
		args.applyImport(patch)
		args.applyManaged(patch)
		args.applyResource(patch)
		Expect(applyObj(ctx, obj, patch)).To(MatchError(ContainSubstring("import may not be specified when policy is managed")))
	})

	It("should not permit managedOptions for unmanaged", func(ctx context.Context) {
		obj := args.createObject(getNamespace())
		patch := args.basePatch(obj)
		args.applyImport(patch)
		args.applyUnmanaged(patch)
		args.applyManagedOptions(patch)
		Expect(applyObj(ctx, obj, patch)).To(MatchError(ContainSubstring("managedOptions may only be provided when policy is managed")))
	})

	It("should permit managedOptions for managed", func(ctx context.Context) {
		obj := args.createObject(getNamespace())
		patch := args.basePatch(obj)
		args.applyResource(patch)
		args.applyManagedOptions(patch)
		Expect(applyObj(ctx, obj, patch)).To(Succeed())
		Expect(args.getOnDelete(obj)).To(Equal(orcv1alpha1.OnDelete("detach")))
	})
}
