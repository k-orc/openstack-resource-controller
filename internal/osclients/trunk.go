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
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/trunks"
	"github.com/gophercloud/utils/v2/openstack/clientconfig"
)

type TrunkClient interface {
	ListTrunks(ctx context.Context, listOpts trunks.ListOptsBuilder) iter.Seq2[*trunks.Trunk, error]
	CreateTrunk(ctx context.Context, opts trunks.CreateOptsBuilder) (*trunks.Trunk, error)
	DeleteTrunk(ctx context.Context, resourceID string) error
	GetTrunk(ctx context.Context, resourceID string) (*trunks.Trunk, error)
	UpdateTrunk(ctx context.Context, id string, opts trunks.UpdateOptsBuilder) (*trunks.Trunk, error)
}

type trunkClient struct{ client *gophercloud.ServiceClient }

// NewTrunkClient returns a new OpenStack client.
func NewTrunkClient(providerClient *gophercloud.ProviderClient, providerClientOpts *clientconfig.ClientOpts) (TrunkClient, error) {
	client, err := openstack.NewNetworkV2(providerClient, gophercloud.EndpointOpts{
		Region:       providerClientOpts.RegionName,
		Availability: clientconfig.GetEndpointType(providerClientOpts.EndpointType),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create trunk service client: %v", err)
	}

	return &trunkClient{client}, nil
}

func (c trunkClient) ListTrunks(ctx context.Context, listOpts trunks.ListOptsBuilder) iter.Seq2[*trunks.Trunk, error] {
	pager := trunks.List(c.client, listOpts)
	return func(yield func(*trunks.Trunk, error) bool) {
		_ = pager.EachPage(ctx, yieldPage(trunks.ExtractTrunks, yield))
	}
}

func (c trunkClient) CreateTrunk(ctx context.Context, opts trunks.CreateOptsBuilder) (*trunks.Trunk, error) {
	return trunks.Create(ctx, c.client, opts).Extract()
}

func (c trunkClient) DeleteTrunk(ctx context.Context, resourceID string) error {
	return trunks.Delete(ctx, c.client, resourceID).ExtractErr()
}

func (c trunkClient) GetTrunk(ctx context.Context, resourceID string) (*trunks.Trunk, error) {
	return trunks.Get(ctx, c.client, resourceID).Extract()
}

func (c trunkClient) UpdateTrunk(ctx context.Context, id string, opts trunks.UpdateOptsBuilder) (*trunks.Trunk, error) {
	return trunks.Update(ctx, c.client, id, opts).Extract()
}

type trunkErrorClient struct{ error }

// NewTrunkErrorClient returns a TrunkClient in which every method returns the given error.
func NewTrunkErrorClient(e error) TrunkClient {
	return trunkErrorClient{e}
}

func (e trunkErrorClient) ListTrunks(_ context.Context, _ trunks.ListOptsBuilder) iter.Seq2[*trunks.Trunk, error] {
	return func(yield func(*trunks.Trunk, error) bool) {
		yield(nil, e.error)
	}
}

func (e trunkErrorClient) CreateTrunk(_ context.Context, _ trunks.CreateOptsBuilder) (*trunks.Trunk, error) {
	return nil, e.error
}

func (e trunkErrorClient) DeleteTrunk(_ context.Context, _ string) error {
	return e.error
}

func (e trunkErrorClient) GetTrunk(_ context.Context, _ string) (*trunks.Trunk, error) {
	return nil, e.error
}

func (e trunkErrorClient) UpdateTrunk(_ context.Context, _ string, _ trunks.UpdateOptsBuilder) (*trunks.Trunk, error) {
	return nil, e.error
}
