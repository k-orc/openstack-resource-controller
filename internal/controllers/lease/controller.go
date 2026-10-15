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

package lease

import (
	"context"
	"errors"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/api/v1alpha1"

	"github.com/k-orc/openstack-resource-controller/v3/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v3/internal/controllers/generic/reconciler"
	"github.com/k-orc/openstack-resource-controller/v3/internal/scope"
	"github.com/k-orc/openstack-resource-controller/v3/internal/util/credentials"
	"github.com/k-orc/openstack-resource-controller/v3/internal/util/dependency"
	"github.com/k-orc/openstack-resource-controller/v3/pkg/predicates"
)

const controllerName = "lease"

// No deletion guard for flavor, because Blazar copies the flavor when the
// reservation is created, so it can be safely deleted while referenced by a
// lease
var flavorDependency = dependency.NewDependency[*orcv1alpha1.LeaseList, *orcv1alpha1.Flavor](
	"spec.resource.reservations.flavorInstance.flavorRef",
	func(lease *orcv1alpha1.Lease) []string {
		resource := lease.Spec.Resource
		if resource == nil {
			return nil
		}

		var refs []string
		for i := range resource.Reservations {
			if flavorInstance := resource.Reservations[i].FlavorInstance; flavorInstance != nil {
				refs = append(refs, string(flavorInstance.FlavorRef))
			}
		}
		return refs
	},
)

// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=leases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=leases/status,verbs=get;update;patch

type leaseReconcilerConstructor struct {
	scopeFactory        scope.Factory
	defaultResyncPeriod time.Duration
}

func New(scopeFactory scope.Factory) interfaces.Controller {
	return &leaseReconcilerConstructor{scopeFactory: scopeFactory}
}

func (leaseReconcilerConstructor) GetName() string {
	return controllerName
}

func (c *leaseReconcilerConstructor) SetDefaultResyncPeriod(d time.Duration) {
	c.defaultResyncPeriod = d
}

// SetupWithManager sets up the controller with the Manager.
func (c *leaseReconcilerConstructor) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	log := ctrl.LoggerFrom(ctx)
	k8sClient := mgr.GetClient()

	flavorWatchEventHandler, err := flavorDependency.WatchEventHandler(log, k8sClient)
	if err != nil {
		return err
	}

	controllerBuilder := ctrl.NewControllerManagedBy(mgr).
		WithOptions(options).
		For(&orcv1alpha1.Lease{}).
		Watches(&orcv1alpha1.Flavor{}, flavorWatchEventHandler,
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.Flavor{})),
		)

	if err := errors.Join(
		flavorDependency.AddToManager(ctx, mgr),
		credentialsDependency.AddToManager(ctx, mgr),
		credentials.AddCredentialsWatch(log, k8sClient, controllerBuilder, credentialsDependency),
	); err != nil {
		return err
	}

	r := reconciler.NewController(controllerName, k8sClient, c.scopeFactory, leaseHelperFactory{}, leaseStatusWriter{}, c.defaultResyncPeriod)
	return controllerBuilder.Complete(&r)
}
