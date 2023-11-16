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

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/go-logr/logr"
	"github.com/gophercloud/gophercloud"
	openstackv1 "github.com/gophercloud/gopherkube/api/v1alpha1"
	"github.com/gophercloud/gopherkube/pkg/cloud"
	"github.com/gophercloud/gopherkube/pkg/util"
)

// OpenStackCloudReconciler reconciles a OpenStackImage object
type OpenStackCloudReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

func finalizerName(cloud *openstackv1.OpenStackCloud) string {
	return openstackv1.Finalizer + "/" + cloud.Name
}

//+kubebuilder:rbac:groups=openstack.gopherkube.dev,resources=openstackclouds,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=openstack.gopherkube.dev,resources=openstackclouds/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=openstack.gopherkube.dev,resources=openstackclouds/finalizers,verbs=update
//+kubebuilder:rbac:resources=secrets,verbs=get;list;watch
//+kubebuilder:rbac:resources=secrets/finalizers,verbs=update

func (r *OpenStackCloudReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	logger := log.FromContext(ctx)
	logger = logger.WithValues("OpenStackCloud", req.Name)

	openStackResource := &openstackv1.OpenStackCloud{}
	err := r.Client.Get(ctx, req.NamespacedName, openStackResource)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("resource not found in the API")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	logger.Info("reconciling resource")

	patchResource := &openstackv1.OpenStackCloud{}
	patchResource.TypeMeta = openStackResource.TypeMeta
	util.InitialiseRequiredConditions(patchResource)
	controllerutil.AddFinalizer(patchResource, openstackv1.Finalizer)

	defer func() {
		// If we're returning an error, report it as a TransientError in the Ready condition
		if reterr != nil {
			updated, condition := util.SetNotReadyConditionTransientError(openStackResource, patchResource, reterr.Error())

			// Emit an event if we're setting the condition for the first time
			if updated {
				util.EmitEventForCondition(r.Recorder, openStackResource, corev1.EventTypeWarning, condition)
			}
		}

		primaryPatch := &openstackv1.OpenStackCloud{}
		primaryPatch.TypeMeta = patchResource.TypeMeta
		primaryPatch.Labels = patchResource.Labels
		primaryPatch.Finalizers = patchResource.Finalizers
		// We must exclude the spec from the patch as it
		// contains required fields which must not be included
		applyerr := util.Apply(ctx, r.Client, openStackResource, primaryPatch, "spec")

		if applyerr == nil {
			statusPatch := &openstackv1.OpenStackCloud{}
			statusPatch.TypeMeta = openStackResource.TypeMeta
			statusPatch.Status = patchResource.Status
			applyerr = util.ApplyStatus(ctx, r.Client, openStackResource, statusPatch)

			// Ignore the error if we get NotFound after removing the finalizer
			if applyerr != nil && apierrors.IsNotFound(applyerr) && len(primaryPatch.Finalizers) == 0 {
				applyerr = nil
			}
		}

		reterr = errors.Join(reterr, applyerr)
	}()

	if !openStackResource.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, logger, openStackResource, patchResource)
	}

	// If our finalizer isn't set, ensure it is persisted before make any changes
	if !controllerutil.ContainsFinalizer(openStackResource, openstackv1.Finalizer) {
		// We will be reconciled again immediately because we're adding the finalizer
		return ctrl.Result{}, nil
	}

	// Ensure the secret label is set
	if openStackResource.Spec.Credentials.Source != openstackv1.OpenStackCloudCredentialsSourceTypeSecret {
		updated, condition := util.SetErrorCondition(openStackResource, patchResource, openstackv1.OpenStackCloudCredentialsSourceInvalid,
			"invalid credentials source "+openStackResource.Spec.Credentials.Source)

		// Emit an event if we're setting the condition for the first time
		if updated {
			util.EmitEventForCondition(r.Recorder, openStackResource, corev1.EventTypeWarning, condition)
		}

		return ctrl.Result{}, nil
	}

	secretName := openStackResource.Spec.Credentials.SecretRef.Name

	// Label the cloud resource with the secret name so we can trigger a reconcile on secret changes
	patchResource.Labels = make(map[string]string)
	patchResource.Labels[openstackv1.OpenStackCloudSecretNameLabel] = secretName

	// Fetch the secret
	secret := &corev1.Secret{}
	{
		secretRef := client.ObjectKey{Namespace: openStackResource.Namespace, Name: secretName}
		err := r.Client.Get(ctx, secretRef, secret)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return ctrl.Result{}, err
			}
		}

		// Secret either doesn't exist, or is being deleted
		if err != nil || !secret.DeletionTimestamp.IsZero() {
			logger.Info("waiting for secret")

			updated, condition := util.SetNotReadyConditionWaiting(openStackResource, patchResource, []util.Dependency{
				{ObjectKey: secretRef, Resource: "secret"}})

			// Emit an event if we're setting the condition for the first time
			if updated {
				util.EmitEventForCondition(r.Recorder, openStackResource, corev1.EventTypeNormal, condition)
			}

			return ctrl.Result{}, nil
		}
	}

	// Set our finalizer on the secret
	if !controllerutil.ContainsFinalizer(secret, finalizerName(openStackResource)) {
		logger.Info("adding finalizer to secret")

		patchSecret := &corev1.Secret{}
		patchSecret.APIVersion = "v1"
		patchSecret.Kind = "Secret"
		controllerutil.AddFinalizer(patchSecret, finalizerName(openStackResource))
		if err := util.Apply(ctx, r.Client, secret, patchSecret); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Test the credentials
	{
		_, _, err := cloud.NewProviderClient(ctx, r.Client, openStackResource)
		if err != nil {
			switch err.(type) {
			// Set BadCredentials for any non-transient error
			case cloud.BadCredentialsError, gophercloud.ErrDefault400, gophercloud.ErrDefault401, gophercloud.ErrDefault403, gophercloud.ErrDefault404, gophercloud.ErrDefault405:
				updated, condition := util.SetErrorCondition(openStackResource, patchResource, util.OpenStackConditionReasonBadCredentials, err.Error())

				// Emit an event if we're setting the condition for the first time
				if updated {
					util.EmitEventForCondition(r.Recorder, openStackResource, corev1.EventTypeWarning, condition)
				}
				return ctrl.Result{}, nil
			default:
				return ctrl.Result{}, fmt.Errorf("validating credentials: %w", err)
			}
		}
	}

	// Set the Ready condition
	{
		updated, condition := util.SetReadyCondition(openStackResource, patchResource)

		// Emit an event if we're setting the condition for the first time
		if updated {
			util.EmitEventForCondition(r.Recorder, openStackResource, corev1.EventTypeNormal, condition)
		}
	}

	return ctrl.Result{}, nil
}

func (r *OpenStackCloudReconciler) reconcileDelete(ctx context.Context, logger logr.Logger, openStackResource, patchResource *openstackv1.OpenStackCloud) (ctrl.Result, error) {
	logger.Info("Reconciling delete")

	if openStackResource.Spec.Credentials.Source != openstackv1.OpenStackCloudCredentialsSourceTypeSecret {
		return ctrl.Result{}, nil
	}

	logger.V(4).Info("Removing finalizer from secret")

	// Remove all our fields from the secret
	patchSecret := &corev1.Secret{}
	patchSecret.APIVersion = "v1"
	patchSecret.Kind = "Secret"
	patchSecret.Name = openStackResource.Spec.Credentials.SecretRef.Name
	patchSecret.Namespace = openStackResource.Namespace
	if err := util.Apply(ctx, r.Client, patchSecret, patchSecret); err != nil {
		return ctrl.Result{}, fmt.Errorf("removing secret finalizer: %w", err)
	}

	controllerutil.RemoveFinalizer(patchResource, openstackv1.Finalizer)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OpenStackCloudReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&openstackv1.OpenStackCloud{}).
		Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o client.Object) []reconcile.Request {
			// Fetch a list of all OpenStackClouds that reference this secret.
			kclient := mgr.GetClient()

			clouds := openstackv1.OpenStackCloudList{}
			if err := kclient.List(ctx, &clouds,
				client.InNamespace(o.GetNamespace()),
				client.MatchingLabels{openstackv1.OpenStackCloudSecretNameLabel: o.GetName()},
			); err != nil {
				mgr.GetLogger().Error(err, "unable to list OpenStackClouds")
			}

			// Reconcile each OpenStackCloud that references this secret.
			reqs := make([]reconcile.Request, len(clouds.Items))
			for i := range clouds.Items {
				cloud := clouds.Items[i]
				reqs[i] = reconcile.Request{
					NamespacedName: client.ObjectKey{
						Namespace: cloud.GetNamespace(),
						Name:      cloud.GetName(),
					},
				}
			}
			return reqs
		})).Complete(r)
}
