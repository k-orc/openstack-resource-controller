/*
Copyright 2025 The ORC Authors.

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
	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/pools"
	"github.com/gophercloud/utils/v2/openstack/clientconfig"
)

type LBPoolClient interface {
	ListLBPools(ctx context.Context, listOpts pools.ListOptsBuilder) iter.Seq2[*pools.Pool, error]
	CreateLBPool(ctx context.Context, opts pools.CreateOptsBuilder) (*pools.Pool, error)
	DeleteLBPool(ctx context.Context, resourceID string) error
	GetLBPool(ctx context.Context, resourceID string) (*pools.Pool, error)
	UpdateLBPool(ctx context.Context, id string, opts pools.UpdateOptsBuilder) (*pools.Pool, error)

	// Member operations
	ListMembers(ctx context.Context, poolID string, opts pools.ListMembersOptsBuilder) iter.Seq2[*pools.Member, error]
	GetMember(ctx context.Context, poolID, memberID string) (*pools.Member, error)
	CreateMember(ctx context.Context, poolID string, opts pools.CreateMemberOptsBuilder) (*pools.Member, error)
	UpdateMember(ctx context.Context, poolID, memberID string, opts pools.UpdateMemberOptsBuilder) (*pools.Member, error)
	DeleteMember(ctx context.Context, poolID, memberID string) error
}

type lbpoolClient struct{ client *gophercloud.ServiceClient }

// NewLBPoolClient returns a new OpenStack client.
func NewLBPoolClient(providerClient *gophercloud.ProviderClient, providerClientOpts *clientconfig.ClientOpts) (LBPoolClient, error) {
	client, err := openstack.NewLoadBalancerV2(providerClient, gophercloud.EndpointOpts{
		Region:       providerClientOpts.RegionName,
		Availability: clientconfig.GetEndpointType(providerClientOpts.EndpointType),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create lbpool service client: %v", err)
	}

	return &lbpoolClient{client}, nil
}

func (c lbpoolClient) ListLBPools(ctx context.Context, listOpts pools.ListOptsBuilder) iter.Seq2[*pools.Pool, error] {
	pager := pools.List(c.client, listOpts)
	return func(yield func(*pools.Pool, error) bool) {
		_ = pager.EachPage(ctx, yieldPage(pools.ExtractPools, yield))
	}
}

func (c lbpoolClient) CreateLBPool(ctx context.Context, opts pools.CreateOptsBuilder) (*pools.Pool, error) {
	return pools.Create(ctx, c.client, opts).Extract()
}

func (c lbpoolClient) DeleteLBPool(ctx context.Context, resourceID string) error {
	return pools.Delete(ctx, c.client, resourceID).ExtractErr()
}

func (c lbpoolClient) GetLBPool(ctx context.Context, resourceID string) (*pools.Pool, error) {
	return pools.Get(ctx, c.client, resourceID).Extract()
}

func (c lbpoolClient) UpdateLBPool(ctx context.Context, id string, opts pools.UpdateOptsBuilder) (*pools.Pool, error) {
	return pools.Update(ctx, c.client, id, opts).Extract()
}

func (c lbpoolClient) ListMembers(ctx context.Context, poolID string, opts pools.ListMembersOptsBuilder) iter.Seq2[*pools.Member, error] {
	pager := pools.ListMembers(c.client, poolID, opts)
	return func(yield func(*pools.Member, error) bool) {
		_ = pager.EachPage(ctx, yieldPage(pools.ExtractMembers, yield))
	}
}

func (c lbpoolClient) GetMember(ctx context.Context, poolID, memberID string) (*pools.Member, error) {
	return pools.GetMember(ctx, c.client, poolID, memberID).Extract()
}

func (c lbpoolClient) CreateMember(ctx context.Context, poolID string, opts pools.CreateMemberOptsBuilder) (*pools.Member, error) {
	return pools.CreateMember(ctx, c.client, poolID, opts).Extract()
}

func (c lbpoolClient) UpdateMember(ctx context.Context, poolID, memberID string, opts pools.UpdateMemberOptsBuilder) (*pools.Member, error) {
	return pools.UpdateMember(ctx, c.client, poolID, memberID, opts).Extract()
}

func (c lbpoolClient) DeleteMember(ctx context.Context, poolID, memberID string) error {
	return pools.DeleteMember(ctx, c.client, poolID, memberID).ExtractErr()
}

type lbpoolErrorClient struct{ error }

// NewLBPoolErrorClient returns a LBPoolClient in which every method returns the given error.
func NewLBPoolErrorClient(e error) LBPoolClient {
	return lbpoolErrorClient{e}
}

func (e lbpoolErrorClient) ListLBPools(_ context.Context, _ pools.ListOptsBuilder) iter.Seq2[*pools.Pool, error] {
	return func(yield func(*pools.Pool, error) bool) {
		yield(nil, e.error)
	}
}

func (e lbpoolErrorClient) CreateLBPool(_ context.Context, _ pools.CreateOptsBuilder) (*pools.Pool, error) {
	return nil, e.error
}

func (e lbpoolErrorClient) DeleteLBPool(_ context.Context, _ string) error {
	return e.error
}

func (e lbpoolErrorClient) GetLBPool(_ context.Context, _ string) (*pools.Pool, error) {
	return nil, e.error
}

func (e lbpoolErrorClient) UpdateLBPool(_ context.Context, _ string, _ pools.UpdateOptsBuilder) (*pools.Pool, error) {
	return nil, e.error
}

func (e lbpoolErrorClient) ListMembers(_ context.Context, _ string, _ pools.ListMembersOptsBuilder) iter.Seq2[*pools.Member, error] {
	return func(yield func(*pools.Member, error) bool) {
		yield(nil, e.error)
	}
}

func (e lbpoolErrorClient) GetMember(_ context.Context, _, _ string) (*pools.Member, error) {
	return nil, e.error
}

func (e lbpoolErrorClient) CreateMember(_ context.Context, _ string, _ pools.CreateMemberOptsBuilder) (*pools.Member, error) {
	return nil, e.error
}

func (e lbpoolErrorClient) UpdateMember(_ context.Context, _, _ string, _ pools.UpdateMemberOptsBuilder) (*pools.Member, error) {
	return nil, e.error
}

func (e lbpoolErrorClient) DeleteMember(_ context.Context, _, _ string) error {
	return e.error
}
