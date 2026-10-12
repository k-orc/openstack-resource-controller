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
	"github.com/gophercloud/utils/v2/openstack/clientconfig"
)

type RecordSetClient interface {
	ListRecordSets(ctx context.Context, zoneID string, listOpts recordsets.ListOptsBuilder) iter.Seq2[*recordsets.RecordSet, error]
	CreateRecordSet(ctx context.Context, zoneID string, opts recordsets.CreateOptsBuilder) (*recordsets.RecordSet, error)
	DeleteRecordSet(ctx context.Context, zoneID, resourceID string) error
	GetRecordSet(ctx context.Context, zoneID, resourceID string) (*recordsets.RecordSet, error)
	UpdateRecordSet(ctx context.Context, zoneID, id string, opts recordsets.UpdateOptsBuilder) (*recordsets.RecordSet, error)
}

type recordsetClient struct{ client *gophercloud.ServiceClient }

// NewRecordSetClient returns a new OpenStack client.
func NewRecordSetClient(providerClient *gophercloud.ProviderClient, providerClientOpts *clientconfig.ClientOpts) (RecordSetClient, error) {
	client, err := openstack.NewDNSV2(providerClient, gophercloud.EndpointOpts{
		Region:       providerClientOpts.RegionName,
		Availability: clientconfig.GetEndpointType(providerClientOpts.EndpointType),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create recordset service client: %v", err)
	}

	return &recordsetClient{client}, nil
}

func (c recordsetClient) ListRecordSets(ctx context.Context, zoneID string, listOpts recordsets.ListOptsBuilder) iter.Seq2[*recordsets.RecordSet, error] {
	pager := recordsets.ListByZone(c.client, zoneID, listOpts)
	return func(yield func(*recordsets.RecordSet, error) bool) {
		_ = pager.EachPage(ctx, yieldPage(recordsets.ExtractRecordSets, yield))
	}
}

func (c recordsetClient) CreateRecordSet(ctx context.Context, zoneID string, opts recordsets.CreateOptsBuilder) (*recordsets.RecordSet, error) {
	return recordsets.Create(ctx, c.client, zoneID, opts).Extract()
}

func (c recordsetClient) DeleteRecordSet(ctx context.Context, zoneID, resourceID string) error {
	return recordsets.Delete(ctx, c.client, zoneID, resourceID).ExtractErr()
}

func (c recordsetClient) GetRecordSet(ctx context.Context, zoneID, resourceID string) (*recordsets.RecordSet, error) {
	return recordsets.Get(ctx, c.client, zoneID, resourceID).Extract()
}

func (c recordsetClient) UpdateRecordSet(ctx context.Context, zoneID, id string, opts recordsets.UpdateOptsBuilder) (*recordsets.RecordSet, error) {
	return recordsets.Update(ctx, c.client, zoneID, id, opts).Extract()
}

type recordsetErrorClient struct{ error }

// NewRecordSetErrorClient returns a RecordSetClient in which every method returns the given error.
func NewRecordSetErrorClient(e error) RecordSetClient {
	return recordsetErrorClient{e}
}

func (e recordsetErrorClient) ListRecordSets(_ context.Context, _ string, _ recordsets.ListOptsBuilder) iter.Seq2[*recordsets.RecordSet, error] {
	return func(yield func(*recordsets.RecordSet, error) bool) {
		yield(nil, e.error)
	}
}

func (e recordsetErrorClient) CreateRecordSet(_ context.Context, _ string, _ recordsets.CreateOptsBuilder) (*recordsets.RecordSet, error) {
	return nil, e.error
}

func (e recordsetErrorClient) DeleteRecordSet(_ context.Context, _, _ string) error {
	return e.error
}

func (e recordsetErrorClient) GetRecordSet(_ context.Context, _, _ string) (*recordsets.RecordSet, error) {
	return nil, e.error
}

func (e recordsetErrorClient) UpdateRecordSet(_ context.Context, _, _ string, _ recordsets.UpdateOptsBuilder) (*recordsets.RecordSet, error) {
	return nil, e.error
}
