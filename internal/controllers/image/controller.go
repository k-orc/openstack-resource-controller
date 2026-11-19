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

package image

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

const controllerName = "image"

// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=images,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=images/status,verbs=get;update;patch

const (
	// The time to wait before reconciling again when we are expecting OpenStack to finish some task and update status.
	externalUpdatePollingPeriod = 15 * time.Second

	// Size of the upload and download buffers.
	transferBufferSizeBytes = 64 * 1024

	// The maximum number of times we will attempt to download content before failing.
	maxDownloadAttempts = 5
)

var projectDependency = dependency.NewDeletionGuardDependency[*orcv1alpha1.ImageList, *orcv1alpha1.Project](
	"spec.resource.members[].projectRef",
	func(image *orcv1alpha1.Image) []string {
		resource := image.Spec.Resource
		if resource == nil {
			return nil
		}
		var projects []string
		for i := range resource.Members {
			if resource.Members[i].ProjectRef != nil {
				projects = append(projects, string(*resource.Members[i].ProjectRef))
			}
		}
		return projects
	},
	finalizer, externalObjectFieldOwner,
)

type imageReconcilerConstructor struct {
	scopeFactory        scope.Factory
	defaultResyncPeriod time.Duration
}

func New(scopeFactory scope.Factory) interfaces.Controller {
	return &imageReconcilerConstructor{scopeFactory: scopeFactory}
}

func (imageReconcilerConstructor) GetName() string {
	return controllerName
}

func (c *imageReconcilerConstructor) SetDefaultResyncPeriod(d time.Duration) {
	c.defaultResyncPeriod = d
}

// SetupWithManager sets up the controller with the Manager.
func (c *imageReconcilerConstructor) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	log := ctrl.LoggerFrom(ctx)

	projectWatchEventHandler, err := projectDependency.WatchEventHandler(log, mgr.GetClient())
	if err != nil {
		return err
	}

	builder := ctrl.NewControllerManagedBy(mgr).
		WithOptions(options).
		For(&orcv1alpha1.Image{}).
		Watches(&orcv1alpha1.Project{}, projectWatchEventHandler,
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.Project{})),
		)

	if err := errors.Join(
		projectDependency.AddToManager(ctx, mgr),
		credentialsDependency.AddToManager(ctx, mgr),
		credentials.AddCredentialsWatch(log, mgr.GetClient(), builder, credentialsDependency),
	); err != nil {
		return err
	}

	r := reconciler.NewController(controllerName, mgr.GetClient(), c.scopeFactory, imageHelperFactory{}, imageStatusWriter{}, c.defaultResyncPeriod)
	return builder.Complete(&r)
}
