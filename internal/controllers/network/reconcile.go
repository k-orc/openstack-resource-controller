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

package network

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/attributestags"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/dns"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/external"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/mtu"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/portsecurity"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/provider"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks"
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

type networkExt struct {
	networks.Network
	dns.NetworkDNSExt
	external.NetworkExternalExt
	mtu.NetworkMTUExt
	portsecurity.PortSecurityExt
	provider.NetworkProviderExt
}

// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=networks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=networks/status,verbs=get;update;patch

func (r *orcNetworkReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	orcObject := &orcv1alpha1.Network{}
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

func (r *orcNetworkReconciler) getNetworkClient(ctx context.Context, orcNetwork *orcv1alpha1.Network) (osclients.NetworkClient, error) {
	log := ctrl.LoggerFrom(ctx)

	clientScope, err := r.scopeFactory.NewClientScopeFromObject(ctx, r.client, log, orcNetwork)
	if err != nil {
		return nil, err
	}
	return clientScope.NewNetworkClient()
}

func (r *orcNetworkReconciler) reconcileNormal(ctx context.Context, orcObject *orcv1alpha1.Network) (_ ctrl.Result, err error) {
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

	if !controllerutil.ContainsFinalizer(orcObject, Finalizer) {
		patch := common.SetFinalizerPatch(orcObject, Finalizer)
		return ctrl.Result{}, r.client.Patch(ctx, orcObject, patch, client.ForceOwnership, ssaFieldOwner(SSAFinalizerTxn))
	}

	networkClient, err := r.getNetworkClient(ctx, orcObject)
	if err != nil {
		return ctrl.Result{}, err
	}

	networkActuator := networkActuator{
		Network:  orcObject,
		osClient: networkClient,
	}

	waitMsgs, osResource, err := generic.GetOrCreateOSResource(ctx, log, r.client, networkActuator)
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

func (r *orcNetworkReconciler) reconcileDelete(ctx context.Context, orcObject *orcv1alpha1.Network) (_ ctrl.Result, err error) {
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

	networkActuator := networkActuator{
		Network:  orcObject,
		osClient: networkClient,
	}

	osResource, result, err := generic.DeleteResource(ctx, log, networkActuator, func() error {
		deleted = true

		// Clear the finalizer
		applyConfig := orcapplyconfigv1alpha1.Network(orcObject.Name, orcObject.Namespace).WithUID(orcObject.UID)
		return r.client.Patch(ctx, orcObject, applyconfigs.Patch(types.ApplyPatchType, applyConfig), client.ForceOwnership, ssaFieldOwner(SSAFinalizerTxn))
	})
	addStatus(withResource(osResource))
	return result, err
}

// getResourceName returns the name of the OpenStack resource we should use.
func getResourceName(orcObject *orcv1alpha1.Network) orcv1alpha1.OpenStackName {
	if orcObject.Spec.Resource.Name != nil {
		return *orcObject.Spec.Resource.Name
	}
	return orcv1alpha1.OpenStackName(orcObject.Name)
}

func listOptsFromImportFilter(filter *orcv1alpha1.NetworkFilter) networks.ListOptsBuilder {
	listOpts := networks.ListOpts{
		Name:        string(ptr.Deref(filter.Name, "")),
		Description: string(ptr.Deref(filter.Description, "")),
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
func listOptsFromCreation(osResource *orcv1alpha1.Network) networks.ListOptsBuilder {
	return networks.ListOpts{Name: string(getResourceName(osResource))}
}

func getResourceFromList(ctx context.Context, listOpts networks.ListOptsBuilder, networkClient osclients.NetworkClient) (*networkExt, error) {
	pages, err := networkClient.ListNetwork(listOpts).AllPages(ctx)
	if err != nil {
		return nil, err
	}

	var osResources []networkExt
	err = networks.ExtractNetworksInto(pages, &osResources)
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
func createResource(ctx context.Context, orcObject *orcv1alpha1.Network, networkClient osclients.NetworkClient) (*networkExt, error) {
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

	var createOpts networks.CreateOptsBuilder
	{
		createOptsBase := networks.CreateOpts{
			Name:         string(getResourceName(orcObject)),
			Description:  string(*resource.Description),
			AdminStateUp: resource.AdminStateUp,
			Shared:       resource.Shared,
		}

		if len(resource.AvailabilityZoneHints) > 0 {
			createOptsBase.AvailabilityZoneHints = make([]string, len(resource.AvailabilityZoneHints))
			for i := range resource.AvailabilityZoneHints {
				createOptsBase.AvailabilityZoneHints[i] = string(resource.AvailabilityZoneHints[i])
			}
		}
		createOpts = createOptsBase
	}

	if resource.DNSDomain != nil {
		createOpts = &dns.NetworkCreateOptsExt{
			CreateOptsBuilder: createOpts,
			DNSDomain:         string(*resource.DNSDomain),
		}
	}

	if resource.MTU != nil {
		createOpts = &mtu.CreateOptsExt{
			CreateOptsBuilder: createOpts,
			MTU:               int(*resource.MTU),
		}
	}

	if resource.PortSecurityEnabled != nil {
		createOpts = &portsecurity.NetworkCreateOptsExt{
			CreateOptsBuilder:   createOpts,
			PortSecurityEnabled: resource.PortSecurityEnabled,
		}
	}

	if resource.External != nil {
		createOpts = &external.CreateOptsExt{
			CreateOptsBuilder: createOpts,
			External:          resource.External,
		}
	}

	osResource := &networkExt{}
	createResult := networkClient.CreateNetwork(ctx, createOpts)
	if err := createResult.ExtractInto(osResource); err != nil {
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
// For network, only the Neutron tags are currently taken into consideration.
func needsUpdate(networkClient osclients.NetworkClient, orcObject *orcv1alpha1.Network, osResource *networkExt) (updateFuncs []func(context.Context) error) {
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
			_, err := networkClient.ReplaceAllAttributesTags(ctx, "networks", osResource.ID, &opts)
			return err
		})
	}
	return updateFuncs
}
