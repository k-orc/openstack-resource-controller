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

package dependency

import (
	"context"
	"fmt"
	"iter"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/k-orc/openstack-resource-controller/internal/util/result"
)

// NewDependency returns a new Dependency, which can perform tasks necessary to manage a dependency between 2 object types. The 2 object types are:
//   - Object: this is the 'source' object.
//   - Dependency: this is the object that a 'source' object may depend on.
//
// For example, a Port may depend on a Subnet, because it references one or more
// Subnets in its Addresses. In this case 'Object' is the Port, and 'Dependency'
// is the subnet.
//
// NewDependency has several type parameters, but only the first 2 are required as all the rest can be inferred. The 2 required parameters are:
//   - pointer to the List type of Object
//   - pointer to the Dependency type
//
// NewDependency takes the following arguments:
//   - indexName: a name representing the path to the Dependency reference in Object.
//   - getDependencyRefs: a function that takes a pointer to Object and returns a slice of strings containing the names of Dependencies
//
// Taking the Port -> Subnet example, the type parameters are:
//   - *PortList: pointer to the list type of Port
//   - *Subnet: pointer to the Dependency type
//
// and the arguments are:
//   - indexName: "spec.resource.addresses[].subnetRef" - a symbolic path to the subnet reference in a Port
//   - getDependencyRefs: func(object *Port) []string{ ... returns a slice containing all subnetRefs in this Port's addresses ... }
func NewDependency[
	objectListTP objectListType[objectListT, objectT],
	depTP dependencyType[depT],

	objectTP objectType[objectT],
	objectT any, objectListT any, depT any,
](indexName string, getDependencyRefs func(objectTP) []string) Dependency[objectTP, objectListTP, depTP, objectT, objectListT, depT] {
	return Dependency[objectTP, objectListTP, depTP, objectT, objectListT, depT]{
		indexName:         indexName,
		getDependencyRefs: getDependencyRefs,
	}
}

type Dependency[
	objectTP objectType[objectT],
	objectListTP objectListType[objectListT, objectT],
	depTP dependencyType[depT],

	objectT any, objectListT any, depT any,
] struct {
	indexName         string
	getDependencyRefs func(objectTP) []string
}

type objectType[objectT any] interface {
	*objectT
	client.Object
}

type objectListType[objectListT any, objectT any] interface {
	client.ObjectList
	*objectListT

	GetItems() []objectT
}

type dependencyType[depT any] interface {
	*depT
	client.Object
}

// GetDependencies returns an iterator over Dependencies for a given Object. For each dependency it returns:
//   - the dependency's name
//   - a Result of fetching the dependency
func (d *Dependency[objectTP, _, depTP, _, _, depT]) GetDependencies(ctx context.Context, k8sClient client.Client, obj objectTP) iter.Seq2[string, result.Result[depT]] {
	depRefs := d.getDependencyRefs(obj)
	return func(yield func(string, result.Result[depT]) bool) {
		for _, depRef := range depRefs {
			var dep depTP = new(depT)
			err := k8sClient.Get(ctx, types.NamespacedName{Name: depRef, Namespace: obj.GetNamespace()}, dep)

			var r result.Result[depT]
			if err != nil {
				r = result.Err[depT](err)
			} else {
				r = result.Ok(dep)
			}
			if !yield(depRef, r) {
				return
			}
		}
	}
}

// GetObjects returns a slice of all Objects which depend on the given Dependency
func (d *Dependency[_, objectListTP, depTP, objectT, objectListT, _]) GetObjects(ctx context.Context, k8sClient client.Client, dep depTP) ([]objectT, error) {
	var objectList objectListTP = new(objectListT)
	if err := k8sClient.List(ctx, objectList, client.InNamespace(dep.GetNamespace()), client.MatchingFields{d.indexName: dep.GetName()}); err != nil {
		return nil, err
	}
	return objectList.GetItems(), nil
}

// AddIndexer adds the required field indexer for this dependency to a manager
func (d *Dependency[objectTP, _, _, objectT, _, _]) AddIndexer(ctx context.Context, mgr ctrl.Manager) error {
	return mgr.GetFieldIndexer().IndexField(ctx, objectTP(new(objectT)), d.indexName, func(cObj client.Object) []string {
		obj, ok := cObj.(objectTP)
		if !ok {
			return nil
		}

		return d.getDependencyRefs(obj)
	})
}

// WatchEventHandler returns an EventHandler which maps a Dependency to all Objects which depend on it
func (d *Dependency[objectTP, _, depTP, _, _, depT]) WatchEventHandler(log logr.Logger, k8sClient client.Client) (handler.EventHandler, error) {
	dependencySpecimen := depTP(new(depT))
	gvks, _, err := k8sClient.Scheme().ObjectKinds(dependencySpecimen)
	if err != nil {
		return nil, err
	}
	if len(gvks) == 0 {
		return nil, fmt.Errorf("no registered GVK for %T", dependencySpecimen)
	}
	log = log.WithValues("watch", gvks[0].Kind)

	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
		log := log.WithValues("name", obj.GetName(), "namespace", obj.GetNamespace())

		dependency, ok := obj.(depTP)
		if !ok {
			log.Info("Watch got unexpected object type", "type", fmt.Sprintf("%T", obj))
			return nil
		}

		objects, err := d.GetObjects(ctx, k8sClient, dependency)
		if err != nil {
			log.Error(err, "listing Routers")
			return nil
		}
		requests := make([]reconcile.Request, len(objects))
		for i := range objects {
			var object objectTP = &objects[i]
			request := &requests[i]

			request.Name = object.GetName()
			request.Namespace = object.GetNamespace()
		}
		return requests
	}), nil
}

// AddDeletionGuard adds a deletion guard controller to the given manager appropriate for this dependency
func (d *Dependency[objectTP, _, depTP, _, _, _]) AddDeletionGuard(mgr ctrl.Manager, finalizer string, fieldOwner client.FieldOwner) error {
	getDependencyRefsForClientObject := func(cObj client.Object) []string {
		obj, ok := cObj.(objectTP)
		if !ok {
			return nil
		}
		return d.getDependencyRefs(obj)
	}

	return AddDeletionGuard[depTP, objectTP](mgr, finalizer, fieldOwner, getDependencyRefsForClientObject, d.GetObjects)
}
