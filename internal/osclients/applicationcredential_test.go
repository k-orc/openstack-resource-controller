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
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
)

const testApplicationCredentialID = "credential"

func TestGetApplicationCredentialForUser(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusNotFound, http.StatusForbidden, http.StatusServiceUnavailable} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			var requests atomic.Int32
			client := applicationcredentialClient{client: newTestServiceClient(t, func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if r.Method != http.MethodGet || r.URL.Path != "/users/owner/application_credentials/credential" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL)
					w.WriteHeader(http.StatusForbidden)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)
				if status == http.StatusOK {
					_, _ = fmt.Fprint(w, `{"application_credential":{"id":"credential"}}`)
				}
			})}

			credential, err := client.GetApplicationCredentialForUser(t.Context(), "owner", testApplicationCredentialID)
			if status == http.StatusOK {
				if err != nil || credential == nil || credential.ID != testApplicationCredentialID {
					t.Errorf("credential = %v, error = %v, want matching credential", credential, err)
				}
			} else {
				if !gophercloud.ResponseCodeIs(err, status) {
					t.Errorf("error = %v, want HTTP %d", err, status)
				}
				if credential != nil {
					t.Errorf("unexpected credential: %v", credential)
				}
			}
			if requests.Load() != 1 {
				t.Errorf("requests = %d, want one owner lookup without discovery", requests.Load())
			}
		})
	}
}

func TestApplicationCredentialErrorClientForUser(t *testing.T) {
	wantErr := errors.New("client error")
	credential, err := NewApplicationCredentialErrorClient(wantErr).GetApplicationCredentialForUser(t.Context(), "owner", testApplicationCredentialID)
	if credential != nil || !errors.Is(err, wantErr) {
		t.Errorf("credential = %v, error = %v, want nil credential and %v", credential, err, wantErr)
	}
}

func TestGetApplicationCredentialUserListing(t *testing.T) {
	for _, tc := range []struct {
		name          string
		firstStatus   int
		nextStatus    int
		found         bool
		empty         bool
		wantStatus    int
		wantUserLists int32
		wantLookups   int32
	}{
		{
			name: "user list forbidden", firstStatus: http.StatusForbidden,
			wantStatus: http.StatusForbidden, wantUserLists: 1,
		},
		{
			name: "user list unavailable", firstStatus: http.StatusServiceUnavailable,
			wantStatus: http.StatusServiceUnavailable, wantUserLists: 1,
		},
		{
			name: "user list fails after partial results", nextStatus: http.StatusServiceUnavailable,
			wantStatus: http.StatusServiceUnavailable, wantUserLists: 2, wantLookups: 1,
		},
		{
			name:          "complete listing without match",
			wantUserLists: 1, wantLookups: 1,
		},
		{
			name: "empty listing", empty: true,
			wantUserLists: 1,
		},
		{
			name:  "match stops before next page",
			found: true, nextStatus: http.StatusServiceUnavailable,
			wantUserLists: 1, wantLookups: 1,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var userLists, lookups atomic.Int32
			client := applicationcredentialClient{client: newTestServiceClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/users":
					userLists.Add(1)
					if r.URL.Query().Get("marker") == "first" {
						w.WriteHeader(tc.nextStatus)
						return
					}
					if tc.firstStatus != 0 {
						w.WriteHeader(tc.firstStatus)
						return
					}
					if tc.empty {
						_, _ = fmt.Fprint(w, `{"users":[]}`)
					} else if tc.nextStatus != 0 {
						_, _ = fmt.Fprintf(w, `{"users":[{"id":"first"}],"links":{"next":"http://%s/users?marker=first"}}`, r.Host)
					} else {
						_, _ = fmt.Fprint(w, `{"users":[{"id":"first"}]}`)
					}
				case "/users/first/application_credentials/credential":
					lookups.Add(1)
					if tc.found {
						_, _ = fmt.Fprint(w, `{"application_credential":{"id":"credential"}}`)
					} else {
						w.WriteHeader(http.StatusNotFound)
					}
				default:
					t.Errorf("unexpected request: %s", r.URL)
					w.WriteHeader(http.StatusNotFound)
				}
			})}
			credential, err := client.GetApplicationCredential(t.Context(), testApplicationCredentialID)
			switch {
			case tc.wantStatus != 0:
				if !gophercloud.ResponseCodeIs(err, tc.wantStatus) {
					t.Errorf("error = %v, want HTTP %d", err, tc.wantStatus)
				}
			case tc.found:
				if err != nil || credential == nil || credential.ID != testApplicationCredentialID {
					t.Errorf("credential = %v, error = %v, want matching credential", credential, err)
				}
			default:
				var notFound gophercloud.ErrResourceNotFound
				if !errors.As(err, &notFound) || notFound.Name != testApplicationCredentialID {
					t.Errorf("error = %v, want credential not found", err)
				}
			}
			if !tc.found && credential != nil {
				t.Errorf("unexpected credential: %v", credential)
			}
			if userLists.Load() != tc.wantUserLists || lookups.Load() != tc.wantLookups {
				t.Errorf("user lists/lookups = %d/%d, want %d/%d",
					userLists.Load(), lookups.Load(), tc.wantUserLists, tc.wantLookups)
			}
		})
	}
}
