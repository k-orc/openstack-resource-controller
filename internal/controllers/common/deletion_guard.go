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
package common

import (
	"context"
	"fmt"
	"slices"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type pointerToObject[T any] interface {
	*T
	client.Object
}

// A deletion guard is a controller which prevents the deletion of objects that objects of another type depend on.
//
// Example: Network and Subnet
//
// We add a deletion guard to network that prevents the network from being
// deleted if it is still in use by any subnet. It is added by the subnet
// controller, but it is a separate controller which reconciles network objects.

func AddDeletionGuard[guardedP pointerToObject[guarded], dependencyP pointerToObject[dependency], guarded, dependency any](
	mgr ctrl.Manager, finalizer string, fieldOwner client.FieldOwner,
	getGuardedFromDependency func(client.Object) []string,
	getDependenciesFromGuarded func(context.Context, client.Client, guardedP) ([]dependency, error),
) error {
	// deletionGuard reconciles the guarded object
	// It adds a finalizer to any guarded object which is not marked as deleted
	// If the guarded object is marked deleted, we remove the finalizer only if there are no dependent objects
	deletionGuard := reconcile.Func(func(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
		log := ctrl.LoggerFrom(ctx, "name", req.Name, "namespace", req.Namespace)
		log.V(5).Info("Reconciling deletion guard")

		k8sClient := mgr.GetClient()

		var guarded guardedP = new(guarded)
		err := k8sClient.Get(ctx, req.NamespacedName, guarded)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return ctrl.Result{}, nil
			}
			return ctrl.Result{}, err
		}

		// If the object hasn't been deleted, we simply check that it has our finalizer
		if guarded.GetDeletionTimestamp().IsZero() {
			if !slices.Contains(guarded.GetFinalizers(), finalizer) {
				log.V(4).Info("Adding finalizer")
				patch := SetFinalizerPatch(guarded, finalizer)
				return ctrl.Result{}, k8sClient.Patch(ctx, guarded, patch, client.ForceOwnership, fieldOwner)
			}

			log.V(5).Info("Finalizer already present")
			return ctrl.Result{}, nil
		}

		log.V(4).Info("Handling delete")

		dependencies, err := getDependenciesFromGuarded(ctx, k8sClient, guarded)
		if err != nil {
			return reconcile.Result{}, nil
		}
		if len(dependencies) == 0 {
			log.V(4).Info("Removing finalizer")
			patch := RemoveFinalizerPatch(guarded)
			return ctrl.Result{}, k8sClient.Patch(ctx, guarded, patch, client.ForceOwnership, fieldOwner)
		}
		log.V(5).Info("Waiting for dependencies", "dependencies", len(dependencies))
		return ctrl.Result{}, nil
	})

	var guardedSpecimen guardedP = new(guarded)
	var dependencySpecimen dependencyP = new(dependency)

	scheme := mgr.GetScheme()
	guardedName, err := prettyName(guardedSpecimen, scheme)
	if err != nil {
		return err
	}
	dependencyName, err := prettyName(dependencySpecimen, scheme)
	if err != nil {
		return err
	}

	controllerName := guardedName + "_deletion_guard_for_" + dependencyName

	// Register deletionGuard with the manager as a reconciler of guarded.
	// We also watch dependency, but we're only interested in deletion events.
	// We need to ensure that if the guarded object is marked deleted we will
	// continue to call deletionGuard every time a dependent object is deleted
	// so that we will eventually be called when the last dependent object is
	// deleted and we can remove the dependency.
	err = builder.ControllerManagedBy(mgr).
		For(guardedSpecimen).
		Watches(dependencySpecimen,
			handler.Funcs{
				DeleteFunc: func(ctx context.Context, evt event.TypedDeleteEvent[client.Object], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
					for _, guarded := range getGuardedFromDependency(evt.Object) {
						q.Add(reconcile.Request{
							NamespacedName: types.NamespacedName{
								Namespace: evt.Object.GetNamespace(),
								Name:      guarded,
							},
						})
					}
				},
			},
		).
		Named(controllerName).
		Complete(deletionGuard)

	if err != nil {
		return fmt.Errorf("failed to construct %s deletion guard for %s controller", guardedName, dependencyName)
	}

	return nil
}

func prettyName(obj runtime.Object, scheme *runtime.Scheme) (string, error) {
	gvks, _, err := scheme.ObjectKinds(obj)
	if err != nil {
		return "", fmt.Errorf("looking up GVK for guarded object %T: %w", obj, err)
	}
	if len(gvks) == 0 {
		return "", fmt.Errorf("no registered kind for guarded object %T", obj)
	}

	return strings.ToLower(gvks[0].Kind), nil
}
