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

package image

import (
	"slices"
	"testing"

	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/members"
)

func TestDiffMembers(t *testing.T) {
	current := func(ids ...string) []members.Member {
		result := make([]members.Member, len(ids))
		for i := range ids {
			result[i] = members.Member{MemberID: ids[i], Status: "pending"}
		}
		return result
	}

	testCases := []struct {
		name       string
		desired    []string
		current    []members.Member
		wantAdd    []string
		wantRemove []string
	}{
		{name: "No members"},
		{name: "Unchanged", desired: []string{"a", "b"}, current: current("b", "a")},
		{name: "Add", desired: []string{"a", "b"}, current: current("a"), wantAdd: []string{"b"}},
		{name: "Remove", desired: []string{"a"}, current: current("a", "b"), wantRemove: []string{"b"}},
		{name: "Remove members added outside ORC", current: current("a"), wantRemove: []string{"a"}},
		{name: "Replace", desired: []string{"b"}, current: current("a"), wantAdd: []string{"b"}, wantRemove: []string{"a"}},
		{name: "Two refs to the same project", desired: []string{"a", "a"}, wantAdd: []string{"a"}},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			add, remove := diffMembers(tt.desired, tt.current)
			if !slices.Equal(add, tt.wantAdd) {
				t.Errorf("add: got %v, want %v", add, tt.wantAdd)
			}
			if !slices.Equal(remove, tt.wantRemove) {
				t.Errorf("remove: got %v, want %v", remove, tt.wantRemove)
			}
		})
	}
}
