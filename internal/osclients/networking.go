/*
Copyright 2021 The ORC Authors.

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

package osclients

import (
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/attributestags"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/floatingips"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/routers"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/groups"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/rules"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/trunks"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/subnets"
	"github.com/gophercloud/gophercloud/v2/pagination"
	"github.com/gophercloud/utils/v2/openstack/clientconfig"
)

type NetworkClient interface {
	ListFloatingIP(opts floatingips.ListOptsBuilder) ([]floatingips.FloatingIP, error)
	CreateFloatingIP(opts floatingips.CreateOptsBuilder) (*floatingips.FloatingIP, error)
	DeleteFloatingIP(id string) error
	GetFloatingIP(id string) (*floatingips.FloatingIP, error)
	UpdateFloatingIP(id string, opts floatingips.UpdateOptsBuilder) (*floatingips.FloatingIP, error)

	ListPort(ctx context.Context, opts ports.ListOptsBuilder) ([]ports.Port, error)
	CreatePort(ctx context.Context, opts ports.CreateOptsBuilder) (*ports.Port, error)
	DeletePort(ctx context.Context, id string) error
	GetPort(ctx context.Context, id string) (*ports.Port, error)
	UpdatePort(id string, opts ports.UpdateOptsBuilder) (*ports.Port, error)

	ListTrunk(opts trunks.ListOptsBuilder) ([]trunks.Trunk, error)
	CreateTrunk(opts trunks.CreateOptsBuilder) (*trunks.Trunk, error)
	DeleteTrunk(id string) error

	ListTrunkSubports(trunkID string) ([]trunks.Subport, error)
	RemoveSubports(id string, opts trunks.RemoveSubportsOpts) error

	ListRouter(ctx context.Context, opts routers.ListOpts) ([]routers.Router, error)
	CreateRouter(ctx context.Context, opts routers.CreateOptsBuilder) (*routers.Router, error)
	DeleteRouter(ctx context.Context, id string) error
	GetRouter(ctx context.Context, id string) (*routers.Router, error)
	UpdateRouter(ctx context.Context, id string, opts routers.UpdateOptsBuilder) (*routers.Router, error)
	AddRouterInterface(ctx context.Context, id string, opts routers.AddInterfaceOptsBuilder) (*routers.InterfaceInfo, error)
	RemoveRouterInterface(ctx context.Context, id string, opts routers.RemoveInterfaceOptsBuilder) (*routers.InterfaceInfo, error)

	ListSecGroup(ctx context.Context, opts groups.ListOpts) ([]groups.SecGroup, error)
	CreateSecGroup(ctx context.Context, opts groups.CreateOptsBuilder) (*groups.SecGroup, error)
	DeleteSecGroup(ctx context.Context, id string) error
	GetSecGroup(ctx context.Context, id string) (*groups.SecGroup, error)
	UpdateSecGroup(ctx context.Context, id string, opts groups.UpdateOptsBuilder) (*groups.SecGroup, error)

	ListSecGroupRule(ctx context.Context, opts rules.ListOpts) ([]rules.SecGroupRule, error)
	CreateSecGroupRule(ctx context.Context, opts rules.CreateOptsBuilder) (*rules.SecGroupRule, error)
	DeleteSecGroupRule(ctx context.Context, id string) error
	GetSecGroupRule(ctx context.Context, id string) (*rules.SecGroupRule, error)

	ListNetwork(opts networks.ListOptsBuilder) pagination.Pager
	CreateNetwork(ctx context.Context, opts networks.CreateOptsBuilder) networks.CreateResult
	DeleteNetwork(ctx context.Context, id string) networks.DeleteResult
	GetNetwork(ctx context.Context, id string) networks.GetResult
	UpdateNetwork(ctx context.Context, id string, opts networks.UpdateOptsBuilder) networks.UpdateResult

	ListSubnet(ctx context.Context, opts subnets.ListOptsBuilder) ([]subnets.Subnet, error)
	CreateSubnet(ctx context.Context, opts subnets.CreateOptsBuilder) (*subnets.Subnet, error)
	DeleteSubnet(ctx context.Context, id string) error
	GetSubnet(ctx context.Context, id string) (*subnets.Subnet, error)
	UpdateSubnet(ctx context.Context, id string, opts subnets.UpdateOptsBuilder) (*subnets.Subnet, error)

	ListExtensions() ([]extensions.Extension, error)

	ReplaceAllAttributesTags(ctx context.Context, resourceType string, resourceID string, opts attributestags.ReplaceAllOptsBuilder) ([]string, error)
}

type networkClient struct {
	serviceClient *gophercloud.ServiceClient
}

var _ NetworkClient = &networkClient{}

// NewNetworkClient returns an instance of the networking service.
func NewNetworkClient(providerClient *gophercloud.ProviderClient, providerClientOpts *clientconfig.ClientOpts) (NetworkClient, error) {
	serviceClient, err := openstack.NewNetworkV2(providerClient, gophercloud.EndpointOpts{
		Region:       providerClientOpts.RegionName,
		Availability: clientconfig.GetEndpointType(providerClientOpts.EndpointType),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create networking service providerClient: %v", err)
	}

	return networkClient{serviceClient}, nil
}

func (c networkClient) AddRouterInterface(ctx context.Context, id string, opts routers.AddInterfaceOptsBuilder) (*routers.InterfaceInfo, error) {
	return routers.AddInterface(ctx, c.serviceClient, id, opts).Extract()
}

func (c networkClient) RemoveRouterInterface(ctx context.Context, id string, opts routers.RemoveInterfaceOptsBuilder) (*routers.InterfaceInfo, error) {
	return routers.RemoveInterface(ctx, c.serviceClient, id, opts).Extract()
}

func (c networkClient) ReplaceAllAttributesTags(ctx context.Context, resourceType string, resourceID string, opts attributestags.ReplaceAllOptsBuilder) ([]string, error) {
	return attributestags.ReplaceAll(ctx, c.serviceClient, resourceType, resourceID, opts).Extract()
}

func (c networkClient) ListRouter(ctx context.Context, opts routers.ListOpts) ([]routers.Router, error) {
	allPages, err := routers.List(c.serviceClient, opts).AllPages(ctx)
	if err != nil {
		return nil, err
	}
	return routers.ExtractRouters(allPages)
}

func (c networkClient) ListFloatingIP(opts floatingips.ListOptsBuilder) ([]floatingips.FloatingIP, error) {
	allPages, err := floatingips.List(c.serviceClient, opts).AllPages(context.TODO())
	if err != nil {
		return nil, err
	}
	return floatingips.ExtractFloatingIPs(allPages)
}

func (c networkClient) CreateFloatingIP(opts floatingips.CreateOptsBuilder) (*floatingips.FloatingIP, error) {
	fip, err := floatingips.Create(context.TODO(), c.serviceClient, opts).Extract()
	if err != nil {
		return nil, err
	}
	return fip, nil
}

func (c networkClient) DeleteFloatingIP(id string) error {
	return floatingips.Delete(context.TODO(), c.serviceClient, id).ExtractErr()
}

func (c networkClient) GetFloatingIP(id string) (*floatingips.FloatingIP, error) {
	return floatingips.Get(context.TODO(), c.serviceClient, id).Extract()
}

func (c networkClient) UpdateFloatingIP(id string, opts floatingips.UpdateOptsBuilder) (*floatingips.FloatingIP, error) {
	return floatingips.Update(context.TODO(), c.serviceClient, id, opts).Extract()
}

func (c networkClient) ListPort(ctx context.Context, opts ports.ListOptsBuilder) ([]ports.Port, error) {
	allPages, err := ports.List(c.serviceClient, opts).AllPages(ctx)
	if err != nil {
		return nil, err
	}
	return ports.ExtractPorts(allPages)
}

func (c networkClient) CreatePort(ctx context.Context, opts ports.CreateOptsBuilder) (*ports.Port, error) {
	return ports.Create(ctx, c.serviceClient, opts).Extract()
}

func (c networkClient) DeletePort(ctx context.Context, id string) error {
	return ports.Delete(ctx, c.serviceClient, id).ExtractErr()
}

func (c networkClient) GetPort(ctx context.Context, id string) (*ports.Port, error) {
	return ports.Get(ctx, c.serviceClient, id).Extract()
}

func (c networkClient) UpdatePort(id string, opts ports.UpdateOptsBuilder) (*ports.Port, error) {
	return ports.Update(context.TODO(), c.serviceClient, id, opts).Extract()
}

func (c networkClient) CreateTrunk(opts trunks.CreateOptsBuilder) (*trunks.Trunk, error) {
	return trunks.Create(context.TODO(), c.serviceClient, opts).Extract()
}

func (c networkClient) DeleteTrunk(id string) error {
	return trunks.Delete(context.TODO(), c.serviceClient, id).ExtractErr()
}

func (c networkClient) ListTrunkSubports(trunkID string) ([]trunks.Subport, error) {
	return trunks.GetSubports(context.TODO(), c.serviceClient, trunkID).Extract()
}

func (c networkClient) RemoveSubports(id string, opts trunks.RemoveSubportsOpts) error {
	_, err := trunks.RemoveSubports(context.TODO(), c.serviceClient, id, opts).Extract()
	return err
}

func (c networkClient) ListTrunk(opts trunks.ListOptsBuilder) ([]trunks.Trunk, error) {
	allPages, err := trunks.List(c.serviceClient, opts).AllPages(context.TODO())
	if err != nil {
		return nil, err
	}
	return trunks.ExtractTrunks(allPages)
}

func (c networkClient) CreateRouter(ctx context.Context, opts routers.CreateOptsBuilder) (*routers.Router, error) {
	return routers.Create(ctx, c.serviceClient, opts).Extract()
}

func (c networkClient) DeleteRouter(ctx context.Context, id string) error {
	return routers.Delete(ctx, c.serviceClient, id).ExtractErr()
}

func (c networkClient) GetRouter(ctx context.Context, id string) (*routers.Router, error) {
	return routers.Get(ctx, c.serviceClient, id).Extract()
}

func (c networkClient) UpdateRouter(ctx context.Context, id string, opts routers.UpdateOptsBuilder) (*routers.Router, error) {
	return routers.Update(context.TODO(), c.serviceClient, id, opts).Extract()
}

func (c networkClient) ListSecGroup(ctx context.Context, opts groups.ListOpts) ([]groups.SecGroup, error) {
	allPages, err := groups.List(c.serviceClient, opts).AllPages(ctx)
	if err != nil {
		return nil, err
	}
	return groups.ExtractGroups(allPages)
}

func (c networkClient) CreateSecGroup(ctx context.Context, opts groups.CreateOptsBuilder) (*groups.SecGroup, error) {
	return groups.Create(ctx, c.serviceClient, opts).Extract()
}

func (c networkClient) DeleteSecGroup(ctx context.Context, id string) error {
	return groups.Delete(ctx, c.serviceClient, id).ExtractErr()
}

func (c networkClient) GetSecGroup(ctx context.Context, id string) (*groups.SecGroup, error) {
	return groups.Get(ctx, c.serviceClient, id).Extract()
}

func (c networkClient) UpdateSecGroup(ctx context.Context, id string, opts groups.UpdateOptsBuilder) (*groups.SecGroup, error) {
	return groups.Update(ctx, c.serviceClient, id, opts).Extract()
}

func (c networkClient) ListSecGroupRule(ctx context.Context, opts rules.ListOpts) ([]rules.SecGroupRule, error) {
	allPages, err := rules.List(c.serviceClient, opts).AllPages(ctx)
	if err != nil {
		return nil, err
	}
	return rules.ExtractRules(allPages)
}

func (c networkClient) CreateSecGroupRule(ctx context.Context, opts rules.CreateOptsBuilder) (*rules.SecGroupRule, error) {
	return rules.Create(ctx, c.serviceClient, opts).Extract()
}

func (c networkClient) DeleteSecGroupRule(ctx context.Context, id string) error {
	return rules.Delete(ctx, c.serviceClient, id).ExtractErr()
}

func (c networkClient) GetSecGroupRule(ctx context.Context, id string) (*rules.SecGroupRule, error) {
	return rules.Get(ctx, c.serviceClient, id).Extract()
}

func (c networkClient) ListNetwork(opts networks.ListOptsBuilder) pagination.Pager {
	return networks.List(c.serviceClient, opts)
}

func (c networkClient) CreateNetwork(ctx context.Context, opts networks.CreateOptsBuilder) networks.CreateResult {
	return networks.Create(ctx, c.serviceClient, opts)
}

func (c networkClient) DeleteNetwork(ctx context.Context, id string) networks.DeleteResult {
	return networks.Delete(ctx, c.serviceClient, id)
}

func (c networkClient) GetNetwork(ctx context.Context, id string) networks.GetResult {
	return networks.Get(ctx, c.serviceClient, id)
}

func (c networkClient) UpdateNetwork(ctx context.Context, id string, opts networks.UpdateOptsBuilder) networks.UpdateResult {
	return networks.Update(ctx, c.serviceClient, id, opts)
}

func (c networkClient) ListSubnet(ctx context.Context, opts subnets.ListOptsBuilder) ([]subnets.Subnet, error) {
	allPages, err := subnets.List(c.serviceClient, opts).AllPages(context.TODO())
	if err != nil {
		return nil, err
	}
	return subnets.ExtractSubnets(allPages)
}

func (c networkClient) CreateSubnet(ctx context.Context, opts subnets.CreateOptsBuilder) (*subnets.Subnet, error) {
	return subnets.Create(ctx, c.serviceClient, opts).Extract()
}

func (c networkClient) DeleteSubnet(ctx context.Context, id string) error {
	return subnets.Delete(ctx, c.serviceClient, id).ExtractErr()
}

func (c networkClient) GetSubnet(ctx context.Context, id string) (*subnets.Subnet, error) {
	return subnets.Get(ctx, c.serviceClient, id).Extract()
}

func (c networkClient) UpdateSubnet(ctx context.Context, id string, opts subnets.UpdateOptsBuilder) (*subnets.Subnet, error) {
	return subnets.Update(ctx, c.serviceClient, id, opts).Extract()
}

func (c networkClient) ListExtensions() ([]extensions.Extension, error) {
	allPages, err := extensions.List(c.serviceClient).AllPages(context.TODO())
	if err != nil {
		return nil, err
	}
	return extensions.ExtractExtensions(allPages)
}
