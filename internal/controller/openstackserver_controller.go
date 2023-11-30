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
	"encoding/json"
	"errors"
	"fmt"
	"time"

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
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	openstackv1 "github.com/gophercloud/openstack-resource-controller/api/v1alpha1"
	"github.com/gophercloud/openstack-resource-controller/pkg/cloud"
)

// OpenStackServerReconciler reconciles a OpenStackServer object
type OpenStackServerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=openstackservers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=openstackservers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=openstackservers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the OpenStackServer object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.0/pkg/reconcile
func (r *OpenStackServerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	logger := log.FromContext(ctx)
	logger = logger.WithValues("OpenStackServer", req.Name)

	openStackResource := &openstackv1.OpenStackServer{}
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

	computeClient, err := cloud.NewServiceClient(log.IntoContext(ctx, logger), r.Client, openStackCloud, "compute")
	if err != nil {
		err = fmt.Errorf("unable to build an OpenStack client: %w", err)
		logger.Info(err.Error())
		return ctrl.Result{}, err
	}

	if !openStackResource.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(log.IntoContext(ctx, logger), computeClient, openStackResource)
	}

	return r.reconcile(log.IntoContext(ctx, logger), computeClient, openStackResource)
}

// reconcile handles creation. No modification is accepted.
// TODO: restrict unhandled modification through a webhook
// TODO: potentially handle (some?) modifications accepted in OpenStack, as in `openstack network set`
func (r *OpenStackServerReconciler) reconcile(ctx context.Context, computeClient *gophercloud.ServiceClient, resource *openstackv1.OpenStackServer) (ctrl.Result, error) {
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

	var openstackResource *servers.Server
	if resource.Spec.ID != "" {
		logger = logger.WithValues("OpenStackID", resource.Spec.ID)

		var err error
		openstackResource, err = servers.Get(computeClient, resource.Spec.ID).Extract()
		if err != nil {
			return ctrl.Result{}, err
		}
		logger.Info("resouce exists in OpenStack")
	} else {
		var flavorID string
		{
			flavor := &openstackv1.OpenStackFlavor{}
			err := r.Client.Get(ctx, types.NamespacedName{
				Namespace: resource.GetNamespace(),
				Name:      resource.Spec.Flavor,
			}, flavor)
			if err != nil {
				if apierrors.IsNotFound(err) {
					return ctrl.Result{}, fmt.Errorf("flavor %q not found", resource.Spec.Flavor)
				}
				return ctrl.Result{}, err
			}
			if flavor.Status.Resource.ID == "" {
				return ctrl.Result{}, fmt.Errorf("flavor %q not found in OpenStack", resource.Spec.Flavor)
			}
			flavorID = flavor.Status.Resource.ID
		}

		var imageID string
		{
			image := &openstackv1.OpenStackImage{}
			err := r.Client.Get(ctx, types.NamespacedName{
				Namespace: resource.GetNamespace(),
				Name:      resource.Spec.Image,
			}, image)
			if err != nil {
				if apierrors.IsNotFound(err) {
					return ctrl.Result{}, fmt.Errorf("image %q not found", resource.Spec.Flavor)
				}
				return ctrl.Result{}, err
			}
			if image.Status.ID == "" {
				return ctrl.Result{}, fmt.Errorf("image %q not found in OpenStack", resource.Spec.Flavor)
			}
			imageID = image.Status.ID
		}

		securityGroupIDs := make([]string, len(resource.Spec.SecurityGroups))
		for i := range resource.Spec.SecurityGroups {
			securityGroup := &openstackv1.OpenStackSecurityGroup{}
			err := r.Client.Get(ctx, types.NamespacedName{
				Namespace: resource.GetNamespace(),
				Name:      resource.Spec.SecurityGroups[i],
			}, securityGroup)
			if err != nil {
				if apierrors.IsNotFound(err) {
					return ctrl.Result{}, fmt.Errorf("security group %q not found", resource.Spec.SecurityGroups[i])
				}
				return ctrl.Result{}, err
			}
			if securityGroup.Status.ID == "" {
				return ctrl.Result{}, fmt.Errorf("security group %q not found in OpenStack", securityGroup.GetName())
			}
			securityGroupIDs[i] = securityGroup.Status.ID
		}

		gophercloudNetworks := make([]servers.Network, len(resource.Spec.Networks))
		for i := range resource.Spec.Networks {
			network := &openstackv1.OpenStackNetwork{}
			err := r.Client.Get(ctx, types.NamespacedName{
				Namespace: resource.GetNamespace(),
				Name:      resource.Spec.Networks[i],
			}, network)
			if err != nil {
				if apierrors.IsNotFound(err) {
					return ctrl.Result{}, fmt.Errorf("network %q not found", resource.Spec.Networks[i])
				}
				return ctrl.Result{}, err
			}
			if network.Status.Resource.ID == "" {
				return ctrl.Result{}, fmt.Errorf("network %q not found in OpenStack", network.GetName())
			}
			gophercloudNetworks[i] = servers.Network{UUID: network.Status.Resource.ID}
		}

		var err error
		openstackResource, err = servers.Create(computeClient, servers.CreateOpts{
			Name:           resource.Spec.Name,
			ImageRef:       imageID,
			FlavorRef:      flavorID,
			Networks:       gophercloudNetworks,
			SecurityGroups: securityGroupIDs,
			UserData:       resource.Spec.UserData,
		}).Extract()
		if err != nil {
			return ctrl.Result{}, err
		}
		resource.Spec.ID = openstackResource.ID
		logger = logger.WithValues("OpenStackID", openstackResource.ID)
		logger.Info("OpenStack resource created")
	}

	jsonImage, err := json.Marshal(openstackResource.Image)
	if err != nil {
		logger.Info("error marshaling image information: " + err.Error())
	}

	jsonFlavor, err := json.Marshal(openstackResource.Flavor)
	if err != nil {
		logger.Info("error marshaling flavor information: " + err.Error())
	}

	jsonAddresses, err := json.Marshal(openstackResource.Addresses)
	if err != nil {
		logger.Info("error marshaling addresses information: " + err.Error())
	}

	jsonMetadata, err := json.Marshal(openstackResource.Metadata)
	if err != nil {
		logger.Info("error marshaling metadata information: " + err.Error())
	}

	jsonSecurityGroups, err := json.Marshal(openstackResource.SecurityGroups)
	if err != nil {
		logger.Info("error marshaling security group information: " + err.Error())
	}

	resource.Status = openstackv1.OpenStackServerStatus{
		ID:               openstackResource.ID,
		TenantID:         openstackResource.TenantID,
		UserID:           openstackResource.UserID,
		Name:             openstackResource.Name,
		UpdatedAt:        openstackResource.Updated.UTC().Format(time.RFC3339),
		CreatedAt:        openstackResource.Created.UTC().Format(time.RFC3339),
		HostID:           openstackResource.HostID,
		Status:           openstackResource.Status,
		Progress:         openstackResource.Progress,
		AccessIPv4:       openstackResource.AccessIPv4,
		AccessIPv6:       openstackResource.AccessIPv6,
		ImageID:          string(jsonImage),
		FlavorID:         string(jsonFlavor),
		Addresses:        string(jsonAddresses),
		Metadata:         string(jsonMetadata),
		Links:            []string{},
		KeyName:          openstackResource.KeyName,
		SecurityGroupIDs: string(jsonSecurityGroups),
		// AttachedVolumeIDs: []string{},
		// Fault:             "",
		// Tags:              []string{},
		// ServerGroupIDs:    []string{},
	}

	return ctrl.Result{}, nil
}

func (r *OpenStackServerReconciler) reconcileDelete(ctx context.Context, computeClient *gophercloud.ServiceClient, resource *openstackv1.OpenStackServer) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if resource.Spec.ID == "" {
		logger.Info("deletion was requested on a resource that hasn't been created yet.")
	} else {
		logger = logger.WithValues("OpenStackID", resource.Spec.ID)
		if resource.Spec.Unmanaged != nil && !*resource.Spec.Unmanaged {
			if err := servers.Delete(computeClient, resource.Spec.ID).ExtractErr(); err != nil {
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
func (r *OpenStackServerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&openstackv1.OpenStackServer{}).
		Complete(r)
}
