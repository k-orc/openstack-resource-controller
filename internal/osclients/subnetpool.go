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
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/subnetpools"
	"github.com/gophercloud/utils/v2/openstack/clientconfig"
)

type SubnetPoolClient interface {
	ListSubnetPools(ctx context.Context, listOpts subnetpools.ListOptsBuilder) iter.Seq2[*subnetpools.SubnetPool, error]
	CreateSubnetPool(ctx context.Context, opts subnetpools.CreateOptsBuilder) (*subnetpools.SubnetPool, error)
	DeleteSubnetPool(ctx context.Context, resourceID string) error
	GetSubnetPool(ctx context.Context, resourceID string) (*subnetpools.SubnetPool, error)
	UpdateSubnetPool(ctx context.Context, id string, opts subnetpools.UpdateOptsBuilder) (*subnetpools.SubnetPool, error)
}

type subnetpoolClient struct{ client *gophercloud.ServiceClient }

// NewSubnetPoolClient returns a new OpenStack client.
func NewSubnetPoolClient(providerClient *gophercloud.ProviderClient, providerClientOpts *clientconfig.ClientOpts) (SubnetPoolClient, error) {
	client, err := openstack.NewNetworkV2(providerClient, gophercloud.EndpointOpts{
		Region:       providerClientOpts.RegionName,
		Availability: clientconfig.GetEndpointType(providerClientOpts.EndpointType),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create subnetpool service client: %v", err)
	}

	return &subnetpoolClient{client}, nil
}

func (c subnetpoolClient) ListSubnetPools(ctx context.Context, listOpts subnetpools.ListOptsBuilder) iter.Seq2[*subnetpools.SubnetPool, error] {
	pager := subnetpools.List(c.client, listOpts)
	return func(yield func(*subnetpools.SubnetPool, error) bool) {
		_ = pager.EachPage(ctx, yieldPage(subnetpools.ExtractSubnetPools, yield))
	}
}

func (c subnetpoolClient) CreateSubnetPool(ctx context.Context, opts subnetpools.CreateOptsBuilder) (*subnetpools.SubnetPool, error) {
	return subnetpools.Create(ctx, c.client, opts).Extract()
}

func (c subnetpoolClient) DeleteSubnetPool(ctx context.Context, resourceID string) error {
	return subnetpools.Delete(ctx, c.client, resourceID).ExtractErr()
}

func (c subnetpoolClient) GetSubnetPool(ctx context.Context, resourceID string) (*subnetpools.SubnetPool, error) {
	return subnetpools.Get(ctx, c.client, resourceID).Extract()
}

func (c subnetpoolClient) UpdateSubnetPool(ctx context.Context, id string, opts subnetpools.UpdateOptsBuilder) (*subnetpools.SubnetPool, error) {
	return subnetpools.Update(ctx, c.client, id, opts).Extract()
}

type subnetpoolErrorClient struct{ error }

// NewSubnetPoolErrorClient returns a SubnetPoolClient in which every method returns the given error.
func NewSubnetPoolErrorClient(e error) SubnetPoolClient {
	return subnetpoolErrorClient{e}
}

func (e subnetpoolErrorClient) ListSubnetPools(_ context.Context, _ subnetpools.ListOptsBuilder) iter.Seq2[*subnetpools.SubnetPool, error] {
	return func(yield func(*subnetpools.SubnetPool, error) bool) {
		yield(nil, e.error)
	}
}

func (e subnetpoolErrorClient) CreateSubnetPool(_ context.Context, _ subnetpools.CreateOptsBuilder) (*subnetpools.SubnetPool, error) {
	return nil, e.error
}

func (e subnetpoolErrorClient) DeleteSubnetPool(_ context.Context, _ string) error {
	return e.error
}

func (e subnetpoolErrorClient) GetSubnetPool(_ context.Context, _ string) (*subnetpools.SubnetPool, error) {
	return nil, e.error
}

func (e subnetpoolErrorClient) UpdateSubnetPool(_ context.Context, _ string, _ subnetpools.UpdateOptsBuilder) (*subnetpools.SubnetPool, error) {
	return nil, e.error
}
