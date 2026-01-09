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

package trunk

import (
	"context"
	"iter"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/trunks"
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

// OpenStack resource types
type (
	osResourceT = trunks.Trunk

	createResourceActuator = interfaces.CreateResourceActuator[orcObjectPT, orcObjectT, filterT, osResourceT]
	deleteResourceActuator = interfaces.DeleteResourceActuator[orcObjectPT, orcObjectT, osResourceT]
	resourceReconciler     = interfaces.ResourceReconciler[orcObjectPT, osResourceT]
	helperFactory          = interfaces.ResourceHelperFactory[orcObjectPT, orcObjectT, resourceSpecT, filterT, osResourceT]
)

type trunkActuator struct {
	osClient  osclients.TrunkClient
	k8sClient client.Client
}

var _ createResourceActuator = trunkActuator{}
var _ deleteResourceActuator = trunkActuator{}

func (trunkActuator) GetResourceID(osResource *osResourceT) string {
	return osResource.ID
}

func (actuator trunkActuator) GetOSResourceByID(ctx context.Context, id string) (*osResourceT, progress.ReconcileStatus) {
	resource, err := actuator.osClient.GetTrunk(ctx, id)
	if err != nil {
		return nil, progress.WrapError(err)
	}
	return resource, nil
}

func (actuator trunkActuator) ListOSResourcesForAdoption(ctx context.Context, orcObject orcObjectPT) (iter.Seq2[*osResourceT, error], bool) {
	resourceSpec := orcObject.Spec.Resource
	if resourceSpec == nil {
		return nil, false
	}

	// TODO(scaffolding) If you need to filter resources on fields that the List() function
	// of gophercloud does not support, it's possible to perform client-side filtering.
	// Check osclients.ResourceFilter

	listOpts := trunks.ListOpts{
		Name:        getResourceName(orcObject),
		Description: ptr.Deref(resourceSpec.Description, ""),
	}

	return actuator.osClient.ListTrunks(ctx, listOpts), true
}

func (actuator trunkActuator) ListOSResourcesForImport(ctx context.Context, obj orcObjectPT, filter filterT) (iter.Seq2[*osResourceT, error], progress.ReconcileStatus) {
	// TODO(scaffolding) If you need to filter resources on fields that the List() function
	// of gophercloud does not support, it's possible to perform client-side filtering.
	// Check osclients.ResourceFilter
	var reconcileStatus progress.ReconcileStatus

	port, rs := dependency.FetchDependency(
		ctx, actuator.k8sClient, obj.Namespace,
		filter.PortRef, "Port",
		func(dep *orcv1alpha1.Port) bool { return orcv1alpha1.IsAvailable(dep) && dep.Status.ID != nil },
	)
	reconcileStatus = reconcileStatus.WithReconcileStatus(rs)

	project, rs := dependency.FetchDependency(
		ctx, actuator.k8sClient, obj.Namespace,
		filter.ProjectRef, "Project",
		func(dep *orcv1alpha1.Project) bool { return orcv1alpha1.IsAvailable(dep) && dep.Status.ID != nil },
	)
	reconcileStatus = reconcileStatus.WithReconcileStatus(rs)

	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return nil, reconcileStatus
	}

	listOpts := trunks.ListOpts{
		Name:        string(ptr.Deref(filter.Name, "")),
		Description: string(ptr.Deref(filter.Description, "")),
		PortID:  ptr.Deref(port.Status.ID, ""),
		ProjectID:  ptr.Deref(project.Status.ID, ""),
		// TODO(scaffolding): Add more import filters
	}

	return actuator.osClient.ListTrunks(ctx, listOpts), reconcileStatus
}

func (actuator trunkActuator) CreateResource(ctx context.Context, obj orcObjectPT) (*osResourceT, progress.ReconcileStatus) {
	resource := obj.Spec.Resource

	if resource == nil {
		// Should have been caught by API validation
		return nil, progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Creation requested, but spec.resource is not set"))
	}
	var reconcileStatus progress.ReconcileStatus

	var portID string
        port, portDepRS := portDependency.GetDependency(
                ctx, actuator.k8sClient, obj, func(dep *orcv1alpha1.Port) bool {
                        return orcv1alpha1.IsAvailable(dep) && dep.Status.ID != nil
                },
        )
        reconcileStatus = reconcileStatus.WithReconcileStatus(portDepRS)
        if port != nil {
                portID = ptr.Deref(port.Status.ID, "")
        }

	var projectID string
	if resource.ProjectRef != nil {
		project, projectDepRS := projectDependency.GetDependency(
			ctx, actuator.k8sClient, obj, func(dep *orcv1alpha1.Project) bool {
				return orcv1alpha1.IsAvailable(dep) && dep.Status.ID != nil
			},
		)
		reconcileStatus = reconcileStatus.WithReconcileStatus(projectDepRS)
		if project != nil {
			projectID = ptr.Deref(project.Status.ID, "")
		}
	}
	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return nil, reconcileStatus
	}
	createOpts := trunks.CreateOpts{
		Name:        getResourceName(obj),
		Description: ptr.Deref(resource.Description, ""),
		PortID:  portID,
		ProjectID:  projectID,
		// TODO(scaffolding): Add more fields
	}

	osResource, err := actuator.osClient.CreateTrunk(ctx, createOpts)
	if err != nil {
		// We should require the spec to be updated before retrying a create which returned a conflict
		if !orcerrors.IsRetryable(err) {
			err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration creating resource: "+err.Error(), err)
		}
		return nil, progress.WrapError(err)
	}

	return osResource, nil
}

func (actuator trunkActuator) DeleteResource(ctx context.Context, _ orcObjectPT, resource *osResourceT) progress.ReconcileStatus {
	return progress.WrapError(actuator.osClient.DeleteTrunk(ctx, resource.ID))
}

func (actuator trunkActuator) updateResource(ctx context.Context, obj orcObjectPT, osResource *osResourceT) progress.ReconcileStatus {
	log := ctrl.LoggerFrom(ctx)
	resource := obj.Spec.Resource
	if resource == nil {
		// Should have been caught by API validation
		return progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Update requested, but spec.resource is not set"))
	}

	updateOpts := trunks.UpdateOpts{}

	handleNameUpdate(&updateOpts, obj, osResource)
	handleDescriptionUpdate(&updateOpts, resource, osResource)

	// TODO(scaffolding): add handler for all fields supporting mutability

	needsUpdate, err := needsUpdate(updateOpts)
	if err != nil {
		return progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration updating resource: "+err.Error(), err))
	}
	if !needsUpdate {
		log.V(logging.Debug).Info("No changes")
		return nil
	}

	_, err = actuator.osClient.UpdateTrunk(ctx, osResource.ID, updateOpts)

	// We should require the spec to be updated before retrying an update which returned a conflict
	if orcerrors.IsConflict(err) {
		err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration updating resource: "+err.Error(), err)
	}

	if err != nil {
		return progress.WrapError(err)
	}

	return progress.NeedsRefresh()
}

func needsUpdate(updateOpts trunks.UpdateOpts) (bool, error) {
	updateOptsMap, err := updateOpts.ToTrunkUpdateMap()
	if err != nil {
		return false, err
	}

	updateMap, ok := updateOptsMap["trunk"].(map[string]any)
	if !ok {
		updateMap = make(map[string]any)
	}

	return len(updateMap) > 0, nil
}

func handleNameUpdate(updateOpts *trunks.UpdateOpts, obj orcObjectPT, osResource *osResourceT) {
	name := getResourceName(obj)
	if osResource.Name != name {
		updateOpts.Name = &name
	}
}

func handleDescriptionUpdate(updateOpts *trunks.UpdateOpts, resource *resourceSpecT, osResource *osResourceT) {
	description := ptr.Deref(resource.Description, "")
	if osResource.Description != description {
		updateOpts.Description = &description
	}
}

func (actuator trunkActuator) GetResourceReconcilers(ctx context.Context, orcObject orcObjectPT, osResource *osResourceT, controller interfaces.ResourceController) ([]resourceReconciler, progress.ReconcileStatus) {
	return []resourceReconciler{
		actuator.updateResource,
	}, nil
}

type trunkHelperFactory struct{}

var _ helperFactory = trunkHelperFactory{}

func newActuator(ctx context.Context, orcObject *orcv1alpha1.Trunk, controller interfaces.ResourceController) (trunkActuator, progress.ReconcileStatus) {
	log := ctrl.LoggerFrom(ctx)

	// Ensure credential secrets exist and have our finalizer
	_, reconcileStatus := credentialsDependency.GetDependencies(ctx, controller.GetK8sClient(), orcObject, func(*corev1.Secret) bool { return true })
	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return trunkActuator{}, reconcileStatus
	}

	clientScope, err := controller.GetScopeFactory().NewClientScopeFromObject(ctx, controller.GetK8sClient(), log, orcObject)
	if err != nil {
		return trunkActuator{}, progress.WrapError(err)
	}
	osClient, err := clientScope.NewTrunkClient()
	if err != nil {
		return trunkActuator{}, progress.WrapError(err)
	}

	return trunkActuator{
		osClient:  osClient,
		k8sClient: controller.GetK8sClient(),
	}, nil
}

func (trunkHelperFactory) NewAPIObjectAdapter(obj orcObjectPT) adapterI {
	return trunkAdapter{obj}
}

func (trunkHelperFactory) NewCreateActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (createResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, orcObject, controller)
}

func (trunkHelperFactory) NewDeleteActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (deleteResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, orcObject, controller)
}
