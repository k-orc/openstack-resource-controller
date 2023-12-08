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
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	"github.com/gophercloud/gophercloud/pagination"
	openstackv1 "github.com/gophercloud/openstack-resource-controller/api/v1alpha1"
	"github.com/gophercloud/openstack-resource-controller/pkg/apply"
	"github.com/gophercloud/openstack-resource-controller/pkg/cloud"
	"github.com/gophercloud/openstack-resource-controller/pkg/conditions"
	"github.com/gophercloud/openstack-resource-controller/pkg/labels"
)

const (
	OpenStackSubnetFinalizer = "openstacksubnet.k-orc.cloud"
)

// OpenStackSubnetReconciler reconciles a OpenStackSubnet object
type OpenStackSubnetReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=openstacksubnets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=openstacksubnets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=openstacksubnets/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=openstackfloatingips,verbs=list
//+kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=openstackservers,verbs=list
//+kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=openstackports,verbs=list

func (r *OpenStackSubnetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	logger := log.FromContext(ctx)
	logger = logger.WithValues("OpenStackSubnet", req.Name)

	resource := &openstackv1.OpenStackSubnet{}
	err := r.Client.Get(ctx, req.NamespacedName, resource)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("resource not found in the API")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if resource.DeletionTimestamp.IsZero() {
		finalizerUpdated := controllerutil.AddFinalizer(resource, OpenStackSubnetFinalizer)

		newLabels := map[string]string{
			openstackv1.OpenStackDependencyLabelCloud(resource.Spec.Cloud):              "",
			openstackv1.OpenStackDependencyLabelNetwork(resource.Spec.Resource.Network): "",
		}

		labelsMerger, labelsUpdated := labels.ReplacePrefixed(openstackv1.OpenStackLabelPrefix, resource.Labels, newLabels)

		if finalizerUpdated || labelsUpdated {
			logger.Info("applying labels and finalizer")
			patch := &openstackv1.OpenStackPort{}
			patch.TypeMeta = resource.TypeMeta
			patch.Finalizers = resource.GetFinalizers()
			patch.Labels = labelsMerger
			return ctrl.Result{}, apply.Apply(ctx, r.Client, resource, patch, "spec")
		}
	}

	statusPatchResource := &openstackv1.OpenStackSubnet{
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
func (r *OpenStackSubnetReconciler) reconcile(ctx context.Context, networkClient *gophercloud.ServiceClient, resource, statusPatchResource *openstackv1.OpenStackSubnet) (_ ctrl.Result, reterr error) {
	logger := log.FromContext(ctx)

	var (
		subnet *subnets.Subnet
		err    error
	)
	if openstackID := coalesce(resource.Spec.ID, resource.Status.Resource.ID); openstackID != "" {
		logger = logger.WithValues("OpenStackID", openstackID)

		subnet, err = subnets.Get(networkClient, openstackID).Extract()
		if err != nil {
			return ctrl.Result{}, err
		}
		logger.Info("OpenStack resource found")
	} else {
		var networkID string
		{
			dependency := &openstackv1.OpenStackNetwork{}
			dependencyKey := client.ObjectKey{Namespace: resource.GetNamespace(), Name: resource.Spec.Resource.Network}
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
			networkID = dependency.Status.Resource.ID
		}

		allocationPools := make([]subnets.AllocationPool, len(resource.Spec.Resource.AllocationPools))
		for i := range resource.Spec.Resource.AllocationPools {
			allocationPools[i] = subnets.AllocationPool{
				Start: resource.Spec.Resource.AllocationPools[i].Start,
				End:   resource.Spec.Resource.AllocationPools[i].End,
			}
		}

		var ipVersion gophercloud.IPVersion
		switch v := resource.Spec.Resource.IPVersion; v {
		case "IPv4":
			ipVersion = gophercloud.IPv4
		case "IPv6":
			ipVersion = gophercloud.IPv6
		default:
			return ctrl.Result{}, fmt.Errorf("invalid IP version %q. Valid instances are %q and %q", v, "IPv4", "IPv6")
		}

		hostRoutes := make([]subnets.HostRoute, len(resource.Spec.Resource.HostRoutes))
		for i := range resource.Spec.Resource.HostRoutes {
			hostRoutes[i] = subnets.HostRoute{
				DestinationCIDR: resource.Spec.Resource.HostRoutes[i].DestinationCIDR,
				NextHop:         resource.Spec.Resource.HostRoutes[i].NextHop,
			}
		}

		createOpts := subnets.CreateOpts{
			NetworkID:       networkID,
			CIDR:            resource.Spec.Resource.CIDR,
			Name:            resource.Spec.Resource.Name,
			Description:     resource.Spec.Resource.Description,
			AllocationPools: allocationPools,
			GatewayIP:       resource.Spec.Resource.GatewayIP,
			IPVersion:       ipVersion,
			EnableDHCP:      resource.Spec.Resource.EnableDHCP,
			DNSNameservers:  resource.Spec.Resource.DNSNameservers,
			ServiceTypes:    resource.Spec.Resource.ServiceTypes,
			HostRoutes:      hostRoutes,
			IPv6AddressMode: resource.Spec.Resource.IPv6AddressMode,
			IPv6RAMode:      resource.Spec.Resource.IPv6RAMode,
		}
		subnet, err = r.findAdoptee(log.IntoContext(ctx, logger), networkClient, resource, createOpts)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("unable to find adoption candidates: %w", err)
		}
		if subnet != nil {
			logger = logger.WithValues("OpenStackID", subnet.ID)
			logger.Info("OpenStack resource adopted")
		} else {
			subnet, err = subnets.Create(networkClient, createOpts).Extract()
			if err != nil {
				return ctrl.Result{}, err
			}
			logger = logger.WithValues("OpenStackID", subnet.ID)
			logger.Info("OpenStack resource created")
		}
	}

	allocationPools := make([]openstackv1.OpenStackSubnetAllocationPool, len(subnet.AllocationPools))
	for i := range subnet.AllocationPools {
		allocationPools[i] = openstackv1.OpenStackSubnetAllocationPool{
			Start: subnet.AllocationPools[i].Start,
			End:   subnet.AllocationPools[i].End,
		}
	}

	hostRoutes := make([]openstackv1.OpenStackSubnetHostRoute, len(subnet.HostRoutes))
	for i := range subnet.HostRoutes {
		hostRoutes[i] = openstackv1.OpenStackSubnetHostRoute{
			DestinationCIDR: subnet.HostRoutes[i].DestinationCIDR,
			NextHop:         subnet.HostRoutes[i].NextHop,
		}
	}

	statusPatchResource.Status.Resource = openstackv1.OpenStackSubnetResourceStatus{
		ID:              subnet.ID,
		NetworkID:       subnet.NetworkID,
		Name:            subnet.Name,
		Description:     subnet.Description,
		IPVersion:       subnet.IPVersion,
		CIDR:            subnet.CIDR,
		GatewayIP:       subnet.GatewayIP,
		DNSNameservers:  subnet.DNSNameservers,
		ServiceTypes:    subnet.ServiceTypes,
		AllocationPools: allocationPools,
		HostRoutes:      hostRoutes,
		EnableDHCP:      subnet.EnableDHCP,
		TenantID:        subnet.TenantID,
		ProjectID:       subnet.ProjectID,
		IPv6AddressMode: subnet.IPv6AddressMode,
		IPv6RAMode:      subnet.IPv6RAMode,
		SubnetPoolID:    subnet.SubnetPoolID,
		Tags:            subnet.Tags,
		RevisionNumber:  subnet.RevisionNumber,
	}

	if updated, condition := conditions.SetReadyCondition(resource, statusPatchResource); updated {
		conditions.EmitEventForCondition(r.Recorder, resource, corev1.EventTypeNormal, condition)
	}
	return ctrl.Result{}, nil
}

func (r *OpenStackSubnetReconciler) reconcileDelete(ctx context.Context, networkClient *gophercloud.ServiceClient, resource, statusPatchResource *openstackv1.OpenStackSubnet) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.V(4).Info("Checking for dependant OpenStack resources")
	referencingResources := []string{}
	for _, resourceList := range []client.ObjectList{
		&openstackv1.OpenStackPortList{},
	} {
		list := &unstructured.UnstructuredList{}
		gvk, err := apiutil.GVKForObject(resourceList, r.Client.Scheme())
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("getting GVK for resource list: %w", err)
		}
		list.SetGroupVersionKind(gvk)
		if err := r.Client.List(ctx, list,
			client.InNamespace(resource.GetNamespace()),
			client.HasLabels{openstackv1.OpenStackDependencyLabelSubnet(resource.GetName())},
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
		logger.Info("OpenStack resources still referencing this subnet", "resources", referencingResources)

		message := fmt.Sprintf("Resources of the following types still reference this subnet: %s", strings.Join(referencingResources, ", "))
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
			if err := ports.Delete(networkClient, resource.Status.Resource.ID).ExtractErr(); err != nil {
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

	if updated := controllerutil.RemoveFinalizer(resource, OpenStackSubnetFinalizer); updated {
		logger.Info("removing finalizer")
		if updated, condition := conditions.SetNotReadyConditionDeleting(resource, statusPatchResource, "Removing finalizer"); updated {
			conditions.EmitEventForCondition(r.Recorder, resource, corev1.EventTypeNormal, condition)
		}
		patch := &openstackv1.OpenStackSubnet{}
		patch.TypeMeta = resource.TypeMeta
		patch.Finalizers = resource.GetFinalizers()
		return ctrl.Result{}, apply.Apply(ctx, r.Client, resource, patch, "spec")
	}
	return ctrl.Result{}, nil
}

func (r *OpenStackSubnetReconciler) findAdoptee(ctx context.Context, networkClient *gophercloud.ServiceClient, resource client.Object, createOpts subnets.CreateOpts) (*subnets.Subnet, error) {
	adoptedIDs := make(map[string]struct{})
	{
		list := &openstackv1.OpenStackSubnetList{}
		if err := r.Client.List(ctx, list,
			client.InNamespace(resource.GetNamespace()),
		); err != nil {
			return nil, fmt.Errorf("listing OpenStackSubnets: %w", err)
		}
		for _, port := range list.Items {
			if port.GetName() != resource.GetName() && port.Status.Resource.ID != "" {
				adoptedIDs[port.Status.Resource.ID] = struct{}{}
			}
		}
	}
	listOpts := subnets.ListOpts{
		Name:            createOpts.Name,
		Description:     createOpts.Description,
		EnableDHCP:      createOpts.EnableDHCP,
		NetworkID:       createOpts.NetworkID,
		TenantID:        createOpts.TenantID,
		ProjectID:       createOpts.ProjectID,
		IPVersion:       int(createOpts.IPVersion),
		CIDR:            createOpts.CIDR,
		IPv6AddressMode: createOpts.IPv6AddressMode,
		IPv6RAMode:      createOpts.IPv6RAMode,
		// TODO: SubnetPoolID
	}

	if createOpts.GatewayIP != nil {
		listOpts.GatewayIP = *createOpts.GatewayIP
	}

	var candidates []subnets.Subnet
	err := subnets.List(networkClient, listOpts).EachPage(func(page pagination.Page) (bool, error) {
		items, err := subnets.ExtractSubnets(page)
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
func (r *OpenStackSubnetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&openstackv1.OpenStackSubnet{}).
		WithEventFilter(apply.IgnoreManagedFieldsOnly{}).
		Watches(&openstackv1.OpenStackCloud{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o client.Object) []reconcile.Request {
			// Fetch a list of all OpenStackSubnets that reference this OpenStackCloud.
			kclient := mgr.GetClient()
			logger := mgr.GetLogger()

			subnets := &openstackv1.OpenStackSubnetList{}
			if err := kclient.List(ctx, subnets,
				client.InNamespace(o.GetNamespace()),
				client.HasLabels{openstackv1.OpenStackDependencyLabelCloud(o.GetName())},
			); err != nil {
				logger.Error(err, "unable to list OpenStackSubnets")
				return nil
			}

			// Reconcile each OpenStackSubnet that is not Ready and that references this OpenStackCloud.
			reqs := make([]reconcile.Request, 0, len(subnets.Items))
			for _, subnet := range subnets.Items {
				if conditions.IsReady(&subnet) {
					continue
				}
				reqs = append(reqs, reconcile.Request{
					NamespacedName: client.ObjectKey{
						Namespace: subnet.GetNamespace(),
						Name:      subnet.GetName(),
					},
				})
				logger.V(5).Info("update of OpenStackCloud triggers reconcile of OpenStackSubnet",
					"namespace", o.GetNamespace(),
					"cloud", o.GetName(),
					"subnet", subnet.GetName())
			}
			return reqs
		})).
		Watches(&openstackv1.OpenStackNetwork{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o client.Object) []reconcile.Request {
			// Fetch a list of all OpenStackSubnets that reference this OpenStackNetwork.
			kclient := mgr.GetClient()
			logger := mgr.GetLogger()

			subnets := &openstackv1.OpenStackSubnetList{}
			if err := kclient.List(ctx, subnets,
				client.InNamespace(o.GetNamespace()),
				client.HasLabels{openstackv1.OpenStackDependencyLabelNetwork(o.GetName())},
			); err != nil {
				logger.Error(err, "unable to list OpenStackSubnets")
				return nil
			}

			// Reconcile each OpenStackSubnet that is not Ready and that references this OpenStackNetwork.
			reqs := make([]reconcile.Request, 0, len(subnets.Items))
			for _, subnet := range subnets.Items {
				if conditions.IsReady(&subnet) {
					continue
				}
				reqs = append(reqs, reconcile.Request{
					NamespacedName: client.ObjectKey{
						Namespace: subnet.GetNamespace(),
						Name:      subnet.GetName(),
					},
				})
				logger.V(5).Info("update of OpenStackNetwork triggers reconcile of OpenStackSubnet",
					"namespace", o.GetNamespace(),
					"cloud", o.GetName(),
					"subnet", subnet.GetName())
			}
			return reqs
		})).
		Complete(r)
}
