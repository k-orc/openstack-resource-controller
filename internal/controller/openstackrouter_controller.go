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
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/routers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/pagination"
	openstackv1 "github.com/gophercloud/openstack-resource-controller/api/v1alpha1"
	"github.com/gophercloud/openstack-resource-controller/pkg/cloud"
)

// OpenStackRouterReconciler reconciles a OpenStackRouter object
type OpenStackRouterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=openstackrouters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=openstackrouters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=openstackrouters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the OpenStackRouter object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *OpenStackRouterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	logger := log.FromContext(ctx)
	logger = logger.WithValues("OpenStackRouter", req.Name)

	openStackRouter := &openstackv1.OpenStackRouter{}
	err := r.Client.Get(ctx, req.NamespacedName, openStackRouter)
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
		Name:      openStackRouter.Spec.Cloud,
	}, openStackCloud); err != nil {
		if apierrors.IsNotFound(err) {
			err = fmt.Errorf("OpenStackCloud %q not found: %w", openStackRouter.Spec.Cloud, err)
			logger.Info(err.Error())
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	// Initialize the patch helper
	resourcePatchHelper, err := patch.NewHelper(openStackRouter, r.Client)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Always patch the resource when exiting this function.
	defer func() {
		reterr = kerrors.NewAggregate([]error{
			reterr,
			resourcePatchHelper.Patch(ctx, openStackRouter),
		})
	}()

	networkClient, err := cloud.NewServiceClient(log.IntoContext(ctx, logger), r.Client, openStackCloud, "network")
	if err != nil {
		err = fmt.Errorf("unable to build an OpenStack client: %w", err)
		logger.Info(err.Error())
		return ctrl.Result{}, err
	}

	if !openStackRouter.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(log.IntoContext(ctx, logger), networkClient, openStackRouter)
	}

	return r.reconcile(log.IntoContext(ctx, logger), networkClient, openStackRouter)
}

// reconcile handles creation. No modification is accepted.
// TODO: restrict unhandled modification through a webhook
// TODO: potentially handle (some?) modifications accepted in OpenStack, as in `openstack network set`
func (r *OpenStackRouterReconciler) reconcile(ctx context.Context, networkClient *gophercloud.ServiceClient, resource *openstackv1.OpenStackRouter) (ctrl.Result, error) {
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

	externalNetwork := &openstackv1.OpenStackNetwork{}
	err := r.Client.Get(ctx, types.NamespacedName{
		Namespace: resource.GetNamespace(),
		Name:      resource.Spec.ExternalGatewayInfo.Network,
	}, externalNetwork)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("external network resource not found in the API")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	var openstackResource *routers.Router
	if resource.Spec.ID != "" {
		logger = logger.WithValues("OpenStackID", resource.Spec.ID)

		openstackResource, err = routers.Get(networkClient, resource.Spec.ID).Extract()
		if err != nil {
			return ctrl.Result{}, err
		}
		logger.Info("resouce exists in OpenStack")
	} else {
		var externalFixedIPs []routers.ExternalFixedIP
		for _, externalFixedIP := range resource.Spec.ExternalGatewayInfo.ExternalFixedIPs {
			subnet, err := r.getSubnet(ctx, externalFixedIP.Subnet, resource)
			if err != nil {
				return ctrl.Result{}, err
			}
			fixedIP := routers.ExternalFixedIP{
				IPAddress: externalFixedIP.IPAddress,
				SubnetID:  subnet.Spec.ID,
			}
			externalFixedIPs = append(externalFixedIPs, fixedIP)
		}
		var err error
		openstackResource, err = routers.Create(networkClient, routers.CreateOpts{
			Name:         resource.Spec.Name,
			Description:  resource.Spec.Description,
			AdminStateUp: resource.Spec.AdminStateUp,
			Distributed:  resource.Spec.Distributed,
			TenantID:     resource.Spec.TenantID,
			ProjectID:    resource.Spec.ProjectID,
			GatewayInfo: &routers.GatewayInfo{
				NetworkID:        externalNetwork.Status.Resource.ID,
				EnableSNAT:       resource.Spec.ExternalGatewayInfo.EnableSNAT,
				ExternalFixedIPs: externalFixedIPs,
			},
			AvailabilityZoneHints: resource.Spec.AvailabilityZoneHints,
		}).Extract()
		if err != nil {
			return ctrl.Result{}, err
		}
		resource.Spec.ID = openstackResource.ID
		logger = logger.WithValues("OpenStackID", openstackResource.ID)
		logger.Info("OpenStack resource created")
	}

	var routes []openstackv1.OpenStackRouterRoute
	for _, route := range openstackResource.Routes {
		ospRoute := openstackv1.OpenStackRouterRoute{
			NextHop:         route.NextHop,
			DestinationCIDR: route.DestinationCIDR,
		}
		routes = append(routes, ospRoute)
	}

	var externalFixedIPs []openstackv1.OpenStackRouterExternalFixedIP
	for _, externalFixedIP := range openstackResource.GatewayInfo.ExternalFixedIPs {
		fixedIP := openstackv1.OpenStackRouterExternalFixedIP{
			IPAddress: externalFixedIP.IPAddress,
			Subnet:    externalFixedIP.SubnetID,
		}
		externalFixedIPs = append(externalFixedIPs, fixedIP)
	}

	gatewayInfo := openstackv1.OpenStackRouterStatusExternalGatewayInfo{
		NetworkID:        openstackResource.GatewayInfo.NetworkID,
		EnableSNAT:       openstackResource.GatewayInfo.EnableSNAT,
		ExternalFixedIPs: externalFixedIPs,
	}

	resource.Status = openstackv1.OpenStackRouterStatus{
		ID:                    openstackResource.ID,
		Description:           openstackResource.Description,
		AdminStateUp:          openstackResource.AdminStateUp,
		Distributed:           openstackResource.Distributed,
		TenantID:              openstackResource.TenantID,
		ProjectID:             openstackResource.ProjectID,
		GatewayInfo:           gatewayInfo,
		AvailabilityZoneHints: openstackResource.AvailabilityZoneHints,
		Status:                openstackResource.Status,
		Tags:                  openstackResource.Tags,
		Routes:                routes,
	}

	err = r.addInterfacesInfo(ctx, networkClient, resource, openstackResource)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *OpenStackRouterReconciler) addInterfacesInfo(ctx context.Context, networkClient *gophercloud.ServiceClient, resource *openstackv1.OpenStackRouter, instance *routers.Router) error {
	logger := log.FromContext(ctx)
	var interfacesInfo []openstackv1.OpenStackRouterInterfaceInfo
	// Retrieve the existing interfaces for the status
	if err := ports.List(networkClient, ports.ListOpts{DeviceID: resource.Spec.ID}).EachPage(func(page pagination.Page) (bool, error) {
		portList, err := ports.ExtractPorts(page)
		if err != nil {
			return false, err
		}
		for _, port := range portList {
			for _, fixedIP := range port.FixedIPs {
				ospInterfaceInfo := openstackv1.OpenStackRouterInterfaceInfo{
					ID:       resource.Spec.ID,
					SubnetID: fixedIP.SubnetID,
					PortID:   port.ID,
					TenantID: port.TenantID,
				}
				interfacesInfo = append(interfacesInfo, ospInterfaceInfo)
			}
		}
		return true, nil
	}); err != nil {
		return err
	}
	resource.Status.InterfacesInfo = interfacesInfo
	// If the user has chosen to provide some subnets for the private interface of the router, we add them.
	for _, subnetName := range resource.Spec.Subnets {
		subnet, err := r.getSubnet(ctx, subnetName, resource)
		if err != nil {
			return err
		}
		for _, intInfo := range interfacesInfo {
			if intInfo.SubnetID == subnet.Spec.ID {
				logger.Info("interface already exists")
				return nil
			}
		}
		interfaceInfo, err := routers.AddInterface(networkClient, instance.ID, routers.AddInterfaceOpts{
			SubnetID: subnet.Status.ID,
		}).Extract()
		if err != nil {
			logger.Info("interface already exists")
			_, err = r.getSubnet(ctx, subnetName, resource)
			if err != nil {
				return err
			}
		} else {
			logger.Info("new interface added")
			ospInterfaceInfo := openstackv1.OpenStackRouterInterfaceInfo{
				ID:       interfaceInfo.ID,
				SubnetID: interfaceInfo.SubnetID,
				PortID:   interfaceInfo.PortID,
				TenantID: interfaceInfo.TenantID,
			}
			interfacesInfo = append(interfacesInfo, ospInterfaceInfo)
		}
	}
	resource.Status.InterfacesInfo = interfacesInfo
	return nil
}

func (r *OpenStackRouterReconciler) getSubnet(ctx context.Context, subnetName string, resource *openstackv1.OpenStackRouter) (*openstackv1.OpenStackSubnet, error) {
	logger := log.FromContext(ctx)

	subnet := &openstackv1.OpenStackSubnet{}
	err := r.Client.Get(ctx, types.NamespacedName{
		Namespace: resource.GetNamespace(),
		Name:      subnetName,
	}, subnet)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("subnet resource not found in the API")
			return nil, err
		}
		return nil, err
	}
	return subnet, nil
}

func (r *OpenStackRouterReconciler) reconcileDelete(ctx context.Context, networkClient *gophercloud.ServiceClient, resource *openstackv1.OpenStackRouter) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if resource.Spec.ID == "" {
		logger.Info("deletion was requested on a resource that hasn't been created yet.")
	} else {
		logger = logger.WithValues("OpenStackID", resource.Spec.ID)
		if resource.Spec.Unmanaged != nil && !*resource.Spec.Unmanaged {
			for _, interfaceInfo := range resource.Status.InterfacesInfo {
				interfaceResult := routers.RemoveInterface(networkClient, resource.Spec.ID, routers.RemoveInterfaceOpts{
					SubnetID: interfaceInfo.SubnetID,
				})
				if interfaceResult.Err != nil {
					var gerr gophercloud.ErrDefault404
					if errors.As(interfaceResult.Err, &gerr) {
						logger.Info("deletion was requested on a resource that can't be found in OpenStack.")
					} else {
						logger.Info("failed to delete resouce in OpenStack")
						return ctrl.Result{}, interfaceResult.Err
					}
				}
			}
			if err := routers.Delete(networkClient, resource.Spec.ID).ExtractErr(); err != nil {
				var gerr gophercloud.ErrDefault404
				if errors.As(err, &gerr) {
					logger.Info("deletion was requested on a resource that can't be found in OpenStack.")
				} else {
					logger.Info("failed to delete resouce in OpenStack")
					return ctrl.Result{}, err
				}
			}
		}
	}

	controllerutil.RemoveFinalizer(resource, openstackv1.Finalizer)
	logger.Info("reconcileDelete succeeded.")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OpenStackRouterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&openstackv1.OpenStackRouter{}).
		Complete(r)
}
