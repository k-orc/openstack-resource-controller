package osclients_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/k-orc/openstack-resource-controller/internal/osclients"
)

func pack[T any](v ...T) []T { return v }

func TestFilter(t *testing.T) {
	checks := pack[func([]osclients.Result[int]) error]
	filters := pack[func(int) bool]

	noError := func(results []osclients.Result[int]) error {
		for _, res := range results {
			if err := res.Err(); err != nil {
				return fmt.Errorf("unexpected error: %w", err)
			}
		}
		return nil
	}

	hasValues := func(want ...int) func([]osclients.Result[int]) error {
		return func(have []osclients.Result[int]) error {
			if len(have) != len(want) {
				return fmt.Errorf("expected %d results, got %d: %v", len(want), len(have), have)
			}
			for i := range want {
				if want[i] != have[i].Ok() {
					return fmt.Errorf("expected element %d to be %d, got %d", i, want[i], have[i].Ok())
				}
			}
			return nil
		}
	}

	iterator := func(ctx context.Context) <-chan osclients.Result[int] {
		ch := make(chan (osclients.Result[int]))
		go func() {
			defer close(ch)
			for i := 0; true; i++ {
				if i > 9 {
					return
				}
				select {
				case <-ctx.Done():
					ch <- osclients.NewResultErr[int](ctx.Err())
					return
				case ch <- osclients.NewResultOk(i):
				}
			}
		}()
		return ch
	}

	filterEq := func(want int) func(int) bool {
		return func(have int) bool {
			return want == have
		}
	}
	filterLT := func(maxN int) func(int) bool {
		return func(have int) bool {
			return have < maxN
		}
	}
	filterGT := func(minN int) func(int) bool {
		return func(have int) bool {
			return have > minN
		}
	}

	for _, tc := range [...]struct {
		name    string
		filters []func(int) bool
		checks  []func([]osclients.Result[int]) error
	}{
		{
			"returns all",
			filters(),
			checks(noError, hasValues(0, 1, 2, 3, 4, 5, 6, 7, 8, 9)),
		},
		{
			"returns one",
			filters(filterEq(5)),
			checks(noError, hasValues(5)),
		},
		{
			"returns multiple",
			filters(filterLT(5)),
			checks(noError, hasValues(0, 1, 2, 3, 4)),
		},
		{
			"applies multiple filters",
			filters(filterLT(5), filterGT(2)),
			checks(noError, hasValues(3, 4)),
		},
		{
			"returns none",
			filters(filterLT(2), filterGT(5)),
			checks(noError, hasValues()),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			var results []osclients.Result[int]
			for res := range osclients.Filter(iterator(ctx), tc.filters...) {
				results = append(results, res)
			}

			for _, check := range tc.checks {
				if e := check(results); e != nil {
					t.Error(e)
				}
			}
		})
	}

	erroredIterator := func(ctx context.Context) <-chan osclients.Result[int] {
		ch := make(chan (osclients.Result[int]))
		go func() {
			defer close(ch)
			for i := 0; true; i++ {
				if i >= 150 {
					return
				}

				if i == 123 {
					ch <- osclients.NewResultErr[int](fmt.Errorf("test error"))
					continue
				}
				select {
				case <-ctx.Done():
					ch <- osclients.NewResultErr[int](ctx.Err())
					return
				case ch <- osclients.NewResultOk(i):
				}
			}
		}()
		return ch
	}

	t.Run("passes errors", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var i int
		for res := range osclients.Filter(erroredIterator(ctx)) {
			if i == 123 {
				if res.Err() == nil {
					t.Errorf("expected error, got nil")
				}
			}
			i++
		}

		if i != 150 {
			t.Errorf("expected 150 values, got %d", i)
		}
	})
}
