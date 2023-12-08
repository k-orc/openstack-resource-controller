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
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/pagination"
	openstackv1 "github.com/gophercloud/openstack-resource-controller/api/v1alpha1"
	"github.com/gophercloud/openstack-resource-controller/pkg/apply"
	"github.com/gophercloud/openstack-resource-controller/pkg/cloud"
	"github.com/gophercloud/openstack-resource-controller/pkg/conditions"
	"github.com/gophercloud/openstack-resource-controller/pkg/labels"
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

	resource := &openstackv1.OpenStackFlavor{}
	err := r.Client.Get(ctx, req.NamespacedName, resource)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("resource not found in the API")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if resource.DeletionTimestamp.IsZero() {
		finalizerUpdated := controllerutil.AddFinalizer(resource, OpenStackFlavorFinalizer)

		newLabels := map[string]string{
			openstackv1.OpenStackDependencyLabelCloud(resource.Spec.Cloud): "",
		}

		labelsMerger, labelsUpdated := labels.ReplacePrefixed(openstackv1.OpenStackLabelPrefix, resource.Labels, newLabels)

		if finalizerUpdated || labelsUpdated {
			logger.Info("applying labels and finalizer")
			patch := &openstackv1.OpenStackFlavor{}
			patch.TypeMeta = resource.TypeMeta
			patch.Finalizers = resource.GetFinalizers()
			patch.Labels = labelsMerger
			return ctrl.Result{}, apply.Apply(ctx, r.Client, resource, patch, "spec")
		}
	}

	statusPatchResource := &openstackv1.OpenStackFlavor{
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

	computeClient, err := cloud.NewServiceClient(ctx, r.Client, openStackCloud, "compute")
	if err != nil {
		err = fmt.Errorf("unable to build an OpenStack client: %w", err)
		logger.Info(err.Error())
		return ctrl.Result{}, err
	}

	if !resource.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(log.IntoContext(ctx, logger), computeClient, resource, statusPatchResource)
	}

	return r.reconcile(log.IntoContext(ctx, logger), computeClient, resource, statusPatchResource)
}

// reconcile handles creation. No modification is accepted.
// TODO: restrict unhandled modification through a webhook
// TODO: potentially handle (some?) modifications accepted in OpenStack
func (r *OpenStackFlavorReconciler) reconcile(ctx context.Context, computeClient *gophercloud.ServiceClient, resource, statusPatchResource *openstackv1.OpenStackFlavor) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var (
		flavor *flavors.Flavor
		err    error
	)
	if openstackID := coalesce(resource.Spec.ID, resource.Status.Resource.ID); openstackID != "" {
		logger = logger.WithValues("OpenStackID", openstackID)

		flavor, err = flavors.Get(computeClient, openstackID).Extract()
		if err != nil {
			return ctrl.Result{}, err
		}
		logger.Info("OpenStack resource found")
	} else {
		var rxtxFactor float64
		if resource.Spec.Resource.RxTxFactor != "" {
			rxtxFactor, err = strconv.ParseFloat(resource.Spec.Resource.RxTxFactor, 64)
			if err != nil {
				conditions.SetErrorCondition(resource, statusPatchResource, openstackv1.OpenStackErrorReasonInvalidSpec, "error parsing rxtxFactor: "+err.Error())
				return ctrl.Result{}, nil
			}
		}

		createOpts := flavors.CreateOpts{
			ID:          resource.Spec.Resource.ID,
			Name:        resource.Spec.Resource.Name,
			RAM:         resource.Spec.Resource.RAM,
			VCPUs:       resource.Spec.Resource.VCPUs,
			Disk:        &resource.Spec.Resource.Disk,
			Swap:        &resource.Spec.Resource.Swap,
			RxTxFactor:  rxtxFactor,
			IsPublic:    resource.Spec.Resource.IsPublic,
			Ephemeral:   &resource.Spec.Resource.Ephemeral,
			Description: resource.Spec.Resource.Description,
		}

		flavor, err = r.findAdoptee(log.IntoContext(ctx, logger), computeClient, resource, createOpts)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("unable to find adoption candidates: %w", err)
		}
		if flavor != nil {
			logger = logger.WithValues("OpenStackID", flavor.ID)
			logger.Info("OpenStack resource adopted")
		} else {
			flavor, err = flavors.Create(computeClient, createOpts).Extract()
			if err != nil {
				return ctrl.Result{}, err
			}
			logger = logger.WithValues("OpenStackID", flavor.ID)
			logger.Info("OpenStack resource created")
		}
	}

	statusPatchResource.Status.Resource = openstackv1.OpenStackFlavorResourceStatus{
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

	if updated, condition := conditions.SetReadyCondition(resource, statusPatchResource); updated {
		conditions.EmitEventForCondition(r.Recorder, resource, corev1.EventTypeNormal, condition)
	}
	return ctrl.Result{}, nil
}

func (r *OpenStackFlavorReconciler) reconcileDelete(ctx context.Context, computeClient *gophercloud.ServiceClient, resource, statusPatchResource *openstackv1.OpenStackFlavor) (ctrl.Result, error) {
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
			client.HasLabels{openstackv1.OpenStackDependencyLabelFlavor(resource.GetName())},
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
		logger.Info("OpenStack resources still referencing this flavor", "resources", referencingResources)

		message := fmt.Sprintf("Resources of the following types still reference this flavor: %s", strings.Join(referencingResources, ", "))
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

	if updated := controllerutil.RemoveFinalizer(resource, OpenStackFlavorFinalizer); updated {
		logger.Info("removing finalizer")
		if updated, condition := conditions.SetNotReadyConditionDeleting(resource, statusPatchResource, "Removing finalizer"); updated {
			conditions.EmitEventForCondition(r.Recorder, resource, corev1.EventTypeNormal, condition)
		}
		patch := &openstackv1.OpenStackFlavor{}
		patch.TypeMeta = resource.TypeMeta
		patch.Finalizers = resource.GetFinalizers()
		return ctrl.Result{}, apply.Apply(ctx, r.Client, resource, patch, "spec")
	}
	return ctrl.Result{}, nil
}

func flavorEquals(candidate flavors.Flavor, resource flavors.CreateOpts) bool {
	if candidate.VCPUs != resource.VCPUs {
		return false
	}
	if resource.ID != "" && resource.ID != candidate.ID {
		return false
	}
	if candidate.Name != resource.Name {
		return false
	}
	if candidate.RAM != resource.RAM {
		return false
	}
	if candidate.Disk != pointer.IntDeref(resource.Disk, 0) {
		return false
	}
	if candidate.Swap != pointer.IntDeref(resource.Swap, 0) {
		return false
	}
	if resource.RxTxFactor != 0 && candidate.RxTxFactor != resource.RxTxFactor {
		return false
	}
	if candidate.Ephemeral != pointer.IntDeref(resource.Ephemeral, 0) {
		return false
	}
	if candidate.Description != resource.Description {
		return false
	}
	return true
}

func (r *OpenStackFlavorReconciler) findAdoptee(ctx context.Context, computeClient *gophercloud.ServiceClient, resource client.Object, createOpts flavors.CreateOpts) (*flavors.Flavor, error) {
	adoptedIDs := make(map[string]struct{})
	{
		list := &openstackv1.OpenStackFlavorList{}
		if err := r.Client.List(ctx, list,
			client.InNamespace(resource.GetNamespace()),
		); err != nil {
			return nil, fmt.Errorf("listing OpenStackFlavors: %w", err)
		}
		for _, item := range list.Items {
			if item.GetName() != resource.GetName() && item.Status.Resource.ID != "" {
				adoptedIDs[item.Status.Resource.ID] = struct{}{}
			}
		}
	}

	var candidates []flavors.Flavor
	err := flavors.ListDetail(computeClient, flavors.ListOpts{
		MinDisk: pointer.IntDeref(createOpts.Disk, 0),
		MinRAM:  createOpts.RAM,
		SortDir: "asc",
		SortKey: "vcpus",
	}).EachPage(func(page pagination.Page) (bool, error) {
		items, err := flavors.ExtractFlavors(page)
		if err != nil {
			return false, fmt.Errorf("extracting resources: %w", err)
		}
		for i := range items {
			if _, ok := adoptedIDs[items[i].ID]; !ok && flavorEquals(items[i], createOpts) {
				candidates = append(candidates, items[i])
			}
		}

		// The list is requested in ascending number of VCPUs; stop
		// listing when the number of VCPUs surpasses our target
		// flavor's.
		return items[len(items)-1].VCPUs == createOpts.VCPUs, nil
	})
	if err != nil {
		return nil, err
	}

	switch n := len(candidates); n {
	case 0:
		return nil, nil
	case 1:
		return &candidates[0], nil
	default:
		return nil, fmt.Errorf("found %d possible candidates", n)
	}
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

			flavors := &openstackv1.OpenStackFlavorList{}
			if err := kclient.List(ctx, flavors,
				client.InNamespace(o.GetNamespace()),
				client.HasLabels{openstackv1.OpenStackDependencyLabelCloud(o.GetName())},
			); err != nil {
				logger.Error(err, "unable to list OpenStackClouds")
				return nil
			}

			// Reconcile each OpenStackFlavor that is not Ready and that references this OpenStackCloud.
			reqs := make([]reconcile.Request, 0, len(flavors.Items))
			for _, flavor := range flavors.Items {
				if conditions.IsReady(&flavor) {
					continue
				}
				reqs = append(reqs, reconcile.Request{
					NamespacedName: client.ObjectKey{
						Namespace: flavor.GetNamespace(),
						Name:      flavor.GetName(),
					},
				})
				logger.V(5).Info("update of OpenStackCloud triggers reconcile of OpenStackFlavor",
					"namespace", o.GetNamespace(),
					"cloud", o.GetName(),
					"flavor", flavor.GetName())
			}
			return reqs
		})).
		Complete(r)
}
