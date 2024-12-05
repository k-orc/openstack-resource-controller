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

package securitygroup

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/attributestags"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/groups"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/rules"
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

// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=securitygroups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=securitygroups/status,verbs=get;update;patch

func (r *orcSecurityGroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	orcObject := &orcv1alpha1.SecurityGroup{}
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

func (r *orcSecurityGroupReconciler) getNetworkClient(ctx context.Context, orcSecurityGroup *orcv1alpha1.SecurityGroup) (osclients.NetworkClient, error) {
	log := ctrl.LoggerFrom(ctx)

	clientScope, err := r.scopeFactory.NewClientScopeFromObject(ctx, r.client, log, orcSecurityGroup)
	if err != nil {
		return nil, err
	}
	return clientScope.NewNetworkClient()
}

func (r *orcSecurityGroupReconciler) reconcileNormal(ctx context.Context, orcObject *orcv1alpha1.SecurityGroup) (_ ctrl.Result, err error) {
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
			osResource, err = createResource(ctx, orcObject, networkClient)
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
		for _, updateFunc := range needsUpdate(networkClient, orcObject, osResource) {
			if err := updateFunc(ctx); err != nil {
				addStatus(withProgressMessage("Updating the OpenStack resource"))
				return ctrl.Result{}, fmt.Errorf("failed to update the OpenStack resource: %w", err)
			}
		}
	}

	return ctrl.Result{}, nil
}

func getOSResourceFromObject(ctx context.Context, log logr.Logger, orcObject *orcv1alpha1.SecurityGroup, networkClient osclients.NetworkClient) (*groups.SecGroup, bool, error) {
	switch {
	case orcObject.Status.ID != nil:
		log.V(4).Info("Fetching existing OpenStack resource", "ID", *orcObject.Status.ID)
		osResource, err := networkClient.GetSecGroup(ctx, *orcObject.Status.ID)
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
		osResource, err := networkClient.GetSecGroup(ctx, *orcObject.Spec.Import.ID)
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

func (r *orcSecurityGroupReconciler) reconcileDelete(ctx context.Context, orcObject *orcv1alpha1.SecurityGroup) (_ ctrl.Result, err error) {
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
	applyConfig := orcapplyconfigv1alpha1.SecurityGroup(orcObject.Name, orcObject.Namespace).WithUID(orcObject.UID)
	return ctrl.Result{}, r.client.Patch(ctx, orcObject, applyconfigs.Patch(types.ApplyPatchType, applyConfig), client.ForceOwnership, ssaFieldOwner(SSAFinalizerTxn))
}

func (r *orcSecurityGroupReconciler) deleteResource(ctx context.Context, log logr.Logger, orcObject *orcv1alpha1.SecurityGroup, addStatus func(updateStatusOpt)) (bool, time.Duration, error) {
	networkClient, err := r.getNetworkClient(ctx, orcObject)
	if err != nil {
		return false, 0, err
	}

	if orcObject.Status.ID != nil {
		// This GET is technically redundant because we could just check the
		// result from DELETE, but it's necessary if we want to report
		// status while deleting
		osResource, err := networkClient.GetSecGroup(ctx, *orcObject.Status.ID)

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

			err := networkClient.DeleteSecGroup(ctx, *orcObject.Status.ID)
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
func getResourceName(orcObject *orcv1alpha1.SecurityGroup) orcv1alpha1.OpenStackName {
	if orcObject.Spec.Resource.Name != nil {
		return *orcObject.Spec.Resource.Name
	}
	return orcv1alpha1.OpenStackName(orcObject.Name)
}

func listOptsFromImportFilter(filter *orcv1alpha1.SecurityGroupFilter) groups.ListOpts {
	listOpts := groups.ListOpts{
		Name:        string(ptr.Deref(filter.Name, "")),
		Description: string(ptr.Deref(filter.Description, "")),
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
func listOptsFromCreation(osResource *orcv1alpha1.SecurityGroup) groups.ListOpts {
	return groups.ListOpts{Name: string(getResourceName(osResource))}
}

func getResourceFromList(ctx context.Context, listOpts groups.ListOpts, networkClient osclients.NetworkClient) (*groups.SecGroup, error) {
	osResources, err := networkClient.ListSecGroup(ctx, listOpts)
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
func createResource(ctx context.Context, orcObject *orcv1alpha1.SecurityGroup, networkClient osclients.NetworkClient) (*groups.SecGroup, error) {
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

	createOpts := groups.CreateOpts{
		Name:        string(getResourceName(orcObject)),
		Description: string(ptr.Deref(resource.Description, "")),
		Stateful:    resource.Stateful,
	}

	// FIXME(mandre) The security group inherits the default security group
	// rules. This could be a problem when we implement `update` if ORC
	// does not takes these rules into account.
	osResource, err := networkClient.CreateSecGroup(ctx, &createOpts)
	if err != nil {
		// We should require the spec to be updated before retrying a create which returned a conflict
		if orcerrors.IsConflict(err) {
			err = orcerrors.Terminal(orcv1alpha1.OpenStackConditionReasonInvalidConfiguration, "invalid configuration creating resource: "+err.Error(), err)
		}
		return nil, err
	}

	// TODO(mandre) bulk create security group rules.
	// Need to implement it in gophercloud first
	for _, rule := range resource.Rules {
		createOpts := rules.CreateOpts{
			SecGroupID:     osResource.ID,
			Description:    string(ptr.Deref(rule.Description, "")),
			Direction:      rules.RuleDirection(ptr.Deref(rule.Direction, "")),
			RemoteGroupID:  string(ptr.Deref(rule.RemoteGroupID, "")),
			RemoteIPPrefix: string(ptr.Deref(rule.RemoteIPPrefix, "")),
			Protocol:       rules.RuleProtocol(ptr.Deref(rule.Protocol, "")),
			EtherType:      rules.RuleEtherType(ptr.Deref(rule.Ethertype, "")),
			PortRangeMin:   int(*rule.PortRangeMin),
			PortRangeMax:   int(*rule.PortRangeMax),
		}

		_, err := networkClient.CreateSecGroupRule(ctx, &createOpts)
		if err != nil {
			// We should require the spec to be updated before retrying a create which returned a conflict
			if orcerrors.IsConflict(err) {
				err = orcerrors.Terminal(orcv1alpha1.OpenStackConditionReasonInvalidConfiguration, "invalid configuration creating resource: "+err.Error(), err)
			}
			return nil, err
		}
	}

	return osResource, nil
}

// needsUpdate returns a slice of functions that call the OpenStack API to
// align the OpenStack resource to its representation in the ORC spec object.
// For network, only the Neutron tags are currently taken into consideration.
func needsUpdate(networkClient osclients.NetworkClient, orcObject *orcv1alpha1.SecurityGroup, osResource *groups.SecGroup) (updateFuncs []func(context.Context) error) {
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
			_, err := networkClient.ReplaceAllAttributesTags(ctx, "security-groups", osResource.ID, &opts)
			return err
		})
	}
	return updateFuncs
}
