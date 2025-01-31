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
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/internal/scope"
	orcerrors "github.com/k-orc/openstack-resource-controller/internal/util/errors"
)

type ResourceController interface {
	GetName() string

	GetK8sClient() client.Client
	GetScopeFactory() scope.Factory
}

func NewController[
	orcObjectPT interface {
		*orcObjectT
		client.Object
		orcv1alpha1.ObjectWithConditions
	}, orcObjectT any,
	resourceSpecT any, filterT any,
	objectApplyPT ORCApplyConfig[objectApplyPT, statusApplyPT],
	statusApplyPT interface {
		*statusApplyT
		ORCStatusApplyConfig[statusApplyPT]
	}, statusApplyT any,
	osResourceT any,
](
	name string, k8sClient client.Client, scopeFactory scope.Factory,
	helperFactory ResourceHelperFactory[orcObjectPT, orcObjectT, resourceSpecT, filterT, osResourceT],
	statusWriter ResourceStatusWriter[orcObjectPT, *osResourceT, objectApplyPT, statusApplyPT],
) Controller[orcObjectPT, orcObjectT, resourceSpecT, filterT, objectApplyPT, statusApplyPT, statusApplyT, osResourceT] {
	return Controller[orcObjectPT, orcObjectT, resourceSpecT, filterT, objectApplyPT, statusApplyPT, statusApplyT, osResourceT]{
		name:          name,
		client:        k8sClient,
		scopeFactory:  scopeFactory,
		helperFactory: helperFactory,
		statusWriter:  statusWriter,
	}
}

type Controller[
	orcObjectPT interface {
		*orcObjectT
		client.Object
		orcv1alpha1.ObjectWithConditions
	},
	orcObjectT any,
	resourceSpecT any,
	filterT any,
	objectApplyPT ORCApplyConfig[objectApplyPT, statusApplyPT],
	statusApplyPT interface {
		*statusApplyT
		ORCStatusApplyConfig[statusApplyPT]
	},
	statusApplyT any,
	osResourceT any,
] struct {
	name         string
	client       client.Client
	scopeFactory scope.Factory

	helperFactory ResourceHelperFactory[orcObjectPT, orcObjectT, resourceSpecT, filterT, osResourceT]
	statusWriter  ResourceStatusWriter[orcObjectPT, *osResourceT, objectApplyPT, statusApplyPT]
}

func (c *Controller[_, _, _, _, _, _, _, _]) GetName() string {
	return c.name
}

func (c *Controller[_, _, _, _, _, _, _, _]) GetK8sClient() client.Client {
	return c.client
}

func (c *Controller[_, _, _, _, _, _, _, _]) GetScopeFactory() scope.Factory {
	return c.scopeFactory
}

func (c *Controller[
	orcObjectPT, orcObjectT,
	resourceSpecT, filterT,
	objectApplyPT,
	statusApplyPT, statusApplyT,
	osResourceT,
]) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var orcObject orcObjectPT = new(orcObjectT)
	err := c.client.Get(ctx, req.NamespacedName, orcObject)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	adapter := c.helperFactory.NewAPIObjectAdapter(orcObject)

	if !orcObject.GetDeletionTimestamp().IsZero() {
		return c.reconcileDelete(ctx, adapter)
	}

	return c.reconcileNormal(ctx, adapter)
}

// shouldReconcile filters events when the object status is up to date, and its
// status indicates that no further reconciliation is required.
//
// Specifically it looks at the Progressing condition. It has the following behaviour:
// - Progressing condition is not present -> reconcile
// - Progressing condition is present and True -> reconcile
// - Progressing condition is present and False, but observedGeneration is old -> reconcile
// - Progressing condition is false and observedGeneration is up to date -> do not reconcile
//
// If shouldReconcile is preventing an object from being reconciled which should
// be reconciled, consider if that object's actuator is correctly returning a
// ProgressStatus indicating that the reconciliation should continue.
func shouldReconcile(obj orcv1alpha1.ObjectWithConditions) bool {
	progressing := meta.FindStatusCondition(obj.GetConditions(), orcv1alpha1.ConditionProgressing)
	if progressing == nil {
		return true
	}

	if progressing.Status == metav1.ConditionTrue {
		return true
	}

	return progressing.ObservedGeneration != obj.GetGeneration()
}

func (c *Controller[
	orcObjectPT, orcObjectT,
	resourceSpecT, filterT,
	objectApplyPT,
	statusApplyPT, statusApplyT,
	osResourceT,
]) reconcileNormal(ctx context.Context, objAdapter APIObjectAdapter[orcObjectPT, resourceSpecT, filterT]) (_ ctrl.Result, err error) {
	log := ctrl.LoggerFrom(ctx)

	// We do this here rather than in a predicate because predicates only cover
	// a single watch. Doing it here means we cover all sources of
	// reconciliation, including our dependencies.
	if !shouldReconcile(objAdapter.GetObject()) {
		log.V(3).Info("Status is up to date: not reconciling")
		return ctrl.Result{}, nil
	}

	log.V(3).Info("Reconciling resource")

	var osResource *osResourceT
	var progressStatus []ProgressStatus

	// Ensure we always update status
	defer func() {
		err = errors.Join(err, UpdateStatus(ctx, c, c.statusWriter, objAdapter.GetObject(), osResource, progressStatus, err))

		var terminalError *orcerrors.TerminalError
		if errors.As(err, &terminalError) {
			log.V(2).Info("not scheduling further reconciles for terminal error", "err", err.Error())
			err = nil
		}
	}()

	progressStatus, actuator, err := c.helperFactory.NewCreateActuator(ctx, objAdapter.GetObject(), c)
	if err != nil {
		return ctrl.Result{}, err
	}

	if len(progressStatus) > 0 {
		log.V(3).Info("Waiting on events before creation")
		return ctrl.Result{RequeueAfter: MaxRequeue(progressStatus)}, nil
	}

	progressStatus, osResource, err = GetOrCreateOSResource(ctx, log, c, objAdapter, actuator)
	if err != nil {
		return ctrl.Result{}, err
	}

	if len(progressStatus) > 0 {
		log.V(3).Info("Waiting on events before creation")
		return ctrl.Result{RequeueAfter: MaxRequeue(progressStatus)}, nil
	}

	if osResource == nil {
		// Programming error: if we don't have a resource we should either have an error or be waiting on something
		return ctrl.Result{}, fmt.Errorf("oResource is not set, but no wait events or error")
	}

	if objAdapter.GetStatusID() == nil {
		resourceID := actuator.GetResourceID(osResource)
		if err := SetStatusID(ctx, c, objAdapter.GetObject(), resourceID, c.statusWriter); err != nil {
			return ctrl.Result{}, err
		}
	}

	log = log.WithValues("ID", actuator.GetResourceID(osResource))
	log.V(4).Info("Got resource")
	ctx = ctrl.LoggerInto(ctx, log)

	if objAdapter.GetManagementPolicy() == orcv1alpha1.ManagementPolicyManaged {
		if reconciler, ok := actuator.(ReconcileResourceActuator[orcObjectPT, osResourceT]); ok {
			// We deliberately execute all reconcilers returned by GetResourceUpdates, even if it returns an error.
			var reconcilers []ResourceReconciler[orcObjectPT, osResourceT]
			reconcilers, err = reconciler.GetResourceReconcilers(ctx, objAdapter.GetObject(), osResource, c)

			// We execute all returned updaters, even if some return errors
			for _, updater := range reconcilers {
				var updaterErr error
				var updaterProgressStatus []ProgressStatus

				updaterProgressStatus, updaterErr = updater(ctx, objAdapter.GetObject(), osResource)
				err = errors.Join(err, updaterErr)
				progressStatus = append(progressStatus, updaterProgressStatus...)
			}

			if err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{RequeueAfter: MaxRequeue(progressStatus)}, nil
}

func (c *Controller[
	orcObjectPT, orcObjectT,
	resourceSpecT,
	filterT,
	objectApplyPT,
	statusApplyPT, statusApplyT,
	osResourceT,
]) reconcileDelete(ctx context.Context, objAdapter APIObjectAdapter[orcObjectPT, resourceSpecT, filterT]) (_ ctrl.Result, err error) {
	log := ctrl.LoggerFrom(ctx)
	log.V(3).Info("Reconciling OpenStack resource delete")

	var osResource *osResourceT
	var progressStatus []ProgressStatus

	deleted := false
	defer func() {
		// No point updating status after removing the finalizer
		if !deleted {
			err = errors.Join(err, UpdateStatus(ctx, c, c.statusWriter, objAdapter.GetObject(), osResource, progressStatus, err))
		}
	}()

	progressStatus, actuator, err := c.helperFactory.NewDeleteActuator(ctx, objAdapter.GetObject(), c)
	if err != nil {
		return ctrl.Result{}, err
	}

	if len(progressStatus) > 0 {
		log.V(3).Info("Waiting on events before deletion")
		return ctrl.Result{RequeueAfter: MaxRequeue(progressStatus)}, nil
	}

	deleted, progressStatus, osResource, err = DeleteResource(ctx, log, c, objAdapter, actuator)
	if err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: MaxRequeue(progressStatus)}, nil
}
