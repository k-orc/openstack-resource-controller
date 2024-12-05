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

package subnet

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/attributestags"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/subnets"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"k8s.io/utils/set"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/common"
	osclients "github.com/k-orc/openstack-resource-controller/internal/osclients"
	"github.com/k-orc/openstack-resource-controller/internal/util/applyconfigs"
	orcerrors "github.com/k-orc/openstack-resource-controller/internal/util/errors"
	"github.com/k-orc/openstack-resource-controller/internal/util/neutrontags"
	orcapplyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/pkg/clients/applyconfiguration/api/v1alpha1"
)

// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=subnets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=subnets/status,verbs=get;update;patch

func (r *orcSubnetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	orcObject := &orcv1alpha1.Subnet{}
	err := r.client.Get(ctx, req.NamespacedName, orcObject)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if !orcObject.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, orcObject)
	}

	return r.reconcileNormal(ctx, orcObject)
}

func (r *orcSubnetReconciler) getNetworkClient(ctx context.Context, orcSubnet *orcv1alpha1.Subnet) (osclients.NetworkClient, error) {
	log := ctrl.LoggerFrom(ctx)

	clientScope, err := r.scopeFactory.NewClientScopeFromObject(ctx, r.client, log, orcSubnet)
	if err != nil {
		return nil, err
	}
	return clientScope.NewNetworkClient()
}

func (r *orcSubnetReconciler) reconcileNormal(ctx context.Context, orcObject *orcv1alpha1.Subnet) (_ ctrl.Result, err error) {
	log := ctrl.LoggerFrom(ctx)
	log.V(3).Info("Reconciling subnet")

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

	orcNetwork := &orcv1alpha1.Network{}
	if err := r.client.Get(ctx, client.ObjectKey{Name: string(orcObject.Spec.NetworkRef), Namespace: orcObject.Namespace}, orcNetwork); err != nil {
		if apierrors.IsNotFound(err) {
			addStatus(withProgressMessage(fmt.Sprintf("waiting for network object %s to be created", orcObject.Spec.NetworkRef)))
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if !orcv1alpha1.IsAvailable(orcNetwork) {
		addStatus(withProgressMessage(fmt.Sprintf("waiting for network object %s to be available", orcObject.Spec.NetworkRef)))
		return ctrl.Result{}, nil
	}

	if orcNetwork.Status.ID == nil {
		return ctrl.Result{}, fmt.Errorf("network %s is available but status.ID is not set", orcNetwork.Name)
	}

	// Don't add finalizer until parent network is available to avoid unnecessary reconcile on delete
	if !controllerutil.ContainsFinalizer(orcObject, Finalizer) {
		patch := common.SetFinalizerPatch(orcObject, Finalizer)
		return ctrl.Result{}, r.client.Patch(ctx, orcObject, patch, client.ForceOwnership, ssaFieldOwner(SSAFinalizerTxn))
	}

	networkClient, err := r.getNetworkClient(ctx, orcObject)
	if err != nil {
		return ctrl.Result{}, err
	}

	networkID := orcv1alpha1.UUID(*orcNetwork.Status.ID)

	osResource, waitingOnExternal, err := getOSResourceFromObject(ctx, log, orcObject, networkID, networkClient)
	if err != nil {
		return ctrl.Result{}, err
	}
	if waitingOnExternal {
		log.V(3).Info("OpenStack resource does not yet exist")
		addStatus(withProgressMessage("Waiting for OpenStack resource to be created externally"))
		return ctrl.Result{RequeueAfter: externalUpdatePollingPeriod}, err
	}

	if osResource == nil {
		if orcObject.Spec.ManagementPolicy == orcv1alpha1.ManagementPolicyManaged {
			osResource, err = createResource(ctx, orcObject, networkID, networkClient)
			if err != nil {
				return ctrl.Result{}, err
			}
		} else {
			// Programming error
			return ctrl.Result{}, fmt.Errorf("unmanaged object does not exist and not waiting on dependency")
		}
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

	if orcObject.Spec.ManagementPolicy == orcv1alpha1.ManagementPolicyManaged {
		for _, updateFunc := range r.needsUpdate(networkClient, orcObject, osResource) {
			if err := updateFunc(ctx, addStatus); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to update the OpenStack resource: %w", err)
			}
		}
	}

	return ctrl.Result{}, nil
}

func getOSResourceFromObject(ctx context.Context, log logr.Logger, orcObject *orcv1alpha1.Subnet, networkID orcv1alpha1.UUID, networkClient osclients.NetworkClient) (*subnets.Subnet, bool, error) {
	switch {
	case orcObject.Status.ID != nil:
		log.V(4).Info("Fetching existing OpenStack resource", "ID", *orcObject.Status.ID)
		osResource, err := networkClient.GetSubnet(ctx, *orcObject.Status.ID)
		if err != nil {
			if orcerrors.IsNotFound(err) {
				// An OpenStack resource we previously referenced has been deleted unexpectedly. We can't recover from this.
				return nil, false, orcerrors.Terminal(orcv1alpha1.OpenStackConditionReasonUnrecoverableError, "resource has been deleted from OpenStack")
			}
			return nil, false, err
		}
		return osResource, false, nil

	case orcObject.Spec.Import != nil && orcObject.Spec.Import.ID != nil:
		log.V(4).Info("Importing existing OpenStack resource by ID")
		osResource, err := networkClient.GetSubnet(ctx, *orcObject.Spec.Import.ID)
		if err != nil {
			if orcerrors.IsNotFound(err) {
				// We assume that a resource imported by ID must already exist. It's a terminal error if it doesn't.
				return nil, false, orcerrors.Terminal(orcv1alpha1.OpenStackConditionReasonUnrecoverableError, "referenced resource does not exist in OpenStack")
			}
			return nil, false, err
		}
		return osResource, false, nil

	case orcObject.Spec.Import != nil && orcObject.Spec.Import.Filter != nil:
		log.V(4).Info("Importing existing OpenStack resource by filter")
		listOpts := listOptsFromImportFilter(orcObject.Spec.Import.Filter, networkID)
		osResource, err := getResourceFromList(ctx, listOpts, networkClient)
		if err != nil {
			return nil, false, err
		}
		if osResource == nil {
			return nil, true, nil
		}
		return osResource, false, nil

	default:
		log.V(4).Info("Checking for previously created OpenStack resource")
		listOpts := listOptsFromCreation(orcObject)
		osResource, err := getResourceFromList(ctx, listOpts, networkClient)
		if err != nil {
			return nil, false, nil
		}
		return osResource, false, nil
	}
}

func (r *orcSubnetReconciler) reconcileDelete(ctx context.Context, orcObject *orcv1alpha1.Subnet) (_ ctrl.Result, err error) {
	log := ctrl.LoggerFrom(ctx)
	log.V(3).Info("Reconciling OpenStack resource delete")

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

	// We won't delete the resource for an unmanaged object, or if onDelete is detach
	if orcObject.Spec.ManagementPolicy == orcv1alpha1.ManagementPolicyUnmanaged || orcObject.Spec.ManagedOptions.GetOnDelete() == orcv1alpha1.OnDeleteDetach {
		logPolicy := []any{"managementPolicy", orcObject.Spec.ManagementPolicy}
		if orcObject.Spec.ManagementPolicy == orcv1alpha1.ManagementPolicyManaged {
			logPolicy = append(logPolicy, "onDelete", orcObject.Spec.ManagedOptions.GetOnDelete())
		}
		log.V(4).Info("Not deleting OpenStack resource due to policy", logPolicy...)
	} else {
		// Delete any RouterInterface first, as this would prevent deletion of the subnet
		routerInterface, err := r.getRouterInterface(ctx, orcObject)
		if err != nil {
			return ctrl.Result{}, err
		}

		if routerInterface != nil {
			// We will be reconciled again when it's gone
			if routerInterface.GetDeletionTimestamp().IsZero() {
				return ctrl.Result{}, r.client.Delete(ctx, routerInterface)
			}
			return ctrl.Result{}, nil
		}

		deleted, requeue, err := r.deleteResource(ctx, log, orcObject, addStatus)
		if err != nil {
			return ctrl.Result{}, err
		}

		if !deleted {
			return ctrl.Result{RequeueAfter: requeue}, nil
		}
		log.V(4).Info("OpenStack resource is deleted")
	}

	deleted = true

	// Clear the finalizer
	applyConfig := orcapplyconfigv1alpha1.Subnet(orcObject.Name, orcObject.Namespace).WithUID(orcObject.UID)
	return ctrl.Result{}, r.client.Patch(ctx, orcObject, applyconfigs.Patch(types.ApplyPatchType, applyConfig), client.ForceOwnership, ssaFieldOwner(SSAFinalizerTxn))
}

func (r *orcSubnetReconciler) deleteResource(ctx context.Context, log logr.Logger, orcObject *orcv1alpha1.Subnet, addStatus func(updateStatusOpt)) (bool, time.Duration, error) {
	networkClient, err := r.getNetworkClient(ctx, orcObject)
	if err != nil {
		return false, 0, err
	}

	if orcObject.Status.ID != nil {
		// This GET is technically redundant because we could just check the
		// result from DELETE, but it's necessary if we want to report
		// status while deleting
		osResource, err := networkClient.GetSubnet(ctx, *orcObject.Status.ID)

		switch {
		case orcerrors.IsNotFound(err):
			// Success!
			return true, 0, nil

		case err != nil:
			return false, 0, err

		default:
			addStatus(withResource(osResource))

			if len(orcObject.GetFinalizers()) > 1 {
				log.V(4).Info("Deferring resource cleanup due to remaining external finalizers")
				return false, 0, nil
			}

			err := networkClient.DeleteSubnet(ctx, *orcObject.Status.ID)
			if err != nil {
				return false, 0, err
			}
			return false, deletePollingPeriod, nil
		}
	}

	// If status.ID is not set we need to check for an orphaned
	// resource. If we don't find one, assume success and continue,
	// otherwise set status.ID and let the controller delete by ID.

	listOpts := listOptsFromCreation(orcObject)
	osResource, err := getResourceFromList(ctx, listOpts, networkClient)
	if err != nil {
		return false, 0, err
	}

	if osResource != nil {
		addStatus(withResource(osResource))
		return false, deletePollingPeriod, r.setStatusID(ctx, orcObject, osResource.ID)
	}

	// Didn't find an orphaned resource. Assume success.
	return true, 0, nil
}

// getResourceName returns the name of the OpenStack resource we should use.
func getResourceName(orcObject *orcv1alpha1.Subnet) orcv1alpha1.OpenStackName {
	if orcObject.Spec.Resource.Name != nil {
		return *orcObject.Spec.Resource.Name
	}
	return orcv1alpha1.OpenStackName(orcObject.Name)
}

func listOptsFromImportFilter(filter *orcv1alpha1.SubnetFilter, networkID orcv1alpha1.UUID) subnets.ListOptsBuilder {
	listOpts := subnets.ListOpts{
		Name:        string(ptr.Deref(filter.Name, "")),
		Description: string(ptr.Deref(filter.Description, "")),
		NetworkID:   string(networkID),
		IPVersion:   int(ptr.Deref(filter.IPVersion, 0)),
		GatewayIP:   string(ptr.Deref(filter.GatewayIP, "")),
		CIDR:        string(ptr.Deref(filter.CIDR, "")),
		Tags:        neutrontags.Join(filter.FilterByNeutronTags.Tags),
		TagsAny:     neutrontags.Join(filter.FilterByNeutronTags.TagsAny),
		NotTags:     neutrontags.Join(filter.FilterByNeutronTags.NotTags),
		NotTagsAny:  neutrontags.Join(filter.FilterByNeutronTags.NotTagsAny),
	}
	if filter.IPv6 != nil {
		listOpts.IPv6AddressMode = string(ptr.Deref(filter.IPv6.AddressMode, ""))
		listOpts.IPv6RAMode = string(ptr.Deref(filter.IPv6.RAMode, ""))
	}

	return &listOpts
}

// listOptsFromCreation returns a listOpts which will return the OpenStack
// resource which would have been created from the current spec and hopefully no
// other. Its purpose is to automatically adopt a resource that we created but
// failed to write to status.id.
func listOptsFromCreation(osResource *orcv1alpha1.Subnet) subnets.ListOptsBuilder {
	return subnets.ListOpts{Name: string(getResourceName(osResource))}
}

func getResourceFromList(ctx context.Context, listOpts subnets.ListOptsBuilder, networkClient osclients.NetworkClient) (*subnets.Subnet, error) {
	osResources, err := networkClient.ListSubnet(ctx, listOpts)
	if err != nil {
		return nil, err
	}

	if len(osResources) == 1 {
		return &osResources[0], nil
	}

	// No resource found
	if len(osResources) == 0 {
		return nil, nil
	}

	// Multiple resources found
	return nil, orcerrors.Terminal(orcv1alpha1.OpenStackConditionReasonInvalidConfiguration, fmt.Sprintf("Expected to find exactly one OpenStack resource to import. Found %d", len(osResources)))
}

// createResource creates an OpenStack resource from an ORC object
func createResource(ctx context.Context, orcObject *orcv1alpha1.Subnet, networkID orcv1alpha1.UUID, networkClient osclients.NetworkClient) (*subnets.Subnet, error) {
	if orcObject.Spec.ManagementPolicy == orcv1alpha1.ManagementPolicyUnmanaged {
		// Should have been caught by API validation
		return nil, orcerrors.Terminal(orcv1alpha1.OpenStackConditionReasonInvalidConfiguration, "Not creating unmanaged resource")
	}

	log := ctrl.LoggerFrom(ctx)
	log.V(3).Info("Creating OpenStack resource")

	resource := orcObject.Spec.Resource

	if resource == nil {
		// Should have been caught by API validation
		return nil, orcerrors.Terminal(orcv1alpha1.OpenStackConditionReasonInvalidConfiguration, "Creation requested, but spec.resource is not set")
	}

	createOpts := subnets.CreateOpts{
		NetworkID:         string(networkID),
		CIDR:              string(resource.CIDR),
		Name:              string(getResourceName(orcObject)),
		Description:       string(ptr.Deref(resource.Description, "")),
		IPVersion:         gophercloud.IPVersion(resource.IPVersion),
		EnableDHCP:        resource.EnableDHCP,
		DNSPublishFixedIP: resource.DNSPublishFixedIP,
	}

	if len(resource.AllocationPools) > 0 {
		createOpts.AllocationPools = make([]subnets.AllocationPool, len(resource.AllocationPools))
		for i := range resource.AllocationPools {
			createOpts.AllocationPools[i].Start = string(resource.AllocationPools[i].Start)
			createOpts.AllocationPools[i].End = string(resource.AllocationPools[i].End)
		}
	}

	if resource.Gateway != nil {
		switch resource.Gateway.Type {
		case orcv1alpha1.SubnetGatewayTypeAutomatic:
			// Nothing to do
		case orcv1alpha1.SubnetGatewayTypeNone:
			createOpts.GatewayIP = ptr.To("")
		case orcv1alpha1.SubnetGatewayTypeIP:
			fallthrough
		default:
			createOpts.GatewayIP = (*string)(resource.Gateway.IP)
		}
	}

	if len(resource.DNSNameservers) > 0 {
		createOpts.DNSNameservers = make([]string, len(resource.DNSNameservers))
		for i := range resource.DNSNameservers {
			createOpts.DNSNameservers[i] = string(resource.DNSNameservers[i])
		}
	}

	if len(resource.HostRoutes) > 0 {
		createOpts.HostRoutes = make([]subnets.HostRoute, len(resource.HostRoutes))
		for i := range resource.HostRoutes {
			createOpts.HostRoutes[i].DestinationCIDR = string(resource.HostRoutes[i].Destination)
			createOpts.HostRoutes[i].NextHop = string(resource.HostRoutes[i].NextHop)
		}
	}

	if resource.IPv6 != nil {
		createOpts.IPv6AddressMode = string(ptr.Deref(resource.IPv6.AddressMode, ""))
		createOpts.IPv6RAMode = string(ptr.Deref(resource.IPv6.RAMode, ""))
	}

	osResource, err := networkClient.CreateSubnet(ctx, &createOpts)

	// We should require the spec to be updated before retrying a create which returned a conflict
	if orcerrors.IsConflict(err) {
		err = orcerrors.Terminal(orcv1alpha1.OpenStackConditionReasonInvalidConfiguration, "invalid configuration creating resource: "+err.Error(), err)
	}

	return osResource, err
}

func getRouterInterfaceName(orcObject *orcv1alpha1.Subnet) string {
	return orcObject.Name + "-subnet"
}

func routerInterfaceMatchesSpec(routerInterface *orcv1alpha1.RouterInterface, objectName string, resource *orcv1alpha1.SubnetResourceSpec) bool {
	// No routerRef -> there should be no routerInterface
	if resource.RouterRef == nil {
		return routerInterface == nil
	}

	// The router interface should:
	// * Exist
	// * Be of Subnet type
	// * Reference this subnet
	// * Reference the router in our spec

	if routerInterface == nil {
		return false
	}

	if routerInterface.Spec.Type != orcv1alpha1.RouterInterfaceTypeSubnet {
		return false
	}

	if string(ptr.Deref(routerInterface.Spec.SubnetRef, "")) != objectName {
		return false
	}

	return routerInterface.Spec.RouterRef == *resource.RouterRef
}

// getRouterInterface returns the router interface for this subnet, identified by its name
// returns nil for routerinterface without returning an error if the routerinterface does not exist
func (r *orcSubnetReconciler) getRouterInterface(ctx context.Context, orcObject *orcv1alpha1.Subnet) (*orcv1alpha1.RouterInterface, error) {
	routerInterface := &orcv1alpha1.RouterInterface{}
	err := r.client.Get(ctx, types.NamespacedName{Name: getRouterInterfaceName(orcObject), Namespace: orcObject.GetNamespace()}, routerInterface)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("fetching RouterInterface: %w", err)
	}

	return routerInterface, nil
}

// needsUpdate returns a slice of functions that call the OpenStack API to
// align the OpenStack resoruce to its representation in the ORC spec object.
// For network, only the Neutron tags are currently taken into consideration.
func (r *orcSubnetReconciler) needsUpdate(networkClient osclients.NetworkClient, orcObject *orcv1alpha1.Subnet, osResource *subnets.Subnet) (updateFuncs []func(context.Context, func(updateStatusOpt)) error) {
	addUpdateFunc := func(updateFunc func(context.Context, func(updateStatusOpt)) error) {
		updateFuncs = append(updateFuncs, updateFunc)
	}

	resource := orcObject.Spec.Resource
	if resource == nil {
		return updateFuncs
	}

	resourceTagSet := set.New[string](osResource.Tags...)
	objectTagSet := set.New[string]()
	for i := range resource.Tags {
		objectTagSet.Insert(string(resource.Tags[i]))
	}
	if !objectTagSet.Equal(resourceTagSet) {
		addUpdateFunc(func(ctx context.Context, addStatus func(updateStatusOpt)) error {
			opts := attributestags.ReplaceAllOpts{Tags: objectTagSet.SortedList()}
			_, err := networkClient.ReplaceAllAttributesTags(ctx, "subnets", osResource.ID, &opts)
			return err
		})
	}

	addUpdateFunc(func(ctx context.Context, addStatus func(updateStatusOpt)) error {
		routerInterface, err := r.getRouterInterface(ctx, orcObject)
		if err != nil {
			return err
		}
		addStatus(withRouterInterface(routerInterface))

		if routerInterfaceMatchesSpec(routerInterface, orcObject.Name, resource) {
			// Nothing to do
			return nil
		}

		// If it doesn't match we should delete any existing interface
		if routerInterface != nil {
			if routerInterface.GetDeletionTimestamp().IsZero() {
				if err := r.client.Delete(ctx, routerInterface); err != nil {
					return fmt.Errorf("deleting RouterInterface %s: %w", client.ObjectKeyFromObject(routerInterface), err)
				}
			}
			return nil
		}

		// Otherwise create it
		routerInterface = &orcv1alpha1.RouterInterface{}
		routerInterface.Name = getRouterInterfaceName(orcObject)
		routerInterface.Namespace = orcObject.Namespace
		routerInterface.OwnerReferences = []metav1.OwnerReference{
			{
				APIVersion:         orcObject.APIVersion,
				Kind:               orcObject.Kind,
				Name:               orcObject.Name,
				UID:                orcObject.UID,
				BlockOwnerDeletion: ptr.To(true),
			},
		}
		routerInterface.Spec = orcv1alpha1.RouterInterfaceSpec{
			Type:      orcv1alpha1.RouterInterfaceTypeSubnet,
			RouterRef: *resource.RouterRef,
			SubnetRef: ptr.To(orcv1alpha1.ORCNameRef(orcObject.Name)),
		}

		if err := r.client.Create(ctx, routerInterface); err != nil {
			return fmt.Errorf("creating RouterInterface %s: %w", client.ObjectKeyFromObject(orcObject), err)
		}

		return nil
	})

	return updateFuncs
}
