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

// DNSClient covers Designate resources on a single shared ServiceClient, the same convention
// used by NetworkClient for all Neutron resources - one OpenStack service, one client, rather
// than a dedicated client per Kubernetes kind. RecordSet adds its own methods to this interface
// in a follow-up PR.
type DNSClient interface {
	ListZones(ctx context.Context, listOpts zones.ListOptsBuilder) iter.Seq2[*zones.Zone, error]
	CreateZone(ctx context.Context, opts zones.CreateOptsBuilder) (*zones.Zone, error)
	DeleteZone(ctx context.Context, id string) error
	GetZone(ctx context.Context, id string) (*zones.Zone, error)
	UpdateZone(ctx context.Context, id string, opts zones.UpdateOptsBuilder) (*zones.Zone, error)

	// Zone shares: Designate's single-credential zone-sharing API (distinct from, and much
	// simpler than, zone ownership transfer - see DNSZoneShare's own doc comment). Note there is
	// no update operation - a share's action/target can't be changed, only created or removed.
	ListZoneShares(ctx context.Context, zoneID string) iter.Seq2[*zones.ZoneShare, error]
	CreateZoneShare(ctx context.Context, zoneID string, opts zones.ShareOptsBuilder) (*zones.ZoneShare, error)
	DeleteZoneShare(ctx context.Context, zoneID, shareID string) error
	GetZoneShare(ctx context.Context, zoneID, shareID string) (*zones.ZoneShare, error)
}

type dnsClient struct{ client *gophercloud.ServiceClient }

// NewDNSClient returns a new Designate client.
func NewDNSClient(providerClient *gophercloud.ProviderClient, providerClientOpts *clientconfig.ClientOpts) (DNSClient, error) {
	client, err := openstack.NewDNSV2(providerClient, gophercloud.EndpointOpts{
		Region:       providerClientOpts.RegionName,
		Availability: clientconfig.GetEndpointType(providerClientOpts.EndpointType),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create dns service client: %v", err)
	}

	return dnsClient{client}, nil
}

func (c dnsClient) ListZones(ctx context.Context, listOpts zones.ListOptsBuilder) iter.Seq2[*zones.Zone, error] {
	pager := zones.List(c.client, listOpts)
	return func(yield func(*zones.Zone, error) bool) {
		_ = pager.EachPage(ctx, yieldPage(zones.ExtractZones, yield))
	}
}

func (c dnsClient) CreateZone(ctx context.Context, opts zones.CreateOptsBuilder) (*zones.Zone, error) {
	return zones.Create(ctx, c.client, opts).Extract()
}

func (c dnsClient) DeleteZone(ctx context.Context, id string) error {
	_, err := zones.Delete(ctx, c.client, id).Extract()
	return err
}

func (c dnsClient) GetZone(ctx context.Context, id string) (*zones.Zone, error) {
	return zones.Get(ctx, c.client, id).Extract()
}

func (c dnsClient) UpdateZone(ctx context.Context, id string, opts zones.UpdateOptsBuilder) (*zones.Zone, error) {
	return zones.Update(ctx, c.client, id, opts).Extract()
}

func (c dnsClient) ListZoneShares(ctx context.Context, zoneID string) iter.Seq2[*zones.ZoneShare, error] {
	pager := zones.ListShares(c.client, zoneID, nil)
	return func(yield func(*zones.ZoneShare, error) bool) {
		_ = pager.EachPage(ctx, yieldPage(zones.ExtractZoneShares, yield))
	}
}

func (c dnsClient) CreateZoneShare(ctx context.Context, zoneID string, opts zones.ShareOptsBuilder) (*zones.ZoneShare, error) {
	return zones.Share(ctx, c.client, zoneID, opts).Extract()
}

func (c dnsClient) DeleteZoneShare(ctx context.Context, zoneID, shareID string) error {
	return zones.Unshare(ctx, c.client, zoneID, shareID).ExtractErr()
}

func (c dnsClient) GetZoneShare(ctx context.Context, zoneID, shareID string) (*zones.ZoneShare, error) {
	return zones.GetShare(ctx, c.client, zoneID, shareID).Extract()
}

type dnsErrorClient struct{ error }

// NewDNSErrorClient returns a DNSClient in which every method returns the given error.
func NewDNSErrorClient(e error) DNSClient {
	return dnsErrorClient{e}
}

func (e dnsErrorClient) ListZones(_ context.Context, _ zones.ListOptsBuilder) iter.Seq2[*zones.Zone, error] {
	return func(yield func(*zones.Zone, error) bool) { yield(nil, e.error) }
}

func (e dnsErrorClient) CreateZone(_ context.Context, _ zones.CreateOptsBuilder) (*zones.Zone, error) {
	return nil, e.error
}

func (e dnsErrorClient) DeleteZone(_ context.Context, _ string) error { return e.error }

func (e dnsErrorClient) GetZone(_ context.Context, _ string) (*zones.Zone, error) {
	return nil, e.error
}

func (e dnsErrorClient) UpdateZone(_ context.Context, _ string, _ zones.UpdateOptsBuilder) (*zones.Zone, error) {
	return nil, e.error
}

func (e dnsErrorClient) ListZoneShares(_ context.Context, _ string) iter.Seq2[*zones.ZoneShare, error] {
	return func(yield func(*zones.ZoneShare, error) bool) { yield(nil, e.error) }
}

func (e dnsErrorClient) CreateZoneShare(_ context.Context, _ string, _ zones.ShareOptsBuilder) (*zones.ZoneShare, error) {
	return nil, e.error
}

func (e dnsErrorClient) DeleteZoneShare(_ context.Context, _, _ string) error { return e.error }

func (e dnsErrorClient) GetZoneShare(_ context.Context, _, _ string) (*zones.ZoneShare, error) {
	return nil, e.error
}
