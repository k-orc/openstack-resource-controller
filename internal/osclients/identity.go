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
	"net/http"
	"slices"
	"strings"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/projects"
	"github.com/gophercloud/utils/v2/openstack/clientconfig"
)

type IdentityClient interface {
	CreateProject(ctx context.Context, opts projects.CreateOptsBuilder) (*projects.Project, error)
	GetProject(ctx context.Context, id string) (*projects.Project, error)
	DeleteProject(ctx context.Context, id string) error
	ListProjects(ctx context.Context, opts projects.ListOptsBuilder) iter.Seq2[*projects.Project, error]
	UpdateProject(ctx context.Context, id string, opts projects.UpdateOptsBuilder) (*projects.Project, error)
}

type identityClient struct{ client *gophercloud.ServiceClient }

// NewIdentityClient returns a new cinder client.
func NewIdentityClient(providerClient *gophercloud.ProviderClient, providerClientOpts *clientconfig.ClientOpts) (IdentityClient, error) {
	identity, err := openstack.NewIdentityV3(providerClient, gophercloud.EndpointOpts{
		Region:       providerClientOpts.RegionName,
		Availability: clientconfig.GetEndpointType(providerClientOpts.EndpointType),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create identity service client: %w", err)
	}

	return &identityClient{identity}, nil
}

// ListProjects lists the projects matching opts. Listing projects needs admin by default: when it
// is forbidden, this lists the projects the caller has access to instead, filtered here.
func (c identityClient) ListProjects(ctx context.Context, opts projects.ListOptsBuilder) iter.Seq2[*projects.Project, error] {
	return func(yield func(*projects.Project, error) bool) {
		yielded := false
		for project, err := range listResources(ctx, projects.List(c.client, opts), projects.ExtractProjects) {
			if err != nil && !yielded && gophercloud.ResponseCodeIs(err, http.StatusForbidden) {
				// Listing all projects is admin-only by default. Fall back to the
				// projects the caller can access, filtered client-side.
				listOpts, ok := projectListOpts(opts)
				if !ok {
					yield(nil, err)
					return
				}
				available := listResources(ctx, projects.ListAvailable(c.client), projects.ExtractProjects)
				for project, err := range Filter(available, func(p *projects.Project) bool { return projectMatches(p, listOpts) }) {
					if !yield(project, err) {
						return
					}
				}
				return
			}
			yielded = true
			if !yield(project, err) {
				return
			}
		}
	}
}

func projectListOpts(opts projects.ListOptsBuilder) (projects.ListOpts, bool) {
	switch o := opts.(type) {
	case projects.ListOpts:
		return o, true
	case *projects.ListOpts:
		return *o, true
	}
	return projects.ListOpts{}, false
}

// projectMatches applies a project list filter the way Keystone does.
func projectMatches(project *projects.Project, opts projects.ListOpts) bool {
	hasAll := func(tags string) bool {
		for _, t := range strings.Split(tags, ",") {
			if !slices.Contains(project.Tags, t) {
				return false
			}
		}
		return true
	}
	hasAny := func(tags string) bool {
		for _, t := range strings.Split(tags, ",") {
			if slices.Contains(project.Tags, t) {
				return true
			}
		}
		return false
	}
	return (opts.Name == "" || project.Name == opts.Name) &&
		(opts.DomainID == "" || project.DomainID == opts.DomainID) &&
		(opts.ParentID == "" || project.ParentID == opts.ParentID) &&
		(opts.Enabled == nil || project.Enabled == *opts.Enabled) &&
		(opts.IsDomain == nil || project.IsDomain == *opts.IsDomain) &&
		(opts.Tags == "" || hasAll(opts.Tags)) &&
		(opts.TagsAny == "" || hasAny(opts.TagsAny)) &&
		(opts.NotTags == "" || !hasAll(opts.NotTags)) &&
		(opts.NotTagsAny == "" || !hasAny(opts.NotTagsAny))
}

func (c identityClient) CreateProject(ctx context.Context, opts projects.CreateOptsBuilder) (*projects.Project, error) {
	return projects.Create(ctx, c.client, opts).Extract()
}

func (c identityClient) DeleteProject(ctx context.Context, id string) error {
	return projects.Delete(ctx, c.client, id).ExtractErr()
}

func (c identityClient) GetProject(ctx context.Context, id string) (*projects.Project, error) {
	return projects.Get(ctx, c.client, id).Extract()
}

func (c identityClient) UpdateProject(ctx context.Context, id string, opts projects.UpdateOptsBuilder) (*projects.Project, error) {
	return projects.Update(ctx, c.client, id, opts).Extract()
}

type identityErrorClient struct{ error }

// NewProjectErrorClient returns a IdentityClient in which every method returns the given error.
func NewIdentityErrorClient(e error) IdentityClient {
	return identityErrorClient{e}
}

func (e identityErrorClient) ListProjects(_ context.Context, _ projects.ListOptsBuilder) iter.Seq2[*projects.Project, error] {
	return func(yield func(*projects.Project, error) bool) {
		yield(nil, e.error)
	}
}

func (e identityErrorClient) CreateProject(_ context.Context, _ projects.CreateOptsBuilder) (*projects.Project, error) {
	return nil, e.error
}

func (e identityErrorClient) DeleteProject(_ context.Context, _ string) error {
	return e.error
}

func (e identityErrorClient) GetProject(_ context.Context, _ string) (*projects.Project, error) {
	return nil, e.error
}

func (e identityErrorClient) UpdateProject(_ context.Context, _ string, _ projects.UpdateOptsBuilder) (*projects.Project, error) {
	return nil, e.error
}
