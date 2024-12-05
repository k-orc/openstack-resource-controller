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

package router

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/attributestags"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/routers"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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

// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=routers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=routers/status,verbs=get;update;patch

func (r *orcRouterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	orcObject := &orcv1alpha1.Router{}
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

func (r *orcRouterReconciler) getNetworkClient(ctx context.Context, orcObject *orcv1alpha1.Router) (osclients.NetworkClient, error) {
	log := ctrl.LoggerFrom(ctx)

	clientScope, err := r.scopeFactory.NewClientScopeFromObject(ctx, r.client, log, orcObject)
	if err != nil {
		return nil, err
	}
	return clientScope.NewNetworkClient()
}

func (r *orcRouterReconciler) reconcileNormal(ctx context.Context, orcObject *orcv1alpha1.Router) (_ ctrl.Result, err error) {
	log := ctrl.LoggerFrom(ctx)
	log.V(3).Info("Reconciling router")

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

	// Don't add finalizer until parent network is available to avoid unnecessary reconcile on delete
	if !controllerutil.ContainsFinalizer(orcObject, Finalizer) {
		patch := common.SetFinalizerPatch(orcObject, Finalizer)
		return ctrl.Result{}, r.client.Patch(ctx, orcObject, patch, client.ForceOwnership, ssaFieldOwner(SSAFinalizerTxn))
	}

	networkClient, err := r.getNetworkClient(ctx, orcObject)
	if err != nil {
		return ctrl.Result{}, err
	}

	osResource, waitingOnExternal, err := getOSResourceFromObject(ctx, log, orcObject, networkClient)
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
			var dependencies []string
			osResource, dependencies, err = r.createResource(ctx, orcObject, networkClient)
			if err != nil {
				return ctrl.Result{}, err
			}

			if len(dependencies) > 0 {
				addStatus(withProgressMessage(strings.Join(dependencies, ", ")))
				return ctrl.Result{}, nil
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
		for _, updateFunc := range needsUpdate(networkClient, orcObject, osResource) {
			if err := updateFunc(ctx); err != nil {
				addStatus(withProgressMessage("Updating the OpenStack resource"))
				return ctrl.Result{}, fmt.Errorf("failed to update the OpenStack resource: %w", err)
			}
		}
	}

	return ctrl.Result{}, nil
}

func getOSResourceFromObject(ctx context.Context, log logr.Logger, orcObject *orcv1alpha1.Router, networkClient osclients.NetworkClient) (*routers.Router, bool, error) {
	switch {
	case orcObject.Status.ID != nil:
		log.V(4).Info("Fetching existing OpenStack resource", "ID", *orcObject.Status.ID)
		osResource, err := networkClient.GetRouter(ctx, *orcObject.Status.ID)
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
		osResource, err := networkClient.GetRouter(ctx, *orcObject.Spec.Import.ID)
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
		listOpts := listOptsFromImportFilter(orcObject.Spec.Import.Filter)
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

func (r *orcRouterReconciler) reconcileDelete(ctx context.Context, orcObject *orcv1alpha1.Router) (_ ctrl.Result, err error) {
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
	applyConfig := orcapplyconfigv1alpha1.Router(orcObject.Name, orcObject.Namespace).WithUID(orcObject.UID)
	return ctrl.Result{}, r.client.Patch(ctx, orcObject, applyconfigs.Patch(types.ApplyPatchType, applyConfig), client.ForceOwnership, ssaFieldOwner(SSAFinalizerTxn))
}

func (r *orcRouterReconciler) deleteResource(ctx context.Context, log logr.Logger, orcObject *orcv1alpha1.Router, addStatus func(updateStatusOpt)) (bool, time.Duration, error) {
	networkClient, err := r.getNetworkClient(ctx, orcObject)
	if err != nil {
		return false, 0, err
	}

	if orcObject.Status.ID != nil {
		// This GET is technically redundant because we could just check the
		// result from DELETE, but it's necessary if we want to report
		// status while deleting
		osResource, err := networkClient.GetRouter(ctx, *orcObject.Status.ID)

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

			err := networkClient.DeleteRouter(ctx, *orcObject.Status.ID)
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
func getResourceName(orcObject *orcv1alpha1.Router) orcv1alpha1.OpenStackName {
	if orcObject.Spec.Resource.Name != nil {
		return *orcObject.Spec.Resource.Name
	}
	return orcv1alpha1.OpenStackName(orcObject.Name)
}

func listOptsFromImportFilter(filter *orcv1alpha1.RouterFilter) routers.ListOpts {
	listOpts := routers.ListOpts{
		Name:        string(filter.Name),
		Description: string(filter.Description),
		ProjectID:   string(filter.ProjectID),
		Tags:        neutrontags.Join(filter.FilterByNeutronTags.Tags),
		TagsAny:     neutrontags.Join(filter.FilterByNeutronTags.TagsAny),
		NotTags:     neutrontags.Join(filter.FilterByNeutronTags.NotTags),
		NotTagsAny:  neutrontags.Join(filter.FilterByNeutronTags.NotTagsAny),
	}
	return listOpts
}

// listOptsFromCreation returns a listOpts which will return the OpenStack
// resource which would have been created from the current spec and hopefully no
// other. Its purpose is to automatically adopt a resource that we created but
// failed to write to status.id.
func listOptsFromCreation(osResource *orcv1alpha1.Router) routers.ListOpts {
	return routers.ListOpts{Name: string(getResourceName(osResource))}
}

func getResourceFromList(ctx context.Context, listOpts routers.ListOpts, networkClient osclients.NetworkClient) (*routers.Router, error) {
	osResources, err := networkClient.ListRouter(ctx, listOpts)
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

func waitingOnCreationMsg(kind string, name orcv1alpha1.KubernetesNameRef) string {
	return fmt.Sprintf("Waiting for %s/%s to exist", kind, name)
}

func waitingOnAvailableMsg(kind string, name orcv1alpha1.KubernetesNameRef) string {
	return fmt.Sprintf("Waiting for %s/%s to be available", kind, name)
}

// createResource creates an OpenStack resource from an ORC object
func (r *orcRouterReconciler) createResource(ctx context.Context, orcObject *orcv1alpha1.Router, networkClient osclients.NetworkClient) (*routers.Router, []string, error) {
	if orcObject.Spec.ManagementPolicy == orcv1alpha1.ManagementPolicyUnmanaged {
		// Should have been caught by API validation
		return nil, nil, orcerrors.Terminal(orcv1alpha1.OpenStackConditionReasonInvalidConfiguration, "Not creating unmanaged resource")
	}

	log := ctrl.LoggerFrom(ctx)
	log.V(3).Info("Creating OpenStack resource")

	resource := orcObject.Spec.Resource

	if resource == nil {
		// Should have been caught by API validation
		return nil, nil, orcerrors.Terminal(orcv1alpha1.OpenStackConditionReasonInvalidConfiguration, "Creation requested, but spec.resource is not set")
	}

	var gatewayInfo *routers.GatewayInfo
	if len(resource.ExternalGateways) > 0 {
		// We only currently support a single external gateway, ensured by API validation
		externalGateway := resource.ExternalGateways[0]

		network := &orcv1alpha1.Network{}
		if err := r.client.Get(ctx, types.NamespacedName{Name: string(externalGateway.NetworkRef), Namespace: orcObject.Namespace}, network); err != nil {
			if apierrors.IsNotFound(err) {
				return nil, []string{waitingOnCreationMsg("network", externalGateway.NetworkRef)}, nil
			}
			return nil, nil, err
		}
		if !orcv1alpha1.IsAvailable(network) {
			return nil, []string{waitingOnAvailableMsg("network", externalGateway.NetworkRef)}, nil
		}

		if network.Status.ID == nil {
			return nil, nil, orcerrors.Terminal(orcv1alpha1.OpenStackConditionReasonUnrecoverableError, "status.id of available external gateway is nil")
		}

		gatewayInfo = &routers.GatewayInfo{
			NetworkID: *network.Status.ID,
		}
	}

	createOpts := routers.CreateOpts{
		Name:         string(ptr.Deref(resource.Name, "")),
		Description:  string(resource.Description),
		AdminStateUp: resource.AdminStateUp,
		Distributed:  resource.Distributed,
		GatewayInfo:  gatewayInfo,
	}

	if len(resource.AvailabilityZoneHints) > 0 {
		createOpts.AvailabilityZoneHints = make([]string, len(resource.AvailabilityZoneHints))
		for i := range resource.AvailabilityZoneHints {
			createOpts.AvailabilityZoneHints[i] = string(resource.AvailabilityZoneHints[i])
		}
	}

	osResource, err := networkClient.CreateRouter(ctx, &createOpts)

	// We should require the spec to be updated before retrying a create which returned a conflict
	if orcerrors.IsConflict(err) {
		err = orcerrors.Terminal(orcv1alpha1.OpenStackConditionReasonInvalidConfiguration, "invalid configuration creating resource: "+err.Error(), err)
	}

	return osResource, nil, err
}

// needsUpdate returns a slice of functions that call the OpenStack API to
// align the OpenStack resoruce to its representation in the ORC spec object.
// For network, only the Neutron tags are currently taken into consideration.
func needsUpdate(networkClient osclients.NetworkClient, orcObject *orcv1alpha1.Router, osResource *routers.Router) (updateFuncs []func(context.Context) error) {
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
			_, err := networkClient.ReplaceAllAttributesTags(ctx, "routers", osResource.ID, &opts)
			return err
		})
	}
	return updateFuncs
}
