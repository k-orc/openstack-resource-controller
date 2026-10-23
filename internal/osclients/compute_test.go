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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/flavors"
)

type invalidFlavorListOpts struct{ err error }

func (o invalidFlavorListOpts) ToFlavorListQuery() (string, error) {
	return "", o.err
}

func TestListFlavors(t *testing.T) {
	optsErr := errors.New("invalid list options")
	for _, tc := range []struct {
		name         string
		opts         flavors.ListOptsBuilder
		firstBody    string
		firstStatus  int
		secondStatus int
		want         []string
		wantErr      error
		wantStatus   int
		wantTypeErr  bool
		wantRequests int32
	}{
		{
			name: "invalid options", opts: invalidFlavorListOpts{err: optsErr},
			wantErr: optsErr,
		},
		{
			name: "first request failure", firstStatus: http.StatusForbidden,
			wantStatus: http.StatusForbidden, wantRequests: 1,
		},
		{
			name:         "second request failure",
			firstBody:    `{"flavors":[{"id":"first"}],"flavors_links":[{"rel":"next","href":"%s/flavors/detail?marker=first"}]}`,
			secondStatus: http.StatusServiceUnavailable,
			want:         []string{"first"}, wantStatus: http.StatusServiceUnavailable, wantRequests: 2,
		},
		{
			name:        "invalid resource data",
			firstBody:   `{"flavors":42}`,
			wantTypeErr: true, wantRequests: 1,
		},
		{
			name:      "invalid next page links",
			firstBody: `{"flavors":[{"id":"first"}],"flavors_links":42}`,
			want:      []string{"first"}, wantTypeErr: true, wantRequests: 1,
		},
		{
			name:      "successful pagination",
			firstBody: `{"flavors":[{"id":"first"}],"flavors_links":[{"rel":"next","href":"%s/flavors/detail?marker=first"}]}`,
			want:      []string{"first", "second"}, wantRequests: 2,
		},
		{
			name: "empty listing", firstBody: `{"flavors":[]}`,
			wantRequests: 1,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var requests atomic.Int32
			client := computeClient{client: newTestServiceClient(t, func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if r.URL.Path != "/flavors/detail" {
					t.Errorf("unexpected request: %s", r.URL)
					w.WriteHeader(http.StatusNotFound)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Query().Get("marker") == "first" {
					if tc.secondStatus != 0 {
						w.WriteHeader(tc.secondStatus)
						return
					}
					_, _ = fmt.Fprint(w, `{"flavors":[{"id":"second"}]}`)
					return
				}
				if tc.firstStatus != 0 {
					w.WriteHeader(tc.firstStatus)
					return
				}
				body := tc.firstBody
				if tc.wantRequests == 2 {
					body = fmt.Sprintf(body, "http://"+r.Host)
				}
				_, _ = fmt.Fprint(w, body)
			})}
			seq := client.ListFlavors(t.Context(), tc.opts)
			if requests.Load() != 0 {
				t.Fatal("listing fetched a page before iteration")
			}
			var got []string
			var gotErrors []error
			for flavor, err := range seq {
				if len(gotErrors) != 0 {
					t.Fatal("iterator yielded after error")
				}
				if err != nil {
					if flavor != nil {
						t.Errorf("non-nil flavor with error: %v", flavor)
					}
					gotErrors = append(gotErrors, err)
					continue
				}
				got = append(got, flavor.ID)
			}
			if !slices.Equal(got, tc.want) {
				t.Errorf("IDs = %v, want %v", got, tc.want)
			}
			if tc.wantTypeErr {
				if len(gotErrors) != 1 {
					t.Fatalf("errors = %v, want one extraction error", gotErrors)
				}
				var typeErr *json.UnmarshalTypeError
				if !errors.As(gotErrors[0], &typeErr) {
					t.Errorf("error = %v, want JSON type error", gotErrors[0])
				}
			} else {
				checkPaginationErrors(t, gotErrors, tc.wantErr, tc.wantStatus, false)
			}
			if requests.Load() != tc.wantRequests {
				t.Errorf("requests = %d, want %d", requests.Load(), tc.wantRequests)
			}
		})
	}
}

var _ flavors.ListOptsBuilder = invalidFlavorListOpts{}
