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

package port

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/attributestags"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"k8s.io/utils/set"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/common"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/generic"
	osclients "github.com/k-orc/openstack-resource-controller/internal/osclients"
	"github.com/k-orc/openstack-resource-controller/internal/util/applyconfigs"
	orcerrors "github.com/k-orc/openstack-resource-controller/internal/util/errors"
	"github.com/k-orc/openstack-resource-controller/internal/util/neutrontags"
	orcapplyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/pkg/clients/applyconfiguration/api/v1alpha1"
)

// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=ports,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=ports/status,verbs=get;update;patch

func (r *orcPortReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	orcObject := &orcv1alpha1.Port{}
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

func (r *orcPortReconciler) getNetworkClient(ctx context.Context, orcPort *orcv1alpha1.Port) (osclients.NetworkClient, error) {
	log := ctrl.LoggerFrom(ctx)

	clientScope, err := r.scopeFactory.NewClientScopeFromObject(ctx, r.client, log, orcPort)
	if err != nil {
		return nil, err
	}
	return clientScope.NewNetworkClient()
}

func (r *orcPortReconciler) reconcileNormal(ctx context.Context, orcObject *orcv1alpha1.Port) (_ ctrl.Result, err error) {
	log := ctrl.LoggerFrom(ctx)
	log.V(3).Info("Reconciling resource")

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
	networkID := orcv1alpha1.UUID(*orcNetwork.Status.ID)

	// Wait for all subnets to be available
	subnetsMapping := make(map[orcv1alpha1.OpenStackName]orcv1alpha1.UUID)
	for _, address := range orcObject.Spec.Resource.Addresses {
		subnetName := *address.SubnetRef
		orcSubnet := &orcv1alpha1.Subnet{}
		if err := r.client.Get(ctx, client.ObjectKey{Name: string(subnetName), Namespace: orcObject.Namespace}, orcSubnet); err != nil {
			if apierrors.IsNotFound(err) {
				addStatus(withProgressMessage(fmt.Sprintf("waiting for subnet object %s to be created", subnetName)))
				return ctrl.Result{}, nil
			}
			return ctrl.Result{}, err
		}

		if !orcv1alpha1.IsAvailable(orcSubnet) {
			addStatus(withProgressMessage(fmt.Sprintf("waiting for subnet object %s to be available", subnetName)))
			return ctrl.Result{}, nil
		}

		if orcSubnet.Status.ID == nil {
			return ctrl.Result{}, fmt.Errorf("subnet %s is available but status.ID is not set", orcSubnet.Name)
		}
		subnetsMapping[subnetName] = orcv1alpha1.UUID(*orcSubnet.Status.ID)
	}

	// Wait for all security groups to be available
	securityGroupsMapping := make(map[orcv1alpha1.OpenStackName]orcv1alpha1.UUID)
	for _, securityGroupName := range orcObject.Spec.Resource.SecurityGroupRefs {
		orcSecurityGroup := &orcv1alpha1.SecurityGroup{}
		if err := r.client.Get(ctx, client.ObjectKey{Name: string(securityGroupName), Namespace: orcObject.Namespace}, orcSecurityGroup); err != nil {
			if apierrors.IsNotFound(err) {
				addStatus(withProgressMessage(fmt.Sprintf("waiting for security group object %s to be created", securityGroupName)))
				return ctrl.Result{}, nil
			}
			return ctrl.Result{}, err
		}

		if !orcv1alpha1.IsAvailable(orcSecurityGroup) {
			addStatus(withProgressMessage(fmt.Sprintf("waiting for security group object %s to be available", securityGroupName)))
			return ctrl.Result{}, nil
		}

		if orcSecurityGroup.Status.ID == nil {
			return ctrl.Result{}, fmt.Errorf("security group %s is available but status.ID is not set", orcSecurityGroup.Name)
		}
		securityGroupsMapping[securityGroupName] = orcv1alpha1.UUID(*orcSecurityGroup.Status.ID)
	}

	// Don't add finalizer until parent dependent resources are available to avoid unnecessary reconcile on delete
	if !controllerutil.ContainsFinalizer(orcObject, Finalizer) {
		patch := common.SetFinalizerPatch(orcObject, Finalizer)
		return ctrl.Result{}, r.client.Patch(ctx, orcObject, patch, client.ForceOwnership, ssaFieldOwner(SSAFinalizerTxn))
	}

	networkClient, err := r.getNetworkClient(ctx, orcObject)
	if err != nil {
		return ctrl.Result{}, err
	}

	portActuator := portActuator{
		Port:                  orcObject,
		osClient:              networkClient,
		networkID:             networkID,
		subnetsMapping:        subnetsMapping,
		securityGroupsMapping: securityGroupsMapping,
	}

	waitMsgs, osResource, err := generic.GetOrCreateOSResource(ctx, log, r.client, portActuator)
	if err != nil {
		return ctrl.Result{}, err
	}

	if osResource == nil {
		log.V(3).Info("OpenStack resource does not yet exist")
		addStatus(withProgressMessage(strings.Join(waitMsgs, ", ")))
		return ctrl.Result{RequeueAfter: externalUpdatePollingPeriod}, nil
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
		for _, updateFunc := range needsUpdate(networkClient, orcObject, osResource) {
			if err := updateFunc(ctx); err != nil {
				addStatus(withProgressMessage("Updating the OpenStack resource"))
				return ctrl.Result{}, fmt.Errorf("failed to update the OpenStack resource: %w", err)
			}
		}
	}

	return ctrl.Result{}, nil
}

func (r *orcPortReconciler) reconcileDelete(ctx context.Context, orcObject *orcv1alpha1.Port) (_ ctrl.Result, err error) {
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

	networkClient, err := r.getNetworkClient(ctx, orcObject)
	if err != nil {
		return ctrl.Result{}, err
	}

	portActuator := portActuator{
		Port:     orcObject,
		osClient: networkClient,
	}

	osResource, result, err := generic.DeleteResource(ctx, log, portActuator, func() error {
		deleted = true

		// Clear the finalizer
		applyConfig := orcapplyconfigv1alpha1.Port(orcObject.Name, orcObject.Namespace).WithUID(orcObject.UID)
		return r.client.Patch(ctx, orcObject, applyconfigs.Patch(types.ApplyPatchType, applyConfig), client.ForceOwnership, ssaFieldOwner(SSAFinalizerTxn))
	})
	addStatus(withResource(osResource))
	return result, err
}

// getResourceName returns the name of the OpenStack resource we should use.
func getResourceName(orcObject *orcv1alpha1.Port) orcv1alpha1.OpenStackName {
	if orcObject.Spec.Resource.Name != nil {
		return *orcObject.Spec.Resource.Name
	}
	return orcv1alpha1.OpenStackName(orcObject.Name)
}

func listOptsFromImportFilter(filter *orcv1alpha1.PortFilter, networkID orcv1alpha1.UUID) ports.ListOptsBuilder {
	listOpts := ports.ListOpts{
		Name:        string(ptr.Deref(filter.Name, "")),
		Description: string(ptr.Deref(filter.Description, "")),
		NetworkID:   string(networkID),
		Tags:        neutrontags.Join(filter.FilterByNeutronTags.Tags),
		TagsAny:     neutrontags.Join(filter.FilterByNeutronTags.TagsAny),
		NotTags:     neutrontags.Join(filter.FilterByNeutronTags.NotTags),
		NotTagsAny:  neutrontags.Join(filter.FilterByNeutronTags.NotTagsAny),
	}

	return &listOpts
}

// listOptsFromCreation returns a listOpts which will return the OpenStack
// resource which would have been created from the current spec and hopefully no
// other. Its purpose is to automatically adopt a resource that we created but
// failed to write to status.id.
func listOptsFromCreation(osResource *orcv1alpha1.Port) ports.ListOptsBuilder {
	return ports.ListOpts{Name: string(getResourceName(osResource))}
}

func getResourceFromList(ctx context.Context, listOpts ports.ListOptsBuilder, networkClient osclients.NetworkClient) (*ports.Port, error) {
	osResources, err := networkClient.ListPort(ctx, listOpts)
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

// createResource creates an OpenStack resource for an ORC object.
func createResource(ctx context.Context, orcObject *orcv1alpha1.Port, networkID orcv1alpha1.UUID,
	subnetsMapping map[orcv1alpha1.OpenStackName]orcv1alpha1.UUID,
	securityGroupsMapping map[orcv1alpha1.OpenStackName]orcv1alpha1.UUID,
	networkClient osclients.NetworkClient) (*ports.Port, error) {
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

	createOpts := ports.CreateOpts{
		NetworkID:   string(networkID),
		Name:        string(getResourceName(orcObject)),
		Description: string(ptr.Deref(resource.Description, "")),
	}

	if len(resource.AllowedAddressPairs) > 0 {
		createOpts.AllowedAddressPairs = make([]ports.AddressPair, len(resource.AllowedAddressPairs))
		for i := range resource.AllowedAddressPairs {
			createOpts.AllowedAddressPairs[i].IPAddress = string(*resource.AllowedAddressPairs[i].IP)
			if resource.AllowedAddressPairs[i].MAC != nil {
				createOpts.AllowedAddressPairs[i].MACAddress = string(*resource.AllowedAddressPairs[i].MAC)
			}
		}
	}

	// We explicitly disable creation of IP addresses by passing an empty
	// value whenever the user does not specifies addresses
	if len(resource.Addresses) > 0 {
		fixedIPs := make([]ports.IP, len(resource.Addresses))
		for i := range resource.Addresses {
			subnetName := *resource.Addresses[i].SubnetRef
			if subnetID, ok := subnetsMapping[subnetName]; !ok {
				return nil, fmt.Errorf("missing subnet ID for %s", subnetName)

			} else {
				fixedIPs[i].SubnetID = string(subnetID)
			}

			if resource.Addresses[i].IP != nil {
				fixedIPs[i].IPAddress = string(*resource.Addresses[i].IP)
			}
		}
		createOpts.FixedIPs = fixedIPs
	} else {
		createOpts.FixedIPs = []string{}
	}

	// We explicitly disable default security groups by passing an empty
	// value whenever the user does not specifies security groups
	securityGroups := make([]string, len(resource.SecurityGroupRefs))
	for i := range resource.SecurityGroupRefs {
		securityGroupName := resource.SecurityGroupRefs[i]
		if securityGroupID, ok := securityGroupsMapping[securityGroupName]; !ok {
			return nil, fmt.Errorf("missing security group ID for %s", securityGroupName)

		} else {
			securityGroups[i] = string(securityGroupID)
		}
	}
	createOpts.SecurityGroups = &securityGroups

	osResource, err := networkClient.CreatePort(ctx, &createOpts)
	if err != nil {
		// We should require the spec to be updated before retrying a create which returned a conflict
		if orcerrors.IsConflict(err) {
			err = orcerrors.Terminal(orcv1alpha1.OpenStackConditionReasonInvalidConfiguration, "invalid configuration creating resource: "+err.Error(), err)
		}
		return nil, err
	}

	return osResource, nil
}

// needsUpdate returns a slice of functions that call the OpenStack API to
// align the OpenStack resoruce to its representation in the ORC spec object.
// For port, only the Neutron tags are currently taken into consideration.
func needsUpdate(networkClient osclients.NetworkClient, orcObject *orcv1alpha1.Port, osResource *ports.Port) (updateFuncs []func(context.Context) error) {
	addUpdateFunc := func(updateFunc func(context.Context) error) {
		updateFuncs = append(updateFuncs, updateFunc)
	}
	resourceTagSet := set.New[string](osResource.Tags...)
	objectTagSet := set.New[string]()
	for i := range orcObject.Spec.Resource.Tags {
		objectTagSet.Insert(string(orcObject.Spec.Resource.Tags[i]))
	}
	if !objectTagSet.Equal(resourceTagSet) {
		addUpdateFunc(func(ctx context.Context) error {
			opts := attributestags.ReplaceAllOpts{Tags: objectTagSet.SortedList()}
			_, err := networkClient.ReplaceAllAttributesTags(ctx, "ports", osResource.ID, &opts)
			return err
		})
	}
	return updateFuncs
}
