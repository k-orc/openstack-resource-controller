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

package rbacpolicy

import (
	"context"
	"errors"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/api/v1alpha1"

	"github.com/k-orc/openstack-resource-controller/v3/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v3/internal/controllers/generic/reconciler"
	"github.com/k-orc/openstack-resource-controller/v3/internal/scope"
	"github.com/k-orc/openstack-resource-controller/v3/internal/util/credentials"
	"github.com/k-orc/openstack-resource-controller/v3/internal/util/dependency"
	"github.com/k-orc/openstack-resource-controller/v3/pkg/predicates"
)

const controllerName = "rbacpolicy"

// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=rbacpolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=rbacpolicies/status,verbs=get;update;patch

type rbacpolicyReconcilerConstructor struct {
	scopeFactory        scope.Factory
	defaultResyncPeriod time.Duration
}

func New(scopeFactory scope.Factory) interfaces.Controller {
	return &rbacpolicyReconcilerConstructor{scopeFactory: scopeFactory}
}

func (rbacpolicyReconcilerConstructor) GetName() string {
	return controllerName
}

func (c *rbacpolicyReconcilerConstructor) SetDefaultResyncPeriod(d time.Duration) {
	c.defaultResyncPeriod = d
}

var networkDependency = dependency.NewDeletionGuardDependency[*orcv1alpha1.RBACPolicyList, *orcv1alpha1.Network](
	"spec.resource.networkRef",
	func(rbacpolicy *orcv1alpha1.RBACPolicy) []string {
		resource := rbacpolicy.Spec.Resource
		if resource == nil {
			return nil
		}
		return []string{string(resource.NetworkRef)}
	},
	finalizer, externalObjectFieldOwner,
)

// SetupWithManager sets up the controller with the Manager.
func (c *rbacpolicyReconcilerConstructor) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	log := ctrl.LoggerFrom(ctx)
	k8sClient := mgr.GetClient()

	networkWatchEventHandler, err := networkDependency.WatchEventHandler(log, k8sClient)
	if err != nil {
		return err
	}

	builder := ctrl.NewControllerManagedBy(mgr).
		WithOptions(options).
		Watches(&orcv1alpha1.Network{}, networkWatchEventHandler,
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.Network{})),
		).
		For(&orcv1alpha1.RBACPolicy{})

	if err := errors.Join(
		networkDependency.AddToManager(ctx, mgr),
		credentialsDependency.AddToManager(ctx, mgr),
		credentials.AddCredentialsWatch(log, mgr.GetClient(), builder, credentialsDependency),
	); err != nil {
		return err
	}

	r := reconciler.NewController(controllerName, mgr.GetClient(), c.scopeFactory, rbacpolicyHelperFactory{}, rbacpolicyStatusWriter{}, c.defaultResyncPeriod)
	return builder.Complete(&r)
}
