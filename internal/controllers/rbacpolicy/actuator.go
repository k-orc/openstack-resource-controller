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

package rbacpolicy

import (
	"context"
	"iter"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/rbacpolicies"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v3/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v3/internal/controllers/generic/progress"
	"github.com/k-orc/openstack-resource-controller/v3/internal/logging"
	"github.com/k-orc/openstack-resource-controller/v3/internal/osclients"
	"github.com/k-orc/openstack-resource-controller/v3/internal/util/dependency"
	orcerrors "github.com/k-orc/openstack-resource-controller/v3/internal/util/errors"
)

// Neutron's RBAC policy API only supports network sharing in this initial implementation -
// object_type is always "network" here. See rbacpolicy_types.go's own comment on
// RBACPolicyResourceSpec for why qos-policy/security-group aren't included yet.
const objectTypeNetwork = "network"

// OpenStack resource types
type (
	osResourceT = rbacpolicies.RBACPolicy

	createResourceActuator = interfaces.CreateResourceActuator[orcObjectPT, orcObjectT, filterT, osResourceT]
	deleteResourceActuator = interfaces.DeleteResourceActuator[orcObjectPT, orcObjectT, osResourceT]
	resourceReconciler     = interfaces.ResourceReconciler[orcObjectPT, osResourceT]
	helperFactory          = interfaces.ResourceHelperFactory[orcObjectPT, orcObjectT, resourceSpecT, filterT, osResourceT]
)

type rbacpolicyActuator struct {
	// RBAC policies are a Neutron (network service) resource - reuses the shared
	// NetworkClient, same as SecurityGroup/Trunk/Router/etc., rather than a dedicated
	// per-resource client. See osclients/networking.go's NetworkClient interface.
	osClient  osclients.NetworkClient
	k8sClient client.Client
}

var _ createResourceActuator = rbacpolicyActuator{}
var _ deleteResourceActuator = rbacpolicyActuator{}

func (rbacpolicyActuator) GetResourceID(osResource *osResourceT) string {
	return osResource.ID
}

func (actuator rbacpolicyActuator) GetOSResourceByID(ctx context.Context, id string) (*osResourceT, progress.ReconcileStatus) {
	resource, err := actuator.osClient.GetRBACPolicy(ctx, id)
	if err != nil {
		return nil, progress.WrapError(err)
	}
	return resource, nil
}

func (actuator rbacpolicyActuator) ListOSResourcesForAdoption(ctx context.Context, orcObject orcObjectPT) (iter.Seq2[*osResourceT, error], bool) {
	resourceSpec := orcObject.Spec.Resource
	if resourceSpec == nil {
		return nil, false
	}

	network, rs := dependency.FetchDependency[*orcv1alpha1.Network](
		ctx, actuator.k8sClient, orcObject.Namespace, &resourceSpec.NetworkRef, "Network",
		orcv1alpha1.IsAvailable,
	)
	if needsReschedule, _ := rs.NeedsReschedule(); needsReschedule {
		return nil, false
	}

	// Matches the full declared spec, same reasoning as every other ListOSResourcesForAdoption
	// in this codebase: adoption must not match a policy that only partially agrees with what
	// we'd otherwise create, or a later reconcile would immediately try to "fix" a field this
	// API doesn't even support updating (Action/ObjectID - see rbacpolicy_types.go).
	listOpts := rbacpolicies.ListOpts{
		ObjectType:   objectTypeNetwork,
		ObjectID:     ptr.Deref(network.Status.ID, ""),
		Action:       rbacpolicies.PolicyAction(resourceSpec.Action),
		TargetTenant: resourceSpec.TargetProjectID,
	}

	return actuator.osClient.ListRBACPolicy(ctx, listOpts), true
}

func (actuator rbacpolicyActuator) ListOSResourcesForImport(ctx context.Context, obj orcObjectPT, filter filterT) (iter.Seq2[*osResourceT, error], progress.ReconcileStatus) {
	listOpts := rbacpolicies.ListOpts{
		ObjectType: objectTypeNetwork,
	}
	if filter.Action != nil {
		listOpts.Action = rbacpolicies.PolicyAction(*filter.Action)
	}
	if filter.TargetProjectID != nil {
		listOpts.TargetTenant = *filter.TargetProjectID
	}

	return actuator.osClient.ListRBACPolicy(ctx, listOpts), nil
}

func (actuator rbacpolicyActuator) CreateResource(ctx context.Context, obj orcObjectPT) (*osResourceT, progress.ReconcileStatus) {
	resource := obj.Spec.Resource

	if resource == nil {
		// Should have been caught by API validation
		return nil, progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Creation requested, but spec.resource is not set"))
	}

	network, reconcileStatus := networkDependency.RequireDependency(
		ctx, actuator.k8sClient, obj, orcv1alpha1.IsAvailable,
	)
	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return nil, reconcileStatus
	}

	createOpts := rbacpolicies.CreateOpts{
		Action:       rbacpolicies.PolicyAction(resource.Action),
		ObjectType:   objectTypeNetwork,
		ObjectID:     ptr.Deref(network.Status.ID, ""),
		TargetTenant: resource.TargetProjectID,
	}

	osResource, err := actuator.osClient.CreateRBACPolicy(ctx, createOpts)
	if err != nil {
		if !orcerrors.IsRetryable(err) {
			err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration creating resource: "+err.Error(), err)
		}
		return nil, progress.WrapError(err)
	}

	return osResource, nil
}

func (actuator rbacpolicyActuator) DeleteResource(ctx context.Context, _ orcObjectPT, resource *osResourceT) progress.ReconcileStatus {
	return progress.WrapError(actuator.osClient.DeleteRBACPolicy(ctx, resource.ID))
}

// updateResource handles targetProjectID changes - the only field Neutron's RBAC policy
// UpdateOpts actually supports (gophercloud's UpdateOpts has just TargetTenant, nothing else).
// Action/networkRef are immutable (see rbacpolicy_types.go) precisely because there's no API to
// update them - changing either means delete+recreate, which K-ORC's generic reconciler already
// handles correctly via the immutability CEL validation rejecting the spec change outright rather
// than silently no-op'ing or erroring at the OpenStack API layer.
func (actuator rbacpolicyActuator) updateResource(ctx context.Context, obj orcObjectPT, osResource *osResourceT) progress.ReconcileStatus {
	log := ctrl.LoggerFrom(ctx)
	resource := obj.Spec.Resource
	if resource == nil {
		// Should have been caught by API validation
		return progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Update requested, but spec.resource is not set"))
	}

	if osResource.TargetTenant == resource.TargetProjectID {
		log.V(logging.Debug).Info("No changes")
		return nil
	}

	updateOpts := rbacpolicies.UpdateOpts{TargetTenant: resource.TargetProjectID}
	_, err := actuator.osClient.UpdateRBACPolicy(ctx, osResource.ID, updateOpts)
	if err != nil {
		if !orcerrors.IsRetryable(err) {
			err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration updating resource: "+err.Error(), err)
		}
		return progress.WrapError(err)
	}

	return progress.NeedsRefresh()
}

func (actuator rbacpolicyActuator) GetResourceReconcilers(ctx context.Context, orcObject orcObjectPT, osResource *osResourceT, controller interfaces.ResourceController) ([]resourceReconciler, progress.ReconcileStatus) {
	return []resourceReconciler{
		actuator.updateResource,
	}, nil
}

type rbacpolicyHelperFactory struct{}

var _ helperFactory = rbacpolicyHelperFactory{}

func newActuator(ctx context.Context, orcObject *orcv1alpha1.RBACPolicy, controller interfaces.ResourceController) (rbacpolicyActuator, progress.ReconcileStatus) {
	log := ctrl.LoggerFrom(ctx)

	// Ensure credential secrets exist and have our finalizer
	_, reconcileStatus := credentialsDependency.RequireDependencies(ctx, controller.GetK8sClient(), orcObject, func(*corev1.Secret) bool { return true })
	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return rbacpolicyActuator{}, reconcileStatus
	}

	clientScope, err := controller.GetScopeFactory().NewClientScopeFromObject(ctx, controller.GetK8sClient(), log, orcObject)
	if err != nil {
		return rbacpolicyActuator{}, progress.WrapError(err)
	}
	osClient, err := clientScope.NewNetworkClient()
	if err != nil {
		return rbacpolicyActuator{}, progress.WrapError(err)
	}

	return rbacpolicyActuator{
		osClient:  osClient,
		k8sClient: controller.GetK8sClient(),
	}, nil
}

func (rbacpolicyHelperFactory) NewAPIObjectAdapter(obj orcObjectPT) adapterI {
	return rbacpolicyAdapter{obj}
}

func (rbacpolicyHelperFactory) NewCreateActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (createResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, orcObject, controller)
}

func (rbacpolicyHelperFactory) NewDeleteActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (deleteResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, orcObject, controller)
}
