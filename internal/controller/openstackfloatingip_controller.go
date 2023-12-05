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
	"time"

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

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/floatingips"
	"github.com/gophercloud/gophercloud/pagination"
	openstackv1 "github.com/gophercloud/openstack-resource-controller/api/v1alpha1"
	"github.com/gophercloud/openstack-resource-controller/pkg/apply"
	"github.com/gophercloud/openstack-resource-controller/pkg/cloud"
	"github.com/gophercloud/openstack-resource-controller/pkg/conditions"
	"github.com/gophercloud/openstack-resource-controller/pkg/labels"
)

const (
	OpenStackFloatingIPFinalizer = "openstackfloatingip.k-orc.cloud"
)

// OpenStackFloatingIPReconciler reconciles a OpenStackFloatingIP object
type OpenStackFloatingIPReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=openstackfloatingips,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=openstackfloatingips/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=openstackfloatingips/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *OpenStackFloatingIPReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	logger := log.FromContext(ctx)
	logger = logger.WithValues("OpenStackFloatingIP", req.Name)

	resource := &openstackv1.OpenStackFloatingIP{}
	err := r.Client.Get(ctx, req.NamespacedName, resource)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("resource not found in the API")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if resource.DeletionTimestamp.IsZero() {
		finalizerUpdated := controllerutil.AddFinalizer(resource, OpenStackFloatingIPFinalizer)

		newLabels := map[string]string{
			openstackv1.OpenStackDependencyLabelCloud(resource.Spec.Cloud):                      "",
			openstackv1.OpenStackDependencyLabelNetwork(resource.Spec.Resource.FloatingNetwork): "",
		}
		if port := resource.Spec.Resource.Port; port != "" {
			newLabels[openstackv1.OpenStackDependencyLabelPort(port)] = ""
		}
		if subnet := resource.Spec.Resource.Subnet; subnet != "" {
			newLabels[openstackv1.OpenStackDependencyLabelSubnet(subnet)] = ""
		}

		labelsMerger, labelsUpdated := labels.ReplacePrefixed(openstackv1.OpenStackLabelPrefix, resource.Labels, newLabels)

		if finalizerUpdated || labelsUpdated {
			logger.Info("applying labels and finalizer")
			patch := &openstackv1.OpenStackFloatingIP{}
			patch.TypeMeta = resource.TypeMeta
			patch.Finalizers = resource.GetFinalizers()
			patch.Labels = labelsMerger
			return ctrl.Result{}, apply.Apply(ctx, r.Client, resource, patch, "spec")
		}
	}

	statusPatchResource := &openstackv1.OpenStackFloatingIP{
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
		if updated, condition := conditions.SetErrorCondition(resource, statusPatchResource, "BadRequest", "One of spec.id or spec.resource must be set"); updated {
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

	networkClient, err := cloud.NewServiceClient(ctx, r.Client, openStackCloud, "network")
	if err != nil {
		err = fmt.Errorf("unable to build an OpenStack client: %w", err)
		logger.Info(err.Error())
		return ctrl.Result{}, err
	}

	if !resource.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(log.IntoContext(ctx, logger), networkClient, resource, statusPatchResource)
	}

	return r.reconcile(log.IntoContext(ctx, logger), networkClient, resource, statusPatchResource)
}

// reconcile handles creation. No modification is accepted.
// TODO: restrict unhandled modification through a webhook
// TODO: potentially handle (some?) modifications accepted in OpenStack, as in `openstack network set`
func (r *OpenStackFloatingIPReconciler) reconcile(ctx context.Context, networkClient *gophercloud.ServiceClient, resource, statusPatchResource *openstackv1.OpenStackFloatingIP) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var (
		floatingIP *floatingips.FloatingIP
		err        error
	)
	if openstackID := coalesce(resource.Spec.ID, resource.Status.Resource.ID); openstackID != "" {
		logger = logger.WithValues("OpenStackID", openstackID)

		floatingIP, err = floatingips.Get(networkClient, openstackID).Extract()
		if err != nil {
			return ctrl.Result{}, err
		}
		logger.Info("OpenStack resource found")
	} else {
		var networkID string
		{
			dependency := &openstackv1.OpenStackNetwork{}
			dependencyKey := client.ObjectKey{Namespace: resource.GetNamespace(), Name: resource.Spec.Resource.FloatingNetwork}
			err = r.Client.Get(ctx, dependencyKey, dependency)
			if err != nil && !apierrors.IsNotFound(err) {
				return ctrl.Result{}, err
			}

			// Dependency either doesn't exist, or is being deleted, or is not ready
			if err != nil || !dependency.DeletionTimestamp.IsZero() || !conditions.IsReady(dependency) || dependency.Status.Resource.ID == "" {
				logger.Info("waiting for network")

				if updated, condition := conditions.SetNotReadyConditionWaiting(resource, statusPatchResource, []conditions.Dependency{
					{ObjectKey: dependencyKey, Resource: "network"},
				}); updated {
					// Emit an event if we're setting the condition for the first time
					conditions.EmitEventForCondition(r.Recorder, resource, corev1.EventTypeNormal, condition)
				}
				return ctrl.Result{}, nil
			}
			networkID = dependency.Status.Resource.ID
		}

		var subnetID string
		if subnetName := resource.Spec.Resource.Subnet; subnetName != "" {
			dependency := &openstackv1.OpenStackSubnet{}
			dependencyKey := client.ObjectKey{Namespace: resource.GetNamespace(), Name: subnetName}
			err = r.Client.Get(ctx, dependencyKey, dependency)
			if err != nil && !apierrors.IsNotFound(err) {
				return ctrl.Result{}, err
			}

			// Dependency either doesn't exist, or is being deleted, or is not ready
			if err != nil || !dependency.DeletionTimestamp.IsZero() || !conditions.IsReady(dependency) || dependency.Status.Resource.ID == "" {
				logger.Info("waiting for subnet")

				if updated, condition := conditions.SetNotReadyConditionWaiting(resource, statusPatchResource, []conditions.Dependency{
					{ObjectKey: dependencyKey, Resource: "subnet"},
				}); updated {
					// Emit an event if we're setting the condition for the first time
					conditions.EmitEventForCondition(r.Recorder, resource, corev1.EventTypeNormal, condition)
				}
				return ctrl.Result{}, nil
			}

			subnetID = dependency.Status.Resource.ID
		}

		var portID string
		if portName := resource.Spec.Resource.Port; portName != "" {
			dependency := &openstackv1.OpenStackPort{}
			dependencyKey := client.ObjectKey{Namespace: resource.GetNamespace(), Name: portName}
			err = r.Client.Get(ctx, dependencyKey, dependency)
			if err != nil && !apierrors.IsNotFound(err) {
				return ctrl.Result{}, err
			}

			// Dependency either doesn't exist, or is being deleted, or is not ready
			if err != nil || !dependency.DeletionTimestamp.IsZero() || !conditions.IsReady(dependency) || dependency.Status.Resource.ID == "" {
				logger.Info("waiting for port")

				if updated, condition := conditions.SetNotReadyConditionWaiting(resource, statusPatchResource, []conditions.Dependency{
					{ObjectKey: dependencyKey, Resource: "port"},
				}); updated {
					// Emit an event if we're setting the condition for the first time
					conditions.EmitEventForCondition(r.Recorder, resource, corev1.EventTypeNormal, condition)
				}
				return ctrl.Result{}, nil
			}
			portID = dependency.Status.Resource.ID
		}

		createOpts := floatingips.CreateOpts{
			Description:       resource.Spec.Resource.Description,
			FloatingNetworkID: networkID,
			FloatingIP:        resource.Spec.Resource.FloatingIPAddress,
			PortID:            portID,
			FixedIP:           resource.Spec.Resource.FixedIPAddress,
			SubnetID:          subnetID,
			TenantID:          resource.Spec.Resource.TenantID,
			ProjectID:         resource.Spec.Resource.ProjectID,
		}

		floatingIP, err = r.floatingIPFind(log.IntoContext(ctx, logger), networkClient, resource, createOpts)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("unable to find adoption candidates: %w", err)
		}
		if floatingIP != nil {
			logger = logger.WithValues("OpenStackID", floatingIP.ID)
			logger.Info("OpenStack resource adopted")
		} else {
			floatingIP, err = floatingips.Create(networkClient, createOpts).Extract()
			if err != nil {
				return ctrl.Result{}, err
			}
			logger = logger.WithValues("OpenStackID", floatingIP.ID)
			logger.Info("OpenStack resource created")
		}
	}

	statusPatchResource.Status.Resource = openstackv1.OpenStackFloatingIPResourceStatus{
		ID:                floatingIP.ID,
		Description:       floatingIP.Description,
		FloatingNetworkID: floatingIP.FloatingNetworkID,
		FloatingIP:        floatingIP.FloatingIP,
		PortID:            floatingIP.PortID,
		FixedIP:           floatingIP.FixedIP,
		TenantID:          floatingIP.TenantID,
		UpdatedAt:         floatingIP.UpdatedAt.UTC().Format(time.RFC3339),
		CreatedAt:         floatingIP.CreatedAt.UTC().Format(time.RFC3339),
		ProjectID:         floatingIP.ProjectID,
		Status:            floatingIP.Status,
		RouterID:          floatingIP.RouterID,
		Tags:              floatingIP.Tags,
	}

	if updated, condition := conditions.SetReadyCondition(resource, statusPatchResource); updated {
		conditions.EmitEventForCondition(r.Recorder, resource, corev1.EventTypeNormal, condition)
	}
	return ctrl.Result{}, nil
}

func (r *OpenStackFloatingIPReconciler) reconcileDelete(ctx context.Context, networkClient *gophercloud.ServiceClient, resource, statusPatchResource *openstackv1.OpenStackFloatingIP) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if resource.Status.Resource.ID == "" {
		logger.Info("deletion was requested on a resource that hasn't been successfully created or adopted yet.")
	} else {
		logger = logger.WithValues("OpenStackID", resource.Status.Resource.ID)
		if !resource.Spec.Unmanaged {
			if err := floatingips.Delete(networkClient, resource.Status.Resource.ID).ExtractErr(); err != nil {
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

	if updated := controllerutil.RemoveFinalizer(resource, OpenStackFloatingIPFinalizer); updated {
		logger.Info("removing finalizer")
		if updated, condition := conditions.SetNotReadyConditionDeleting(resource, statusPatchResource, "Removing finalizer"); updated {
			conditions.EmitEventForCondition(r.Recorder, resource, corev1.EventTypeNormal, condition)
		}
		patch := &openstackv1.OpenStackFloatingIP{}
		patch.TypeMeta = resource.TypeMeta
		patch.Finalizers = resource.GetFinalizers()
		return ctrl.Result{}, apply.Apply(ctx, r.Client, resource, patch, "spec")
	}
	return ctrl.Result{}, nil
}

func (r *OpenStackFloatingIPReconciler) floatingIPFind(ctx context.Context, networkClient *gophercloud.ServiceClient, resource client.Object, createOpts floatingips.CreateOpts) (*floatingips.FloatingIP, error) {
	adoptedIDs := make(map[string]struct{})
	{
		floatingIPs := &openstackv1.OpenStackFloatingIPList{}
		if err := r.Client.List(ctx, floatingIPs,
			client.InNamespace(resource.GetNamespace()),
		); err != nil {
			return nil, fmt.Errorf("listing OpenStackFloatingIPs: %w", err)
		}
		for _, fip := range floatingIPs.Items {
			if fip.GetName() != resource.GetName() && fip.Status.Resource.ID != "" {
				adoptedIDs[fip.Status.Resource.ID] = struct{}{}
			}
		}
	}

	var candidates []floatingips.FloatingIP
	err := floatingips.List(networkClient, floatingips.ListOpts{
		Description:       createOpts.Description,
		FloatingNetworkID: createOpts.FloatingNetworkID,
		PortID:            createOpts.PortID,
		FixedIP:           createOpts.FixedIP,
		FloatingIP:        createOpts.FloatingIP,
		TenantID:          createOpts.TenantID,
		ProjectID:         createOpts.ProjectID,
	}).EachPage(func(page pagination.Page) (bool, error) {
		items, err := floatingips.ExtractFloatingIPs(page)
		if err != nil {
			return false, fmt.Errorf("extracting resources: %w", err)
		}
		for i := range items {
			if _, ok := adoptedIDs[items[i].ID]; !ok {
				candidates = append(candidates, items[i])
			}
		}

		return true, nil
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
func (r *OpenStackFloatingIPReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&openstackv1.OpenStackFloatingIP{}).
		WithEventFilter(apply.IgnoreManagedFieldsOnly{}).
		Watches(&openstackv1.OpenStackCloud{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o client.Object) []reconcile.Request {
			// Fetch a list of all OpenStackFloatingIPs that reference this OpenStackCloud.
			kclient := mgr.GetClient()
			logger := mgr.GetLogger()

			floatingIPs := &openstackv1.OpenStackFloatingIPList{}
			if err := kclient.List(ctx, floatingIPs,
				client.InNamespace(o.GetNamespace()),
				client.HasLabels{openstackv1.OpenStackDependencyLabelCloud(o.GetName())},
			); err != nil {
				logger.Error(err, "unable to list OpenStackFloatingIPs")
				return nil
			}

			// Reconcile each OpenStackFloatingIP that is not Ready and that references this OpenStackCloud.
			reqs := make([]reconcile.Request, 0, len(floatingIPs.Items))
			for _, floatingIP := range floatingIPs.Items {
				if conditions.IsReady(&floatingIP) {
					continue
				}
				reqs = append(reqs, reconcile.Request{
					NamespacedName: client.ObjectKey{
						Namespace: floatingIP.GetNamespace(),
						Name:      floatingIP.GetName(),
					},
				})
				logger.V(5).Info("update of OpenStackCloud triggers reconcile of OpenStackFloatingIP",
					"namespace", o.GetNamespace(),
					"cloud", o.GetName(),
					"floating ip", floatingIP.GetName())
			}
			return reqs
		})).
		Watches(&openstackv1.OpenStackPort{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o client.Object) []reconcile.Request {
			// Fetch a list of all OpenStackFloatingIPs that reference this OpenStackPort.
			kclient := mgr.GetClient()
			logger := mgr.GetLogger()

			floatingIPs := &openstackv1.OpenStackFloatingIPList{}
			if err := kclient.List(ctx, floatingIPs,
				client.InNamespace(o.GetNamespace()),
				client.HasLabels{openstackv1.OpenStackDependencyLabelPort(o.GetName())},
			); err != nil {
				logger.Error(err, "unable to list OpenStackFloatingIPs")
				return nil
			}

			// Reconcile each OpenStackFloatingIP that is not Ready and that references this OpenStackPort.
			reqs := make([]reconcile.Request, 0, len(floatingIPs.Items))
			for _, floatingIP := range floatingIPs.Items {
				if conditions.IsReady(&floatingIP) {
					continue
				}
				reqs = append(reqs, reconcile.Request{
					NamespacedName: client.ObjectKey{
						Namespace: floatingIP.GetNamespace(),
						Name:      floatingIP.GetName(),
					},
				})
				logger.V(5).Info("update of OpenStackPort triggers reconcile of OpenStackFloatingIP",
					"namespace", o.GetNamespace(),
					"cloud", o.GetName(),
					"floating ip", floatingIP.GetName())
			}
			return reqs
		})).
		Watches(&openstackv1.OpenStackSubnet{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o client.Object) []reconcile.Request {
			// Fetch a list of all OpenStackFloatingIPs that reference this OpenStackSubnet.
			kclient := mgr.GetClient()
			logger := mgr.GetLogger()

			floatingIPs := &openstackv1.OpenStackFloatingIPList{}
			if err := kclient.List(ctx, floatingIPs,
				client.InNamespace(o.GetNamespace()),
				client.HasLabels{openstackv1.OpenStackDependencyLabelSubnet(o.GetName())},
			); err != nil {
				logger.Error(err, "unable to list OpenStackFloatingIPs")
				return nil
			}

			// Reconcile each OpenStackFloatingIP that is not Ready and that references this OpenStackSubnet.
			reqs := make([]reconcile.Request, 0, len(floatingIPs.Items))
			for _, floatingIP := range floatingIPs.Items {
				if conditions.IsReady(&floatingIP) {
					continue
				}
				reqs = append(reqs, reconcile.Request{
					NamespacedName: client.ObjectKey{
						Namespace: floatingIP.GetNamespace(),
						Name:      floatingIP.GetName(),
					},
				})
				logger.V(5).Info("update of OpenStackSubnet triggers reconcile of OpenStackFloatingIP",
					"namespace", o.GetNamespace(),
					"cloud", o.GetName(),
					"floating ip", floatingIP.GetName())
			}
			return reqs
		})).
		Complete(r)
}
