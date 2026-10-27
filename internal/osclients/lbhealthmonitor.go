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
	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/monitors"
	"github.com/gophercloud/utils/v2/openstack/clientconfig"
)

type LBHealthMonitorClient interface {
	ListLBHealthMonitors(ctx context.Context, listOpts monitors.ListOptsBuilder) iter.Seq2[*monitors.Monitor, error]
	CreateLBHealthMonitor(ctx context.Context, opts monitors.CreateOptsBuilder) (*monitors.Monitor, error)
	DeleteLBHealthMonitor(ctx context.Context, resourceID string) error
	GetLBHealthMonitor(ctx context.Context, resourceID string) (*monitors.Monitor, error)
	UpdateLBHealthMonitor(ctx context.Context, id string, opts monitors.UpdateOptsBuilder) (*monitors.Monitor, error)
}

type lbhealthmonitorClient struct{ client *gophercloud.ServiceClient }

// NewLBHealthMonitorClient returns a new OpenStack client.
func NewLBHealthMonitorClient(providerClient *gophercloud.ProviderClient, providerClientOpts *clientconfig.ClientOpts) (LBHealthMonitorClient, error) {
	client, err := openstack.NewLoadBalancerV2(providerClient, gophercloud.EndpointOpts{
		Region:       providerClientOpts.RegionName,
		Availability: clientconfig.GetEndpointType(providerClientOpts.EndpointType),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create lbhealthmonitor service client: %w", err)
	}

	return &lbhealthmonitorClient{client}, nil
}

func (c lbhealthmonitorClient) ListLBHealthMonitors(ctx context.Context, listOpts monitors.ListOptsBuilder) iter.Seq2[*monitors.Monitor, error] {
	pager := monitors.List(c.client, listOpts)
	return listResources(ctx, pager, monitors.ExtractMonitors)
}

func (c lbhealthmonitorClient) CreateLBHealthMonitor(ctx context.Context, opts monitors.CreateOptsBuilder) (*monitors.Monitor, error) {
	return monitors.Create(ctx, c.client, opts).Extract()
}

func (c lbhealthmonitorClient) DeleteLBHealthMonitor(ctx context.Context, resourceID string) error {
	return monitors.Delete(ctx, c.client, resourceID).ExtractErr()
}

func (c lbhealthmonitorClient) GetLBHealthMonitor(ctx context.Context, resourceID string) (*monitors.Monitor, error) {
	return monitors.Get(ctx, c.client, resourceID).Extract()
}

func (c lbhealthmonitorClient) UpdateLBHealthMonitor(ctx context.Context, id string, opts monitors.UpdateOptsBuilder) (*monitors.Monitor, error) {
	return monitors.Update(ctx, c.client, id, opts).Extract()
}

type lbhealthmonitorErrorClient struct{ error }

// NewLBHealthMonitorErrorClient returns a LBHealthMonitorClient in which every method returns the given error.
func NewLBHealthMonitorErrorClient(e error) LBHealthMonitorClient {
	return lbhealthmonitorErrorClient{e}
}

func (e lbhealthmonitorErrorClient) ListLBHealthMonitors(_ context.Context, _ monitors.ListOptsBuilder) iter.Seq2[*monitors.Monitor, error] {
	return func(yield func(*monitors.Monitor, error) bool) {
		yield(nil, e.error)
	}
}

func (e lbhealthmonitorErrorClient) CreateLBHealthMonitor(_ context.Context, _ monitors.CreateOptsBuilder) (*monitors.Monitor, error) {
	return nil, e.error
}

func (e lbhealthmonitorErrorClient) DeleteLBHealthMonitor(_ context.Context, _ string) error {
	return e.error
}

func (e lbhealthmonitorErrorClient) GetLBHealthMonitor(_ context.Context, _ string) (*monitors.Monitor, error) {
	return nil, e.error
}

func (e lbhealthmonitorErrorClient) UpdateLBHealthMonitor(_ context.Context, _ string, _ monitors.UpdateOptsBuilder) (*monitors.Monitor, error) {
	return nil, e.error
}
