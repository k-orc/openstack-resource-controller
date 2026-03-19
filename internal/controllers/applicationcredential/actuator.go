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

package applicationcredential

import (
	"context"
	"fmt"
	"iter"

	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/applicationcredentials"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/progress"
	"github.com/k-orc/openstack-resource-controller/v2/internal/osclients"
	"github.com/k-orc/openstack-resource-controller/v2/internal/util/dependency"
	orcerrors "github.com/k-orc/openstack-resource-controller/v2/internal/util/errors"
)

// OpenStack resource types
type (
	osResourceT = applicationcredentials.ApplicationCredential

	createResourceActuator = interfaces.CreateResourceActuator[orcObjectPT, orcObjectT, filterT, osResourceT]
	deleteResourceActuator = interfaces.DeleteResourceActuator[orcObjectPT, orcObjectT, osResourceT]
	helperFactory          = interfaces.ResourceHelperFactory[orcObjectPT, orcObjectT, resourceSpecT, filterT, osResourceT]
)

type applicationcredentialActuator struct {
	osClient  osclients.ApplicationCredentialClient
	k8sClient client.Client
}

var _ createResourceActuator = applicationcredentialActuator{}
var _ deleteResourceActuator = applicationcredentialActuator{}

func (applicationcredentialActuator) GetResourceID(osResource *osResourceT) string {
	return osResource.ID
}

func (actuator applicationcredentialActuator) GetOSResourceByID(ctx context.Context, id string) (*osResourceT, progress.ReconcileStatus) {
	resource, err := actuator.osClient.GetApplicationCredential(ctx, id)
	if err != nil {
		return nil, progress.WrapError(err)
	}
	return resource, nil
}

func (actuator applicationcredentialActuator) ResolveUserIDInFilter(ctx context.Context, orcObject orcObjectPT, userRef *orcv1alpha1.KubernetesNameRef) (*string, progress.ReconcileStatus) {
	user, rs := dependency.FetchDependency(
		ctx, actuator.k8sClient, orcObject.Namespace,
		userRef, "User",
		func(dep *orcv1alpha1.User) bool { return orcv1alpha1.IsAvailable(dep) && dep.Status.ID != nil },
	)

	if user == nil {
		return nil, rs
	}

	return user.Status.ID, rs
}

func (actuator applicationcredentialActuator) ResolveUserIDDependency(ctx context.Context, orcObject orcObjectPT) (*string, progress.ReconcileStatus) {
	user, rs := userDependency.GetDependency(
		ctx, actuator.k8sClient, orcObject, func(dep *orcv1alpha1.User) bool {
			return orcv1alpha1.IsAvailable(dep) && dep.Status.ID != nil
		},
	)

	if user == nil {
		return nil, rs
	}

	return user.Status.ID, rs
}

func (actuator applicationcredentialActuator) ListOSResourcesForAdoption(ctx context.Context, orcObject orcObjectPT) (iter.Seq2[*osResourceT, error], bool) {
	resourceSpec := orcObject.Spec.Resource
	if resourceSpec == nil {
		return nil, false
	}

	userID, _ := actuator.ResolveUserIDInFilter(ctx, orcObject, resourceSpec.UserRef)
	if userID == nil {
		return nil, false
	}

	var filters []osclients.ResourceFilter[osResourceT]

	// Add client-side filters
	if resourceSpec.Description != nil {
		filters = append(filters, func(f *applicationcredentials.ApplicationCredential) bool {
			return f.Description == *resourceSpec.Description
		})
	}

	listOpts := applicationcredentials.ListOpts{
		Name: getResourceName(orcObject),
	}

	return actuator.listOSResources(ctx, ptr.Deref(userID, ""), filters, listOpts), true
}

func (actuator applicationcredentialActuator) ListOSResourcesForImport(ctx context.Context, obj orcObjectPT, filter filterT) (iter.Seq2[*osResourceT, error], progress.ReconcileStatus) {
	var reconcileStatus progress.ReconcileStatus

	userID, rs := actuator.ResolveUserIDInFilter(ctx, obj, filter.UserRef)
	reconcileStatus = reconcileStatus.WithReconcileStatus(rs)

	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return nil, reconcileStatus
	}

	var filters []osclients.ResourceFilter[osResourceT]

	// Add client-side filters
	if filter.Description != nil {
		filters = append(filters, func(f *applicationcredentials.ApplicationCredential) bool {
			return f.Description == *filter.Description
		})
	}

	listOpts := applicationcredentials.ListOpts{
		Name: string(ptr.Deref(filter.Name, "")),
	}

	return actuator.listOSResources(ctx, ptr.Deref(userID, ""), filters, listOpts), nil
}

func (actuator applicationcredentialActuator) listOSResources(ctx context.Context, userID string, filters []osclients.ResourceFilter[osResourceT], listOpts applicationcredentials.ListOptsBuilder) iter.Seq2[*applicationcredentials.ApplicationCredential, error] {
	applicationCredentials := actuator.osClient.ListApplicationCredentials(ctx, userID, listOpts)
	return osclients.Filter(applicationCredentials, filters...)
}

func (actuator applicationcredentialActuator) CreateResource(ctx context.Context, obj orcObjectPT) (*osResourceT, progress.ReconcileStatus) {
	resource := obj.Spec.Resource

	if resource == nil {
		// Should have been caught by API validation
		return nil, progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Creation requested, but spec.resource is not set"))
	}

	var reconcileStatus progress.ReconcileStatus

	userID, userDepRS := actuator.ResolveUserIDDependency(ctx, obj)

	rolesMap, roleDepRs := roleDependency.GetDependencies(
		ctx, actuator.k8sClient, obj, func(dep *orcv1alpha1.Role) bool {
			return orcv1alpha1.IsAvailable(dep) && dep.Status.ID != nil
		},
	)

	reconcileStatus = reconcileStatus.
		WithReconcileStatus(userDepRS).
		WithReconcileStatus(roleDepRs)

	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return nil, reconcileStatus
	}

	roleList := make([]applicationcredentials.Role, len(resource.RoleRefs))
	for i := range resource.RoleRefs {
		roleName := string(resource.RoleRefs[i])
		role, ok := rolesMap[roleName]
		if !ok {
			// Programming error
			return nil, progress.WrapError(fmt.Errorf("role %s was not returned by GetDependencies", roleName))
		}
		roleList[i].ID = *role.Status.ID
		roleList[i].DomainID = role.Status.Resource.DomainID
		roleList[i].Name = role.Status.Resource.Name
	}

	accessRuleList := make([]applicationcredentials.AccessRule, len(resource.AccessRules))
	for i := range resource.AccessRules {
		accessRuleSpec := &resource.AccessRules[i]
		accessRule := &accessRuleList[i]

		if accessRuleSpec.Path != nil {
			accessRule.Path = *accessRuleSpec.Path
		}

		if accessRuleSpec.Service != nil {
			accessRule.Service = *accessRuleSpec.Service
		}

		if accessRuleSpec.Method != nil {
			accessRule.Method = string(*accessRuleSpec.Method)
		}
	}

	var secretData []byte

	secret, secretReconcileStatus := dependency.FetchDependency(
		ctx, actuator.k8sClient, obj.Namespace,
		resource.Secret.SecretRef, "Secret",
		func(*corev1.Secret) bool { return true }, // Secrets don't have availability status
	)
	reconcileStatus = reconcileStatus.WithReconcileStatus(secretReconcileStatus)
	if secretReconcileStatus == nil {
		var ok bool
		secretData, ok = secret.Data["value"]
		if !ok {
			reconcileStatus = reconcileStatus.WithReconcileStatus(
				progress.NewReconcileStatus().WithProgressMessage("Application credential secret does not contain \"value\" key"))
		}
	}

	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return nil, reconcileStatus
	}

	createOpts := applicationcredentials.CreateOpts{
		Name:         getResourceName(obj),
		Description:  ptr.Deref(resource.Description, ""),
		Unrestricted: ptr.Deref(resource.Unrestricted, false),
		Secret:       string(secretData),
		Roles:        roleList,
		AccessRules:  accessRuleList,
	}

	if resource.ExpiresAt != nil {
		createOpts.ExpiresAt = &resource.ExpiresAt.Time
	}

	osResource, err := actuator.osClient.CreateApplicationCredential(ctx, ptr.Deref(userID, ""), createOpts)
	if err != nil {
		// We should require the spec to be updated before retrying a create which returned a conflict
		if !orcerrors.IsRetryable(err) {
			err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration creating resource: "+err.Error(), err)
		}
		return nil, progress.WrapError(err)
	}

	return osResource, nil
}

func (actuator applicationcredentialActuator) DeleteResource(ctx context.Context, orcObject orcObjectPT, resource *osResourceT) progress.ReconcileStatus {
	var reconcileStatus progress.ReconcileStatus

	userID, userDepRS := actuator.ResolveUserIDDependency(ctx, orcObject)
	reconcileStatus = reconcileStatus.WithReconcileStatus(userDepRS)

	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return reconcileStatus
	}

	return progress.WrapError(actuator.osClient.DeleteApplicationCredential(ctx, ptr.Deref(userID, ""), resource.ID))
}

type applicationcredentialHelperFactory struct{}

var _ helperFactory = applicationcredentialHelperFactory{}

func newActuator(ctx context.Context, orcObject *orcv1alpha1.ApplicationCredential, controller interfaces.ResourceController) (applicationcredentialActuator, progress.ReconcileStatus) {
	log := ctrl.LoggerFrom(ctx)

	// Ensure credential secrets exist and have our finalizer
	_, reconcileStatus := credentialsDependency.GetDependencies(ctx, controller.GetK8sClient(), orcObject, func(*corev1.Secret) bool { return true })
	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return applicationcredentialActuator{}, reconcileStatus
	}

	clientScope, err := controller.GetScopeFactory().NewClientScopeFromObject(ctx, controller.GetK8sClient(), log, orcObject)
	if err != nil {
		return applicationcredentialActuator{}, progress.WrapError(err)
	}
	osClient, err := clientScope.NewApplicationCredentialClient()
	if err != nil {
		return applicationcredentialActuator{}, progress.WrapError(err)
	}

	return applicationcredentialActuator{
		osClient:  osClient,
		k8sClient: controller.GetK8sClient(),
	}, nil
}

func (applicationcredentialHelperFactory) NewAPIObjectAdapter(obj orcObjectPT) adapterI {
	return applicationcredentialAdapter{obj}
}

func (applicationcredentialHelperFactory) NewCreateActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (createResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, orcObject, controller)
}

func (applicationcredentialHelperFactory) NewDeleteActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (deleteResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, orcObject, controller)
}
