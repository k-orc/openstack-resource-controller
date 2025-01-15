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

// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=ports,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=ports/status,verbs=get;update;patch

var (
	networkDependency = dependency.NewDependency[*orcv1alpha1.PortList, *orcv1alpha1.Network]("spec.resource.networkRef", func(port *orcv1alpha1.Port) []string {
		return []string{string(port.Spec.NetworkRef)}
	})

	subnetDependency = dependency.NewDependency[*orcv1alpha1.PortList, *orcv1alpha1.Subnet]("spec.resource.addresses[].subnetRef", func(port *orcv1alpha1.Port) []string {
		subnets := make([]string, len(port.Spec.Resource.Addresses))
		for i := range port.Spec.Resource.Addresses {
			subnets[i] = string(*port.Spec.Resource.Addresses[i].SubnetRef)
		}
		return subnets
	})

	securityGroupDependency = dependency.NewDependency[*orcv1alpha1.PortList, *orcv1alpha1.SecurityGroup]("spec.resource.securityGroupRefs", func(port *orcv1alpha1.Port) []string {
		securityGroups := make([]string, len(port.Spec.Resource.SecurityGroupRefs))
		for i := range port.Spec.Resource.SecurityGroupRefs {
			securityGroups[i] = string(port.Spec.Resource.SecurityGroupRefs[i])
		}
		return securityGroups
	})
)

type portReconcilerConstructor struct {
	scopeFactory scope.Factory
}

func New(scopeFactory scope.Factory) ctrlexport.Controller {
	return portReconcilerConstructor{scopeFactory: scopeFactory}
}

func (portReconcilerConstructor) GetName() string {
	return "port"
}

// SetupWithManager sets up the controller with the Manager.
func (c portReconcilerConstructor) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	reconciler := generic.NewController(c.GetName(), mgr.GetClient(), c.scopeFactory, portActuatorFactory{}, portStatusWriter{})

	log := mgr.GetLogger().WithValues("controller", c.GetName())

	finalizer := generic.GetFinalizerName(&reconciler)
	fieldOwner := generic.GetSSAFieldOwner(&reconciler)

	if err := errors.Join(
		networkDependency.AddIndexer(ctx, mgr),
		networkDependency.AddDeletionGuard(mgr, finalizer, fieldOwner),
		subnetDependency.AddIndexer(ctx, mgr),
		subnetDependency.AddDeletionGuard(mgr, finalizer, fieldOwner),
		securityGroupDependency.AddIndexer(ctx, mgr),
		securityGroupDependency.AddDeletionGuard(mgr, finalizer, fieldOwner),
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
		For(&orcv1alpha1.Port{}).
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
