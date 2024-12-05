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

package securitygroup

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/groups"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/common"
	"github.com/k-orc/openstack-resource-controller/internal/util/applyconfigs"
	orcapplyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/pkg/clients/applyconfiguration/api/v1alpha1"
)

// setStatusID sets status.ID in its own SSA transaction.
func (r *orcSecurityGroupReconciler) setStatusID(ctx context.Context, obj client.Object, id string) error {
	applyConfig := orcapplyconfigv1alpha1.SecurityGroup(obj.GetName(), obj.GetNamespace()).
		WithUID(obj.GetUID()).
		WithStatus(orcapplyconfigv1alpha1.SecurityGroupStatus().
			WithID(id))

	return r.client.Status().Patch(ctx, obj, applyconfigs.Patch(types.ApplyPatchType, applyConfig), client.ForceOwnership, ssaFieldOwner(SSAIDTxn))
}

type updateStatusOpts struct {
	resource        *groups.SecGroup
	progressMessage *string
	err             error
}

type updateStatusOpt func(*updateStatusOpts)

func withResource(resource *groups.SecGroup) updateStatusOpt {
	return func(opts *updateStatusOpts) {
		opts.resource = resource
	}
}

func withError(err error) updateStatusOpt {
	return func(opts *updateStatusOpts) {
		opts.err = err
	}
}

// withProgressMessage sets a custom progressing message if and only if the reconcile is progressing.
func withProgressMessage(message string) updateStatusOpt {
	return func(opts *updateStatusOpts) {
		opts.progressMessage = &message
	}
}

func getOSResourceStatus(_ logr.Logger, osResource *groups.SecGroup) *orcapplyconfigv1alpha1.SecurityGroupResourceStatusApplyConfiguration {
	securitygroupResourceStatus := (&orcapplyconfigv1alpha1.SecurityGroupResourceStatusApplyConfiguration{}).
		WithName(osResource.Name).
		WithDescription(osResource.Description).
		WithProjectID(osResource.ProjectID).
		WithTags(osResource.Tags...).
		WithStateful(osResource.Stateful).
		WithCreatedAt(metav1.NewTime(osResource.CreatedAt)).
		WithUpdatedAt(metav1.NewTime(osResource.UpdatedAt))

	if len(osResource.Rules) > 0 {
		rules := make([]*orcapplyconfigv1alpha1.SecurityGroupRuleStatusApplyConfiguration, len(osResource.Rules))
		for i := range osResource.Rules {
			rules[i] = orcapplyconfigv1alpha1.SecurityGroupRuleStatus().
				WithID(osResource.Rules[i].ID).
				WithDescription(osResource.Rules[i].Description).
				WithDirection(osResource.Rules[i].Direction).
				WithRemoteGroupID(osResource.Rules[i].RemoteGroupID).
				WithRemoteIPPrefix(osResource.Rules[i].RemoteIPPrefix).
				WithProtocol(osResource.Rules[i].Protocol).
				WithEthertype(osResource.Rules[i].EtherType).
				WithPortRangeMin(osResource.Rules[i].PortRangeMin).
				WithPortRangeMax(osResource.Rules[i].PortRangeMax)
		}
		securitygroupResourceStatus.WithRules(rules...)
	}

	return securitygroupResourceStatus
}

// createStatusUpdate computes a complete status update based on the given
// observed state. This is separated from updateStatus to facilitate unit
// testing, as the version of k8s we currently import does not support patch
// apply in the fake client.
// Needs: https://github.com/kubernetes/kubernetes/pull/125560
func createStatusUpdate(ctx context.Context, orcSecurityGroup *orcv1alpha1.SecurityGroup, now metav1.Time, opts ...updateStatusOpt) *orcapplyconfigv1alpha1.SecurityGroupApplyConfiguration {
	log := ctrl.LoggerFrom(ctx)

	statusOpts := updateStatusOpts{}
	for i := range opts {
		opts[i](&statusOpts)
	}

	osResource := statusOpts.resource

	applyConfigStatus := orcapplyconfigv1alpha1.SecurityGroupStatus()
	applyConfig := orcapplyconfigv1alpha1.SecurityGroup(orcSecurityGroup.Name, orcSecurityGroup.Namespace).WithStatus(applyConfigStatus)

	if osResource != nil {
		resourceStatus := getOSResourceStatus(log, osResource)
		applyConfigStatus.WithResource(resourceStatus)
	}

	// FIXME(mandre) Don't return available unless all security group rules are created
	available := osResource != nil
	common.SetCommonConditions(orcSecurityGroup, applyConfigStatus, available, available, statusOpts.progressMessage, statusOpts.err, now)

	return applyConfig
}

// updateStatus computes a complete status based on the given observed state and writes it to status.
func (r *orcSecurityGroupReconciler) updateStatus(ctx context.Context, orcObject *orcv1alpha1.SecurityGroup, opts ...updateStatusOpt) error {
	now := metav1.NewTime(time.Now())

	statusUpdate := createStatusUpdate(ctx, orcObject, now, opts...)

	return r.client.Status().Patch(ctx, orcObject, applyconfigs.Patch(types.ApplyPatchType, statusUpdate), client.ForceOwnership, ssaFieldOwner(SSAStatusTxn))
}
