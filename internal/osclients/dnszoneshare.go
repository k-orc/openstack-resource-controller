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

// DNSZoneShareClient covers Designate zone-share operations. Unlike DNSZone/RecordSet, the
// zone-share API has no update operation at all - only list, create (share) and delete (unshare).
type DNSZoneShareClient interface {
	ListZoneShares(ctx context.Context, zoneID string) iter.Seq2[*zones.ZoneShare, error]
	CreateZoneShare(ctx context.Context, zoneID string, opts zones.ShareOptsBuilder) (*zones.ZoneShare, error)
	DeleteZoneShare(ctx context.Context, zoneID, shareID string) error
	GetZoneShare(ctx context.Context, zoneID, shareID string) (*zones.ZoneShare, error)
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

func (c dnszoneshareClient) ListZoneShares(ctx context.Context, zoneID string) iter.Seq2[*zones.ZoneShare, error] {
	pager := zones.ListShares(c.client, zoneID, nil)
	return func(yield func(*zones.ZoneShare, error) bool) {
		_ = pager.EachPage(ctx, yieldPage(zones.ExtractZoneShares, yield))
	}
}

func (c dnszoneshareClient) CreateZoneShare(ctx context.Context, zoneID string, opts zones.ShareOptsBuilder) (*zones.ZoneShare, error) {
	return zones.Share(ctx, c.client, zoneID, opts).Extract()
}

func (c dnszoneshareClient) DeleteZoneShare(ctx context.Context, zoneID, shareID string) error {
	return zones.Unshare(ctx, c.client, zoneID, shareID).ExtractErr()
}

func (c dnszoneshareClient) GetZoneShare(ctx context.Context, zoneID, shareID string) (*zones.ZoneShare, error) {
	return zones.GetShare(ctx, c.client, zoneID, shareID).Extract()
}

type dnszoneshareErrorClient struct{ error }

// NewDNSZoneShareErrorClient returns a DNSZoneShareClient in which every method returns the given error.
func NewDNSZoneShareErrorClient(e error) DNSZoneShareClient {
	return dnszoneshareErrorClient{e}
}

func (e dnszoneshareErrorClient) ListZoneShares(_ context.Context, _ string) iter.Seq2[*zones.ZoneShare, error] {
	return func(yield func(*zones.ZoneShare, error) bool) {
		yield(nil, e.error)
	}
}

func (e dnszoneshareErrorClient) CreateZoneShare(_ context.Context, _ string, _ zones.ShareOptsBuilder) (*zones.ZoneShare, error) {
	return nil, e.error
}

func (e dnszoneshareErrorClient) DeleteZoneShare(_ context.Context, _, _ string) error {
	return e.error
}

func (e dnszoneshareErrorClient) GetZoneShare(_ context.Context, _, _ string) (*zones.ZoneShare, error) {
	return nil, e.error
}
