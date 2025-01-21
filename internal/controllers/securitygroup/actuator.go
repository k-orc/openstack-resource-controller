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
	"time"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/attributestags"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/groups"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/rules"
	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/generic"
	osclients "github.com/k-orc/openstack-resource-controller/internal/osclients"
	orcerrors "github.com/k-orc/openstack-resource-controller/internal/util/errors"
	"github.com/k-orc/openstack-resource-controller/internal/util/neutrontags"
	"k8s.io/utils/ptr"
	"k8s.io/utils/set"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type osResourcePT = *groups.SecGroup
type orcObjectPT = *orcv1alpha1.SecurityGroup

type securityGroupActuator struct {
	obj        *orcv1alpha1.SecurityGroup
	osClient   osclients.NetworkClient
	controller generic.ResourceController
}

var _ generic.DeleteResourceActuator[*groups.SecGroup] = securityGroupActuator{}
var _ generic.CreateResourceActuator[*groups.SecGroup] = securityGroupActuator{}

func (actuator securityGroupActuator) GetObject() client.Object {
	return actuator.obj
}

func (actuator securityGroupActuator) GetController() generic.ResourceController {
	return actuator.controller
}

func (actuator securityGroupActuator) GetManagementPolicy() orcv1alpha1.ManagementPolicy {
	return actuator.obj.Spec.ManagementPolicy
}
func (actuator securityGroupActuator) GetManagedOptions() *orcv1alpha1.ManagedOptions {
	return actuator.obj.Spec.ManagedOptions
}

func (actuator securityGroupActuator) GetResourceID(securityGroup *groups.SecGroup) string {
	return securityGroup.ID
}

func (actuator securityGroupActuator) GetStatusID() *string {
	return actuator.obj.Status.ID
}

func (actuator securityGroupActuator) GetOSResourceByStatusID(ctx context.Context) (bool, *groups.SecGroup, error) {
	if actuator.obj.Status.ID == nil {
		return false, nil, nil
	}

	osResource, err := actuator.osClient.GetSecGroup(ctx, *actuator.obj.Status.ID)
	return true, osResource, err
}

func (actuator securityGroupActuator) GetOSResourceBySpec(ctx context.Context) (*groups.SecGroup, error) {
	if actuator.obj.Spec.Resource == nil {
		return nil, nil
	}

	listOpts := listOptsFromCreation(actuator.obj)
	return getResourceFromList(ctx, listOpts, actuator.osClient)
}

func (actuator securityGroupActuator) GetOSResourceByImportID(ctx context.Context) (bool, *groups.SecGroup, error) {
	if actuator.obj.Spec.Import == nil {
		return false, nil, nil
	}
	if actuator.obj.Spec.Import.ID == nil {
		return false, nil, nil
	}

	osResource, err := actuator.osClient.GetSecGroup(ctx, *actuator.obj.Spec.Import.ID)
	return true, osResource, err
}

func (actuator securityGroupActuator) GetOSResourceByImportFilter(ctx context.Context) (bool, *groups.SecGroup, error) {
	if actuator.obj.Spec.Import == nil {
		return false, nil, nil
	}
	if actuator.obj.Spec.Import.Filter == nil {
		return false, nil, nil
	}

	listOpts := listOptsFromImportFilter(actuator.obj.Spec.Import.Filter)
	osResource, err := getResourceFromList(ctx, listOpts, actuator.osClient)
	return true, osResource, err
}

func (actuator securityGroupActuator) CreateResource(ctx context.Context) ([]generic.WaitingOnEvent, *groups.SecGroup, error) {
	resource := actuator.obj.Spec.Resource
	if resource == nil {
		// Should have been caught by API validation
		return nil, nil, orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Creation requested, but spec.resource is not set")
	}

	createOpts := groups.CreateOpts{
		Name:        string(getResourceName(actuator.obj)),
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

func (actuator securityGroupActuator) DeleteResource(ctx context.Context, osResource *groups.SecGroup) ([]generic.WaitingOnEvent, error) {
	return nil, actuator.osClient.DeleteSecGroup(ctx, osResource.ID)
}

// getResourceName returns the name of the OpenStack resource we should use.
func getResourceName(orcObject *orcv1alpha1.SecurityGroup) orcv1alpha1.OpenStackName {
	if orcObject.Spec.Resource.Name != nil {
		return *orcObject.Spec.Resource.Name
	}
	return orcv1alpha1.OpenStackName(orcObject.Name)
}

func listOptsFromImportFilter(filter *orcv1alpha1.SecurityGroupFilter) groups.ListOpts {
	listOpts := groups.ListOpts{
		Name:        string(ptr.Deref(filter.Name, "")),
		Description: string(ptr.Deref(filter.Description, "")),
		Tags:        neutrontags.Join(filter.FilterByNeutronTags.Tags),
		TagsAny:     neutrontags.Join(filter.FilterByNeutronTags.TagsAny),
		NotTags:     neutrontags.Join(filter.FilterByNeutronTags.NotTags),
		NotTagsAny:  neutrontags.Join(filter.FilterByNeutronTags.NotTagsAny),
	}

	return listOpts
}

// listOptsFromCreation returns a listOpts which will return the OpenStack
// resource which would have been created from the current spec and hopefully no
// other. Its purpose is to automatically adopt a resource that we created but
// failed to write to status.id.
func listOptsFromCreation(osResource *orcv1alpha1.SecurityGroup) groups.ListOpts {
	return groups.ListOpts{Name: string(getResourceName(osResource))}
}

func getResourceFromList(ctx context.Context, listOpts groups.ListOpts, networkClient osclients.NetworkClient) (*groups.SecGroup, error) {
	osResources, err := networkClient.ListSecGroup(ctx, listOpts)
	if err != nil {
		return nil, err
	}

	if len(osResources) == 1 {
		return &osResources[0], nil
	}

	// No resource found
	if len(osResources) == 0 {
		return nil, nil
	}

	// Multiple resources found
	return nil, orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, fmt.Sprintf("Expected to find exactly one OpenStack resource to import. Found %d", len(osResources)))
}

var _ generic.UpdateResourceActuator[orcObjectPT, osResourcePT] = securityGroupActuator{}

type resourceUpdater = generic.ResourceUpdater[orcObjectPT, osResourcePT]

func (actuator securityGroupActuator) GetResourceUpdaters(ctx context.Context, orcObject orcObjectPT, osResource osResourcePT, controller generic.ResourceController) ([]resourceUpdater, error) {
	return []resourceUpdater{
		actuator.updateTags,
		actuator.updateRules,
	}, nil
}

func (actuator securityGroupActuator) updateTags(ctx context.Context, orcObject orcObjectPT, osResource osResourcePT) ([]generic.WaitingOnEvent, orcObjectPT, osResourcePT, error) {
	resourceTagSet := set.New[string](osResource.Tags...)
	objectTagSet := set.New[string]()
	for i := range orcObject.Spec.Resource.Tags {
		objectTagSet.Insert(string(orcObject.Spec.Resource.Tags[i]))
	}
	var err error
	if !objectTagSet.Equal(resourceTagSet) {
		opts := attributestags.ReplaceAllOpts{Tags: objectTagSet.SortedList()}
		_, err = actuator.osClient.ReplaceAllAttributesTags(ctx, "security-groups", osResource.ID, &opts)
	}
	return nil, orcObject, osResource, err
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

func (actuator securityGroupActuator) updateRules(ctx context.Context, orcObject orcObjectPT, osResource osResourcePT) ([]generic.WaitingOnEvent, orcObjectPT, osResourcePT, error) {
	resource := orcObject.Spec.Resource
	if resource == nil {
		return nil, orcObject, osResource, nil
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

	var waitEvents []generic.WaitingOnEvent

	// If we added or removed any rules above, schedule another reconcile so we can observe the updated security group
	if len(ruleCreateOpts) > 0 || len(deleteRuleIDs) > 0 {
		waitEvents = []generic.WaitingOnEvent{generic.WaitingOnOpenStackUpdate(time.Second)}
	}

	return waitEvents, orcObject, osResource, err
}

type securityGroupActuatorFactory struct{}

var _ generic.ActuatorFactory[orcObjectPT, osResourcePT] = securityGroupActuatorFactory{}

func (securityGroupActuatorFactory) NewCreateActuator(ctx context.Context, orcObject orcObjectPT, controller generic.ResourceController) ([]generic.WaitingOnEvent, generic.CreateResourceActuator[osResourcePT], error) {
	actuator, err := newActuator(ctx, orcObject, controller)
	return nil, actuator, err
}

func (securityGroupActuatorFactory) NewDeleteActuator(ctx context.Context, orcObject orcObjectPT, controller generic.ResourceController) ([]generic.WaitingOnEvent, generic.DeleteResourceActuator[osResourcePT], error) {
	actuator, err := newActuator(ctx, orcObject, controller)
	return nil, actuator, err
}

func newActuator(ctx context.Context, orcObject *orcv1alpha1.SecurityGroup, controller generic.ResourceController) (securityGroupActuator, error) {
	log := ctrl.LoggerFrom(ctx)

	clientScope, err := controller.GetScopeFactory().NewClientScopeFromObject(ctx, controller.GetK8sClient(), log, orcObject)
	if err != nil {
		return securityGroupActuator{}, err
	}
	osClient, err := clientScope.NewNetworkClient()
	if err != nil {
		return securityGroupActuator{}, err
	}

	return securityGroupActuator{
		obj:        orcObject,
		osClient:   osClient,
		controller: controller,
	}, nil
}
