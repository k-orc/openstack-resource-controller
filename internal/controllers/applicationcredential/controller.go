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

package applicationcredential

import (
	"context"
	"errors"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"

	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/reconciler"
	"github.com/k-orc/openstack-resource-controller/v2/internal/scope"
	"github.com/k-orc/openstack-resource-controller/v2/internal/util/credentials"
	"github.com/k-orc/openstack-resource-controller/v2/internal/util/dependency"
	"github.com/k-orc/openstack-resource-controller/v2/pkg/predicates"
)

const controllerName = "applicationcredential"

// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=applicationcredentials,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=applicationcredentials/status,verbs=get;update;patch

type applicationcredentialReconcilerConstructor struct {
	scopeFactory scope.Factory
}

func New(scopeFactory scope.Factory) interfaces.Controller {
	return applicationcredentialReconcilerConstructor{scopeFactory: scopeFactory}
}

func (applicationcredentialReconcilerConstructor) GetName() string {
	return controllerName
}

var userDependency = dependency.NewDeletionGuardDependency[*orcv1alpha1.ApplicationCredentialList, *orcv1alpha1.User](
	"spec.resource.userRef",
	func(applicationcredential *orcv1alpha1.ApplicationCredential) []string {
		resource := applicationcredential.Spec.Resource
		if resource == nil {
			return nil
		}
		return []string{string(resource.UserRef)}
	},
	finalizer, externalObjectFieldOwner,
)

var userImportDependency = dependency.NewDependency[*orcv1alpha1.ApplicationCredentialList, *orcv1alpha1.User](
	"spec.import.filter.userRef",
	func(applicationcredential *orcv1alpha1.ApplicationCredential) []string {
		resource := applicationcredential.Spec.Import
		if resource == nil || resource.Filter == nil || resource.Filter.UserRef == nil {
			return nil
		}
		return []string{string(*resource.Filter.UserRef)}
	},
)

// SetupWithManager sets up the controller with the Manager.
func (c applicationcredentialReconcilerConstructor) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	log := ctrl.LoggerFrom(ctx)
	k8sClient := mgr.GetClient()

	userWatchEventHandler, err := userDependency.WatchEventHandler(log, k8sClient)
	if err != nil {
		return err
	}

	userImportWatchEventHandler, err := userImportDependency.WatchEventHandler(log, k8sClient)
	if err != nil {
		return err
	}

	builder := ctrl.NewControllerManagedBy(mgr).
		WithOptions(options).
		Watches(&orcv1alpha1.User{}, userWatchEventHandler,
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.User{})),
		).
		// A second watch is necessary because we need a different handler that omits deletion guards
		Watches(&orcv1alpha1.User{}, userImportWatchEventHandler,
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.User{})),
		).
		For(&orcv1alpha1.ApplicationCredential{})

	if err := errors.Join(
		userDependency.AddToManager(ctx, mgr),
		userImportDependency.AddToManager(ctx, mgr),
		credentialsDependency.AddToManager(ctx, mgr),
		credentials.AddCredentialsWatch(log, mgr.GetClient(), builder, credentialsDependency),
	); err != nil {
		return err
	}

	r := reconciler.NewController(controllerName, mgr.GetClient(), c.scopeFactory, applicationcredentialHelperFactory{}, applicationcredentialStatusWriter{})
	return builder.Complete(&r)
}
