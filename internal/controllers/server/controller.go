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

package server

import (
	"context"
	"errors"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	ctrlexport "github.com/k-orc/openstack-resource-controller/internal/controllers/export"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/generic"
	"github.com/k-orc/openstack-resource-controller/internal/scope"
	"github.com/k-orc/openstack-resource-controller/internal/util/dependency"
	"github.com/k-orc/openstack-resource-controller/pkg/predicates"
)

const (
	FieldOwner = "openstack.k-orc.cloud/servercontroller"
	// Field owner of transient status.
	SSAStatusTxn = "status"
)

// ssaFieldOwner returns the field owner for a specific named SSA transaction.
func ssaFieldOwner(txn string) client.FieldOwner {
	return client.FieldOwner(FieldOwner + "/" + txn)
}

const (
	// The time to wait before reconciling again when we are expecting OpenStack to finish some task and update status.
	externalUpdatePollingPeriod = 15 * time.Second
)

type serverReconcilerConstructor struct {
	scopeFactory scope.Factory
}

func New(scopeFactory scope.Factory) ctrlexport.Controller {
	return serverReconcilerConstructor{scopeFactory: scopeFactory}
}

func (serverReconcilerConstructor) GetName() string {
	return "server"
}

// orcServerReconciler reconciles an ORC Router.
type orcServerReconciler struct {
	client   client.Client
	recorder record.EventRecorder

	serverReconcilerConstructor
}

var _ generic.ResourceController = &orcServerReconciler{}

func (r *orcServerReconciler) GetK8sClient() client.Client {
	return r.client
}

func (r *orcServerReconciler) GetScopeFactory() scope.Factory {
	return r.scopeFactory
}

var (
	flavorDependency = dependency.NewDependency[*orcv1alpha1.ServerList, *orcv1alpha1.Flavor](
		"spec.resource.flavorRef",
		func(server *orcv1alpha1.Server) []string {
			resource := server.Spec.Resource
			if resource == nil {
				return nil
			}

			return []string{string(resource.FlavorRef)}
		},
	)

	imageDependency = dependency.NewDependency[*orcv1alpha1.ServerList, *orcv1alpha1.Image](
		"spec.resource.imageRef",
		func(server *orcv1alpha1.Server) []string {
			resource := server.Spec.Resource
			if resource == nil {
				return nil
			}

			return []string{string(resource.ImageRef)}
		},
	)

	portDependency = dependency.NewDependency[*orcv1alpha1.ServerList, *orcv1alpha1.Port](
		"spec.resource.ports",
		func(server *orcv1alpha1.Server) []string {
			resource := server.Spec.Resource
			if resource == nil {
				return nil
			}

			refs := make([]string, 0, len(resource.Ports))
			for i := range resource.Ports {
				port := &resource.Ports[i]
				if port.PortRef != nil {
					refs = append(refs, string(*port.PortRef))
				}
			}
			return refs
		},
	)

	secretDependency = dependency.NewDependency[*orcv1alpha1.ServerList, *corev1.Secret](
		"spec.resource.userData.secretRef",
		func(server *orcv1alpha1.Server) []string {
			resource := server.Spec.Resource
			if resource == nil || resource.UserData == nil || resource.UserData.SecretRef == nil {
				return nil
			}

			return []string{string(*resource.UserData.SecretRef)}
		},
	)
)

// SetupWithManager sets up the controller with the Manager.
func (c serverReconcilerConstructor) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	log := mgr.GetLogger().WithValues("controller", c.GetName())

	reconciler := orcServerReconciler{
		client:   mgr.GetClient(),
		recorder: mgr.GetEventRecorderFor("orc-server-controller"),

		serverReconcilerConstructor: c,
	}

	finalizer := generic.GetFinalizerName(&reconciler)
	fieldOwner := generic.GetSSAFieldOwner(&reconciler)

	if err := errors.Join(
		flavorDependency.AddIndexer(ctx, mgr),
		// No deletion guard for flavor, because flavors can be safely deleted
		// while referenced by a server
		imageDependency.AddIndexer(ctx, mgr),
		// Image can sometimes, but not always (e.g. when an RBD-backed image
		// has been cloned), be safely deleted while referenced by a server. We
		// just prevent it always.
		imageDependency.AddDeletionGuard(mgr, finalizer, fieldOwner),
		portDependency.AddIndexer(ctx, mgr),
		portDependency.AddDeletionGuard(mgr, finalizer, fieldOwner),
		secretDependency.AddIndexer(ctx, mgr),
		// We don't need a deletion guard on the user-data secret because it's
		// only used on creation.
	); err != nil {
		return err
	}

	flavorWatchEventHandler, err := flavorDependency.WatchEventHandler(log, mgr.GetClient())
	if err != nil {
		return err
	}
	imageWatchEventHandler, err := imageDependency.WatchEventHandler(log, mgr.GetClient())
	if err != nil {
		return err
	}
	portWatchEventHandler, err := portDependency.WatchEventHandler(log, mgr.GetClient())
	if err != nil {
		return err
	}
	secretWatchEventHandler, err := secretDependency.WatchEventHandler(log, mgr.GetClient())
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&orcv1alpha1.Server{}).
		WithOptions(options).
		Watches(&orcv1alpha1.Flavor{}, flavorWatchEventHandler,
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.Flavor{})),
		).
		Watches(&orcv1alpha1.Image{}, imageWatchEventHandler,
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.Image{})),
		).
		Watches(&orcv1alpha1.Port{}, portWatchEventHandler,
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.Port{})),
		).
		// XXX: This is a general watch on secrets. A general watch on secrets
		// is undesirable because:
		// - It requires problematic RBAC
		// - Secrets are arbitrarily large, and we don't want to cache their contents
		//
		// These will require separate solutions. For the latter we should
		// probably use a MetadataOnly watch only secrets.
		Watches(&corev1.Secret{}, secretWatchEventHandler).
		Complete(&reconciler)
}
