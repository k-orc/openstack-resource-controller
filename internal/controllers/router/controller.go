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

	"github.com/k-orc/openstack-resource-controller/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/generic/reconciler"
	"github.com/k-orc/openstack-resource-controller/internal/scope"
	"github.com/k-orc/openstack-resource-controller/internal/util/credentials"
	"github.com/k-orc/openstack-resource-controller/internal/util/dependency"
)

// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=routers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=routers/status,verbs=get;update;patch

type routerReconcilerConstructor struct {
	scopeFactory scope.Factory
}

func New(scopeFactory scope.Factory) interfaces.Controller {
	return routerReconcilerConstructor{scopeFactory: scopeFactory}
}

func (routerReconcilerConstructor) GetName() string {
	return controllerName
}

const controllerName = "router"

var (
	// Router depends on its external gateways, which are Networks
	externalGWDep = dependency.NewDeletionGuardDependency[*orcv1alpha1.RouterList, *orcv1alpha1.Network](
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
		finalizer, externalObjectFieldOwner,
	)
)

// SetupWithManager sets up the controller with the Manager.
func (c routerReconcilerConstructor) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	log := mgr.GetLogger().WithValues("controller", controllerName)
	k8sClient := mgr.GetClient()

	externalGWHandler, err := externalGWDep.WatchEventHandler(log, k8sClient)
	if err != nil {
		return err
	}

	builder := ctrl.NewControllerManagedBy(mgr).
		WithOptions(options).
		For(&orcv1alpha1.Router{}).
		Watches(&orcv1alpha1.Network{}, externalGWHandler,
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.Network{})),
		)

	if err := errors.Join(
		externalGWDep.AddToManager(ctx, mgr),
		credentialsDependency.AddToManager(ctx, mgr),
		credentials.AddCredentialsWatch(log, k8sClient, builder, credentialsDependency),
	); err != nil {
		return err
	}

	r := reconciler.NewController(controllerName, k8sClient, c.scopeFactory, routerHelperFactory{}, routerStatusWriter{})
	return builder.Complete(&r)
}
