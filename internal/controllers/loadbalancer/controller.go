/*
Copyright 2026 The ORC Authors.

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

	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/reconciler"
	"github.com/k-orc/openstack-resource-controller/v2/internal/scope"
	"github.com/k-orc/openstack-resource-controller/v2/internal/util/credentials"
	"github.com/k-orc/openstack-resource-controller/v2/internal/util/dependency"
	"github.com/k-orc/openstack-resource-controller/v2/pkg/predicates"
)

const controllerName = "loadbalancer"

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

var subnetDependency = dependency.NewDeletionGuardDependency[*orcv1alpha1.LoadBalancerList, *orcv1alpha1.Subnet](
	"spec.resource.subnetRef",
	func(loadbalancer *orcv1alpha1.LoadBalancer) []string {
		resource := loadbalancer.Spec.Resource
		if resource == nil || resource.VipSubnetRef == nil {
			return nil
		}
		return []string{string(*resource.VipSubnetRef)}
	},
	finalizer, externalObjectFieldOwner,
)

var networkDependency = dependency.NewDeletionGuardDependency[*orcv1alpha1.LoadBalancerList, *orcv1alpha1.Network](
	"spec.resource.networkRef",
	func(loadbalancer *orcv1alpha1.LoadBalancer) []string {
		resource := loadbalancer.Spec.Resource
		if resource == nil || resource.VipNetworkRef == nil {
			return nil
		}
		return []string{string(*resource.VipNetworkRef)}
	},
	finalizer, externalObjectFieldOwner,
)

var portDependency = dependency.NewDeletionGuardDependency[*orcv1alpha1.LoadBalancerList, *orcv1alpha1.Port](
	"spec.resource.portRef",
	func(loadbalancer *orcv1alpha1.LoadBalancer) []string {
		resource := loadbalancer.Spec.Resource
		if resource == nil || resource.VipPortRef == nil {
			return nil
		}
		return []string{string(*resource.VipPortRef)}
	},
	finalizer, externalObjectFieldOwner,
)

var flavorDependency = dependency.NewDeletionGuardDependency[*orcv1alpha1.LoadBalancerList, *orcv1alpha1.Flavor](
	"spec.resource.flavorRef",
	func(loadbalancer *orcv1alpha1.LoadBalancer) []string {
		resource := loadbalancer.Spec.Resource
		if resource == nil || resource.FlavorRef == nil {
			return nil
		}
		return []string{string(*resource.FlavorRef)}
	},
	finalizer, externalObjectFieldOwner,
)

var projectDependency = dependency.NewDeletionGuardDependency[*orcv1alpha1.LoadBalancerList, *orcv1alpha1.Project](
	"spec.resource.projectRef",
	func(loadbalancer *orcv1alpha1.LoadBalancer) []string {
		resource := loadbalancer.Spec.Resource
		if resource == nil || resource.ProjectRef == nil {
			return nil
		}
		return []string{string(*resource.ProjectRef)}
	},
	finalizer, externalObjectFieldOwner,
)

var vipNetworkImportDependency = dependency.NewDependency[*orcv1alpha1.LoadBalancerList, *orcv1alpha1.Network](
	"spec.import.filter.vipNetworkRef",
	func(loadbalancer *orcv1alpha1.LoadBalancer) []string {
		resource := loadbalancer.Spec.Import
		if resource == nil || resource.Filter == nil || resource.Filter.VipNetworkRef == nil {
			return nil
		}
		return []string{string(*resource.Filter.VipNetworkRef)}
	},
)

var projectImportDependency = dependency.NewDependency[*orcv1alpha1.LoadBalancerList, *orcv1alpha1.Project](
	"spec.import.filter.projectRef",
	func(loadbalancer *orcv1alpha1.LoadBalancer) []string {
		resource := loadbalancer.Spec.Import
		if resource == nil || resource.Filter == nil || resource.Filter.ProjectRef == nil {
			return nil
		}
		return []string{string(*resource.Filter.ProjectRef)}
	},
)

var vipSubnetImportDependency = dependency.NewDependency[*orcv1alpha1.LoadBalancerList, *orcv1alpha1.Subnet](
	"spec.import.filter.vipSubnetRef",
	func(loadbalancer *orcv1alpha1.LoadBalancer) []string {
		resource := loadbalancer.Spec.Import
		if resource == nil || resource.Filter == nil || resource.Filter.VipSubnetRef == nil {
			return nil
		}
		return []string{string(*resource.Filter.VipSubnetRef)}
	},
)

var vipPortImportDependency = dependency.NewDependency[*orcv1alpha1.LoadBalancerList, *orcv1alpha1.Port](
	"spec.import.filter.vipPortRef",
	func(loadbalancer *orcv1alpha1.LoadBalancer) []string {
		resource := loadbalancer.Spec.Import
		if resource == nil || resource.Filter == nil || resource.Filter.VipPortRef == nil {
			return nil
		}
		return []string{string(*resource.Filter.VipPortRef)}
	},
)

// SetupWithManager sets up the controller with the Manager.
func (c loadbalancerReconcilerConstructor) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	log := ctrl.LoggerFrom(ctx)
	k8sClient := mgr.GetClient()

	subnetWatchEventHandler, err := subnetDependency.WatchEventHandler(log, k8sClient)
	if err != nil {
		return err
	}

	networkWatchEventHandler, err := networkDependency.WatchEventHandler(log, k8sClient)
	if err != nil {
		return err
	}

	portWatchEventHandler, err := portDependency.WatchEventHandler(log, k8sClient)
	if err != nil {
		return err
	}

	flavorWatchEventHandler, err := flavorDependency.WatchEventHandler(log, k8sClient)
	if err != nil {
		return err
	}

	projectWatchEventHandler, err := projectDependency.WatchEventHandler(log, k8sClient)
	if err != nil {
		return err
	}

	vipNetworkImportWatchEventHandler, err := vipNetworkImportDependency.WatchEventHandler(log, k8sClient)
	if err != nil {
		return err
	}

	projectImportWatchEventHandler, err := projectImportDependency.WatchEventHandler(log, k8sClient)
	if err != nil {
		return err
	}

	vipSubnetImportWatchEventHandler, err := vipSubnetImportDependency.WatchEventHandler(log, k8sClient)
	if err != nil {
		return err
	}

	vipPortImportWatchEventHandler, err := vipPortImportDependency.WatchEventHandler(log, k8sClient)
	if err != nil {
		return err
	}

	builder := ctrl.NewControllerManagedBy(mgr).
		WithOptions(options).
		Watches(&orcv1alpha1.Subnet{}, subnetWatchEventHandler,
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.Subnet{})),
		).
		Watches(&orcv1alpha1.Network{}, networkWatchEventHandler,
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.Network{})),
		).
		Watches(&orcv1alpha1.Port{}, portWatchEventHandler,
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.Port{})),
		).
		Watches(&orcv1alpha1.Flavor{}, flavorWatchEventHandler,
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.Flavor{})),
		).
		Watches(&orcv1alpha1.Project{}, projectWatchEventHandler,
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.Project{})),
		).
		// A second watch is necessary because we need a different handler that omits deletion guards
		Watches(&orcv1alpha1.Network{}, vipNetworkImportWatchEventHandler,
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.Network{})),
		).
		// A second watch is necessary because we need a different handler that omits deletion guards
		Watches(&orcv1alpha1.Project{}, projectImportWatchEventHandler,
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.Project{})),
		).
		// A second watch is necessary because we need a different handler that omits deletion guards
		Watches(&orcv1alpha1.Subnet{}, vipSubnetImportWatchEventHandler,
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.Subnet{})),
		).
		// A second watch is necessary because we need a different handler that omits deletion guards
		Watches(&orcv1alpha1.Port{}, vipPortImportWatchEventHandler,
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.Port{})),
		).
		For(&orcv1alpha1.LoadBalancer{})

	if err := errors.Join(
		subnetDependency.AddToManager(ctx, mgr),
		networkDependency.AddToManager(ctx, mgr),
		portDependency.AddToManager(ctx, mgr),
		flavorDependency.AddToManager(ctx, mgr),
		projectDependency.AddToManager(ctx, mgr),
		vipNetworkImportDependency.AddToManager(ctx, mgr),
		projectImportDependency.AddToManager(ctx, mgr),
		vipSubnetImportDependency.AddToManager(ctx, mgr),
		vipPortImportDependency.AddToManager(ctx, mgr),
		credentialsDependency.AddToManager(ctx, mgr),
		credentials.AddCredentialsWatch(log, mgr.GetClient(), builder, credentialsDependency),
	); err != nil {
		return err
	}

	r := reconciler.NewController(controllerName, mgr.GetClient(), c.scopeFactory, loadbalancerHelperFactory{}, loadbalancerStatusWriter{})
	return builder.Complete(&r)
}
