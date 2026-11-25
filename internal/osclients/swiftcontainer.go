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
	"strings"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/containers"
	"github.com/gophercloud/gophercloud/v2/pagination"
	"github.com/gophercloud/utils/v2/openstack/clientconfig"
)

// SwiftContainer is a Swift container. Swift returns container attributes as
// response headers, which gophercloud does not combine with the name.
type SwiftContainer struct {
	Name string

	// Header is nil until the container has been read, e.g. after it is
	// listed or created.
	Header *containers.GetHeader

	// Metadata contains the X-Container-Meta-* headers, except TempURL keys.
	Metadata map[string]string
}

type SwiftContainerClient interface {
	ListSwiftContainers(ctx context.Context, listOpts containers.ListOptsBuilder) iter.Seq2[*SwiftContainer, error]
	CreateSwiftContainer(ctx context.Context, name string, opts containers.CreateOptsBuilder) (*SwiftContainer, error)
	DeleteSwiftContainer(ctx context.Context, name string) error
	GetSwiftContainer(ctx context.Context, name string) (*SwiftContainer, error)
}

type swiftcontainerClient struct{ client *gophercloud.ServiceClient }

// NewSwiftContainerClient returns a new OpenStack client.
func NewSwiftContainerClient(providerClient *gophercloud.ProviderClient, providerClientOpts *clientconfig.ClientOpts) (SwiftContainerClient, error) {
	client, err := openstack.NewObjectStorageV1(providerClient, gophercloud.EndpointOpts{
		Region:       providerClientOpts.RegionName,
		Availability: clientconfig.GetEndpointType(providerClientOpts.EndpointType),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create swiftcontainer service client: %w", err)
	}

	return &swiftcontainerClient{client}, nil
}

func (c swiftcontainerClient) ListSwiftContainers(ctx context.Context, listOpts containers.ListOptsBuilder) iter.Seq2[*SwiftContainer, error] {
	pager := containers.List(c.client, listOpts)
	return listResources(ctx, pager, func(page pagination.Page) ([]SwiftContainer, error) {
		names, err := containers.ExtractNames(page)
		if err != nil {
			return nil, err
		}
		result := make([]SwiftContainer, len(names))
		for i := range names {
			result[i].Name = names[i]
		}
		return result, nil
	})
}

// CreateSwiftContainer creates a container. The response to a Swift PUT does
// not contain the container's attributes, so only the name is returned.
func (c swiftcontainerClient) CreateSwiftContainer(ctx context.Context, name string, opts containers.CreateOptsBuilder) (*SwiftContainer, error) {
	if err := containers.Create(ctx, c.client, name, opts).Err; err != nil {
		return nil, err
	}
	return &SwiftContainer{Name: name}, nil
}

func (c swiftcontainerClient) DeleteSwiftContainer(ctx context.Context, name string) error {
	return containers.Delete(ctx, c.client, name).Err
}

func (c swiftcontainerClient) GetSwiftContainer(ctx context.Context, name string) (*SwiftContainer, error) {
	return newSwiftContainer(name, containers.Get(ctx, c.client, name, nil))
}

func newSwiftContainer(name string, result containers.GetResult) (*SwiftContainer, error) {
	header, err := result.Extract()
	if err != nil {
		return nil, err
	}
	metadata, err := result.ExtractMetadata()
	if err != nil {
		return nil, err
	}

	// TempURL keys are secrets used to sign URLs. Don't expose them.
	header.TempURLKey, header.TempURLKey2 = "", ""
	for key := range metadata {
		if strings.EqualFold(key, "Temp-URL-Key") || strings.EqualFold(key, "Temp-URL-Key-2") {
			delete(metadata, key)
		}
	}

	return &SwiftContainer{Name: name, Header: header, Metadata: metadata}, nil
}

type swiftcontainerErrorClient struct{ error }

// NewSwiftContainerErrorClient returns a SwiftContainerClient in which every method returns the given error.
func NewSwiftContainerErrorClient(e error) SwiftContainerClient {
	return swiftcontainerErrorClient{e}
}

func (e swiftcontainerErrorClient) ListSwiftContainers(_ context.Context, _ containers.ListOptsBuilder) iter.Seq2[*SwiftContainer, error] {
	return func(yield func(*SwiftContainer, error) bool) {
		yield(nil, e.error)
	}
}

func (e swiftcontainerErrorClient) CreateSwiftContainer(_ context.Context, _ string, _ containers.CreateOptsBuilder) (*SwiftContainer, error) {
	return nil, e.error
}

func (e swiftcontainerErrorClient) DeleteSwiftContainer(_ context.Context, _ string) error {
	return e.error
}

func (e swiftcontainerErrorClient) GetSwiftContainer(_ context.Context, _ string) (*SwiftContainer, error) {
	return nil, e.error
}
