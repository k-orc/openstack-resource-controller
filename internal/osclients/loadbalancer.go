/*
Copyright 2026 The ORC Authors.

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

type LoadBalancerClient interface {
	ListLoadBalancers(ctx context.Context, listOpts loadbalancers.ListOptsBuilder) iter.Seq2[*loadbalancers.LoadBalancer, error]
	CreateLoadBalancer(ctx context.Context, opts loadbalancers.CreateOptsBuilder) (*loadbalancers.LoadBalancer, error)
	DeleteLoadBalancer(ctx context.Context, resourceID string) error
	GetLoadBalancer(ctx context.Context, resourceID string) (*loadbalancers.LoadBalancer, error)
	UpdateLoadBalancer(ctx context.Context, id string, opts loadbalancers.UpdateOptsBuilder) (*loadbalancers.LoadBalancer, error)
}

type loadbalancerClient struct{ client *gophercloud.ServiceClient }

// NewLoadBalancerClient returns a new OpenStack client.
func NewLoadBalancerClient(providerClient *gophercloud.ProviderClient, providerClientOpts *clientconfig.ClientOpts) (LoadBalancerClient, error) {
	client, err := openstack.NewLoadBalancerV2(providerClient, gophercloud.EndpointOpts{
		Region:       providerClientOpts.RegionName,
		Availability: clientconfig.GetEndpointType(providerClientOpts.EndpointType),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create loadbalancer service client: %v", err)
	}

	return &loadbalancerClient{client}, nil
}

func (c loadbalancerClient) ListLoadBalancers(ctx context.Context, listOpts loadbalancers.ListOptsBuilder) iter.Seq2[*loadbalancers.LoadBalancer, error] {
	pager := loadbalancers.List(c.client, listOpts)
	return func(yield func(*loadbalancers.LoadBalancer, error) bool) {
		_ = pager.EachPage(ctx, yieldPage(loadbalancers.ExtractLoadBalancers, yield))
	}
}

func (c loadbalancerClient) CreateLoadBalancer(ctx context.Context, opts loadbalancers.CreateOptsBuilder) (*loadbalancers.LoadBalancer, error) {
	return loadbalancers.Create(ctx, c.client, opts).Extract()
}

func (c loadbalancerClient) DeleteLoadBalancer(ctx context.Context, resourceID string) error {
	return loadbalancers.Delete(ctx, c.client, resourceID, nil).ExtractErr()
}

func (c loadbalancerClient) GetLoadBalancer(ctx context.Context, resourceID string) (*loadbalancers.LoadBalancer, error) {
	return loadbalancers.Get(ctx, c.client, resourceID).Extract()
}

func (c loadbalancerClient) UpdateLoadBalancer(ctx context.Context, id string, opts loadbalancers.UpdateOptsBuilder) (*loadbalancers.LoadBalancer, error) {
	return loadbalancers.Update(ctx, c.client, id, opts).Extract()
}

type loadbalancerErrorClient struct{ error }

// NewLoadBalancerErrorClient returns a LoadBalancerClient in which every method returns the given error.
func NewLoadBalancerErrorClient(e error) LoadBalancerClient {
	return loadbalancerErrorClient{e}
}

func (e loadbalancerErrorClient) ListLoadBalancers(_ context.Context, _ loadbalancers.ListOptsBuilder) iter.Seq2[*loadbalancers.LoadBalancer, error] {
	return func(yield func(*loadbalancers.LoadBalancer, error) bool) {
		yield(nil, e.error)
	}
}

func (e loadbalancerErrorClient) CreateLoadBalancer(_ context.Context, _ loadbalancers.CreateOptsBuilder) (*loadbalancers.LoadBalancer, error) {
	return nil, e.error
}

func (e loadbalancerErrorClient) DeleteLoadBalancer(_ context.Context, _ string) error {
	return e.error
}

func (e loadbalancerErrorClient) GetLoadBalancer(_ context.Context, _ string) (*loadbalancers.LoadBalancer, error) {
	return nil, e.error
}

func (e loadbalancerErrorClient) UpdateLoadBalancer(_ context.Context, _ string, _ loadbalancers.UpdateOptsBuilder) (*loadbalancers.LoadBalancer, error) {
	return nil, e.error
}
