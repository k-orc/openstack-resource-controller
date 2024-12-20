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
	"fmt"

	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/generic"
	osclients "github.com/k-orc/openstack-resource-controller/internal/osclients"
	orcerrors "github.com/k-orc/openstack-resource-controller/internal/util/errors"
)

// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=servers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=servers/status,verbs=get;update;patch

func (r *orcServerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	orcObject := &orcv1alpha1.Server{}
	err := r.client.Get(ctx, req.NamespacedName, orcObject)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if !orcObject.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, orcObject)
	}

	return r.reconcileNormal(ctx, orcObject)
}

func (r *orcServerReconciler) reconcileNormal(ctx context.Context, orcObject *orcv1alpha1.Server) (_ ctrl.Result, err error) {
	log := ctrl.LoggerFrom(ctx)
	log.V(3).Info("Reconciling server")

	var statusOpts []updateStatusOpt
	addStatus := func(opt updateStatusOpt) {
		statusOpts = append(statusOpts, opt)
	}

	// Ensure we always update status
	defer func() {
		if err != nil {
			addStatus(withError(err))
		}

		err = errors.Join(err, r.updateStatus(ctx, orcObject, statusOpts...))

		var terminalError *orcerrors.TerminalError
		if errors.As(err, &terminalError) {
			log.Error(err, "not scheduling further reconciles for terminal error")
			err = nil
		}
	}()

	actuator, err := newCreateActuator(ctx, r, orcObject)
	if err != nil {
		return ctrl.Result{}, err
	}

	waitEvents, osResource, err := generic.GetOrCreateOSResource(ctx, log, r.client, actuator)
	if err != nil {
		return ctrl.Result{}, err
	}

	if len(waitEvents) > 0 {
		log.V(3).Info("Waiting on events before creation")
		addStatus(withProgressMessage(waitEvents[0].Message()))
		return ctrl.Result{RequeueAfter: generic.MaxRequeue(waitEvents)}, nil
	}

	if osResource == nil {
		// Programming error: if we don't have a resource we should either have an error or be waiting on something
		return ctrl.Result{}, fmt.Errorf("oResource is not set, but no wait events or error")
	}

	addStatus(withResource(osResource))
	if orcObject.Status.ID == nil {
		if err := r.setStatusID(ctx, orcObject, osResource.ID); err != nil {
			return ctrl.Result{}, err
		}
	}

	switch osResource.Status {
	case ServerStatusError:
		return ctrl.Result{}, orcerrors.Terminal(orcv1alpha1.ConditionReasonUnrecoverableError, "Server is in ERROR state")
	case ServerStatusActive:
		// fall through
	default:
		log.V(3).Info("Waiting for OpenStack resource to be ACTIVE")
		return ctrl.Result{RequeueAfter: externalUpdatePollingPeriod}, nil
	}

	log = log.WithValues("ID", osResource.ID)
	log.V(4).Info("Got resource")
	ctx = ctrl.LoggerInto(ctx, log)

	if orcObject.Spec.ManagementPolicy == orcv1alpha1.ManagementPolicyManaged {
		for _, updateFunc := range needsUpdate(actuator.osClient, orcObject, osResource) {
			if err := updateFunc(ctx); err != nil {
				addStatus(withProgressMessage("Updating the OpenStack resource"))
				return ctrl.Result{}, fmt.Errorf("failed to update the OpenStack resource: %w", err)
			}
		}
	}

	return ctrl.Result{}, nil
}

func (r *orcServerReconciler) reconcileDelete(ctx context.Context, orcObject *orcv1alpha1.Server) (_ ctrl.Result, err error) {
	log := ctrl.LoggerFrom(ctx)
	log.V(3).Info("Reconciling OpenStack resource delete")

	var statusOpts []updateStatusOpt
	addStatus := func(opt updateStatusOpt) {
		statusOpts = append(statusOpts, opt)
	}

	deleted := false
	defer func() {
		// No point updating status after removing the finalizer
		if !deleted {
			if err != nil {
				addStatus(withError(err))
			}
			err = errors.Join(err, r.updateStatus(ctx, orcObject, statusOpts...))
		}
	}()

	actuator, err := newActuator(ctx, r, orcObject)
	if err != nil {
		return ctrl.Result{}, nil
	}
	deleted, waitEvents, osResource, err := generic.DeleteResource(ctx, log, r.client, actuator)
	addStatus(withResource(osResource))
	return ctrl.Result{RequeueAfter: generic.MaxRequeue(waitEvents)}, err
}

// needsUpdate returns a slice of functions that call the OpenStack API to
// align the OpenStack resource to its representation in the ORC spec object.
// For server, updates are not implemented.
func needsUpdate(_ osclients.ComputeClient, _ *orcv1alpha1.Server, _ *servers.Server) (updateFuncs []func(context.Context) error) {
	return nil
}
