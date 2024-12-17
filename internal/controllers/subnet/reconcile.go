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
	"github.com/k-orc/openstack-resource-controller/internal/controllers/generic"
	osclients "github.com/k-orc/openstack-resource-controller/internal/osclients"
	"github.com/k-orc/openstack-resource-controller/internal/util/applyconfigs"
	orcerrors "github.com/k-orc/openstack-resource-controller/internal/util/errors"
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

	waitEvents, actuator, err := newCreateActuator(ctx, r.client, r.scopeFactory, orcObject)
	if err != nil {
		return ctrl.Result{}, err
	}

	if len(waitEvents) > 0 {
		log.V(3).Info("Waiting on events before initialising actuator")
		addStatus(withProgressMessage(waitEvents[0].Message()))
		return ctrl.Result{RequeueAfter: generic.MaxRequeue(waitEvents)}, nil
	}

	// Don't add finalizer until parent network is available to avoid unnecessary reconcile on delete
	if !controllerutil.ContainsFinalizer(orcObject, Finalizer) {
		patch := common.SetFinalizerPatch(orcObject, Finalizer)
		return ctrl.Result{}, r.client.Patch(ctx, orcObject, patch, client.ForceOwnership, ssaFieldOwner(SSAFinalizerTxn))
	}

	waitEvents, osResource, err := generic.GetOrCreateOSResource(ctx, log, r.client, actuator)
	if err != nil {
		return ctrl.Result{}, err
	}

	if len(waitEvents) > 0 {
		log.V(3).Info("Waiting on events before creation")
		addStatus(withProgressMessage(waitEvents[0].Message()))
		return ctrl.Result{RequeueAfter: generic.MaxRequeue(waitEvents)}, nil
	}

	if osResource == nil {
		// Programming error: if we don't have a resource we should either have an error or be waiting on something
		return ctrl.Result{}, fmt.Errorf("osResource is not set, but no wait events or error")
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
		for _, updateFunc := range r.needsUpdate(actuator.osClient, orcObject, osResource) {
			if err := updateFunc(ctx, addStatus); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to update the OpenStack resource: %w", err)
			}
		}
	}

	return ctrl.Result{}, nil
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

	actuator, err := newDeleteActuator(ctx, r.client, r.scopeFactory, orcObject)
	if err != nil {
		return ctrl.Result{}, nil
	}

	osResource, result, err := generic.DeleteResource(ctx, log, actuator, func() error {
		deleted = true

		// Clear the finalizer
		applyConfig := orcapplyconfigv1alpha1.Subnet(orcObject.Name, orcObject.Namespace).WithUID(orcObject.UID)
		return r.client.Patch(ctx, orcObject, applyconfigs.Patch(types.ApplyPatchType, applyConfig), client.ForceOwnership, ssaFieldOwner(SSAFinalizerTxn))
	})
	addStatus(withResource(osResource))
	return result, err
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
func getRouterInterface(ctx context.Context, k8sClient client.Client, orcObject *orcv1alpha1.Subnet) (*orcv1alpha1.RouterInterface, error) {
	routerInterface := &orcv1alpha1.RouterInterface{}
	err := k8sClient.Get(ctx, types.NamespacedName{Name: getRouterInterfaceName(orcObject), Namespace: orcObject.GetNamespace()}, routerInterface)
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
		routerInterface, err := getRouterInterface(ctx, r.client, orcObject)
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
