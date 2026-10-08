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

package osclients

import (
	"context"
	"fmt"
	"iter"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/dns/v2/zones"
	"github.com/gophercloud/utils/v2/openstack/clientconfig"
)

type DNSZoneShareClient interface {
	ListDNSZoneShares(ctx context.Context, listOpts zones.ListOptsBuilder) iter.Seq2[*zones.ZoneShare, error]
	CreateDNSZoneShare(ctx context.Context, opts zones.CreateOptsBuilder) (*zones.ZoneShare, error)
	DeleteDNSZoneShare(ctx context.Context, resourceID string) error
	GetDNSZoneShare(ctx context.Context, resourceID string) (*zones.ZoneShare, error)
	UpdateDNSZoneShare(ctx context.Context, id string, opts zones.UpdateOptsBuilder) (*zones.ZoneShare, error)
}

type dnszoneshareClient struct{ client *gophercloud.ServiceClient }

// NewDNSZoneShareClient returns a new OpenStack client.
func NewDNSZoneShareClient(providerClient *gophercloud.ProviderClient, providerClientOpts *clientconfig.ClientOpts) (DNSZoneShareClient, error) {
	client, err := openstack.NewDNSV2(providerClient, gophercloud.EndpointOpts{
		Region:       providerClientOpts.RegionName,
		Availability: clientconfig.GetEndpointType(providerClientOpts.EndpointType),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create dnszoneshare service client: %v", err)
	}

	return &dnszoneshareClient{client}, nil
}

func (c dnszoneshareClient) ListDNSZoneShares(ctx context.Context, listOpts zones.ListOptsBuilder) iter.Seq2[*zones.ZoneShare, error] {
	pager := zones.List(c.client, listOpts)
	return func(yield func(*zones.ZoneShare, error) bool) {
		_ = pager.EachPage(ctx, yieldPage(zones.ExtractZoneShares, yield))
	}
}

func (c dnszoneshareClient) CreateDNSZoneShare(ctx context.Context, opts zones.CreateOptsBuilder) (*zones.ZoneShare, error) {
	return zones.Create(ctx, c.client, opts).Extract()
}

func (c dnszoneshareClient) DeleteDNSZoneShare(ctx context.Context, resourceID string) error {
	return zones.Delete(ctx, c.client, resourceID).ExtractErr()
}

func (c dnszoneshareClient) GetDNSZoneShare(ctx context.Context, resourceID string) (*zones.ZoneShare, error) {
	return zones.Get(ctx, c.client, resourceID).Extract()
}

func (c dnszoneshareClient) UpdateDNSZoneShare(ctx context.Context, id string, opts zones.UpdateOptsBuilder) (*zones.ZoneShare, error) {
	return zones.Update(ctx, c.client, id, opts).Extract()
}

type dnszoneshareErrorClient struct{ error }

// NewDNSZoneShareErrorClient returns a DNSZoneShareClient in which every method returns the given error.
func NewDNSZoneShareErrorClient(e error) DNSZoneShareClient {
	return dnszoneshareErrorClient{e}
}

func (e dnszoneshareErrorClient) ListDNSZoneShares(_ context.Context, _ zones.ListOptsBuilder) iter.Seq2[*zones.ZoneShare, error] {
	return func(yield func(*zones.ZoneShare, error) bool) {
		yield(nil, e.error)
	}
}

func (e dnszoneshareErrorClient) CreateDNSZoneShare(_ context.Context, _ zones.CreateOptsBuilder) (*zones.ZoneShare, error) {
	return nil, e.error
}

func (e dnszoneshareErrorClient) DeleteDNSZoneShare(_ context.Context, _ string) error {
	return e.error
}

func (e dnszoneshareErrorClient) GetDNSZoneShare(_ context.Context, _ string) (*zones.ZoneShare, error) {
	return nil, e.error
}

func (e dnszoneshareErrorClient) UpdateDNSZoneShare(_ context.Context, _ string, _ zones.UpdateOptsBuilder) (*zones.ZoneShare, error) {
	return nil, e.error
}
