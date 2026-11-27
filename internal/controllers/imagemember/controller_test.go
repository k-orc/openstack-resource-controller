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

package imagemember

import (
	"testing"
	"time"
)

func TestSetDefaultResyncPeriod(t *testing.T) {
	testCases := []struct {
		name   string
		global time.Duration
		want   time.Duration
	}{
		{name: "disabled globally", global: 0, want: defaultResyncPeriod},
		{name: "set globally", global: 5 * time.Minute, want: 5 * time.Minute},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			c := &imagememberReconcilerConstructor{}
			c.SetDefaultResyncPeriod(tt.global)
			if c.defaultResyncPeriod != tt.want {
				t.Errorf("got %v, want %v", c.defaultResyncPeriod, tt.want)
			}
		})
	}
}
