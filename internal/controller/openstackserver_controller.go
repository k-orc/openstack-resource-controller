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
	"encoding/json"
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
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/pagination"
	openstackv1 "github.com/gophercloud/openstack-resource-controller/api/v1alpha1"
	"github.com/gophercloud/openstack-resource-controller/pkg/apply"
	"github.com/gophercloud/openstack-resource-controller/pkg/cloud"
	"github.com/gophercloud/openstack-resource-controller/pkg/conditions"
	"github.com/gophercloud/openstack-resource-controller/pkg/labels"
)

const (
	OpenStackServerFinalizer = "openstackserver.k-orc.cloud"
)

// OpenStackServerReconciler reconciles a OpenStackServer object
type OpenStackServerReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=openstackservers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=openstackservers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=openstackservers/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *OpenStackServerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	logger := log.FromContext(ctx)
	logger = logger.WithValues("OpenStackServer", req.Name)

	resource := &openstackv1.OpenStackServer{}
	err := r.Client.Get(ctx, req.NamespacedName, resource)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("resource not found in the API")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if resource.DeletionTimestamp.IsZero() {
		finalizerUpdated := controllerutil.AddFinalizer(resource, OpenStackServerFinalizer)

		newLabels := map[string]string{
			openstackv1.OpenStackDependencyLabelCloud(resource.Spec.Cloud):            "",
			openstackv1.OpenStackDependencyLabelFlavor(resource.Spec.Resource.Flavor): "",
		}
		for _, sg := range resource.Spec.Resource.SecurityGroups {
			newLabels[openstackv1.OpenStackDependencyLabelSecurityGroup(sg)] = ""
		}
		for _, iface := range resource.Spec.Resource.Networks {
			if network := iface.Network; network != "" {
				newLabels[openstackv1.OpenStackDependencyLabelNetwork(network)] = ""
			}
			if port := iface.Port; port != "" {
				newLabels[openstackv1.OpenStackDependencyLabelPort(port)] = ""
			}
		}

		labelsMerger, labelsUpdated := labels.ReplacePrefixed(openstackv1.OpenStackLabelPrefix, resource.Labels, newLabels)

		if finalizerUpdated || labelsUpdated {
			logger.Info("applying labels and finalizer")
			patch := &openstackv1.OpenStackServer{}
			patch.TypeMeta = resource.TypeMeta
			patch.Finalizers = resource.GetFinalizers()
			patch.Labels = labelsMerger
			return ctrl.Result{}, apply.Apply(ctx, r.Client, resource, patch, "spec")
		}
	}

	statusPatchResource := &openstackv1.OpenStackServer{
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
func (r *OpenStackServerReconciler) reconcile(ctx context.Context, computeClient *gophercloud.ServiceClient, resource, statusPatchResource *openstackv1.OpenStackServer) (_ ctrl.Result, reterr error) {
	logger := log.FromContext(ctx)

	var (
		server *servers.Server
		err    error
	)
	if openstackID := coalesce(resource.Spec.ID, resource.Status.Resource.ID); openstackID != "" {
		logger = logger.WithValues("OpenStackID", openstackID)

		server, err = servers.Get(computeClient, openstackID).Extract()
		if err != nil {
			return ctrl.Result{}, err
		}
		logger.Info("OpenStack resource found")
	} else {
		serverNetworks := make([]servers.Network, len(resource.Spec.Resource.Networks))
		for i := range resource.Spec.Resource.Networks {
			n := servers.Network{
				FixedIP: resource.Spec.Resource.Networks[i].FixedIP,
				Tag:     resource.Spec.Resource.Networks[i].Tag,
			}
			if network := resource.Spec.Resource.Networks[i].Network; network != "" {
				dependency := &openstackv1.OpenStackNetwork{}
				dependencyKey := client.ObjectKey{Namespace: resource.GetNamespace(), Name: network}
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
				n.UUID = dependency.Status.Resource.ID
			}
			if port := resource.Spec.Resource.Networks[i].Port; port != "" {
				dependency := &openstackv1.OpenStackPort{}
				dependencyKey := client.ObjectKey{Namespace: resource.GetNamespace(), Name: port}
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
				n.Port = dependency.Status.Resource.ID
			}
			serverNetworks[i] = n
		}

		var imageID string
		{
			dependency := &openstackv1.OpenStackImage{}
			dependencyKey := client.ObjectKey{Namespace: resource.GetNamespace(), Name: resource.Spec.Resource.Image}
			err = r.Client.Get(ctx, dependencyKey, dependency)
			if err != nil && !apierrors.IsNotFound(err) {
				return ctrl.Result{}, err
			}

			// Dependency either doesn't exist, or is being deleted, or is not ready
			if err != nil || !dependency.DeletionTimestamp.IsZero() || !conditions.IsReady(dependency) || dependency.Status.Resource.ID == "" {
				logger.Info("waiting for image")

				if updated, condition := conditions.SetNotReadyConditionWaiting(resource, statusPatchResource, []conditions.Dependency{
					{ObjectKey: dependencyKey, Resource: "image"},
				}); updated {
					// Emit an event if we're setting the condition for the first time
					conditions.EmitEventForCondition(r.Recorder, resource, corev1.EventTypeNormal, condition)
				}
				return ctrl.Result{}, nil
			}
			imageID = dependency.Status.Resource.ID
		}

		var flavorID string
		{
			dependency := &openstackv1.OpenStackFlavor{}
			dependencyKey := client.ObjectKey{Namespace: resource.GetNamespace(), Name: resource.Spec.Resource.Flavor}
			err = r.Client.Get(ctx, dependencyKey, dependency)
			if err != nil && !apierrors.IsNotFound(err) {
				return ctrl.Result{}, err
			}

			// Dependency either doesn't exist, or is being deleted, or is not ready
			if err != nil || !dependency.DeletionTimestamp.IsZero() || !conditions.IsReady(dependency) || dependency.Status.Resource.ID == "" {
				logger.Info("waiting for flavor")

				if updated, condition := conditions.SetNotReadyConditionWaiting(resource, statusPatchResource, []conditions.Dependency{
					{ObjectKey: dependencyKey, Resource: "flavor"},
				}); updated {
					// Emit an event if we're setting the condition for the first time
					conditions.EmitEventForCondition(r.Recorder, resource, corev1.EventTypeNormal, condition)
				}
				return ctrl.Result{}, nil
			}
			flavorID = dependency.Status.Resource.ID
		}

		securityGroupIDs := make([]string, len(resource.Spec.Resource.SecurityGroups))
		for i, securityGroupName := range resource.Spec.Resource.SecurityGroups {
			dependency := &openstackv1.OpenStackSecurityGroup{}
			dependencyKey := client.ObjectKey{Namespace: resource.GetNamespace(), Name: securityGroupName}
			err = r.Client.Get(ctx, dependencyKey, dependency)
			if err != nil && !apierrors.IsNotFound(err) {
				return ctrl.Result{}, err
			}

			// Dependency either doesn't exist, or is being deleted, or is not ready
			if err != nil || !dependency.DeletionTimestamp.IsZero() || !conditions.IsReady(dependency) || dependency.Status.Resource.ID == "" {
				logger.Info("waiting for security group")

				if updated, condition := conditions.SetNotReadyConditionWaiting(resource, statusPatchResource, []conditions.Dependency{
					{ObjectKey: dependencyKey, Resource: "security group"},
				}); updated {
					// Emit an event if we're setting the condition for the first time
					conditions.EmitEventForCondition(r.Recorder, resource, corev1.EventTypeNormal, condition)
				}
				return ctrl.Result{}, nil
			}
			securityGroupIDs[i] = dependency.Status.Resource.ID
		}

		createOpts := servers.CreateOpts{
			Name:           resource.Spec.Resource.Name,
			ImageRef:       imageID,
			FlavorRef:      flavorID,
			Networks:       serverNetworks,
			SecurityGroups: securityGroupIDs,
			UserData:       resource.Spec.Resource.UserData,
		}
		server, err = r.findAdoptee(log.IntoContext(ctx, logger), computeClient, resource, createOpts)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("unable to find adoption candidates: %w", err)
		}
		if server != nil {
			logger = logger.WithValues("OpenStackID", server.ID)
			logger.Info("OpenStack resource adopted")
		} else {
			server, err = servers.Create(computeClient, createOpts).Extract()
			if err != nil {
				return ctrl.Result{}, err
			}
			logger = logger.WithValues("OpenStackID", server.ID)
			logger.Info("OpenStack resource created")
		}
	}

	jsonImage, err := json.Marshal(server.Image)
	if err != nil {
		logger.Info("error marshaling image information: " + err.Error())
	}

	jsonFlavor, err := json.Marshal(server.Flavor)
	if err != nil {
		logger.Info("error marshaling flavor information: " + err.Error())
	}

	jsonAddresses, err := json.Marshal(server.Addresses)
	if err != nil {
		logger.Info("error marshaling addresses information: " + err.Error())
	}

	jsonMetadata, err := json.Marshal(server.Metadata)
	if err != nil {
		logger.Info("error marshaling metadata information: " + err.Error())
	}

	jsonSecurityGroups, err := json.Marshal(server.SecurityGroups)
	if err != nil {
		logger.Info("error marshaling security group information: " + err.Error())
	}

	statusPatchResource.Status.Resource = openstackv1.OpenStackServerResourceStatus{
		ID:               server.ID,
		TenantID:         server.TenantID,
		UserID:           server.UserID,
		Name:             server.Name,
		UpdatedAt:        server.Updated.UTC().Format(time.RFC3339),
		CreatedAt:        server.Created.UTC().Format(time.RFC3339),
		HostID:           server.HostID,
		Status:           server.Status,
		Progress:         server.Progress,
		AccessIPv4:       server.AccessIPv4,
		AccessIPv6:       server.AccessIPv6,
		ImageID:          string(jsonImage),
		FlavorID:         string(jsonFlavor),
		Addresses:        string(jsonAddresses),
		Metadata:         string(jsonMetadata),
		Links:            []string{},
		KeyName:          server.KeyName,
		SecurityGroupIDs: string(jsonSecurityGroups),
	}

	if updated, condition := conditions.SetReadyCondition(resource, statusPatchResource); updated {
		conditions.EmitEventForCondition(r.Recorder, resource, corev1.EventTypeNormal, condition)
	}
	return ctrl.Result{}, nil
}

func (r *OpenStackServerReconciler) reconcileDelete(ctx context.Context, networkClient *gophercloud.ServiceClient, resource, statusPatchResource *openstackv1.OpenStackServer) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if resource.Status.Resource.ID == "" {
		logger.Info("deletion was requested on a resource that hasn't been successfully created or adopted yet.")
	} else {
		logger = logger.WithValues("OpenStackID", resource.Status.Resource.ID)
		if !resource.Spec.Unmanaged {
			if err := servers.Delete(networkClient, resource.Status.Resource.ID).ExtractErr(); err != nil {
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

	if updated := controllerutil.RemoveFinalizer(resource, OpenStackServerFinalizer); updated {
		logger.Info("removing finalizer")
		if updated, condition := conditions.SetNotReadyConditionDeleting(resource, statusPatchResource, "Removing finalizer"); updated {
			conditions.EmitEventForCondition(r.Recorder, resource, corev1.EventTypeNormal, condition)
		}
		patch := &openstackv1.OpenStackServer{}
		patch.TypeMeta = resource.TypeMeta
		patch.Finalizers = resource.GetFinalizers()
		return ctrl.Result{}, apply.Apply(ctx, r.Client, resource, patch, "spec")
	}
	return ctrl.Result{}, nil
}

func (r *OpenStackServerReconciler) findAdoptee(ctx context.Context, computeClient *gophercloud.ServiceClient, resource client.Object, createOpts servers.CreateOpts) (*servers.Server, error) {
	adoptedIDs := make(map[string]struct{})
	{
		list := &openstackv1.OpenStackServerList{}
		if err := r.Client.List(ctx, list,
			client.InNamespace(resource.GetNamespace()),
		); err != nil {
			return nil, fmt.Errorf("listing OpenStackServers: %w", err)
		}
		for _, port := range list.Items {
			if port.GetName() != resource.GetName() && port.Status.Resource.ID != "" {
				adoptedIDs[port.Status.Resource.ID] = struct{}{}
			}
		}
	}

	listOpts := servers.ListOpts{
		Image:            createOpts.ImageRef,
		Flavor:           createOpts.FlavorRef,
		IP:               createOpts.AccessIPv4,
		IP6:              createOpts.AccessIPv6,
		Name:             createOpts.Name,
		AvailabilityZone: createOpts.AvailabilityZone,
	}
	var candidates []servers.Server
	err := servers.List(computeClient, listOpts).EachPage(func(page pagination.Page) (bool, error) {
		items, err := servers.ExtractServers(page)
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
func (r *OpenStackServerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&openstackv1.OpenStackServer{}).
		WithEventFilter(apply.IgnoreManagedFieldsOnly{}).
		Watches(&openstackv1.OpenStackCloud{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o client.Object) []reconcile.Request {
			// Fetch a list of all OpenStackServers that reference this OpenStackCloud.
			kclient := mgr.GetClient()
			logger := mgr.GetLogger()

			servers := &openstackv1.OpenStackServerList{}
			if err := kclient.List(ctx, servers,
				client.InNamespace(o.GetNamespace()),
				client.HasLabels{openstackv1.OpenStackDependencyLabelCloud(o.GetName())},
			); err != nil {
				logger.Error(err, "unable to list OpenStackServers")
				return nil
			}

			// Reconcile each OpenStackServer that is not Ready and that references this OpenStackCloud.
			reqs := make([]reconcile.Request, 0, len(servers.Items))
			for _, server := range servers.Items {
				if conditions.IsReady(&server) {
					continue
				}
				reqs = append(reqs, reconcile.Request{
					NamespacedName: client.ObjectKey{
						Namespace: server.GetNamespace(),
						Name:      server.GetName(),
					},
				})
				logger.V(5).Info("update of OpenStackCloud triggers reconcile of OpenStackServer",
					"namespace", o.GetNamespace(),
					"cloud", o.GetName(),
					"server", server.GetName())
			}
			return reqs
		})).
		Watches(&openstackv1.OpenStackFlavor{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o client.Object) []reconcile.Request {
			// Fetch a list of all OpenStackServers that reference this OpenStackFlavor.
			kclient := mgr.GetClient()
			logger := mgr.GetLogger()

			servers := &openstackv1.OpenStackServerList{}
			if err := kclient.List(ctx, servers,
				client.InNamespace(o.GetNamespace()),
				client.HasLabels{openstackv1.OpenStackDependencyLabelFlavor(o.GetName())},
			); err != nil {
				logger.Error(err, "unable to list OpenStackServers")
				return nil
			}

			// Reconcile each OpenStackServer that is not Ready and that references this OpenStackFlavor.
			reqs := make([]reconcile.Request, 0, len(servers.Items))
			for _, server := range servers.Items {
				if conditions.IsReady(&server) {
					continue
				}
				reqs = append(reqs, reconcile.Request{
					NamespacedName: client.ObjectKey{
						Namespace: server.GetNamespace(),
						Name:      server.GetName(),
					},
				})
				logger.V(5).Info("update of OpenStackFlavor triggers reconcile of OpenStackServer",
					"namespace", o.GetNamespace(),
					"cloud", o.GetName(),
					"server", server.GetName())
			}
			return reqs
		})).
		Watches(&openstackv1.OpenStackSecurityGroup{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o client.Object) []reconcile.Request {
			// Fetch a list of all OpenStackServers that reference this OpenStackSecurityGroup.
			kclient := mgr.GetClient()
			logger := mgr.GetLogger()

			servers := &openstackv1.OpenStackServerList{}
			if err := kclient.List(ctx, servers,
				client.InNamespace(o.GetNamespace()),
				client.HasLabels{openstackv1.OpenStackDependencyLabelSecurityGroup(o.GetName())},
			); err != nil {
				logger.Error(err, "unable to list OpenStackServers")
				return nil
			}

			// Reconcile each OpenStackServer that is not Ready and that references this OpenStackSecurityGroup.
			reqs := make([]reconcile.Request, 0, len(servers.Items))
			for _, server := range servers.Items {
				if conditions.IsReady(&server) {
					continue
				}
				reqs = append(reqs, reconcile.Request{
					NamespacedName: client.ObjectKey{
						Namespace: server.GetNamespace(),
						Name:      server.GetName(),
					},
				})
				logger.V(5).Info("update of OpenStackSecurityGroup triggers reconcile of OpenStackServer",
					"namespace", o.GetNamespace(),
					"cloud", o.GetName(),
					"server", server.GetName())
			}
			return reqs
		})).
		Watches(&openstackv1.OpenStackNetwork{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o client.Object) []reconcile.Request {
			// Fetch a list of all OpenStackServers that reference this OpenStackNetwork.
			kclient := mgr.GetClient()
			logger := mgr.GetLogger()

			servers := &openstackv1.OpenStackServerList{}
			if err := kclient.List(ctx, servers,
				client.InNamespace(o.GetNamespace()),
				client.HasLabels{openstackv1.OpenStackDependencyLabelNetwork(o.GetName())},
			); err != nil {
				logger.Error(err, "unable to list OpenStackServers")
				return nil
			}

			// Reconcile each OpenStackServer that is not Ready and that references this OpenStackNetwork.
			reqs := make([]reconcile.Request, 0, len(servers.Items))
			for _, server := range servers.Items {
				if conditions.IsReady(&server) {
					continue
				}
				reqs = append(reqs, reconcile.Request{
					NamespacedName: client.ObjectKey{
						Namespace: server.GetNamespace(),
						Name:      server.GetName(),
					},
				})
				logger.V(5).Info("update of OpenStackNetwork triggers reconcile of OpenStackServer",
					"namespace", o.GetNamespace(),
					"cloud", o.GetName(),
					"server", server.GetName())
			}
			return reqs
		})).
		Watches(&openstackv1.OpenStackPort{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o client.Object) []reconcile.Request {
			// Fetch a list of all OpenStackServers that reference this OpenStackPort.
			kclient := mgr.GetClient()
			logger := mgr.GetLogger()

			servers := &openstackv1.OpenStackServerList{}
			if err := kclient.List(ctx, servers,
				client.InNamespace(o.GetNamespace()),
				client.HasLabels{openstackv1.OpenStackDependencyLabelPort(o.GetName())},
			); err != nil {
				logger.Error(err, "unable to list OpenStackServers")
				return nil
			}

			// Reconcile each OpenStackServer that is not Ready and that references this OpenStackPort.
			reqs := make([]reconcile.Request, 0, len(servers.Items))
			for _, server := range servers.Items {
				if conditions.IsReady(&server) {
					continue
				}
				reqs = append(reqs, reconcile.Request{
					NamespacedName: client.ObjectKey{
						Namespace: server.GetNamespace(),
						Name:      server.GetName(),
					},
				})
				logger.V(5).Info("update of OpenStackPort triggers reconcile of OpenStackServer",
					"namespace", o.GetNamespace(),
					"cloud", o.GetName(),
					"server", server.GetName())
			}
			return reqs
		})).
		Complete(r)
}
