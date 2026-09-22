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
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/qos/policies"
	"github.com/gophercloud/utils/v2/openstack/clientconfig"
)

type QosPolicyClient interface {
	ListQosPolicys(ctx context.Context, listOpts policies.PolicyListOptsBuilder) iter.Seq2[*policies.Policy, error]
	CreateQosPolicy(ctx context.Context, opts policies.CreateOptsBuilder) (*policies.Policy, error)
	DeleteQosPolicy(ctx context.Context, resourceID string) error
	GetQosPolicy(ctx context.Context, resourceID string) (*policies.Policy, error)
	UpdateQosPolicy(ctx context.Context, id string, opts policies.UpdateOptsBuilder) (*policies.Policy, error)
}

type qospolicyClient struct{ client *gophercloud.ServiceClient }

// NewQosPolicyClient returns a new OpenStack client.
func NewQosPolicyClient(providerClient *gophercloud.ProviderClient, providerClientOpts *clientconfig.ClientOpts) (QosPolicyClient, error) {
	client, err := openstack.NewNetworkV2(providerClient, gophercloud.EndpointOpts{
		Region:       providerClientOpts.RegionName,
		Availability: clientconfig.GetEndpointType(providerClientOpts.EndpointType),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create qospolicy service client: %v", err)
	}

	return &qospolicyClient{client}, nil
}

func (c qospolicyClient) ListQosPolicys(ctx context.Context, listOpts policies.PolicyListOptsBuilder) iter.Seq2[*policies.Policy, error] {
	pager := policies.List(c.client, listOpts)
	return func(yield func(*policies.Policy, error) bool) {
		_ = pager.EachPage(ctx, yieldPage(policies.ExtractPolicies, yield))
	}
}

func (c qospolicyClient) CreateQosPolicy(ctx context.Context, opts policies.CreateOptsBuilder) (*policies.Policy, error) {
	return policies.Create(ctx, c.client, opts).Extract()
}

func (c qospolicyClient) DeleteQosPolicy(ctx context.Context, resourceID string) error {
	return policies.Delete(ctx, c.client, resourceID).ExtractErr()
}

func (c qospolicyClient) GetQosPolicy(ctx context.Context, resourceID string) (*policies.Policy, error) {
	return policies.Get(ctx, c.client, resourceID).Extract()
}

func (c qospolicyClient) UpdateQosPolicy(ctx context.Context, id string, opts policies.UpdateOptsBuilder) (*policies.Policy, error) {
	return policies.Update(ctx, c.client, id, opts).Extract()
}

type qospolicyErrorClient struct{ error }

// NewQosPolicyErrorClient returns a QosPolicyClient in which every method returns the given error.
func NewQosPolicyErrorClient(e error) QosPolicyClient {
	return qospolicyErrorClient{e}
}

func (e qospolicyErrorClient) ListQosPolicys(_ context.Context, _ policies.PolicyListOptsBuilder) iter.Seq2[*policies.Policy, error] {
	return func(yield func(*policies.Policy, error) bool) {
		yield(nil, e.error)
	}
}

func (e qospolicyErrorClient) CreateQosPolicy(_ context.Context, _ policies.CreateOptsBuilder) (*policies.Policy, error) {
	return nil, e.error
}

func (e qospolicyErrorClient) DeleteQosPolicy(_ context.Context, _ string) error {
	return e.error
}

func (e qospolicyErrorClient) GetQosPolicy(_ context.Context, _ string) (*policies.Policy, error) {
	return nil, e.error
}

func (e qospolicyErrorClient) UpdateQosPolicy(_ context.Context, _ string, _ policies.UpdateOptsBuilder) (*policies.Policy, error) {
	return nil, e.error
}
