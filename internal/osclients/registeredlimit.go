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
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/registeredlimits"
	"github.com/gophercloud/utils/v2/openstack/clientconfig"
)

type RegisteredLimitClient interface {
	ListRegisteredLimits(ctx context.Context, listOpts registeredlimits.ListOptsBuilder) iter.Seq2[*registeredlimits.RegisteredLimit, error]
	CreateRegisteredLimit(ctx context.Context, opts registeredlimits.BatchCreateOptsBuilder) (*registeredlimits.RegisteredLimit, error)
	DeleteRegisteredLimit(ctx context.Context, resourceID string) error
	GetRegisteredLimit(ctx context.Context, resourceID string) (*registeredlimits.RegisteredLimit, error)
	UpdateRegisteredLimit(ctx context.Context, id string, opts registeredlimits.UpdateOptsBuilder) (*registeredlimits.RegisteredLimit, error)
}

type registeredlimitClient struct{ client *gophercloud.ServiceClient }

// NewRegisteredLimitClient returns a new OpenStack client.
func NewRegisteredLimitClient(providerClient *gophercloud.ProviderClient, providerClientOpts *clientconfig.ClientOpts) (RegisteredLimitClient, error) {
	client, err := openstack.NewIdentityV3(providerClient, gophercloud.EndpointOpts{
		Region:       providerClientOpts.RegionName,
		Availability: clientconfig.GetEndpointType(providerClientOpts.EndpointType),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create registeredlimit service client: %v", err)
	}

	return &registeredlimitClient{client}, nil
}

func (c registeredlimitClient) ListRegisteredLimits(ctx context.Context, listOpts registeredlimits.ListOptsBuilder) iter.Seq2[*registeredlimits.RegisteredLimit, error] {
	pager := registeredlimits.List(c.client, listOpts)
	return func(yield func(*registeredlimits.RegisteredLimit, error) bool) {
		_ = pager.EachPage(ctx, yieldPage(registeredlimits.ExtractRegisteredLimits, yield))
	}
}

func (c registeredlimitClient) CreateRegisteredLimit(ctx context.Context, opts registeredlimits.BatchCreateOptsBuilder) (*registeredlimits.RegisteredLimit, error) {
	batch, err := registeredlimits.BatchCreate(ctx, c.client, opts).Extract()
	return &batch[0], err
}

func (c registeredlimitClient) DeleteRegisteredLimit(ctx context.Context, resourceID string) error {
	return registeredlimits.Delete(ctx, c.client, resourceID).ExtractErr()
}

func (c registeredlimitClient) GetRegisteredLimit(ctx context.Context, resourceID string) (*registeredlimits.RegisteredLimit, error) {
	return registeredlimits.Get(ctx, c.client, resourceID).Extract()
}

func (c registeredlimitClient) UpdateRegisteredLimit(ctx context.Context, id string, opts registeredlimits.UpdateOptsBuilder) (*registeredlimits.RegisteredLimit, error) {
	return registeredlimits.Update(ctx, c.client, id, opts).Extract()
}

type registeredlimitErrorClient struct{ error }

// NewRegisteredLimitErrorClient returns a RegisteredLimitClient in which every method returns the given error.
func NewRegisteredLimitErrorClient(e error) RegisteredLimitClient {
	return registeredlimitErrorClient{e}
}

func (e registeredlimitErrorClient) ListRegisteredLimits(_ context.Context, _ registeredlimits.ListOptsBuilder) iter.Seq2[*registeredlimits.RegisteredLimit, error] {
	return func(yield func(*registeredlimits.RegisteredLimit, error) bool) {
		yield(nil, e.error)
	}
}

func (e registeredlimitErrorClient) CreateRegisteredLimit(_ context.Context, _ registeredlimits.BatchCreateOptsBuilder) (*registeredlimits.RegisteredLimit, error) {
	return nil, e.error
}

func (e registeredlimitErrorClient) DeleteRegisteredLimit(_ context.Context, _ string) error {
	return e.error
}

func (e registeredlimitErrorClient) GetRegisteredLimit(_ context.Context, _ string) (*registeredlimits.RegisteredLimit, error) {
	return nil, e.error
}

func (e registeredlimitErrorClient) UpdateRegisteredLimit(_ context.Context, _ string, _ registeredlimits.UpdateOptsBuilder) (*registeredlimits.RegisteredLimit, error) {
	return nil, e.error
}
