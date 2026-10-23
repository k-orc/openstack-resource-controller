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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/pagination"
)

type iteratorTestPage struct {
	pagination.SinglePageBase
	items      []int
	next       string
	emptyErr   error
	nextErr    error
	extractErr error
	nextCalls  *int
}

func (p iteratorTestPage) IsEmpty() (bool, error) {
	return len(p.items) == 0, p.emptyErr
}

func (p iteratorTestPage) NextPageURL() (string, error) {
	(*p.nextCalls)++
	return p.next, p.nextErr
}

func newTestServiceClient(t *testing.T, handler http.HandlerFunc) *gophercloud.ServiceClient {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return &gophercloud.ServiceClient{
		ProviderClient: &gophercloud.ProviderClient{HTTPClient: *server.Client()},
		Endpoint:       server.URL + "/",
	}
}

func checkPaginationErrors(t *testing.T, got []error, wantErr error, wantStatus int, wantJSONError bool) {
	t.Helper()
	wantCount := 0
	if wantErr != nil || wantStatus != 0 || wantJSONError {
		wantCount = 1
	}
	if len(got) != wantCount {
		t.Fatalf("errors = %v, want %d errors", got, wantCount)
	}
	if wantCount == 0 {
		return
	}
	err := got[0]
	if wantErr != nil && !errors.Is(err, wantErr) {
		t.Errorf("error = %v, want %v", err, wantErr)
	}
	if wantStatus != 0 && !gophercloud.ResponseCodeIs(err, wantStatus) {
		t.Errorf("error = %v, want HTTP %d", err, wantStatus)
	}
	var syntaxErr *json.SyntaxError
	if wantJSONError && !errors.As(err, &syntaxErr) {
		t.Errorf("error = %v, want JSON syntax error", err)
	}
}

func TestListResources(t *testing.T) {
	testErr := errors.New("pagination failure")
	for _, tc := range []struct {
		name          string
		pages         []iteratorTestPage
		initErr       error
		failPage      string
		malformedPage string
		cancelBefore  bool
		cancelExtract bool
		cancelAfter   int
		stopAfter     int
		want          []int
		wantErr       error
		wantStatus    int
		wantJSONError bool
		wantRequests  int32
		wantExtracts  int
		wantNextCalls int
	}{
		{
			name:  "single page",
			pages: []iteratorTestPage{{items: []int{1, 2}}},
			want:  []int{1, 2}, wantRequests: 1, wantExtracts: 1, wantNextCalls: 1,
		},
		{
			name: "multiple pages",
			pages: []iteratorTestPage{
				{items: []int{1, 2}, next: "2"},
				{items: []int{3, 4}},
			},
			want: []int{1, 2, 3, 4}, wantRequests: 2, wantExtracts: 2, wantNextCalls: 2,
		},
		{
			name: "empty listing", pages: []iteratorTestPage{{}},
			wantRequests: 1,
		},
		{
			name:  "empty final page",
			pages: []iteratorTestPage{{items: []int{1}, next: "2"}, {}},
			want:  []int{1}, wantRequests: 2, wantExtracts: 1, wantNextCalls: 1,
		},
		{
			name:    "pager initialization error",
			initErr: testErr, wantErr: testErr,
		},
		{
			name: "first page fetch failure", failPage: "/1",
			wantStatus: http.StatusServiceUnavailable, wantRequests: 1,
		},
		{
			name:     "later page fetch failure",
			pages:    []iteratorTestPage{{items: []int{1, 2}, next: "2"}},
			failPage: "/2", want: []int{1, 2}, wantStatus: http.StatusServiceUnavailable,
			wantRequests: 2, wantExtracts: 1, wantNextCalls: 1,
		},
		{
			name: "malformed first response", malformedPage: "/1",
			wantJSONError: true, wantRequests: 1,
		},
		{
			name:          "malformed later response",
			pages:         []iteratorTestPage{{items: []int{1}, next: "2"}},
			malformedPage: "/2", want: []int{1}, wantJSONError: true,
			wantRequests: 2, wantExtracts: 1, wantNextCalls: 1,
		},
		{
			name:    "IsEmpty error",
			pages:   []iteratorTestPage{{items: []int{1}, emptyErr: testErr}},
			wantErr: testErr, wantRequests: 1,
		},
		{
			name:  "NextPageURL error",
			pages: []iteratorTestPage{{items: []int{1, 2}, nextErr: testErr}},
			want:  []int{1, 2}, wantErr: testErr,
			wantRequests: 1, wantExtracts: 1, wantNextCalls: 1,
		},
		{
			name:    "extraction error",
			pages:   []iteratorTestPage{{items: []int{1}, extractErr: testErr}},
			wantErr: testErr, wantRequests: 1, wantExtracts: 1,
		},
		{
			name: "extraction error after partial results",
			pages: []iteratorTestPage{
				{items: []int{1}, next: "2"},
				{items: []int{2}, extractErr: testErr},
			},
			want: []int{1}, wantErr: testErr,
			wantRequests: 2, wantExtracts: 2, wantNextCalls: 1,
		},
		{
			name: "canceled before iteration", cancelBefore: true,
			wantErr: context.Canceled,
		},
		{
			name:  "canceled during extraction",
			pages: []iteratorTestPage{{items: []int{1}}}, cancelExtract: true,
			wantErr: context.Canceled, wantRequests: 1, wantExtracts: 1,
		},
		{
			name:  "canceled within page",
			pages: []iteratorTestPage{{items: []int{1, 2}}}, cancelAfter: 1,
			want: []int{1}, wantErr: context.Canceled,
			wantRequests: 1, wantExtracts: 1,
		},
		{
			name:        "canceled between pages",
			pages:       []iteratorTestPage{{items: []int{1}, next: "2"}},
			cancelAfter: 1, want: []int{1}, wantErr: context.Canceled,
			wantRequests: 1, wantExtracts: 1, wantNextCalls: 1,
		},
		{
			name:      "stop within page",
			pages:     []iteratorTestPage{{items: []int{1, 2}, next: "2"}},
			stopAfter: 1, want: []int{1}, wantRequests: 1, wantExtracts: 1,
		},
		{
			name:      "stop at page boundary",
			pages:     []iteratorTestPage{{items: []int{1, 2}, next: "2", nextErr: testErr}},
			stopAfter: 2, want: []int{1, 2}, wantRequests: 1, wantExtracts: 1,
		},
		{
			name:        "cancel and stop within page",
			pages:       []iteratorTestPage{{items: []int{1, 2}, next: "2"}},
			cancelAfter: 1, stopAfter: 1, want: []int{1},
			wantRequests: 1, wantExtracts: 1,
		},
		{
			name:        "cancel and stop at page boundary",
			pages:       []iteratorTestPage{{items: []int{1}, next: "2", nextErr: testErr}},
			cancelAfter: 1, stopAfter: 1, want: []int{1},
			wantRequests: 1, wantExtracts: 1,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// An error is terminal whether or not the consumer asks to continue.
			for _, stopOnError := range []bool{false, true} {
				t.Run(fmt.Sprintf("stopOnError=%t", stopOnError), func(t *testing.T) {
					ctx, cancel := context.WithCancel(t.Context())
					defer cancel()
					var requests atomic.Int32
					client := newTestServiceClient(t, func(w http.ResponseWriter, r *http.Request) {
						requests.Add(1)
						w.Header().Set("Content-Type", "application/json")
						if r.URL.Path == tc.failPage {
							w.WriteHeader(http.StatusServiceUnavailable)
							return
						}
						if r.URL.Path == tc.malformedPage {
							_, _ = fmt.Fprint(w, "{invalid")
							return
						}
						_, _ = fmt.Fprint(w, "{}")
					})
					nextCalls, extracts := 0, 0
					pager := pagination.NewPager(client, client.Endpoint+"1", func(r pagination.PageResult) pagination.Page {
						var index int
						if _, err := fmt.Sscanf(r.Path, "/%d", &index); err != nil || index < 1 || index > len(tc.pages) {
							t.Fatalf("unexpected page URL: %s", r.String())
						}
						page := tc.pages[index-1]
						page.SinglePageBase = pagination.SinglePageBase(r)
						page.nextCalls = &nextCalls
						if page.next != "" {
							page.next = client.Endpoint + page.next
						}
						return page
					})
					pager.Err = tc.initErr
					seq := listResources(ctx, pager, func(p pagination.Page) ([]int, error) {
						extracts++
						if tc.cancelExtract {
							cancel()
						}
						page := p.(iteratorTestPage)
						return page.items, page.extractErr
					})
					if requests.Load() != 0 {
						t.Fatal("listing fetched a page before iteration")
					}
					if tc.cancelBefore {
						cancel()
					}
					var got []int
					var gotErrors []error
					stopped := false
					seq(func(item *int, err error) bool {
						if stopped || len(gotErrors) != 0 {
							t.Fatal("yield called after termination or error")
						}
						if err != nil {
							if item != nil {
								t.Errorf("error yielded with non-nil item: %v", item)
							}
							gotErrors = append(gotErrors, err)
							stopped = stopOnError
							return !stopped
						}
						if item == nil {
							t.Fatal("nil item without error")
						}
						got = append(got, *item)
						if tc.cancelAfter != 0 && len(got) == tc.cancelAfter {
							cancel()
						}
						stopped = tc.stopAfter != 0 && len(got) == tc.stopAfter
						return !stopped
					})
					if !slices.Equal(got, tc.want) {
						t.Errorf("items = %v, want %v", got, tc.want)
					}
					checkPaginationErrors(t, gotErrors, tc.wantErr, tc.wantStatus, tc.wantJSONError)
					if requests.Load() != tc.wantRequests || extracts != tc.wantExtracts || nextCalls != tc.wantNextCalls {
						t.Errorf("requests/extracts/next calls = %d/%d/%d, want %d/%d/%d",
							requests.Load(), extracts, nextCalls, tc.wantRequests, tc.wantExtracts, tc.wantNextCalls)
					}
				})
			}
		})
	}
}
