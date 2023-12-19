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
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/groups"
	"github.com/gophercloud/gophercloud/pagination"
	openstackv1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/pkg/apply"
	"github.com/k-orc/openstack-resource-controller/pkg/cloud"
	"github.com/k-orc/openstack-resource-controller/pkg/conditions"
	"github.com/k-orc/openstack-resource-controller/pkg/labels"
)

const (
	OpenStackSecurityGroupFinalizer = "openstacksecuritygroup.k-orc.cloud"
)

// OpenStackSecurityGroupReconciler reconciles a OpenStackSecurityGroup object
type OpenStackSecurityGroupReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=openstacksecuritygroups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=openstacksecuritygroups/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=openstacksecuritygroups/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=openstacksecuritygrouprules,verbs=list
//+kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=openstackports,verbs=list
//+kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=openstackservers,verbs=list

func (r *OpenStackSecurityGroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	logger := log.FromContext(ctx)
	logger = logger.WithValues("OpenStackSecurityGroup", req.Name)

	resource := &openstackv1.OpenStackSecurityGroup{}
	err := r.Client.Get(ctx, req.NamespacedName, resource)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("resource not found in the API")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if resource.DeletionTimestamp.IsZero() {
		finalizerUpdated := controllerutil.AddFinalizer(resource, OpenStackSecurityGroupFinalizer)

		newLabels := map[string]string{
			openstackv1.OpenStackDependencyLabelCloud(resource.Spec.Cloud): "",
		}

		labelsMerger, labelsUpdated := labels.ReplacePrefixed(openstackv1.OpenStackLabelPrefix, resource.Labels, newLabels)

		if finalizerUpdated || labelsUpdated {
			logger.Info("applying labels and finalizer")
			patch := &openstackv1.OpenStackSecurityGroup{}
			patch.TypeMeta = resource.TypeMeta
			patch.Finalizers = resource.GetFinalizers()
			patch.Labels = labelsMerger
			return ctrl.Result{}, apply.Apply(ctx, r.Client, resource, patch, "spec")
		}
	}

	statusPatchResource := &openstackv1.OpenStackSecurityGroup{
		Status:   *resource.Status.DeepCopy(),
		TypeMeta: resource.TypeMeta,
	}
	defer func() {
		// If we're returning an error, report it as a TransientError in the Ready condition
		if reterr != nil {
			if updated, condition := conditions.SetNotReadyConditionTransientError(resource, statusPatchResource, reterr.Error()); updated {
				// Emit an event if we're setting the condition for the first time
				conditions.EmitEventForCondition(r.Recorder, resource, corev1.EventTypeWarning, condition)
			}
		}
		if err := apply.ApplyStatus(ctx, r.Client, resource, statusPatchResource); err != nil && !(apierrors.IsNotFound(err) && len(resource.Finalizers) == 0) {
			reterr = errors.Join(reterr, err)
		}

	}()
	if len(resource.Status.Conditions) == 0 {
		conditions.InitialiseRequiredConditions(resource, statusPatchResource)
	}

	if resource.Spec.ID == "" && resource.Spec.Resource == nil {
		if updated, condition := conditions.SetErrorCondition(resource, statusPatchResource, openstackv1.OpenStackErrorReasonInvalidSpec, "One of spec.id or spec.resource must be set"); updated {
			conditions.EmitEventForCondition(r.Recorder, resource, corev1.EventTypeNormal, condition)
		}
		return ctrl.Result{}, nil
	}

	// Get the OpenStackCloud resource
	openStackCloud := &openstackv1.OpenStackCloud{}
	{
		openStackCloudRef := client.ObjectKey{
			Namespace: req.Namespace,
			Name:      resource.Spec.Cloud,
		}
		err := r.Client.Get(ctx, openStackCloudRef, openStackCloud)
		if err != nil && !apierrors.IsNotFound(err) {
			return ctrl.Result{}, fmt.Errorf("fetching OpenStackCloud %s: %w", resource.Spec.Cloud, err)
		}

		// XXX(mbooth): We should check IsReady(openStackCloud) here, but we can't because this breaks us while the cloud is Deleting.
		// We probably need another Condition 'Deleting' so an object can be both Ready and Deleting during the cleanup phase.
		if err != nil {
			conditions.SetNotReadyConditionWaiting(resource, statusPatchResource, []conditions.Dependency{
				{ObjectKey: openStackCloudRef, Resource: "OpenStackCloud"},
			})
			return ctrl.Result{}, nil
		}
	}

	networkClient, err := cloud.NewServiceClient(ctx, r.Client, openStackCloud, "network")
	if err != nil {
		err = fmt.Errorf("unable to build an OpenStack client: %w", err)
		logger.Info(err.Error())
		return ctrl.Result{}, err
	}

	if !resource.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(log.IntoContext(ctx, logger), networkClient, resource, statusPatchResource)
	}

	return r.reconcile(log.IntoContext(ctx, logger), networkClient, resource, statusPatchResource)
}

// reconcile handles creation. No modification is accepted.
// TODO: restrict unhandled modification through a webhook
// TODO: potentially handle (some?) modifications accepted in OpenStack
func (r *OpenStackSecurityGroupReconciler) reconcile(ctx context.Context, networkClient *gophercloud.ServiceClient, resource, statusPatchResource *openstackv1.OpenStackSecurityGroup) (_ ctrl.Result, reterr error) {
	logger := log.FromContext(ctx)

	var (
		securityGroup *groups.SecGroup
		err           error
	)
	if openstackID := coalesce(resource.Spec.ID, resource.Status.Resource.ID); openstackID != "" {
		logger = logger.WithValues("OpenStackID", openstackID)

		securityGroup, err = groups.Get(networkClient, openstackID).Extract()
		if err != nil {
			return ctrl.Result{}, err
		}
		logger.Info("OpenStack resource found")
	} else {
		createOpts := groups.CreateOpts{
			Name:        resource.Spec.Resource.Name,
			Description: resource.Spec.Resource.Description,
		}

		securityGroup, err = r.findAdoptee(log.IntoContext(ctx, logger), networkClient, resource, createOpts)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("unable to find adoption candidates: %w", err)
		}
		if securityGroup != nil {
			logger = logger.WithValues("OpenStackID", securityGroup.ID)
			logger.Info("OpenStack resource adopted")
		} else {
			securityGroup, err = groups.Create(networkClient, createOpts).Extract()
			if err != nil {
				return ctrl.Result{}, err
			}
			logger = logger.WithValues("OpenStackID", securityGroup.ID)
			logger.Info("OpenStack resource created")
		}
	}

	secGroupRuleIDs := make([]string, len(securityGroup.Rules))
	for i := range securityGroup.Rules {
		secGroupRuleIDs[i] = securityGroup.Rules[i].ID
	}

	statusPatchResource.Status.Resource = openstackv1.OpenStackSecurityGroupResourceStatus{
		ID:          securityGroup.ID,
		Name:        securityGroup.Name,
		Description: securityGroup.Description,
		Rules:       secGroupRuleIDs,
		TenantID:    securityGroup.TenantID,
		UpdatedAt:   securityGroup.UpdatedAt.UTC().Format(time.RFC3339),
		CreatedAt:   securityGroup.CreatedAt.UTC().Format(time.RFC3339),
		ProjectID:   securityGroup.ProjectID,
		Tags:        securityGroup.Tags,
	}

	if updated, condition := conditions.SetReadyCondition(resource, statusPatchResource); updated {
		conditions.EmitEventForCondition(r.Recorder, resource, corev1.EventTypeNormal, condition)
	}
	return ctrl.Result{}, nil
}

func (r *OpenStackSecurityGroupReconciler) reconcileDelete(ctx context.Context, networkClient *gophercloud.ServiceClient, resource, statusPatchResource *openstackv1.OpenStackSecurityGroup) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.V(4).Info("Checking for dependant OpenStack resources")
	referencingResources := []string{}
	for _, resourceList := range []client.ObjectList{
		&openstackv1.OpenStackSecurityGroupRuleList{},
		&openstackv1.OpenStackPortList{},
		&openstackv1.OpenStackServerList{},
	} {
		list := &unstructured.UnstructuredList{}
		gvk, err := apiutil.GVKForObject(resourceList, r.Client.Scheme())
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("getting GVK for resource list: %w", err)
		}
		list.SetGroupVersionKind(gvk)
		if err := r.Client.List(ctx, list,
			client.InNamespace(resource.GetNamespace()),
			client.HasLabels{openstackv1.OpenStackDependencyLabelSecurityGroup(resource.GetName())},
			client.Limit(1),
		); err != nil {
			logger.Error(err, "unable to list resources", "type", list.GetKind())
			return ctrl.Result{}, err
		}

		if len(list.Items) > 0 {
			referencingResources = append(referencingResources, list.Items[0].GetKind())
		}
	}

	if len(referencingResources) > 0 {
		logger.Info("OpenStack resources still referencing this security group", "resources", referencingResources)

		message := fmt.Sprintf("Resources of the following types still reference this security group: %s", strings.Join(referencingResources, ", "))
		if updated, condition := conditions.SetNotReadyConditionDeleting(resource, statusPatchResource, message); updated {
			conditions.EmitEventForCondition(r.Recorder, resource, corev1.EventTypeWarning, condition)
		}

		// We don't have (and probably don't want) watches on every resource type, so we just have poll here
		return ctrl.Result{RequeueAfter: time.Second * 10}, nil
	}

	if resource.Status.Resource.ID == "" {
		logger.Info("deletion was requested on a resource that hasn't been successfully created or adopted yet.")
	} else {
		logger = logger.WithValues("OpenStackID", resource.Status.Resource.ID)
		if !resource.Spec.Unmanaged {
			if err := groups.Delete(networkClient, resource.Status.Resource.ID).ExtractErr(); err != nil {
				var gerr gophercloud.ErrDefault404
				if errors.As(err, &gerr) {
					logger.Info("deletion was requested on a resource that can't be found in OpenStack.")
				} else {
					logger.Info("failed to delete resource in OpenStack; requeuing.")
					return ctrl.Result{}, err
				}
			}
		}
	}

	if updated := controllerutil.RemoveFinalizer(resource, OpenStackSecurityGroupFinalizer); updated {
		logger.Info("removing finalizer")
		if updated, condition := conditions.SetNotReadyConditionDeleting(resource, statusPatchResource, "Removing finalizer"); updated {
			conditions.EmitEventForCondition(r.Recorder, resource, corev1.EventTypeNormal, condition)
		}
		patch := &openstackv1.OpenStackSecurityGroup{}
		patch.TypeMeta = resource.TypeMeta
		patch.Finalizers = resource.GetFinalizers()
		return ctrl.Result{}, apply.Apply(ctx, r.Client, resource, patch, "spec")
	}
	return ctrl.Result{}, nil
}

func (r *OpenStackSecurityGroupReconciler) findAdoptee(ctx context.Context, networkClient *gophercloud.ServiceClient, resource client.Object, createOpts groups.CreateOpts) (*groups.SecGroup, error) {
	adoptedIDs := make(map[string]struct{})
	{
		list := &openstackv1.OpenStackSecurityGroupList{}
		if err := r.Client.List(ctx, list,
			client.InNamespace(resource.GetNamespace()),
		); err != nil {
			return nil, fmt.Errorf("listing OpenStackSecurityGroups: %w", err)
		}
		for _, port := range list.Items {
			if port.GetName() != resource.GetName() && port.Status.Resource.ID != "" {
				adoptedIDs[port.Status.Resource.ID] = struct{}{}
			}
		}
	}

	listOpts := groups.ListOpts{
		Name:        createOpts.Name,
		Description: createOpts.Description,
		TenantID:    createOpts.TenantID,
		ProjectID:   createOpts.ProjectID,
	}
	var candidates []groups.SecGroup
	err := groups.List(networkClient, listOpts).EachPage(func(page pagination.Page) (bool, error) {
		items, err := groups.ExtractGroups(page)
		if err != nil {
			return false, fmt.Errorf("extracting resources: %w", err)
		}
		for i := range items {
			if _, ok := adoptedIDs[items[i].ID]; !ok {
				candidates = append(candidates, items[i])
			}
		}

		return true, nil
	})
	if err != nil {
		return nil, err
	}

	switch n := len(candidates); n {
	case 0:
		return nil, nil
	case 1:
		return &candidates[0], nil
	default:
		return nil, fmt.Errorf("found %d possible candidates", n)
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *OpenStackSecurityGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&openstackv1.OpenStackSecurityGroup{}).
		WithEventFilter(apply.IgnoreManagedFieldsOnly{}).
		Watches(&openstackv1.OpenStackCloud{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o client.Object) []reconcile.Request {
			// Fetch a list of all OpenStackSecurityGroups that reference this OpenStackCloud.
			kclient := mgr.GetClient()
			logger := mgr.GetLogger()

			securityGroups := &openstackv1.OpenStackSecurityGroupList{}
			if err := kclient.List(ctx, securityGroups,
				client.InNamespace(o.GetNamespace()),
				client.HasLabels{openstackv1.OpenStackDependencyLabelCloud(o.GetName())},
			); err != nil {
				logger.Error(err, "unable to list OpenStackSecurityGroups")
				return nil
			}

			// Reconcile each OpenStackSecurityGroup that is not Ready and that references this OpenStackCloud.
			reqs := make([]reconcile.Request, 0, len(securityGroups.Items))
			for _, securityGroup := range securityGroups.Items {
				if conditions.IsReady(&securityGroup) {
					continue
				}
				reqs = append(reqs, reconcile.Request{
					NamespacedName: client.ObjectKey{
						Namespace: securityGroup.GetNamespace(),
						Name:      securityGroup.GetName(),
					},
				})
				logger.V(5).Info("update of OpenStackCloud triggers reconcile of OpenStackSecurityGroup",
					"namespace", o.GetNamespace(),
					"cloud", o.GetName(),
					"security group", securityGroup.GetName())
			}
			return reqs
		})).
		Complete(r)
}
