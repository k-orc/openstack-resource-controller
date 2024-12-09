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
	"time"

	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/pkg/predicates"

	ctrlcommon "github.com/k-orc/openstack-resource-controller/internal/controllers/common"
	ctrlexport "github.com/k-orc/openstack-resource-controller/internal/controllers/export"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/generic"
	"github.com/k-orc/openstack-resource-controller/internal/scope"
)

const (
	Finalizer = "openstack.k-orc.cloud/router"

	FieldOwner = "openstack.k-orc.cloud/routercontroller"
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
	// The time to wait before reconciling again when we are expecting glance to finish some task and update status.
	externalUpdatePollingPeriod = 15 * time.Second
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

// Router depends on its external gateways, which are Networks
var externalGWDep = generic.NewDependency[*orcv1alpha1.RouterList, *orcv1alpha1.Network](
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
	log := mgr.GetLogger().WithValues("controller", "router")

	if err := errors.Join(
		externalGWDep.AddIndexer(ctx, mgr),
		externalGWDep.AddDeletionGuard(mgr, Finalizer, FieldOwner),
	); err != nil {
		return err
	}

	externalGWHandler, err := externalGWDep.WatchEventHandler(log, mgr.GetClient())
	if err != nil {
		return err
	}

	reconciler := orcRouterReconciler{
		client:       mgr.GetClient(),
		recorder:     mgr.GetEventRecorderFor("orc-router-controller"),
		scopeFactory: c.scopeFactory,
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&orcv1alpha1.Router{}, builder.WithPredicates(ctrlcommon.NeedsReconcilePredicate(log))).
		Watches(&orcv1alpha1.Network{}, externalGWHandler,
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.Network{})),
		).
		WithOptions(options).
		Complete(&reconciler)
}
