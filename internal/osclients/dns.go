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
	"github.com/gophercloud/gophercloud/v2/openstack/dns/v2/recordsets"
	"github.com/gophercloud/gophercloud/v2/openstack/dns/v2/zones"
	"github.com/gophercloud/utils/v2/openstack/clientconfig"
)

// DNSClient covers Designate resources on a single shared ServiceClient, the same convention
// used by NetworkClient for all Neutron resources - one OpenStack service, one client, rather
// than a dedicated client per Kubernetes kind. DNSZoneShare adds its own methods to this
// interface in a follow-up PR.
type DNSClient interface {
	ListZones(ctx context.Context, listOpts zones.ListOptsBuilder) iter.Seq2[*zones.Zone, error]
	CreateZone(ctx context.Context, opts zones.CreateOptsBuilder) (*zones.Zone, error)
	DeleteZone(ctx context.Context, id string) error
	GetZone(ctx context.Context, id string) (*zones.Zone, error)
	UpdateZone(ctx context.Context, id string, opts zones.UpdateOptsBuilder) (*zones.Zone, error)

	ListRecordSets(ctx context.Context, zoneID string, listOpts recordsets.ListOptsBuilder) iter.Seq2[*recordsets.RecordSet, error]
	CreateRecordSet(ctx context.Context, zoneID string, opts recordsets.CreateOptsBuilder) (*recordsets.RecordSet, error)
	DeleteRecordSet(ctx context.Context, zoneID, id string) error
	GetRecordSet(ctx context.Context, zoneID, id string) (*recordsets.RecordSet, error)
	UpdateRecordSet(ctx context.Context, zoneID, id string, opts recordsets.UpdateOptsBuilder) (*recordsets.RecordSet, error)
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

func (c dnsClient) ListRecordSets(ctx context.Context, zoneID string, listOpts recordsets.ListOptsBuilder) iter.Seq2[*recordsets.RecordSet, error] {
	pager := recordsets.ListByZone(c.client, zoneID, listOpts)
	return func(yield func(*recordsets.RecordSet, error) bool) {
		_ = pager.EachPage(ctx, yieldPage(recordsets.ExtractRecordSets, yield))
	}
}

func (c dnsClient) CreateRecordSet(ctx context.Context, zoneID string, opts recordsets.CreateOptsBuilder) (*recordsets.RecordSet, error) {
	return recordsets.Create(ctx, c.client, zoneID, opts).Extract()
}

func (c dnsClient) DeleteRecordSet(ctx context.Context, zoneID, id string) error {
	return recordsets.Delete(ctx, c.client, zoneID, id).ExtractErr()
}

func (c dnsClient) GetRecordSet(ctx context.Context, zoneID, id string) (*recordsets.RecordSet, error) {
	return recordsets.Get(ctx, c.client, zoneID, id).Extract()
}

func (c dnsClient) UpdateRecordSet(ctx context.Context, zoneID, id string, opts recordsets.UpdateOptsBuilder) (*recordsets.RecordSet, error) {
	return recordsets.Update(ctx, c.client, zoneID, id, opts).Extract()
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

func (e dnsErrorClient) ListRecordSets(_ context.Context, _ string, _ recordsets.ListOptsBuilder) iter.Seq2[*recordsets.RecordSet, error] {
	return func(yield func(*recordsets.RecordSet, error) bool) { yield(nil, e.error) }
}

func (e dnsErrorClient) CreateRecordSet(_ context.Context, _ string, _ recordsets.CreateOptsBuilder) (*recordsets.RecordSet, error) {
	return nil, e.error
}

func (e dnsErrorClient) DeleteRecordSet(_ context.Context, _, _ string) error { return e.error }

func (e dnsErrorClient) GetRecordSet(_ context.Context, _, _ string) (*recordsets.RecordSet, error) {
	return nil, e.error
}

func (e dnsErrorClient) UpdateRecordSet(_ context.Context, _, _ string, _ recordsets.UpdateOptsBuilder) (*recordsets.RecordSet, error) {
	return nil, e.error
}
