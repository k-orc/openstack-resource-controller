package flavor_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/flavors"
	"github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/flavor"
	"github.com/k-orc/openstack-resource-controller/internal/osclients"
	orcerrors "github.com/k-orc/openstack-resource-controller/internal/util/errors"
	"k8s.io/utils/ptr"
)

type flavorLister struct {
	flavors []flavors.Flavor
}

func (l flavorLister) ListFlavors(_ context.Context, _ flavors.ListOptsBuilder) <-chan (osclients.Result[*flavors.Flavor]) {
	ch := make(chan (osclients.Result[*flavors.Flavor]), len(l.flavors))
	defer close(ch)
	for i := range l.flavors {
		ch <- osclients.NewResultOk(&l.flavors[i])
	}
	return ch
}

func TestGetFlavorByFilter(t *testing.T) {
	type checkFunc func(*flavors.Flavor, error) error
	checks := func(fns ...checkFunc) []checkFunc { return fns }
	noError := func(_ *flavors.Flavor, err error) error {
		if err != nil {
			return fmt.Errorf("unexpected error: %w", err)
		}
		return nil
	}
	errorMessage := func(want string) checkFunc {
		return func(_ *flavors.Flavor, err error) error {
			if err == nil {
				return fmt.Errorf("expected error, got nil")
			}
			if have := err.Error(); want != have {
				return fmt.Errorf("unexpected error message: %q", have)
			}
			return nil
		}
	}
	terminalError := func(_ *flavors.Flavor, have error) error {
		if have == nil {
			return fmt.Errorf("expected terminal error, got nil")
		}
		var terminalError *orcerrors.TerminalError
		if !errors.As(have, &terminalError) {
			return fmt.Errorf("unexpected error: %w", have)
		}
		return nil
	}
	findsNone := func(f *flavors.Flavor, _ error) error {
		if f != nil {
			return fmt.Errorf("expected nil, got flavor (ID: %q)", f.ID)
		}
		return nil
	}
	findsID := func(want string) checkFunc {
		return func(f *flavors.Flavor, _ error) error {
			if f == nil {
				return fmt.Errorf("expected flavor (ID: %q), got nil", want)
			}
			if have := f.ID; want != have {
				return fmt.Errorf("got unexpected flavor (expected ID %q, got ID %q)", want, have)
			}
			return nil
		}
	}
	type lister interface {
		ListFlavors(ctx context.Context, listOpts flavors.ListOptsBuilder) <-chan (osclients.Result[*flavors.Flavor])
	}

	for _, tc := range [...]struct {
		name   string
		filter v1alpha1.FlavorFilter
		lister lister
		checks []checkFunc
	}{
		{
			"finds one by name",
			v1alpha1.FlavorFilter{Name: ptr.To[v1alpha1.OpenStackName]("one")},
			&flavorLister{[]flavors.Flavor{
				{ID: "1", Name: "one"},
				{ID: "2", Name: "two"},
				{ID: "3", Name: "three"},
			}},
			checks(noError, findsID("1")),
		},
		{
			"finds none by name",
			v1alpha1.FlavorFilter{Name: ptr.To[v1alpha1.OpenStackName]("four")},
			&flavorLister{[]flavors.Flavor{
				{ID: "1", Name: "one"},
				{ID: "2", Name: "two"},
				{ID: "3", Name: "three"},
			}},
			checks(noError, findsNone),
		},
		{
			"errors if multiple when finding by name",
			v1alpha1.FlavorFilter{Name: ptr.To[v1alpha1.OpenStackName]("one")},
			&flavorLister{[]flavors.Flavor{
				{ID: "1", Name: "one"},
				{ID: "2", Name: "two"},
				{ID: "3", Name: "one"},
			}},
			checks(terminalError),
		},
		{
			"finds one by RAM and disk",
			v1alpha1.FlavorFilter{RAM: ptr.To[int32](2), Disk: ptr.To[int32](2)},
			&flavorLister{[]flavors.Flavor{
				{ID: "1", RAM: 1, Disk: 1},
				{ID: "2", RAM: 2, Disk: 2},
				{ID: "3", RAM: 3, Disk: 3},
			}},
			checks(noError, findsID("2")),
		},
		{
			"finds one by name RAM and disk",
			v1alpha1.FlavorFilter{
				Name: ptr.To[v1alpha1.OpenStackName]("two"),
				RAM:  ptr.To[int32](2),
				Disk: ptr.To[int32](2),
			},
			&flavorLister{[]flavors.Flavor{
				{ID: "1", Name: "one", RAM: 1, Disk: 1},
				{ID: "2", Name: "two", RAM: 2, Disk: 2},
				{ID: "3", Name: "three", RAM: 3, Disk: 3},
			}},
			checks(noError, findsID("2")),
		},
		{
			"checks RAM",
			v1alpha1.FlavorFilter{
				Name: ptr.To[v1alpha1.OpenStackName]("two"),
				RAM:  ptr.To[int32](2),
				Disk: ptr.To[int32](2),
			},
			&flavorLister{[]flavors.Flavor{
				{ID: "1", Name: "one", RAM: 1, Disk: 1},
				{ID: "2", Name: "two", RAM: 200, Disk: 2},
				{ID: "3", Name: "three", RAM: 3, Disk: 3},
			}},
			checks(noError, findsNone),
		},
		{
			"checks disk",
			v1alpha1.FlavorFilter{
				Name: ptr.To[v1alpha1.OpenStackName]("two"),
				RAM:  ptr.To[int32](2),
				Disk: ptr.To[int32](2),
			},
			&flavorLister{[]flavors.Flavor{
				{ID: "1", Name: "one", RAM: 1, Disk: 1},
				{ID: "2", Name: "two", RAM: 2, Disk: -12},
				{ID: "3", Name: "three", RAM: 3, Disk: 3},
			}},
			checks(noError, findsNone),
		},
		{
			"returns lister errors",
			v1alpha1.FlavorFilter{
				Name: ptr.To[v1alpha1.OpenStackName]("one"),
			},
			osclients.NewComputeErrorClient(errors.New("don't panic")),
			checks(errorMessage("don't panic")),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			f, err := flavor.GetByFilter(ctx, tc.lister, tc.filter)

			for _, check := range tc.checks {
				if e := check(f, err); e != nil {
					t.Error(e)
				}
			}
		})
	}
}
