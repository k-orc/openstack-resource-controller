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

package network

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"

	ctrlexport "github.com/k-orc/openstack-resource-controller/internal/controllers/export"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/generic"
	"github.com/k-orc/openstack-resource-controller/internal/scope"
)

// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=networks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=networks/status,verbs=get;update;patch

type networkReconcilerConstructor struct {
	scopeFactory scope.Factory
}

func New(scopeFactory scope.Factory) ctrlexport.Controller {
	return networkReconcilerConstructor{
		scopeFactory: scopeFactory,
	}
}

func (networkReconcilerConstructor) GetName() string {
	return "network"
}

// SetupWithManager sets up the controller with the Manager.
func (c networkReconcilerConstructor) SetupWithManager(_ context.Context, mgr ctrl.Manager, options controller.Options) error {
	reconciler := generic.NewController(c.GetName(), mgr.GetClient(), c.scopeFactory, networkActuatorFactory{}, networkStatusWriter{})

	return ctrl.NewControllerManagedBy(mgr).
		For(&orcv1alpha1.Network{}).
		WithOptions(options).
		Complete(&reconciler)
}
