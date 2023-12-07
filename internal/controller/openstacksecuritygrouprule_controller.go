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

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/rules"
	"github.com/gophercloud/gophercloud/pagination"
	openstackv1 "github.com/gophercloud/openstack-resource-controller/api/v1alpha1"
	"github.com/gophercloud/openstack-resource-controller/pkg/apply"
	"github.com/gophercloud/openstack-resource-controller/pkg/cloud"
	"github.com/gophercloud/openstack-resource-controller/pkg/conditions"
	"github.com/gophercloud/openstack-resource-controller/pkg/labels"
)

const (
	OpenStackSecurityGroupRuleFinalizer = "openstacksecuritygrouprule.k-orc.cloud"
)

// OpenStackSecurityGroupRuleReconciler reconciles a OpenStackSecurityGroupRule object
type OpenStackSecurityGroupRuleReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=openstacksecuritygrouprules,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=openstacksecuritygrouprules/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=openstacksecuritygrouprules/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *OpenStackSecurityGroupRuleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	logger := log.FromContext(ctx)
	logger = logger.WithValues("OpenStackSecurityGroupRule", req.Name)

	resource := &openstackv1.OpenStackSecurityGroupRule{}
	err := r.Client.Get(ctx, req.NamespacedName, resource)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("resource not found in the API")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if resource.DeletionTimestamp.IsZero() {
		finalizerUpdated := controllerutil.AddFinalizer(resource, OpenStackSecurityGroupRuleFinalizer)

		newLabels := map[string]string{
			openstackv1.OpenStackDependencyLabelCloud(resource.Spec.Cloud):                          "",
			openstackv1.OpenStackDependencyLabelSecurityGroup(resource.Spec.Resource.SecurityGroup): "",
		}

		labelsMerger, labelsUpdated := labels.ReplacePrefixed(openstackv1.OpenStackLabelPrefix, resource.Labels, newLabels)

		if finalizerUpdated || labelsUpdated {
			logger.Info("applying labels and finalizer")
			patch := &openstackv1.OpenStackSecurityGroupRule{}
			patch.TypeMeta = resource.TypeMeta
			patch.Finalizers = resource.GetFinalizers()
			patch.Labels = labelsMerger
			return ctrl.Result{}, apply.Apply(ctx, r.Client, resource, patch, "spec")
		}
	}

	statusPatchResource := &openstackv1.OpenStackSecurityGroupRule{
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
		if updated, condition := conditions.SetErrorCondition(resource, statusPatchResource, "BadRequest", "One of spec.id or spec.resource must be set"); updated {
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
func (r *OpenStackSecurityGroupRuleReconciler) reconcile(ctx context.Context, networkClient *gophercloud.ServiceClient, resource, statusPatchResource *openstackv1.OpenStackSecurityGroupRule) (_ ctrl.Result, reterr error) {
	logger := log.FromContext(ctx)

	var (
		securityGroupRule *rules.SecGroupRule
		err               error
	)
	if openstackID := coalesce(resource.Spec.ID, resource.Status.Resource.ID); openstackID != "" {
		logger = logger.WithValues("OpenStackID", openstackID)

		securityGroupRule, err = rules.Get(networkClient, openstackID).Extract()
		if err != nil {
			return ctrl.Result{}, err
		}
		logger.Info("OpenStack resource found")
	} else {
		var securityGroupID string
		{
			dependency := &openstackv1.OpenStackSecurityGroup{}
			dependencyKey := client.ObjectKey{Namespace: resource.GetNamespace(), Name: resource.Spec.Resource.SecurityGroup}
			err = r.Client.Get(ctx, dependencyKey, dependency)
			if err != nil && !apierrors.IsNotFound(err) {
				return ctrl.Result{}, err
			}

			// Dependency either doesn't exist, or is being deleted, or is not ready
			if err != nil || !dependency.DeletionTimestamp.IsZero() || !conditions.IsReady(dependency) || dependency.Status.Resource.ID == "" {
				logger.Info("waiting for security group")

				if updated, condition := conditions.SetNotReadyConditionWaiting(resource, statusPatchResource, []conditions.Dependency{
					{ObjectKey: dependencyKey, Resource: "security group"},
				}); updated {
					// Emit an event if we're setting the condition for the first time
					conditions.EmitEventForCondition(r.Recorder, resource, corev1.EventTypeNormal, condition)
				}
				return ctrl.Result{}, nil
			}
			securityGroupID = dependency.Status.Resource.ID
		}

		createOpts := rules.CreateOpts{
			Direction:      rules.RuleDirection(resource.Spec.Resource.Direction),
			Description:    resource.Spec.Resource.Description,
			EtherType:      rules.RuleEtherType(resource.Spec.Resource.EtherType),
			SecGroupID:     securityGroupID,
			PortRangeMax:   resource.Spec.Resource.PortRangeMax,
			PortRangeMin:   resource.Spec.Resource.PortRangeMin,
			Protocol:       rules.RuleProtocol(resource.Spec.Resource.Protocol),
			RemoteGroupID:  resource.Spec.Resource.RemoteGroupID,
			RemoteIPPrefix: resource.Spec.Resource.RemoteIPPrefix,
			ProjectID:      resource.Spec.Resource.ProjectID,
		}

		securityGroupRule, err = r.findAdoptee(log.IntoContext(ctx, logger), networkClient, resource, createOpts)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("unable to find adoption candidates: %w", err)
		}
		if securityGroupRule != nil {
			logger = logger.WithValues("OpenStackID", securityGroupRule.ID)
			logger.Info("OpenStack resource adopted")
		} else {
			securityGroupRule, err = rules.Create(networkClient, createOpts).Extract()
			if err != nil {
				return ctrl.Result{}, err
			}
			logger = logger.WithValues("OpenStackID", securityGroupRule.ID)
			logger.Info("OpenStack resource created")
		}
	}

	statusPatchResource.Status.Resource = openstackv1.OpenStackSecurityGroupRuleResourceStatus{
		ID:              securityGroupRule.ID,
		Direction:       securityGroupRule.Direction,
		Description:     securityGroupRule.Description,
		EtherType:       securityGroupRule.EtherType,
		SecurityGroupID: securityGroupRule.SecGroupID,
		PortRangeMin:    securityGroupRule.PortRangeMin,
		PortRangeMax:    securityGroupRule.PortRangeMax,
		Protocol:        securityGroupRule.Protocol,
		RemoteGroupID:   securityGroupRule.RemoteGroupID,
		RemoteIPPrefix:  securityGroupRule.RemoteIPPrefix,
		TenantID:        securityGroupRule.TenantID,
		ProjectID:       securityGroupRule.ProjectID,
	}

	if updated, condition := conditions.SetReadyCondition(resource, statusPatchResource); updated {
		conditions.EmitEventForCondition(r.Recorder, resource, corev1.EventTypeNormal, condition)
	}
	return ctrl.Result{}, nil
}

func (r *OpenStackSecurityGroupRuleReconciler) reconcileDelete(ctx context.Context, networkClient *gophercloud.ServiceClient, resource, statusPatchResource *openstackv1.OpenStackSecurityGroupRule) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if resource.Status.Resource.ID == "" {
		logger.Info("deletion was requested on a resource that hasn't been successfully created or adopted yet.")
	} else {
		logger = logger.WithValues("OpenStackID", resource.Status.Resource.ID)
		if !resource.Spec.Unmanaged {
			if err := rules.Delete(networkClient, resource.Status.Resource.ID).ExtractErr(); err != nil {
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

	if updated := controllerutil.RemoveFinalizer(resource, OpenStackSecurityGroupRuleFinalizer); updated {
		logger.Info("removing finalizer")
		if updated, condition := conditions.SetNotReadyConditionDeleting(resource, statusPatchResource, "Removing finalizer"); updated {
			conditions.EmitEventForCondition(r.Recorder, resource, corev1.EventTypeNormal, condition)
		}
		patch := &openstackv1.OpenStackSecurityGroupRule{}
		patch.TypeMeta = resource.TypeMeta
		patch.Finalizers = resource.GetFinalizers()
		return ctrl.Result{}, apply.Apply(ctx, r.Client, resource, patch, "spec")
	}
	return ctrl.Result{}, nil
}

func (r *OpenStackSecurityGroupRuleReconciler) findAdoptee(ctx context.Context, networkClient *gophercloud.ServiceClient, resource client.Object, createOpts rules.CreateOpts) (*rules.SecGroupRule, error) {
	adoptedIDs := make(map[string]struct{})
	{
		list := &openstackv1.OpenStackSecurityGroupRuleList{}
		if err := r.Client.List(ctx, list,
			client.InNamespace(resource.GetNamespace()),
		); err != nil {
			return nil, fmt.Errorf("listing OpenStackSecurityGroupRules: %w", err)
		}
		for _, port := range list.Items {
			if port.GetName() != resource.GetName() && port.Status.Resource.ID != "" {
				adoptedIDs[port.Status.Resource.ID] = struct{}{}
			}
		}
	}

	listOpts := rules.ListOpts{
		Direction:      string(createOpts.Direction),
		EtherType:      string(createOpts.EtherType),
		Description:    createOpts.Description,
		PortRangeMax:   createOpts.PortRangeMax,
		PortRangeMin:   createOpts.PortRangeMin,
		Protocol:       string(createOpts.Protocol),
		RemoteGroupID:  createOpts.RemoteGroupID,
		RemoteIPPrefix: createOpts.RemoteIPPrefix,
		SecGroupID:     createOpts.SecGroupID,
		ProjectID:      createOpts.ProjectID,
	}
	var candidates []rules.SecGroupRule
	err := rules.List(networkClient, listOpts).EachPage(func(page pagination.Page) (bool, error) {
		items, err := rules.ExtractRules(page)
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
func (r *OpenStackSecurityGroupRuleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&openstackv1.OpenStackSecurityGroupRule{}).
		WithEventFilter(apply.IgnoreManagedFieldsOnly{}).
		Watches(&openstackv1.OpenStackCloud{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o client.Object) []reconcile.Request {
			// Fetch a list of all OpenStackSecurityGroupRules that reference this OpenStackCloud.
			kclient := mgr.GetClient()
			logger := mgr.GetLogger()

			securityGroupRules := &openstackv1.OpenStackSecurityGroupRuleList{}
			if err := kclient.List(ctx, securityGroupRules,
				client.InNamespace(o.GetNamespace()),
				client.HasLabels{openstackv1.OpenStackDependencyLabelCloud(o.GetName())},
			); err != nil {
				logger.Error(err, "unable to list OpenStackSecurityGroupRules")
				return nil
			}

			// Reconcile each OpenStackSecurityGroupRule that is not Ready and that references this OpenStackCloud.
			reqs := make([]reconcile.Request, 0, len(securityGroupRules.Items))
			for _, securityGroupRule := range securityGroupRules.Items {
				if conditions.IsReady(&securityGroupRule) {
					continue
				}
				reqs = append(reqs, reconcile.Request{
					NamespacedName: client.ObjectKey{
						Namespace: securityGroupRule.GetNamespace(),
						Name:      securityGroupRule.GetName(),
					},
				})
				logger.V(5).Info("update of OpenStackCloud triggers reconcile of OpenStackSecurityGroupRule",
					"namespace", o.GetNamespace(),
					"cloud", o.GetName(),
					"security group rule", securityGroupRule.GetName())
			}
			return reqs
		})).
		Watches(&openstackv1.OpenStackSecurityGroup{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o client.Object) []reconcile.Request {
			// Fetch a list of all OpenStackSecurityGroupRules that reference this OpenStackSecurityGroup.
			kclient := mgr.GetClient()
			logger := mgr.GetLogger()

			securityGroupRules := &openstackv1.OpenStackSecurityGroupRuleList{}
			if err := kclient.List(ctx, securityGroupRules,
				client.InNamespace(o.GetNamespace()),
				client.HasLabels{openstackv1.OpenStackDependencyLabelSecurityGroup(o.GetName())},
			); err != nil {
				logger.Error(err, "unable to list OpenStackSecurityGroupRules")
				return nil
			}

			// Reconcile each OpenStackSecurityGroupRule that is not Ready and that references this OpenStackSecurityGroup.
			reqs := make([]reconcile.Request, 0, len(securityGroupRules.Items))
			for _, securityGroupRule := range securityGroupRules.Items {
				if conditions.IsReady(&securityGroupRule) {
					continue
				}
				reqs = append(reqs, reconcile.Request{
					NamespacedName: client.ObjectKey{
						Namespace: securityGroupRule.GetNamespace(),
						Name:      securityGroupRule.GetName(),
					},
				})
				logger.V(5).Info("update of OpenStackSecurityGroup triggers reconcile of OpenStackSecurityGroupRule",
					"namespace", o.GetNamespace(),
					"cloud", o.GetName(),
					"security group rule", securityGroupRule.GetName())
			}
			return reqs
		})).
		Complete(r)
}
