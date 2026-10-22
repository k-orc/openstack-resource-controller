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

	targetProjectID, rs := actuator.targetProjectID(ctx, orcObject)
	if needsReschedule, _ := rs.NeedsReschedule(); needsReschedule {
		return nil, false
	}

	listOpts := rbacpolicies.ListOpts{
		ObjectType:   objectTypeNetwork,
		ObjectID:     ptr.Deref(network.Status.ID, ""),
		Action:       rbacpolicies.PolicyAction(resourceSpec.Action),
		TargetTenant: targetProjectID,
	}

	return actuator.osClient.ListRBACPolicy(ctx, listOpts), true
}

// targetProjectID returns spec.resource.targetProjectID, or the ID of the referenced Project.
func (actuator rbacpolicyActuator) targetProjectID(ctx context.Context, obj orcObjectPT) (string, progress.ReconcileStatus) {
	resource := obj.Spec.Resource
	if resource.TargetProjectID != nil {
		return *resource.TargetProjectID, nil
	}
	project, rs := projectDependency.RequireDependency(ctx, actuator.k8sClient, obj, orcv1alpha1.IsAvailable)
	if needsReschedule, _ := rs.NeedsReschedule(); needsReschedule {
		return "", rs
	}
	return ptr.Deref(project.Status.ID, ""), nil
}

func (actuator rbacpolicyActuator) ListOSResourcesForImport(ctx context.Context, obj orcObjectPT, filter filterT) (iter.Seq2[*osResourceT, error], progress.ReconcileStatus) {
	project, rs := dependency.FetchDependency[*orcv1alpha1.Project](
		ctx, actuator.k8sClient, obj.Namespace, filter.TargetProjectRef, "Project",
		orcv1alpha1.IsAvailable,
	)
	if needsReschedule, _ := rs.NeedsReschedule(); needsReschedule {
		return nil, rs
	}

	listOpts := rbacpolicies.ListOpts{
		ObjectType:   objectTypeNetwork,
		Action:       rbacpolicies.PolicyAction(ptr.Deref(filter.Action, "")),
		TargetTenant: ptr.Deref(filter.TargetProjectID, ptr.Deref(project.Status.ID, "")),
	}

	return actuator.osClient.ListRBACPolicy(ctx, listOpts), nil
}

func (actuator rbacpolicyActuator) CreateResource(ctx context.Context, obj orcObjectPT) (*osResourceT, progress.ReconcileStatus) {
	resource := obj.Spec.Resource

	network, reconcileStatus := networkDependency.RequireDependency(
		ctx, actuator.k8sClient, obj, orcv1alpha1.IsAvailable,
	)
	targetProjectID, rs := actuator.targetProjectID(ctx, obj)
	reconcileStatus = reconcileStatus.WithReconcileStatus(rs)
	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return nil, reconcileStatus
	}

	createOpts := rbacpolicies.CreateOpts{
		Action:       rbacpolicies.PolicyAction(resource.Action),
		ObjectType:   objectTypeNetwork,
		ObjectID:     ptr.Deref(network.Status.ID, ""),
		TargetTenant: targetProjectID,
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

// The target project is the only field Neutron allows updating.
func (actuator rbacpolicyActuator) updateResource(ctx context.Context, obj orcObjectPT, osResource *osResourceT) progress.ReconcileStatus {
	log := ctrl.LoggerFrom(ctx)

	targetProjectID, rs := actuator.targetProjectID(ctx, obj)
	if needsReschedule, _ := rs.NeedsReschedule(); needsReschedule {
		return rs
	}

	if osResource.TargetTenant == targetProjectID {
		log.V(logging.Debug).Info("No changes")
		return nil
	}

	updateOpts := rbacpolicies.UpdateOpts{TargetTenant: targetProjectID}
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
