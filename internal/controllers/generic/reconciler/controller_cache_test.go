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

package reconciler

import (
	"context"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v3/internal/controllers/generic/interfaces"
	orcapplyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/pkg/clients/applyconfiguration/api/v1alpha1"
)

// flavorAdapterFactory only implements NewAPIObjectAdapter.
type flavorAdapterFactory struct {
	interfaces.ResourceHelperFactory[*orcv1alpha1.Flavor, orcv1alpha1.Flavor, orcv1alpha1.FlavorResourceSpec, orcv1alpha1.FlavorFilter, fakeOSResource]
}

func (flavorAdapterFactory) NewAPIObjectAdapter(obj *orcv1alpha1.Flavor) interfaces.APIObjectAdapter[*orcv1alpha1.Flavor, orcv1alpha1.FlavorResourceSpec, orcv1alpha1.FlavorFilter] {
	return fakeAdapter{Flavor: obj}
}

func newCacheTestController(t *testing.T, objs ...client.Object) *Controller[
	*orcv1alpha1.Flavor, orcv1alpha1.Flavor,
	orcv1alpha1.FlavorResourceSpec, orcv1alpha1.FlavorFilter,
	*orcapplyconfigv1alpha1.FlavorApplyConfiguration,
	*orcapplyconfigv1alpha1.FlavorStatusApplyConfiguration, orcapplyconfigv1alpha1.FlavorStatusApplyConfiguration,
	fakeOSResource,
] {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := orcv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add orcv1alpha1 to scheme: %v", err)
	}
	return &Controller[
		*orcv1alpha1.Flavor, orcv1alpha1.Flavor,
		orcv1alpha1.FlavorResourceSpec, orcv1alpha1.FlavorFilter,
		*orcapplyconfigv1alpha1.FlavorApplyConfiguration,
		*orcapplyconfigv1alpha1.FlavorStatusApplyConfiguration, orcapplyconfigv1alpha1.FlavorStatusApplyConfiguration,
		fakeOSResource,
	]{
		client:        fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build(),
		helperFactory: flavorAdapterFactory{},
	}
}

func TestWaitForCachedStatusID(t *testing.T) {
	t.Parallel()

	key := client.ObjectKey{Namespace: "default", Name: "test-flavor"}
	flavor := func(id *string) *orcv1alpha1.Flavor {
		return &orcv1alpha1.Flavor{
			ObjectMeta: metav1.ObjectMeta{Name: key.Name, Namespace: key.Namespace},
			Status:     orcv1alpha1.FlavorStatus{ID: id},
		}
	}

	t.Run("status ID in cache", func(t *testing.T) {
		t.Parallel()
		c := newCacheTestController(t, flavor(ptr.To("flavor-id")))
		if err := c.waitForCachedStatusID(context.Background(), key); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("object deleted", func(t *testing.T) {
		t.Parallel()
		c := newCacheTestController(t)
		if err := c.waitForCachedStatusID(context.Background(), key); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("status ID not in cache", func(t *testing.T) {
		t.Parallel()
		c := newCacheTestController(t, flavor(nil))
		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer cancel()
		if err := c.waitForCachedStatusID(ctx, key); err == nil {
			t.Error("expected an error when the status ID never appears in the cache")
		}
	})
}
