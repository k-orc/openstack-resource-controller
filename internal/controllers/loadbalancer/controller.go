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

package loadbalancer

import (
	"context"
	"errors"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v2/pkg/predicates"

	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/reconciler"
	"github.com/k-orc/openstack-resource-controller/v2/internal/scope"
	"github.com/k-orc/openstack-resource-controller/v2/internal/util/credentials"
)

// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=loadbalancers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=loadbalancers/status,verbs=get;update;patch

type loadbalancerReconcilerConstructor struct {
	scopeFactory scope.Factory
}

func New(scopeFactory scope.Factory) interfaces.Controller {
	return loadbalancerReconcilerConstructor{scopeFactory: scopeFactory}
}

func (loadbalancerReconcilerConstructor) GetName() string {
	return controllerName
}

// SetupWithManager sets up the controller with the Manager.
func (c loadbalancerReconcilerConstructor) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	log := mgr.GetLogger().WithValues("controller", controllerName)
	k8sClient := mgr.GetClient()

	vipSubnetWatchEventHandler, err := subnetDependency.WatchEventHandler(log, k8sClient)
	if err != nil {
		return err
	}

	vipNetworkWatchEventHandler, err := networkDependency.WatchEventHandler(log, k8sClient)
	if err != nil {
		return err
	}

	vipPortWatchEventHandler, err := portDependency.WatchEventHandler(log, k8sClient)
	if err != nil {
		return err
	}

	projectWatchEventHandler, err := projectDependency.WatchEventHandler(log, k8sClient)
	if err != nil {
		return err
	}

	subnetImportWatchEventHandler, err := subnetImportDependency.WatchEventHandler(log, k8sClient)
	if err != nil {
		return err
	}

	networkImportWatchEventHandler, err := networkImportDependency.WatchEventHandler(log, k8sClient)
	if err != nil {
		return err
	}

	projectImportWatchEventHandler, err := projectImportDependency.WatchEventHandler(log, k8sClient)
	if err != nil {
		return err
	}

	builder := ctrl.NewControllerManagedBy(mgr).
		WithOptions(options).
		For(&orcv1alpha1.LoadBalancer{}).
		Watches(&orcv1alpha1.Subnet{}, vipSubnetWatchEventHandler,
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.Subnet{})),
		).
		// A second watch is necessary because we need a different handler that omits deletion guards
		Watches(&orcv1alpha1.Subnet{}, subnetImportWatchEventHandler,
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.Subnet{})),
		).
		Watches(&orcv1alpha1.Network{}, vipNetworkWatchEventHandler,
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.Network{})),
		).
		// A second watch is necessary because we need a different handler that omits deletion guards
		Watches(&orcv1alpha1.Network{}, networkImportWatchEventHandler,
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.Network{})),
		).
		Watches(&orcv1alpha1.Port{}, vipPortWatchEventHandler,
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.Port{})),
		).
		Watches(&orcv1alpha1.Project{}, projectWatchEventHandler,
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.Project{})),
		).
		// A second watch is necessary because we need a different handler that omits deletion guards
		Watches(&orcv1alpha1.Project{}, projectImportWatchEventHandler,
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.Project{})),
		)

	if err := errors.Join(
		subnetDependency.AddToManager(ctx, mgr),
		subnetImportDependency.AddToManager(ctx, mgr),
		networkDependency.AddToManager(ctx, mgr),
		networkImportDependency.AddToManager(ctx, mgr),
		portDependency.AddToManager(ctx, mgr),
		projectDependency.AddToManager(ctx, mgr),
		projectImportDependency.AddToManager(ctx, mgr),
		credentialsDependency.AddToManager(ctx, mgr),
		credentials.AddCredentialsWatch(log, k8sClient, builder, credentialsDependency),
	); err != nil {
		return err
	}

	r := reconciler.NewController(controllerName, k8sClient, c.scopeFactory, loadbalancerHelperFactory{}, loadbalancerStatusWriter{})
	return builder.Complete(&r)
}
