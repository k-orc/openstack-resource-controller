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
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/subnets"
	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/pkg/predicates"

	ctrlcommon "github.com/k-orc/openstack-resource-controller/internal/controllers/common"
	ctrlexport "github.com/k-orc/openstack-resource-controller/internal/controllers/export"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/generic"
	"github.com/k-orc/openstack-resource-controller/internal/scope"
)

const (
	FieldOwner = "openstack.k-orc.cloud/subnetcontroller"
	// Field owner of transient status.
	SSAStatusTxn = "status"
)

// ssaFieldOwner returns the field owner for a specific named SSA transaction.
func ssaFieldOwner(txn string) client.FieldOwner {
	return client.FieldOwner(FieldOwner + "/" + txn)
}

type subnetReconcilerConstructor struct {
	scopeFactory scope.Factory
}

func New(scopeFactory scope.Factory) ctrlexport.Controller {
	return subnetReconcilerConstructor{scopeFactory: scopeFactory}
}

func (subnetReconcilerConstructor) GetName() string {
	return "subnet"
}

// orcSubnetReconciler reconciles an ORC Subnet.
type orcSubnetReconciler struct {
	client   client.Client
	recorder record.EventRecorder

	subnetReconcilerConstructor
}

var _ generic.ResourceController[*orcv1alpha1.Subnet, *subnets.Subnet] = &orcSubnetReconciler{}

func (r *orcSubnetReconciler) GetK8sClient() client.Client {
	return r.client
}

func (r *orcSubnetReconciler) GetScopeFactory() scope.Factory {
	return r.scopeFactory
}

func (r *orcSubnetReconciler) NewCreateActuator(ctx context.Context, orcObject *orcv1alpha1.Subnet) ([]generic.WaitingOnEvent, generic.CreateResourceActuator[*subnets.Subnet], error) {
	return newCreateActuator(ctx, r, orcObject)
}

func (r *orcSubnetReconciler) NewDeleteActuator(ctx context.Context, orcObject *orcv1alpha1.Subnet) ([]generic.WaitingOnEvent, generic.DeleteResourceActuator[*subnets.Subnet], error) {
	actuator, err := newDeleteActuator(ctx, r, orcObject)
	return nil, actuator, err
}

// SetupWithManager sets up the controller with the Manager.
func (c subnetReconcilerConstructor) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	reconciler := orcSubnetReconciler{
		client:                      mgr.GetClient(),
		recorder:                    mgr.GetEventRecorderFor("orc-subnet-controller"),
		subnetReconcilerConstructor: c,
	}

	log := mgr.GetLogger().WithValues("controller", c.GetName())

	getNetworkRefsForSubnet := func(obj client.Object) []string {
		subnet, ok := obj.(*orcv1alpha1.Subnet)
		if !ok {
			return nil
		}
		return []string{string(subnet.Spec.NetworkRef)}
	}

	// Index subnets by referenced network
	const networkRefPath = "spec.resource.networkRef"

	if err := mgr.GetFieldIndexer().IndexField(ctx, &orcv1alpha1.Subnet{}, networkRefPath, func(obj client.Object) []string {
		return getNetworkRefsForSubnet(obj)
	}); err != nil {
		return fmt.Errorf("adding subnets by network index: %w", err)
	}

	getSubnetsForNetwork := func(ctx context.Context, k8sClient client.Client, obj *orcv1alpha1.Network) ([]orcv1alpha1.Subnet, error) {
		subnetList := &orcv1alpha1.SubnetList{}
		if err := k8sClient.List(ctx, subnetList, client.InNamespace(obj.Namespace), client.MatchingFields{networkRefPath: obj.Name}); err != nil {
			return nil, err
		}

		return subnetList.Items, nil
	}

	finalizer := generic.GetFinalizerName(&reconciler)
	fieldOwner := generic.GetSSAFieldOwner(&reconciler)

	err := ctrlcommon.AddDeletionGuard(mgr, finalizer, fieldOwner, getNetworkRefsForSubnet, getSubnetsForNetwork)
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&orcv1alpha1.Subnet{}).
		Watches(&orcv1alpha1.Network{},
			handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
				log := log.WithValues("watch", "Network", "name", obj.GetName(), "namespace", obj.GetNamespace())

				network, ok := obj.(*orcv1alpha1.Network)
				if !ok {
					log.Info("Watch got unexpected object type", "type", fmt.Sprintf("%T", obj))
					return nil
				}

				subnets, err := getSubnetsForNetwork(ctx, mgr.GetClient(), network)
				if err != nil {
					log.Error(err, "listing Subnets")
					return nil
				}
				requests := make([]reconcile.Request, len(subnets))
				for i := range subnets {
					subnet := &subnets[i]
					request := &requests[i]

					request.Name = subnet.Name
					request.Namespace = subnet.Namespace
				}
				return requests
			}),
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
				return []reconcile.Request{
					{NamespacedName: types.NamespacedName{Namespace: routerInterface.Namespace, Name: string(*routerInterface.Spec.SubnetRef)}},
				}
			}),
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.RouterInterface{})),
		).
		WithOptions(options).
		Complete(&reconciler)
}
