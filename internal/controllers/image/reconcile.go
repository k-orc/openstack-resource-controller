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

package image

import (
	"context"
	"errors"
	"fmt"

	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/images"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/generic"
	osclients "github.com/k-orc/openstack-resource-controller/internal/osclients"
	orcerrors "github.com/k-orc/openstack-resource-controller/internal/util/errors"
)

// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=images,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=images/status,verbs=get;update;patch

func (r *orcImageReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	orcImage := &orcv1alpha1.Image{}
	err := r.client.Get(ctx, req.NamespacedName, orcImage)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if !orcImage.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, orcImage)
	}

	return r.reconcileNormal(ctx, orcImage)
}

func (r *orcImageReconciler) reconcileNormal(ctx context.Context, orcObject *orcv1alpha1.Image) (_ ctrl.Result, err error) {
	log := ctrl.LoggerFrom(ctx)
	log.V(3).Info("Reconciling image")

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

	actuator, err := newActuator(ctx, r, orcObject)
	if err != nil {
		return ctrl.Result{}, err
	}

	adapter := imageAdapter{orcObject}
	waitEvents, osResource, err := generic.GetOrCreateOSResource2(ctx, log, r, adapter, actuator)
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

	log = log.WithValues("ID", osResource.ID)
	log.V(4).Info("Got resource")
	ctx = ctrl.LoggerInto(ctx, log)

	return r.handleImageUpload(ctx, actuator.osClient, orcObject, osResource, addStatus)
}

func (r *orcImageReconciler) handleImageUpload(ctx context.Context, imageClient osclients.ImageClient, orcImage *orcv1alpha1.Image, glanceImage *images.Image, addStatus func(updateStatusOpt)) (_ ctrl.Result, err error) {
	switch glanceImage.Status {
	// Cases where we're not going to take any action until the next resync
	case images.ImageStatusActive, images.ImageStatusDeactivated:
		return ctrl.Result{}, nil

	// Content is being saved. Check back in a minute
	// "importing" is seen during web-download
	// "saving" is seen while uploading, but might be seen because our upload failed and glance hasn't reset yet.
	case images.ImageStatusImporting, images.ImageStatusSaving:
		addStatus(withProgressMessage(downloadingMessage("Glance is downloading image content", orcImage)))
		return ctrl.Result{RequeueAfter: externalUpdatePollingPeriod}, nil

	// Newly created image, waiting for upload, or... previous upload was interrupted and has now reset
	case images.ImageStatusQueued:
		// Don't attempt image creation if we're not managing the image
		if orcImage.Spec.ManagementPolicy == orcv1alpha1.ManagementPolicyUnmanaged {
			addStatus(withProgressMessage("Waiting for glance image content to be uploaded externally"))

			return ctrl.Result{
				RequeueAfter: externalUpdatePollingPeriod,
			}, err
		}

		if ptr.Deref(orcImage.Status.DownloadAttempts, 0) >= maxDownloadAttempts {
			return ctrl.Result{}, orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, fmt.Sprintf("Unable to download content after %d attempts", maxDownloadAttempts))
		}

		canWebDownload, err := r.canWebDownload(ctx, orcImage, imageClient)
		if err != nil {
			return ctrl.Result{}, err
		}

		if canWebDownload {
			// We frequently hit a race with glance here. There is a
			// delay after doing an import before glance updates the
			// status from queued, meaning we frequently attempt to
			// start a second import. Although the status isn't
			// updated yet, glance still returns a 409 error when
			// this happens due to the existing task. This is
			// harmless.

			err := r.webDownload(ctx, orcImage, imageClient, glanceImage)
			if err != nil {
				return ctrl.Result{}, err
			}

			// Don't increment DownloadAttempts unless webDownload returned success
			addStatus(withIncrementDownloadAttempts())

			return ctrl.Result{}, nil
		} else {
			return ctrl.Result{}, r.uploadImageContent(ctx, orcImage, imageClient, glanceImage)
		}

	// Error cases
	case images.ImageStatusKilled:
		return ctrl.Result{}, orcerrors.Terminal(orcv1alpha1.ConditionReasonUnrecoverableError, "a glance error occurred while saving image content")
	case images.ImageStatusDeleted, images.ImageStatusPendingDelete:
		return ctrl.Result{}, orcerrors.Terminal(orcv1alpha1.ConditionReasonUnrecoverableError, "image status is deleting")
	default:
		return ctrl.Result{}, errors.New("unknown image status: " + string(glanceImage.Status))
	}
}

func (r *orcImageReconciler) reconcileDelete(ctx context.Context, orcObject *orcv1alpha1.Image) (_ ctrl.Result, err error) {
	log := ctrl.LoggerFrom(ctx)
	log.V(3).Info("Reconciling image delete")

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
		return ctrl.Result{}, err
	}

	adapter := imageAdapter{orcObject}
	deleted, waitEvents, osResource, err := generic.DeleteResource2(ctx, log, r, adapter, actuator)
	addStatus(withResource(osResource))
	return ctrl.Result{RequeueAfter: generic.MaxRequeue(waitEvents)}, err
}

func downloadingMessage(msg string, orcImage *orcv1alpha1.Image) string {
	if ptr.Deref(orcImage.Status.DownloadAttempts, 0) > 1 {
		return fmt.Sprintf("%s: attempt %d", msg, *orcImage.Status.DownloadAttempts)
	}
	return msg
}
