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
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/limits"
	"github.com/gophercloud/utils/v2/openstack/clientconfig"
)

type LimitClient interface {
	ListLimits(ctx context.Context, listOpts limits.ListOptsBuilder) iter.Seq2[*limits.Limit, error]
	CreateLimit(ctx context.Context, opts limits.CreateOptsBuilder) (*limits.Limit, error)
	DeleteLimit(ctx context.Context, resourceID string) error
	GetLimit(ctx context.Context, resourceID string) (*limits.Limit, error)
	UpdateLimit(ctx context.Context, id string, opts limits.UpdateOptsBuilder) (*limits.Limit, error)
}

type limitClient struct{ client *gophercloud.ServiceClient }

// NewLimitClient returns a new OpenStack client.
func NewLimitClient(providerClient *gophercloud.ProviderClient, providerClientOpts *clientconfig.ClientOpts) (LimitClient, error) {
	client, err := openstack.NewIdentityV3(providerClient, gophercloud.EndpointOpts{
		Region:       providerClientOpts.RegionName,
		Availability: clientconfig.GetEndpointType(providerClientOpts.EndpointType),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create limit service client: %v", err)
	}

	return &limitClient{client}, nil
}

func (c limitClient) ListLimits(ctx context.Context, listOpts limits.ListOptsBuilder) iter.Seq2[*limits.Limit, error] {
	pager := limits.List(c.client, listOpts)
	return func(yield func(*limits.Limit, error) bool) {
		_ = pager.EachPage(ctx, yieldPage(limits.ExtractLimits, yield))
	}
}

func (c limitClient) CreateLimit(ctx context.Context, opts limits.CreateOptsBuilder) (*limits.Limit, error) {
	return limits.Create(ctx, c.client, opts).Extract()
}

func (c limitClient) DeleteLimit(ctx context.Context, resourceID string) error {
	return limits.Delete(ctx, c.client, resourceID).ExtractErr()
}

func (c limitClient) GetLimit(ctx context.Context, resourceID string) (*limits.Limit, error) {
	return limits.Get(ctx, c.client, resourceID).Extract()
}

func (c limitClient) UpdateLimit(ctx context.Context, id string, opts limits.UpdateOptsBuilder) (*limits.Limit, error) {
	return limits.Update(ctx, c.client, id, opts).Extract()
}

type limitErrorClient struct{ error }

// NewLimitErrorClient returns a LimitClient in which every method returns the given error.
func NewLimitErrorClient(e error) LimitClient {
	return limitErrorClient{e}
}

func (e limitErrorClient) ListLimits(_ context.Context, _ limits.ListOptsBuilder) iter.Seq2[*limits.Limit, error] {
	return func(yield func(*limits.Limit, error) bool) {
		yield(nil, e.error)
	}
}

func (e limitErrorClient) CreateLimit(_ context.Context, _ limits.CreateOptsBuilder) (*limits.Limit, error) {
	return nil, e.error
}

func (e limitErrorClient) DeleteLimit(_ context.Context, _ string) error {
	return e.error
}

func (e limitErrorClient) GetLimit(_ context.Context, _ string) (*limits.Limit, error) {
	return nil, e.error
}

func (e limitErrorClient) UpdateLimit(_ context.Context, _ string, _ limits.UpdateOptsBuilder) (*limits.Limit, error) {
	return nil, e.error
}
