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

package sharetype

import (
	"context"
	"iter"

	"github.com/gophercloud/gophercloud/v2/openstack/sharedfilesystems/v2/sharetypes"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/progress"
	"github.com/k-orc/openstack-resource-controller/v2/internal/osclients"
	orcerrors "github.com/k-orc/openstack-resource-controller/v2/internal/util/errors"
)

// OpenStack resource types
type (
	osResourceT = sharetypes.ShareType

	createResourceActuator = interfaces.CreateResourceActuator[orcObjectPT, orcObjectT, filterT, osResourceT]
	deleteResourceActuator = interfaces.DeleteResourceActuator[orcObjectPT, orcObjectT, osResourceT]
	helperFactory          = interfaces.ResourceHelperFactory[orcObjectPT, orcObjectT, resourceSpecT, filterT, osResourceT]
)

type sharetypeActuator struct {
	osClient  osclients.ShareTypeClient
	k8sClient client.Client
}

var _ createResourceActuator = sharetypeActuator{}
var _ deleteResourceActuator = sharetypeActuator{}

func (sharetypeActuator) GetResourceID(osResource *osResourceT) string {
	return osResource.ID
}

func (actuator sharetypeActuator) GetOSResourceByID(ctx context.Context, id string) (*osResourceT, progress.ReconcileStatus) {
	// ShareTypes don't have a Get by ID API, so we list and filter
	// We need to list without isPublic filter to get all share types (public + private)
	listOpts := sharetypes.ListOpts{}
	for shareType, err := range actuator.osClient.ListShareTypes(ctx, listOpts) {
		if err != nil {
			return nil, progress.WrapError(err)
		}
		if shareType.ID == id {
			return shareType, nil
		}
	}
	// Not found - return nil with no error so generic controller will attempt creation
	return nil, nil
}

func (actuator sharetypeActuator) ListOSResourcesForAdoption(ctx context.Context, orcObject orcObjectPT) (iter.Seq2[*osResourceT, error], bool) {
	resourceSpec := orcObject.Spec.Resource
	if resourceSpec == nil {
		return nil, false
	}

	listOpts := sharetypes.ListOpts{}

	if resourceSpec.IsPublic != nil {
		if *resourceSpec.IsPublic {
			listOpts.IsPublic = "true"
		} else {
			listOpts.IsPublic = "false"
		}
	}

	filters := make([]osclients.ResourceFilter[osResourceT], 0, 1)
	filters = append(filters, func(st *sharetypes.ShareType) bool {
		return st.Name == getResourceName(orcObject)
	})

	resources := actuator.osClient.ListShareTypes(ctx, listOpts)
	return osclients.Filter(resources, filters...), true
}

func (actuator sharetypeActuator) ListOSResourcesForImport(ctx context.Context, obj orcObjectPT, filter filterT) (iter.Seq2[*osResourceT, error], progress.ReconcileStatus) {
	listOpts := sharetypes.ListOpts{}

	if filter.IsPublic != nil {
		if *filter.IsPublic {
			listOpts.IsPublic = "true"
		} else {
			listOpts.IsPublic = "false"
		}
	} else {
		// When isPublic filter is not specified, list ALL share types (both public and private)
		listOpts.IsPublic = "all"
	}

	var filters []osclients.ResourceFilter[osResourceT]
	if filter.Name != nil {
		name := string(*filter.Name)
		filters = append(filters, func(st *sharetypes.ShareType) bool {
			return st.Name == name
		})
	}

	resources := actuator.osClient.ListShareTypes(ctx, listOpts)
	return osclients.Filter(resources, filters...), nil
}

func (actuator sharetypeActuator) CreateResource(ctx context.Context, obj orcObjectPT) (*osResourceT, progress.ReconcileStatus) {
	resource := obj.Spec.Resource

	if resource == nil {
		// Should have been caught by API validation
		return nil, progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Creation requested, but spec.resource is not set"))
	}

	// Build extra specs manually because gophercloud's ExtraSpecsOpts doesn't handle
	// boolean false values correctly with the required:"true" tag
	dhss := ptr.Deref(resource.DriverHandlesShareServers, true)
	extraSpecsMap := map[string]interface{}{
		"driver_handles_share_servers": dhss,
	}
	if resource.SnapshotSupport != nil {
		extraSpecsMap["snapshot_support"] = *resource.SnapshotSupport
	}

	// Create custom CreateOpts with manually built extra specs
	createOptsMap := map[string]interface{}{
		"share_type": map[string]interface{}{
			"name":                           getResourceName(obj),
			"os-share-type-access:is_public": ptr.Deref(resource.IsPublic, true),
			"extra_specs":                    extraSpecsMap,
		},
	}

	osResource, err := actuator.osClient.CreateShareType(ctx, &customCreateOpts{optsMap: createOptsMap})
	if err != nil {
		if !orcerrors.IsRetryable(err) {
			err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration creating resource: "+err.Error(), err)
		}
		return nil, progress.WrapError(err)
	}

	return osResource, nil
}

// customCreateOpts implements CreateOptsBuilder to avoid gophercloud's BuildRequestBody issues with bool fields
type customCreateOpts struct {
	optsMap map[string]interface{}
}

func (opts *customCreateOpts) ToShareTypeCreateMap() (map[string]any, error) {
	return opts.optsMap, nil
}

func (actuator sharetypeActuator) DeleteResource(ctx context.Context, _ orcObjectPT, resource *osResourceT) progress.ReconcileStatus {
	return progress.WrapError(actuator.osClient.DeleteShareType(ctx, resource.ID))
}

type sharetypeHelperFactory struct{}

var _ helperFactory = sharetypeHelperFactory{}

func newActuator(ctx context.Context, orcObject *orcv1alpha1.ShareType, controller interfaces.ResourceController) (sharetypeActuator, progress.ReconcileStatus) {
	log := ctrl.LoggerFrom(ctx)

	// Ensure credential secrets exist and have our finalizer
	_, reconcileStatus := credentialsDependency.GetDependencies(ctx, controller.GetK8sClient(), orcObject, func(*corev1.Secret) bool { return true })
	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return sharetypeActuator{}, reconcileStatus
	}

	clientScope, err := controller.GetScopeFactory().NewClientScopeFromObject(ctx, controller.GetK8sClient(), log, orcObject)
	if err != nil {
		return sharetypeActuator{}, progress.WrapError(err)
	}
	osClient, err := clientScope.NewShareTypeClient()
	if err != nil {
		return sharetypeActuator{}, progress.WrapError(err)
	}

	return sharetypeActuator{
		osClient:  osClient,
		k8sClient: controller.GetK8sClient(),
	}, nil
}

func (sharetypeHelperFactory) NewAPIObjectAdapter(obj orcObjectPT) adapterI {
	return sharetypeAdapter{obj}
}

func (sharetypeHelperFactory) NewCreateActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (createResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, orcObject, controller)
}

func (sharetypeHelperFactory) NewDeleteActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (deleteResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, orcObject, controller)
}
