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
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/gophercloud/gophercloud"
	openstackv1 "github.com/gophercloud/openstack-resource-controller/api/v1alpha1"
	"github.com/gophercloud/openstack-resource-controller/pkg/apply"
	"github.com/gophercloud/openstack-resource-controller/pkg/cloud"
	"github.com/gophercloud/openstack-resource-controller/pkg/conditions"
)

const (
	OpenStackCloudFinalizer = "openstackcloud.k-orc.cloud"
)

// OpenStackCloudReconciler reconciles a OpenStackImage object
type OpenStackCloudReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

func finalizerName(cloud *openstackv1.OpenStackCloud) string {
	return OpenStackCloudFinalizer + "/" + cloud.Name
}

//+kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=openstackclouds,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=openstackclouds/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=openstackclouds/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=secrets,verbs=patch
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *OpenStackCloudReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	logger := log.FromContext(ctx)
	logger = logger.WithValues("OpenStackCloud", req.Name)
	ctx = log.IntoContext(ctx, logger)

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
	conditions.InitialiseRequiredConditions(openStackResource, patchResource)
	controllerutil.AddFinalizer(patchResource, OpenStackCloudFinalizer)

	defer func() {
		// If we're returning an error, report it as a TransientError in the Ready condition
		if reterr != nil {
			updated, condition := conditions.SetNotReadyConditionTransientError(openStackResource, patchResource, reterr.Error())

			// Emit an event if we're setting the condition for the first time
			if updated {
				conditions.EmitEventForCondition(r.Recorder, openStackResource, corev1.EventTypeWarning, condition)
			}
		}

		primaryPatch := &openstackv1.OpenStackCloud{}
		primaryPatch.TypeMeta = patchResource.TypeMeta
		primaryPatch.Labels = patchResource.Labels
		primaryPatch.Finalizers = patchResource.Finalizers

		statusPatch := &openstackv1.OpenStackCloud{}
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

	if !openStackResource.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, openStackResource, patchResource)
	}

	// Ensure the secret label is set
	if openStackResource.Spec.Credentials.Source != openstackv1.OpenStackCloudCredentialsSourceTypeSecret {
		updated, condition := conditions.SetErrorCondition(openStackResource, patchResource, openstackv1.OpenStackCloudCredentialsSourceInvalid,
			"invalid credentials source "+openStackResource.Spec.Credentials.Source)

		// Emit an event if we're setting the condition for the first time
		if updated {
			conditions.EmitEventForCondition(r.Recorder, openStackResource, corev1.EventTypeWarning, condition)
		}

		return ctrl.Result{}, nil
	}

	secretName := openStackResource.Spec.Credentials.SecretRef.Name

	// Label the cloud resource with the secret name so we can trigger a reconcile on secret changes
	patchResource.Labels = make(map[string]string)
	patchResource.Labels[openstackv1.OpenStackCloudSecretNameLabel] = secretName

	// If our finalizer isn't set, ensure it is persisted before make any changes
	if !controllerutil.ContainsFinalizer(openStackResource, OpenStackCloudFinalizer) {
		// We will be reconciled again immediately because we're adding the finalizer
		return ctrl.Result{}, nil
	}

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

			updated, condition := conditions.SetNotReadyConditionWaiting(openStackResource, patchResource, []conditions.Dependency{
				{ObjectKey: secretRef, Resource: "secret"}})

			// Emit an event if we're setting the condition for the first time
			if updated {
				conditions.EmitEventForCondition(r.Recorder, openStackResource, corev1.EventTypeNormal, condition)
			}

			return ctrl.Result{}, nil
		}
	}

	// Set our finalizer on the secret
	if !controllerutil.ContainsFinalizer(secret, finalizerName(openStackResource)) {
		logger.Info("adding finalizer to secret")

		patchSecret, err := r.getEmptySecretPatch()
		if err != nil {
			return ctrl.Result{}, err
		}
		controllerutil.AddFinalizer(patchSecret, finalizerName(openStackResource))
		if err := apply.Apply(ctx, r.Client, secret, patchSecret); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Test the credentials
	{
		_, _, err := cloud.NewProviderClient(ctx, r.Client, openStackResource)
		if err != nil {
			logger.Error(err, "validating credentials")

			switch err.(type) {
			// Set BadCredentials for any non-transient error
			case cloud.BadCredentialsError, gophercloud.ErrDefault400, gophercloud.ErrDefault401, gophercloud.ErrDefault403, gophercloud.ErrDefault404, gophercloud.ErrDefault405:
				updated, condition := conditions.SetErrorCondition(openStackResource, patchResource, conditions.OpenStackConditionReasonBadCredentials, err.Error())

				// Emit an event if we're setting the condition for the first time
				if updated {
					conditions.EmitEventForCondition(r.Recorder, openStackResource, corev1.EventTypeWarning, condition)
				}
				return ctrl.Result{}, nil
			default:
				return ctrl.Result{}, fmt.Errorf("validating credentials: %w", err)
			}
		}
	}

	// Set the Ready condition
	{
		updated, condition := conditions.SetReadyCondition(openStackResource, patchResource)

		// Emit an event if we're setting the condition for the first time
		if updated {
			conditions.EmitEventForCondition(r.Recorder, openStackResource, corev1.EventTypeNormal, condition)
		}
	}

	return ctrl.Result{}, nil
}

func (r *OpenStackCloudReconciler) getEmptySecretPatch() (*corev1.Secret, error) {
	patchSecret := &corev1.Secret{}
	gvk, err := apiutil.GVKForObject(patchSecret, r.Client.Scheme())
	if err != nil {
		return nil, fmt.Errorf("getting GVK for secret: %w", err)
	}
	patchSecret.APIVersion = gvk.GroupVersion().String()
	patchSecret.Kind = gvk.Kind
	return patchSecret, nil
}

func (r *OpenStackCloudReconciler) reconcileDelete(ctx context.Context, openStackResource, patchResource *openstackv1.OpenStackCloud) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling delete")

	logger.V(4).Info("Checking for OpenStack resources which reference this cloud")

	referencingResources := []string{}
	for _, resourceList := range []client.ObjectList{
		&openstackv1.OpenStackFlavorList{},
	} {
		list := &unstructured.UnstructuredList{}
		gvk, err := apiutil.GVKForObject(resourceList, r.Client.Scheme())
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("getting GVK for resource list: %w", err)
		}
		list.SetGroupVersionKind(gvk)
		if err := r.Client.List(ctx, list,
			client.InNamespace(openStackResource.GetNamespace()),
			client.MatchingLabels{openstackv1.OpenStackCloudLabel: openStackResource.GetName()},
			client.Limit(1),
		); err != nil {
			logger.Error(err, "unable to list resources", "type", list.GetKind())
			return ctrl.Result{}, err
		}

		if len(list.Items) > 0 {
			referencingResources = append(referencingResources, list.GetKind())
		}
	}

	if len(referencingResources) > 0 {
		logger.Info("OpenStack resources still referencing this cloud", "resources", referencingResources)

		message := fmt.Sprintf("Resources of the following types still reference this cloud: %s", strings.Join(referencingResources, ", "))
		updated, condition := conditions.SetNotReadyConditionDeleting(openStackResource, patchResource, message)

		if updated {
			conditions.EmitEventForCondition(r.Recorder, openStackResource, corev1.EventTypeWarning, condition)
		}

		// We don't have (and probably don't want) watches on every resource type, so we just have poll here
		return ctrl.Result{RequeueAfter: time.Second * 10}, nil
	}

	logger.V(4).Info("Removing finalizer from secret")

	if openStackResource.Spec.Credentials.Source != openstackv1.OpenStackCloudCredentialsSourceTypeSecret {
		return ctrl.Result{}, nil
	}

	// Remove all our fields from the secret
	patchSecret, err := r.getEmptySecretPatch()
	if err != nil {
		return ctrl.Result{}, err
	}
	patchSecret.Name = openStackResource.Spec.Credentials.SecretRef.Name
	patchSecret.Namespace = openStackResource.Namespace
	if err := apply.Apply(ctx, r.Client, patchSecret, patchSecret); err != nil {
		return ctrl.Result{}, fmt.Errorf("removing secret finalizer: %w", err)
	}

	logger.V(4).Info("Removing finalizer from cloud")

	controllerutil.RemoveFinalizer(patchResource, OpenStackCloudFinalizer)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OpenStackCloudReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&openstackv1.OpenStackCloud{}).
		WithEventFilter(apply.IgnoreManagedFieldsOnly{}).
		Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o client.Object) []reconcile.Request {
			logger := mgr.GetLogger()

			// Fetch a list of all OpenStackClouds that reference this secret.
			kclient := mgr.GetClient()

			clouds := openstackv1.OpenStackCloudList{}
			if err := kclient.List(ctx, &clouds,
				client.InNamespace(o.GetNamespace()),
				client.MatchingLabels{openstackv1.OpenStackCloudSecretNameLabel: o.GetName()},
			); err != nil {
				logger.Error(err, "unable to list OpenStackClouds")
				return nil
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
				logger.V(5).Info("update of Secret triggers reconcile of OpenStackCloud",
					"namespace", o.GetNamespace(),
					"secret", o.GetName(),
					"cloud", cloud.GetName())
			}
			return reqs
		})).
		Complete(r)
}
