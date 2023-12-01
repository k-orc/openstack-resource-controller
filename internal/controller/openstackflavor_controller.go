/*
Copyright 2023.

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

package controller

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	openstackv1 "github.com/gophercloud/openstack-resource-controller/api/v1alpha1"
	"github.com/gophercloud/openstack-resource-controller/pkg/apply"
	"github.com/gophercloud/openstack-resource-controller/pkg/cloud"
	"github.com/gophercloud/openstack-resource-controller/pkg/conditions"
)

const (
	OpenStackFlavorFinalizer = "openstackflavor.k-orc.cloud"
)

// OpenStackFlavorReconciler reconciles a OpenStackFlavor object
type OpenStackFlavorReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=openstackflavors,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=openstackflavors/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=openstackflavors/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *OpenStackFlavorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	logger := log.FromContext(ctx)
	logger = logger.WithValues("OpenStackFlavor", req.Name)
	ctx = log.IntoContext(ctx, logger)

	openStackResource := &openstackv1.OpenStackFlavor{}
	err := r.Client.Get(ctx, req.NamespacedName, openStackResource)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("resource not found in the API")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	patchResource := &openstackv1.OpenStackFlavor{}
	patchResource.TypeMeta = openStackResource.TypeMeta
	patchResource.Name = openStackResource.Name
	patchResource.Namespace = openStackResource.Namespace
	conditions.InitialiseRequiredConditions(openStackResource, patchResource)
	controllerutil.AddFinalizer(patchResource, OpenStackFlavorFinalizer)
	patchResource.Labels = map[string]string{
		openstackv1.OpenStackDependencyLabelCloud(openStackResource.Spec.Cloud): "",
	}
	patchResource.Status.Resource = openstackv1.OpenStackFlavorResourceStatus{
		// XXX: This is a hack because the apiserver won't let us patch the status subresource witn an empty resource object
		Description: "Waiting for reconciliation",
	}

	defer func() {
		// If we're returning an error, report it as a TransientError in the Ready condition
		if reterr != nil {
			updated, condition := conditions.SetNotReadyConditionTransientError(openStackResource, patchResource, reterr.Error())

			// Emit an event if we're setting the condition for the first time
			if updated {
				conditions.EmitEventForCondition(r.Recorder, openStackResource, corev1.EventTypeWarning, condition)
			}
		}

		primaryPatch := &openstackv1.OpenStackFlavor{}
		primaryPatch.TypeMeta = patchResource.TypeMeta
		primaryPatch.Labels = patchResource.Labels
		primaryPatch.Finalizers = patchResource.Finalizers

		statusPatch := &openstackv1.OpenStackFlavor{}
		statusPatch.TypeMeta = openStackResource.TypeMeta
		statusPatch.Status = patchResource.Status

		reterr = errors.Join(
			reterr,

			// We must exclude the spec from the patch as it
			// contains required fields which must not be included
			apply.Apply(ctx, r.Client, openStackResource, primaryPatch, "spec"),

			// Ignore a NotFound error after removing the finalizer
			func() error {
				err := apply.ApplyStatus(ctx, r.Client, openStackResource, statusPatch)
				if err != nil && (!apierrors.IsNotFound(err) || len(primaryPatch.Finalizers) != 0) {
					return err
				}
				return nil
			}(),
		)
	}()

	// Get the OpenStackCloud resource
	openStackCloud := &openstackv1.OpenStackCloud{}
	{
		openStackCloudRef := types.NamespacedName{
			Namespace: req.Namespace,
			Name:      openStackResource.Spec.Cloud,
		}
		err := r.Client.Get(ctx, openStackCloudRef, openStackCloud)
		if err != nil && !apierrors.IsNotFound(err) {
			return ctrl.Result{}, fmt.Errorf("fetching OpenStackCloud %s: %w", openStackResource.Spec.Cloud, err)
		}

		// XXX(mbooth): We should check IsReady(openStackCloud) here, but we can't because this breaks us while the cloud is Deleting.
		// We probably need another Condition 'Deleting' so an object can be both Ready and Deleting during the cleanup phase.
		if err != nil {
			conditions.SetNotReadyConditionWaiting(openStackResource, patchResource, []conditions.Dependency{
				{ObjectKey: openStackCloudRef, Resource: "OpenStackCloud"},
			})
			return ctrl.Result{}, nil
		}
	}

	computeClient, err := cloud.NewServiceClient(ctx, r.Client, openStackCloud, "compute")
	if err != nil {
		err = fmt.Errorf("unable to build an OpenStack client: %w", err)
		logger.Info(err.Error())
		return ctrl.Result{}, err
	}

	if !openStackResource.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, computeClient, openStackResource, patchResource)
	}

	return r.reconcile(ctx, computeClient, openStackResource, patchResource)
}

// reconcile handles creation. No modification is accepted.
// TODO: restrict unhandled modification through a webhook
// TODO: potentially handle (some?) modifications accepted in OpenStack, as in `openstack network set`
func (r *OpenStackFlavorReconciler) reconcile(ctx context.Context, computeClient *gophercloud.ServiceClient, resource, patchResource *openstackv1.OpenStackFlavor) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// If the resource doesn't have our finalizer, exit now to add it before creating any resources.
	if !controllerutil.ContainsFinalizer(resource, OpenStackFlavorFinalizer) {
		return ctrl.Result{}, nil
	}

	var flavor *flavors.Flavor
	if resource.Spec.Resource.ID != "" {
		logger = logger.WithValues("OpenStackID", resource.Spec.Resource.ID)

		var err error
		flavor, err = flavors.Get(computeClient, resource.Spec.Resource.ID).Extract()
		if err != nil {
			return ctrl.Result{}, err
		}
		logger.V(4).Info("resource exists in OpenStack")
	} else {
		rxtxFactor, err := strconv.ParseFloat(resource.Spec.Resource.RxTxFactor, 64)
		if err != nil {
			conditions.SetErrorCondition(resource, patchResource, openstackv1.OpenStackErrorReasonInvalidSpec, "error parsing rxtxFactor: "+err.Error())
			return ctrl.Result{}, nil
		}
		flavor, err = flavors.Create(computeClient, flavors.CreateOpts{
			ID:          resource.Spec.Resource.ID,
			Name:        resource.Spec.Resource.Name,
			RAM:         resource.Spec.Resource.RAM,
			VCPUs:       resource.Spec.Resource.VCPUs,
			Disk:        resource.Spec.Resource.Disk,
			Swap:        resource.Spec.Resource.Swap,
			RxTxFactor:  rxtxFactor,
			IsPublic:    resource.Spec.Resource.IsPublic,
			Ephemeral:   resource.Spec.Resource.Ephemeral,
			Description: resource.Spec.Resource.Description,
		}).Extract()
		if err != nil {
			return ctrl.Result{}, err
		}
		logger = logger.WithValues("OpenStackID", flavor.ID)
		logger.Info("OpenStack resource created")
	}

	patchResource.Status.Resource = openstackv1.OpenStackFlavorResourceStatus{
		ID:          flavor.ID,
		Disk:        flavor.Disk,
		RAM:         flavor.RAM,
		Name:        flavor.Name,
		RxTxFactor:  strconv.FormatFloat(flavor.RxTxFactor, 'f', -1, 64),
		Swap:        flavor.Swap,
		VCPUs:       flavor.VCPUs,
		IsPublic:    flavor.IsPublic,
		Ephemeral:   flavor.Ephemeral,
		Description: flavor.Description,
	}

	conditions.SetReadyCondition(resource, patchResource)

	return ctrl.Result{}, nil
}

func (r *OpenStackFlavorReconciler) reconcileDelete(ctx context.Context, computeClient *gophercloud.ServiceClient, resource, patchResource *openstackv1.OpenStackFlavor) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if resource.Spec.Resource.ID == "" {
		logger.Info("deletion was requested on a resource that hasn't been created yet.")
	} else {
		logger = logger.WithValues("OpenStackID", resource.Spec.Resource.ID)
		if !resource.Spec.Unmanaged {
			if resource.Status.Resource.ID != "" {
				if err := flavors.Delete(computeClient, resource.Status.Resource.ID).ExtractErr(); err != nil {
					var gerr gophercloud.ErrDefault404
					if errors.As(err, &gerr) {
						logger.Info("deletion was requested on a resource that can't be found in OpenStack.")
					} else {
						logger.Info("failed to delete resouce in OpenStack; requeuing.")
						return ctrl.Result{}, err
					}
				}
			}
		}
		logger.Info("resource deleted in OpenStack")
	}

	logger.Info("removing finalizer")
	controllerutil.RemoveFinalizer(patchResource, OpenStackFlavorFinalizer)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OpenStackFlavorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&openstackv1.OpenStackFlavor{}).
		WithEventFilter(apply.IgnoreManagedFieldsOnly{}).
		Watches(&openstackv1.OpenStackCloud{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o client.Object) []reconcile.Request {
			// Fetch a list of all OpenStackFlavors that reference this cloud.
			kclient := mgr.GetClient()
			logger := mgr.GetLogger()

			flavors := openstackv1.OpenStackFlavorList{}
			if err := kclient.List(ctx, &flavors,
				client.InNamespace(o.GetNamespace()),
				client.HasLabels{openstackv1.OpenStackDependencyLabelCloud(o.GetName())},
			); err != nil {
				logger.Error(err, "unable to list OpenStackClouds")
				return nil
			}

			// Reconcile each OpenStackCloud that references this secret.
			reqs := make([]reconcile.Request, len(flavors.Items))
			for i := range flavors.Items {
				flavor := flavors.Items[i]
				reqs[i] = reconcile.Request{
					NamespacedName: client.ObjectKey{
						Namespace: flavor.GetNamespace(),
						Name:      flavor.GetName(),
					},
				}
				logger.V(5).Info("update of OpenStackCloud triggers reconcile of OpenStackFlavor",
					"namespace", o.GetNamespace(),
					"cloud", o.GetName(),
					"flavor", flavor.GetName())
			}
			return reqs
		})).
		Complete(r)
}
