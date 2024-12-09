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

package port

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
	Finalizer = "openstack.k-orc.cloud/port"

	FieldOwner = "openstack.k-orc.cloud/portcontroller"
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
)

type portReconcilerConstructor struct {
	scopeFactory scope.Factory
}

func New(scopeFactory scope.Factory) ctrlexport.Controller {
	return portReconcilerConstructor{scopeFactory: scopeFactory}
}

// orcPortReconciler reconciles an ORC Port.
type orcPortReconciler struct {
	client       client.Client
	recorder     record.EventRecorder
	scopeFactory scope.Factory
}

func (portReconcilerConstructor) GetName() string {
	return "port"
}

var (
	networkDependency = generic.NewDependency[*orcv1alpha1.PortList, *orcv1alpha1.Network]("spec.resource.networkRef", func(port *orcv1alpha1.Port) []string {
		return []string{string(port.Spec.NetworkRef)}
	})

	subnetDependency = generic.NewDependency[*orcv1alpha1.PortList, *orcv1alpha1.Subnet]("spec.resource.addresses[].subnetRef", func(port *orcv1alpha1.Port) []string {
		subnets := make([]string, len(port.Spec.Resource.Addresses))
		for i := range port.Spec.Resource.Addresses {
			subnets[i] = string(*port.Spec.Resource.Addresses[i].SubnetRef)
		}
		return subnets
	})

	securityGroupDependency = generic.NewDependency[*orcv1alpha1.PortList, *orcv1alpha1.SecurityGroup]("spec.resource.securityGroupRefs", func(port *orcv1alpha1.Port) []string {
		securityGroups := make([]string, len(port.Spec.Resource.SecurityGroupRefs))
		for i := range port.Spec.Resource.SecurityGroupRefs {
			securityGroups[i] = string(port.Spec.Resource.SecurityGroupRefs[i])
		}
		return securityGroups
	})
)

// SetupWithManager sets up the controller with the Manager.
func (c portReconcilerConstructor) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	log := mgr.GetLogger().WithValues("controller", "port")

	reconciler := orcPortReconciler{
		client:       mgr.GetClient(),
		recorder:     mgr.GetEventRecorderFor("orc-port-controller"),
		scopeFactory: c.scopeFactory,
	}

	if err := errors.Join(
		networkDependency.AddIndexer(ctx, mgr),
		networkDependency.AddDeletionGuard(mgr, Finalizer, FieldOwner),
		subnetDependency.AddIndexer(ctx, mgr),
		subnetDependency.AddDeletionGuard(mgr, Finalizer, FieldOwner),
		securityGroupDependency.AddIndexer(ctx, mgr),
		securityGroupDependency.AddDeletionGuard(mgr, Finalizer, FieldOwner),
	); err != nil {
		return err
	}

	networkWatchEventHandler, err := networkDependency.WatchEventHandler(log, mgr.GetClient())
	if err != nil {
		return err
	}

	subnetWatchEventHandler, err := subnetDependency.WatchEventHandler(log, mgr.GetClient())
	if err != nil {
		return err
	}

	securityGroupWatchEventHandler, err := securityGroupDependency.WatchEventHandler(log, mgr.GetClient())
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&orcv1alpha1.Port{}, builder.WithPredicates(ctrlcommon.NeedsReconcilePredicate(log))).
		Watches(&orcv1alpha1.Network{}, networkWatchEventHandler,
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.Network{})),
		).
		Watches(&orcv1alpha1.Subnet{}, subnetWatchEventHandler,
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.Subnet{})),
		).
		Watches(&orcv1alpha1.SecurityGroup{}, securityGroupWatchEventHandler,
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.SecurityGroup{})),
		).
		WithOptions(options).
		Complete(&reconciler)
}
