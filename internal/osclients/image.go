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
	"errors"
	"fmt"
	"io"
	"iter"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"
	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/imagedata"
	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/imageimport"
	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/images"
	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/members"
	"github.com/gophercloud/utils/v2/openstack/clientconfig"
)

type ImageClient interface {
	ListImages(ctx context.Context, listOpts images.ListOptsBuilder) iter.Seq2[*images.Image, error]
	GetImage(ctx context.Context, id string) (*images.Image, error)
	CreateImage(ctx context.Context, createOpts images.CreateOptsBuilder) (*images.Image, error)
	DeleteImage(ctx context.Context, id string) error
	UpdateImage(ctx context.Context, id string, updateOpts images.UpdateOptsBuilder) (*images.Image, error)
	UploadData(ctx context.Context, id string, data io.Reader) error
	GetImportInfo(ctx context.Context) (*imageimport.ImportInfo, error)
	CreateImport(ctx context.Context, id string, createOpts imageimport.CreateOptsBuilder) error
	ListImageMembers(ctx context.Context, imageID string) iter.Seq2[*members.Member, error]
	CreateImageMember(ctx context.Context, imageID, memberID string) (*members.Member, error)
	DeleteImageMember(ctx context.Context, imageID, memberID string) error
	GetImageMember(ctx context.Context, imageID, memberID string) (*members.Member, error)
	UpdateImageMember(ctx context.Context, imageID, memberID string, opts members.UpdateOptsBuilder) (*members.Member, error)
	GetAuthenticatedProjectID() (string, error)
}

type imageClient struct{ client *gophercloud.ServiceClient }

// NewImageClient returns a new glance client.
func NewImageClient(providerClient *gophercloud.ProviderClient, providerClientOpts *clientconfig.ClientOpts) (ImageClient, error) {
	images, err := openstack.NewImageV2(providerClient, gophercloud.EndpointOpts{
		Region:       providerClientOpts.RegionName,
		Availability: clientconfig.GetEndpointType(providerClientOpts.EndpointType),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create image service client: %w", err)
	}

	return imageClient{images}, nil
}

func (c imageClient) ListImages(ctx context.Context, listOpts images.ListOptsBuilder) iter.Seq2[*images.Image, error] {
	pager := images.List(c.client, listOpts)
	return listResources(ctx, pager, images.ExtractImages)
}

func (c imageClient) GetImage(ctx context.Context, id string) (*images.Image, error) {
	image := &images.Image{}
	err := images.Get(ctx, c.client, id).ExtractInto(image)
	if err != nil {
		return nil, err
	}
	return image, nil
}

func (c imageClient) CreateImage(ctx context.Context, createOpts images.CreateOptsBuilder) (*images.Image, error) {
	image, err := images.Create(ctx, c.client, createOpts).Extract()
	if err != nil {
		return nil, err
	}
	return image, nil
}

func (c imageClient) DeleteImage(ctx context.Context, id string) error {
	return images.Delete(ctx, c.client, id).ExtractErr()
}

func (c imageClient) UpdateImage(ctx context.Context, id string, opts images.UpdateOptsBuilder) (*images.Image, error) {
	return images.Update(ctx, c.client, id, opts).Extract()
}

func (c imageClient) UploadData(ctx context.Context, id string, data io.Reader) error {
	return imagedata.Upload(ctx, c.client, id, data).ExtractErr()
}

func (c imageClient) GetImportInfo(ctx context.Context) (*imageimport.ImportInfo, error) {
	return imageimport.Get(ctx, c.client).Extract()
}

func (c imageClient) CreateImport(ctx context.Context, id string, createOpts imageimport.CreateOptsBuilder) error {
	return imageimport.Create(ctx, c.client, id, createOpts).ExtractErr()
}

func (c imageClient) ListImageMembers(ctx context.Context, imageID string) iter.Seq2[*members.Member, error] {
	pager := members.List(c.client, imageID)
	return listResources(ctx, pager, members.ExtractMembers)
}

func (c imageClient) CreateImageMember(ctx context.Context, imageID, memberID string) (*members.Member, error) {
	return members.Create(ctx, c.client, imageID, memberID).Extract()
}

func (c imageClient) DeleteImageMember(ctx context.Context, imageID, memberID string) error {
	return members.Delete(ctx, c.client, imageID, memberID).ExtractErr()
}

func (c imageClient) GetImageMember(ctx context.Context, imageID, memberID string) (*members.Member, error) {
	return members.Get(ctx, c.client, imageID, memberID).Extract()
}

func (c imageClient) UpdateImageMember(ctx context.Context, imageID, memberID string, opts members.UpdateOptsBuilder) (*members.Member, error) {
	return members.Update(ctx, c.client, imageID, memberID, opts).Extract()
}

// GetAuthenticatedProjectID returns the ID of the project of the token
func (c imageClient) GetAuthenticatedProjectID() (string, error) {
	r := c.client.GetAuthResult()
	if r == nil {
		return "", errors.New("no AuthResult available")
	}
	result, ok := r.(tokens.CreateResult)
	if !ok {
		return "", errors.New("wrong AuthResult version")
	}
	project, err := result.ExtractProject()
	if err != nil {
		return "", err
	}
	if project == nil {
		return "", errors.New("token is not project scoped")
	}
	return project.ID, nil
}

type imageErrorClient struct{ error }

// NewImageErrorClient returns an ImageClient in which every method returns the given error.
func NewImageErrorClient(e error) ImageClient {
	return imageErrorClient{e}
}

func (e imageErrorClient) ListImages(_ context.Context, _ images.ListOptsBuilder) iter.Seq2[*images.Image, error] {
	return func(yield func(*images.Image, error) bool) {
		yield(nil, e.error)
	}
}

func (e imageErrorClient) GetImage(_ context.Context, _ string) (*images.Image, error) {
	return nil, e.error
}

func (e imageErrorClient) CreateImage(_ context.Context, _ images.CreateOptsBuilder) (*images.Image, error) {
	return nil, e.error
}

func (e imageErrorClient) DeleteImage(_ context.Context, _ string) error {
	return e.error
}

func (e imageErrorClient) UpdateImage(_ context.Context, _ string, _ images.UpdateOptsBuilder) (*images.Image, error) {
	return nil, e.error
}

func (e imageErrorClient) UploadData(_ context.Context, _ string, _ io.Reader) error {
	return e.error
}

func (e imageErrorClient) GetImportInfo(_ context.Context) (*imageimport.ImportInfo, error) {
	return nil, e.error
}

func (e imageErrorClient) CreateImport(_ context.Context, _ string, _ imageimport.CreateOptsBuilder) error {
	return e.error
}

func (e imageErrorClient) ListImageMembers(_ context.Context, _ string) iter.Seq2[*members.Member, error] {
	return func(yield func(*members.Member, error) bool) {
		yield(nil, e.error)
	}
}

func (e imageErrorClient) CreateImageMember(_ context.Context, _, _ string) (*members.Member, error) {
	return nil, e.error
}

func (e imageErrorClient) DeleteImageMember(_ context.Context, _, _ string) error {
	return e.error
}

func (e imageErrorClient) GetImageMember(_ context.Context, _, _ string) (*members.Member, error) {
	return nil, e.error
}

func (e imageErrorClient) UpdateImageMember(_ context.Context, _, _ string, _ members.UpdateOptsBuilder) (*members.Member, error) {
	return nil, e.error
}

func (e imageErrorClient) GetAuthenticatedProjectID() (string, error) {
	return "", e.error
}
