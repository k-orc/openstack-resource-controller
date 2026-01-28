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

package lbpool

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

const controllerName = "lbpool"

// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=lbpools,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=lbpools/status,verbs=get;update;patch

type lbpoolReconcilerConstructor struct {
	scopeFactory scope.Factory
}

func New(scopeFactory scope.Factory) interfaces.Controller {
	return lbpoolReconcilerConstructor{scopeFactory: scopeFactory}
}

func (lbpoolReconcilerConstructor) GetName() string {
	return controllerName
}

var loadBalancerDependency = dependency.NewDeletionGuardDependency[*orcv1alpha1.LBPoolList, *orcv1alpha1.LoadBalancer](
	"spec.resource.loadBalancerRef",
	func(lbpool *orcv1alpha1.LBPool) []string {
		resource := lbpool.Spec.Resource
		if resource == nil || resource.LoadBalancerRef == nil {
			return nil
		}
		return []string{string(*resource.LoadBalancerRef)}
	},
	finalizer, externalObjectFieldOwner,
)

var listenerDependency = dependency.NewDeletionGuardDependency[*orcv1alpha1.LBPoolList, *orcv1alpha1.Listener](
	"spec.resource.listenerRef",
	func(lbpool *orcv1alpha1.LBPool) []string {
		resource := lbpool.Spec.Resource
		if resource == nil || resource.ListenerRef == nil {
			return nil
		}
		return []string{string(*resource.ListenerRef)}
	},
	finalizer, externalObjectFieldOwner,
)

var projectDependency = dependency.NewDeletionGuardDependency[*orcv1alpha1.LBPoolList, *orcv1alpha1.Project](
	"spec.resource.projectRef",
	func(lbpool *orcv1alpha1.LBPool) []string {
		resource := lbpool.Spec.Resource
		if resource == nil || resource.ProjectRef == nil {
			return nil
		}
		return []string{string(*resource.ProjectRef)}
	},
	finalizer, externalObjectFieldOwner,
)

var loadBalancerImportDependency = dependency.NewDependency[*orcv1alpha1.LBPoolList, *orcv1alpha1.LoadBalancer](
	"spec.import.filter.loadBalancerRef",
	func(lbpool *orcv1alpha1.LBPool) []string {
		resource := lbpool.Spec.Import
		if resource == nil || resource.Filter == nil || resource.Filter.LoadBalancerRef == nil {
			return nil
		}
		return []string{string(*resource.Filter.LoadBalancerRef)}
	},
)

var listenerImportDependency = dependency.NewDependency[*orcv1alpha1.LBPoolList, *orcv1alpha1.Listener](
	"spec.import.filter.listenerRef",
	func(lbpool *orcv1alpha1.LBPool) []string {
		resource := lbpool.Spec.Import
		if resource == nil || resource.Filter == nil || resource.Filter.ListenerRef == nil {
			return nil
		}
		return []string{string(*resource.Filter.ListenerRef)}
	},
)

var projectImportDependency = dependency.NewDependency[*orcv1alpha1.LBPoolList, *orcv1alpha1.Project](
	"spec.import.filter.projectRef",
	func(lbpool *orcv1alpha1.LBPool) []string {
		resource := lbpool.Spec.Import
		if resource == nil || resource.Filter == nil || resource.Filter.ProjectRef == nil {
			return nil
		}
		return []string{string(*resource.Filter.ProjectRef)}
	},
)

// Member dependencies - for resolving member subnet references
var subnetMemberDependency = dependency.NewDeletionGuardDependency[*orcv1alpha1.LBPoolList, *orcv1alpha1.Subnet](
	"spec.resource.members[*].subnetRef",
	func(lbpool *orcv1alpha1.LBPool) []string {
		resource := lbpool.Spec.Resource
		if resource == nil {
			return nil
		}
		var refs []string
		for _, member := range resource.Members {
			if member.SubnetRef != nil {
				refs = append(refs, string(*member.SubnetRef))
			}
		}
		return refs
	},
	finalizer, externalObjectFieldOwner,
)

// SetupWithManager sets up the controller with the Manager.
func (c lbpoolReconcilerConstructor) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	log := ctrl.LoggerFrom(ctx)
	k8sClient := mgr.GetClient()

	loadBalancerWatchEventHandler, err := loadBalancerDependency.WatchEventHandler(log, k8sClient)
	if err != nil {
		return err
	}

	listenerWatchEventHandler, err := listenerDependency.WatchEventHandler(log, k8sClient)
	if err != nil {
		return err
	}

	projectWatchEventHandler, err := projectDependency.WatchEventHandler(log, k8sClient)
	if err != nil {
		return err
	}

	loadBalancerImportWatchEventHandler, err := loadBalancerImportDependency.WatchEventHandler(log, k8sClient)
	if err != nil {
		return err
	}

	listenerImportWatchEventHandler, err := listenerImportDependency.WatchEventHandler(log, k8sClient)
	if err != nil {
		return err
	}

	projectImportWatchEventHandler, err := projectImportDependency.WatchEventHandler(log, k8sClient)
	if err != nil {
		return err
	}

	subnetMemberWatchEventHandler, err := subnetMemberDependency.WatchEventHandler(log, k8sClient)
	if err != nil {
		return err
	}

	builder := ctrl.NewControllerManagedBy(mgr).
		WithOptions(options).
		Watches(&orcv1alpha1.LoadBalancer{}, loadBalancerWatchEventHandler,
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.LoadBalancer{})),
		).
		Watches(&orcv1alpha1.Listener{}, listenerWatchEventHandler,
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.Listener{})),
		).
		Watches(&orcv1alpha1.Project{}, projectWatchEventHandler,
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.Project{})),
		).
		// A second watch is necessary because we need a different handler that omits deletion guards
		Watches(&orcv1alpha1.LoadBalancer{}, loadBalancerImportWatchEventHandler,
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.LoadBalancer{})),
		).
		// A second watch is necessary because we need a different handler that omits deletion guards
		Watches(&orcv1alpha1.Listener{}, listenerImportWatchEventHandler,
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.Listener{})),
		).
		// A second watch is necessary because we need a different handler that omits deletion guards
		Watches(&orcv1alpha1.Project{}, projectImportWatchEventHandler,
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.Project{})),
		).
		// Member subnet dependency
		Watches(&orcv1alpha1.Subnet{}, subnetMemberWatchEventHandler,
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.Subnet{})),
		).
		For(&orcv1alpha1.LBPool{})

	if err := errors.Join(
		loadBalancerDependency.AddToManager(ctx, mgr),
		listenerDependency.AddToManager(ctx, mgr),
		projectDependency.AddToManager(ctx, mgr),
		loadBalancerImportDependency.AddToManager(ctx, mgr),
		listenerImportDependency.AddToManager(ctx, mgr),
		projectImportDependency.AddToManager(ctx, mgr),
		subnetMemberDependency.AddToManager(ctx, mgr),
		credentialsDependency.AddToManager(ctx, mgr),
		credentials.AddCredentialsWatch(log, mgr.GetClient(), builder, credentialsDependency),
	); err != nil {
		return err
	}

	r := reconciler.NewController(controllerName, mgr.GetClient(), c.scopeFactory, lbpoolHelperFactory{}, lbpoolStatusWriter{})
	return builder.Complete(&r)
}
