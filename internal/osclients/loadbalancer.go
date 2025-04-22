/*
Copyright 2022 The ORC Authors.

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
	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/apiversions"
	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/flavors"
	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/listeners"
	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/loadbalancers"
	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/monitors"
	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/pools"
	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/providers"
	"github.com/gophercloud/utils/v2/openstack/clientconfig"
)

type LbClient interface {
	CreateLoadBalancer(ctx context.Context, opts loadbalancers.CreateOptsBuilder) (*loadbalancers.LoadBalancer, error)
	ListLoadBalancers(ctx context.Context, opts loadbalancers.ListOptsBuilder) ([]loadbalancers.LoadBalancer, error)
	GetLoadBalancer(ctx context.Context, id string) (*loadbalancers.LoadBalancer, error)
	DeleteLoadBalancer(ctx context.Context, id string, opts loadbalancers.DeleteOptsBuilder) error
	CreateListener(ctx context.Context, opts listeners.CreateOptsBuilder) (*listeners.Listener, error)
	ListListeners(ctx context.Context, opts listeners.ListOptsBuilder) ([]listeners.Listener, error)
	UpdateListener(ctx context.Context, id string, opts listeners.UpdateOpts) (*listeners.Listener, error)
	GetListener(ctx context.Context, id string) (*listeners.Listener, error)
	DeleteListener(ctx context.Context, id string) error
	CreatePool(ctx context.Context, opts pools.CreateOptsBuilder) (*pools.Pool, error)
	ListPools(ctx context.Context, opts pools.ListOptsBuilder) ([]pools.Pool, error)
	GetPool(ctx context.Context, id string) (*pools.Pool, error)
	DeletePool(ctx context.Context, id string) error
	CreatePoolMember(ctx context.Context, poolID string, opts pools.CreateMemberOptsBuilder) (*pools.Member, error)
	ListPoolMember(ctx context.Context, poolID string, opts pools.ListMembersOptsBuilder) ([]pools.Member, error)
	DeletePoolMember(ctx context.Context, poolID string, lbMemberID string) error
	CreateMonitor(ctx context.Context, opts monitors.CreateOptsBuilder) (*monitors.Monitor, error)
	ListMonitors(ctx context.Context, opts monitors.ListOptsBuilder) ([]monitors.Monitor, error)
	DeleteMonitor(ctx context.Context, id string) error
	ListLoadBalancerProviders(ctx context.Context) ([]providers.Provider, error)
	ListOctaviaVersions(ctx context.Context) ([]apiversions.APIVersion, error)
	ListLoadBalancerFlavors(ctx context.Context) ([]flavors.Flavor, error)
}

type lbClient struct {
	serviceClient *gophercloud.ServiceClient
}

// NewLbClient returns a new loadbalancer client.
func NewLbClient(providerClient *gophercloud.ProviderClient, providerClientOpts *clientconfig.ClientOpts) (LbClient, error) {
	loadbalancerClient, err := openstack.NewLoadBalancerV2(providerClient, gophercloud.EndpointOpts{
		Region:       providerClientOpts.RegionName,
		Availability: clientconfig.GetEndpointType(providerClientOpts.EndpointType),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create load balancer service client: %v", err)
	}

	return &lbClient{loadbalancerClient}, nil
}

func (l lbClient) CreateLoadBalancer(ctx context.Context, opts loadbalancers.CreateOptsBuilder) (*loadbalancers.LoadBalancer, error) {
	return loadbalancers.Create(ctx, l.serviceClient, opts).Extract()
}

func (l lbClient) ListLoadBalancers(ctx context.Context, opts loadbalancers.ListOptsBuilder) ([]loadbalancers.LoadBalancer, error) {
	allPages, err := loadbalancers.List(l.serviceClient, opts).AllPages(ctx)
	if err != nil {
		return nil, err
	}
	return loadbalancers.ExtractLoadBalancers(allPages)
}

func (l lbClient) GetLoadBalancer(ctx context.Context, id string) (*loadbalancers.LoadBalancer, error) {
	return loadbalancers.Get(ctx, l.serviceClient, id).Extract()
}

func (l lbClient) DeleteLoadBalancer(ctx context.Context, id string, opts loadbalancers.DeleteOptsBuilder) error {
	return loadbalancers.Delete(ctx, l.serviceClient, id, opts).ExtractErr()
}

func (l lbClient) CreateListener(ctx context.Context, opts listeners.CreateOptsBuilder) (*listeners.Listener, error) {
	return listeners.Create(ctx, l.serviceClient, opts).Extract()
}

func (l lbClient) UpdateListener(ctx context.Context, id string, opts listeners.UpdateOpts) (*listeners.Listener, error) {
	return listeners.Update(ctx, l.serviceClient, id, opts).Extract()
}

func (l lbClient) ListListeners(ctx context.Context, opts listeners.ListOptsBuilder) ([]listeners.Listener, error) {
	allPages, err := listeners.List(l.serviceClient, opts).AllPages(ctx)
	if err != nil {
		return nil, err
	}
	return listeners.ExtractListeners(allPages)
}

func (l lbClient) GetListener(ctx context.Context, id string) (*listeners.Listener, error) {
	return listeners.Get(ctx, l.serviceClient, id).Extract()
}

func (l lbClient) DeleteListener(ctx context.Context, id string) error {
	return listeners.Delete(ctx, l.serviceClient, id).ExtractErr()
}

func (l lbClient) CreatePool(ctx context.Context, opts pools.CreateOptsBuilder) (*pools.Pool, error) {
	return pools.Create(ctx, l.serviceClient, opts).Extract()
}

func (l lbClient) ListPools(ctx context.Context, opts pools.ListOptsBuilder) ([]pools.Pool, error) {
	allPages, err := pools.List(l.serviceClient, opts).AllPages(ctx)
	if err != nil {
		return nil, err
	}
	return pools.ExtractPools(allPages)
}

func (l lbClient) GetPool(ctx context.Context, id string) (*pools.Pool, error) {
	return pools.Get(ctx, l.serviceClient, id).Extract()
}

func (l lbClient) DeletePool(ctx context.Context, id string) error {
	return pools.Delete(ctx, l.serviceClient, id).ExtractErr()
}

func (l lbClient) CreatePoolMember(ctx context.Context, poolID string, lbMemberOpts pools.CreateMemberOptsBuilder) (*pools.Member, error) {
	return pools.CreateMember(ctx, l.serviceClient, poolID, lbMemberOpts).Extract()
}

func (l lbClient) ListPoolMember(ctx context.Context, poolID string, opts pools.ListMembersOptsBuilder) ([]pools.Member, error) {
	allPages, err := pools.ListMembers(l.serviceClient, poolID, opts).AllPages(ctx)
	if err != nil {
		return nil, err
	}
	return pools.ExtractMembers(allPages)
}

func (l lbClient) DeletePoolMember(ctx context.Context, poolID string, lbMemberID string) error {
	return pools.DeleteMember(ctx, l.serviceClient, poolID, lbMemberID).ExtractErr()
}

func (l lbClient) CreateMonitor(ctx context.Context, opts monitors.CreateOptsBuilder) (*monitors.Monitor, error) {
	return monitors.Create(ctx, l.serviceClient, opts).Extract()
}

func (l lbClient) ListMonitors(ctx context.Context, opts monitors.ListOptsBuilder) ([]monitors.Monitor, error) {
	allPages, err := monitors.List(l.serviceClient, opts).AllPages(ctx)
	if err != nil {
		return nil, err
	}
	return monitors.ExtractMonitors(allPages)
}

func (l lbClient) DeleteMonitor(ctx context.Context, id string) error {
	return monitors.Delete(ctx, l.serviceClient, id).ExtractErr()
}

func (l lbClient) ListLoadBalancerProviders(ctx context.Context) ([]providers.Provider, error) {
	allPages, err := providers.List(l.serviceClient, providers.ListOpts{}).AllPages(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing providers: %v", err)
	}
	return providers.ExtractProviders(allPages)
}

func (l lbClient) ListOctaviaVersions(ctx context.Context) ([]apiversions.APIVersion, error) {
	allPages, err := apiversions.List(l.serviceClient).AllPages(ctx)
	if err != nil {
		return nil, err
	}
	return apiversions.ExtractAPIVersions(allPages)
}

func (l lbClient) ListLoadBalancerFlavors(ctx context.Context) ([]flavors.Flavor, error) {
	allPages, err := flavors.List(l.serviceClient, flavors.ListOpts{}).AllPages(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing flavors: %v", err)
	}
	return flavors.ExtractFlavors(allPages)
}
