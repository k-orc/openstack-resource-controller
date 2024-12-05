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
	"fmt"
	"time"

	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/pkg/predicates"

	ctrlcommon "github.com/k-orc/openstack-resource-controller/internal/controllers/common"
	ctrlexport "github.com/k-orc/openstack-resource-controller/internal/controllers/export"
	"github.com/k-orc/openstack-resource-controller/internal/scope"
)

const (
	Finalizer = "openstack.k-orc.cloud/router"

	FieldOwner = "openstack.k-orc.cloud/routercontroller"
	// Field owner of the object finalizer.
	SSAFinalizerTxn = "finalizer"
	// Field owner of transient status.
	SSAStatusTxn = "status"
	// Field owner of persistent id field.
	SSAIDTxn = "id"
)

// ssaFieldOwner returns the field owner for a specific named SSA transaction.
func ssaFieldOwner(txn string) client.FieldOwner {
	return client.FieldOwner(FieldOwner + "/" + txn)
}

const (
	// The time to wait before reconciling again when we are expecting glance to finish some task and update status.
	externalUpdatePollingPeriod = 15 * time.Second

	// The time to wait between polling for resource deletion
	deletePollingPeriod = 1 * time.Second
)

type routerReconcilerConstructor struct {
	scopeFactory scope.Factory
}

func New(scopeFactory scope.Factory) ctrlexport.Controller {
	return routerReconcilerConstructor{scopeFactory: scopeFactory}
}

func (routerReconcilerConstructor) GetName() string {
	return "router"
}

// orcRouterReconciler reconciles an ORC Router.
type orcRouterReconciler struct {
	client       client.Client
	recorder     record.EventRecorder
	scopeFactory scope.Factory
}

// SetupWithManager sets up the controller with the Manager.
func (c routerReconcilerConstructor) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	log := mgr.GetLogger().WithValues("controller", "router")

	reconciler := orcRouterReconciler{
		client:       mgr.GetClient(),
		recorder:     mgr.GetEventRecorderFor("orc-router-controller"),
		scopeFactory: c.scopeFactory,
	}

	getExternalGatewayRefsForResource := func(obj client.Object) []string {
		router, ok := obj.(*orcv1alpha1.Router)
		if !ok {
			return nil
		}

		resource := router.Spec.Resource
		if resource == nil {
			return nil
		}

		networks := make([]string, len(resource.ExternalGateways))
		for i := range resource.ExternalGateways {
			networks[i] = string(resource.ExternalGateways[i].NetworkRef)
		}
		return networks
	}

	// Index routers by referenced external gateway
	const routerNetworkRefPath = "spec.resource.externalGateways.networkRef"

	if err := mgr.GetFieldIndexer().IndexField(ctx, &orcv1alpha1.Router{}, routerNetworkRefPath, func(obj client.Object) []string {
		return getExternalGatewayRefsForResource(obj)
	}); err != nil {
		return fmt.Errorf("adding networks by router index: %w", err)
	}

	getRoutersForExternalGateway := func(ctx context.Context, k8sClient client.Client, obj *orcv1alpha1.Network) ([]orcv1alpha1.Router, error) {
		routerList := &orcv1alpha1.RouterList{}
		if err := k8sClient.List(ctx, routerList, client.InNamespace(obj.Namespace), client.MatchingFields{routerNetworkRefPath: obj.Name}); err != nil {
			return nil, err
		}

		return routerList.Items, nil
	}

	err := ctrlcommon.AddDeletionGuard(mgr, Finalizer, FieldOwner, getExternalGatewayRefsForResource, getRoutersForExternalGateway)
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&orcv1alpha1.Router{}, builder.WithPredicates(ctrlcommon.NeedsReconcilePredicate(log))).
		Watches(&orcv1alpha1.Network{},
			handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
				log := log.WithValues("watch", "Network", "name", obj.GetName(), "namespace", obj.GetNamespace())

				network, ok := obj.(*orcv1alpha1.Network)
				if !ok {
					log.Info("Watch got unexpected object type", "type", fmt.Sprintf("%T", obj))
					return nil
				}

				routers, err := getRoutersForExternalGateway(ctx, mgr.GetClient(), network)
				if err != nil {
					log.Error(err, "listing Routers")
					return nil
				}
				requests := make([]reconcile.Request, len(routers))
				for i := range routers {
					router := &routers[i]
					request := &requests[i]

					request.Name = router.Name
					request.Namespace = router.Namespace
				}
				return requests
			}),
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.Network{})),
		).
		WithOptions(options).
		Complete(&reconciler)
}
