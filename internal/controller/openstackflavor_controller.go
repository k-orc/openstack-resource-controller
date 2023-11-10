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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/gophercloud/gophercloud"
	openstackv1 "github.com/gophercloud/gopherkube/api/v1alpha1"
	"github.com/gophercloud/gopherkube/pkg/cloud"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
)

// OpenStackFlavorReconciler reconciles a OpenStackFlavor object
type OpenStackFlavorReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=openstack.gophercloud.io,resources=openstackflavors,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=openstack.gophercloud.io,resources=openstackflavors/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=openstack.gophercloud.io,resources=openstackflavors/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the OpenStackFlavor object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.0/pkg/reconcile
func (r *OpenStackFlavorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	logger := log.FromContext(ctx)
	logger = logger.WithValues("OpenStackFlavor", req.Name)

	openStackResource := &openstackv1.OpenStackFlavor{}
	err := r.Client.Get(ctx, req.NamespacedName, openStackResource)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("resource not found in the API")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	openStackCloud := &openstackv1.OpenStackCloud{}
	if err := r.Client.Get(ctx, types.NamespacedName{
		Namespace: req.Namespace,
		Name:      openStackResource.Spec.Cloud,
	}, openStackCloud); err != nil {
		if apierrors.IsNotFound(err) {
			err = fmt.Errorf("OpenStackCloud %q not found: %w", openStackResource.Spec.Cloud, err)
			logger.Info(err.Error())
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	// Initialize the patch helper
	resourcePatchHelper, err := patch.NewHelper(openStackResource, r.Client)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Always patch the resource when exiting this function.
	defer func() {
		reterr = kerrors.NewAggregate([]error{
			reterr,
			resourcePatchHelper.Patch(ctx, openStackResource),
		})
	}()

	networkClient, err := cloud.NewClient(log.IntoContext(ctx, logger), r.Client, openStackCloud, "compute")
	if err != nil {
		err = fmt.Errorf("unable to build an OpenStack client: %w", err)
		logger.Info(err.Error())
		return ctrl.Result{}, err
	}

	if !openStackResource.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(log.IntoContext(ctx, logger), networkClient, openStackResource)
	}

	return r.reconcile(log.IntoContext(ctx, logger), networkClient, openStackResource)
}

// reconcile handles creation. No modification is accepted.
// TODO: restrict unhandled modification through a webhook
// TODO: potentially handle (some?) modifications accepted in OpenStack, as in `openstack network set`
func (r *OpenStackFlavorReconciler) reconcile(ctx context.Context, computeClient *gophercloud.ServiceClient, resource *openstackv1.OpenStackFlavor) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// If the resource doesn't have our finalizer, add it.
	if controllerutil.AddFinalizer(resource, openstackv1.Finalizer) {
		// Register the finalizer immediately to avoid orphaning OpenStack resources on delete
		return ctrl.Result{}, nil
	}

	// If the resource has an ID set but hasn't been created by us, then
	// it's unmanaged by default.
	if resource.Spec.Unmanaged == nil {
		var unmanaged bool
		if resource.Spec.ID != "" && resource.Status.ID == "" {
			unmanaged = true
		}
		resource.Spec.Unmanaged = &unmanaged
		return ctrl.Result{}, nil
	}

	var openstackResource *flavors.Flavor
	if resource.Spec.ID != "" {
		logger = logger.WithValues("OpenStackID", resource.Spec.ID)

		var err error
		openstackResource, err = flavors.Get(computeClient, resource.Spec.ID).Extract()
		if err != nil {
			return ctrl.Result{}, err
		}
		logger.Info("resouce exists in OpenStack")
	} else {
		rxtxFactor, err := strconv.ParseFloat(resource.Spec.RxTxFactor, 64)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("error parsing rxtxFactor: %v", err)
		}
		openstackResource, err = flavors.Create(computeClient, flavors.CreateOpts{
			ID:          resource.Spec.ID,
			Name:        resource.Spec.Name,
			RAM:         resource.Spec.RAM,
			VCPUs:       resource.Spec.VCPUs,
			Disk:        resource.Spec.Disk,
			Swap:        resource.Spec.Swap,
			RxTxFactor:  rxtxFactor,
			IsPublic:    resource.Spec.IsPublic,
			Ephemeral:   resource.Spec.Ephemeral,
			Description: resource.Spec.Description,
		}).Extract()
		if err != nil {
			return ctrl.Result{}, err
		}
		resource.Spec.ID = openstackResource.ID
		logger = logger.WithValues("OpenStackID", openstackResource.ID)
		logger.Info("OpenStack resource created")
	}

	resource.Status = openstackv1.OpenStackFlavorStatus{
		ID:          openstackResource.ID,
		Disk:        openstackResource.Disk,
		RAM:         openstackResource.RAM,
		Name:        openstackResource.Name,
		RxTxFactor:  strconv.FormatFloat(openstackResource.RxTxFactor, 'f', -1, 64),
		Swap:        openstackResource.Swap,
		VCPUs:       openstackResource.VCPUs,
		IsPublic:    openstackResource.IsPublic,
		Ephemeral:   openstackResource.Ephemeral,
		Description: openstackResource.Description,
	}

	return ctrl.Result{}, nil
}

func (r *OpenStackFlavorReconciler) reconcileDelete(ctx context.Context, computeClient *gophercloud.ServiceClient, resource *openstackv1.OpenStackFlavor) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if resource.Spec.ID == "" {
		logger.Info("deletion was requested on a resource that hasn't been created yet.")
	} else {
		logger = logger.WithValues("OpenStackID", resource.Spec.ID)
		if resource.Spec.Unmanaged != nil && !*resource.Spec.Unmanaged {
			if err := flavors.Delete(computeClient, resource.Spec.ID).ExtractErr(); err != nil {
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

	controllerutil.RemoveFinalizer(resource, openstackv1.Finalizer)
	logger.Info("resouce deleted in OpenStack.")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OpenStackFlavorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&openstackv1.OpenStackFlavor{}).
		Complete(r)
}
