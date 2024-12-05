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
	"time"

	"github.com/go-logr/logr"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/common"
	osclients "github.com/k-orc/openstack-resource-controller/internal/osclients"
	"github.com/k-orc/openstack-resource-controller/internal/util/applyconfigs"
	orcerrors "github.com/k-orc/openstack-resource-controller/internal/util/errors"
	orcapplyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/pkg/clients/applyconfiguration/api/v1alpha1"
)

// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=servers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=servers/status,verbs=get;update;patch

func (r *orcServerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	orcObject := &v1alpha1.Server{}
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

func (r *orcServerReconciler) getOpenStackClient(ctx context.Context, orcObject *v1alpha1.Server) (osclients.ComputeClient, error) {
	log := ctrl.LoggerFrom(ctx)

	clientScope, err := r.scopeFactory.NewClientScopeFromObject(ctx, r.client, log, orcObject)
	if err != nil {
		return nil, err
	}
	return clientScope.NewComputeClient()
}

func (r *orcServerReconciler) reconcileNormal(ctx context.Context, orcObject *v1alpha1.Server) (_ ctrl.Result, err error) {
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

	// Don't add finalizer until parent network is available to avoid unnecessary reconcile on delete
	if !controllerutil.ContainsFinalizer(orcObject, Finalizer) {
		patch := common.SetFinalizerPatch(orcObject, Finalizer)
		return ctrl.Result{}, r.client.Patch(ctx, orcObject, patch, client.ForceOwnership, ssaFieldOwner(SSAFinalizerTxn))
	}

	computeClient, err := r.getOpenStackClient(ctx, orcObject)
	if err != nil {
		return ctrl.Result{}, err
	}

	// TODO: fetch from the dependencies' status
	var flavorID, imageID string

	osResource, err := getOSResourceFromObject(ctx, log, orcObject, computeClient, flavorID, imageID)
	if err != nil {
		return ctrl.Result{}, err
	}

	if osResource == nil && orcObject.Spec.Import != nil && orcObject.Spec.Import.Filter != nil {
		log.V(3).Info("OpenStack resource does not yet exist")
		addStatus(withProgressMessage("Waiting for OpenStack resource to be created externally"))
		return ctrl.Result{RequeueAfter: externalUpdatePollingPeriod}, err
	}

	addStatus(withResource(osResource))

	if orcObject.Status.ID == nil {
		if err := r.setStatusID(ctx, orcObject, osResource.ID); err != nil {
			return ctrl.Result{}, err
		}
	}

	log = log.WithValues("ID", osResource.ID)
	log.V(4).Info("Got resource")
	ctx = ctrl.LoggerInto(ctx, log)

	if orcObject.Spec.ManagementPolicy == v1alpha1.ManagementPolicyManaged {
		for _, updateFunc := range needsUpdate(computeClient, orcObject, osResource) {
			if err := updateFunc(ctx); err != nil {
				addStatus(withProgressMessage("Updating the OpenStack resource"))
				return ctrl.Result{}, fmt.Errorf("failed to update the OpenStack resource: %w", err)
			}
		}
	}

	return ctrl.Result{}, nil
}

func getOSResourceFromObject(ctx context.Context, log logr.Logger, orcObject *v1alpha1.Server, osClient osclients.ComputeClient, flavorID, imageID string) (*servers.Server, error) {
	switch {
	case orcObject.Status.ID != nil:
		log.V(4).Info("Fetching existing OpenStack resource", "ID", *orcObject.Status.ID)
		osResource, err := osClient.GetServer(ctx, *orcObject.Status.ID)
		if err != nil {
			if orcerrors.IsNotFound(err) {
				// An OpenStack resource we previously referenced has been deleted unexpectedly. We can't recover from this.
				return nil, orcerrors.Terminal(v1alpha1.OpenStackConditionReasonUnrecoverableError, "resource has been deleted from OpenStack")
			}
			return nil, err
		}
		return osResource, nil

	case orcObject.Spec.Import != nil && orcObject.Spec.Import.ID != nil:
		log.V(4).Info("Importing existing OpenStack resource by ID")
		osResource, err := osClient.GetServer(ctx, *orcObject.Spec.Import.ID)
		if orcerrors.IsNotFound(err) {
			// We assume that a resource imported by ID must already exist. It's a terminal error if it doesn't.
			return nil, orcerrors.Terminal(v1alpha1.OpenStackConditionReasonUnrecoverableError, "referenced resource does not exist in OpenStack")
		}
		return osResource, err

	case orcObject.Spec.Import != nil && orcObject.Spec.Import.Filter != nil:
		log.V(4).Info("Importing existing OpenStack resource by filter")
		return GetByFilter(ctx, osClient, *orcObject.Spec.Import.Filter, flavorID, imageID)

	default:
		log.V(4).Info("Checking for previously created OpenStack resource")
		osResource, err := GetByFilter(ctx, osClient, specToFilter(*orcObject.Spec.Resource), flavorID, imageID)
		if err != nil {
			return nil, err
		}

		if osResource == nil {
			return createResource(ctx, orcObject, osClient, flavorID, imageID)
		}

		return osResource, nil
	}
}

func (r *orcServerReconciler) reconcileDelete(ctx context.Context, orcObject *v1alpha1.Server) (_ ctrl.Result, err error) {
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

	// We won't delete the resource for an unmanaged object, or if onDelete is detach
	if orcObject.Spec.ManagementPolicy == v1alpha1.ManagementPolicyUnmanaged || orcObject.Spec.ManagedOptions.GetOnDelete() == v1alpha1.OnDeleteDetach {
		logPolicy := []any{"managementPolicy", orcObject.Spec.ManagementPolicy}
		if orcObject.Spec.ManagementPolicy == v1alpha1.ManagementPolicyManaged {
			logPolicy = append(logPolicy, "onDelete", orcObject.Spec.ManagedOptions.GetOnDelete())
		}
		log.V(4).Info("Not deleting OpenStack resource due to policy", logPolicy...)
	} else {
		deleted, requeue, err := r.deleteResource(ctx, log, orcObject, addStatus)
		if err != nil {
			return ctrl.Result{}, err
		}

		if !deleted {
			return ctrl.Result{RequeueAfter: requeue}, nil
		}
		log.V(4).Info("OpenStack resource is deleted")
	}

	deleted = true

	// Clear the finalizer
	applyConfig := orcapplyconfigv1alpha1.Server(orcObject.Name, orcObject.Namespace).WithUID(orcObject.UID)
	return ctrl.Result{}, r.client.Patch(ctx, orcObject, applyconfigs.Patch(types.ApplyPatchType, applyConfig), client.ForceOwnership, ssaFieldOwner(SSAFinalizerTxn))
}

func (r *orcServerReconciler) deleteResource(ctx context.Context, log logr.Logger, orcObject *v1alpha1.Server, addStatus func(updateStatusOpt)) (bool, time.Duration, error) {
	computeClient, err := r.getOpenStackClient(ctx, orcObject)
	if err != nil {
		return false, 0, err
	}

	if orcObject.Status.ID != nil {
		// This GET is technically redundant because we could just check the
		// result from DELETE, but it's necessary if we want to report
		// status while deleting
		osResource, err := computeClient.GetServer(ctx, *orcObject.Status.ID)

		switch {
		case orcerrors.IsNotFound(err):
			// Success!
			return true, 0, nil

		case err != nil:
			return false, 0, err

		default:
			addStatus(withResource(osResource))

			if len(orcObject.GetFinalizers()) > 1 {
				log.V(4).Info("Deferring resource cleanup due to remaining external finalizers")
				return false, 0, nil
			}

			err := computeClient.DeleteServer(ctx, *orcObject.Status.ID)
			if err != nil {
				return false, 0, err
			}
			return false, deletePollingPeriod, nil
		}
	}

	// If status.ID is not set we need to check for an orphaned
	// resource. If we don't find one, assume success and continue,
	// otherwise set status.ID and let the controller delete by ID.

	// TODO: fetch from the dependencies' status
	var flavorID, imageID string

	osResource, err := GetByFilter(ctx, computeClient, specToFilter(*orcObject.Spec.Resource), flavorID, imageID)
	if err != nil {
		return false, 0, err
	}

	if osResource != nil {
		addStatus(withResource(osResource))
		return false, deletePollingPeriod, r.setStatusID(ctx, orcObject, osResource.ID)
	}

	// Didn't find an orphaned resource. Assume success.
	return true, 0, nil
}

// getResourceName returns the name of the OpenStack resource we should use.
func getResourceName(orcObject *v1alpha1.Server) v1alpha1.OpenStackName {
	if orcObject.Spec.Resource.Name != nil {
		return *orcObject.Spec.Resource.Name
	}
	return v1alpha1.OpenStackName(orcObject.Name)
}

// createResource creates an OpenStack resource from an ORC object
func createResource(ctx context.Context, orcObject *v1alpha1.Server, computeClient osclients.ComputeClient, flavorID, imageID string) (*servers.Server, error) {
	if orcObject.Spec.ManagementPolicy == v1alpha1.ManagementPolicyUnmanaged {
		// Should have been caught by API validation
		return nil, orcerrors.Terminal(v1alpha1.OpenStackConditionReasonInvalidConfiguration, "Not creating unmanaged resource")
	}

	log := ctrl.LoggerFrom(ctx)
	log.V(3).Info("Creating OpenStack resource")

	resource := orcObject.Spec.Resource

	if resource == nil {
		// Should have been caught by API validation
		return nil, orcerrors.Terminal(v1alpha1.OpenStackConditionReasonInvalidConfiguration, "Creation requested, but spec.resource is not set")
	}

	createOpts := servers.CreateOpts{
		Name:      string(getResourceName(orcObject)),
		ImageRef:  imageID,
		FlavorRef: flavorID,
	}

	schedulerHints := servers.SchedulerHintOpts{}

	osResource, err := computeClient.CreateServer(ctx, &createOpts, schedulerHints)

	// We should require the spec to be updated before retrying a create which returned a conflict
	if orcerrors.IsConflict(err) {
		return nil, orcerrors.Terminal(v1alpha1.OpenStackConditionReasonInvalidConfiguration, "invalid configuration creating resource: "+err.Error(), err)
	}

	return osResource, err
}

// needsUpdate returns a slice of functions that call the OpenStack API to
// align the OpenStack resource to its representation in the ORC spec object.
// For server, updates are not implemented.
func needsUpdate(_ osclients.ComputeClient, _ *v1alpha1.Server, _ *servers.Server) (updateFuncs []func(context.Context) error) {
	return nil
}
