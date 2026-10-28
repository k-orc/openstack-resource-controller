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

package dependency

import (
	"slices"
	"testing"
)

func TestRemovedRefs(t *testing.T) {
	tests := []struct {
		name    string
		oldRefs []string
		newRefs []string
		want    []string
	}{
		{name: "unchanged", oldRefs: []string{"a", "b"}, newRefs: []string{"a", "b"}},
		{name: "reordered", oldRefs: []string{"a", "b"}, newRefs: []string{"b", "a"}},
		{name: "added", oldRefs: []string{"a"}, newRefs: []string{"a", "b"}},
		{name: "removed one", oldRefs: []string{"a", "b"}, newRefs: []string{"b"}, want: []string{"a"}},
		{name: "removed all", oldRefs: []string{"a", "b"}, newRefs: nil, want: []string{"a", "b"}},
		{name: "replaced", oldRefs: []string{"a"}, newRefs: []string{"b"}, want: []string{"a"}},
		{name: "duplicate", oldRefs: []string{"a", "a"}, newRefs: nil, want: []string{"a"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := removedRefs(tt.oldRefs, tt.newRefs); !slices.Equal(got, tt.want) {
				t.Errorf("removedRefs(%v, %v) = %v, want %v", tt.oldRefs, tt.newRefs, got, tt.want)
			}
		})
	}
}
