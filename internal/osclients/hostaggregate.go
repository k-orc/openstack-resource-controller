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
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/aggregates"
	"github.com/gophercloud/utils/v2/openstack/clientconfig"
)

type HostAggregateClient interface {
	ListHostAggregates(ctx context.Context) iter.Seq2[*aggregates.Aggregate, error]
	CreateHostAggregate(ctx context.Context, opts aggregates.CreateOptsBuilder) (*aggregates.Aggregate, error)
	DeleteHostAggregate(ctx context.Context, resourceID int) error
	GetHostAggregate(ctx context.Context, resourceID int) (*aggregates.Aggregate, error)
	UpdateHostAggregate(ctx context.Context, id int, opts aggregates.UpdateOptsBuilder) (*aggregates.Aggregate, error)
}

type hostaggregateClient struct{ client *gophercloud.ServiceClient }

// NewHostAggregateClient returns a new OpenStack client.
func NewHostAggregateClient(providerClient *gophercloud.ProviderClient, providerClientOpts *clientconfig.ClientOpts) (HostAggregateClient, error) {
	client, err := openstack.NewComputeV2(providerClient, gophercloud.EndpointOpts{
		Region:       providerClientOpts.RegionName,
		Availability: clientconfig.GetEndpointType(providerClientOpts.EndpointType),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create hostaggregate service client: %v", err)
	}

	return &hostaggregateClient{client}, nil
}

func (c hostaggregateClient) ListHostAggregates(ctx context.Context) iter.Seq2[*aggregates.Aggregate, error] {
	pager := aggregates.List(c.client)
	return func(yield func(*aggregates.Aggregate, error) bool) {
		_ = pager.EachPage(ctx, yieldPage(aggregates.ExtractAggregates, yield))
	}
}

func (c hostaggregateClient) CreateHostAggregate(ctx context.Context, opts aggregates.CreateOptsBuilder) (*aggregates.Aggregate, error) {
	return aggregates.Create(ctx, c.client, opts).Extract()
}

func (c hostaggregateClient) DeleteHostAggregate(ctx context.Context, resourceID int) error {
	return aggregates.Delete(ctx, c.client, resourceID).ExtractErr()
}

func (c hostaggregateClient) GetHostAggregate(ctx context.Context, resourceID int) (*aggregates.Aggregate, error) {
	return aggregates.Get(ctx, c.client, resourceID).Extract()
}

func (c hostaggregateClient) UpdateHostAggregate(ctx context.Context, resourceID int, opts aggregates.UpdateOptsBuilder) (*aggregates.Aggregate, error) {
	return aggregates.Update(ctx, c.client, resourceID, opts).Extract()
}

type hostaggregateErrorClient struct{ error }

// NewHostAggregateErrorClient returns a HostAggregateClient in which every method returns the given error.
func NewHostAggregateErrorClient(e error) HostAggregateClient {
	return hostaggregateErrorClient{e}
}

func (e hostaggregateErrorClient) ListHostAggregates(_ context.Context) iter.Seq2[*aggregates.Aggregate, error] {
	return func(yield func(*aggregates.Aggregate, error) bool) {
		yield(nil, e.error)
	}
}

func (e hostaggregateErrorClient) CreateHostAggregate(_ context.Context, _ aggregates.CreateOptsBuilder) (*aggregates.Aggregate, error) {
	return nil, e.error
}

func (e hostaggregateErrorClient) DeleteHostAggregate(_ context.Context, _ int) error {
	return e.error
}

func (e hostaggregateErrorClient) GetHostAggregate(_ context.Context, _ int) (*aggregates.Aggregate, error) {
	return nil, e.error
}

func (e hostaggregateErrorClient) UpdateHostAggregate(_ context.Context, _ int, _ aggregates.UpdateOptsBuilder) (*aggregates.Aggregate, error) {
	return nil, e.error
}
