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
	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/listeners"
	"github.com/gophercloud/utils/v2/openstack/clientconfig"
)

type ListenerClient interface {
	ListListeners(ctx context.Context, listOpts listeners.ListOptsBuilder) iter.Seq2[*listeners.Listener, error]
	CreateListener(ctx context.Context, opts listeners.CreateOptsBuilder) (*listeners.Listener, error)
	DeleteListener(ctx context.Context, resourceID string) error
	GetListener(ctx context.Context, resourceID string) (*listeners.Listener, error)
	UpdateListener(ctx context.Context, id string, opts listeners.UpdateOptsBuilder) (*listeners.Listener, error)
}

type listenerClient struct{ client *gophercloud.ServiceClient }

// NewListenerClient returns a new OpenStack client.
func NewListenerClient(providerClient *gophercloud.ProviderClient, providerClientOpts *clientconfig.ClientOpts) (ListenerClient, error) {
	client, err := openstack.NewLoadBalancerV2(providerClient, gophercloud.EndpointOpts{
		Region:       providerClientOpts.RegionName,
		Availability: clientconfig.GetEndpointType(providerClientOpts.EndpointType),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create listener service client: %v", err)
	}

	return &listenerClient{client}, nil
}

func (c listenerClient) ListListeners(ctx context.Context, listOpts listeners.ListOptsBuilder) iter.Seq2[*listeners.Listener, error] {
	pager := listeners.List(c.client, listOpts)
	return func(yield func(*listeners.Listener, error) bool) {
		_ = pager.EachPage(ctx, yieldPage(listeners.ExtractListeners, yield))
	}
}

func (c listenerClient) CreateListener(ctx context.Context, opts listeners.CreateOptsBuilder) (*listeners.Listener, error) {
	return listeners.Create(ctx, c.client, opts).Extract()
}

func (c listenerClient) DeleteListener(ctx context.Context, resourceID string) error {
	return listeners.Delete(ctx, c.client, resourceID).ExtractErr()
}

func (c listenerClient) GetListener(ctx context.Context, resourceID string) (*listeners.Listener, error) {
	return listeners.Get(ctx, c.client, resourceID).Extract()
}

func (c listenerClient) UpdateListener(ctx context.Context, id string, opts listeners.UpdateOptsBuilder) (*listeners.Listener, error) {
	return listeners.Update(ctx, c.client, id, opts).Extract()
}

type listenerErrorClient struct{ error }

// NewListenerErrorClient returns a ListenerClient in which every method returns the given error.
func NewListenerErrorClient(e error) ListenerClient {
	return listenerErrorClient{e}
}

func (e listenerErrorClient) ListListeners(_ context.Context, _ listeners.ListOptsBuilder) iter.Seq2[*listeners.Listener, error] {
	return func(yield func(*listeners.Listener, error) bool) {
		yield(nil, e.error)
	}
}

func (e listenerErrorClient) CreateListener(_ context.Context, _ listeners.CreateOptsBuilder) (*listeners.Listener, error) {
	return nil, e.error
}

func (e listenerErrorClient) DeleteListener(_ context.Context, _ string) error {
	return e.error
}

func (e listenerErrorClient) GetListener(_ context.Context, _ string) (*listeners.Listener, error) {
	return nil, e.error
}

func (e listenerErrorClient) UpdateListener(_ context.Context, _ string, _ listeners.UpdateOptsBuilder) (*listeners.Listener, error) {
	return nil, e.error
}
