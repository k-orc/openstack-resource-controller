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
	"github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/containers"
	"github.com/gophercloud/utils/v2/openstack/clientconfig"
)

type SwiftContainerClient interface {
	ListSwiftContainers(ctx context.Context, listOpts containers.ListOptsBuilder) iter.Seq2[*containers.Container, error]
	CreateSwiftContainer(ctx context.Context, opts containers.CreateOptsBuilder) (*containers.Container, error)
	DeleteSwiftContainer(ctx context.Context, resourceID string) error
	GetSwiftContainer(ctx context.Context, resourceID string) (*containers.Container, error)
	UpdateSwiftContainer(ctx context.Context, id string, opts containers.UpdateOptsBuilder) (*containers.Container, error)
}

type swiftcontainerClient struct{ client *gophercloud.ServiceClient }

// NewSwiftContainerClient returns a new OpenStack client.
func NewSwiftContainerClient(providerClient *gophercloud.ProviderClient, providerClientOpts *clientconfig.ClientOpts) (SwiftContainerClient, error) {
	client, err := openstack.NewObjectStorageV1(providerClient, gophercloud.EndpointOpts{
		Region:       providerClientOpts.RegionName,
		Availability: clientconfig.GetEndpointType(providerClientOpts.EndpointType),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create swiftcontainer service client: %v", err)
	}

	return &swiftcontainerClient{client}, nil
}

func (c swiftcontainerClient) ListSwiftContainers(ctx context.Context, listOpts containers.ListOptsBuilder) iter.Seq2[*containers.Container, error] {
	pager := containers.List(c.client, listOpts)
	return func(yield func(*containers.Container, error) bool) {
		_ = pager.EachPage(ctx, yieldPage(containers.ExtractContainers, yield))
	}
}

func (c swiftcontainerClient) CreateSwiftContainer(ctx context.Context, opts containers.CreateOptsBuilder) (*containers.Container, error) {
	return containers.Create(ctx, c.client, opts).Extract()
}

func (c swiftcontainerClient) DeleteSwiftContainer(ctx context.Context, resourceID string) error {
	return containers.Delete(ctx, c.client, resourceID).ExtractErr()
}

func (c swiftcontainerClient) GetSwiftContainer(ctx context.Context, resourceID string) (*containers.Container, error) {
	return containers.Get(ctx, c.client, resourceID).Extract()
}

func (c swiftcontainerClient) UpdateSwiftContainer(ctx context.Context, id string, opts containers.UpdateOptsBuilder) (*containers.Container, error) {
	return containers.Update(ctx, c.client, id, opts).Extract()
}

type swiftcontainerErrorClient struct{ error }

// NewSwiftContainerErrorClient returns a SwiftContainerClient in which every method returns the given error.
func NewSwiftContainerErrorClient(e error) SwiftContainerClient {
	return swiftcontainerErrorClient{e}
}

func (e swiftcontainerErrorClient) ListSwiftContainers(_ context.Context, _ containers.ListOptsBuilder) iter.Seq2[*containers.Container, error] {
	return func(yield func(*containers.Container, error) bool) {
		yield(nil, e.error)
	}
}

func (e swiftcontainerErrorClient) CreateSwiftContainer(_ context.Context, _ containers.CreateOptsBuilder) (*containers.Container, error) {
	return nil, e.error
}

func (e swiftcontainerErrorClient) DeleteSwiftContainer(_ context.Context, _ string) error {
	return e.error
}

func (e swiftcontainerErrorClient) GetSwiftContainer(_ context.Context, _ string) (*containers.Container, error) {
	return nil, e.error
}

func (e swiftcontainerErrorClient) UpdateSwiftContainer(_ context.Context, _ string, _ containers.UpdateOptsBuilder) (*containers.Container, error) {
	return nil, e.error
}
