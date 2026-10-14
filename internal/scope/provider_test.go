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

package scope

import (
	"fmt"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"k8s.io/apimachinery/pkg/util/cache"
)

func newProviderScope() *providerScope {
	s := &providerScope{
		providerClient:         nil,
		providerClientOpts:     nil,
		providerClientCache:    cache.NewLRUExpireCache(5),
		providerClientCacheKey: "key",
	}

	s.providerClientCache.Add(s.providerClientCacheKey, s, time.Minute*5)

	return s
}

func TestCheckNewClientError(t *testing.T) {
	t.Run("cache evicted on ErrEndpointNotFound", func(t *testing.T) {
		s := newProviderScope()
		cache := s.providerClientCache

		s.checkNewClientError(&gophercloud.ErrEndpointNotFound{})

		if s.providerClientCache != nil {
			t.Errorf("Expected providerClientCache to be nil, got value instead")
		}

		_, found := cache.Get(s.providerClientCacheKey)

		if found {
			t.Errorf("Expected cache entry to be cleared, but key is still found after eviction")
		}
	})

	t.Run("missing cache handled gracefully", func(t *testing.T) {
		s := newProviderScope()
		s.providerClientCache = nil

		s.checkNewClientError(&gophercloud.ErrEndpointNotFound{})
	})

	tests := []struct {
		name string
		err  error
		s    *providerScope
	}{
		{
			name: "nil error does not evict cache",
			err:  nil,
			s:    newProviderScope(),
		},
		{
			name: "generic error does not evict cache",
			err:  fmt.Errorf("some generic error"),
			s:    newProviderScope(),
		},
		{
			name: "concurrent cache modification does not evict cache",
			err:  &gophercloud.ErrEndpointNotFound{},
			s: func() *providerScope {
				s := newProviderScope()
				s.providerClientCache.Add(s.providerClientCacheKey, &providerScope{}, time.Minute*5)
				return s
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.s.checkNewClientError(tt.err)

			if tt.s.providerClientCache == nil {
				t.Errorf("Expected providerClientCache to be valid, got nil instead")
			}

			_, found := tt.s.providerClientCache.Get(tt.s.providerClientCacheKey)

			if !found {
				t.Errorf("Expected cache entry to be found, but key is missing from cache")
			}
		})
	}
}
