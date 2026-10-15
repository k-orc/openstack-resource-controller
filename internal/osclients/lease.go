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
	"github.com/gophercloud/gophercloud/v2/openstack/reservation/v1/leases"
	"github.com/gophercloud/utils/v2/openstack/clientconfig"
)

type LeaseClient interface {
	ListLeases(ctx context.Context, listOpts leases.ListOptsBuilder) iter.Seq2[*leases.Lease, error]
	CreateLease(ctx context.Context, opts leases.CreateOptsBuilder) (*leases.Lease, error)
	DeleteLease(ctx context.Context, resourceID string) error
	GetLease(ctx context.Context, resourceID string) (*leases.Lease, error)
	UpdateLease(ctx context.Context, id string, opts leases.UpdateOptsBuilder) (*leases.Lease, error)
}

type leaseClient struct{ client *gophercloud.ServiceClient }

// NewLeaseClient returns a new OpenStack client.
func NewLeaseClient(providerClient *gophercloud.ProviderClient, providerClientOpts *clientconfig.ClientOpts) (LeaseClient, error) {
	client, err := openstack.NewReservationV1(providerClient, gophercloud.EndpointOpts{
		Region:       providerClientOpts.RegionName,
		Availability: clientconfig.GetEndpointType(providerClientOpts.EndpointType),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create lease service client: %v", err)
	}

	return &leaseClient{client}, nil
}

func (c leaseClient) ListLeases(ctx context.Context, listOpts leases.ListOptsBuilder) iter.Seq2[*leases.Lease, error] {
	pager := leases.List(c.client, listOpts)
	return func(yield func(*leases.Lease, error) bool) {
		_ = pager.EachPage(ctx, yieldPage(leases.ExtractLeases, yield))
	}
}

func (c leaseClient) CreateLease(ctx context.Context, opts leases.CreateOptsBuilder) (*leases.Lease, error) {
	return leases.Create(ctx, c.client, opts).Extract()
}

func (c leaseClient) DeleteLease(ctx context.Context, resourceID string) error {
	return leases.Delete(ctx, c.client, resourceID).ExtractErr()
}

func (c leaseClient) GetLease(ctx context.Context, resourceID string) (*leases.Lease, error) {
	return leases.Get(ctx, c.client, resourceID).Extract()
}

func (c leaseClient) UpdateLease(ctx context.Context, id string, opts leases.UpdateOptsBuilder) (*leases.Lease, error) {
	return leases.Update(ctx, c.client, id, opts).Extract()
}

type leaseErrorClient struct{ error }

// NewLeaseErrorClient returns a LeaseClient in which every method returns the given error.
func NewLeaseErrorClient(e error) LeaseClient {
	return leaseErrorClient{e}
}

func (e leaseErrorClient) ListLeases(_ context.Context, _ leases.ListOptsBuilder) iter.Seq2[*leases.Lease, error] {
	return func(yield func(*leases.Lease, error) bool) {
		yield(nil, e.error)
	}
}

func (e leaseErrorClient) CreateLease(_ context.Context, _ leases.CreateOptsBuilder) (*leases.Lease, error) {
	return nil, e.error
}

func (e leaseErrorClient) DeleteLease(_ context.Context, _ string) error {
	return e.error
}

func (e leaseErrorClient) GetLease(_ context.Context, _ string) (*leases.Lease, error) {
	return nil, e.error
}

func (e leaseErrorClient) UpdateLease(_ context.Context, _ string, _ leases.UpdateOptsBuilder) (*leases.Lease, error) {
	return nil, e.error
}
