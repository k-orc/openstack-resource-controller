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
	"fmt"
	"net/http"
	"testing"

	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/projects"
	"k8s.io/utils/ptr"
)

func Test_projectMatches(t *testing.T) {
	project := &projects.Project{
		Name: "demo", DomainID: "default", ParentID: "default", Enabled: true,
		Tags: []string{"a", "b"},
	}
	tests := []struct {
		name string
		opts projects.ListOpts
		want bool
	}{
		{"no filter", projects.ListOpts{}, true},
		{"name", projects.ListOpts{Name: "demo"}, true},
		{"other name", projects.ListOpts{Name: "other"}, false},
		{"domain", projects.ListOpts{DomainID: "default"}, true},
		{"other domain", projects.ListOpts{DomainID: "other"}, false},
		{"other parent", projects.ListOpts{ParentID: "other"}, false},
		{"enabled", projects.ListOpts{Enabled: ptr.To(true)}, true},
		{"disabled", projects.ListOpts{Enabled: ptr.To(false)}, false},
		{"is domain", projects.ListOpts{IsDomain: ptr.To(true)}, false},
		{"tags all present", projects.ListOpts{Tags: "a,b"}, true},
		{"tags one missing", projects.ListOpts{Tags: "a,c"}, false},
		{"tags-any one present", projects.ListOpts{TagsAny: "c,b"}, true},
		{"tags-any none present", projects.ListOpts{TagsAny: "c,d"}, false},
		{"not-tags all present", projects.ListOpts{NotTags: "a,b"}, false},
		{"not-tags one missing", projects.ListOpts{NotTags: "a,c"}, true},
		{"not-tags-any one present", projects.ListOpts{NotTagsAny: "c,a"}, false},
		{"not-tags-any none present", projects.ListOpts{NotTagsAny: "c,d"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := projectMatches(project, tt.opts); got != tt.want {
				t.Errorf("projectMatches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestListProjects(t *testing.T) {
	const projectsJSON = `{"projects": [
		{"id": "1", "name": "demo", "domain_id": "default", "enabled": true},
		{"id": "2", "name": "other", "domain_id": "default", "enabled": true}
	], "links": {"next": null}}`

	testCases := []struct {
		name          string
		listStatus    int
		availStatus   int
		wantIDs       []string
		wantErr       bool
		wantAvailable bool
	}{
		{name: "admin list", listStatus: http.StatusOK, wantIDs: []string{"1"}},
		{name: "forbidden falls back to available projects", listStatus: http.StatusForbidden, availStatus: http.StatusOK, wantIDs: []string{"1"}, wantAvailable: true},
		{name: "other errors are returned", listStatus: http.StatusInternalServerError, wantErr: true},
		{name: "fallback errors are returned", listStatus: http.StatusForbidden, availStatus: http.StatusInternalServerError, wantErr: true, wantAvailable: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			calledAvailable := false
			client := newTestServiceClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				status := tc.listStatus
				body := projectsJSON
				if r.URL.Path == "/auth/projects" {
					calledAvailable = true
					status = tc.availStatus
				} else if r.URL.Query().Get("name") == "demo" {
					// Keystone filters server-side
					body = `{"projects": [{"id": "1", "name": "demo", "domain_id": "default", "enabled": true}], "links": {"next": null}}`
				}
				w.WriteHeader(status)
				if status == http.StatusOK {
					_, _ = fmt.Fprint(w, body)
				}
			})

			var gotIDs []string
			var gotErr error
			for project, err := range (identityClient{client}).ListProjects(t.Context(), projects.ListOpts{Name: "demo"}) {
				if err != nil {
					gotErr = err
					continue
				}
				gotIDs = append(gotIDs, project.ID)
			}

			if (gotErr != nil) != tc.wantErr {
				t.Errorf("err = %v, wantErr %t", gotErr, tc.wantErr)
			}
			if fmt.Sprint(gotIDs) != fmt.Sprint(tc.wantIDs) {
				t.Errorf("ids = %v, want %v", gotIDs, tc.wantIDs)
			}
			if calledAvailable != tc.wantAvailable {
				t.Errorf("called /auth/projects = %t, want %t", calledAvailable, tc.wantAvailable)
			}
		})
	}
}
