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
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/imageimport"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/images"
	openstackv1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/pkg/apply"
	"github.com/k-orc/openstack-resource-controller/pkg/cloud"
	"github.com/k-orc/openstack-resource-controller/pkg/conditions"
	"github.com/k-orc/openstack-resource-controller/pkg/labels"
)

const (
	OpenStackImageFinalizer = "openstackimage.k-orc.cloud"
)

// OpenStackImageReconciler reconciles a OpenStackImage object
type OpenStackImageReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=openstackimages,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=openstackimages/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=openstackimages/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=openstackservers,verbs=list

func (r *OpenStackImageReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	logger := log.FromContext(ctx)
	logger = logger.WithValues("OpenStackImage", req.Name)

	resource := &openstackv1.OpenStackImage{}
	err := r.Client.Get(ctx, req.NamespacedName, resource)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("resource not found in the API")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if resource.DeletionTimestamp.IsZero() {
		finalizerUpdated := controllerutil.AddFinalizer(resource, OpenStackImageFinalizer)

		newLabels := map[string]string{
			openstackv1.OpenStackDependencyLabelCloud(resource.Spec.Cloud): "",
		}

		labelsMerger, labelsUpdated := labels.ReplacePrefixed(openstackv1.OpenStackLabelPrefix, resource.Labels, newLabels)

		if finalizerUpdated || labelsUpdated {
			logger.Info("applying labels and finalizer")
			patch := &openstackv1.OpenStackImage{}
			patch.TypeMeta = resource.TypeMeta
			patch.Finalizers = resource.GetFinalizers()
			patch.Labels = labelsMerger
			return ctrl.Result{}, apply.Apply(ctx, r.Client, resource, patch, "spec")
		}
	}

	statusPatchResource := &openstackv1.OpenStackImage{
		Status:   *resource.Status.DeepCopy(),
		TypeMeta: resource.TypeMeta,
	}
	defer func() {
		// If we're returning an error, report it as a TransientError in the Ready condition
		if reterr != nil {
			if updated, condition := conditions.SetNotReadyConditionTransientError(resource, statusPatchResource, reterr.Error()); updated {
				// Emit an event if we're setting the condition for the first time
				conditions.EmitEventForCondition(r.Recorder, resource, corev1.EventTypeWarning, condition)
			}
		}
		if err := apply.ApplyStatus(ctx, r.Client, resource, statusPatchResource); err != nil && !(apierrors.IsNotFound(err) && len(resource.Finalizers) == 0) {
			reterr = errors.Join(reterr, err)
		}

	}()
	if len(resource.Status.Conditions) == 0 {
		conditions.InitialiseRequiredConditions(resource, statusPatchResource)
	}

	if resource.Spec.ID == "" && resource.Spec.Resource == nil {
		if updated, condition := conditions.SetErrorCondition(resource, statusPatchResource, openstackv1.OpenStackErrorReasonInvalidSpec, "One of spec.id or spec.resource must be set"); updated {
			conditions.EmitEventForCondition(r.Recorder, resource, corev1.EventTypeNormal, condition)
		}
		return ctrl.Result{}, nil
	}

	// Get the OpenStackCloud resource
	openStackCloud := &openstackv1.OpenStackCloud{}
	{
		openStackCloudRef := client.ObjectKey{
			Namespace: req.Namespace,
			Name:      resource.Spec.Cloud,
		}
		err := r.Client.Get(ctx, openStackCloudRef, openStackCloud)
		if err != nil && !apierrors.IsNotFound(err) {
			return ctrl.Result{}, fmt.Errorf("fetching OpenStackCloud %s: %w", resource.Spec.Cloud, err)
		}

		// XXX(mbooth): We should check IsReady(openStackCloud) here, but we can't because this breaks us while the cloud is Deleting.
		// We probably need another Condition 'Deleting' so an object can be both Ready and Deleting during the cleanup phase.
		if err != nil {
			conditions.SetNotReadyConditionWaiting(resource, statusPatchResource, []conditions.Dependency{
				{ObjectKey: openStackCloudRef, Resource: "OpenStackCloud"},
			})
			return ctrl.Result{}, nil
		}
	}

	imageClient, err := cloud.NewServiceClient(ctx, r.Client, openStackCloud, "image")
	if err != nil {
		err = fmt.Errorf("unable to build an OpenStack client: %w", err)
		logger.Info(err.Error())
		return ctrl.Result{}, err
	}

	if !resource.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(log.IntoContext(ctx, logger), imageClient, resource, statusPatchResource)
	}

	return r.reconcile(log.IntoContext(ctx, logger), imageClient, resource, statusPatchResource)
}

// reconcile handles creation. No modification is accepted.
// TODO: restrict unhandled modification through a webhook
// TODO: potentially handle (some?) modifications accepted in OpenStack
func (r *OpenStackImageReconciler) reconcile(ctx context.Context, imageClient *gophercloud.ServiceClient, resource, statusPatchResource *openstackv1.OpenStackImage) (_ ctrl.Result, reterr error) {
	logger := log.FromContext(ctx)

	var (
		image *images.Image
		err   error
	)
	if openstackID := coalesce(resource.Spec.ID, resource.Status.Resource.ID); openstackID != "" {
		logger = logger.WithValues("OpenStackID", openstackID)

		image, err = images.Get(imageClient, openstackID).Extract()
		if err != nil {
			return ctrl.Result{}, err
		}
		logger.Info("OpenStack resource found")
	} else {
		id := resource.ComputedSpecID()
		image, err = r.findOrphan(log.IntoContext(ctx, logger), imageClient, resource, id)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("unable to find adoption candidates: %w", err)
		}
		if image != nil {
			logger = logger.WithValues("OpenStackID", image.ID)
			logger.Info("OpenStack resource adopted")
		} else {
			var imageVisibility *images.ImageVisibility
			if visibility := resource.Spec.Resource.Visibility; visibility != nil {
				v := images.ImageVisibility(*visibility)
				imageVisibility = &v
			}
			image, err = images.Create(imageClient, images.CreateOpts{
				ID:              id,
				Name:            resource.Spec.Resource.Name,
				Tags:            append(resource.Spec.Resource.Tags, orcTag(resource)),
				ContainerFormat: resource.Spec.Resource.ContainerFormat,
				DiskFormat:      resource.Spec.Resource.DiskFormat,
				MinDisk:         resource.Spec.Resource.MinDisk,
				MinRAM:          resource.Spec.Resource.MinRAM,
				Protected:       resource.Spec.Resource.Protected,
				Visibility:      imageVisibility,
			}).Extract()
			if err != nil {
				return ctrl.Result{}, err
			}
			logger = logger.WithValues("OpenStackID", image.ID)
			logger.Info("OpenStack resource created")
		}
	}

	properties := make(map[string]string)
	for k, v := range image.Properties {
		properties[k] = fmt.Sprint(v)
	}

	statusPatchResource.Status.Resource = openstackv1.OpenStackImageResourceStatus{
		ID:              image.ID,
		Name:            image.Name,
		Status:          string(image.Status),
		Tags:            image.Tags,
		ContainerFormat: image.ContainerFormat,
		DiskFormat:      image.DiskFormat,
		MinDisk:         image.MinDiskGigabytes,
		MinRAM:          image.MinRAMMegabytes,
		Owner:           image.Owner,
		Protected:       image.Protected,
		Visibility:      string(image.Visibility),
		Hidden:          image.Hidden,
		Checksum:        image.Checksum,
		Size:            image.SizeBytes,
		Metadata:        image.Metadata,
		Properties:      properties,
		UpdatedAt:       image.UpdatedAt.UTC().Format(time.RFC3339),
		CreatedAt:       image.CreatedAt.UTC().Format(time.RFC3339),
		File:            image.File,
		Schema:          image.Schema,
		VirtualSize:     image.VirtualSize,
		ImportMethods:   image.OpenStackImageImportMethods,
		StoreIDs:        image.OpenStackImageStoreIDs,
	}

	switch image.Status {
	case images.ImageStatusImporting, images.ImageStatusSaving, "uploading":
		if updated, condition := conditions.SetNotReadyConditionPending(resource, statusPatchResource); updated {
			conditions.EmitEventForCondition(r.Recorder, resource, corev1.EventTypeNormal, condition)
		}
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	case images.ImageStatusQueued, images.ImageStatusKilled:
		if resource.Spec.Resource.Method == string(imageimport.WebDownloadMethod) && resource.Spec.Resource.WebDownload != nil {
			if err := imageimport.Create(imageClient, image.ID, imageimport.CreateOpts{
				Name: imageimport.WebDownloadMethod,
				URI:  resource.Spec.Resource.WebDownload.URL,
			}).ExtractErr(); err != nil {
				return ctrl.Result{}, err
			}
			if updated, condition := conditions.SetNotReadyConditionPending(resource, statusPatchResource); updated {
				conditions.EmitEventForCondition(r.Recorder, resource, corev1.EventTypeNormal, condition)
			}
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, nil
	case images.ImageStatusDeactivated, images.ImageStatusPendingDelete, images.ImageStatusDeleted:
		if updated, condition := conditions.SetErrorCondition(resource, statusPatchResource, openstackv1.OpenStackErrorReasonImageNotAvailable, "Image "+string(image.Status)); updated {
			conditions.EmitEventForCondition(r.Recorder, resource, corev1.EventTypeNormal, condition)
		}
		return ctrl.Result{}, nil
	case images.ImageStatusActive:
		if expected := resource.Status.Resource.Checksum; expected != "" && expected != image.Checksum {
			if updated, condition := conditions.SetErrorCondition(resource, statusPatchResource, openstackv1.OpenStackErrorReasonImageImportFailed, "checksum mismatch"); updated {
				conditions.EmitEventForCondition(r.Recorder, resource, corev1.EventTypeNormal, condition)
			}
			return ctrl.Result{}, nil
		}
		if updated, condition := conditions.SetReadyCondition(resource, statusPatchResource); updated {
			conditions.EmitEventForCondition(r.Recorder, resource, corev1.EventTypeNormal, condition)
		}
		return ctrl.Result{}, nil
	default:
		return ctrl.Result{}, fmt.Errorf("unknown status %q", image.Status)
	}
}

func (r *OpenStackImageReconciler) reconcileDelete(ctx context.Context, imageClient *gophercloud.ServiceClient, resource, statusPatchResource *openstackv1.OpenStackImage) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.V(4).Info("Checking for dependant OpenStack resources")
	referencingResources := []string{}
	for _, resourceList := range []client.ObjectList{
		&openstackv1.OpenStackServerList{},
	} {
		list := &unstructured.UnstructuredList{}
		gvk, err := apiutil.GVKForObject(resourceList, r.Client.Scheme())
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("getting GVK for resource list: %w", err)
		}
		list.SetGroupVersionKind(gvk)
		if err := r.Client.List(ctx, list,
			client.InNamespace(resource.GetNamespace()),
			client.HasLabels{openstackv1.OpenStackDependencyLabelImage(resource.GetName())},
			client.Limit(1),
		); err != nil {
			logger.Error(err, "unable to list resources", "type", list.GetKind())
			return ctrl.Result{}, err
		}

		if len(list.Items) > 0 {
			referencingResources = append(referencingResources, list.Items[0].GetKind())
		}
	}

	if len(referencingResources) > 0 {
		logger.Info("OpenStack resources still referencing this image", "resources", referencingResources)

		message := fmt.Sprintf("Resources of the following types still reference this image: %s", strings.Join(referencingResources, ", "))
		if updated, condition := conditions.SetNotReadyConditionDeleting(resource, statusPatchResource, message); updated {
			conditions.EmitEventForCondition(r.Recorder, resource, corev1.EventTypeWarning, condition)
		}

		// We don't have (and probably don't want) watches on every resource type, so we just have poll here
		return ctrl.Result{RequeueAfter: time.Second * 10}, nil
	}

	if resource.Status.Resource.ID == "" {
		logger.Info("deletion was requested on a resource that hasn't been successfully created or adopted yet.")
	} else {
		logger = logger.WithValues("OpenStackID", resource.Status.Resource.ID)
		if !resource.Spec.Unmanaged {
			if err := images.Delete(imageClient, resource.Status.Resource.ID).ExtractErr(); err != nil {
				var gerr gophercloud.ErrDefault404
				if errors.As(err, &gerr) {
					logger.Info("deletion was requested on a resource that can't be found in OpenStack.")
				} else {
					logger.Info("failed to delete resource in OpenStack; requeuing.")
					return ctrl.Result{}, err
				}
			}
		}
	}

	if updated := controllerutil.RemoveFinalizer(resource, OpenStackImageFinalizer); updated {
		logger.Info("removing finalizer")
		if updated, condition := conditions.SetNotReadyConditionDeleting(resource, statusPatchResource, "Removing finalizer"); updated {
			conditions.EmitEventForCondition(r.Recorder, resource, corev1.EventTypeNormal, condition)
		}
		patch := &openstackv1.OpenStackImage{}
		patch.TypeMeta = resource.TypeMeta
		patch.Finalizers = resource.GetFinalizers()
		return ctrl.Result{}, apply.Apply(ctx, r.Client, resource, patch, "spec")
	}
	return ctrl.Result{}, nil
}

func (r *OpenStackImageReconciler) findOrphan(ctx context.Context, imageClient *gophercloud.ServiceClient, resource client.Object, id string) (*images.Image, error) {
	{
		list := &openstackv1.OpenStackImageList{}
		if err := r.Client.List(ctx, list,
			client.InNamespace(resource.GetNamespace()),
		); err != nil {
			return nil, fmt.Errorf("listing OpenStackImages: %w", err)
		}
		for _, image := range list.Items {
			if image.GetName() != resource.GetName() && image.Status.Resource.ID == id {
				return nil, nil
			}
		}
	}

	image, err := images.Get(imageClient, id).Extract()
	if err != nil {
		var gerr gophercloud.ErrDefault404
		if errors.As(err, &gerr) {
			return nil, nil
		} else {
			return nil, err
		}
	}

	return image, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OpenStackImageReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&openstackv1.OpenStackImage{}).
		WithEventFilter(apply.IgnoreManagedFieldsOnly{}).
		Watches(&openstackv1.OpenStackCloud{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o client.Object) []reconcile.Request {
			// Fetch a list of all OpenStackImages that reference this OpenStackCloud.
			kclient := mgr.GetClient()
			logger := mgr.GetLogger()

			images := &openstackv1.OpenStackImageList{}
			if err := kclient.List(ctx, images,
				client.InNamespace(o.GetNamespace()),
				client.HasLabels{openstackv1.OpenStackDependencyLabelCloud(o.GetName())},
			); err != nil {
				logger.Error(err, "unable to list OpenStackImages")
				return nil
			}

			// Reconcile each OpenStackImage that is not Ready and that references this OpenStackCloud.
			reqs := make([]reconcile.Request, 0, len(images.Items))
			for _, image := range images.Items {
				if conditions.IsReady(&image) {
					continue
				}
				reqs = append(reqs, reconcile.Request{
					NamespacedName: client.ObjectKey{
						Namespace: image.GetNamespace(),
						Name:      image.GetName(),
					},
				})
				logger.V(5).Info("update of OpenStackCloud triggers reconcile of OpenStackImage",
					"namespace", o.GetNamespace(),
					"cloud", o.GetName(),
					"image", image.GetName())
			}
			return reqs
		})).
		Complete(r)
}
