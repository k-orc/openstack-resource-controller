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

package routerinterface

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/types"
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
	Finalizer = "openstack.k-orc.cloud/routerinterface"

	FieldOwner = "openstack.k-orc.cloud/routerinterfacecontroller"
	// Field owner of the object finalizer.
	SSAFinalizerTxn = "finalizer"
	// Field owner of transient status.
	SSAStatusTxn = "status"
)

// ssaFieldOwner returns the field owner for a specific named SSA transaction.
func ssaFieldOwner(txn string) client.FieldOwner {
	return client.FieldOwner(FieldOwner + "/" + txn)
}

const (
	// The time to wait between polling a port for ACTIVE status
	portStatusPollingPeriod = 1 * time.Second
)

type routerInterfaceReconcilerConstructor struct {
	scopeFactory scope.Factory
}

func New(scopeFactory scope.Factory) ctrlexport.Controller {
	return routerInterfaceReconcilerConstructor{scopeFactory: scopeFactory}
}

func (routerInterfaceReconcilerConstructor) GetName() string {
	return "router interface"
}

// orcRouterInterfaceReconciler reconciles an ORC Subnet.
type orcRouterInterfaceReconciler struct {
	client       client.Client
	recorder     record.EventRecorder
	scopeFactory scope.Factory
}

// Index subnets by referenced network
const routerRefPath = "spec.routerRef"

func getRouterInterfacesForRouter(ctx context.Context, k8sClient client.Client, obj *orcv1alpha1.Router) ([]orcv1alpha1.RouterInterface, error) {
	routerInterfaceList := &orcv1alpha1.RouterInterfaceList{}
	if err := k8sClient.List(ctx, routerInterfaceList, client.InNamespace(obj.Namespace), client.MatchingFields{routerRefPath: obj.Name}); err != nil {
		return nil, err
	}

	return routerInterfaceList.Items, nil
}

// SetupWithManager sets up the controller with the Manager.
func (c routerInterfaceReconcilerConstructor) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	log := mgr.GetLogger().WithValues("controller", "router interface")

	reconciler := orcRouterInterfaceReconciler{
		client:       mgr.GetClient(),
		recorder:     mgr.GetEventRecorderFor("orc-router-interface-controller"),
		scopeFactory: c.scopeFactory,
	}

	getRouterRefsForRouterInterface := func(obj client.Object) []string {
		routerInterface, ok := obj.(*orcv1alpha1.RouterInterface)
		if !ok {
			return nil
		}
		return []string{string(routerInterface.Spec.RouterRef)}
	}

	if err := mgr.GetFieldIndexer().IndexField(ctx, &orcv1alpha1.RouterInterface{}, routerRefPath, func(obj client.Object) []string {
		return getRouterRefsForRouterInterface(obj)
	}); err != nil {
		return fmt.Errorf("adding routers by routerinterface index: %w", err)
	}

	err := ctrlcommon.AddDeletionGuard(mgr, Finalizer, FieldOwner, getRouterRefsForRouterInterface, getRouterInterfacesForRouter)
	if err != nil {
		return err
	}

	const subnetRefPath = "spec.subnetRef"

	if err := mgr.GetFieldIndexer().IndexField(ctx, &orcv1alpha1.RouterInterface{}, subnetRefPath, func(obj client.Object) []string {
		routerInterface, ok := obj.(*orcv1alpha1.RouterInterface)
		if !ok {
			return nil
		}
		subnetRef := routerInterface.Spec.SubnetRef
		if subnetRef == nil {
			return nil
		}
		return []string{string(*subnetRef)}
	}); err != nil {
		return err
	}

	getRoutersForSubnet := func(ctx context.Context, k8sClient client.Client, subnet *orcv1alpha1.Subnet) ([]reconcile.Request, error) {
		routerInterfaceList := &orcv1alpha1.RouterInterfaceList{}
		err := k8sClient.List(ctx, routerInterfaceList, client.InNamespace(subnet.Namespace), client.MatchingFields{subnetRefPath: subnet.Name})
		if err != nil {
			return nil, fmt.Errorf("fetching router interfaces for %s: %w", client.ObjectKeyFromObject(subnet), err)
		}

		// Get a list of unique router names
		routerSet := make(map[string]struct{})
		for i := range routerInterfaceList.Items {
			routerInterface := &routerInterfaceList.Items[i]
			routerSet[string(routerInterface.Spec.RouterRef)] = struct{}{}
		}

		i := 0
		routers := make([]reconcile.Request, len(routerSet))
		for router := range routerSet {
			routers[i].Name = router
			routers[i].Namespace = subnet.Namespace
			i++
		}
		return routers, nil
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&orcv1alpha1.Router{}, builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.Router{}))).
		Named("router_interface").
		Watches(&orcv1alpha1.RouterInterface{},
			handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
				log := ctrl.LoggerFrom(ctx).WithValues("watch", "RouterInterface", "name", obj.GetName(), "namespace", obj.GetNamespace())
				routerInterface, ok := obj.(*orcv1alpha1.RouterInterface)
				if !ok {
					log.Info("Watch got unexpected object type", "type", fmt.Sprintf("%T", obj))
					return nil
				}
				return []reconcile.Request{
					{NamespacedName: types.NamespacedName{Namespace: routerInterface.Namespace, Name: string(routerInterface.Spec.RouterRef)}},
				}
			}),
		).
		Watches(&orcv1alpha1.Subnet{},
			handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
				log := ctrl.LoggerFrom(ctx).WithValues("watch", "Subnet", "name", obj.GetName(), "namespace", obj.GetNamespace())
				subnet, ok := obj.(*orcv1alpha1.Subnet)
				if !ok {
					log.Info("Watch got unexpected object type", "type", fmt.Sprintf("%T", obj))
					return nil
				}

				routers, err := getRoutersForSubnet(ctx, mgr.GetClient(), subnet)
				if err != nil {
					log.Error(err, "fetching routers for subnet")
					return nil
				}
				return routers
			}),
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.Subnet{})),
		).
		WithOptions(options).
		Complete(&reconciler)
}
