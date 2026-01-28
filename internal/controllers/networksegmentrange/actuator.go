/*
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

package networksegmentrange

import (
	"context"
	"iter"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/networksegmentranges"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/progress"
	"github.com/k-orc/openstack-resource-controller/v2/internal/osclients"
	"github.com/k-orc/openstack-resource-controller/v2/internal/util/dependency"
	orcerrors "github.com/k-orc/openstack-resource-controller/v2/internal/util/errors"
)

type (
	orcObjectPT  = *orcv1alpha1.NetworkSegmentRange
	orcObjectT   = orcv1alpha1.NetworkSegmentRange
	resourceSpecT = orcv1alpha1.NetworkSegmentRangeResourceSpec
	filterT      = orcv1alpha1.NetworkSegmentRangeFilter
	osResourceT  = networksegmentranges.NetworkSegmentRange

	createResourceActuator    = interfaces.CreateResourceActuator[orcObjectPT, orcObjectT, filterT, osResourceT]
	deleteResourceActuator    = interfaces.DeleteResourceActuator[orcObjectPT, orcObjectT, osResourceT]
	reconcileResourceActuator = interfaces.ReconcileResourceActuator[orcObjectPT, osResourceT]
	resourceReconciler        = interfaces.ResourceReconciler[orcObjectPT, osResourceT]
	helperFactory             = interfaces.ResourceHelperFactory[orcObjectPT, orcObjectT, resourceSpecT, filterT, osResourceT]
	adapterI                  = interfaces.ResourceAdapter[orcObjectPT, resourceSpecT]
)

type networksegmentrangeActuator struct {
	osClient  osclients.NetworkClient
	k8sClient client.Client
}

var _ createResourceActuator = networksegmentrangeActuator{}
var _ deleteResourceActuator = networksegmentrangeActuator{}
var _ reconcileResourceActuator = networksegmentrangeActuator{}

type networksegmentrangeHelperFactory struct{}

var _ helperFactory = networksegmentrangeHelperFactory{}

func (networksegmentrangeHelperFactory) NewAPIObjectAdapter(obj orcObjectPT) adapterI {
	return &apiObjectAdapter{obj}
}

func (networksegmentrangeHelperFactory) NewCreateActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (createResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, orcObject, controller)
}

func (networksegmentrangeHelperFactory) NewDeleteActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (deleteResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, orcObject, controller)
}

func newActuator(ctx context.Context, orcObject *orcv1alpha1.NetworkSegmentRange, controller interfaces.ResourceController) (networksegmentrangeActuator, progress.ReconcileStatus) {
	log := controller.GetLogger()

	networkClient, err := controller.GetScopeFactory().NewNetworkClient(ctx, orcObject, log)
	if err != nil {
		return networksegmentrangeActuator{}, progress.WrapError(orcerrors.Terminal(orcv1alpha1.OpenStackConditionReasonInvalidConfiguration, "Unable to obtain network client", err))
	}

	return networksegmentrangeActuator{
		osClient:  networkClient,
		k8sClient: controller.GetK8sClient(),
	}, nil
}

func (networksegmentrangeActuator) GetResourceID(osResource *osResourceT) string {
	return osResource.ID
}

func (actuator networksegmentrangeActuator) GetOSResourceByID(ctx context.Context, id string) (*osResourceT, progress.ReconcileStatus) {
	resource, err := actuator.osClient.GetNetworkSegmentRange(ctx, id)
	if err != nil {
		return nil, progress.WrapError(err)
	}
	return resource, nil
}

func (actuator networksegmentrangeActuator) ListOSResourcesForAdoption(ctx context.Context, obj orcObjectPT) (iter.Seq2[*osResourceT, error], bool) {
	if obj.Spec.Resource == nil {
		return nil, false
	}

	listOpts := networksegmentranges.ListOpts{Name: getResourceName(obj)}
	return actuator.osClient.ListNetworkSegmentRange(ctx, listOpts), true
}

func (actuator networksegmentrangeActuator) ListOSResourcesForImport(ctx context.Context, obj orcObjectPT, filter filterT) (iter.Seq2[*osResourceT, error], progress.ReconcileStatus) {
	var reconcileStatus progress.ReconcileStatus

	project, rs := dependency.FetchDependency(
		ctx, actuator.k8sClient, obj.Namespace, filter.ProjectRef, "Project",
		func(dep *orcv1alpha1.Project) bool {
			return orcv1alpha1.IsAvailable(dep) && dep.Status.ID != nil
		},
	)
	reconcileStatus = reconcileStatus.WithReconcileStatus(rs)

	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return nil, reconcileStatus
	}

	listOpts := networksegmentranges.ListOpts{
		Name:            string(ptr.Deref(filter.Name, "")),
		NetworkType:     ptr.Deref(filter.NetworkType, ""),
		PhysicalNetwork: ptr.Deref(filter.PhysicalNetwork, ""),
		Shared:          filter.Shared,
	}

	if project != nil {
		listOpts.ProjectID = ptr.Deref(project.Status.ID, "")
	}

	return actuator.osClient.ListNetworkSegmentRange(ctx, listOpts), reconcileStatus
}

func (actuator networksegmentrangeActuator) CreateResource(ctx context.Context, obj orcObjectPT) (*osResourceT, progress.ReconcileStatus) {
	resource := obj.Spec.Resource
	if resource == nil {
		return nil, progress.WrapError(orcerrors.Terminal(orcv1alpha1.OpenStackConditionReasonInvalidConfiguration, "Creation requested, but spec.resource is not set"))
	}

	var reconcileStatus progress.ReconcileStatus

	project, rs := dependency.FetchDependency(
		ctx, actuator.k8sClient, obj.Namespace, resource.ProjectRef, "Project",
		func(dep *orcv1alpha1.Project) bool {
			return orcv1alpha1.IsAvailable(dep) && dep.Status.ID != nil
		},
	)
	reconcileStatus = reconcileStatus.WithReconcileStatus(rs)

	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return nil, reconcileStatus
	}

	createOpts := networksegmentranges.CreateOpts{
		Name:            getResourceName(obj),
		NetworkType:     resource.NetworkType,
		PhysicalNetwork: ptr.Deref(resource.PhysicalNetwork, ""),
		Minimum:         resource.Minimum,
		Maximum:         resource.Maximum,
		Shared:          resource.Shared,
	}

	if project != nil {
		createOpts.ProjectID = ptr.Deref(project.Status.ID, "")
	}

	osResource, err := actuator.osClient.CreateNetworkSegmentRange(ctx, createOpts)
	if err != nil {
		return nil, progress.WrapError(err)
	}

	return osResource, reconcileStatus
}

func (actuator networksegmentrangeActuator) DeleteResource(ctx context.Context, _ orcObjectPT, osResource *osResourceT) progress.ReconcileStatus {
	if err := actuator.osClient.DeleteNetworkSegmentRange(ctx, osResource.ID); err != nil {
		return progress.WrapError(err)
	}
	return nil
}

func (actuator networksegmentrangeActuator) GetResourceReconcilers(ctx context.Context, orcObject orcObjectPT, osResource *osResourceT, controller interfaces.ResourceController) ([]resourceReconciler, progress.ReconcileStatus) {
	return []resourceReconciler{
		func(ctx context.Context, orcObject orcObjectPT, osResource *osResourceT) progress.ReconcileStatus {
			return actuator.updateResource(ctx, orcObject, osResource)
		},
	}, nil
}

func needsUpdate(updateOpts networksegmentranges.UpdateOpts) (bool, error) {
	return updateOpts.Name != nil || updateOpts.Minimum != nil || updateOpts.Maximum != nil, nil
}

func handleNameUpdate(updateOpts *networksegmentranges.UpdateOpts, obj orcObjectPT, osResource *osResourceT) {
	if getResourceName(obj) != osResource.Name {
		updateOpts.Name = ptr.To(getResourceName(obj))
	}
}

func (actuator networksegmentrangeActuator) updateResource(ctx context.Context, obj orcObjectPT, osResource *osResourceT) progress.ReconcileStatus {
	resource := obj.Spec.Resource
	if resource == nil {
		return nil
	}

	updateOpts := networksegmentranges.UpdateOpts{}
	handleNameUpdate(&updateOpts, obj, osResource)

	if needsUpdate, err := needsUpdate(updateOpts); err != nil {
		return progress.WrapError(err)
	} else if !needsUpdate {
		return nil
	}

	if _, err := actuator.osClient.UpdateNetworkSegmentRange(ctx, osResource.ID, updateOpts); err != nil {
		return progress.WrapError(err)
	}

	return progress.NeedsRefresh()
}
