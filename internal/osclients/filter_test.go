package osclients_test

import (
	"errors"
	"fmt"
	"iter"
	"testing"

	"github.com/k-orc/openstack-resource-controller/internal/osclients"
	"k8s.io/utils/ptr"
)

func pack[T any](v ...T) []T { return v }

func TestFilter(t *testing.T) {
	checks := pack[func([]*int) error]
	filters := pack[osclients.ResourceFilter[int]]

	hasValues := func(want ...int) func([]*int) error {
		return func(have []*int) error {
			if len(have) != len(want) {
				return fmt.Errorf("expected %d results, got %d: %v", len(want), len(have), have)
			}
			for i := range want {
				if have[i] == nil || want[i] != *have[i] {
					return fmt.Errorf("expected element %d to be %d, got %d", i, want[i], have[i])
				}
			}
			return nil
		}
	}

	iterator := func(errorOn int) iter.Seq2[*int, error] {
		return func(yield func(*int, error) bool) {
			for i := range 10 {
				if i == errorOn {
					_ = yield(ptr.To(0), errors.New("test error"))
					return
				}
				if !yield(ptr.To(i), nil) {
					return
				}
			}
		}
	}

	filterEq := func(want int) func(*int) bool {
		return func(have *int) bool {
			return have != nil && want == *have
		}
	}
	filterLT := func(maxN int) func(*int) bool {
		return func(have *int) bool {
			return have != nil && *have < maxN
		}
	}
	filterGT := func(minN int) func(*int) bool {
		return func(have *int) bool {
			return have != nil && *have > minN
		}
	}

	for _, tc := range [...]struct {
		name    string
		filters []osclients.ResourceFilter[int]
		checks  []func([]*int) error
	}{
		{
			"returns all",
			filters(),
			checks(hasValues(0, 1, 2, 3, 4, 5, 6, 7, 8, 9)),
		},
		{
			"returns one",
			filters(filterEq(5)),
			checks(hasValues(5)),
		},
		{
			"returns multiple",
			filters(filterLT(5)),
			checks(hasValues(0, 1, 2, 3, 4)),
		},
		{
			"applies multiple filters",
			filters(filterLT(5), filterGT(2)),
			checks(hasValues(3, 4)),
		},
		{
			"returns none",
			filters(filterLT(2), filterGT(5)),
			checks(hasValues()),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var results []*int
			for res := range osclients.Filter(iterator(-1), tc.filters...) {
				results = append(results, res)
			}

			for _, check := range tc.checks {
				if e := check(results); e != nil {
					t.Error(e)
				}
			}
		})
	}

	t.Run("passes errors", func(t *testing.T) {
		for value, err := range osclients.Filter(iterator(123)) {
			if value != nil && *value == 123 {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			}
		}
	})
}
