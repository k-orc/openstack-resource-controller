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
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/regions"
	"github.com/gophercloud/utils/v2/openstack/clientconfig"
)

type RegionClient interface {
	ListRegions(ctx context.Context, listOpts regions.ListOptsBuilder) iter.Seq2[*regions.Region, error]
	CreateRegion(ctx context.Context, opts regions.CreateOptsBuilder) (*regions.Region, error)
	DeleteRegion(ctx context.Context, resourceID string) error
	GetRegion(ctx context.Context, resourceID string) (*regions.Region, error)
	UpdateRegion(ctx context.Context, id string, opts regions.UpdateOptsBuilder) (*regions.Region, error)
}

type regionClient struct{ client *gophercloud.ServiceClient }

// NewRegionClient returns a new OpenStack client.
func NewRegionClient(providerClient *gophercloud.ProviderClient, providerClientOpts *clientconfig.ClientOpts) (RegionClient, error) {
	client, err := openstack.NewIdentityV3(providerClient, gophercloud.EndpointOpts{
		Region:       providerClientOpts.RegionName,
		Availability: clientconfig.GetEndpointType(providerClientOpts.EndpointType),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create region service client: %v", err)
	}

	return &regionClient{client}, nil
}

func (c regionClient) ListRegions(ctx context.Context, listOpts regions.ListOptsBuilder) iter.Seq2[*regions.Region, error] {
	pager := regions.List(c.client, listOpts)
	return func(yield func(*regions.Region, error) bool) {
		_ = pager.EachPage(ctx, yieldPage(regions.ExtractRegions, yield))
	}
}

func (c regionClient) CreateRegion(ctx context.Context, opts regions.CreateOptsBuilder) (*regions.Region, error) {
	return regions.Create(ctx, c.client, opts).Extract()
}

func (c regionClient) DeleteRegion(ctx context.Context, resourceID string) error {
	return regions.Delete(ctx, c.client, resourceID).ExtractErr()
}

func (c regionClient) GetRegion(ctx context.Context, resourceID string) (*regions.Region, error) {
	return regions.Get(ctx, c.client, resourceID).Extract()
}

func (c regionClient) UpdateRegion(ctx context.Context, id string, opts regions.UpdateOptsBuilder) (*regions.Region, error) {
	return regions.Update(ctx, c.client, id, opts).Extract()
}

type regionErrorClient struct{ error }

// NewRegionErrorClient returns a RegionClient in which every method returns the given error.
func NewRegionErrorClient(e error) RegionClient {
	return regionErrorClient{e}
}

func (e regionErrorClient) ListRegions(_ context.Context, _ regions.ListOptsBuilder) iter.Seq2[*regions.Region, error] {
	return func(yield func(*regions.Region, error) bool) {
		yield(nil, e.error)
	}
}

func (e regionErrorClient) CreateRegion(_ context.Context, _ regions.CreateOptsBuilder) (*regions.Region, error) {
	return nil, e.error
}

func (e regionErrorClient) DeleteRegion(_ context.Context, _ string) error {
	return e.error
}

func (e regionErrorClient) GetRegion(_ context.Context, _ string) (*regions.Region, error) {
	return nil, e.error
}

func (e regionErrorClient) UpdateRegion(_ context.Context, _ string, _ regions.UpdateOptsBuilder) (*regions.Region, error) {
	return nil, e.error
}
