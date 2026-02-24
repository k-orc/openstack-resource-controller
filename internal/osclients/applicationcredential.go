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
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/applicationcredentials"
	"github.com/gophercloud/utils/v2/openstack/clientconfig"
)

type ApplicationCredentialClient interface {
	ListApplicationCredentials(ctx context.Context, listOpts applicationcredentials.ListOptsBuilder) iter.Seq2[*applicationcredentials.ApplicationCredential, error]
	CreateApplicationCredential(ctx context.Context, opts applicationcredentials.CreateOptsBuilder) (*applicationcredentials.ApplicationCredential, error)
	DeleteApplicationCredential(ctx context.Context, resourceID string) error
	GetApplicationCredential(ctx context.Context, resourceID string) (*applicationcredentials.ApplicationCredential, error)
	UpdateApplicationCredential(ctx context.Context, id string, opts applicationcredentials.UpdateOptsBuilder) (*applicationcredentials.ApplicationCredential, error)
}

type applicationcredentialClient struct{ client *gophercloud.ServiceClient }

// NewApplicationCredentialClient returns a new OpenStack client.
func NewApplicationCredentialClient(providerClient *gophercloud.ProviderClient, providerClientOpts *clientconfig.ClientOpts) (ApplicationCredentialClient, error) {
	client, err := openstack.NewIdentityV3(providerClient, gophercloud.EndpointOpts{
		Region:       providerClientOpts.RegionName,
		Availability: clientconfig.GetEndpointType(providerClientOpts.EndpointType),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create applicationcredential service client: %v", err)
	}

	return &applicationcredentialClient{client}, nil
}

func (c applicationcredentialClient) ListApplicationCredentials(ctx context.Context, listOpts applicationcredentials.ListOptsBuilder) iter.Seq2[*applicationcredentials.ApplicationCredential, error] {
	pager := applicationcredentials.List(c.client, listOpts)
	return func(yield func(*applicationcredentials.ApplicationCredential, error) bool) {
		_ = pager.EachPage(ctx, yieldPage(applicationcredentials.ExtractApplicationCredentials, yield))
	}
}

func (c applicationcredentialClient) CreateApplicationCredential(ctx context.Context, opts applicationcredentials.CreateOptsBuilder) (*applicationcredentials.ApplicationCredential, error) {
	return applicationcredentials.Create(ctx, c.client, opts).Extract()
}

func (c applicationcredentialClient) DeleteApplicationCredential(ctx context.Context, resourceID string) error {
	return applicationcredentials.Delete(ctx, c.client, resourceID).ExtractErr()
}

func (c applicationcredentialClient) GetApplicationCredential(ctx context.Context, resourceID string) (*applicationcredentials.ApplicationCredential, error) {
	return applicationcredentials.Get(ctx, c.client, resourceID).Extract()
}

func (c applicationcredentialClient) UpdateApplicationCredential(ctx context.Context, id string, opts applicationcredentials.UpdateOptsBuilder) (*applicationcredentials.ApplicationCredential, error) {
	return applicationcredentials.Update(ctx, c.client, id, opts).Extract()
}

type applicationcredentialErrorClient struct{ error }

// NewApplicationCredentialErrorClient returns a ApplicationCredentialClient in which every method returns the given error.
func NewApplicationCredentialErrorClient(e error) ApplicationCredentialClient {
	return applicationcredentialErrorClient{e}
}

func (e applicationcredentialErrorClient) ListApplicationCredentials(_ context.Context, _ applicationcredentials.ListOptsBuilder) iter.Seq2[*applicationcredentials.ApplicationCredential, error] {
	return func(yield func(*applicationcredentials.ApplicationCredential, error) bool) {
		yield(nil, e.error)
	}
}

func (e applicationcredentialErrorClient) CreateApplicationCredential(_ context.Context, _ applicationcredentials.CreateOptsBuilder) (*applicationcredentials.ApplicationCredential, error) {
	return nil, e.error
}

func (e applicationcredentialErrorClient) DeleteApplicationCredential(_ context.Context, _ string) error {
	return e.error
}

func (e applicationcredentialErrorClient) GetApplicationCredential(_ context.Context, _ string) (*applicationcredentials.ApplicationCredential, error) {
	return nil, e.error
}

func (e applicationcredentialErrorClient) UpdateApplicationCredential(_ context.Context, _ string, _ applicationcredentials.UpdateOptsBuilder) (*applicationcredentials.ApplicationCredential, error) {
	return nil, e.error
}
