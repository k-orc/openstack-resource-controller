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

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/attributestags"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/dns"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/external"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/mtu"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/portsecurity"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/provider"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/utils/set"
	ctrl "sigs.k8s.io/controller-runtime"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/generic"
	osclients "github.com/k-orc/openstack-resource-controller/internal/osclients"
	orcerrors "github.com/k-orc/openstack-resource-controller/internal/util/errors"
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

func (r *orcNetworkReconciler) reconcileNormal(ctx context.Context, orcObject *orcv1alpha1.Network) (_ ctrl.Result, err error) {
	log := ctrl.LoggerFrom(ctx)
	log.V(3).Info("Reconciling resource")

	statusWriter := networkStatusWriter{}
	var osResource *networkExt
	var waitEvents []generic.WaitingOnEvent

	// Ensure we always update status
	defer func() {
		err = errors.Join(err, generic.UpdateStatus(ctx, r, statusWriter, orcObject, osResource, nil, waitEvents, err))

		var terminalError *orcerrors.TerminalError
		if errors.As(err, &terminalError) {
			log.Error(err, "not scheduling further reconciles for terminal error")
			err = nil
		}
	}()

	actuator, err := newActuator(ctx, r, orcObject)
	if err != nil {
		return ctrl.Result{}, err
	}

	waitEvents, osResource, err = generic.GetOrCreateOSResource(ctx, log, r.client, actuator)
	if err != nil {
		return ctrl.Result{}, err
	}

	if len(waitEvents) > 0 {
		log.V(3).Info("Waiting on events before creation")
		return ctrl.Result{RequeueAfter: generic.MaxRequeue(waitEvents)}, nil
	}

	if osResource == nil {
		// Programming error: if we don't have a resource we should either have an error or be waiting on something
		return ctrl.Result{}, fmt.Errorf("oResource is not set, but no wait events or error")
	}

	if orcObject.Status.ID == nil {
		if err := generic.SetStatusID(ctx, actuator, statusWriter, osResource); err != nil {
			return ctrl.Result{}, err
		}
	}

	log = log.WithValues("ID", osResource.ID)
	log.V(4).Info("Got resource")
	ctx = ctrl.LoggerInto(ctx, log)

	if orcObject.Spec.ManagementPolicy == orcv1alpha1.ManagementPolicyManaged {
		for _, updateFunc := range needsUpdate(actuator.osClient, orcObject, osResource) {
			if err := updateFunc(ctx); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to update the OpenStack resource: %w", err)
			}
		}
	}

	return ctrl.Result{}, nil
}

func (r *orcNetworkReconciler) reconcileDelete(ctx context.Context, orcObject *orcv1alpha1.Network) (_ ctrl.Result, err error) {
	log := ctrl.LoggerFrom(ctx)
	log.V(3).Info("Reconciling OpenStack resource delete")

	statusWriter := networkStatusWriter{}
	var osResource *networkExt
	var waitEvents []generic.WaitingOnEvent

	deleted := false
	defer func() {
		// No point updating status after removing the finalizer
		if !deleted {
			err = errors.Join(err, generic.UpdateStatus(ctx, r, statusWriter, orcObject, osResource, nil, waitEvents, err))
		}
	}()

	actuator, err := newActuator(ctx, r, orcObject)
	if err != nil {
		return ctrl.Result{}, err
	}

	deleted, waitEvents, osResource, err = generic.DeleteResource(ctx, log, r.client, actuator)
	return ctrl.Result{RequeueAfter: generic.MaxRequeue(waitEvents)}, err
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
