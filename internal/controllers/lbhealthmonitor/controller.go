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

package lbhealthmonitor

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

const controllerName = "lbhealthmonitor"

// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=lbhealthmonitors,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=lbhealthmonitors/status,verbs=get;update;patch

type lbhealthmonitorReconcilerConstructor struct {
	scopeFactory        scope.Factory
	defaultResyncPeriod time.Duration
}

func New(scopeFactory scope.Factory) interfaces.Controller {
	return &lbhealthmonitorReconcilerConstructor{scopeFactory: scopeFactory}
}

func (lbhealthmonitorReconcilerConstructor) GetName() string {
	return controllerName
}

func (c *lbhealthmonitorReconcilerConstructor) SetDefaultResyncPeriod(d time.Duration) {
	c.defaultResyncPeriod = d
}

var poolDependency = dependency.NewDeletionGuardDependency[*orcv1alpha1.LBHealthMonitorList, *orcv1alpha1.LBPool](
	"spec.resource.poolRef",
	func(lbhealthmonitor *orcv1alpha1.LBHealthMonitor) []string {
		resource := lbhealthmonitor.Spec.Resource
		if resource == nil {
			return nil
		}
		return []string{string(resource.PoolRef)}
	},
	finalizer, externalObjectFieldOwner,
)

var poolImportDependency = dependency.NewDependency[*orcv1alpha1.LBHealthMonitorList, *orcv1alpha1.LBPool](
	"spec.import.filter.poolRef",
	func(lbhealthmonitor *orcv1alpha1.LBHealthMonitor) []string {
		resource := lbhealthmonitor.Spec.Import
		if resource == nil || resource.Filter == nil || resource.Filter.PoolRef == nil {
			return nil
		}
		return []string{string(*resource.Filter.PoolRef)}
	},
)

// SetupWithManager sets up the controller with the Manager.
func (c *lbhealthmonitorReconcilerConstructor) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	log := ctrl.LoggerFrom(ctx)
	k8sClient := mgr.GetClient()

	poolWatchEventHandler, err := poolDependency.WatchEventHandler(log, k8sClient)
	if err != nil {
		return err
	}

	poolImportWatchEventHandler, err := poolImportDependency.WatchEventHandler(log, k8sClient)
	if err != nil {
		return err
	}

	builder := ctrl.NewControllerManagedBy(mgr).
		WithOptions(options).
		Watches(&orcv1alpha1.LBPool{}, poolWatchEventHandler,
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.LBPool{})),
		).
		// A second watch is necessary because we need a different handler that omits deletion guards
		Watches(&orcv1alpha1.LBPool{}, poolImportWatchEventHandler,
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.LBPool{})),
		).
		For(&orcv1alpha1.LBHealthMonitor{})

	if err := errors.Join(
		poolDependency.AddToManager(ctx, mgr),
		poolImportDependency.AddToManager(ctx, mgr),
		credentialsDependency.AddToManager(ctx, mgr),
		credentials.AddCredentialsWatch(log, mgr.GetClient(), builder, credentialsDependency),
	); err != nil {
		return err
	}

	r := reconciler.NewController(controllerName, mgr.GetClient(), c.scopeFactory, lbhealthmonitorHelperFactory{}, lbhealthmonitorStatusWriter{}, c.defaultResyncPeriod)
	return builder.Complete(&r)
}
