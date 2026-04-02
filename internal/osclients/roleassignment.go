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
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/roles"
	"github.com/gophercloud/utils/v2/openstack/clientconfig"
)

type RoleAssignmentClient interface {
	ListRoleAssignments(ctx context.Context, listOpts roles.ListOptsBuilder) iter.Seq2[*roles.RoleAssignment, error]
	CreateRoleAssignment(ctx context.Context, opts roles.CreateOptsBuilder) (*roles.RoleAssignment, error)
	DeleteRoleAssignment(ctx context.Context, resourceID string) error
	GetRoleAssignment(ctx context.Context, resourceID string) (*roles.RoleAssignment, error)
	UpdateRoleAssignment(ctx context.Context, id string, opts roles.UpdateOptsBuilder) (*roles.RoleAssignment, error)
}

type roleassignmentClient struct{ client *gophercloud.ServiceClient }

// NewRoleAssignmentClient returns a new OpenStack client.
func NewRoleAssignmentClient(providerClient *gophercloud.ProviderClient, providerClientOpts *clientconfig.ClientOpts) (RoleAssignmentClient, error) {
	client, err := openstack.NewIdentityV3(providerClient, gophercloud.EndpointOpts{
		Region:       providerClientOpts.RegionName,
		Availability: clientconfig.GetEndpointType(providerClientOpts.EndpointType),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create roleassignment service client: %v", err)
	}

	return &roleassignmentClient{client}, nil
}

func (c roleassignmentClient) ListRoleAssignments(ctx context.Context, listOpts roles.ListOptsBuilder) iter.Seq2[*roles.RoleAssignment, error] {
	pager := roles.List(c.client, listOpts)
	return func(yield func(*roles.RoleAssignment, error) bool) {
		_ = pager.EachPage(ctx, yieldPage(roles.ExtractRoleAssignments, yield))
	}
}

func (c roleassignmentClient) CreateRoleAssignment(ctx context.Context, opts roles.CreateOptsBuilder) (*roles.RoleAssignment, error) {
	return roles.Create(ctx, c.client, opts).Extract()
}

func (c roleassignmentClient) DeleteRoleAssignment(ctx context.Context, resourceID string) error {
	return roles.Delete(ctx, c.client, resourceID).ExtractErr()
}

func (c roleassignmentClient) GetRoleAssignment(ctx context.Context, resourceID string) (*roles.RoleAssignment, error) {
	return roles.Get(ctx, c.client, resourceID).Extract()
}

func (c roleassignmentClient) UpdateRoleAssignment(ctx context.Context, id string, opts roles.UpdateOptsBuilder) (*roles.RoleAssignment, error) {
	return roles.Update(ctx, c.client, id, opts).Extract()
}

type roleassignmentErrorClient struct{ error }

// NewRoleAssignmentErrorClient returns a RoleAssignmentClient in which every method returns the given error.
func NewRoleAssignmentErrorClient(e error) RoleAssignmentClient {
	return roleassignmentErrorClient{e}
}

func (e roleassignmentErrorClient) ListRoleAssignments(_ context.Context, _ roles.ListOptsBuilder) iter.Seq2[*roles.RoleAssignment, error] {
	return func(yield func(*roles.RoleAssignment, error) bool) {
		yield(nil, e.error)
	}
}

func (e roleassignmentErrorClient) CreateRoleAssignment(_ context.Context, _ roles.CreateOptsBuilder) (*roles.RoleAssignment, error) {
	return nil, e.error
}

func (e roleassignmentErrorClient) DeleteRoleAssignment(_ context.Context, _ string) error {
	return e.error
}

func (e roleassignmentErrorClient) GetRoleAssignment(_ context.Context, _ string) (*roles.RoleAssignment, error) {
	return nil, e.error
}

func (e roleassignmentErrorClient) UpdateRoleAssignment(_ context.Context, _ string, _ roles.UpdateOptsBuilder) (*roles.RoleAssignment, error) {
	return nil, e.error
}
