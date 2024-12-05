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

package routerinterface

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/routers"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/common"
	osclients "github.com/k-orc/openstack-resource-controller/internal/osclients"
	"github.com/k-orc/openstack-resource-controller/internal/util/applyconfigs"
	orcerrors "github.com/k-orc/openstack-resource-controller/internal/util/errors"
	orcapplyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/pkg/clients/applyconfiguration/api/v1alpha1"
)

const noRequeue time.Duration = 0

// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=routerinterfaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=routerinterfaces/status,verbs=get;update;patch

func (r *orcRouterInterfaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	router := &orcv1alpha1.Router{}
	err := r.client.Get(ctx, req.NamespacedName, router)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if !orcv1alpha1.IsAvailable(router) {
		log.V(4).Info("Not reconciling interfaces for not-Available router")
		return ctrl.Result{}, nil
	}

	routerInterfaces, err := getRouterInterfacesForRouter(ctx, r.client, router)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("fetching router interfaces: %w", err)
	}

	// We don't need to query neutron for ports if there are no interfaces to reconcile
	if len(routerInterfaces) == 0 {
		return ctrl.Result{}, nil
	}

	if router.Status.ID == nil {
		// Programming error
		// No point return an error here because we need the router to be updated before trying again
		log.Info("router is available, but router ID is not set")
		return ctrl.Result{}, nil
	}

	listOpts := ports.ListOpts{
		DeviceOwner: "network:router_interface",
		DeviceID:    *router.Status.ID,
	}

	networkClient, err := r.getNetworkClient(ctx, router)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("getting network client: %w", err)
	}

	routerInterfacePorts, err := networkClient.ListPort(ctx, &listOpts)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("fetching router interface ports: %w", err)
	}

	errs := make([]error, len(routerInterfaces))
	requeues := make([]time.Duration, len(routerInterfaces))
	for i := range routerInterfaces {
		err := &errs[i]
		requeue := &requeues[i]
		routerInterface := &routerInterfaces[i]

		if routerInterface.GetDeletionTimestamp().IsZero() {
			*requeue, *err = r.reconcileNormal(ctx, router, routerInterface, routerInterfacePorts, networkClient)
		} else {
			*requeue, *err = r.reconcileDelete(ctx, router, routerInterface, routerInterfacePorts, networkClient)
		}
	}

	err = errors.Join(errs...)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Find the minimum requeue which is greater than zero (no requeue)
	var minRequeue time.Duration
	for _, requeue := range requeues {
		if requeue == noRequeue {
			continue
		}
		if minRequeue == 0 {
			minRequeue = requeue
		} else if requeue < minRequeue {
			minRequeue = requeue
		}
	}
	return ctrl.Result{RequeueAfter: minRequeue}, nil
}

func (r *orcRouterInterfaceReconciler) getNetworkClient(ctx context.Context, obj orcv1alpha1.CloudCredentialsRefProvider) (osclients.NetworkClient, error) {
	log := ctrl.LoggerFrom(ctx)

	clientScope, err := r.scopeFactory.NewClientScopeFromObject(ctx, r.client, log, obj)
	if err != nil {
		return nil, err
	}
	return clientScope.NewNetworkClient()
}

func (r *orcRouterInterfaceReconciler) reconcileNormal(ctx context.Context, router *orcv1alpha1.Router, routerInterface *orcv1alpha1.RouterInterface, routerInterfacePorts []ports.Port, networkClient osclients.NetworkClient) (_ time.Duration, err error) {
	log := ctrl.LoggerFrom(ctx)
	log.V(3).Info("Reconciling router interface", "name", routerInterface.Name)

	var statusOpts updateStatusOpts

	// Ensure we always update status
	defer func() {
		if err != nil {
			statusOpts.err = err
		}

		err = errors.Join(err, r.updateStatus(ctx, routerInterface, &statusOpts))

		var terminalError *orcerrors.TerminalError
		if errors.As(err, &terminalError) {
			log.Error(err, "not scheduling further reconciles for terminal error")
			err = nil
		}
	}()

	var poll bool
	var createOpts routers.AddInterfaceOptsBuilder
	switch routerInterface.Spec.Type {
	case orcv1alpha1.RouterInterfaceTypeSubnet:
		poll, createOpts, err = r.reconcileNormalSubnet(ctx, routerInterface, routerInterfacePorts, &statusOpts)
	default:
		err = orcerrors.Terminal(orcv1alpha1.OpenStackConditionReasonInvalidConfiguration, fmt.Sprintf("Invalid type %s", routerInterface.Spec.Type))
	}

	if err != nil {
		return noRequeue, err
	}

	if createOpts != nil {
		// Add finalizer immediately before creating a resource
		// Adding the finalizer only when creating a resource means we don't add
		// it until all dependent resources are available, which means we don't
		// have to handle unavailable dependencies in the delete flow
		if !controllerutil.ContainsFinalizer(routerInterface, Finalizer) {
			patch := common.SetFinalizerPatch(routerInterface, Finalizer)
			if err := r.client.Patch(ctx, routerInterface, patch, client.ForceOwnership, ssaFieldOwner(SSAFinalizerTxn)); err != nil {
				return noRequeue, fmt.Errorf("setting finalizer for %s: %w", client.ObjectKeyFromObject(routerInterface), err)
			}
		}

		_, err := networkClient.AddRouterInterface(ctx, *router.Status.ID, createOpts)
		if err != nil {
			return noRequeue, fmt.Errorf("adding router interface: %w", err)
		}

		// We're going to have to poll the interface port anyway, so rather than fetching it here we just schedule the next poll and we'll fetch it next time
		return portStatusPollingPeriod, nil
	}

	if poll {
		return portStatusPollingPeriod, nil
	}

	log.V(3).Info("Router interface is available")
	return noRequeue, nil
}

func findPortBySubnetID(routerInterfacePorts []ports.Port, subnetID string) *ports.Port {
	for i := range routerInterfacePorts {
		routerInterfacePort := &routerInterfacePorts[i]

		for j := range routerInterfacePort.FixedIPs {
			fixedIP := &routerInterfacePort.FixedIPs[j]
			if fixedIP.SubnetID == subnetID {
				return routerInterfacePort
			}
		}
	}
	return nil
}

func (r *orcRouterInterfaceReconciler) reconcileNormalSubnet(ctx context.Context, routerInterface *orcv1alpha1.RouterInterface, routerInterfacePorts []ports.Port, statusOpts *updateStatusOpts) (bool, routers.AddInterfaceOptsBuilder, error) {
	subnet := &orcv1alpha1.Subnet{}
	subnetKey := client.ObjectKey{
		Namespace: routerInterface.Namespace,
		Name:      string(ptr.Deref(routerInterface.Spec.SubnetRef, "")),
	}
	if err := r.client.Get(ctx, subnetKey, subnet); err != nil {
		if apierrors.IsNotFound(err) {
			// We'll be re-executed by the watch on Subnet
			return false, nil, nil
		}

		return false, nil, fmt.Errorf("fetching subnet %s: %w", subnetKey, err)
	}
	statusOpts.subnet = subnet

	if subnet.Status.ID == nil {
		// We don't wait on Available here, but the subnet won't be available until this interface is up
		return false, nil, nil
	}
	subnetID := *subnet.Status.ID

	routerInterfacePort := findPortBySubnetID(routerInterfacePorts, subnetID)
	statusOpts.port = routerInterfacePort

	if routerInterfacePort != nil {
		if routerInterfacePort.Status == portStatusActive {
			// We're done
			return false, nil, nil
		}

		// Port is not active. Requeue so we poll it.
		return true, nil, nil
	}

	return false, &routers.AddInterfaceOpts{SubnetID: subnetID}, nil
}

func (r *orcRouterInterfaceReconciler) reconcileDelete(ctx context.Context, router *orcv1alpha1.Router, routerInterface *orcv1alpha1.RouterInterface, routerInterfacePorts []ports.Port, networkClient osclients.NetworkClient) (_ time.Duration, err error) {
	log := ctrl.LoggerFrom(ctx).WithValues("interface name", routerInterface.Name)

	if !controllerutil.ContainsFinalizer(routerInterface, Finalizer) {
		log.V(4).Info("Not reconciling delete for router interface without finalizer")
		return noRequeue, nil
	}

	if len(routerInterface.GetFinalizers()) > 1 {
		log.V(4).Info("Not reconciling delete for router interface with external finalizers")
		return noRequeue, nil
	}

	log.V(3).Info("Reconciling router interface delete")

	var statusOpts updateStatusOpts

	deleted := false
	defer func() {
		// No point updating status after removing the finalizer
		if !deleted {
			if err != nil {
				statusOpts.err = err
			}
			err = errors.Join(err, r.updateStatus(ctx, routerInterface, &statusOpts))
		}
	}()

	var deleteOpts routers.RemoveInterfaceOptsBuilder
	switch routerInterface.Spec.Type {
	case orcv1alpha1.RouterInterfaceTypeSubnet:
		deleted, deleteOpts, err = r.reconcileDeleteSubnet(ctx, routerInterface, routerInterfacePorts, &statusOpts)
	default:
		err = orcerrors.Terminal(orcv1alpha1.OpenStackConditionReasonInvalidConfiguration, fmt.Sprintf("Invalid type %s", routerInterface.Spec.Type))
	}

	if err != nil {
		return noRequeue, err
	}

	if deleteOpts != nil {
		_, err = networkClient.RemoveRouterInterface(ctx, *router.Status.ID, deleteOpts)
		if err != nil {
			return noRequeue, fmt.Errorf("removing router interface: %w", err)
		}

		return portStatusPollingPeriod, nil
	}

	if !deleted {
		return portStatusPollingPeriod, nil
	}

	// Clear the finalizer
	log.V(3).Info("Router interface deleted")
	applyConfig := orcapplyconfigv1alpha1.RouterInterface(routerInterface.Name, routerInterface.Namespace).WithUID(routerInterface.UID)
	return noRequeue, r.client.Patch(ctx, routerInterface, applyconfigs.Patch(types.ApplyPatchType, applyConfig), client.ForceOwnership, ssaFieldOwner(SSAFinalizerTxn))
}

func (r *orcRouterInterfaceReconciler) reconcileDeleteSubnet(ctx context.Context, routerInterface *orcv1alpha1.RouterInterface, routerInterfacePorts []ports.Port, statusOpts *updateStatusOpts) (bool, routers.RemoveInterfaceOptsBuilder, error) {
	subnet := &orcv1alpha1.Subnet{}
	subnetKey := client.ObjectKey{
		Namespace: routerInterface.Namespace,
		Name:      string(ptr.Deref(routerInterface.Spec.SubnetRef, "")),
	}
	if err := r.client.Get(ctx, subnetKey, subnet); err != nil {
		if apierrors.IsNotFound(err) {
			// This should not happen unless something external messed with our
			// finalizers. We can't continue in this case because we don't know
			// the subnet ID so we can't check if the interface has been
			// removed. We will be automatically reconciled again if the subnet
			// is recreated.
			return false, nil, orcerrors.Terminal(orcv1alpha1.OpenStackConditionReasonUnrecoverableError, "Subnet has been deleted unexpectedly")
		}

		return false, nil, fmt.Errorf("fetching subnet %s: %w", subnetKey, err)
	}
	statusOpts.subnet = subnet
	if subnet.Status.ID == nil {
		return false, nil, orcerrors.Terminal(orcv1alpha1.OpenStackConditionReasonUnrecoverableError, "Subnet ID is not set")
	}
	subnetID := *subnet.Status.ID

	routerInterfacePort := findPortBySubnetID(routerInterfacePorts, subnetID)
	statusOpts.port = routerInterfacePort

	if routerInterfacePort == nil {
		return true, nil, nil
	}

	return false, &routers.RemoveInterfaceOpts{SubnetID: subnetID}, nil
}
