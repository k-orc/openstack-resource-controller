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
	"github.com/gophercloud/gophercloud/pagination"
	openstackv1 "github.com/gophercloud/openstack-resource-controller/api/v1alpha1"
	"github.com/gophercloud/openstack-resource-controller/pkg/apply"
	"github.com/gophercloud/openstack-resource-controller/pkg/cloud"
	"github.com/gophercloud/openstack-resource-controller/pkg/conditions"
	"github.com/gophercloud/openstack-resource-controller/pkg/labels"
)

const (
	OpenStackPortFinalizer = "openstackport.k-orc.cloud"
)

// OpenStackPortReconciler reconciles a OpenStackPort object
type OpenStackPortReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=openstackports,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=openstackports/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=openstackports/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=openstackfloatingips,verbs=list
//+kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=openstackservers,verbs=list

func (r *OpenStackPortReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	logger := log.FromContext(ctx)
	logger = logger.WithValues("OpenStackPort", req.Name)

	resource := &openstackv1.OpenStackPort{}
	err := r.Client.Get(ctx, req.NamespacedName, resource)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("resource not found in the API")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if resource.DeletionTimestamp.IsZero() {
		finalizerUpdated := controllerutil.AddFinalizer(resource, OpenStackPortFinalizer)

		newLabels := map[string]string{
			openstackv1.OpenStackDependencyLabelCloud(resource.Spec.Cloud):              "",
			openstackv1.OpenStackDependencyLabelNetwork(resource.Spec.Resource.Network): "",
		}
		for _, sg := range resource.Spec.Resource.SecurityGroups {
			newLabels[openstackv1.OpenStackDependencyLabelSecurityGroup(sg)] = ""
		}
		for _, ip := range resource.Spec.Resource.FixedIPs {
			if ip.Subnet != "" {
				newLabels[openstackv1.OpenStackDependencyLabelSubnet(ip.Subnet)] = ""
			}
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

	statusPatchResource := &openstackv1.OpenStackPort{
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
func (r *OpenStackPortReconciler) reconcile(ctx context.Context, networkClient *gophercloud.ServiceClient, resource, statusPatchResource *openstackv1.OpenStackPort) (_ ctrl.Result, reterr error) {
	logger := log.FromContext(ctx)

	var (
		port *ports.Port
		err  error
	)
	if openstackID := coalesce(resource.Spec.ID, resource.Status.Resource.ID); openstackID != "" {
		logger = logger.WithValues("OpenStackID", openstackID)

		port, err = ports.Get(networkClient, openstackID).Extract()
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

		type FixedIP struct {
			IPAddress string `json:"ip_address,omitempty"`
			SubnetID  string `json:"subnet_id,omitempty"`
		}
		fixedIPs := make([]FixedIP, len(resource.Spec.Resource.FixedIPs))
		for i, fixedIP := range resource.Spec.Resource.FixedIPs {
			gophercloudFixedIP := FixedIP{IPAddress: fixedIP.IPAddress}
			if fixedIP.Subnet != "" {
				dependency := &openstackv1.OpenStackSubnet{}
				dependencyKey := client.ObjectKey{Namespace: resource.GetNamespace(), Name: fixedIP.Subnet}
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
				gophercloudFixedIP.SubnetID = dependency.Status.Resource.ID
			}
			fixedIPs[i] = gophercloudFixedIP
		}

		allowedAddressPairs := make([]ports.AddressPair, len(resource.Spec.Resource.AllowedAddressPairs))
		for i, pair := range resource.Spec.Resource.AllowedAddressPairs {
			allowedAddressPairs[i] = ports.AddressPair{
				IPAddress:  pair.IPAddress,
				MACAddress: pair.MACAddress,
			}
		}

		createOpts := ports.CreateOpts{
			NetworkID:             networkID,
			Name:                  resource.Spec.Resource.Name,
			Description:           resource.Spec.Resource.Description,
			AdminStateUp:          resource.Spec.Resource.AdminStateUp,
			MACAddress:            resource.Spec.Resource.MACAddress,
			FixedIPs:              fixedIPs,
			DeviceOwner:           resource.Spec.Resource.DeviceOwner,
			TenantID:              resource.Spec.Resource.TenantID,
			ProjectID:             resource.Spec.Resource.ProjectID,
			SecurityGroups:        &securityGroupIDs,
			AllowedAddressPairs:   allowedAddressPairs,
			PropagateUplinkStatus: resource.Spec.Resource.PropagateUplinkStatus,
			// TODO: What is ValueSpecs? Can it be a way to set the
			// properties I'm missing in Gophercloud?
			// ValueSpecs: nil,
		}

		fixedIPOpts := make([]ports.FixedIPOpts, len(fixedIPs))
		for i := range fixedIPs {
			fixedIPOpts[i] = ports.FixedIPOpts{
				IPAddress: fixedIPs[i].IPAddress,
				SubnetID:  fixedIPs[i].SubnetID,
			}
		}

		port, err = r.portFind(log.IntoContext(ctx, logger), networkClient, resource, createOpts, fixedIPOpts)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("unable to find adoption candidates: %w", err)
		}
		if port != nil {
			logger = logger.WithValues("OpenStackID", port.ID)
			logger.Info("OpenStack resource adopted")
		} else {
			port, err = ports.Create(networkClient, createOpts).Extract()
			if err != nil {
				return ctrl.Result{}, err
			}
			logger = logger.WithValues("OpenStackID", port.ID)
			logger.Info("OpenStack resource created")
		}
	}

	fixedIPs := make([]openstackv1.OpenStackPortStatusFixedIP, len(port.FixedIPs))
	for i, ip := range port.FixedIPs {
		fixedIPs[i] = openstackv1.OpenStackPortStatusFixedIP{
			IPAddress: ip.IPAddress,
			SubnetID:  ip.SubnetID,
		}
	}

	allowedAddressPairs := make([]openstackv1.OpenStackPortAllowedAddressPair, len(port.AllowedAddressPairs))
	for i, ap := range port.AllowedAddressPairs {
		allowedAddressPairs[i] = openstackv1.OpenStackPortAllowedAddressPair{
			IPAddress:  ap.IPAddress,
			MACAddress: ap.MACAddress,
		}
	}

	statusPatchResource.Status.Resource = openstackv1.OpenStackPortResourceStatus{
		ID:                    port.ID,
		NetworkID:             port.NetworkID,
		Name:                  port.Name,
		Description:           port.Description,
		AdminStateUp:          port.AdminStateUp,
		Status:                port.Status,
		MACAddress:            port.MACAddress,
		FixedIPs:              fixedIPs,
		TenantID:              port.TenantID,
		ProjectID:             port.ProjectID,
		DeviceOwner:           port.DeviceOwner,
		SecurityGroups:        port.SecurityGroups,
		DeviceID:              port.DeviceID,
		AllowedAddressPairs:   allowedAddressPairs,
		Tags:                  port.Tags,
		PropagateUplinkStatus: port.PropagateUplinkStatus,
		ValueSpecs:            nil,
		RevisionNumber:        port.RevisionNumber,
		CreatedAt:             port.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:             port.UpdatedAt.UTC().Format(time.RFC3339),
	}

	if updated, condition := conditions.SetReadyCondition(resource, statusPatchResource); updated {
		conditions.EmitEventForCondition(r.Recorder, resource, corev1.EventTypeNormal, condition)
	}
	return ctrl.Result{}, nil
}

func (r *OpenStackPortReconciler) reconcileDelete(ctx context.Context, networkClient *gophercloud.ServiceClient, resource, statusPatchResource *openstackv1.OpenStackPort) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.V(4).Info("Checking for dependant OpenStack resources")
	referencingResources := []string{}
	for _, resourceList := range []client.ObjectList{
		&openstackv1.OpenStackFloatingIPList{},
		&openstackv1.OpenStackRouterList{},
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
			client.HasLabels{openstackv1.OpenStackDependencyLabelPort(resource.GetName())},
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
		logger.Info("OpenStack resources still referencing this port", "resources", referencingResources)

		message := fmt.Sprintf("Resources of the following types still reference this port: %s", strings.Join(referencingResources, ", "))
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

	if updated := controllerutil.RemoveFinalizer(resource, OpenStackPortFinalizer); updated {
		logger.Info("removing finalizer")
		if updated, condition := conditions.SetNotReadyConditionDeleting(resource, statusPatchResource, "Removing finalizer"); updated {
			conditions.EmitEventForCondition(r.Recorder, resource, corev1.EventTypeNormal, condition)
		}
		patch := &openstackv1.OpenStackPort{}
		patch.TypeMeta = resource.TypeMeta
		patch.Finalizers = resource.GetFinalizers()
		return ctrl.Result{}, apply.Apply(ctx, r.Client, resource, patch, "spec")
	}
	return ctrl.Result{}, nil
}

func portEquals(candidate ports.Port, resource ports.CreateOpts) bool {
	if len(candidate.AllowedAddressPairs) != len(resource.AllowedAddressPairs) {
		return false
	}
	reciprocated := make(map[int]struct{})
	for i := range candidate.AllowedAddressPairs {
		var foundInOpts bool
		for j := range resource.AllowedAddressPairs {
			if _, ok := reciprocated[j]; !ok && candidate.AllowedAddressPairs[i] == resource.AllowedAddressPairs[j] {
				foundInOpts = true
				reciprocated[j] = struct{}{}
				break
			}
		}
		if !foundInOpts {
			return false
		}
	}
	if resource.SecurityGroups != nil {
		if len(candidate.SecurityGroups) != len(*resource.SecurityGroups) {
			return false
		}
		reciprocated := make(map[int]struct{})
		for i := range candidate.SecurityGroups {
			var foundInOpts bool
			for j := range *resource.SecurityGroups {
				if _, ok := reciprocated[j]; !ok && candidate.SecurityGroups[i] == (*resource.SecurityGroups)[j] {
					foundInOpts = true
					reciprocated[j] = struct{}{}
					break
				}
			}
			if !foundInOpts {
				return false
			}
		}
	}
	if resource.PropagateUplinkStatus != nil && candidate.PropagateUplinkStatus != *resource.PropagateUplinkStatus {
		return false
	}
	return true
}

func (r *OpenStackPortReconciler) portFind(ctx context.Context, imageClient *gophercloud.ServiceClient, resource client.Object, createOpts ports.CreateOpts, fixedIPOpts []ports.FixedIPOpts) (*ports.Port, error) {
	adoptedIDs := make(map[string]struct{})
	{
		images := &openstackv1.OpenStackImageList{}
		if err := r.Client.List(ctx, images,
			client.InNamespace(resource.GetNamespace()),
		); err != nil {
			return nil, fmt.Errorf("listing OpenStackPorts: %w", err)
		}
		for _, port := range images.Items {
			if port.GetName() != resource.GetName() && port.Status.Resource.ID != "" {
				adoptedIDs[port.Status.Resource.ID] = struct{}{}
			}
		}
	}
	listOpts := ports.ListOpts{
		Name:         createOpts.Name,
		Description:  createOpts.Description,
		AdminStateUp: createOpts.AdminStateUp,
		NetworkID:    createOpts.NetworkID,
		TenantID:     createOpts.TenantID,
		ProjectID:    createOpts.ProjectID,
		DeviceOwner:  createOpts.DeviceOwner,
		MACAddress:   createOpts.MACAddress,
		DeviceID:     createOpts.DeviceID,
		FixedIPs:     fixedIPOpts,
	}

	var candidates []ports.Port
	err := ports.List(imageClient, listOpts).EachPage(func(page pagination.Page) (bool, error) {
		items, err := ports.ExtractPorts(page)
		if err != nil {
			return false, fmt.Errorf("extracting resources: %w", err)
		}
		for i := range items {
			if _, ok := adoptedIDs[items[i].ID]; !ok && portEquals(items[i], createOpts) {
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
func (r *OpenStackPortReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&openstackv1.OpenStackPort{}).
		WithEventFilter(apply.IgnoreManagedFieldsOnly{}).
		Watches(&openstackv1.OpenStackCloud{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o client.Object) []reconcile.Request {
			// Fetch a list of all OpenStackPorts that reference this OpenStackCloud.
			kclient := mgr.GetClient()
			logger := mgr.GetLogger()

			ports := &openstackv1.OpenStackPortList{}
			if err := kclient.List(ctx, ports,
				client.InNamespace(o.GetNamespace()),
				client.HasLabels{openstackv1.OpenStackDependencyLabelCloud(o.GetName())},
			); err != nil {
				logger.Error(err, "unable to list OpenStackPorts")
				return nil
			}

			// Reconcile each OpenStackPort that is not Ready and that references this OpenStackCloud.
			reqs := make([]reconcile.Request, 0, len(ports.Items))
			for _, port := range ports.Items {
				if conditions.IsReady(&port) {
					continue
				}
				reqs = append(reqs, reconcile.Request{
					NamespacedName: client.ObjectKey{
						Namespace: port.GetNamespace(),
						Name:      port.GetName(),
					},
				})
				logger.V(5).Info("update of OpenStackCloud triggers reconcile of OpenStackPort",
					"namespace", o.GetNamespace(),
					"cloud", o.GetName(),
					"port", port.GetName())
			}
			return reqs
		})).
		Watches(&openstackv1.OpenStackNetwork{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o client.Object) []reconcile.Request {
			// Fetch a list of all OpenStackPorts that reference this OpenStackNetwork.
			kclient := mgr.GetClient()
			logger := mgr.GetLogger()

			ports := &openstackv1.OpenStackPortList{}
			if err := kclient.List(ctx, ports,
				client.InNamespace(o.GetNamespace()),
				client.HasLabels{openstackv1.OpenStackDependencyLabelNetwork(o.GetName())},
			); err != nil {
				logger.Error(err, "unable to list OpenStackPorts")
				return nil
			}

			// Reconcile each OpenStackPort that is not Ready and that references this OpenStackNetwork.
			reqs := make([]reconcile.Request, 0, len(ports.Items))
			for _, port := range ports.Items {
				if conditions.IsReady(&port) {
					continue
				}
				reqs = append(reqs, reconcile.Request{
					NamespacedName: client.ObjectKey{
						Namespace: port.GetNamespace(),
						Name:      port.GetName(),
					},
				})
				logger.V(5).Info("update of OpenStackNetwork triggers reconcile of OpenStackPort",
					"namespace", o.GetNamespace(),
					"cloud", o.GetName(),
					"port", port.GetName())
			}
			return reqs
		})).
		Watches(&openstackv1.OpenStackSecurityGroup{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o client.Object) []reconcile.Request {
			// Fetch a list of all OpenStackPorts that reference this OpenStackSecurityGroup.
			kclient := mgr.GetClient()
			logger := mgr.GetLogger()

			ports := &openstackv1.OpenStackPortList{}
			if err := kclient.List(ctx, ports,
				client.InNamespace(o.GetNamespace()),
				client.HasLabels{openstackv1.OpenStackDependencyLabelSecurityGroup(o.GetName())},
			); err != nil {
				logger.Error(err, "unable to list OpenStackPorts")
				return nil
			}

			// Reconcile each OpenStackPort that is not Ready and that references this OpenStackSecurityGroup.
			reqs := make([]reconcile.Request, 0, len(ports.Items))
			for _, port := range ports.Items {
				if conditions.IsReady(&port) {
					continue
				}
				reqs = append(reqs, reconcile.Request{
					NamespacedName: client.ObjectKey{
						Namespace: port.GetNamespace(),
						Name:      port.GetName(),
					},
				})
				logger.V(5).Info("update of OpenStackSecurityGroup triggers reconcile of OpenStackPort",
					"namespace", o.GetNamespace(),
					"cloud", o.GetName(),
					"port", port.GetName())
			}
			return reqs
		})).
		Watches(&openstackv1.OpenStackSubnet{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o client.Object) []reconcile.Request {
			// Fetch a list of all OpenStackPorts that reference this OpenStackSubnet.
			kclient := mgr.GetClient()
			logger := mgr.GetLogger()

			ports := &openstackv1.OpenStackPortList{}
			if err := kclient.List(ctx, ports,
				client.InNamespace(o.GetNamespace()),
				client.HasLabels{openstackv1.OpenStackDependencyLabelSubnet(o.GetName())},
			); err != nil {
				logger.Error(err, "unable to list OpenStackPorts")
				return nil
			}

			// Reconcile each OpenStackPort that is not Ready and that references this OpenStackSubnet.
			reqs := make([]reconcile.Request, 0, len(ports.Items))
			for _, port := range ports.Items {
				if conditions.IsReady(&port) {
					continue
				}
				reqs = append(reqs, reconcile.Request{
					NamespacedName: client.ObjectKey{
						Namespace: port.GetNamespace(),
						Name:      port.GetName(),
					},
				})
				logger.V(5).Info("update of OpenStackSubnet triggers reconcile of OpenStackPort",
					"namespace", o.GetNamespace(),
					"cloud", o.GetName(),
					"port", port.GetName())
			}
			return reqs
		})).
		Complete(r)
}
