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

// SwiftContainerClient is an interface for interacting with OpenStack Swift containers.
type SwiftContainerClient interface {
	ListContainers(ctx context.Context, listOpts containers.ListOptsBuilder) iter.Seq2[*containers.Container, error]
	CreateContainer(ctx context.Context, containerName string, opts containers.CreateOptsBuilder) (*containers.CreateHeader, error)
	GetContainer(ctx context.Context, containerName string, opts containers.GetOptsBuilder) (*containers.GetHeader, error)
	GetContainerMetadata(ctx context.Context, containerName string) (map[string]string, error)
	DeleteContainer(ctx context.Context, containerName string) error
	UpdateContainer(ctx context.Context, containerName string, opts containers.UpdateOptsBuilder) (*containers.UpdateHeader, error)
}

type swiftContainerClient struct{ client *gophercloud.ServiceClient }

// NewSwiftContainerClient returns a new OpenStack Swift object storage client.
func NewSwiftContainerClient(providerClient *gophercloud.ProviderClient, providerClientOpts *clientconfig.ClientOpts) (SwiftContainerClient, error) {
	client, err := openstack.NewObjectStorageV1(providerClient, gophercloud.EndpointOpts{
		Region:       providerClientOpts.RegionName,
		Availability: clientconfig.GetEndpointType(providerClientOpts.EndpointType),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create swiftcontainer service client: %v", err)
	}

	return NewSwiftContainerClientFromServiceClient(client), nil
}

// NewSwiftContainerClientFromServiceClient returns a SwiftContainerClient wrapping
// the given gophercloud ServiceClient. This is primarily useful for testing.
func NewSwiftContainerClientFromServiceClient(client *gophercloud.ServiceClient) SwiftContainerClient {
	return &swiftContainerClient{client}
}

func (c swiftContainerClient) ListContainers(ctx context.Context, listOpts containers.ListOptsBuilder) iter.Seq2[*containers.Container, error] {
	pager := containers.List(c.client, listOpts)
	return func(yield func(*containers.Container, error) bool) {
		_ = pager.EachPage(ctx, yieldPage(containers.ExtractInfo, yield))
	}
}

func (c swiftContainerClient) CreateContainer(ctx context.Context, containerName string, opts containers.CreateOptsBuilder) (*containers.CreateHeader, error) {
	return containers.Create(ctx, c.client, containerName, opts).Extract()
}

func (c swiftContainerClient) GetContainer(ctx context.Context, containerName string, opts containers.GetOptsBuilder) (*containers.GetHeader, error) {
	return containers.Get(ctx, c.client, containerName, opts).Extract()
}

func (c swiftContainerClient) GetContainerMetadata(ctx context.Context, containerName string) (map[string]string, error) {
	return containers.Get(ctx, c.client, containerName, nil).ExtractMetadata()
}

func (c swiftContainerClient) DeleteContainer(ctx context.Context, containerName string) error {
	_, err := containers.Delete(ctx, c.client, containerName).Extract()
	return err
}

func (c swiftContainerClient) UpdateContainer(ctx context.Context, containerName string, opts containers.UpdateOptsBuilder) (*containers.UpdateHeader, error) {
	return containers.Update(ctx, c.client, containerName, opts).Extract()
}

type swiftContainerErrorClient struct{ error }

// NewSwiftContainerErrorClient returns a SwiftContainerClient in which every method returns the given error.
func NewSwiftContainerErrorClient(e error) SwiftContainerClient {
	return swiftContainerErrorClient{e}
}

func (e swiftContainerErrorClient) ListContainers(_ context.Context, _ containers.ListOptsBuilder) iter.Seq2[*containers.Container, error] {
	return func(yield func(*containers.Container, error) bool) {
		yield(nil, e.error)
	}
}

func (e swiftContainerErrorClient) CreateContainer(_ context.Context, _ string, _ containers.CreateOptsBuilder) (*containers.CreateHeader, error) {
	return nil, e.error
}

func (e swiftContainerErrorClient) GetContainer(_ context.Context, _ string, _ containers.GetOptsBuilder) (*containers.GetHeader, error) {
	return nil, e.error
}

func (e swiftContainerErrorClient) GetContainerMetadata(_ context.Context, _ string) (map[string]string, error) {
	return nil, e.error
}

func (e swiftContainerErrorClient) DeleteContainer(_ context.Context, _ string) error {
	return e.error
}

func (e swiftContainerErrorClient) UpdateContainer(_ context.Context, _ string, _ containers.UpdateOptsBuilder) (*containers.UpdateHeader, error) {
	return nil, e.error
}
