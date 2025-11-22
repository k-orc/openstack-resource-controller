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

package trunk

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
	"github.com/k-orc/openstack-resource-controller/v2/internal/util/dependency"
)

// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=trunks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=trunks/status,verbs=get;update;patch

const controllerName = "trunk"

var (
	portDependency = dependency.NewDeletionGuardDependency[*orcv1alpha1.TrunkList, *orcv1alpha1.Port](
		"spec.resource.portRef",
		func(trunk *orcv1alpha1.Trunk) []string {
			resource := trunk.Spec.Resource
			if resource == nil {
				return nil
			}
			return []string{string(resource.PortRef)}
		},
		finalizer, externalObjectFieldOwner,
	)

	portImportDependency = dependency.NewDependency[*orcv1alpha1.TrunkList, *orcv1alpha1.Port](
		"spec.import.filter.portRef",
		func(trunk *orcv1alpha1.Trunk) []string {
			resource := trunk.Spec.Import
			if resource == nil || resource.Filter == nil || resource.Filter.PortRef == nil {
				return nil
			}
			return []string{string(*resource.Filter.PortRef)}
		},
	)

	subportDependency = dependency.NewDeletionGuardDependency[*orcv1alpha1.TrunkList, *orcv1alpha1.Port](
		"spec.resource.subports[].portRef",
		func(trunk *orcv1alpha1.Trunk) []string {
			if trunk.Spec.Resource == nil {
				return nil
			}
			ports := make([]string, len(trunk.Spec.Resource.Subports))
			for i := range trunk.Spec.Resource.Subports {
				ports[i] = string(trunk.Spec.Resource.Subports[i].PortRef)
			}
			return ports
		},
		finalizer, externalObjectFieldOwner,
	)

	projectDependency = dependency.NewDeletionGuardDependency[*orcv1alpha1.TrunkList, *orcv1alpha1.Project](
		"spec.resource.projectRef",
		func(trunk *orcv1alpha1.Trunk) []string {
			resource := trunk.Spec.Resource
			if resource == nil || resource.ProjectRef == nil {
				return nil
			}
			return []string{string(*resource.ProjectRef)}
		},
		finalizer, externalObjectFieldOwner,
	)

	projectImportDependency = dependency.NewDependency[*orcv1alpha1.TrunkList, *orcv1alpha1.Project](
		"spec.import.filter.projectRef",
		func(trunk *orcv1alpha1.Trunk) []string {
			resource := trunk.Spec.Import
			if resource == nil || resource.Filter == nil || resource.Filter.ProjectRef == nil {
				return nil
			}
			return []string{string(*resource.Filter.ProjectRef)}
		},
	)
)

type trunkReconcilerConstructor struct {
	scopeFactory scope.Factory
}

func New(scopeFactory scope.Factory) interfaces.Controller {
	return trunkReconcilerConstructor{
		scopeFactory: scopeFactory,
	}
}

func (trunkReconcilerConstructor) GetName() string {
	return controllerName
}

// SetupWithManager sets up the controller with the Manager.
func (c trunkReconcilerConstructor) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	log := ctrl.LoggerFrom(ctx)
	k8sClient := mgr.GetClient()

	portWatchEventHandler, err := portDependency.WatchEventHandler(log, k8sClient)
	if err != nil {
		return err
	}

	portImportWatchEventHandler, err := portImportDependency.WatchEventHandler(log, k8sClient)
	if err != nil {
		return err
	}

	subportWatchEventHandler, err := subportDependency.WatchEventHandler(log, k8sClient)
	if err != nil {
		return err
	}

	projectWatchEventHandler, err := projectDependency.WatchEventHandler(log, k8sClient)
	if err != nil {
		return err
	}

	projectImportWatchEventHandler, err := projectImportDependency.WatchEventHandler(log, k8sClient)
	if err != nil {
		return err
	}

	builder := ctrl.NewControllerManagedBy(mgr).
		WithOptions(options).
		For(&orcv1alpha1.Trunk{}).
		Watches(&orcv1alpha1.Port{}, portWatchEventHandler,
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.Port{})),
		).
		// A second watch is necessary because we need a different handler that omits deletion guards
		Watches(&orcv1alpha1.Port{}, portImportWatchEventHandler,
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.Port{})),
		).
		Watches(&orcv1alpha1.Port{}, subportWatchEventHandler,
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
		portDependency.AddToManager(ctx, mgr),
		portImportDependency.AddToManager(ctx, mgr),
		subportDependency.AddToManager(ctx, mgr),
		projectDependency.AddToManager(ctx, mgr),
		projectImportDependency.AddToManager(ctx, mgr),
		credentialsDependency.AddToManager(ctx, mgr),
		credentials.AddCredentialsWatch(log, mgr.GetClient(), builder, credentialsDependency),
	); err != nil {
		return err
	}

	r := reconciler.NewController(controllerName, mgr.GetClient(), c.scopeFactory, trunkHelperFactory{}, trunkStatusWriter{})
	return builder.Complete(&r)
}

