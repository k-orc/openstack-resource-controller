/*
Copyright 2025 The ORC Authors.

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
	"fmt"
	"iter"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/trunks"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/progress"
	"github.com/k-orc/openstack-resource-controller/v2/internal/logging"
	osclients "github.com/k-orc/openstack-resource-controller/v2/internal/osclients"
	orcerrors "github.com/k-orc/openstack-resource-controller/v2/internal/util/errors"
	"github.com/k-orc/openstack-resource-controller/v2/internal/util/tags"
)

type (
	osResourceT = trunks.Trunk

	createResourceActuator    = interfaces.CreateResourceActuator[orcObjectPT, orcObjectT, filterT, osResourceT]
	deleteResourceActuator    = interfaces.DeleteResourceActuator[orcObjectPT, orcObjectT, osResourceT]
	reconcileResourceActuator = interfaces.ReconcileResourceActuator[orcObjectPT, osResourceT]
	resourceReconciler        = interfaces.ResourceReconciler[orcObjectPT, osResourceT]
	helperFactory             = interfaces.ResourceHelperFactory[orcObjectPT, orcObjectT, resourceSpecT, filterT, osResourceT]
	trunkIterator             = iter.Seq2[*osResourceT, error]
)

type trunkActuator struct {
	osClient  osclients.NetworkClient
	k8sClient client.Client
}

var _ createResourceActuator = trunkActuator{}
var _ deleteResourceActuator = trunkActuator{}

func (trunkActuator) GetResourceID(osResource *osResourceT) string {
	return osResource.ID
}

func (actuator trunkActuator) GetOSResourceByID(ctx context.Context, id string) (*osResourceT, progress.ReconcileStatus) {
	trunk, err := actuator.osClient.GetTrunk(ctx, id)
	if err != nil {
		return nil, progress.WrapError(err)
	}
	return trunk, nil
}

func (actuator trunkActuator) ListOSResourcesForAdoption(ctx context.Context, obj *orcv1alpha1.Trunk) (trunkIterator, bool) {
	if obj.Spec.Resource == nil {
		return nil, false
	}

	listOpts := trunks.ListOpts{Name: getResourceName(obj)}
	trunks, err := actuator.osClient.ListTrunk(ctx, listOpts)
	if err != nil {
		return func(yield func(*osResourceT, error) bool) {
			yield(nil, err)
		}, true
	}
	return func(yield func(*osResourceT, error) bool) {
		for i := range trunks {
			if !yield(&trunks[i], nil) {
				return
			}
		}
	}, true
}

func (actuator trunkActuator) ListOSResourcesForImport(ctx context.Context, obj orcObjectPT, filter filterT) (iter.Seq2[*osResourceT, error], progress.ReconcileStatus) {
	var reconcileStatus progress.ReconcileStatus

	port := &orcv1alpha1.Port{}
	if filter.PortRef != nil {
		portKey := client.ObjectKey{Name: string(*filter.PortRef), Namespace: obj.Namespace}
		if err := actuator.k8sClient.Get(ctx, portKey, port); err != nil {
			if apierrors.IsNotFound(err) {
				reconcileStatus = reconcileStatus.WithReconcileStatus(
					progress.WaitingOnObject("Port", portKey.Name, progress.WaitingOnCreation))
			} else {
				reconcileStatus = reconcileStatus.WithReconcileStatus(
					progress.WrapError(fmt.Errorf("fetching port %s: %w", portKey.Name, err)))
			}
		} else {
			if !orcv1alpha1.IsAvailable(port) || port.Status.ID == nil {
				reconcileStatus = reconcileStatus.WithReconcileStatus(
					progress.WaitingOnObject("Port", portKey.Name, progress.WaitingOnReady))
			}
		}
	}

	project := &orcv1alpha1.Project{}
	if filter.ProjectRef != nil {
		projectKey := client.ObjectKey{Name: string(*filter.ProjectRef), Namespace: obj.Namespace}
		if err := actuator.k8sClient.Get(ctx, projectKey, project); err != nil {
			if apierrors.IsNotFound(err) {
				reconcileStatus = reconcileStatus.WithReconcileStatus(
					progress.WaitingOnObject("Project", projectKey.Name, progress.WaitingOnCreation))
			} else {
				reconcileStatus = reconcileStatus.WithReconcileStatus(
					progress.WrapError(fmt.Errorf("fetching project %s: %w", projectKey.Name, err)))
			}
		} else {
			if !orcv1alpha1.IsAvailable(project) || project.Status.ID == nil {
				reconcileStatus = reconcileStatus.WithReconcileStatus(
					progress.WaitingOnObject("Project", projectKey.Name, progress.WaitingOnReady))
			}
		}
	}

	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return nil, reconcileStatus
	}

	listOpts := trunks.ListOpts{
		Name:      string(ptr.Deref(filter.Name, "")),
		Description: string(ptr.Deref(filter.Description, "")),
		PortID:    ptr.Deref(port.Status.ID, ""),
		ProjectID: ptr.Deref(project.Status.ID, ""),
		Tags:      tags.Join(filter.Tags),
		TagsAny:   tags.Join(filter.TagsAny),
		NotTags:   tags.Join(filter.NotTags),
		NotTagsAny: tags.Join(filter.NotTagsAny),
	}

	trunksList, err := actuator.osClient.ListTrunk(ctx, listOpts)
	if err != nil {
		return func(yield func(*osResourceT, error) bool) {
			yield(nil, err)
		}, nil
	}
	return func(yield func(*osResourceT, error) bool) {
		for i := range trunksList {
			if !yield(&trunksList[i], nil) {
				return
			}
		}
	}, nil
}

func (actuator trunkActuator) CreateResource(ctx context.Context, obj *orcv1alpha1.Trunk) (*osResourceT, progress.ReconcileStatus) {
	resource := obj.Spec.Resource
	if resource == nil {
		// Should have been caught by API validation
		return nil, progress.WrapError(orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Creation requested, but spec.resource is not set"))
	}

	// Fetch all dependencies and ensure they have our finalizer
	port, portDepRS := portDependency.GetDependency(
		ctx, actuator.k8sClient, obj, func(dep *orcv1alpha1.Port) bool {
			return orcv1alpha1.IsAvailable(dep) && dep.Status.ID != nil
		},
	)
	portMap, subportDepRS := subportDependency.GetDependencies(
		ctx, actuator.k8sClient, obj, func(dep *orcv1alpha1.Port) bool {
			return orcv1alpha1.IsAvailable(dep) && dep.Status.ID != nil
		},
	)
	reconcileStatus := progress.NewReconcileStatus().
		WithReconcileStatus(portDepRS).
		WithReconcileStatus(subportDepRS)

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
		PortID:       *port.Status.ID,
		Name:         getResourceName(obj),
		Description:  string(ptr.Deref(resource.Description, "")),
		AdminStateUp: resource.AdminStateUp,
		ProjectID:    projectID,
	}

	// Convert subports from spec to OpenStack format
	if len(resource.Subports) > 0 {
		createOpts.Subports = make([]trunks.Subport, len(resource.Subports))
		for i := range resource.Subports {
			portName := string(resource.Subports[i].PortRef)
			subportPort, ok := portMap[portName]
			if !ok {
				// Programming error
				return nil, progress.WrapError(fmt.Errorf("subport port %s was not returned by GetDependencies", portName))
			}
			createOpts.Subports[i] = trunks.Subport{
				PortID:           *subportPort.Status.ID,
				SegmentationType: resource.Subports[i].SegmentationType,
				SegmentationID:   int(resource.Subports[i].SegmentationID),
			}
		}
	}

	osResource, err := actuator.osClient.CreateTrunk(ctx, createOpts)
	if err != nil {
		// We should require the spec to be updated before retrying a create which returned a conflict
		if orcerrors.IsConflict(err) {
			err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration creating resource: "+err.Error(), err)
		}
		return nil, progress.WrapError(err)
	}

	return osResource, nil
}

func (actuator trunkActuator) DeleteResource(ctx context.Context, _ *orcv1alpha1.Trunk, osResource *osResourceT) progress.ReconcileStatus {
	return progress.WrapError(actuator.osClient.DeleteTrunk(ctx, osResource.ID))
}

var _ reconcileResourceActuator = trunkActuator{}

func (actuator trunkActuator) GetResourceReconcilers(ctx context.Context, orcObject orcObjectPT, osResource *osResourceT, controller interfaces.ResourceController) ([]resourceReconciler, progress.ReconcileStatus) {
	return []resourceReconciler{
		tags.ReconcileTags[orcObjectPT, osResourceT](orcObject.Spec.Resource.Tags, osResource.Tags, tags.NewNeutronTagReplacer(actuator.osClient, "trunks", osResource.ID)),
		actuator.updateResource,
		actuator.reconcileSubports,
	}, nil
}

func (actuator trunkActuator) updateResource(ctx context.Context, obj orcObjectPT, osResource *osResourceT) progress.ReconcileStatus {
	log := ctrl.LoggerFrom(ctx)
	resource := obj.Spec.Resource
	if resource == nil {
		return progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Update requested, but spec.resource is not set"))
	}

	var updateOpts trunks.UpdateOpts
	needsUpdate := false

	// Handle name update
	name := getResourceName(obj)
	if osResource.Name != name {
		updateOpts.Name = &name
		needsUpdate = true
	}

	// Handle description update
	description := string(ptr.Deref(resource.Description, ""))
	if osResource.Description != description {
		updateOpts.Description = &description
		needsUpdate = true
	}

	// Handle adminStateUp update
	if resource.AdminStateUp != nil && *resource.AdminStateUp != osResource.AdminStateUp {
		updateOpts.AdminStateUp = resource.AdminStateUp
		needsUpdate = true
	}

	if !needsUpdate {
		log.V(logging.Debug).Info("No changes")
		return nil
	}

	updateOpts.RevisionNumber = &osResource.RevisionNumber

	_, err := actuator.osClient.UpdateTrunk(ctx, osResource.ID, updateOpts)

	// We should require the spec to be updated before retrying an update which returned a conflict
	if orcerrors.IsConflict(err) {
		err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration updating resource: "+err.Error(), err)
	}

	if err != nil {
		return progress.WrapError(err)
	}

	return progress.NeedsRefresh()
}

func (actuator trunkActuator) reconcileSubports(ctx context.Context, obj orcObjectPT, osResource *osResourceT) progress.ReconcileStatus {
	log := ctrl.LoggerFrom(ctx)
	resource := obj.Spec.Resource
	if resource == nil {
		return nil
	}

	// Get current subports from OpenStack
	currentSubports, err := actuator.osClient.ListTrunkSubports(ctx, osResource.ID)
	if err != nil {
		return progress.WrapError(fmt.Errorf("failed to list trunk subports: %w", err))
	}

	// Get desired subports from spec
	portMap, subportDepRS := subportDependency.GetDependencies(
		ctx, actuator.k8sClient, obj, func(dep *orcv1alpha1.Port) bool {
			return orcv1alpha1.IsAvailable(dep) && dep.Status.ID != nil
		},
	)
	if needsReschedule, _ := subportDepRS.NeedsReschedule(); needsReschedule {
		return subportDepRS
	}

	desiredSubports := make([]trunks.Subport, 0, len(resource.Subports))
	for i := range resource.Subports {
		portName := string(resource.Subports[i].PortRef)
		subportPort, ok := portMap[portName]
		if !ok {
			// Port not ready yet, will be retried
			continue
		}
		desiredSubports = append(desiredSubports, trunks.Subport{
			PortID:           *subportPort.Status.ID,
			SegmentationType: resource.Subports[i].SegmentationType,
			SegmentationID:   int(resource.Subports[i].SegmentationID),
		})
	}

	// Build maps for comparison
	currentMap := make(map[string]trunks.Subport)
	for _, subport := range currentSubports {
		currentMap[subport.PortID] = subport
	}

	desiredMap := make(map[string]trunks.Subport)
	for _, subport := range desiredSubports {
		desiredMap[subport.PortID] = subport
	}

	// Find subports to add
	toAdd := []trunks.Subport{}
	for portID, desired := range desiredMap {
		if current, exists := currentMap[portID]; !exists {
			toAdd = append(toAdd, desired)
		} else if current.SegmentationType != desired.SegmentationType || current.SegmentationID != desired.SegmentationID {
			// Subport exists but with different segmentation, need to remove and re-add
			// First remove the old one
			removeOpts := trunks.RemoveSubportsOpts{
				Subports: []trunks.RemoveSubport{{PortID: portID}},
			}
			if err := actuator.osClient.RemoveSubports(ctx, osResource.ID, removeOpts); err != nil {
				return progress.WrapError(fmt.Errorf("failed to remove subport %s: %w", portID, err))
			}
			toAdd = append(toAdd, desired)
		}
	}

	// Find subports to remove
	toRemove := []trunks.RemoveSubport{}
	for portID := range currentMap {
		if _, exists := desiredMap[portID]; !exists {
			toRemove = append(toRemove, trunks.RemoveSubport{PortID: portID})
		}
	}

	// Apply changes
	if len(toRemove) > 0 {
		removeOpts := trunks.RemoveSubportsOpts{Subports: toRemove}
		if err := actuator.osClient.RemoveSubports(ctx, osResource.ID, removeOpts); err != nil {
			return progress.WrapError(fmt.Errorf("failed to remove subports: %w", err))
		}
		log.V(logging.Debug).Info("Removed subports", "count", len(toRemove))
	}

	if len(toAdd) > 0 {
		addOpts := trunks.AddSubportsOpts{Subports: toAdd}
		if _, err := actuator.osClient.AddSubports(ctx, osResource.ID, addOpts); err != nil {
			return progress.WrapError(fmt.Errorf("failed to add subports: %w", err))
		}
		log.V(logging.Debug).Info("Added subports", "count", len(toAdd))
	}

	if len(toAdd) > 0 || len(toRemove) > 0 {
		return progress.NeedsRefresh()
	}

	return nil
}

type trunkHelperFactory struct{}

var _ helperFactory = trunkHelperFactory{}

func (trunkHelperFactory) NewAPIObjectAdapter(obj orcObjectPT) adapterI {
	return trunkAdapter{obj}
}

func (trunkHelperFactory) NewCreateActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (createResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, controller, orcObject)
}

func (trunkHelperFactory) NewDeleteActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (deleteResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, controller, orcObject)
}

func newActuator(ctx context.Context, controller interfaces.ResourceController, orcObject *orcv1alpha1.Trunk) (trunkActuator, progress.ReconcileStatus) {
	if orcObject == nil {
		return trunkActuator{}, progress.WrapError(fmt.Errorf("orcObject may not be nil"))
	}

	// Ensure credential secrets exist and have our finalizer
	_, reconcileStatus := credentialsDependency.GetDependencies(ctx, controller.GetK8sClient(), orcObject, func(*corev1.Secret) bool { return true })
	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return trunkActuator{}, reconcileStatus
	}

	log := ctrl.LoggerFrom(ctx)
	clientScope, err := controller.GetScopeFactory().NewClientScopeFromObject(ctx, controller.GetK8sClient(), log, orcObject)
	if err != nil {
		return trunkActuator{}, progress.WrapError(err)
	}
	osClient, err := clientScope.NewNetworkClient()
	if err != nil {
		return trunkActuator{}, progress.WrapError(err)
	}

	return trunkActuator{
		osClient:  osClient,
		k8sClient: controller.GetK8sClient(),
	}, nil
}

