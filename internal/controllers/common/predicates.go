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
	"fmt"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
)

func NeedsReconcilePredicate(log logr.Logger) predicate.Predicate {
	filter := func(obj client.Object, event string) bool {
		log := log.WithValues("predicate", "NeedsReconcile", "event", event, "type", fmt.Sprintf("%T", obj))

		orcObject, ok := obj.(orcv1alpha1.ObjectWithConditions)
		if !ok {
			log.V(0).Info("Expected ObjectWithConditions")
			return false
		}

		// Always reconcile deleted objects. Note that we don't always
		// get a Delete event for a deleted object. If the object was
		// deleted while the controller was not running we will get a
		// Create event for it when the controller syncs.
		if !orcObject.GetDeletionTimestamp().IsZero() {
			return true
		}

		return true
	}

	// We always reconcile create. We get a create event for every object when
	// the controller restarts as the controller has no previously observed
	// state at that time. This means that upgrading the controller will always
	// re-reconcile objects. This has the advantage of being a way to address
	// invalid state from controller bugs, but the disadvantage of potentially
	// causing a 'thundering herd' when the controller restarts.
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			return filter(e.ObjectNew, "Update")
		},
	}
}
