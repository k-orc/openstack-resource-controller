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

const PlacementMinimumMicroversion = "1.2"

type ResourceClassClient interface {
	ListResourceClasses(ctx context.Context) iter.Seq2[*resourceclasses.ResourceClass, error]
	CreateResourceClass(ctx context.Context, opts resourceclasses.CreateOptsBuilder) (*resourceclasses.ResourceClass, error)
	DeleteResourceClass(ctx context.Context, resourceID string) error
	GetResourceClass(ctx context.Context, resourceID string) (*resourceclasses.ResourceClass, error)
}

type resourceclassClient struct{ client *gophercloud.ServiceClient }

// NewResourceClassClient returns a new OpenStack client.
func NewResourceClassClient(providerClient *gophercloud.ProviderClient, providerClientOpts *clientconfig.ClientOpts) (ResourceClassClient, error) {
	client, err := openstack.NewPlacementV1(providerClient, gophercloud.EndpointOpts{
		Region:       providerClientOpts.RegionName,
		Availability: clientconfig.GetEndpointType(providerClientOpts.EndpointType),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create resourceclass service client: %w", err)
	}
	client.Microversion = PlacementMinimumMicroversion

	return &resourceclassClient{client}, nil
}

func (c resourceclassClient) ListResourceClasses(ctx context.Context) iter.Seq2[*resourceclasses.ResourceClass, error] {
	pager := resourceclasses.List(c.client)
	return listResources(ctx, pager, resourceclasses.ExtractResourceClasses)
}

func (c resourceclassClient) CreateResourceClass(ctx context.Context, opts resourceclasses.CreateOptsBuilder) (*resourceclasses.ResourceClass, error) {
	err := resourceclasses.Create(ctx, c.client, opts).ExtractErr()
	if err != nil {
		return nil, err
	}

	optsMap, err := opts.ToResourceClassCreateMap()
	if err != nil {
		return nil, err
	}

	return c.GetResourceClass(ctx, optsMap["name"].(string))
}

func (c resourceclassClient) DeleteResourceClass(ctx context.Context, resourceID string) error {
	return resourceclasses.Delete(ctx, c.client, resourceID).ExtractErr()
}

func (c resourceclassClient) GetResourceClass(ctx context.Context, resourceID string) (*resourceclasses.ResourceClass, error) {
	return resourceclasses.Get(ctx, c.client, resourceID).Extract()
}

type resourceclassErrorClient struct{ error }

// NewResourceClassErrorClient returns a ResourceClassClient in which every method returns the given error.
func NewResourceClassErrorClient(e error) ResourceClassClient {
	return resourceclassErrorClient{e}
}

func (e resourceclassErrorClient) ListResourceClasses(_ context.Context) iter.Seq2[*resourceclasses.ResourceClass, error] {
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
