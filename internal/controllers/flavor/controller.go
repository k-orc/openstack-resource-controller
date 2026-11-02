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

package flavor

import (
	"context"
	"errors"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v3/pkg/predicates"

	"github.com/k-orc/openstack-resource-controller/v3/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v3/internal/controllers/generic/reconciler"
	"github.com/k-orc/openstack-resource-controller/v3/internal/scope"
	"github.com/k-orc/openstack-resource-controller/v3/internal/util/credentials"
	"github.com/k-orc/openstack-resource-controller/v3/internal/util/dependency"
)

const controllerName = "flavor"

// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=flavors,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=flavors/status,verbs=get;update;patch

type flavorReconcilerConstructor struct {
	scopeFactory        scope.Factory
	defaultResyncPeriod time.Duration
}

func New(scopeFactory scope.Factory) interfaces.Controller {
	return &flavorReconcilerConstructor{scopeFactory: scopeFactory}
}

func (flavorReconcilerConstructor) GetName() string {
	return controllerName
}

func (c *flavorReconcilerConstructor) SetDefaultResyncPeriod(d time.Duration) {
	c.defaultResyncPeriod = d
}

var projectDependency = dependency.NewDeletionGuardDependency[*orcv1alpha1.FlavorList, *orcv1alpha1.Project](
	"spec.resource.projectRefs",
	func(group *orcv1alpha1.Flavor) []string {
		resource := group.Spec.Resource
		if resource == nil || len(resource.ProjectRefs) == 0 {
			return nil
		}

		projects := make([]string, 0, len(resource.ProjectRefs))
		for _, u := range resource.ProjectRefs {
			projects = append(projects, string(u))
		}

		return projects
	},
	finalizer, externalObjectFieldOwner,
)

// SetupWithManager sets up the controller with the Manager.
func (c *flavorReconcilerConstructor) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	log := ctrl.LoggerFrom(ctx)
	k8sClient := mgr.GetClient()

	projectWatchEventHandler, err := projectDependency.WatchEventHandler(log, k8sClient)
	if err != nil {
		return err
	}

	builder := ctrl.NewControllerManagedBy(mgr).
		WithOptions(options).
		Watches(&orcv1alpha1.Project{}, projectWatchEventHandler,
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.Project{})),
		).
		For(&orcv1alpha1.Flavor{})

	if err := errors.Join(
		projectDependency.AddToManager(ctx, mgr),
		credentialsDependency.AddToManager(ctx, mgr),
		credentials.AddCredentialsWatch(log, mgr.GetClient(), builder, credentialsDependency),
	); err != nil {
		return err
	}

	r := reconciler.NewController(controllerName, mgr.GetClient(), c.scopeFactory, flavorHelperFactory{}, flavorStatusWriter{}, c.defaultResyncPeriod)
	return builder.Complete(&r)
}
