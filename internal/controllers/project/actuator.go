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

package project

import (
	"context"
	"iter"
	"slices"

	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/projects"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	generic "github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/progress"
	orcerrors "github.com/k-orc/openstack-resource-controller/v2/internal/util/errors"
	"github.com/k-orc/openstack-resource-controller/v2/internal/util/neutrontags"
)

// OpenStack resource types
type (
	osResourceT = projects.Project

	createResourceActuator = generic.CreateResourceActuator[orcObjectPT, orcObjectT, filterT, osResourceT]
	deleteResourceActuator = generic.DeleteResourceActuator[orcObjectPT, orcObjectT, osResourceT]
	helperFactory          = generic.ResourceHelperFactory[orcObjectPT, orcObjectT, resourceSpecT, filterT, osResourceT]
)

type projectClient interface {
	GetProject(context.Context, string) (*projects.Project, error)
	ListProjects(context.Context, projects.ListOptsBuilder) iter.Seq2[*projects.Project, error]
	CreateProject(context.Context, projects.CreateOptsBuilder) (*projects.Project, error)
	DeleteProject(context.Context, string) error
}

type projectActuator struct {
	osClient projectClient
}

var _ createResourceActuator = projectActuator{}
var _ deleteResourceActuator = projectActuator{}

func (projectActuator) GetResourceID(osResource *projects.Project) string {
	return osResource.ID
}

func (actuator projectActuator) GetOSResourceByID(ctx context.Context, id string) (*projects.Project, progress.ReconcileStatus) {
	project, err := actuator.osClient.GetProject(ctx, id)
	if err != nil {
		return nil, progress.WrapError(err)
	}
	return project, nil
}

func (actuator projectActuator) ListOSResourcesForAdoption(ctx context.Context, obj orcObjectPT) (iter.Seq2[*projects.Project, error], bool) {
	if obj.Spec.Resource == nil {
		return nil, false
	}

	listOpts := projects.ListOpts{
		Name: getResourceName(obj),
		Tags: neutrontags.Join(obj.Spec.Resource.Tags),
	}

	return actuator.osClient.ListProjects(ctx, listOpts), true
}

func (actuator projectActuator) ListOSResourcesForImport(ctx context.Context, orcObject orcObjectPT, filter filterT) (iter.Seq2[*osResourceT, error], progress.ReconcileStatus) {
	listOpts := projects.ListOpts{
		Name:       string(ptr.Deref(filter.Name, "")),
		Tags:       neutrontags.Join(filter.Tags),
		TagsAny:    neutrontags.Join(filter.TagsAny),
		NotTags:    neutrontags.Join(filter.NotTags),
		NotTagsAny: neutrontags.Join(filter.NotTagsAny),
	}

	return actuator.osClient.ListProjects(ctx, listOpts), nil
}

func (actuator projectActuator) CreateResource(ctx context.Context, obj orcObjectPT) (*projects.Project, progress.ReconcileStatus) {
	resource := obj.Spec.Resource

	if resource == nil {
		// Should have been caught by API validation
		return nil, progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Creation requested, but spec.resource is not set"))
	}

	tags := make([]string, len(resource.Tags))
	for i := range resource.Tags {
		tags[i] = string(resource.Tags[i])
	}
	// Sort tags before creation to simplify comparisons
	slices.Sort(tags)

	createOpts := projects.CreateOpts{
		Name:        getResourceName(obj),
		Description: ptr.Deref(resource.Description, ""),
		Enabled:     resource.Enabled,
		Tags:        tags,
	}

	osResource, err := actuator.osClient.CreateProject(ctx, createOpts)
	if err != nil {
		// We should require the spec to be updated before retrying a create which returned a conflict
		if !orcerrors.IsRetryable(err) {
			err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration creating resource: "+err.Error(), err)
		}
		return nil, progress.WrapError(err)
	}

	return osResource, nil
}

func (actuator projectActuator) DeleteResource(ctx context.Context, _ orcObjectPT, project *projects.Project) progress.ReconcileStatus {
	return progress.WrapError(actuator.osClient.DeleteProject(ctx, project.ID))
}

type projectHelperFactory struct{}

var _ helperFactory = projectHelperFactory{}

func newActuator(ctx context.Context, orcObject *orcv1alpha1.Project, controller generic.ResourceController) (projectActuator, progress.ReconcileStatus) {
	log := ctrl.LoggerFrom(ctx)

	// Ensure credential secrets exist and have our finalizer
	_, reconcileStatus := credentialsDependency.GetDependencies(ctx, controller.GetK8sClient(), orcObject, func(*corev1.Secret) bool { return true })
	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return projectActuator{}, reconcileStatus
	}

	clientScope, err := controller.GetScopeFactory().NewClientScopeFromObject(ctx, controller.GetK8sClient(), log, orcObject)
	if err != nil {
		return projectActuator{}, progress.WrapError(err)
	}
	osClient, err := clientScope.NewIdentityClient()
	if err != nil {
		return projectActuator{}, progress.WrapError(err)
	}

	return projectActuator{
		osClient: osClient,
	}, nil
}

func (projectHelperFactory) NewAPIObjectAdapter(obj orcObjectPT) adapterI {
	return projectAdapter{obj}
}

func (projectHelperFactory) NewCreateActuator(ctx context.Context, orcObject orcObjectPT, controller generic.ResourceController) (createResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, orcObject, controller)
}

func (projectHelperFactory) NewDeleteActuator(ctx context.Context, orcObject orcObjectPT, controller generic.ResourceController) (deleteResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, orcObject, controller)
}
