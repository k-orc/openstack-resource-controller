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

// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	apiv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	labels "k8s.io/apimachinery/pkg/labels"
	listers "k8s.io/client-go/listers"
	cache "k8s.io/client-go/tools/cache"
)

// RouterLister helps list Routers.
// All objects returned here must be treated as read-only.
type RouterLister interface {
	// List lists all Routers in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*apiv1alpha1.Router, err error)
	// Routers returns an object that can list and get Routers.
	Routers(namespace string) RouterNamespaceLister
	RouterListerExpansion
}

// routerLister implements the RouterLister interface.
type routerLister struct {
	listers.ResourceIndexer[*apiv1alpha1.Router]
}

// NewRouterLister returns a new RouterLister.
func NewRouterLister(indexer cache.Indexer) RouterLister {
	return &routerLister{listers.New[*apiv1alpha1.Router](indexer, apiv1alpha1.Resource("router"))}
}

// Routers returns an object that can list and get Routers.
func (s *routerLister) Routers(namespace string) RouterNamespaceLister {
	return routerNamespaceLister{listers.NewNamespaced[*apiv1alpha1.Router](s.ResourceIndexer, namespace)}
}

// RouterNamespaceLister helps list and get Routers.
// All objects returned here must be treated as read-only.
type RouterNamespaceLister interface {
	// List lists all Routers in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*apiv1alpha1.Router, err error)
	// Get retrieves the Router from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*apiv1alpha1.Router, error)
	RouterNamespaceListerExpansion
}

// routerNamespaceLister implements the RouterNamespaceLister
// interface.
type routerNamespaceLister struct {
	listers.ResourceIndexer[*apiv1alpha1.Router]
}
