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

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/routers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/pagination"
	openstackv1 "github.com/gophercloud/openstack-resource-controller/api/v1alpha1"
	"github.com/gophercloud/openstack-resource-controller/pkg/apply"
	"github.com/gophercloud/openstack-resource-controller/pkg/cloud"
	"github.com/gophercloud/openstack-resource-controller/pkg/conditions"
	"github.com/gophercloud/openstack-resource-controller/pkg/labels"
)

const (
	OpenStackRouterFinalizer = "openstackrouter.k-orc.cloud"
)

// OpenStackRouterReconciler reconciles a OpenStackRouter object
type OpenStackRouterReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=openstackrouters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=openstackrouters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=openstackrouters/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *OpenStackRouterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	logger := log.FromContext(ctx)
	logger = logger.WithValues("OpenStackRouter", req.Name)

	resource := &openstackv1.OpenStackRouter{}
	err := r.Client.Get(ctx, req.NamespacedName, resource)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("resource not found in the API")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if resource.DeletionTimestamp.IsZero() {
		finalizerUpdated := controllerutil.AddFinalizer(resource, OpenStackRouterFinalizer)

		newLabels := map[string]string{
			openstackv1.OpenStackDependencyLabelCloud(resource.Spec.Cloud): "",
		}
		if gateway := resource.Spec.Resource.ExternalGateway; gateway != nil {
			if gateway.Network != "" {
				newLabels[openstackv1.OpenStackDependencyLabelNetwork(resource.Spec.Resource.ExternalGateway.Network)] = ""
			}
			for _, ip := range gateway.ExternalFixedIPs {
				newLabels[openstackv1.OpenStackDependencyLabelSubnet(ip.Subnet)] = ""
			}
		}
		for _, port := range resource.Spec.Resource.Ports {
			newLabels[openstackv1.OpenStackDependencyLabelPort(port)] = ""
		}

		labelsMerger, labelsUpdated := labels.ReplacePrefixed(openstackv1.OpenStackLabelPrefix, resource.Labels, newLabels)

		if finalizerUpdated || labelsUpdated {
			logger.Info("applying labels and finalizer")
			patch := &openstackv1.OpenStackRouter{}
			patch.TypeMeta = resource.TypeMeta
			patch.Finalizers = resource.GetFinalizers()
			patch.Labels = labelsMerger
			return ctrl.Result{}, apply.Apply(ctx, r.Client, resource, patch, "spec")
		}
	}

	statusPatchResource := &openstackv1.OpenStackRouter{
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
// TODO: potentially handle (some?) modifications accepted in OpenStack
func (r *OpenStackRouterReconciler) reconcile(ctx context.Context, networkClient *gophercloud.ServiceClient, resource, statusPatchResource *openstackv1.OpenStackRouter) (_ ctrl.Result, reterr error) {
	logger := log.FromContext(ctx)

	var (
		router  *routers.Router
		err     error
		created bool
	)
	if openstackID := coalesce(resource.Spec.ID, resource.Status.Resource.ID); openstackID != "" {
		logger = logger.WithValues("OpenStackID", openstackID)

		router, err = routers.Get(networkClient, openstackID).Extract()
		if err != nil {
			return ctrl.Result{}, err
		}
		logger.Info("OpenStack resource found")
	} else {
		var gatewayInfo *routers.GatewayInfo
		if gateway := resource.Spec.Resource.ExternalGateway; gateway != nil {
			externalFixedIPs := make([]routers.ExternalFixedIP, len(gateway.ExternalFixedIPs))
			for i := range gateway.ExternalFixedIPs {
				dependency := &openstackv1.OpenStackSubnet{}
				dependencyKey := client.ObjectKey{Namespace: resource.GetNamespace(), Name: gateway.ExternalFixedIPs[i].Subnet}
				err = r.Client.Get(ctx, dependencyKey, dependency)
				if err != nil && !apierrors.IsNotFound(err) {
					return ctrl.Result{}, err
				}

				// Dependency either doesn't exist, or is being deleted
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
				externalFixedIPs[i] = routers.ExternalFixedIP{
					IPAddress: gateway.ExternalFixedIPs[i].IPAddress,
					SubnetID:  dependency.Status.Resource.ID,
				}
			}

			dependency := &openstackv1.OpenStackNetwork{}
			dependencyKey := client.ObjectKey{Namespace: resource.GetNamespace(), Name: gateway.Network}
			err = r.Client.Get(ctx, dependencyKey, dependency)
			if err != nil && !apierrors.IsNotFound(err) {
				return ctrl.Result{}, err
			}

			// Dependency either doesn't exist, or is being deleted
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
			gatewayInfo = &routers.GatewayInfo{
				NetworkID:        dependency.Status.Resource.ID,
				EnableSNAT:       gateway.EnableSNAT,
				ExternalFixedIPs: externalFixedIPs,
			}
			created = true
		}

		createOpts := routers.CreateOpts{
			Name:                  resource.Spec.Resource.Name,
			Description:           resource.Spec.Resource.Description,
			AdminStateUp:          resource.Spec.Resource.AdminStateUp,
			Distributed:           resource.Spec.Resource.Distributed,
			TenantID:              resource.Spec.Resource.TenantID,
			ProjectID:             resource.Spec.Resource.ProjectID,
			GatewayInfo:           gatewayInfo,
			AvailabilityZoneHints: resource.Spec.Resource.AvailabilityZoneHints,
		}
		router, err = r.findAdoptee(log.IntoContext(ctx, logger), networkClient, resource, createOpts)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("unable to find adoption candidates: %w", err)
		}
		if router != nil {
			logger = logger.WithValues("OpenStackID", router.ID)
			logger.Info("OpenStack resource adopted")
		} else {
			router, err = routers.Create(networkClient, createOpts).Extract()
			if err != nil {
				return ctrl.Result{}, err
			}
			logger = logger.WithValues("OpenStackID", router.ID)
			logger.Info("OpenStack resource created")
		}
	}

	routes := make([]openstackv1.OpenStackRouterRoute, len(router.Routes))
	for i := range router.Routes {
		routes[i] = openstackv1.OpenStackRouterRoute{
			NextHop:         router.Routes[i].NextHop,
			DestinationCIDR: router.Routes[i].DestinationCIDR,
		}

	}

	externalFixedIPs := make([]openstackv1.OpenStackRouterExternalFixedIP, len(router.GatewayInfo.ExternalFixedIPs))
	for i := range router.GatewayInfo.ExternalFixedIPs {
		externalFixedIPs[i] = openstackv1.OpenStackRouterExternalFixedIP{
			IPAddress: router.GatewayInfo.ExternalFixedIPs[i].IPAddress,
			Subnet:    router.GatewayInfo.ExternalFixedIPs[i].SubnetID,
		}
	}

	gatewayInfo := openstackv1.OpenStackRouterStatusExternalGatewayInfo{
		NetworkID:        router.GatewayInfo.NetworkID,
		EnableSNAT:       router.GatewayInfo.EnableSNAT,
		ExternalFixedIPs: externalFixedIPs,
	}

	statusPatchResource.Status.Resource = openstackv1.OpenStackRouterResourceStatus{
		ID:                    router.ID,
		Description:           router.Description,
		AdminStateUp:          router.AdminStateUp,
		Distributed:           router.Distributed,
		TenantID:              router.TenantID,
		ProjectID:             router.ProjectID,
		GatewayInfo:           gatewayInfo,
		AvailabilityZoneHints: router.AvailabilityZoneHints,
		Status:                router.Status,
		Tags:                  router.Tags,
		Routes:                routes,
	}

	if created {
		if updated, condition := conditions.SetNotReadyConditionPending(resource, statusPatchResource); updated {
			conditions.EmitEventForCondition(r.Recorder, resource, corev1.EventTypeNormal, condition)
		}
		return ctrl.Result{Requeue: true}, nil
	}

	currentPortSet := make(map[string]struct{})
	var currentPortList []string
	if err := ports.List(networkClient, ports.ListOpts{DeviceID: resource.Status.Resource.ID}).EachPage(func(page pagination.Page) (bool, error) {
		portList, err := ports.ExtractPorts(page)
		if err != nil {
			return false, err
		}
		for i := range portList {
			currentPortSet[portList[i].ID] = struct{}{}
			currentPortList = append(currentPortList, portList[i].ID)
		}
		return true, nil
	}); err != nil {
		return ctrl.Result{}, err
	}
	statusPatchResource.Status.Resource.Ports = currentPortList

	portIDs := make([]string, len(resource.Spec.Resource.Ports))
	for i := range resource.Spec.Resource.Ports {
		dependency := &openstackv1.OpenStackPort{}
		dependencyKey := client.ObjectKey{Namespace: resource.GetNamespace(), Name: resource.Spec.Resource.Ports[i]}
		err = r.Client.Get(ctx, dependencyKey, dependency)
		if err != nil && !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		// Dependency either doesn't exist, or is being deleted
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

		portIDs[i] = dependency.Status.Resource.ID
	}

	for _, portID := range portIDs {
		if _, ok := currentPortSet[portID]; ok {
			continue
		}

		interfaceInfo, err := routers.AddInterface(networkClient, resource.Status.Resource.ID, routers.AddInterfaceOpts{
			PortID: portID,
		}).Extract()
		if err != nil {
			return ctrl.Result{}, err
		}
		statusPatchResource.Status.Resource.Ports = append(statusPatchResource.Status.Resource.Ports, interfaceInfo.PortID)
	}

	if updated, condition := conditions.SetReadyCondition(resource, statusPatchResource); updated {
		conditions.EmitEventForCondition(r.Recorder, resource, corev1.EventTypeNormal, condition)
	}
	return ctrl.Result{}, nil
}

func (r *OpenStackRouterReconciler) reconcileDelete(ctx context.Context, networkClient *gophercloud.ServiceClient, resource, statusPatchResource *openstackv1.OpenStackRouter) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if resource.Status.Resource.ID == "" {
		logger.Info("deletion was requested on a resource that hasn't been successfully created or adopted yet.")
	} else {
		logger = logger.WithValues("OpenStackID", resource.Status.Resource.ID)
		if !resource.Spec.Unmanaged {
			if err := ports.List(networkClient, ports.ListOpts{DeviceID: resource.Status.Resource.ID}).EachPage(func(page pagination.Page) (bool, error) {
				portList, err := ports.ExtractPorts(page)
				if err != nil {
					return false, err
				}
				var err404 gophercloud.ErrDefault404
				for _, port := range portList {
					if _, err := routers.RemoveInterface(networkClient, resource.Status.Resource.ID, routers.RemoveInterfaceOpts{
						PortID: port.ID,
					}).Extract(); err != nil && !errors.As(err, &err404) {
						return false, err
					}
				}
				return true, nil
			}); err != nil {
				return ctrl.Result{}, err
			}

			if err := routers.Delete(networkClient, resource.Status.Resource.ID).ExtractErr(); err != nil {
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

	if updated := controllerutil.RemoveFinalizer(resource, OpenStackRouterFinalizer); updated {
		logger.Info("removing finalizer")
		if updated, condition := conditions.SetNotReadyConditionDeleting(resource, statusPatchResource, "Removing finalizer"); updated {
			conditions.EmitEventForCondition(r.Recorder, resource, corev1.EventTypeNormal, condition)
		}
		patch := &openstackv1.OpenStackRouter{}
		patch.TypeMeta = resource.TypeMeta
		patch.Finalizers = resource.GetFinalizers()
		return ctrl.Result{}, apply.Apply(ctx, r.Client, resource, patch, "spec")
	}
	return ctrl.Result{}, nil
}

func routerEquals(candidate routers.Router, opts routers.CreateOpts) bool {
	if opts.GatewayInfo != nil {
		if candidate.GatewayInfo.NetworkID != opts.GatewayInfo.NetworkID {
			return false
		}
		if opts.GatewayInfo.EnableSNAT != nil && candidate.GatewayInfo.EnableSNAT != opts.GatewayInfo.EnableSNAT {
			return false
		}
		if len(opts.GatewayInfo.ExternalFixedIPs) > 0 {
			if !sliceContentEquals(candidate.GatewayInfo.ExternalFixedIPs, opts.GatewayInfo.ExternalFixedIPs) {
				fmt.Printf("%+v\n", candidate.GatewayInfo.ExternalFixedIPs)
				fmt.Printf("%+v\n", opts.GatewayInfo.ExternalFixedIPs)
				return false
			}
		}
	}
	if len(candidate.AvailabilityZoneHints) != len(opts.AvailabilityZoneHints) {
		return false
	}
	if opts.AvailabilityZoneHints != nil {
		if !sliceContentEquals(candidate.AvailabilityZoneHints, opts.AvailabilityZoneHints) {
			return false
		}
	}
	return true
}

func (r *OpenStackRouterReconciler) findAdoptee(ctx context.Context, imageClient *gophercloud.ServiceClient, resource client.Object, createOpts routers.CreateOpts) (*routers.Router, error) {
	adoptedIDs := make(map[string]struct{})
	{
		list := &openstackv1.OpenStackRouterList{}
		if err := r.Client.List(ctx, list,
			client.InNamespace(resource.GetNamespace()),
		); err != nil {
			return nil, fmt.Errorf("listing OpenStackRouters: %w", err)
		}
		for _, port := range list.Items {
			if port.GetName() != resource.GetName() && port.Status.Resource.ID != "" {
				adoptedIDs[port.Status.Resource.ID] = struct{}{}
			}
		}
	}
	listOpts := routers.ListOpts{
		Name:         createOpts.Name,
		Description:  createOpts.Description,
		AdminStateUp: createOpts.AdminStateUp,
		Distributed:  createOpts.Distributed,
		TenantID:     createOpts.TenantID,
		ProjectID:    createOpts.ProjectID,
	}

	var candidates []routers.Router
	err := routers.List(imageClient, listOpts).EachPage(func(page pagination.Page) (bool, error) {
		items, err := routers.ExtractRouters(page)
		if err != nil {
			return false, fmt.Errorf("extracting resources: %w", err)
		}
		for i := range items {
			if _, ok := adoptedIDs[items[i].ID]; !ok && routerEquals(items[i], createOpts) {
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
func (r *OpenStackRouterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&openstackv1.OpenStackRouter{}).
		WithEventFilter(apply.IgnoreManagedFieldsOnly{}).
		Watches(&openstackv1.OpenStackCloud{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o client.Object) []reconcile.Request {
			// Fetch a list of all OpenStackRouters that reference this OpenStackCloud.
			kclient := mgr.GetClient()
			logger := mgr.GetLogger()

			routers := &openstackv1.OpenStackRouterList{}
			if err := kclient.List(ctx, routers,
				client.InNamespace(o.GetNamespace()),
				client.HasLabels{openstackv1.OpenStackDependencyLabelCloud(o.GetName())},
			); err != nil {
				logger.Error(err, "unable to list OpenStackPorts")
				return nil
			}

			// Reconcile each OpenStackPort that is not Ready and that references this OpenStackCloud.
			reqs := make([]reconcile.Request, 0, len(routers.Items))
			for _, router := range routers.Items {
				if conditions.IsReady(&router) {
					continue
				}
				reqs = append(reqs, reconcile.Request{
					NamespacedName: client.ObjectKey{
						Namespace: router.GetNamespace(),
						Name:      router.GetName(),
					},
				})
				logger.V(5).Info("update of OpenStackCloud triggers reconcile of OpenStackRouter",
					"namespace", o.GetNamespace(),
					"cloud", o.GetName(),
					"router", router.GetName())
			}
			return reqs
		})).
		Watches(&openstackv1.OpenStackNetwork{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o client.Object) []reconcile.Request {
			// Fetch a list of all OpenStackRouters that reference this OpenStackNetwork.
			kclient := mgr.GetClient()
			logger := mgr.GetLogger()

			routers := &openstackv1.OpenStackRouterList{}
			if err := kclient.List(ctx, routers,
				client.InNamespace(o.GetNamespace()),
				client.HasLabels{openstackv1.OpenStackDependencyLabelNetwork(o.GetName())},
			); err != nil {
				logger.Error(err, "unable to list OpenStackPorts")
				return nil
			}

			// Reconcile each OpenStackPort that is not Ready and that references this OpenStackNetwork.
			reqs := make([]reconcile.Request, 0, len(routers.Items))
			for _, router := range routers.Items {
				if conditions.IsReady(&router) {
					continue
				}
				reqs = append(reqs, reconcile.Request{
					NamespacedName: client.ObjectKey{
						Namespace: router.GetNamespace(),
						Name:      router.GetName(),
					},
				})
				logger.V(5).Info("update of OpenStackCloud triggers reconcile of OpenStackRouter",
					"namespace", o.GetNamespace(),
					"cloud", o.GetName(),
					"router", router.GetName())
			}
			return reqs
		})).
		Watches(&openstackv1.OpenStackSubnet{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o client.Object) []reconcile.Request {
			// Fetch a list of all OpenStackRouters that reference this OpenStackSubnet.
			kclient := mgr.GetClient()
			logger := mgr.GetLogger()

			routers := &openstackv1.OpenStackRouterList{}
			if err := kclient.List(ctx, routers,
				client.InNamespace(o.GetNamespace()),
				client.HasLabels{openstackv1.OpenStackDependencyLabelSubnet(o.GetName())},
			); err != nil {
				logger.Error(err, "unable to list OpenStackPorts")
				return nil
			}

			// Reconcile each OpenStackPort that is not Ready and that references this OpenStackSubnet.
			reqs := make([]reconcile.Request, 0, len(routers.Items))
			for _, router := range routers.Items {
				if conditions.IsReady(&router) {
					continue
				}
				reqs = append(reqs, reconcile.Request{
					NamespacedName: client.ObjectKey{
						Namespace: router.GetNamespace(),
						Name:      router.GetName(),
					},
				})
				logger.V(5).Info("update of OpenStackSubnet triggers reconcile of OpenStackRouter",
					"namespace", o.GetNamespace(),
					"cloud", o.GetName(),
					"router", router.GetName())
			}
			return reqs
		})).
		Watches(&openstackv1.OpenStackPort{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o client.Object) []reconcile.Request {
			// Fetch a list of all OpenStackRouters that reference this OpenStackPort.
			kclient := mgr.GetClient()
			logger := mgr.GetLogger()

			routers := &openstackv1.OpenStackRouterList{}
			if err := kclient.List(ctx, routers,
				client.InNamespace(o.GetNamespace()),
				client.HasLabels{openstackv1.OpenStackDependencyLabelPort(o.GetName())},
			); err != nil {
				logger.Error(err, "unable to list OpenStackPorts")
				return nil
			}

			// Reconcile each OpenStackPort that is not Ready and that references this OpenStackPort.
			reqs := make([]reconcile.Request, 0, len(routers.Items))
			for _, router := range routers.Items {
				if conditions.IsReady(&router) {
					continue
				}
				reqs = append(reqs, reconcile.Request{
					NamespacedName: client.ObjectKey{
						Namespace: router.GetNamespace(),
						Name:      router.GetName(),
					},
				})
				logger.V(5).Info("update of OpenStackPort triggers reconcile of OpenStackRouter",
					"namespace", o.GetNamespace(),
					"cloud", o.GetName(),
					"router", router.GetName())
			}
			return reqs
		})).
		Complete(r)
}
