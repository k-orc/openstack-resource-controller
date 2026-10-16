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

package dnszoneshare

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

const controllerName = "dnszoneshare"

// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=dnszoneshares,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=dnszoneshares/status,verbs=get;update;patch

type dnszoneshareReconcilerConstructor struct {
	scopeFactory        scope.Factory
	defaultResyncPeriod time.Duration
}

func New(scopeFactory scope.Factory) interfaces.Controller {
	return &dnszoneshareReconcilerConstructor{scopeFactory: scopeFactory}
}

func (dnszoneshareReconcilerConstructor) GetName() string {
	return controllerName
}

func (c *dnszoneshareReconcilerConstructor) SetDefaultResyncPeriod(d time.Duration) {
	c.defaultResyncPeriod = d
}

var zoneDependency = dependency.NewDeletionGuardDependency[*orcv1alpha1.DNSZoneShareList, *orcv1alpha1.DNSZone](
	"spec.resource.zoneRef",
	func(dnszoneshare *orcv1alpha1.DNSZoneShare) []string {
		resource := dnszoneshare.Spec.Resource
		if resource == nil {
			return nil
		}
		return []string{string(resource.ZoneRef)}
	},
	finalizer, externalObjectFieldOwner,
)

// zoneImportDependency indexes spec.import.filter.zoneRef, used for the unmanaged/import path -
// see newActuator for why this needs to be a separate dependency from zoneDependency above (no
// deletion guard: an unmanaged object never owns the zone, so it must not block its deletion).
var zoneImportDependency = dependency.NewDependency[*orcv1alpha1.DNSZoneShareList, *orcv1alpha1.DNSZone](
	"spec.import.filter.zoneRef",
	func(dnszoneshare *orcv1alpha1.DNSZoneShare) []string {
		imp := dnszoneshare.Spec.Import
		if imp == nil || imp.Filter == nil {
			return nil
		}
		return []string{string(imp.Filter.ZoneRef)}
	},
)

// SetupWithManager sets up the controller with the Manager.
func (c *dnszoneshareReconcilerConstructor) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	log := ctrl.LoggerFrom(ctx)
	k8sClient := mgr.GetClient()

	zoneWatchEventHandler, err := zoneDependency.WatchEventHandler(log, k8sClient)
	if err != nil {
		return err
	}

	zoneImportWatchEventHandler, err := zoneImportDependency.WatchEventHandler(log, k8sClient)
	if err != nil {
		return err
	}

	builder := ctrl.NewControllerManagedBy(mgr).
		WithOptions(options).
		Watches(&orcv1alpha1.DNSZone{}, zoneWatchEventHandler,
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.DNSZone{})),
		).
		// A second watch is necessary because we need a different handler that omits deletion guards
		Watches(&orcv1alpha1.DNSZone{}, zoneImportWatchEventHandler,
			builder.WithPredicates(predicates.NewBecameAvailable(log, &orcv1alpha1.DNSZone{})),
		).
		For(&orcv1alpha1.DNSZoneShare{})

	if err := errors.Join(
		zoneDependency.AddToManager(ctx, mgr),
		zoneImportDependency.AddToManager(ctx, mgr),
		credentialsDependency.AddToManager(ctx, mgr),
		credentials.AddCredentialsWatch(log, mgr.GetClient(), builder, credentialsDependency),
	); err != nil {
		return err
	}

	r := reconciler.NewController(controllerName, mgr.GetClient(), c.scopeFactory, dnszoneshareHelperFactory{}, dnszoneshareStatusWriter{}, c.defaultResyncPeriod)
	return builder.Complete(&r)
}
