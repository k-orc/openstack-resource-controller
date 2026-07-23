/*
Copyright The ORC Authors.

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

package limit

import (
	"context"
	"errors"
	"iter"

	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/limits"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/progress"
	"github.com/k-orc/openstack-resource-controller/v2/internal/logging"
	"github.com/k-orc/openstack-resource-controller/v2/internal/osclients"
	"github.com/k-orc/openstack-resource-controller/v2/internal/util/dependency"
	orcerrors "github.com/k-orc/openstack-resource-controller/v2/internal/util/errors"
)

var (
	errInvalidDomainRefUpdate  = errors.New("limit cannot be updated with domainRef when projectRef has been used")
	errInvalidProjectRefUpdate = errors.New("limit cannot be updated with projectRef when domainRef has been used")
)

// OpenStack resource types
type (
	osResourceT = limits.Limit

	createResourceActuator = interfaces.CreateResourceActuator[orcObjectPT, orcObjectT, filterT, osResourceT]
	deleteResourceActuator = interfaces.DeleteResourceActuator[orcObjectPT, orcObjectT, osResourceT]
	resourceReconciler     = interfaces.ResourceReconciler[orcObjectPT, osResourceT]
	helperFactory          = interfaces.ResourceHelperFactory[orcObjectPT, orcObjectT, resourceSpecT, filterT, osResourceT]
)

type limitActuator struct {
	osClient  osclients.LimitClient
	k8sClient client.Client
}

var _ createResourceActuator = limitActuator{}
var _ deleteResourceActuator = limitActuator{}

func (limitActuator) GetResourceID(osResource *osResourceT) string {
	return osResource.ID
}

func (actuator limitActuator) GetOSResourceByID(ctx context.Context, id string) (*osResourceT, progress.ReconcileStatus) {
	resource, err := actuator.osClient.GetLimit(ctx, id)
	if err != nil {
		return nil, progress.WrapError(err)
	}
	return resource, nil
}

func (actuator limitActuator) ListOSResourcesForAdoption(ctx context.Context, orcObject orcObjectPT) (iter.Seq2[*osResourceT, error], bool) {
	resourceSpec := orcObject.Spec.Resource
	if resourceSpec == nil {
		return nil, false
	}

	var rs progress.ReconcileStatus

	svc, rs1 := dependency.FetchDependency(
		ctx, actuator.k8sClient, orcObject.Namespace, &resourceSpec.ServiceRef, "Service",
		func(dep *orcv1alpha1.Service) bool {
			return orcv1alpha1.IsAvailable(dep) && dep.Status.ID != nil
		},
	)
	rs.WithReconcileStatus(rs1)

	project, rs1 := dependency.FetchDependency(
		ctx, actuator.k8sClient, orcObject.Namespace, &resourceSpec.ServiceRef, "Project",
		func(dep *orcv1alpha1.Service) bool {
			return orcv1alpha1.IsAvailable(dep) && dep.Status.ID != nil
		},
	)
	rs.WithReconcileStatus(rs1)

	domain, rs1 := dependency.FetchDependency(
		ctx, actuator.k8sClient, orcObject.Namespace, &resourceSpec.ServiceRef, "Domain",
		func(dep *orcv1alpha1.Service) bool {
			return orcv1alpha1.IsAvailable(dep) && dep.Status.ID != nil
		},
	)
	rs.WithReconcileStatus(rs1)

	if needsReschedule, err := rs.NeedsReschedule(); needsReschedule {
		if err != nil {
			ctrl.LoggerFrom(ctx).Info("fetch dependency before listing limit for adoption", "error", err)
		}

		return nil, false
	}

	listOpts := limits.ListOpts{
		ServiceID:    ptr.Deref(svc.Status.ID, ""),
		ProjectID:    ptr.Deref(project.Status.ID, ""),
		DomainID:     ptr.Deref(domain.Status.ID, ""),
		ResourceName: resourceSpec.ResourceName,
	}

	return actuator.osClient.ListLimits(ctx, listOpts), true
}

func (actuator limitActuator) ListOSResourcesForImport(ctx context.Context, obj orcObjectPT, filter filterT) (iter.Seq2[*osResourceT, error], progress.ReconcileStatus) {
	var reconcileStatus progress.ReconcileStatus

	service, rs := dependency.FetchDependency[*orcv1alpha1.Service](
		ctx, actuator.k8sClient, obj.Namespace,
		filter.ServiceRef, "Service",
		orcv1alpha1.IsAvailable,
	)
	reconcileStatus = reconcileStatus.WithReconcileStatus(rs)

	project, rs := dependency.FetchDependency[*orcv1alpha1.Project](
		ctx, actuator.k8sClient, obj.Namespace,
		filter.ProjectRef, "Project",
		orcv1alpha1.IsAvailable,
	)
	reconcileStatus = reconcileStatus.WithReconcileStatus(rs)

	domain, rs := dependency.FetchDependency[*orcv1alpha1.Domain](
		ctx, actuator.k8sClient, obj.Namespace,
		filter.DomainRef, "Domain",
		orcv1alpha1.IsAvailable,
	)
	reconcileStatus = reconcileStatus.WithReconcileStatus(rs)

	if needsReschedule, err := reconcileStatus.NeedsReschedule(); needsReschedule {
		if err != nil {
			ctrl.LoggerFrom(ctx).Info("fetch dependency before listing limit for import", "error", err)
		}

		return nil, reconcileStatus
	}

	listOpts := limits.ListOpts{
		ServiceID:    ptr.Deref(service.Status.ID, ""),
		ProjectID:    ptr.Deref(project.Status.ID, ""),
		DomainID:     ptr.Deref(domain.Status.ID, ""),
		ResourceName: filter.ResourceName,
	}

	return actuator.osClient.ListLimits(ctx, listOpts), reconcileStatus
}

func (actuator limitActuator) CreateResource(ctx context.Context, obj orcObjectPT) (*osResourceT, progress.ReconcileStatus) {
	resource := obj.Spec.Resource

	if resource == nil {
		// Should have been caught by API validation
		return nil, progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Creation requested, but spec.resource is not set"))
	}
	var reconcileStatus progress.ReconcileStatus

	var serviceID string
	service, serviceDepRS := serviceDependency.GetDependency(
		ctx, actuator.k8sClient, obj, orcv1alpha1.IsAvailable,
	)
	reconcileStatus = reconcileStatus.WithReconcileStatus(serviceDepRS)
	if service != nil {
		serviceID = ptr.Deref(service.Status.ID, "")
	}

	var projectID string
	if resource.ProjectRef != nil {
		project, projectDepRS := projectDependency.GetDependency(
			ctx, actuator.k8sClient, obj, orcv1alpha1.IsAvailable,
		)
		reconcileStatus = reconcileStatus.WithReconcileStatus(projectDepRS)
		if project != nil {
			projectID = ptr.Deref(project.Status.ID, "")
		}
	}

	var domainID string
	if resource.DomainRef != nil {
		domain, domainDepRS := domainDependency.GetDependency(
			ctx, actuator.k8sClient, obj, orcv1alpha1.IsAvailable,
		)
		reconcileStatus = reconcileStatus.WithReconcileStatus(domainDepRS)
		if domain != nil {
			domainID = ptr.Deref(domain.Status.ID, "")
		}
	}
	if needsReschedule, err := reconcileStatus.NeedsReschedule(); needsReschedule {
		if err != nil {
			ctrl.LoggerFrom(ctx).Info("fetch dependency before creating limit", "error", err)
		}

		return nil, reconcileStatus
	}
	createOpts := limits.CreateOpts{
		Description:   ptr.Deref(resource.Description, ""),
		ServiceID:     serviceID,
		ProjectID:     projectID,
		DomainID:      domainID,
		ResourceName:  resource.ResourceName,
		ResourceLimit: int(resource.ResourceLimit),
	}

	osResource, err := actuator.osClient.CreateLimit(ctx, limits.BatchCreateOpts{createOpts})
	if err != nil {
		if !orcerrors.IsRetryable(err) {
			err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration creating resource: "+err.Error(), err)
		}
		return nil, progress.WrapError(err)
	}

	return osResource, nil
}

func (actuator limitActuator) DeleteResource(ctx context.Context, _ orcObjectPT, resource *osResourceT) progress.ReconcileStatus {
	return progress.WrapError(actuator.osClient.DeleteLimit(ctx, resource.ID))
}

func (actuator limitActuator) updateResource(ctx context.Context, obj orcObjectPT, osResource *osResourceT) progress.ReconcileStatus {
	log := ctrl.LoggerFrom(ctx)
	resource := obj.Spec.Resource
	if resource == nil {
		// Should have been caught by API validation
		return progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Update requested, but spec.resource is not set"))
	}

	if err := validateUpdate(resource, osResource); err != nil {
		return progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration updating resource: "+err.Error(), err))
	}

	updateOpts := limits.UpdateOpts{}

	// There seems to be a bug that keystone doesn't clear the description field when receiving a PATCH request with an empty description.
	// This will cause the `Progressing` condition stuck with `Resource status will be refreshed`.
	// Tested with `openstack limit set --description ""`
	handleDescriptionUpdate(&updateOpts, resource, osResource)
	// The same issue exists with resourceLimit. Updating resourceLimit with 0 doesn't work.
	handleResourceLimitUpdate(&updateOpts, resource, osResource)

	needsUpdate, err := needsUpdate(updateOpts)
	if err != nil {
		return progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration updating resource: "+err.Error(), err))
	}
	if !needsUpdate {
		log.V(logging.Debug).Info("No changes")
		return nil
	}

	_, err = actuator.osClient.UpdateLimit(ctx, osResource.ID, updateOpts)

	if err != nil {
		if !orcerrors.IsRetryable(err) {
			err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration updating resource: "+err.Error(), err)
		}
		return progress.WrapError(err)
	}

	return progress.NeedsRefresh()
}

func needsUpdate(updateOpts limits.UpdateOpts) (bool, error) {
	updateOptsMap, err := updateOpts.ToLimitUpdateMap()
	if err != nil {
		return false, err
	}

	updateMap, ok := updateOptsMap["limit"].(map[string]any)
	if !ok {
		updateMap = make(map[string]any)
	}

	return len(updateMap) > 0, nil
}

func handleDescriptionUpdate(updateOpts *limits.UpdateOpts, resource *resourceSpecT, osResource *osResourceT) {
	description := ptr.Deref(resource.Description, "")
	if osResource.Description != description {
		updateOpts.Description = &description
	}
}

func handleResourceLimitUpdate(updateOpts *limits.UpdateOpts, resource *resourceSpecT, osResource *osResourceT) {
	rl := int(resource.ResourceLimit)
	if osResource.ResourceLimit != rl {
		updateOpts.ResourceLimit = &rl
	}
}

func validateUpdate(resource *resourceSpecT, osResource *osResourceT) error {
	if resource.DomainRef != nil && osResource.ProjectID != "" {
		return errInvalidDomainRefUpdate
	}

	if resource.ProjectRef != nil && osResource.DomainID != "" {
		return errInvalidProjectRefUpdate
	}

	return nil
}

func (actuator limitActuator) GetResourceReconcilers(ctx context.Context, orcObject orcObjectPT, osResource *osResourceT, controller interfaces.ResourceController) ([]resourceReconciler, progress.ReconcileStatus) {
	return []resourceReconciler{
		actuator.updateResource,
	}, nil
}

type limitHelperFactory struct{}

var _ helperFactory = limitHelperFactory{}

func newActuator(ctx context.Context, orcObject *orcv1alpha1.Limit, controller interfaces.ResourceController) (limitActuator, progress.ReconcileStatus) {
	log := ctrl.LoggerFrom(ctx)

	// Ensure credential secrets exist and have our finalizer
	_, reconcileStatus := credentialsDependency.GetDependencies(ctx, controller.GetK8sClient(), orcObject, func(*corev1.Secret) bool { return true })
	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return limitActuator{}, reconcileStatus
	}

	clientScope, err := controller.GetScopeFactory().NewClientScopeFromObject(ctx, controller.GetK8sClient(), log, orcObject)
	if err != nil {
		return limitActuator{}, progress.WrapError(err)
	}
	osClient, err := clientScope.NewLimitClient()
	if err != nil {
		return limitActuator{}, progress.WrapError(err)
	}

	return limitActuator{
		osClient:  osClient,
		k8sClient: controller.GetK8sClient(),
	}, nil
}

func (limitHelperFactory) NewAPIObjectAdapter(obj orcObjectPT) adapterI {
	return limitAdapter{obj}
}

func (limitHelperFactory) NewCreateActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (createResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, orcObject, controller)
}

func (limitHelperFactory) NewDeleteActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (deleteResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, orcObject, controller)
}
