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

package subnetpool

import (
	"context"
	"iter"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/subnetpools"
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
	"github.com/k-orc/openstack-resource-controller/v2/internal/util/tags"
)

// OpenStack resource types
type (
	osResourceT = subnetpools.SubnetPool

	createResourceActuator = interfaces.CreateResourceActuator[orcObjectPT, orcObjectT, filterT, osResourceT]
	deleteResourceActuator = interfaces.DeleteResourceActuator[orcObjectPT, orcObjectT, osResourceT]
	resourceReconciler     = interfaces.ResourceReconciler[orcObjectPT, osResourceT]
	helperFactory          = interfaces.ResourceHelperFactory[orcObjectPT, orcObjectT, resourceSpecT, filterT, osResourceT]
)

type subnetpoolActuator struct {
	osClient  osclients.SubnetPoolClient
	k8sClient client.Client
}

var (
	_ createResourceActuator = subnetpoolActuator{}
	_ deleteResourceActuator = subnetpoolActuator{}
)

func (subnetpoolActuator) GetResourceID(osResource *osResourceT) string {
	return osResource.ID
}

func (actuator subnetpoolActuator) GetOSResourceByID(ctx context.Context, id string) (*osResourceT, progress.ReconcileStatus) {
	resource, err := actuator.osClient.GetSubnetPool(ctx, id)
	if err != nil {
		return nil, progress.WrapError(err)
	}
	return resource, nil
}

func (actuator subnetpoolActuator) ListOSResourcesForAdoption(ctx context.Context, orcObject orcObjectPT) (iter.Seq2[*osResourceT, error], bool) {
	resourceSpec := orcObject.Spec.Resource
	if resourceSpec == nil {
		return nil, false
	}

	var projectID string
	if resourceSpec.ProjectRef != nil {
		project, rs := dependency.FetchDependency(
			ctx, actuator.k8sClient, orcObject.Namespace, resourceSpec.ProjectRef, "Project",
			func(dep *orcv1alpha1.Project) bool {
				return orcv1alpha1.IsAvailable(dep) && dep.Status.ID != nil
			},
		)
		if needsReschedule, _ := rs.NeedsReschedule(); needsReschedule {
			return nil, false
		}
		projectID = ptr.Deref(project.Status.ID, "")
	}

	listOpts := subnetpools.ListOpts{
		Name:      getResourceName(orcObject),
		ProjectID: projectID,
	}

	return actuator.osClient.ListSubnetPools(ctx, listOpts), true
}

func (actuator subnetpoolActuator) ListOSResourcesForImport(ctx context.Context, obj orcObjectPT, filter filterT) (iter.Seq2[*osResourceT, error], progress.ReconcileStatus) {
	var reconcileStatus progress.ReconcileStatus

	project, rs := dependency.FetchDependency[*orcv1alpha1.Project](
		ctx, actuator.k8sClient, obj.Namespace,
		filter.ProjectRef, "Project",
		orcv1alpha1.IsAvailable,
	)
	reconcileStatus = reconcileStatus.WithReconcileStatus(rs)

	addressScope, rs := dependency.FetchDependency[*orcv1alpha1.AddressScope](
		ctx, actuator.k8sClient, obj.Namespace,
		filter.AddressScopeRef, "AddressScope",
		orcv1alpha1.IsAvailable,
	)
	reconcileStatus = reconcileStatus.WithReconcileStatus(rs)

	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return nil, reconcileStatus
	}

	listOpts := subnetpools.ListOpts{
		Name:             string(ptr.Deref(filter.Name, "")),
		Description:      string(ptr.Deref(filter.Description, "")),
		ProjectID:        ptr.Deref(project.Status.ID, ""),
		AddressScopeID:   ptr.Deref(addressScope.Status.ID, ""),
		MinPrefixLen:     int(filter.MinPrefixLength),
		MaxPrefixLen:     int(filter.MaxPrefixLength),
		IPVersion:        int(filter.IPVersion),
		Shared:           filter.Shared,
		DefaultPrefixLen: int(filter.DefaultPrefixLength),
		IsDefault:        filter.IsDefault,
		RevisionNumber:   int(filter.RevisionNumber),
		Tags:             tags.Join(filter.Tags),
		TagsAny:          tags.Join(filter.TagsAny),
		NotTags:          tags.Join(filter.NotTags),
		NotTagsAny:       tags.Join(filter.NotTagsAny),
	}

	return actuator.osClient.ListSubnetPools(ctx, listOpts), reconcileStatus
}

func (actuator subnetpoolActuator) CreateResource(ctx context.Context, obj orcObjectPT) (*osResourceT, progress.ReconcileStatus) {
	resource := obj.Spec.Resource

	if resource == nil {
		// Should have been caught by API validation
		return nil, progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Creation requested, but spec.resource is not set"),
		)
	}
	var reconcileStatus progress.ReconcileStatus

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

	var addressScopeID string
	if resource.AddressScopeRef != nil {
		addressScope, addressScopeDepRS := addressScopeDependency.GetDependency(
			ctx, actuator.k8sClient, obj, orcv1alpha1.IsAvailable,
		)
		reconcileStatus = reconcileStatus.WithReconcileStatus(addressScopeDepRS)
		if addressScope != nil {
			addressScopeID = ptr.Deref(addressScope.Status.ID, "")
		}
	}
	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return nil, reconcileStatus
	}

	prefixes := make([]string, len(resource.Prefixes))
	for i, prefix := range resource.Prefixes {
		prefixes[i] = string(prefix)
	}

	createOpts := subnetpools.CreateOpts{
		Name:             getResourceName(obj),
		Description:      ptr.Deref(resource.Description, ""),
		ProjectID:        projectID,
		AddressScopeID:   addressScopeID,
		Prefixes:         prefixes,
		MinPrefixLen:     int(resource.MinPrefixLength),
		MaxPrefixLen:     int(resource.MaxPrefixLength),
		Shared:           ptr.Deref(resource.Shared, false),
		DefaultPrefixLen: int(resource.DefaultPrefixLength),
		IsDefault:        ptr.Deref(resource.IsDefault, false),
	}

	osResource, err := actuator.osClient.CreateSubnetPool(ctx, createOpts)
	if err != nil {
		if !orcerrors.IsRetryable(err) {
			err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration creating resource: "+err.Error(), err)
		}
		return nil, progress.WrapError(err)
	}

	return osResource, nil
}

func (actuator subnetpoolActuator) DeleteResource(ctx context.Context, _ orcObjectPT, resource *osResourceT) progress.ReconcileStatus {
	return progress.WrapError(actuator.osClient.DeleteSubnetPool(ctx, resource.ID))
}

func (actuator subnetpoolActuator) updateResource(ctx context.Context, obj orcObjectPT, osResource *osResourceT) progress.ReconcileStatus {
	log := ctrl.LoggerFrom(ctx)
	resource := obj.Spec.Resource
	if resource == nil {
		// Should have been caught by API validation
		return progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Update requested, but spec.resource is not set"),
		)
	}

	updateOpts := subnetpools.UpdateOpts{}

	handleNameUpdate(&updateOpts, obj, osResource)
	handleDescriptionUpdate(&updateOpts, resource, osResource)
	handleMinPrefixLengthUpdate(&updateOpts, resource, osResource)
	handleMaxPrefixLengthUpdate(&updateOpts, resource, osResource)
	handleIsDefaultUpdate(&updateOpts, resource, osResource)

	needsUpdate, err := needsUpdate(updateOpts)
	if err != nil {
		return progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration updating resource: "+err.Error(), err),
		)
	}
	if !needsUpdate {
		log.V(logging.Debug).Info("No changes")
		return nil
	}

	_, err = actuator.osClient.UpdateSubnetPool(ctx, osResource.ID, updateOpts)
	if err != nil {
		if !orcerrors.IsRetryable(err) {
			err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration updating resource: "+err.Error(), err)
		}
		return progress.WrapError(err)
	}

	return progress.NeedsRefresh()
}

func needsUpdate(updateOpts subnetpools.UpdateOpts) (bool, error) {
	updateOptsMap, err := updateOpts.ToSubnetPoolUpdateMap()
	if err != nil {
		return false, err
	}

	updateMap, ok := updateOptsMap["subnetpool"].(map[string]any)
	if !ok {
		updateMap = make(map[string]any)
	}

	return len(updateMap) > 0, nil
}

func handleNameUpdate(updateOpts *subnetpools.UpdateOpts, obj orcObjectPT, osResource *osResourceT) {
	name := getResourceName(obj)
	if osResource.Name != name {
		updateOpts.Name = name
	}
}

func handleDescriptionUpdate(updateOpts *subnetpools.UpdateOpts, resource *resourceSpecT, osResource *osResourceT) {
	description := ptr.Deref(resource.Description, "")
	if osResource.Description != description {
		updateOpts.Description = &description
	}
}

func handleMinPrefixLengthUpdate(updateOpts *subnetpools.UpdateOpts, resource *resourceSpecT, osResource *osResourceT) {
	minPrefixLen := resource.MinPrefixLength
	if minPrefixLen != int32(osResource.MinPrefixLen) {
		updateOpts.MinPrefixLen = int(minPrefixLen)
	}
}

func handleMaxPrefixLengthUpdate(updateOpts *subnetpools.UpdateOpts, resource *resourceSpecT, osResource *osResourceT) {
	maxPrefixLength := resource.MaxPrefixLength
	if maxPrefixLength != int32(osResource.MaxPrefixLen) {
		updateOpts.MaxPrefixLen = int(maxPrefixLength)
	}
}

func handleIsDefaultUpdate(updateOpts *subnetpools.UpdateOpts, resource *resourceSpecT, osResource *osResourceT) {
	// fallback to the default value if unset.
	isDefault := ptr.Deref(resource.IsDefault, false)
	if isDefault != osResource.IsDefault {
		updateOpts.IsDefault = &isDefault
	}
}

func (actuator subnetpoolActuator) GetResourceReconcilers(ctx context.Context, orcObject orcObjectPT, osResource *osResourceT, controller interfaces.ResourceController) ([]resourceReconciler, progress.ReconcileStatus) {
	return []resourceReconciler{
		actuator.updateResource,
	}, nil
}

type subnetpoolHelperFactory struct{}

var _ helperFactory = subnetpoolHelperFactory{}

func newActuator(ctx context.Context, orcObject *orcv1alpha1.SubnetPool, controller interfaces.ResourceController) (subnetpoolActuator, progress.ReconcileStatus) {
	log := ctrl.LoggerFrom(ctx)

	// Ensure credential secrets exist and have our finalizer
	_, reconcileStatus := credentialsDependency.GetDependencies(ctx, controller.GetK8sClient(), orcObject, func(*corev1.Secret) bool { return true })
	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return subnetpoolActuator{}, reconcileStatus
	}

	clientScope, err := controller.GetScopeFactory().NewClientScopeFromObject(ctx, controller.GetK8sClient(), log, orcObject)
	if err != nil {
		return subnetpoolActuator{}, progress.WrapError(err)
	}
	osClient, err := clientScope.NewSubnetPoolClient()
	if err != nil {
		return subnetpoolActuator{}, progress.WrapError(err)
	}

	return subnetpoolActuator{
		osClient:  osClient,
		k8sClient: controller.GetK8sClient(),
	}, nil
}

func (subnetpoolHelperFactory) NewAPIObjectAdapter(obj orcObjectPT) adapterI {
	return subnetpoolAdapter{obj}
}

func (subnetpoolHelperFactory) NewCreateActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (createResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, orcObject, controller)
}

func (subnetpoolHelperFactory) NewDeleteActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (deleteResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, orcObject, controller)
}
