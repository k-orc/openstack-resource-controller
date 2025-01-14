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

package generic

import (
	"context"
	"errors"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/internal/scope"
	orcerrors "github.com/k-orc/openstack-resource-controller/internal/util/errors"
)

func NewController[
	orcObjectPT interface {
		*orcObjectT
		client.Object
		orcv1alpha1.ObjectWithConditions
	},
	orcObjectT any,
	osResourcePT *osResourceT,
	osResourceT any,
	objectApplyPT ORCApplyConfig[objectApplyPT, statusApplyPT],
	statusApplyPT interface {
		*statusApplyT
		ORCStatusApplyConfig[statusApplyPT]
	},
	statusApplyT any,
](
	name string, k8sClient client.Client, scopeFactory scope.Factory,
	actuatorFactory ActuatorFactory[orcObjectPT, osResourcePT],
	statusWriter ResourceStatusWriter[orcObjectPT, osResourcePT, objectApplyPT, statusApplyPT],
) Controller[orcObjectPT, orcObjectT, osResourcePT, osResourceT, objectApplyPT, statusApplyPT, statusApplyT] {
	return Controller[orcObjectPT, orcObjectT, osResourcePT, osResourceT, objectApplyPT, statusApplyPT, statusApplyT]{
		name:            name,
		client:          k8sClient,
		scopeFactory:    scopeFactory,
		actuatorFactory: actuatorFactory,
		statusWriter:    statusWriter,
	}
}

type Controller[
	orcObjectPT interface {
		*orcObjectT
		client.Object
		orcv1alpha1.ObjectWithConditions
	},
	orcObjectT any,
	osResourcePT *osResourceT,
	osResourceT any,
	objectApplyPT ORCApplyConfig[objectApplyPT, statusApplyPT],
	statusApplyPT interface {
		*statusApplyT
		ORCStatusApplyConfig[statusApplyPT]
	},
	statusApplyT any,
] struct {
	name         string
	client       client.Client
	scopeFactory scope.Factory

	actuatorFactory ActuatorFactory[orcObjectPT, osResourcePT]
	statusWriter    ResourceStatusWriter[orcObjectPT, osResourcePT, objectApplyPT, statusApplyPT]
}

func (c *Controller[_, _, _, _, _, _, _]) GetName() string {
	return c.name
}

func (c *Controller[_, _, _, _, _, _, _]) GetK8sClient() client.Client {
	return c.client
}

func (c *Controller[_, _, _, _, _, _, _]) GetScopeFactory() scope.Factory {
	return c.scopeFactory
}

func (c *Controller[orcObjectPT, orcObjectT, _, _, _, _, _]) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var orcObject orcObjectPT = new(orcObjectT)
	err := c.client.Get(ctx, req.NamespacedName, orcObject)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if !orcObject.GetDeletionTimestamp().IsZero() {
		return c.reconcileDelete(ctx, orcObject)
	}

	return c.reconcileNormal(ctx, orcObject)
}

func (c *Controller[orcObjectPT, _, osResourcePT, _, _, _, _]) reconcileNormal(ctx context.Context, orcObject orcObjectPT) (_ ctrl.Result, err error) {
	log := ctrl.LoggerFrom(ctx)
	log.V(3).Info("Reconciling resource")

	var osResource osResourcePT
	var waitEvents []WaitingOnEvent

	// Ensure we always update status
	defer func() {
		err = errors.Join(err, UpdateStatus(ctx, c, c.statusWriter, orcObject, osResource, nil, waitEvents, err))

		var terminalError *orcerrors.TerminalError
		if errors.As(err, &terminalError) {
			log.Error(err, "not scheduling further reconciles for terminal error")
			err = nil
		}
	}()

	waitEvents, actuator, err := c.actuatorFactory.NewCreateActuator(ctx, orcObject, c)
	if err != nil {
		return ctrl.Result{}, err
	}

	if len(waitEvents) > 0 {
		log.V(3).Info("Waiting on events before creation")
		return ctrl.Result{RequeueAfter: MaxRequeue(waitEvents)}, nil
	}

	waitEvents, osResource, err = GetOrCreateOSResource(ctx, log, c.client, actuator)
	if err != nil {
		return ctrl.Result{}, err
	}

	if len(waitEvents) > 0 {
		log.V(3).Info("Waiting on events before creation")
		return ctrl.Result{RequeueAfter: MaxRequeue(waitEvents)}, nil
	}

	if osResource == nil {
		// Programming error: if we don't have a resource we should either have an error or be waiting on something
		return ctrl.Result{}, fmt.Errorf("oResource is not set, but no wait events or error")
	}

	if actuator.GetStatusID() == nil {
		if err := SetStatusID(ctx, actuator, c.statusWriter, osResource); err != nil {
			return ctrl.Result{}, err
		}
	}

	log = log.WithValues("ID", actuator.GetResourceID(osResource))
	log.V(4).Info("Got resource")
	ctx = ctrl.LoggerInto(ctx, log)

	if actuator.GetManagementPolicy() == orcv1alpha1.ManagementPolicyManaged {
		if updater, ok := actuator.(UpdateResourceActuator[orcObjectPT, osResourcePT]); ok {
			// We deliberately execute all updaters returned by GetResourceUpdates, even if it returns an error.
			var updaters []ResourceUpdater[orcObjectPT, osResourcePT]
			updaters, err = updater.GetResourceUpdaters(ctx, orcObject, osResource, c)

			// We execute all returned updaters, even if some return errors
			for _, updater := range updaters {
				var updaterErr error
				var updaterWaitEvents []WaitingOnEvent

				updaterWaitEvents, orcObject, osResource, updaterErr = updater(ctx, orcObject, osResource)
				err = errors.Join(err, updaterErr)
				waitEvents = append(waitEvents, updaterWaitEvents...)
			}

			if err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

func (c *Controller[orcObjectPT, _, osResourcePT, _, _, _, _]) reconcileDelete(ctx context.Context, orcObject orcObjectPT) (_ ctrl.Result, err error) {
	log := ctrl.LoggerFrom(ctx)
	log.V(3).Info("Reconciling OpenStack resource delete")

	var osResource osResourcePT
	var waitEvents []WaitingOnEvent

	deleted := false
	defer func() {
		// No point updating status after removing the finalizer
		if !deleted {
			err = errors.Join(err, UpdateStatus(ctx, c, c.statusWriter, orcObject, osResource, nil, waitEvents, err))
		}
	}()

	waitEvents, actuator, err := c.actuatorFactory.NewDeleteActuator(ctx, orcObject, c)
	if err != nil {
		return ctrl.Result{}, err
	}

	if len(waitEvents) > 0 {
		log.V(3).Info("Waiting on events before deletion")
		return ctrl.Result{RequeueAfter: MaxRequeue(waitEvents)}, nil
	}

	deleted, waitEvents, osResource, err = DeleteResource(ctx, log, c.client, actuator)
	return ctrl.Result{RequeueAfter: MaxRequeue(waitEvents)}, err
}
