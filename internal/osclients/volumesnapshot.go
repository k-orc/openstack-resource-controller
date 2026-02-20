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
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/snapshots"
	"github.com/gophercloud/utils/v2/openstack/clientconfig"
)

type VolumeSnapshotClient interface {
	ListVolumeSnapshots(ctx context.Context, listOpts snapshots.ListOptsBuilder) iter.Seq2[*snapshots.Snapshot, error]
	CreateVolumeSnapshot(ctx context.Context, opts snapshots.CreateOptsBuilder) (*snapshots.Snapshot, error)
	DeleteVolumeSnapshot(ctx context.Context, resourceID string) error
	GetVolumeSnapshot(ctx context.Context, resourceID string) (*snapshots.Snapshot, error)
	UpdateVolumeSnapshot(ctx context.Context, id string, opts snapshots.UpdateOptsBuilder) (*snapshots.Snapshot, error)
}

type volumesnapshotClient struct{ client *gophercloud.ServiceClient }

// NewVolumeSnapshotClient returns a new OpenStack client.
func NewVolumeSnapshotClient(providerClient *gophercloud.ProviderClient, providerClientOpts *clientconfig.ClientOpts) (VolumeSnapshotClient, error) {
	client, err := openstack.NewBlockStorageV3(providerClient, gophercloud.EndpointOpts{
		Region:       providerClientOpts.RegionName,
		Availability: clientconfig.GetEndpointType(providerClientOpts.EndpointType),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create volumesnapshot service client: %v", err)
	}

	return &volumesnapshotClient{client}, nil
}

func (c volumesnapshotClient) ListVolumeSnapshots(ctx context.Context, listOpts snapshots.ListOptsBuilder) iter.Seq2[*snapshots.Snapshot, error] {
	pager := snapshots.List(c.client, listOpts)
	return func(yield func(*snapshots.Snapshot, error) bool) {
		_ = pager.EachPage(ctx, yieldPage(snapshots.ExtractSnapshots, yield))
	}
}

func (c volumesnapshotClient) CreateVolumeSnapshot(ctx context.Context, opts snapshots.CreateOptsBuilder) (*snapshots.Snapshot, error) {
	return snapshots.Create(ctx, c.client, opts).Extract()
}

func (c volumesnapshotClient) DeleteVolumeSnapshot(ctx context.Context, resourceID string) error {
	return snapshots.Delete(ctx, c.client, resourceID).ExtractErr()
}

func (c volumesnapshotClient) GetVolumeSnapshot(ctx context.Context, resourceID string) (*snapshots.Snapshot, error) {
	return snapshots.Get(ctx, c.client, resourceID).Extract()
}

func (c volumesnapshotClient) UpdateVolumeSnapshot(ctx context.Context, id string, opts snapshots.UpdateOptsBuilder) (*snapshots.Snapshot, error) {
	return snapshots.Update(ctx, c.client, id, opts).Extract()
}

type volumesnapshotErrorClient struct{ error }

// NewVolumeSnapshotErrorClient returns a VolumeSnapshotClient in which every method returns the given error.
func NewVolumeSnapshotErrorClient(e error) VolumeSnapshotClient {
	return volumesnapshotErrorClient{e}
}

func (e volumesnapshotErrorClient) ListVolumeSnapshots(_ context.Context, _ snapshots.ListOptsBuilder) iter.Seq2[*snapshots.Snapshot, error] {
	return func(yield func(*snapshots.Snapshot, error) bool) {
		yield(nil, e.error)
	}
}

func (e volumesnapshotErrorClient) CreateVolumeSnapshot(_ context.Context, _ snapshots.CreateOptsBuilder) (*snapshots.Snapshot, error) {
	return nil, e.error
}

func (e volumesnapshotErrorClient) DeleteVolumeSnapshot(_ context.Context, _ string) error {
	return e.error
}

func (e volumesnapshotErrorClient) GetVolumeSnapshot(_ context.Context, _ string) (*snapshots.Snapshot, error) {
	return nil, e.error
}

func (e volumesnapshotErrorClient) UpdateVolumeSnapshot(_ context.Context, _ string, _ snapshots.UpdateOptsBuilder) (*snapshots.Snapshot, error) {
	return nil, e.error
}
