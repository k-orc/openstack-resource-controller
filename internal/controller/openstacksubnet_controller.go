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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	openstackv1 "github.com/gophercloud/openstack-resource-controller/api/v1alpha1"
	"github.com/gophercloud/openstack-resource-controller/pkg/cloud"
)

// OpenStackSubnetReconciler reconciles a OpenStackSubnet object
type OpenStackSubnetReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=openstacksubnets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=openstacksubnets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=openstacksubnets/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the OpenStackSubnet object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.0/pkg/reconcile
func (r *OpenStackSubnetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	logger := log.FromContext(ctx)
	logger = logger.WithValues("OpenStackSubnet", req.Name)

	openStackResource := &openstackv1.OpenStackSubnet{}
	err := r.Client.Get(ctx, req.NamespacedName, openStackResource)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("resource not found in the API")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	openStackCloud := &openstackv1.OpenStackCloud{}
	if err := r.Client.Get(ctx, types.NamespacedName{
		Namespace: req.Namespace,
		Name:      openStackResource.Spec.Cloud,
	}, openStackCloud); err != nil {
		if apierrors.IsNotFound(err) {
			err = fmt.Errorf("OpenStackCloud %q not found: %w", openStackResource.Spec.Cloud, err)
			logger.Info(err.Error())
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	// Initialize the patch helper
	resourcePatchHelper, err := patch.NewHelper(openStackResource, r.Client)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Always patch the resource when exiting this function.
	defer func() {
		reterr = kerrors.NewAggregate([]error{
			reterr,
			resourcePatchHelper.Patch(ctx, openStackResource),
		})
	}()

	networkClient, err := cloud.NewServiceClient(log.IntoContext(ctx, logger), r.Client, openStackCloud, "network")
	if err != nil {
		err = fmt.Errorf("unable to build an OpenStack client: %w", err)
		logger.Info(err.Error())
		return ctrl.Result{}, err
	}

	if !openStackResource.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(log.IntoContext(ctx, logger), networkClient, openStackResource)
	}

	return r.reconcile(log.IntoContext(ctx, logger), networkClient, openStackResource)
}

// reconcile handles creation. No modification is accepted.
// TODO: restrict unhandled modification through a webhook
// TODO: potentially handle (some?) modifications accepted in OpenStack, as in `openstack network set`
func (r *OpenStackSubnetReconciler) reconcile(ctx context.Context, networkClient *gophercloud.ServiceClient, resource *openstackv1.OpenStackSubnet) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// If the resource doesn't have our finalizer, add it.
	if controllerutil.AddFinalizer(resource, openstackv1.Finalizer) {
		// Register the finalizer immediately to avoid orphaning OpenStack resources on delete
		return ctrl.Result{}, nil
	}

	// If the resource has an ID set but hasn't been created by us, then
	// it's unmanaged by default.
	if resource.Spec.Unmanaged == nil {
		var unmanaged bool
		if resource.Spec.ID != "" && resource.Status.ID == "" {
			unmanaged = true
		}
		resource.Spec.Unmanaged = &unmanaged
		return ctrl.Result{}, nil
	}

	var openstackResource *subnets.Subnet
	if resource.Spec.ID != "" {
		logger = logger.WithValues("OpenStackID", resource.Spec.ID)

		var err error
		openstackResource, err = subnets.Get(networkClient, resource.Spec.ID).Extract()
		if err != nil {
			return ctrl.Result{}, err
		}
		logger.Info("resouce exists in OpenStack")
	} else {
		var networkID string
		{
			network := &openstackv1.OpenStackNetwork{}
			err := r.Client.Get(ctx, types.NamespacedName{
				Namespace: resource.GetNamespace(),
				Name:      resource.Spec.Network,
			}, network)
			if err != nil {
				if apierrors.IsNotFound(err) {
					return ctrl.Result{}, fmt.Errorf("network %q not found", resource.Spec.Network)
				}
				return ctrl.Result{}, err
			}
			if network.Status.Resource.ID == "" {
				return ctrl.Result{}, fmt.Errorf("network %q not found in OpenStack", network.GetName())
			}
			networkID = network.Status.Resource.ID
		}

		allocationPools := make([]subnets.AllocationPool, len(resource.Spec.AllocationPools))
		for i := range resource.Spec.AllocationPools {
			allocationPools[i] = subnets.AllocationPool{
				Start: resource.Spec.AllocationPools[i].Start,
				End:   resource.Spec.AllocationPools[i].End,
			}
		}

		var ipVersion gophercloud.IPVersion
		switch resource.Spec.IPVersion {
		case "IPv4":
			ipVersion = gophercloud.IPv4
		case "IPv6":
			ipVersion = gophercloud.IPv6
		default:
			return ctrl.Result{}, fmt.Errorf("invalid IP version %q. Valid instances are %q and %q", resource.Spec.IPVersion, "IPv4", "IPv6")
		}

		hostRoutes := make([]subnets.HostRoute, len(resource.Spec.HostRoutes))
		for i := range resource.Spec.HostRoutes {
			hostRoutes[i] = subnets.HostRoute{
				DestinationCIDR: resource.Spec.HostRoutes[i].DestinationCIDR,
				NextHop:         resource.Spec.HostRoutes[i].NextHop,
			}
		}

		var err error
		openstackResource, err = subnets.Create(networkClient, subnets.CreateOpts{
			NetworkID:       networkID,
			CIDR:            resource.Spec.CIDR,
			Name:            resource.Spec.Name,
			Description:     resource.Spec.Description,
			AllocationPools: allocationPools,
			GatewayIP:       resource.Spec.GatewayIP,
			IPVersion:       ipVersion,
			EnableDHCP:      resource.Spec.EnableDHCP,
			DNSNameservers:  resource.Spec.DNSNameservers,
			ServiceTypes:    resource.Spec.ServiceTypes,
			HostRoutes:      hostRoutes,
			IPv6AddressMode: resource.Spec.IPv6AddressMode,
			IPv6RAMode:      resource.Spec.IPv6RAMode,
		}).Extract()
		if err != nil {
			return ctrl.Result{}, err
		}
		resource.Spec.ID = openstackResource.ID
		logger = logger.WithValues("OpenStackID", openstackResource.ID)
		logger.Info("OpenStack resource created")
	}

	allocationPools := make([]openstackv1.OpenStackSubnetAllocationPool, len(openstackResource.AllocationPools))
	for i := range openstackResource.AllocationPools {
		allocationPools[i] = openstackv1.OpenStackSubnetAllocationPool{
			Start: openstackResource.AllocationPools[i].Start,
			End:   openstackResource.AllocationPools[i].End,
		}
	}

	hostRoutes := make([]openstackv1.OpenStackSubnetHostRoute, len(openstackResource.HostRoutes))
	for i := range openstackResource.HostRoutes {
		hostRoutes[i] = openstackv1.OpenStackSubnetHostRoute{
			DestinationCIDR: openstackResource.HostRoutes[i].DestinationCIDR,
			NextHop:         openstackResource.HostRoutes[i].NextHop,
		}
	}

	resource.Status = openstackv1.OpenStackSubnetStatus{
		ID:              openstackResource.ID,
		NetworkID:       openstackResource.NetworkID,
		Name:            openstackResource.Name,
		Description:     openstackResource.Description,
		IPVersion:       openstackResource.IPVersion,
		CIDR:            openstackResource.CIDR,
		GatewayIP:       openstackResource.GatewayIP,
		DNSNameservers:  openstackResource.DNSNameservers,
		ServiceTypes:    openstackResource.ServiceTypes,
		AllocationPools: allocationPools,
		HostRoutes:      hostRoutes,
		EnableDHCP:      openstackResource.EnableDHCP,
		TenantID:        openstackResource.TenantID,
		ProjectID:       openstackResource.ProjectID,
		IPv6AddressMode: openstackResource.IPv6AddressMode,
		IPv6RAMode:      openstackResource.IPv6RAMode,
		SubnetPoolID:    openstackResource.SubnetPoolID,
		Tags:            openstackResource.Tags,
		RevisionNumber:  openstackResource.RevisionNumber,
	}

	return ctrl.Result{}, nil
}

func (r *OpenStackSubnetReconciler) reconcileDelete(ctx context.Context, networkClient *gophercloud.ServiceClient, resource *openstackv1.OpenStackSubnet) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if resource.Spec.ID == "" {
		logger.Info("deletion was requested on a resource that hasn't been created yet.")
	} else {
		logger = logger.WithValues("OpenStackID", resource.Spec.ID)
		if resource.Spec.Unmanaged != nil && !*resource.Spec.Unmanaged {
			if err := subnets.Delete(networkClient, resource.Spec.ID).ExtractErr(); err != nil {
				var gerr gophercloud.ErrDefault404
				if errors.As(err, &gerr) {
					logger.Info("deletion was requested on a resource that can't be found in OpenStack.")
				} else {
					logger.Info("failed to delete resouce in OpenStack; requeuing.")
					return ctrl.Result{}, err
				}
			}
		}
	}

	controllerutil.RemoveFinalizer(resource, openstackv1.Finalizer)
	logger.Info("resouce deleted in OpenStack.")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OpenStackSubnetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&openstackv1.OpenStackSubnet{}).
		Complete(r)
}
