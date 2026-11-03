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
	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/members"
	"github.com/gophercloud/utils/v2/openstack/clientconfig"
)

type ImageMemberClient interface {
	ListImageMembers(ctx context.Context, listOpts members.ListOptsBuilder) iter.Seq2[*members.Member, error]
	CreateImageMember(ctx context.Context, opts members.CreateOptsBuilder) (*members.Member, error)
	DeleteImageMember(ctx context.Context, resourceID string) error
	GetImageMember(ctx context.Context, resourceID string) (*members.Member, error)
	UpdateImageMember(ctx context.Context, id string, opts members.UpdateOptsBuilder) (*members.Member, error)
}

type imagememberClient struct{ client *gophercloud.ServiceClient }

// NewImageMemberClient returns a new OpenStack client.
func NewImageMemberClient(providerClient *gophercloud.ProviderClient, providerClientOpts *clientconfig.ClientOpts) (ImageMemberClient, error) {
	client, err := openstack.NewImageV2(providerClient, gophercloud.EndpointOpts{
		Region:       providerClientOpts.RegionName,
		Availability: clientconfig.GetEndpointType(providerClientOpts.EndpointType),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create imagemember service client: %w", err)
	}

	return &imagememberClient{client}, nil
}

func (c imagememberClient) ListImageMembers(ctx context.Context, listOpts members.ListOptsBuilder) iter.Seq2[*members.Member, error] {
	pager := members.List(c.client, listOpts)
	return listResources(ctx, pager, members.ExtractMembers)
}

func (c imagememberClient) CreateImageMember(ctx context.Context, opts members.CreateOptsBuilder) (*members.Member, error) {
	return members.Create(ctx, c.client, opts).Extract()
}

func (c imagememberClient) DeleteImageMember(ctx context.Context, resourceID string) error {
	return members.Delete(ctx, c.client, resourceID).ExtractErr()
}

func (c imagememberClient) GetImageMember(ctx context.Context, resourceID string) (*members.Member, error) {
	return members.Get(ctx, c.client, resourceID).Extract()
}

func (c imagememberClient) UpdateImageMember(ctx context.Context, id string, opts members.UpdateOptsBuilder) (*members.Member, error) {
	return members.Update(ctx, c.client, id, opts).Extract()
}

type imagememberErrorClient struct{ error }

// NewImageMemberErrorClient returns a ImageMemberClient in which every method returns the given error.
func NewImageMemberErrorClient(e error) ImageMemberClient {
	return imagememberErrorClient{e}
}

func (e imagememberErrorClient) ListImageMembers(_ context.Context, _ members.ListOptsBuilder) iter.Seq2[*members.Member, error] {
	return func(yield func(*members.Member, error) bool) {
		yield(nil, e.error)
	}
}

func (e imagememberErrorClient) CreateImageMember(_ context.Context, _ members.CreateOptsBuilder) (*members.Member, error) {
	return nil, e.error
}

func (e imagememberErrorClient) DeleteImageMember(_ context.Context, _ string) error {
	return e.error
}

func (e imagememberErrorClient) GetImageMember(_ context.Context, _ string) (*members.Member, error) {
	return nil, e.error
}

func (e imagememberErrorClient) UpdateImageMember(_ context.Context, _ string, _ members.UpdateOptsBuilder) (*members.Member, error) {
	return nil, e.error
}
