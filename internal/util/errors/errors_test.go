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

package errors

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
)

func newHTTPError(statusCode int, body string) error {
	return gophercloud.ErrUnexpectedResponseCode{
		Actual: statusCode,
		Body:   []byte(body),
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error is not retryable",
			err:  nil,
			want: false,
		},
		{
			name: "non-HTTP error is not retryable",
			err:  fmt.Errorf("some client-side validation error"),
			want: false,
		},
		{
			name: "409 Conflict is not retryable",
			err:  newHTTPError(http.StatusConflict, `{"NeutronError": {"type": "IpAddressInUse"}}`),
			want: false,
		},
		{
			name: "409 Conflict with Neutron OverQuota is retryable",
			err:  newHTTPError(http.StatusConflict, `{"NeutronError": {"type": "OverQuota", "message": "Quota exceeded for resources: port."}}`),
			want: true,
		},
		{
			name: "501 Not Implemented is not retryable",
			err:  newHTTPError(http.StatusNotImplemented, ""),
			want: false,
		},
		{
			name: "400 Bad Request is retryable",
			err:  newHTTPError(http.StatusBadRequest, `{"NeutronError": {"type": "BadRequest"}}`),
			want: true,
		},
		{
			name: "403 Forbidden is retryable",
			err:  newHTTPError(http.StatusForbidden, ""),
			want: true,
		},
		{
			name: "500 Internal Server Error is retryable",
			err:  newHTTPError(http.StatusInternalServerError, ""),
			want: true,
		},
		{
			name: "503 Service Unavailable is retryable",
			err:  newHTTPError(http.StatusServiceUnavailable, ""),
			want: true,
		},
		{
			name: "wrapped non-HTTP error is not retryable",
			err:  fmt.Errorf("wrapping: %w", fmt.Errorf("banned key")),
			want: false,
		},
		{
			name: "wrapped 409 is not retryable",
			err:  fmt.Errorf("wrapping: %w", newHTTPError(http.StatusConflict, "")),
			want: false,
		},
		{
			name: "wrapped 409 with OverQuota is retryable",
			err:  fmt.Errorf("wrapping: %w", newHTTPError(http.StatusConflict, `{"NeutronError": {"type": "OverQuota"}}`)),
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRetryable(tt.err); got != tt.want {
				t.Errorf("IsRetryable() = %v, want %v", got, tt.want)
			}
		})
	}
}
