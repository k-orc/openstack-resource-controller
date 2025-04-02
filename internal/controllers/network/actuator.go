/*
Copyright 2024 The ORC Authors.

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

package network

import (
	"context"
	"iter"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/dns"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/external"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/mtu"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/portsecurity"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/progress"
	"github.com/k-orc/openstack-resource-controller/v2/internal/osclients"
	orcerrors "github.com/k-orc/openstack-resource-controller/v2/internal/util/errors"
	"github.com/k-orc/openstack-resource-controller/v2/internal/util/neutrontags"
)

type (
	osResourceT = osclients.NetworkExt

	createResourceActuator    = interfaces.CreateResourceActuator[orcObjectPT, orcObjectT, filterT, osResourceT]
	deleteResourceActuator    = interfaces.DeleteResourceActuator[orcObjectPT, orcObjectT, osResourceT]
	reconcileResourceActuator = interfaces.ReconcileResourceActuator[orcObjectPT, osResourceT]
	resourceReconciler        = interfaces.ResourceReconciler[orcObjectPT, osResourceT]
	helperFactory             = interfaces.ResourceHelperFactory[orcObjectPT, orcObjectT, resourceSpecT, filterT, osResourceT]
)

type networkActuator struct {
	osClient osclients.NetworkClient
}

var _ createResourceActuator = networkActuator{}
var _ deleteResourceActuator = networkActuator{}
var _ reconcileResourceActuator = networkActuator{}

func (networkActuator) GetResourceID(osResource *osclients.NetworkExt) string {
	return osResource.ID
}

func (actuator networkActuator) GetOSResourceByID(ctx context.Context, id string) (*osclients.NetworkExt, error) {
	return actuator.osClient.GetNetwork(ctx, id)
}

func (actuator networkActuator) ListOSResourcesForAdoption(ctx context.Context, obj orcObjectPT) (iter.Seq2[*osclients.NetworkExt, error], bool) {
	if obj.Spec.Resource == nil {
		return nil, false
	}

	listOpts := networks.ListOpts{Name: string(getResourceName(obj))}
	return actuator.osClient.ListNetwork(ctx, listOpts), true
}

func (actuator networkActuator) ListOSResourcesForImport(ctx context.Context, obj orcObjectPT, filter filterT) ([]progress.ProgressStatus, iter.Seq2[*osResourceT, error], error) {
	listOpts := networks.ListOpts{
		Name:        string(ptr.Deref(filter.Name, "")),
		Description: string(ptr.Deref(filter.Description, "")),
		Tags:        neutrontags.Join(filter.Tags),
		TagsAny:     neutrontags.Join(filter.TagsAny),
		NotTags:     neutrontags.Join(filter.NotTags),
		NotTagsAny:  neutrontags.Join(filter.NotTagsAny),
	}

	return nil, actuator.osClient.ListNetwork(ctx, listOpts), nil
}

func (actuator networkActuator) CreateResource(ctx context.Context, obj orcObjectPT) ([]progress.ProgressStatus, *osclients.NetworkExt, error) {
	resource := obj.Spec.Resource
	if resource == nil {
		// Should have been caught by API validation
		return nil, nil, orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Creation requested, but spec.resource is not set")
	}

	var createOpts networks.CreateOptsBuilder
	{
		createOptsBase := networks.CreateOpts{
			Name:         string(getResourceName(obj)),
			Description:  string(ptr.Deref(resource.Description, "")),
			AdminStateUp: resource.AdminStateUp,
			Shared:       resource.Shared,
		}

		if len(resource.AvailabilityZoneHints) > 0 {
			createOptsBase.AvailabilityZoneHints = make([]string, len(resource.AvailabilityZoneHints))
			for i := range resource.AvailabilityZoneHints {
				createOptsBase.AvailabilityZoneHints[i] = string(resource.AvailabilityZoneHints[i])
			}
		}
		createOpts = createOptsBase
	}

	if resource.DNSDomain != nil {
		createOpts = &dns.NetworkCreateOptsExt{
			CreateOptsBuilder: createOpts,
			DNSDomain:         string(*resource.DNSDomain),
		}
	}

	if resource.MTU != nil {
		createOpts = &mtu.CreateOptsExt{
			CreateOptsBuilder: createOpts,
			MTU:               int(*resource.MTU),
		}
	}

	if resource.PortSecurityEnabled != nil {
		createOpts = &portsecurity.NetworkCreateOptsExt{
			CreateOptsBuilder:   createOpts,
			PortSecurityEnabled: resource.PortSecurityEnabled,
		}
	}

	if resource.External != nil {
		createOpts = &external.CreateOptsExt{
			CreateOptsBuilder: createOpts,
			External:          resource.External,
		}
	}

	osResource, err := actuator.osClient.CreateNetwork(ctx, createOpts)
	if err != nil {
		// We should require the spec to be updated before retrying a create which returned a conflict
		if orcerrors.IsConflict(err) {
			err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration creating resource: "+err.Error(), err)
		}
		return nil, nil, err
	}

	return nil, osResource, nil
}

func (actuator networkActuator) DeleteResource(ctx context.Context, _ orcObjectPT, network *osclients.NetworkExt) ([]progress.ProgressStatus, error) {
	return nil, actuator.osClient.DeleteNetwork(ctx, network.ID)
}

func (actuator networkActuator) GetResourceReconcilers(ctx context.Context, orcObject orcObjectPT, osResource *osclients.NetworkExt, controller interfaces.ResourceController) ([]resourceReconciler, error) {
	return []resourceReconciler{
		neutrontags.ReconcileTags[orcObjectPT, osResourceT](actuator.osClient, "networks", osResource.ID, orcObject.Spec.Resource.Tags, osResource.Tags),
	}, nil
}

type networkHelperFactory struct{}

var _ helperFactory = networkHelperFactory{}

func newActuator(ctx context.Context, orcObject *orcv1alpha1.Network, controller interfaces.ResourceController) (networkActuator, []progress.ProgressStatus, error) {
	log := ctrl.LoggerFrom(ctx)

	// Ensure credential secrets exist and have our finalizer
	_, progressStatus, err := credentialsDependency.GetDependencies(ctx, controller.GetK8sClient(), orcObject, func(*corev1.Secret) bool { return true })
	if len(progressStatus) > 0 || err != nil {
		return networkActuator{}, progressStatus, err
	}

	clientScope, err := controller.GetScopeFactory().NewClientScopeFromObject(ctx, controller.GetK8sClient(), log, orcObject)
	if err != nil {
		return networkActuator{}, nil, err
	}
	osClient, err := clientScope.NewNetworkClient()
	if err != nil {
		return networkActuator{}, nil, err
	}

	return networkActuator{
		osClient: osClient,
	}, nil, nil
}

func (networkHelperFactory) NewAPIObjectAdapter(obj orcObjectPT) adapterI {
	return networkAdapter{obj}
}

func (networkHelperFactory) NewCreateActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) ([]progress.ProgressStatus, createResourceActuator, error) {
	actuator, progressStatus, err := newActuator(ctx, orcObject, controller)
	return progressStatus, actuator, err
}

func (networkHelperFactory) NewDeleteActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) ([]progress.ProgressStatus, deleteResourceActuator, error) {
	actuator, progressStatus, err := newActuator(ctx, orcObject, controller)
	return progressStatus, actuator, err
}
