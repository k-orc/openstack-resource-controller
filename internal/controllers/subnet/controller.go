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

package subnet

import (
	"context"
	"errors"
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	ctrlexport "github.com/k-orc/openstack-resource-controller/internal/controllers/export"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/generic"
	"github.com/k-orc/openstack-resource-controller/internal/scope"
	"github.com/k-orc/openstack-resource-controller/internal/util/credentials"
	"github.com/k-orc/openstack-resource-controller/internal/util/dependency"
	"github.com/k-orc/openstack-resource-controller/pkg/predicates"
)

type subnetReconcilerConstructor struct {
	scopeFactory scope.Factory
}

func New(scopeFactory scope.Factory) ctrlexport.Controller {
	return subnetReconcilerConstructor{scopeFactory: scopeFactory}
}

func (subnetReconcilerConstructor) GetName() string {
	return controllerName
}

const controllerName = "subnet"

var (
	networkDependency = dependency.NewDeletionGuardDependency[*orcv1alpha1.SubnetList, *orcv1alpha1.Network](
		"spec.resource.networkRef",
		func(subnet *orcv1alpha1.Subnet) []string {
			return []string{string(subnet.Spec.NetworkRef)}
		},
		finalizer, externalObjectFieldOwner,
	)
)

// SetupWithManager sets up the controller with the Manager.
func (c subnetReconcilerConstructor) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	controllerName := c.GetName()
	log := mgr.GetLogger().WithValues("controller", controllerName)
	k8sClient := mgr.GetClient()

	networkWatchEventHandler, err := networkDependency.WatchEventHandler(log, k8sClient)
	if err != nil {
		return err
	}

	builder := ctrl.NewControllerManagedBy(mgr).
		WithOptions(options).
		For(&orcv1alpha1.Subnet{}).
		Watches(&orcv1alpha1.Network{}, networkWatchEventHandler,
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.Network{})),
		).
		Watches(&orcv1alpha1.RouterInterface{},
			handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
				log := log.WithValues("watch", "RouterInterface", "name", obj.GetName(), "namespace", obj.GetNamespace())
				routerInterface, ok := obj.(*orcv1alpha1.RouterInterface)
				if !ok {
					log.Info("Watch got unexpected object type", "type", fmt.Sprintf("%T", obj))
					return nil
				}
				subnetRef := routerInterface.Spec.SubnetRef
				if subnetRef == nil {
					return nil
				}
				return []reconcile.Request{
					{NamespacedName: types.NamespacedName{Namespace: routerInterface.Namespace, Name: string(*subnetRef)}},
				}
			}),
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.RouterInterface{})),
		)

	if err := errors.Join(
		networkDependency.AddToManager(ctx, mgr),
		credentialsDependency.AddToManager(ctx, mgr),
		credentials.AddCredentialsWatch(log, k8sClient, builder, credentialsDependency),
	); err != nil {
		return err
	}

	reconciler := generic.NewController(controllerName, k8sClient, c.scopeFactory, subnetHelperFactory{}, subnetStatusWriter{})
	return builder.Complete(&reconciler)
}
