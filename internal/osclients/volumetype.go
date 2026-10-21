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
	"slices"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumetypes"
	"github.com/gophercloud/gophercloud/v2/pagination"
	"github.com/gophercloud/utils/v2/openstack/clientconfig"
)

type VolumeTypeExt struct {
	volumetypes.VolumeType

	ProjectIDs []string
}

type VolumeTypeClient interface {
	ListVolumeTypes(ctx context.Context, listOpts volumetypes.ListOptsBuilder) iter.Seq2[*VolumeTypeExt, error]
	CreateVolumeType(ctx context.Context, opts volumetypes.CreateOptsBuilder) (*VolumeTypeExt, error)
	DeleteVolumeType(ctx context.Context, resourceID string) error
	GetVolumeType(ctx context.Context, resourceID string) (*VolumeTypeExt, error)
	UpdateVolumeType(ctx context.Context, id string, opts volumetypes.UpdateOptsBuilder) (*VolumeTypeExt, error)

	AddAccess(ctx context.Context, volumetypeID string, opts volumetypes.AddAccessOpts) error
	RemoveAccess(ctx context.Context, volumetypeID string, opts volumetypes.RemoveAccessOpts) error
	ListAccesses(ctx context.Context, volumetypeID string) ([]volumetypes.VolumeTypeAccess, error)
}

type volumetypeClient struct{ client *gophercloud.ServiceClient }

// NewVolumeTypeClient returns a new OpenStack client.
func NewVolumeTypeClient(providerClient *gophercloud.ProviderClient, providerClientOpts *clientconfig.ClientOpts) (VolumeTypeClient, error) {
	client, err := openstack.NewBlockStorageV3(providerClient, gophercloud.EndpointOpts{
		Region:       providerClientOpts.RegionName,
		Availability: clientconfig.GetEndpointType(providerClientOpts.EndpointType),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create volumetype service client: %w", err)
	}

	return &volumetypeClient{client}, nil
}

func (c volumetypeClient) ListVolumeTypes(ctx context.Context, listOpts volumetypes.ListOptsBuilder) iter.Seq2[*VolumeTypeExt, error] {
	extractVolumeTypesExt := func(r pagination.Page) ([]VolumeTypeExt, error) {
		vts, err := volumetypes.ExtractVolumeTypes(r)
		if err != nil {
			return nil, err
		}

		volumeTypeExts := make([]VolumeTypeExt, 0, len(vts))
		for _, v := range vts {
			projectIDs, err := c.getProjectIDs(ctx, v)
			if err != nil {
				return nil, err
			}

			volumeTypeExts = append(volumeTypeExts, VolumeTypeExt{VolumeType: v, ProjectIDs: projectIDs})
		}

		return volumeTypeExts, nil
	}

	pager := volumetypes.List(c.client, listOpts)
	return listResources(ctx, pager, extractVolumeTypesExt)
}

func (c volumetypeClient) CreateVolumeType(ctx context.Context, opts volumetypes.CreateOptsBuilder) (*VolumeTypeExt, error) {
	v, err := volumetypes.Create(ctx, c.client, opts).Extract()
	if err != nil {
		return nil, err
	}
	return &VolumeTypeExt{VolumeType: *v}, nil
}

func (c volumetypeClient) DeleteVolumeType(ctx context.Context, resourceID string) error {
	return volumetypes.Delete(ctx, c.client, resourceID).ExtractErr()
}

func (c volumetypeClient) GetVolumeType(ctx context.Context, resourceID string) (*VolumeTypeExt, error) {
	v, err := volumetypes.Get(ctx, c.client, resourceID).Extract()
	if err != nil {
		return nil, err
	}

	projectIDs, err := c.getProjectIDs(ctx, *v)
	if err != nil {
		return nil, err
	}

	return &VolumeTypeExt{VolumeType: *v, ProjectIDs: projectIDs}, nil
}

func (c volumetypeClient) UpdateVolumeType(ctx context.Context, id string, opts volumetypes.UpdateOptsBuilder) (*VolumeTypeExt, error) {
	v, err := volumetypes.Update(ctx, c.client, id, opts).Extract()
	if err != nil {
		return nil, err
	}

	return &VolumeTypeExt{VolumeType: *v}, nil
}

func (c volumetypeClient) AddAccess(ctx context.Context, volumetypeID string, opts volumetypes.AddAccessOpts) error {
	return volumetypes.AddAccess(ctx, c.client, volumetypeID, opts).ExtractErr()
}

func (c volumetypeClient) RemoveAccess(ctx context.Context, volumetypeID string, opts volumetypes.RemoveAccessOpts) error {
	return volumetypes.RemoveAccess(ctx, c.client, volumetypeID, opts).ExtractErr()
}

func (c volumetypeClient) ListAccesses(ctx context.Context, volumetypeID string) ([]volumetypes.VolumeTypeAccess, error) {
	p, err := volumetypes.ListAccesses(c.client, volumetypeID).AllPages(ctx)
	if err != nil {
		return nil, err
	}

	return volumetypes.ExtractAccesses(p)
}

func (c volumetypeClient) getProjectIDs(ctx context.Context, volumetype volumetypes.VolumeType) ([]string, error) {
	if volumetype.IsPublic {
		return nil, nil
	}

	volumeTypeAccesses, err := c.ListAccesses(ctx, volumetype.ID)
	if err != nil {
		return nil, err
	}

	projectIDs := make([]string, 0, len(volumeTypeAccesses))
	for _, p := range volumeTypeAccesses {
		projectIDs = append(projectIDs, p.ProjectID)
	}
	slices.Sort(projectIDs)

	return projectIDs, nil
}

type volumetypeErrorClient struct{ error }

// NewVolumeTypeErrorClient returns a VolumeTypeClient in which every method returns the given error.
func NewVolumeTypeErrorClient(e error) VolumeTypeClient {
	return volumetypeErrorClient{e}
}

func (e volumetypeErrorClient) ListVolumeTypes(_ context.Context, _ volumetypes.ListOptsBuilder) iter.Seq2[*VolumeTypeExt, error] {
	return func(yield func(*VolumeTypeExt, error) bool) {
		yield(nil, e.error)
	}
}

func (e volumetypeErrorClient) CreateVolumeType(_ context.Context, _ volumetypes.CreateOptsBuilder) (*VolumeTypeExt, error) {
	return nil, e.error
}

func (e volumetypeErrorClient) DeleteVolumeType(_ context.Context, _ string) error {
	return e.error
}

func (e volumetypeErrorClient) GetVolumeType(_ context.Context, _ string) (*VolumeTypeExt, error) {
	return nil, e.error
}

func (e volumetypeErrorClient) UpdateVolumeType(_ context.Context, _ string, _ volumetypes.UpdateOptsBuilder) (*VolumeTypeExt, error) {
	return nil, e.error
}

func (e volumetypeErrorClient) AddAccess(_ context.Context, _ string, _ volumetypes.AddAccessOpts) error {
	return e.error
}

func (e volumetypeErrorClient) RemoveAccess(_ context.Context, _ string, _ volumetypes.RemoveAccessOpts) error {
	return e.error
}

func (e volumetypeErrorClient) ListAccesses(_ context.Context, _ string) ([]volumetypes.VolumeTypeAccess, error) {
	return nil, e.error
}
