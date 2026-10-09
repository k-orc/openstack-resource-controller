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

type DNSZoneClient interface {
	ListDNSZones(ctx context.Context, listOpts zones.ListOptsBuilder) iter.Seq2[*zones.Zone, error]
	CreateDNSZone(ctx context.Context, opts zones.CreateOptsBuilder) (*zones.Zone, error)
	DeleteDNSZone(ctx context.Context, resourceID string) error
	GetDNSZone(ctx context.Context, resourceID string) (*zones.Zone, error)
	UpdateDNSZone(ctx context.Context, id string, opts zones.UpdateOptsBuilder) (*zones.Zone, error)
}

type dnszoneClient struct{ client *gophercloud.ServiceClient }

// NewDNSZoneClient returns a new OpenStack client.
func NewDNSZoneClient(providerClient *gophercloud.ProviderClient, providerClientOpts *clientconfig.ClientOpts) (DNSZoneClient, error) {
	client, err := openstack.NewDNSV2(providerClient, gophercloud.EndpointOpts{
		Region:       providerClientOpts.RegionName,
		Availability: clientconfig.GetEndpointType(providerClientOpts.EndpointType),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create dnszone service client: %v", err)
	}

	return &dnszoneClient{client}, nil
}

func (c dnszoneClient) ListDNSZones(ctx context.Context, listOpts zones.ListOptsBuilder) iter.Seq2[*zones.Zone, error] {
	pager := zones.List(c.client, listOpts)
	return func(yield func(*zones.Zone, error) bool) {
		_ = pager.EachPage(ctx, yieldPage(zones.ExtractZones, yield))
	}
}

func (c dnszoneClient) CreateDNSZone(ctx context.Context, opts zones.CreateOptsBuilder) (*zones.Zone, error) {
	return zones.Create(ctx, c.client, opts).Extract()
}

func (c dnszoneClient) DeleteDNSZone(ctx context.Context, resourceID string) error {
	_, err := zones.Delete(ctx, c.client, resourceID).Extract()
	return err
}

func (c dnszoneClient) GetDNSZone(ctx context.Context, resourceID string) (*zones.Zone, error) {
	return zones.Get(ctx, c.client, resourceID).Extract()
}

func (c dnszoneClient) UpdateDNSZone(ctx context.Context, id string, opts zones.UpdateOptsBuilder) (*zones.Zone, error) {
	return zones.Update(ctx, c.client, id, opts).Extract()
}

type dnszoneErrorClient struct{ error }

// NewDNSZoneErrorClient returns a DNSZoneClient in which every method returns the given error.
func NewDNSZoneErrorClient(e error) DNSZoneClient {
	return dnszoneErrorClient{e}
}

func (e dnszoneErrorClient) ListDNSZones(_ context.Context, _ zones.ListOptsBuilder) iter.Seq2[*zones.Zone, error] {
	return func(yield func(*zones.Zone, error) bool) {
		yield(nil, e.error)
	}
}

func (e dnszoneErrorClient) CreateDNSZone(_ context.Context, _ zones.CreateOptsBuilder) (*zones.Zone, error) {
	return nil, e.error
}

func (e dnszoneErrorClient) DeleteDNSZone(_ context.Context, _ string) error {
	return e.error
}

func (e dnszoneErrorClient) GetDNSZone(_ context.Context, _ string) (*zones.Zone, error) {
	return nil, e.error
}

func (e dnszoneErrorClient) UpdateDNSZone(_ context.Context, _ string, _ zones.UpdateOptsBuilder) (*zones.Zone, error) {
	return nil, e.error
}
