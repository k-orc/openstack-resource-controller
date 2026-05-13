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

package subnetpool

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

const controllerName = "subnetpool"

// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=subnetpools,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=subnetpools/status,verbs=get;update;patch

type subnetpoolReconcilerConstructor struct {
	scopeFactory scope.Factory
}

func New(scopeFactory scope.Factory) interfaces.Controller {
	return subnetpoolReconcilerConstructor{scopeFactory: scopeFactory}
}

func (subnetpoolReconcilerConstructor) GetName() string {
	return controllerName
}

var projectDependency = dependency.NewDeletionGuardDependency[*orcv1alpha1.SubnetPoolList, *orcv1alpha1.Project](
	"spec.resource.projectRef",
	func(subnetpool *orcv1alpha1.SubnetPool) []string {
		resource := subnetpool.Spec.Resource
		if resource == nil || resource.ProjectRef == nil {
			return nil
		}
		return []string{string(*resource.ProjectRef)}
	},
	finalizer, externalObjectFieldOwner,
)

var addressScopeDependency = dependency.NewDeletionGuardDependency[*orcv1alpha1.SubnetPoolList, *orcv1alpha1.AddressScope](
	"spec.resource.addressScopeRef",
	func(subnetpool *orcv1alpha1.SubnetPool) []string {
		resource := subnetpool.Spec.Resource
		if resource == nil || resource.AddressScopeRef == nil {
			return nil
		}
		return []string{string(*resource.AddressScopeRef)}
	},
	finalizer, externalObjectFieldOwner,
)

var projectImportDependency = dependency.NewDependency[*orcv1alpha1.SubnetPoolList, *orcv1alpha1.Project](
	"spec.import.filter.projectRef",
	func(subnetpool *orcv1alpha1.SubnetPool) []string {
		resource := subnetpool.Spec.Import
		if resource == nil || resource.Filter == nil || resource.Filter.ProjectRef == nil {
			return nil
		}
		return []string{string(*resource.Filter.ProjectRef)}
	},
)

var addressScopeImportDependency = dependency.NewDependency[*orcv1alpha1.SubnetPoolList, *orcv1alpha1.AddressScope](
	"spec.import.filter.addressScopeRef",
	func(subnetpool *orcv1alpha1.SubnetPool) []string {
		resource := subnetpool.Spec.Import
		if resource == nil || resource.Filter == nil || resource.Filter.AddressScopeRef == nil {
			return nil
		}
		return []string{string(*resource.Filter.AddressScopeRef)}
	},
)

// SetupWithManager sets up the controller with the Manager.
func (c subnetpoolReconcilerConstructor) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	log := ctrl.LoggerFrom(ctx)
	k8sClient := mgr.GetClient()

	projectWatchEventHandler, err := projectDependency.WatchEventHandler(log, k8sClient)
	if err != nil {
		return err
	}

	addressScopeWatchEventHandler, err := addressScopeDependency.WatchEventHandler(log, k8sClient)
	if err != nil {
		return err
	}

	projectImportWatchEventHandler, err := projectImportDependency.WatchEventHandler(log, k8sClient)
	if err != nil {
		return err
	}

	addressScopeImportWatchEventHandler, err := addressScopeImportDependency.WatchEventHandler(log, k8sClient)
	if err != nil {
		return err
	}

	builder := ctrl.NewControllerManagedBy(mgr).
		WithOptions(options).
		Watches(&orcv1alpha1.Project{}, projectWatchEventHandler,
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.Project{})),
		).
		Watches(&orcv1alpha1.AddressScope{}, addressScopeWatchEventHandler,
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.AddressScope{})),
		).
		// A second watch is necessary because we need a different handler that omits deletion guards
		Watches(&orcv1alpha1.Project{}, projectImportWatchEventHandler,
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.Project{})),
		).
		// A second watch is necessary because we need a different handler that omits deletion guards
		Watches(&orcv1alpha1.AddressScope{}, addressScopeImportWatchEventHandler,
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.AddressScope{})),
		).
		For(&orcv1alpha1.SubnetPool{})

	if err := errors.Join(
		projectDependency.AddToManager(ctx, mgr),
		addressScopeDependency.AddToManager(ctx, mgr),
		projectImportDependency.AddToManager(ctx, mgr),
		addressScopeImportDependency.AddToManager(ctx, mgr),
		credentialsDependency.AddToManager(ctx, mgr),
		credentials.AddCredentialsWatch(log, mgr.GetClient(), builder, credentialsDependency),
	); err != nil {
		return err
	}

	r := reconciler.NewController(controllerName, mgr.GetClient(), c.scopeFactory, subnetpoolHelperFactory{}, subnetpoolStatusWriter{})
	return builder.Complete(&r)
}
