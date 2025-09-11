/*
Copyright 2025 The ORC Authors.

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

package volume

import (
	"context"
	"errors"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v2/pkg/predicates"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/reconciler"
	"github.com/k-orc/openstack-resource-controller/v2/internal/scope"
	"github.com/k-orc/openstack-resource-controller/v2/internal/util/credentials"
	"github.com/k-orc/openstack-resource-controller/v2/internal/util/dependency"
)

const controllerName = "volume"

// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=volumes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=volumes/status,verbs=get;update;patch

type volumeReconcilerConstructor struct {
	scopeFactory scope.Factory
}

func New(scopeFactory scope.Factory) interfaces.Controller {
	return volumeReconcilerConstructor{scopeFactory: scopeFactory}
}

func (volumeReconcilerConstructor) GetName() string {
	return controllerName
}

var volumetypeDependency = dependency.NewDeletionGuardDependency[*orcv1alpha1.VolumeList, *orcv1alpha1.VolumeType](
	"spec.resource.volumetypeRef",
	func(volume *orcv1alpha1.Volume) []string {
		resource := volume.Spec.Resource
		if resource == nil || resource.VolumeTypeRef == nil {
			return nil
		}
		return []string{string(*resource.VolumeTypeRef)}
	},
	finalizer, externalObjectFieldOwner,
)

// SetupWithManager sets up the controller with the Manager.
func (c volumeReconcilerConstructor) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	log := ctrl.LoggerFrom(ctx)
	k8sClient := mgr.GetClient()

	volumetypeWatchEventHandler, err := volumetypeDependency.WatchEventHandler(log, k8sClient)
	if err != nil {
		return err
	}

	builder := ctrl.NewControllerManagedBy(mgr).
		WithOptions(options).
		Watches(&orcv1alpha1.VolumeType{}, volumetypeWatchEventHandler,
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.VolumeType{})),
		).
		For(&orcv1alpha1.Volume{})

	if err := errors.Join(
		volumetypeDependency.AddToManager(ctx, mgr),
		credentialsDependency.AddToManager(ctx, mgr),
		credentials.AddCredentialsWatch(log, mgr.GetClient(), builder, credentialsDependency),
	); err != nil {
		return err
	}

	r := reconciler.NewController(controllerName, mgr.GetClient(), c.scopeFactory, volumeHelperFactory{}, volumeStatusWriter{})
	return builder.Complete(&r)
}
