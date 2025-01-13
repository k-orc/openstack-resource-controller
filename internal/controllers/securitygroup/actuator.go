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
	"fmt"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/groups"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/rules"
	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/generic"
	osclients "github.com/k-orc/openstack-resource-controller/internal/osclients"
	orcerrors "github.com/k-orc/openstack-resource-controller/internal/util/errors"
	"github.com/k-orc/openstack-resource-controller/internal/util/neutrontags"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type securityGroupActuator struct {
	*orcv1alpha1.SecurityGroup
	osClient   osclients.NetworkClient
	controller generic.ResourceController
}

func newActuator(ctx context.Context, controller generic.ResourceController, orcObject *orcv1alpha1.SecurityGroup) (securityGroupActuator, error) {
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
		SecurityGroup: orcObject,
		osClient:      osClient,
		controller:    controller,
	}, nil
}

var _ generic.DeleteResourceActuator[*groups.SecGroup] = securityGroupActuator{}
var _ generic.CreateResourceActuator[*groups.SecGroup] = securityGroupActuator{}

func (obj securityGroupActuator) GetObject() client.Object {
	return obj.SecurityGroup
}

func (obj securityGroupActuator) GetController() generic.ResourceController {
	return obj.controller
}

func (obj securityGroupActuator) GetManagementPolicy() orcv1alpha1.ManagementPolicy {
	return obj.Spec.ManagementPolicy
}
func (obj securityGroupActuator) GetManagedOptions() *orcv1alpha1.ManagedOptions {
	return obj.Spec.ManagedOptions
}

func (obj securityGroupActuator) GetResourceID(securityGroup *groups.SecGroup) string {
	return securityGroup.ID
}

func (obj securityGroupActuator) GetOSResourceByStatusID(ctx context.Context) (bool, *groups.SecGroup, error) {
	if obj.Status.ID == nil {
		return false, nil, nil
	}

	osResource, err := obj.osClient.GetSecGroup(ctx, *obj.Status.ID)
	return true, osResource, err
}

func (obj securityGroupActuator) GetOSResourceBySpec(ctx context.Context) (*groups.SecGroup, error) {
	if obj.Spec.Resource == nil {
		return nil, nil
	}

	listOpts := listOptsFromCreation(obj.SecurityGroup)
	return getResourceFromList(ctx, listOpts, obj.osClient)
}

func (obj securityGroupActuator) GetOSResourceByImportID(ctx context.Context) (bool, *groups.SecGroup, error) {
	if obj.Spec.Import == nil {
		return false, nil, nil
	}
	if obj.Spec.Import.ID == nil {
		return false, nil, nil
	}

	osResource, err := obj.osClient.GetSecGroup(ctx, *obj.Spec.Import.ID)
	return true, osResource, err
}

func (obj securityGroupActuator) GetOSResourceByImportFilter(ctx context.Context) (bool, *groups.SecGroup, error) {
	if obj.Spec.Import == nil {
		return false, nil, nil
	}
	if obj.Spec.Import.Filter == nil {
		return false, nil, nil
	}

	listOpts := listOptsFromImportFilter(obj.Spec.Import.Filter)
	osResource, err := getResourceFromList(ctx, listOpts, obj.osClient)
	return true, osResource, err
}

func (obj securityGroupActuator) CreateResource(ctx context.Context) ([]generic.WaitingOnEvent, *groups.SecGroup, error) {
	resource := obj.Spec.Resource
	if resource == nil {
		// Should have been caught by API validation
		return nil, nil, orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Creation requested, but spec.resource is not set")
	}

	createOpts := groups.CreateOpts{
		Name:        string(getResourceName(obj.SecurityGroup)),
		Description: string(ptr.Deref(resource.Description, "")),
		Stateful:    resource.Stateful,
	}

	// FIXME(mandre) The security group inherits the default security group
	// rules. This could be a problem when we implement `update` if ORC
	// does not takes these rules into account.
	osResource, err := obj.osClient.CreateSecGroup(ctx, &createOpts)
	if err != nil {
		// We should require the spec to be updated before retrying a create which returned a conflict
		if orcerrors.IsConflict(err) {
			err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration creating resource: "+err.Error(), err)
		}
		return nil, nil, err
	}

	ruleCreateOpts := make([]rules.CreateOpts, len(resource.Rules))

	for i := range resource.Rules {
		ruleCreateOpts[i] = rules.CreateOpts{
			SecGroupID:     osResource.ID,
			Description:    string(ptr.Deref(resource.Rules[i].Description, "")),
			Direction:      rules.RuleDirection(ptr.Deref(resource.Rules[i].Direction, "")),
			RemoteIPPrefix: string(ptr.Deref(resource.Rules[i].RemoteIPPrefix, "")),
			Protocol:       rules.RuleProtocol(ptr.Deref(resource.Rules[i].Protocol, "")),
			EtherType:      rules.RuleEtherType(resource.Rules[i].Ethertype),
		}
		if resource.Rules[i].PortRange != nil {
			ruleCreateOpts[i].PortRangeMin = int(resource.Rules[i].PortRange.Min)
			ruleCreateOpts[i].PortRangeMax = int(resource.Rules[i].PortRange.Max)
		}
	}

	if _, err := obj.osClient.CreateSecGroupRules(ctx, ruleCreateOpts); err != nil {
		// We should require the spec to be updated before retrying a create which returned a conflict
		if orcerrors.IsConflict(err) {
			err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration creating resource: "+err.Error(), err)
		}
		return nil, nil, err
	}

	return nil, osResource, nil
}

func (obj securityGroupActuator) DeleteResource(ctx context.Context, osResource *groups.SecGroup) ([]generic.WaitingOnEvent, error) {
	return nil, obj.osClient.DeleteSecGroup(ctx, osResource.ID)
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
