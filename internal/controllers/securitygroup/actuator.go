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
	"errors"
	"fmt"
	"iter"
	"time"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/groups"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/rules"
	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/progress"
	osclients "github.com/k-orc/openstack-resource-controller/v2/internal/osclients"
	orcerrors "github.com/k-orc/openstack-resource-controller/v2/internal/util/errors"
	"github.com/k-orc/openstack-resource-controller/v2/internal/util/neutrontags"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	"k8s.io/utils/set"
	ctrl "sigs.k8s.io/controller-runtime"
)

type (
	osResourceT = groups.SecGroup

	createResourceActuator    = interfaces.CreateResourceActuator[orcObjectPT, orcObjectT, filterT, osResourceT]
	deleteResourceActuator    = interfaces.DeleteResourceActuator[orcObjectPT, orcObjectT, osResourceT]
	reconcileResourceActuator = interfaces.ReconcileResourceActuator[orcObjectPT, osResourceT]
	resourceReconciler        = interfaces.ResourceReconciler[orcObjectPT, osResourceT]
	helperFactory             = interfaces.ResourceHelperFactory[orcObjectPT, orcObjectT, resourceSpecT, filterT, osResourceT]
	securityGroupIterator     = iter.Seq2[*osResourceT, error]
)

type securityGroupActuator struct {
	osClient osclients.NetworkClient
}

var _ createResourceActuator = securityGroupActuator{}
var _ deleteResourceActuator = securityGroupActuator{}

func (actuator securityGroupActuator) GetResourceID(securityGroup *groups.SecGroup) string {
	return securityGroup.ID
}

func (actuator securityGroupActuator) GetOSResourceByID(ctx context.Context, id string) (*groups.SecGroup, error) {
	return actuator.osClient.GetSecGroup(ctx, id)
}

func (actuator securityGroupActuator) ListOSResourcesForAdoption(ctx context.Context, obj *orcv1alpha1.SecurityGroup) (securityGroupIterator, bool) {
	if obj.Spec.Resource == nil {
		return nil, false
	}

	listOpts := groups.ListOpts{Name: string(getResourceName(obj))}
	return actuator.osClient.ListSecGroup(ctx, listOpts), true
}

func (actuator securityGroupActuator) ListOSResourcesForImport(ctx context.Context, obj orcObjectPT, filter filterT) ([]progress.ProgressStatus, iter.Seq2[*osResourceT, error], error) {

	listOpts := groups.ListOpts{
		Name:        string(ptr.Deref(filter.Name, "")),
		Description: string(ptr.Deref(filter.Description, "")),
		Tags:        neutrontags.Join(filter.Tags),
		TagsAny:     neutrontags.Join(filter.TagsAny),
		NotTags:     neutrontags.Join(filter.NotTags),
		NotTagsAny:  neutrontags.Join(filter.NotTagsAny),
	}
	return nil, actuator.osClient.ListSecGroup(ctx, listOpts), nil
}

func (actuator securityGroupActuator) CreateResource(ctx context.Context, obj *orcv1alpha1.SecurityGroup) ([]progress.ProgressStatus, *groups.SecGroup, error) {
	resource := obj.Spec.Resource
	if resource == nil {
		// Should have been caught by API validation
		return nil, nil, orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Creation requested, but spec.resource is not set")
	}

	createOpts := groups.CreateOpts{
		Name:        string(getResourceName(obj)),
		Description: string(ptr.Deref(resource.Description, "")),
		Stateful:    resource.Stateful,
	}

	// FIXME(mandre) The security group inherits the default security group
	// rules. This could be a problem when we implement `update` if ORC
	// does not takes these rules into account.
	osResource, err := actuator.osClient.CreateSecGroup(ctx, &createOpts)
	if err != nil {
		// We should require the spec to be updated before retrying a create which returned a conflict
		if orcerrors.IsConflict(err) {
			err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration creating resource: "+err.Error(), err)
		}
		return nil, nil, err
	}

	return nil, osResource, nil
}

func (actuator securityGroupActuator) DeleteResource(ctx context.Context, _ *orcv1alpha1.SecurityGroup, osResource *groups.SecGroup) ([]progress.ProgressStatus, error) {
	return nil, actuator.osClient.DeleteSecGroup(ctx, osResource.ID)
}

var _ reconcileResourceActuator = securityGroupActuator{}

func (actuator securityGroupActuator) GetResourceReconcilers(ctx context.Context, orcObject orcObjectPT, osResource *osResourceT, controller interfaces.ResourceController) ([]resourceReconciler, error) {
	return []resourceReconciler{
		neutrontags.ReconcileTags[orcObjectPT, osResourceT](actuator.osClient, "security-groups", osResource.ID, orcObject.Spec.Resource.Tags, osResource.Tags),
		actuator.updateRules,
	}, nil
}

func rulesMatch(orcRule *orcv1alpha1.SecurityGroupRule, osRule *rules.SecGroupRule) bool {
	// Don't compare description if it's not set in the spec
	if orcRule.Description != nil && string(*orcRule.Description) != osRule.Description {
		return false
	}

	// Don't compare direction if it's not set in the spec.
	// TODO check what we get from neutron in this field if we didn't set it in the spec
	if orcRule.Direction != nil && string(*orcRule.Direction) != osRule.Direction {
		return false
	}

	// Always compare RemoteIPPrefix. If unset in ORC it must be empty in OpenStack
	if string(ptr.Deref(orcRule.RemoteIPPrefix, "")) != osRule.RemoteIPPrefix {
		return false
	}

	// Always compare protocol. Unset == "" from gophercloud
	if string(ptr.Deref(orcRule.Protocol, "")) != osRule.Protocol {
		return false
	}

	if string(orcRule.Ethertype) != osRule.EtherType {
		return false
	}

	if orcRule.PortRange == nil {
		if osRule.PortRangeMin != 0 || osRule.PortRangeMax != 0 {
			return false
		}
	} else {
		if int(orcRule.PortRange.Min) != osRule.PortRangeMin || int(orcRule.PortRange.Max) != osRule.PortRangeMax {
			return false
		}
	}

	return true
}

func (actuator securityGroupActuator) updateRules(ctx context.Context, orcObject orcObjectPT, osResource *osResourceT) ([]progress.ProgressStatus, error) {
	resource := orcObject.Spec.Resource
	if resource == nil {
		return nil, nil
	}

	matchedRuleIDs := set.New[string]()
	allRuleIDS := set.New[string]()
	var createRules []*orcv1alpha1.SecurityGroupRule

orcRules:
	for i := range resource.Rules {
		orcRule := &resource.Rules[i]
		for j := range osResource.Rules {
			osRule := &osResource.Rules[j]

			if rulesMatch(orcRule, osRule) {
				matchedRuleIDs.Insert(osRule.ID)
				continue orcRules
			}
		}
		createRules = append(createRules, orcRule)
	}

	for i := range osResource.Rules {
		allRuleIDS.Insert(osResource.Rules[i].ID)
	}
	deleteRuleIDs := allRuleIDS.Difference(matchedRuleIDs)

	ruleCreateOpts := make([]rules.CreateOpts, len(createRules))
	for i := range createRules {
		ruleCreateOpts[i] = rules.CreateOpts{
			SecGroupID:     osResource.ID,
			Description:    string(ptr.Deref(createRules[i].Description, "")),
			Direction:      rules.RuleDirection(ptr.Deref(createRules[i].Direction, "")),
			RemoteIPPrefix: string(ptr.Deref(createRules[i].RemoteIPPrefix, "")),
			Protocol:       rules.RuleProtocol(ptr.Deref(createRules[i].Protocol, "")),
			EtherType:      rules.RuleEtherType(createRules[i].Ethertype),
		}
		if createRules[i].PortRange != nil {
			ruleCreateOpts[i].PortRangeMin = int(resource.Rules[i].PortRange.Min)
			ruleCreateOpts[i].PortRangeMax = int(resource.Rules[i].PortRange.Max)
		}
	}

	var err error
	if len(ruleCreateOpts) > 0 {
		if _, createErr := actuator.osClient.CreateSecGroupRules(ctx, ruleCreateOpts); createErr != nil {
			// We should require the spec to be updated before retrying a create which returned a conflict
			if orcerrors.IsRetryable(createErr) {
				createErr = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration creating resource: "+createErr.Error(), createErr)
			} else {
				createErr = fmt.Errorf("creating security group rules: %w", createErr)
			}

			err = errors.Join(err, createErr)
		}
	}

	for _, id := range deleteRuleIDs.UnsortedList() {
		if deleteErr := actuator.osClient.DeleteSecGroupRule(ctx, id); deleteErr != nil {
			err = errors.Join(err, fmt.Errorf("deleting security group rule %s: %w", id, deleteErr))
		}
	}

	var waitEvents []progress.ProgressStatus

	// If we added or removed any rules above, schedule another reconcile so we can observe the updated security group
	if len(ruleCreateOpts) > 0 || len(deleteRuleIDs) > 0 {
		waitEvents = []progress.ProgressStatus{progress.WaitingOnOpenStackUpdate(time.Second)}
	}

	return waitEvents, err
}

type securityGroupHelperFactory struct{}

var _ helperFactory = securityGroupHelperFactory{}

func (securityGroupHelperFactory) NewAPIObjectAdapter(obj orcObjectPT) adapterI {
	return securitygroupAdapter{obj}
}

func (securityGroupHelperFactory) NewCreateActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) ([]progress.ProgressStatus, createResourceActuator, error) {
	actuator, progressStatus, err := newActuator(ctx, orcObject, controller)
	return progressStatus, actuator, err
}

func (securityGroupHelperFactory) NewDeleteActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) ([]progress.ProgressStatus, deleteResourceActuator, error) {
	actuator, progressStatus, err := newActuator(ctx, orcObject, controller)
	return progressStatus, actuator, err
}

func newActuator(ctx context.Context, orcObject *orcv1alpha1.SecurityGroup, controller interfaces.ResourceController) (securityGroupActuator, []progress.ProgressStatus, error) {
	log := ctrl.LoggerFrom(ctx)

	// Ensure credential secrets exist and have our finalizer
	_, progressStatus, err := credentialsDependency.GetDependencies(ctx, controller.GetK8sClient(), orcObject, func(*corev1.Secret) bool { return true })
	if len(progressStatus) > 0 || err != nil {
		return securityGroupActuator{}, progressStatus, err
	}

	clientScope, err := controller.GetScopeFactory().NewClientScopeFromObject(ctx, controller.GetK8sClient(), log, orcObject)
	if err != nil {
		return securityGroupActuator{}, nil, err
	}
	osClient, err := clientScope.NewNetworkClient()
	if err != nil {
		return securityGroupActuator{}, nil, err
	}

	return securityGroupActuator{
		osClient: osClient,
	}, nil, nil
}
