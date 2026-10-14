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
	"slices"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/groups"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/users"
	"github.com/gophercloud/gophercloud/v2/pagination"
	"github.com/gophercloud/utils/v2/openstack/clientconfig"
)

type GroupExt struct {
	groups.Group

	UserIDs []string
}

type GroupClient interface {
	ListGroups(ctx context.Context, listOpts groups.ListOptsBuilder) iter.Seq2[*GroupExt, error]
	CreateGroup(ctx context.Context, opts groups.CreateOptsBuilder) (*GroupExt, error)
	DeleteGroup(ctx context.Context, resourceID string) error
	GetGroup(ctx context.Context, resourceID string) (*GroupExt, error)
	UpdateGroup(ctx context.Context, id string, opts groups.UpdateOptsBuilder) (*GroupExt, error)

	AddUser(ctx context.Context, groupID, userID string) error
	RemoveUser(ctx context.Context, groupID, userID string) error
	ListUsers(ctx context.Context, groupID string) ([]users.User, error)
}

type groupClient struct{ client *gophercloud.ServiceClient }

// NewGroupClient returns a new OpenStack client.
func NewGroupClient(providerClient *gophercloud.ProviderClient, providerClientOpts *clientconfig.ClientOpts) (GroupClient, error) {
	client, err := openstack.NewIdentityV3(providerClient, gophercloud.EndpointOpts{
		Region:       providerClientOpts.RegionName,
		Availability: clientconfig.GetEndpointType(providerClientOpts.EndpointType),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create group service client: %w", err)
	}

	return &groupClient{client}, nil
}

func (c groupClient) ListGroups(ctx context.Context, listOpts groups.ListOptsBuilder) iter.Seq2[*GroupExt, error] {
	extractGroupsExt := func(r pagination.Page) ([]GroupExt, error) {
		gps, err := groups.ExtractGroups(r)
		if err != nil {
			return nil, err
		}

		groupExts := make([]GroupExt, 0, len(gps))
		for _, g := range gps {
			groupExts = append(groupExts, GroupExt{Group: g})
		}

		return groupExts, nil
	}
	pager := groups.List(c.client, listOpts)
	return func(yield func(*GroupExt, error) bool) {
		_ = pager.EachPage(ctx, yieldPage(extractGroupsExt, yield))
	}
}

func (c groupClient) CreateGroup(ctx context.Context, opts groups.CreateOptsBuilder) (*GroupExt, error) {
	g, err := groups.Create(ctx, c.client, opts).Extract()
	if err != nil {
		return nil, err
	}
	return &GroupExt{Group: *g}, nil
}

func (c groupClient) DeleteGroup(ctx context.Context, resourceID string) error {
	return groups.Delete(ctx, c.client, resourceID).ExtractErr()
}

func (c groupClient) GetGroup(ctx context.Context, resourceID string) (*GroupExt, error) {
	g, err := groups.Get(ctx, c.client, resourceID).Extract()
	if err != nil {
		return nil, err
	}

	p, err := users.ListInGroup(c.client, resourceID, users.ListOpts{}).AllPages(ctx)
	if err != nil {
		return nil, err
	}
	groupUsers, err := users.ExtractUsers(p)
	if err != nil {
		return nil, err
	}

	userIDs := make([]string, 0, len(groupUsers))
	for _, u := range groupUsers {
		userIDs = append(userIDs, u.ID)
	}
	slices.Sort(userIDs)

	return &GroupExt{Group: *g, UserIDs: userIDs}, nil
}

func (c groupClient) UpdateGroup(ctx context.Context, id string, opts groups.UpdateOptsBuilder) (*GroupExt, error) {
	g, err := groups.Update(ctx, c.client, id, opts).Extract()
	if err != nil {
		return nil, err
	}

	return &GroupExt{Group: *g}, nil
}

func (c groupClient) AddUser(ctx context.Context, groupID, userID string) error {
	return users.AddToGroup(ctx, c.client, groupID, userID).ExtractErr()
}

func (c groupClient) RemoveUser(ctx context.Context, groupID, userID string) error {
	return users.RemoveFromGroup(ctx, c.client, groupID, userID).ExtractErr()
}

func (c groupClient) ListUsers(ctx context.Context, groupID string) ([]users.User, error) {
	pager, err := users.ListInGroup(c.client, groupID, users.ListOpts{}).AllPages(ctx)
	if err != nil {
		return nil, err
	}

	return users.ExtractUsers(pager)
}

type groupErrorClient struct{ error }

// NewGroupErrorClient returns a GroupClient in which every method returns the given error.
func NewGroupErrorClient(e error) GroupClient {
	return groupErrorClient{e}
}

func (e groupErrorClient) ListGroups(_ context.Context, _ groups.ListOptsBuilder) iter.Seq2[*GroupExt, error] {
	return func(yield func(*GroupExt, error) bool) {
		yield(nil, e.error)
	}
}

func (e groupErrorClient) CreateGroup(_ context.Context, _ groups.CreateOptsBuilder) (*GroupExt, error) {
	return nil, e.error
}

func (e groupErrorClient) DeleteGroup(_ context.Context, _ string) error {
	return e.error
}

func (e groupErrorClient) GetGroup(_ context.Context, _ string) (*GroupExt, error) {
	return nil, e.error
}

func (e groupErrorClient) UpdateGroup(_ context.Context, _ string, _ groups.UpdateOptsBuilder) (*GroupExt, error) {
	return nil, e.error
}

func (e groupErrorClient) AddUser(_ context.Context, _, _ string) error {
	return e.error
}

func (e groupErrorClient) RemoveUser(_ context.Context, _, _ string) error {
	return e.error
}

func (e groupErrorClient) ListUsers(_ context.Context, _ string) ([]users.User, error) {
	return nil, e.error
}
