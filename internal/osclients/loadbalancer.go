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

package osclients

import (
	"context"
	"fmt"
	"iter"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/loadbalancers"
	"github.com/gophercloud/utils/v2/openstack/clientconfig"
)

// LoadBalancerClient is an interface for managing Octavia load balancers.
type LoadBalancerClient interface {
	GetLoadBalancer(ctx context.Context, id string) (*loadbalancers.LoadBalancer, error)
	ListLoadBalancer(ctx context.Context, opts loadbalancers.ListOptsBuilder) iter.Seq2[*loadbalancers.LoadBalancer, error]
	CreateLoadBalancer(ctx context.Context, opts loadbalancers.CreateOptsBuilder) (*loadbalancers.LoadBalancer, error)
	UpdateLoadBalancer(ctx context.Context, id string, opts loadbalancers.UpdateOptsBuilder) (*loadbalancers.LoadBalancer, error)
	DeleteLoadBalancer(ctx context.Context, id string, opts loadbalancers.DeleteOptsBuilder) error
}

type loadBalancerClient struct{ client *gophercloud.ServiceClient }

var _ LoadBalancerClient = &loadBalancerClient{}

// NewLoadBalancerClient returns a new Octavia load balancer client.
func NewLoadBalancerClient(providerClient *gophercloud.ProviderClient, providerClientOpts *clientconfig.ClientOpts) (LoadBalancerClient, error) {
	client, err := openstack.NewLoadBalancerV2(providerClient, gophercloud.EndpointOpts{
		Region:       providerClientOpts.RegionName,
		Availability: clientconfig.GetEndpointType(providerClientOpts.EndpointType),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create load balancer service client: %v", err)
	}

	return &loadBalancerClient{client}, nil
}

// NewLoadBalancerClientFromServiceClient returns a LoadBalancerClient using
// the given pre-configured gophercloud service client. This is intended for
// use in tests.
func NewLoadBalancerClientFromServiceClient(sc *gophercloud.ServiceClient) LoadBalancerClient {
	return &loadBalancerClient{sc}
}

func (c loadBalancerClient) GetLoadBalancer(ctx context.Context, id string) (*loadbalancers.LoadBalancer, error) {
	return loadbalancers.Get(ctx, c.client, id).Extract()
}

func (c loadBalancerClient) ListLoadBalancer(ctx context.Context, opts loadbalancers.ListOptsBuilder) iter.Seq2[*loadbalancers.LoadBalancer, error] {
	pager := loadbalancers.List(c.client, opts)
	return func(yield func(*loadbalancers.LoadBalancer, error) bool) {
		_ = pager.EachPage(ctx, yieldPage(loadbalancers.ExtractLoadBalancers, yield))
	}
}

func (c loadBalancerClient) CreateLoadBalancer(ctx context.Context, opts loadbalancers.CreateOptsBuilder) (*loadbalancers.LoadBalancer, error) {
	return loadbalancers.Create(ctx, c.client, opts).Extract()
}

func (c loadBalancerClient) UpdateLoadBalancer(ctx context.Context, id string, opts loadbalancers.UpdateOptsBuilder) (*loadbalancers.LoadBalancer, error) {
	return loadbalancers.Update(ctx, c.client, id, opts).Extract()
}

func (c loadBalancerClient) DeleteLoadBalancer(ctx context.Context, id string, opts loadbalancers.DeleteOptsBuilder) error {
	return loadbalancers.Delete(ctx, c.client, id, opts).ExtractErr()
}
