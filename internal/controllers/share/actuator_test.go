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

package share

import (
	"testing"

	"github.com/gophercloud/gophercloud/v2/openstack/sharedfilesystems/v2/shares"
	"k8s.io/utils/ptr"
)

func TestNeedsUpdate(t *testing.T) {
	testCases := []struct {
		name         string
		updateOpts   shares.UpdateOpts
		expectChange bool
	}{
		{
			name:         "Empty base opts",
			updateOpts:   shares.UpdateOpts{},
			expectChange: false,
		},
		{
			name:         "Updated opts with IsPublic",
			updateOpts:   shares.UpdateOpts{IsPublic: ptr.To(true)},
			expectChange: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := needsUpdate(tt.updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})
	}
}

// NOTE: Tests for handleNameUpdate and handleDescriptionUpdate removed
// because Manila API doesn't support updating name or description
