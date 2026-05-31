/*
Copyright 2024 The ORC Authors.

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

package resync

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDetermineResyncPeriod(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		specValue     *metav1.Duration
		globalDefault time.Duration
		want          time.Duration
	}{
		{
			// TS-004: spec nil, global disabled → disabled
			name:          "spec nil, global 0, returns 0 (disabled)",
			specValue:     nil,
			globalDefault: 0,
			want:          0,
		},
		{
			// TS-005: spec nil, global set → use global
			name:          "spec nil, global 1h, returns 1h",
			specValue:     nil,
			globalDefault: time.Hour,
			want:          time.Hour,
		},
		{
			// TS-010: spec overrides global
			name:          "spec 30m, global 1h, returns 30m (spec overrides)",
			specValue:     &metav1.Duration{Duration: 30 * time.Minute},
			globalDefault: time.Hour,
			want:          30 * time.Minute,
		},
		{
			// TS-004 variant: explicit 0s in spec disables resync regardless of global
			name:          "spec 0s (explicit), global 1h, returns 0 (explicitly disabled)",
			specValue:     &metav1.Duration{Duration: 0},
			globalDefault: time.Hour,
			want:          0,
		},
		{
			// TS-010 variant: spec enables resync even when global is disabled
			name:          "spec 2h, global 0, returns 2h (spec enables despite global disabled)",
			specValue:     &metav1.Duration{Duration: 2 * time.Hour},
			globalDefault: 0,
			want:          2 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := DetermineResyncPeriod(tt.specValue, tt.globalDefault)
			if got != tt.want {
				t.Errorf("DetermineResyncPeriod() = %v, want %v", got, tt.want)
			}
		})
	}
}
