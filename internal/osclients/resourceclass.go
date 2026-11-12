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
	"github.com/gophercloud/gophercloud/v2/openstack/placement/v1/resourceclasses"
	"github.com/gophercloud/utils/v2/openstack/clientconfig"
)

type ResourceClassClient interface {
	ListResourceClasses(ctx context.Context, listOpts resourceclasses.ListOptsBuilder) iter.Seq2[*resourceclasses.ResourceClass, error]
	CreateResourceClass(ctx context.Context, opts resourceclasses.CreateOptsBuilder) (*resourceclasses.ResourceClass, error)
	DeleteResourceClass(ctx context.Context, resourceID string) error
	GetResourceClass(ctx context.Context, resourceID string) (*resourceclasses.ResourceClass, error)
	UpdateResourceClass(ctx context.Context, id string, opts resourceclasses.UpdateOptsBuilder) (*resourceclasses.ResourceClass, error)
}

type resourceclassClient struct{ client *gophercloud.ServiceClient }

// NewResourceClassClient returns a new OpenStack client.
func NewResourceClassClient(providerClient *gophercloud.ProviderClient, providerClientOpts *clientconfig.ClientOpts) (ResourceClassClient, error) {
	client, err := openstack.PlacementV1(providerClient, gophercloud.EndpointOpts{
		Region:       providerClientOpts.RegionName,
		Availability: clientconfig.GetEndpointType(providerClientOpts.EndpointType),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create resourceclass service client: %w", err)
	}

	return &resourceclassClient{client}, nil
}

func (c resourceclassClient) ListResourceClasses(ctx context.Context, listOpts resourceclasses.ListOptsBuilder) iter.Seq2[*resourceclasses.ResourceClass, error] {
	pager := resourceclasses.List(c.client, listOpts)
	return listResources(ctx, pager, resourceclasses.ExtractResourceClasses)
}

func (c resourceclassClient) CreateResourceClass(ctx context.Context, opts resourceclasses.CreateOptsBuilder) (*resourceclasses.ResourceClass, error) {
	return resourceclasses.Create(ctx, c.client, opts).Extract()
}

func (c resourceclassClient) DeleteResourceClass(ctx context.Context, resourceID string) error {
	return resourceclasses.Delete(ctx, c.client, resourceID).ExtractErr()
}

func (c resourceclassClient) GetResourceClass(ctx context.Context, resourceID string) (*resourceclasses.ResourceClass, error) {
	return resourceclasses.Get(ctx, c.client, resourceID).Extract()
}

func (c resourceclassClient) UpdateResourceClass(ctx context.Context, id string, opts resourceclasses.UpdateOptsBuilder) (*resourceclasses.ResourceClass, error) {
	return resourceclasses.Update(ctx, c.client, id, opts).Extract()
}

type resourceclassErrorClient struct{ error }

// NewResourceClassErrorClient returns a ResourceClassClient in which every method returns the given error.
func NewResourceClassErrorClient(e error) ResourceClassClient {
	return resourceclassErrorClient{e}
}

func (e resourceclassErrorClient) ListResourceClasses(_ context.Context, _ resourceclasses.ListOptsBuilder) iter.Seq2[*resourceclasses.ResourceClass, error] {
	return func(yield func(*resourceclasses.ResourceClass, error) bool) {
		yield(nil, e.error)
	}
}

func (e resourceclassErrorClient) CreateResourceClass(_ context.Context, _ resourceclasses.CreateOptsBuilder) (*resourceclasses.ResourceClass, error) {
	return nil, e.error
}

func (e resourceclassErrorClient) DeleteResourceClass(_ context.Context, _ string) error {
	return e.error
}

func (e resourceclassErrorClient) GetResourceClass(_ context.Context, _ string) (*resourceclasses.ResourceClass, error) {
	return nil, e.error
}

func (e resourceclassErrorClient) UpdateResourceClass(_ context.Context, _ string, _ resourceclasses.UpdateOptsBuilder) (*resourceclasses.ResourceClass, error) {
	return nil, e.error
}
