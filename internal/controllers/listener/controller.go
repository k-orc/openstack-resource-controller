/*
Copyright 2025 The ORC Authors.

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

package listener

import (
	"context"
	"errors"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"

	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/reconciler"
	"github.com/k-orc/openstack-resource-controller/v2/internal/scope"
	"github.com/k-orc/openstack-resource-controller/v2/internal/util/credentials"
	"github.com/k-orc/openstack-resource-controller/v2/internal/util/dependency"
	"github.com/k-orc/openstack-resource-controller/v2/pkg/predicates"
)

const controllerName = "listener"

// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=listeners,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=listeners/status,verbs=get;update;patch

type listenerReconcilerConstructor struct {
	scopeFactory scope.Factory
}

func New(scopeFactory scope.Factory) interfaces.Controller {
	return listenerReconcilerConstructor{scopeFactory: scopeFactory}
}

func (listenerReconcilerConstructor) GetName() string {
	return controllerName
}

var loadBalancerDependency = dependency.NewDeletionGuardDependency[*orcv1alpha1.ListenerList, *orcv1alpha1.LoadBalancer](
	"spec.resource.loadBalancerRef",
	func(listener *orcv1alpha1.Listener) []string {
		resource := listener.Spec.Resource
		if resource == nil {
			return nil
		}
		return []string{string(resource.LoadBalancerRef)}
	},
	finalizer, externalObjectFieldOwner,
)

var poolDependency = dependency.NewDeletionGuardDependency[*orcv1alpha1.ListenerList, *orcv1alpha1.Pool](
	"spec.resource.poolRef",
	func(listener *orcv1alpha1.Listener) []string {
		resource := listener.Spec.Resource
		if resource == nil || resource.PoolRef == nil {
			return nil
		}
		return []string{string(*resource.PoolRef)}
	},
	finalizer, externalObjectFieldOwner,
)

var loadBalancerImportDependency = dependency.NewDependency[*orcv1alpha1.ListenerList, *orcv1alpha1.LoadBalancer](
	"spec.import.filter.loadBalancerRef",
	func(listener *orcv1alpha1.Listener) []string {
		resource := listener.Spec.Import
		if resource == nil || resource.Filter == nil || resource.Filter.LoadBalancerRef == nil {
			return nil
		}
		return []string{string(*resource.Filter.LoadBalancerRef)}
	},
)

// SetupWithManager sets up the controller with the Manager.
func (c listenerReconcilerConstructor) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	log := ctrl.LoggerFrom(ctx)
	k8sClient := mgr.GetClient()

	loadBalancerWatchEventHandler, err := loadBalancerDependency.WatchEventHandler(log, k8sClient)
	if err != nil {
		return err
	}

	poolWatchEventHandler, err := poolDependency.WatchEventHandler(log, k8sClient)
	if err != nil {
		return err
	}

	loadBalancerImportWatchEventHandler, err := loadBalancerImportDependency.WatchEventHandler(log, k8sClient)
	if err != nil {
		return err
	}

	builder := ctrl.NewControllerManagedBy(mgr).
		WithOptions(options).
		Watches(&orcv1alpha1.LoadBalancer{}, loadBalancerWatchEventHandler,
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.LoadBalancer{})),
		).
		Watches(&orcv1alpha1.Pool{}, poolWatchEventHandler,
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.Pool{})),
		).
		// A second watch is necessary because we need a different handler that omits deletion guards
		Watches(&orcv1alpha1.LoadBalancer{}, loadBalancerImportWatchEventHandler,
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.LoadBalancer{})),
		).
		For(&orcv1alpha1.Listener{})

	if err := errors.Join(
		loadBalancerDependency.AddToManager(ctx, mgr),
		poolDependency.AddToManager(ctx, mgr),
		loadBalancerImportDependency.AddToManager(ctx, mgr),
		credentialsDependency.AddToManager(ctx, mgr),
		credentials.AddCredentialsWatch(log, mgr.GetClient(), builder, credentialsDependency),
	); err != nil {
		return err
	}

	r := reconciler.NewController(controllerName, mgr.GetClient(), c.scopeFactory, listenerHelperFactory{}, listenerStatusWriter{})
	return builder.Complete(&r)
}
