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

package router

import (
	"context"
	"errors"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/pkg/predicates"

	ctrlexport "github.com/k-orc/openstack-resource-controller/internal/controllers/export"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/generic"
	"github.com/k-orc/openstack-resource-controller/internal/scope"
	"github.com/k-orc/openstack-resource-controller/internal/util/dependency"
)

// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=routers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=routers/status,verbs=get;update;patch

type routerReconcilerConstructor struct {
	scopeFactory scope.Factory
}

func New(scopeFactory scope.Factory) ctrlexport.Controller {
	return routerReconcilerConstructor{scopeFactory: scopeFactory}
}

func (routerReconcilerConstructor) GetName() string {
	return "router"
}

// Router depends on its external gateways, which are Networks
var externalGWDep = dependency.NewDependency[*orcv1alpha1.RouterList, *orcv1alpha1.Network](
	"spec.resource.externalGateways[].networkRef",
	func(router *orcv1alpha1.Router) []string {
		resource := router.Spec.Resource
		if resource == nil {
			return nil
		}

		networks := make([]string, len(resource.ExternalGateways))
		for i := range resource.ExternalGateways {
			networks[i] = string(resource.ExternalGateways[i].NetworkRef)
		}
		return networks
	},
)

// SetupWithManager sets up the controller with the Manager.
func (c routerReconcilerConstructor) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	log := mgr.GetLogger().WithValues("controller", c.GetName())

	reconciler := generic.NewController(c.GetName(), mgr.GetClient(), c.scopeFactory, routerActuatorFactory{}, routerStatusWriter{})

	finalizer := generic.GetFinalizerName(&reconciler)
	fieldOwner := generic.GetSSAFieldOwner(&reconciler)

	if err := errors.Join(
		externalGWDep.AddIndexer(ctx, mgr),
		externalGWDep.AddDeletionGuard(mgr, finalizer, fieldOwner),
	); err != nil {
		return err
	}

	externalGWHandler, err := externalGWDep.WatchEventHandler(log, mgr.GetClient())
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&orcv1alpha1.Router{}).
		Watches(&orcv1alpha1.Network{}, externalGWHandler,
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.Network{})),
		).
		WithOptions(options).
		Complete(&reconciler)
}
