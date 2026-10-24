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
	"iter"

	"github.com/gophercloud/gophercloud/v2/pagination"
)

// listResources yields resources from each page, followed by any pagination error.
func listResources[T any](ctx context.Context, pager pagination.Pager, extract func(pagination.Page) ([]T, error)) iter.Seq2[*T, error] {
	return func(yield func(*T, error) bool) {
		err := pager.EachPage(ctx, func(ctx context.Context, page pagination.Page) (bool, error) {
			items, err := extract(page)
			if err != nil {
				return false, err
			}
			for i := range items {
				if err := ctx.Err(); err != nil {
					return false, err
				}
				if !yield(&items[i], nil) {
					return false, nil
				}
			}
			return true, nil
		})
		if err != nil {
			yield(nil, err)
		}
	}
}
