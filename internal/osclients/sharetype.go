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
	"github.com/gophercloud/gophercloud/v2/openstack/sharedfilesystems/v2/sharetypes"
	"github.com/gophercloud/utils/v2/openstack/clientconfig"
)

type ShareTypeClient interface {
	ListShareTypes(ctx context.Context, listOpts sharetypes.ListOptsBuilder) iter.Seq2[*sharetypes.ShareType, error]
	CreateShareType(ctx context.Context, opts sharetypes.CreateOptsBuilder) (*sharetypes.ShareType, error)
	DeleteShareType(ctx context.Context, resourceID string) error
	GetShareType(ctx context.Context, resourceID string) (*sharetypes.ShareType, error)
	UpdateShareType(ctx context.Context, id string, opts sharetypes.UpdateOptsBuilder) (*sharetypes.ShareType, error)
}

type sharetypeClient struct{ client *gophercloud.ServiceClient }

// NewShareTypeClient returns a new OpenStack client.
func NewShareTypeClient(providerClient *gophercloud.ProviderClient, providerClientOpts *clientconfig.ClientOpts) (ShareTypeClient, error) {
	client, err := openstack.NewSharedFileSystemV2(providerClient, gophercloud.EndpointOpts{
		Region:       providerClientOpts.RegionName,
		Availability: clientconfig.GetEndpointType(providerClientOpts.EndpointType),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create sharetype service client: %v", err)
	}

	return &sharetypeClient{client}, nil
}

func (c sharetypeClient) ListShareTypes(ctx context.Context, listOpts sharetypes.ListOptsBuilder) iter.Seq2[*sharetypes.ShareType, error] {
	pager := sharetypes.List(c.client, listOpts)
	return func(yield func(*sharetypes.ShareType, error) bool) {
		_ = pager.EachPage(ctx, yieldPage(sharetypes.ExtractShareTypes, yield))
	}
}

func (c sharetypeClient) CreateShareType(ctx context.Context, opts sharetypes.CreateOptsBuilder) (*sharetypes.ShareType, error) {
	return sharetypes.Create(ctx, c.client, opts).Extract()
}

func (c sharetypeClient) DeleteShareType(ctx context.Context, resourceID string) error {
	return sharetypes.Delete(ctx, c.client, resourceID).ExtractErr()
}

func (c sharetypeClient) GetShareType(ctx context.Context, resourceID string) (*sharetypes.ShareType, error) {
	return sharetypes.Get(ctx, c.client, resourceID).Extract()
}

func (c sharetypeClient) UpdateShareType(ctx context.Context, id string, opts sharetypes.UpdateOptsBuilder) (*sharetypes.ShareType, error) {
	return sharetypes.Update(ctx, c.client, id, opts).Extract()
}

type sharetypeErrorClient struct{ error }

// NewShareTypeErrorClient returns a ShareTypeClient in which every method returns the given error.
func NewShareTypeErrorClient(e error) ShareTypeClient {
	return sharetypeErrorClient{e}
}

func (e sharetypeErrorClient) ListShareTypes(_ context.Context, _ sharetypes.ListOptsBuilder) iter.Seq2[*sharetypes.ShareType, error] {
	return func(yield func(*sharetypes.ShareType, error) bool) {
		yield(nil, e.error)
	}
}

func (e sharetypeErrorClient) CreateShareType(_ context.Context, _ sharetypes.CreateOptsBuilder) (*sharetypes.ShareType, error) {
	return nil, e.error
}

func (e sharetypeErrorClient) DeleteShareType(_ context.Context, _ string) error {
	return e.error
}

func (e sharetypeErrorClient) GetShareType(_ context.Context, _ string) (*sharetypes.ShareType, error) {
	return nil, e.error
}

func (e sharetypeErrorClient) UpdateShareType(_ context.Context, _ string, _ sharetypes.UpdateOptsBuilder) (*sharetypes.ShareType, error) {
	return nil, e.error
}
