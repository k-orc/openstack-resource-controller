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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/dns"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/external"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/mtu"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/portsecurity"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/provider"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/qos/policies"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/vlantransparent"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/pagination"
	openstackv1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/pkg/apply"
	"github.com/k-orc/openstack-resource-controller/pkg/cloud"
	"github.com/k-orc/openstack-resource-controller/pkg/conditions"
	"github.com/k-orc/openstack-resource-controller/pkg/labels"
)

const (
	OpenStackNetworkFinalizer = "openstacknetwork.k-orc.cloud"
)

// OpenStackNetworkReconciler reconciles a OpenStackNetwork object
type OpenStackNetworkReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=openstacknetworks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=openstacknetworks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=openstacknetworks/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *OpenStackNetworkReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	logger := log.FromContext(ctx)
	logger = logger.WithValues("OpenStackNetwork", req.Name)

	resource := &openstackv1.OpenStackNetwork{}
	err := r.Client.Get(ctx, req.NamespacedName, resource)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("resource not found in the API")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if resource.DeletionTimestamp.IsZero() {
		finalizerUpdated := controllerutil.AddFinalizer(resource, OpenStackNetworkFinalizer)

		newLabels := map[string]string{
			openstackv1.OpenStackDependencyLabelCloud(resource.Spec.Cloud): "",
		}

		labelsMerger, labelsUpdated := labels.ReplacePrefixed(openstackv1.OpenStackLabelPrefix, resource.Labels, newLabels)

		if finalizerUpdated || labelsUpdated {
			logger.Info("applying labels and finalizer")
			patch := &openstackv1.OpenStackNetwork{}
			patch.TypeMeta = resource.TypeMeta
			patch.Finalizers = resource.GetFinalizers()
			patch.Labels = labelsMerger
			return ctrl.Result{}, apply.Apply(ctx, r.Client, resource, patch, "spec")
		}
	}

	statusPatchResource := &openstackv1.OpenStackNetwork{
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
// TODO: potentially handle (some?) modifications accepted in OpenStack, as in `openstack network set`
func (r *OpenStackNetworkReconciler) reconcile(ctx context.Context, networkClient *gophercloud.ServiceClient, resource, statusPatchResource *openstackv1.OpenStackNetwork) (_ ctrl.Result, reterr error) {
	logger := log.FromContext(ctx)

	var (
		network *networkExtended
		err     error
	)
	if openstackID := coalesce(resource.Spec.ID, resource.Status.Resource.ID); openstackID != "" {
		logger = logger.WithValues("OpenStackID", openstackID)

		var n networkExtended
		err = networks.Get(networkClient, openstackID).ExtractInto(&n)
		if err != nil {
			return ctrl.Result{}, err
		}
		network = &n
		logger.Info("OpenStack resource found")
	} else {
		segments := make([]provider.Segment, len(resource.Spec.Resource.Segments))
		for i := range resource.Spec.Resource.Segments {
			segments[i] = provider.Segment{
				PhysicalNetwork: resource.Spec.Resource.Segments[i].ProviderPhysicalNetwork,
				NetworkType:     resource.Spec.Resource.Segments[i].ProviderNetworkType,
				SegmentationID:  int(resource.Spec.Resource.Segments[i].ProviderSegmentationID),
			}
		}
		createOpts := networkOpts{
			CreateOpts: networks.CreateOpts{
				AdminStateUp:          resource.Spec.Resource.AdminStateUp,
				Name:                  resource.Spec.Resource.Name,
				Description:           resource.Spec.Resource.Description,
				Shared:                resource.Spec.Resource.Shared,
				TenantID:              resource.Spec.Resource.TenantID,
				ProjectID:             resource.Spec.Resource.ProjectID,
				AvailabilityZoneHints: resource.Spec.Resource.AvailabilityZoneHints,
			},
			MTU:                     int(resource.Spec.Resource.MTU),
			DNSDomain:               resource.Spec.Resource.DNSDomain,
			PortSecurityEnabled:     resource.Spec.Resource.PortSecurityEnabled,
			QoSPolicyID:             resource.Spec.Resource.QoSPolicyID,
			External:                resource.Spec.Resource.External,
			ProviderPhysicalNetwork: resource.Spec.Resource.Segment.ProviderPhysicalNetwork,
			ProviderNetworkType:     resource.Spec.Resource.Segment.ProviderNetworkType,
			ProviderSegmentationID:  int(resource.Spec.Resource.Segment.ProviderSegmentationID),
			Segments:                segments,
			VLANTransparent:         resource.Spec.Resource.VLANTransparent,
			IsDefault:               resource.Spec.Resource.IsDefault,
		}

		network, err = r.findAdoptee(log.IntoContext(ctx, logger), networkClient, resource, createOpts)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("unable to find adoption candidates: %w", err)
		}
		if network != nil {
			logger = logger.WithValues("OpenStackID", network.ID)
			logger.Info("OpenStack resource adopted")
		} else {
			var wrapper struct {
				Resource networkExtended `json:"network"`
			}
			err = networks.Create(networkClient, createOpts).ExtractInto(&wrapper)
			if err != nil {
				return ctrl.Result{}, err
			}
			network = &wrapper.Resource
			logger = logger.WithValues("OpenStackID", network.ID)
			logger.Info("OpenStack resource created")
		}
	}

	segments := make([]openstackv1.OpenStackNetworkSegment, len(network.Segments))
	for i := range network.Segments {
		segments[i] = openstackv1.OpenStackNetworkSegment{
			ProviderNetworkType:     network.Segments[i].NetworkType,
			ProviderPhysicalNetwork: network.Segments[i].PhysicalNetwork,
			ProviderSegmentationID:  int32(network.Segments[i].SegmentationID),
		}
	}

	statusPatchResource.Status.Resource = openstackv1.OpenStackNetworkResourceStatus{
		AdminStateUp:          network.AdminStateUp,
		AvailabilityZoneHints: network.AvailabilityZoneHints,
		AvailabilityZones:     network.AvailabilityZones,
		CreatedAt:             network.CreatedAt.UTC().Format(time.RFC3339),
		DNSDomain:             network.DNSDomain,
		ID:                    network.ID,
		IPV4AddressScope:      network.IPV4AddressScope,
		IPV6AddressScope:      network.IPV6AddressScope,
		L2Adjacency:           network.L2Adjacency,
		MTU:                   int32(network.MTU),
		Name:                  network.Name,
		PortSecurityEnabled:   network.PortSecurityEnabled,
		ProjectID:             network.ProjectID,
		Segment:               openstackv1.OpenStackNetworkSegment{},
		QoSPolicyID:           network.QoSPolicyID,
		RevisionNumber:        int32(network.RevisionNumber),
		External:              network.External,
		Segments:              segments,
		Shared:                network.Shared,
		Status:                network.Status,
		Subnets:               network.Subnets,
		TenantID:              network.TenantID,
		UpdatedAt:             network.UpdatedAt.UTC().Format(time.RFC3339),
		VLANTransparent:       network.VLANTransparent,
		Description:           network.Description,
		IsDefault:             network.IsDefault,
		Tags:                  network.Tags,
	}

	if updated, condition := conditions.SetReadyCondition(resource, statusPatchResource); updated {
		conditions.EmitEventForCondition(r.Recorder, resource, corev1.EventTypeNormal, condition)
	}
	return ctrl.Result{}, nil
}

func (r *OpenStackNetworkReconciler) reconcileDelete(ctx context.Context, networkClient *gophercloud.ServiceClient, resource, statusPatchResource *openstackv1.OpenStackNetwork) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.V(4).Info("Checking for dependant OpenStack resources")
	referencingResources := []string{}
	for _, resourceList := range []client.ObjectList{
		&openstackv1.OpenStackNetworkList{},
		&openstackv1.OpenStackSubnetList{},
		&openstackv1.OpenStackPortList{},
		&openstackv1.OpenStackFloatingIPList{},
	} {
		list := &unstructured.UnstructuredList{}
		gvk, err := apiutil.GVKForObject(resourceList, r.Client.Scheme())
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("getting GVK for resource list: %w", err)
		}
		list.SetGroupVersionKind(gvk)
		if err := r.Client.List(ctx, list,
			client.InNamespace(resource.GetNamespace()),
			client.HasLabels{openstackv1.OpenStackDependencyLabelNetwork(resource.GetName())},
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
		logger.Info("OpenStack resources still referencing this network", "resources", referencingResources)

		message := fmt.Sprintf("Resources of the following types still reference this network: %s", strings.Join(referencingResources, ", "))
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
			if err := networks.Delete(networkClient, resource.Status.Resource.ID).ExtractErr(); err != nil {
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

	if updated := controllerutil.RemoveFinalizer(resource, OpenStackNetworkFinalizer); updated {
		logger.Info("removing finalizer")
		if updated, condition := conditions.SetNotReadyConditionDeleting(resource, statusPatchResource, "Removing finalizer"); updated {
			conditions.EmitEventForCondition(r.Recorder, resource, corev1.EventTypeNormal, condition)
		}
		patch := &openstackv1.OpenStackNetwork{}
		patch.TypeMeta = resource.TypeMeta
		patch.Finalizers = resource.GetFinalizers()
		return ctrl.Result{}, apply.Apply(ctx, r.Client, resource, patch, "spec")
	}
	return ctrl.Result{}, nil
}

func (r *OpenStackNetworkReconciler) findAdoptee(ctx context.Context, networkClient *gophercloud.ServiceClient, resource client.Object, createOpts networkOpts) (*networkExtended, error) {
	adoptedIDs := make(map[string]struct{})
	{
		list := &openstackv1.OpenStackNetworkList{}
		if err := r.Client.List(ctx, list,
			client.InNamespace(resource.GetNamespace()),
		); err != nil {
			return nil, fmt.Errorf("listing OpenStackNetworks: %w", err)
		}
		for _, item := range list.Items {
			if item.GetName() != resource.GetName() && item.Status.Resource.ID != "" {
				adoptedIDs[item.Status.Resource.ID] = struct{}{}
			}
		}
	}

	var candidates []*networkExtended
	err := networks.List(networkClient, createOpts).EachPage(func(page pagination.Page) (bool, error) {
		var items []*networkExtended
		if err := networks.ExtractNetworksInto(page, &items); err != nil {
			return false, fmt.Errorf("extracting resources: %w", err)
		}

		for i := range items {
			if _, ok := adoptedIDs[items[i].ID]; !ok && items[i].Equals(createOpts) {
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
		return candidates[0], nil
	default:
		return nil, fmt.Errorf("found %d possible candidates", n)
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *OpenStackNetworkReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&openstackv1.OpenStackNetwork{}).
		WithEventFilter(apply.IgnoreManagedFieldsOnly{}).
		Watches(&openstackv1.OpenStackCloud{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o client.Object) []reconcile.Request {
			// Fetch a list of all OpenStackNetworks that reference this OpenStackCloud.
			kclient := mgr.GetClient()
			logger := mgr.GetLogger()

			networks := &openstackv1.OpenStackNetworkList{}
			if err := kclient.List(ctx, networks,
				client.InNamespace(o.GetNamespace()),
				client.HasLabels{openstackv1.OpenStackDependencyLabelCloud(o.GetName())},
			); err != nil {
				logger.Error(err, "unable to list OpenStackNetworks")
				return nil
			}

			// Reconcile each OpenStackNetwork that is not Ready and that references this OpenStackCloud.
			reqs := make([]reconcile.Request, 0, len(networks.Items))
			for _, network := range networks.Items {
				if conditions.IsReady(&network) {
					continue
				}
				reqs = append(reqs, reconcile.Request{
					NamespacedName: client.ObjectKey{
						Namespace: network.GetNamespace(),
						Name:      network.GetName(),
					},
				})
				logger.V(5).Info("update of OpenStackCloud triggers reconcile of OpenStackNetwork",
					"namespace", o.GetNamespace(),
					"cloud", o.GetName(),
					"network", network.GetName())
			}
			return reqs
		})).
		Complete(r)
}

type networkOpts struct {
	networks.CreateOpts
	MTU                     int
	DNSDomain               string
	PortSecurityEnabled     *bool
	QoSPolicyID             string
	External                *bool
	ProviderPhysicalNetwork string
	ProviderNetworkType     string
	ProviderSegmentationID  int
	Segments                []provider.Segment
	VLANTransparent         *bool
	IsDefault               *bool
}

func (opts networkOpts) ToNetworkCreateMap() (map[string]interface{}, error) {
	var createOpts networks.CreateOptsBuilder = opts.CreateOpts
	if opts.MTU != 0 {
		createOpts = mtu.CreateOptsExt{
			CreateOptsBuilder: createOpts,
			MTU:               opts.MTU,
		}
	}
	if opts.DNSDomain != "" {
		createOpts = dns.NetworkCreateOptsExt{
			CreateOptsBuilder: createOpts,
			DNSDomain:         opts.DNSDomain,
		}
	}
	if opts.PortSecurityEnabled != nil {
		createOpts = portsecurity.NetworkCreateOptsExt{
			CreateOptsBuilder:   createOpts,
			PortSecurityEnabled: opts.PortSecurityEnabled,
		}
	}
	if opts.QoSPolicyID != "" {
		createOpts = policies.NetworkCreateOptsExt{
			CreateOptsBuilder: createOpts,
			QoSPolicyID:       opts.QoSPolicyID,
		}
	}
	if opts.External != nil {
		createOpts = external.CreateOptsExt{
			CreateOptsBuilder: createOpts,
			External:          opts.External,
		}
	}
	if len(opts.Segments) > 0 {
		createOpts = provider.CreateOptsExt{
			CreateOptsBuilder: createOpts,
			Segments:          opts.Segments,
		}
	}
	if opts.VLANTransparent != nil {
		createOpts = vlantransparent.CreateOptsExt{
			CreateOptsBuilder: createOpts,
			VLANTransparent:   opts.VLANTransparent,
		}
	}

	base, err := createOpts.ToNetworkCreateMap()
	if err != nil {
		return nil, err
	}

	providerMap := base["network"].(map[string]interface{})
	if opts.ProviderPhysicalNetwork != "" {
		providerMap["provider:physical_network"] = opts.ProviderPhysicalNetwork
	}
	if opts.ProviderNetworkType != "" {
		providerMap["provider:network_type"] = opts.ProviderNetworkType
	}
	if opts.ProviderSegmentationID != 0 {
		providerMap["provider:segmentation_id"] = opts.ProviderSegmentationID
	}
	if opts.IsDefault != nil {
		providerMap["is_default"] = opts.IsDefault
	}

	return base, nil
}

func (opts networkOpts) ToNetworkListQuery() (string, error) {
	// missing: mtu, dns, policies, portsecurity, provider
	var listOpts networks.ListOptsBuilder = networks.ListOpts{
		Name:         opts.Name,
		Description:  opts.Description,
		AdminStateUp: opts.AdminStateUp,
		TenantID:     opts.TenantID,
		ProjectID:    opts.ProjectID,
		Shared:       opts.Shared,
	}

	if opts.VLANTransparent != nil {
		listOpts = vlantransparent.ListOptsExt{
			ListOptsBuilder: listOpts,
			VLANTransparent: opts.VLANTransparent,
		}
	}

	if opts.External != nil {
		listOpts = external.ListOptsExt{
			ListOptsBuilder: listOpts,
			External:        opts.External,
		}
	}
	return listOpts.ToNetworkListQuery()
}

type networkExtended struct {
	networks.Network
	mtu.NetworkMTUExt
	dns.NetworkDNSExt
	external.NetworkExternalExt
	policies.NetworkCreateOptsExt
	provider.NetworkProviderExt
	vlantransparent.TransparentExt
	AvailabilityZones   []string `json:"availability_zones"`
	PortSecurityEnabled *bool    `json:"port_security_enabled"`
	QoSPolicyID         string   `json:"qos_policy_id"`
	IsDefault           *bool    `json:"is_default"`
	IPV4AddressScope    string   `json:"ipv4_address_scope"`
	IPV6AddressScope    string   `json:"ipv6_address_scope"`
	L2Adjacency         *bool    `json:"l2_adjacency"`
}

// Equals checks for equality the fields that coulnd't be added to the List
// query
func (n *networkExtended) Equals(opts networkOpts) bool {
	// mtu
	if opts.MTU != 0 && n.MTU != opts.MTU {
		return false
	}

	// dns
	if opts.DNSDomain != "" && n.DNSDomain != opts.DNSDomain {
		return false
	}

	// policies
	if opts.QoSPolicyID != "" && n.QoSPolicyID != opts.QoSPolicyID {
		return false
	}

	// portsecurity
	if opts.PortSecurityEnabled != nil && n.PortSecurityEnabled != opts.PortSecurityEnabled {
		return false
	}

	// provider
	if n.PhysicalNetwork != opts.ProviderPhysicalNetwork {
		return false
	}
	if n.NetworkType != opts.ProviderNetworkType {
		return false
	}
	if opts.ProviderSegmentationID != 0 && n.SegmentationID != strconv.Itoa(opts.ProviderSegmentationID) {
		return false
	}
	if len(n.Segments) != len(opts.Segments) {
		return false
	}
	if !sliceContentCompare(opts.Segments, n.Segments, func(createOpts, current provider.Segment) bool {
		return createOpts.NetworkType == current.NetworkType &&
			createOpts.PhysicalNetwork == current.PhysicalNetwork &&
			(createOpts.SegmentationID == 0 || createOpts.SegmentationID == current.SegmentationID)
	}) {
		return false
	}
	return true
}
