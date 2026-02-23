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
	"github.com/gophercloud/gophercloud/v2/openstack/sharedfilesystems/v2/shares"
	"github.com/gophercloud/utils/v2/openstack/clientconfig"
)

type ShareClient interface {
	ListShares(ctx context.Context, listOpts shares.ListOptsBuilder) iter.Seq2[*shares.Share, error]
	CreateShare(ctx context.Context, opts shares.CreateOptsBuilder) (*shares.Share, error)
	DeleteShare(ctx context.Context, resourceID string) error
	GetShare(ctx context.Context, resourceID string) (*shares.Share, error)
	UpdateShare(ctx context.Context, id string, opts shares.UpdateOptsBuilder) (*shares.Share, error)
}

type shareClient struct{ client *gophercloud.ServiceClient }

// NewShareClient returns a new OpenStack client.
func NewShareClient(providerClient *gophercloud.ProviderClient, providerClientOpts *clientconfig.ClientOpts) (ShareClient, error) {
	client, err := openstack.NewSharedFileSystemV2(providerClient, gophercloud.EndpointOpts{
		Region:       providerClientOpts.RegionName,
		Availability: clientconfig.GetEndpointType(providerClientOpts.EndpointType),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create share service client: %v", err)
	}

	return &shareClient{client}, nil
}

func (c shareClient) ListShares(ctx context.Context, listOpts shares.ListOptsBuilder) iter.Seq2[*shares.Share, error] {
	pager := shares.ListDetail(c.client, listOpts)
	return func(yield func(*shares.Share, error) bool) {
		_ = pager.EachPage(ctx, yieldPage(shares.ExtractShares, yield))
	}
}

func (c shareClient) CreateShare(ctx context.Context, opts shares.CreateOptsBuilder) (*shares.Share, error) {
	return shares.Create(ctx, c.client, opts).Extract()
}

func (c shareClient) DeleteShare(ctx context.Context, resourceID string) error {
	return shares.Delete(ctx, c.client, resourceID).ExtractErr()
}

func (c shareClient) GetShare(ctx context.Context, resourceID string) (*shares.Share, error) {
	return shares.Get(ctx, c.client, resourceID).Extract()
}

func (c shareClient) UpdateShare(ctx context.Context, id string, opts shares.UpdateOptsBuilder) (*shares.Share, error) {
	return shares.Update(ctx, c.client, id, opts).Extract()
}

type shareErrorClient struct{ error }

// NewShareErrorClient returns a ShareClient in which every method returns the given error.
func NewShareErrorClient(e error) ShareClient {
	return shareErrorClient{e}
}

func (e shareErrorClient) ListShares(_ context.Context, _ shares.ListOptsBuilder) iter.Seq2[*shares.Share, error] {
	return func(yield func(*shares.Share, error) bool) {
		yield(nil, e.error)
	}
}

func (e shareErrorClient) CreateShare(_ context.Context, _ shares.CreateOptsBuilder) (*shares.Share, error) {
	return nil, e.error
}

func (e shareErrorClient) DeleteShare(_ context.Context, _ string) error {
	return e.error
}

func (e shareErrorClient) GetShare(_ context.Context, _ string) (*shares.Share, error) {
	return nil, e.error
}

func (e shareErrorClient) UpdateShare(_ context.Context, _ string, _ shares.UpdateOptsBuilder) (*shares.Share, error) {
	return nil, e.error
}
