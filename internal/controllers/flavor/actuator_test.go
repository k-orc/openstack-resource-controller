package flavor

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"testing"

	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/flavors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/internal/osclients"
)

var (
	errNotImplemented = errors.New("not implemented")
	errTest           = errors.New("test error")
)

type mockFlavorClient struct {
	flavors []flavors.Flavor
}

var _ flavorClient = mockFlavorClient{}

func (l mockFlavorClient) ListFlavors(_ context.Context, _ flavors.ListOptsBuilder) iter.Seq2[*flavors.Flavor, error] {
	return func(yield func(*flavors.Flavor, error) bool) {
		for i := range l.flavors {
			if !yield(&l.flavors[i], nil) {
				return
			}
		}
	}
}

func (l mockFlavorClient) GetFlavor(_ context.Context, _ string) (*flavors.Flavor, error) {
	return nil, errNotImplemented
}

func (l mockFlavorClient) CreateFlavor(_ context.Context, _ flavors.CreateOptsBuilder) (*flavors.Flavor, error) {
	return nil, errNotImplemented
}

func (l mockFlavorClient) DeleteFlavor(_ context.Context, _ string) error {
	return errNotImplemented
}

type flavorResult struct {
	flavor *flavors.Flavor
	err    error
}

type checkFunc func([]flavorResult) error

func checks(fns ...checkFunc) []checkFunc { return fns }

func noError(results []flavorResult) error {
	for _, result := range results {
		if result.err != nil {
			return fmt.Errorf("unexpected error: %w", result.err)
		}
	}
	return nil
}

func wantError(wantErr error) checkFunc {
	return func(results []flavorResult) error {
		for _, result := range results {
			if result.err == nil {
				continue
			}
			if errors.Is(result.err, wantErr) {
				return nil
			} else {
				return fmt.Errorf("unexpected error message: %w", result.err)
			}
		}
		return nil
	}
}

func findsN(wantN int) checkFunc {
	return func(results []flavorResult) error {
		found := len(results)
		if found != wantN {
			return fmt.Errorf("expected no results, got %d", found)
		}
		return nil
	}
}

func findsID(wantID string) checkFunc {
	return func(results []flavorResult) error {
		for _, result := range results {
			if result.flavor == nil {
				continue
			}
			if result.flavor.ID == wantID {
				return nil
			}
		}
		return fmt.Errorf("did not find flavor with id %s", wantID)
	}
}

func TestGetFlavorByImportFilter(t *testing.T) {
	for _, tc := range [...]struct {
		name   string
		filter orcv1alpha1.FlavorFilter
		client flavorClient
		checks []checkFunc
	}{
		{
			"finds one by name",
			orcv1alpha1.FlavorFilter{Name: ptr.To[orcv1alpha1.OpenStackName]("one")},
			&mockFlavorClient{[]flavors.Flavor{
				{ID: "1", Name: "one"},
				{ID: "2", Name: "two"},
				{ID: "3", Name: "three"},
			}},
			checks(noError, findsID("1"), findsN(1)),
		},
		{
			"finds none by name",
			orcv1alpha1.FlavorFilter{Name: ptr.To[orcv1alpha1.OpenStackName]("four")},
			&mockFlavorClient{[]flavors.Flavor{
				{ID: "1", Name: "one"},
				{ID: "2", Name: "two"},
				{ID: "3", Name: "three"},
			}},
			checks(noError, findsN(0)),
		},
		{
			"finds multiple",
			orcv1alpha1.FlavorFilter{Name: ptr.To[orcv1alpha1.OpenStackName]("one")},
			&mockFlavorClient{[]flavors.Flavor{
				{ID: "1", Name: "one"},
				{ID: "2", Name: "two"},
				{ID: "3", Name: "one"},
			}},
			checks(noError, findsN(2)),
		},
		{
			"finds one by RAM and disk",
			orcv1alpha1.FlavorFilter{RAM: ptr.To[int32](2), Disk: ptr.To[int32](2)},
			&mockFlavorClient{[]flavors.Flavor{
				{ID: "1", RAM: 1, Disk: 1, VCPUs: 1},
				{ID: "2", RAM: 2, Disk: 2, VCPUs: 2},
				{ID: "3", RAM: 3, Disk: 3, VCPUs: 3},
			}},
			checks(noError, findsID("2"), findsN(1)),
		},
		{
			"finds one by name, VCPUs, RAM and disk",
			orcv1alpha1.FlavorFilter{
				Name:  ptr.To[orcv1alpha1.OpenStackName]("two"),
				RAM:   ptr.To[int32](2),
				Disk:  ptr.To[int32](2),
				Vcpus: ptr.To[int32](2),
			},
			&mockFlavorClient{[]flavors.Flavor{
				{ID: "1", Name: "one", RAM: 1, Disk: 1, VCPUs: 1},
				{ID: "2", Name: "two", RAM: 2, Disk: 2, VCPUs: 2},
				{ID: "3", Name: "three", RAM: 3, Disk: 3, VCPUs: 3},
			}},
			checks(noError, findsID("2"), findsN(1)),
		},
		{
			"checks RAM",
			orcv1alpha1.FlavorFilter{
				Name: ptr.To[orcv1alpha1.OpenStackName]("two"),
				RAM:  ptr.To[int32](2),
				Disk: ptr.To[int32](2),
			},
			&mockFlavorClient{[]flavors.Flavor{
				{ID: "1", Name: "one", RAM: 1, Disk: 1},
				{ID: "2", Name: "two", RAM: 200, Disk: 2},
				{ID: "3", Name: "three", RAM: 3, Disk: 3},
			}},
			checks(noError, findsN(0)),
		},
		{
			"checks disk",
			orcv1alpha1.FlavorFilter{
				Name: ptr.To[orcv1alpha1.OpenStackName]("two"),
				RAM:  ptr.To[int32](2),
				Disk: ptr.To[int32](2),
			},
			&mockFlavorClient{[]flavors.Flavor{
				{ID: "1", Name: "one", RAM: 1, Disk: 1},
				{ID: "2", Name: "two", RAM: 2, Disk: -12},
				{ID: "3", Name: "three", RAM: 3, Disk: 3},
			}},
			checks(noError, findsN(0)),
		},
		{
			"returns lister errors",
			orcv1alpha1.FlavorFilter{
				Name: ptr.To[orcv1alpha1.OpenStackName]("one"),
			},
			osclients.NewComputeErrorClient(errTest),
			checks(wantError(errTest)),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			actuator := flavorActuator{tc.client}
			_, flavorIter, _ := actuator.ListOSResourcesForImport(ctx, &orcv1alpha1.Flavor{
				ObjectMeta: metav1.ObjectMeta{
					Name: "flavor",
				},
				Spec: orcv1alpha1.FlavorSpec{
					Import: &orcv1alpha1.FlavorImport{
						Filter: &tc.filter,
					},
				},
			}, tc.filter)

			var flavorResults []flavorResult
			for flavor, err := range flavorIter {
				flavorResults = append(flavorResults, flavorResult{flavor, err})
			}

			for _, check := range tc.checks {
				if e := check(flavorResults); e != nil {
					t.Error(e)
				}
			}
		})
	}
}

func TestGetFlavorBySpec(t *testing.T) {
	for _, tc := range [...]struct {
		name         string
		resourceName string
		resourceSpec orcv1alpha1.FlavorResourceSpec
		client       flavorClient
		checks       []checkFunc
	}{
		{
			"finds one by resource name",
			"foo",
			orcv1alpha1.FlavorResourceSpec{Name: ptr.To[orcv1alpha1.OpenStackName]("one")},
			&mockFlavorClient{[]flavors.Flavor{
				{ID: "1", Name: "one"},
				{ID: "2", Name: "two"},
				{ID: "3", Name: "three"},
			}},
			checks(noError, findsID("1"), findsN(1)),
		},
		{
			"finds one by object name",
			"one",
			orcv1alpha1.FlavorResourceSpec{},
			&mockFlavorClient{[]flavors.Flavor{
				{ID: "1", Name: "one"},
				{ID: "2", Name: "two"},
				{ID: "3", Name: "three"},
			}},
			checks(noError, findsID("1"), findsN(1)),
		},
		{
			"finds none by name",
			"four",
			orcv1alpha1.FlavorResourceSpec{},
			&mockFlavorClient{[]flavors.Flavor{
				{ID: "1", Name: "one"},
				{ID: "2", Name: "two"},
				{ID: "3", Name: "three"},
			}},
			checks(noError, findsN(0)),
		},
		{
			"finds multiple",
			"one",
			orcv1alpha1.FlavorResourceSpec{},
			&mockFlavorClient{[]flavors.Flavor{
				{ID: "1", Name: "one"},
				{ID: "2", Name: "two"},
				{ID: "3", Name: "one"},
			}},
			checks(noError, findsN(2)),
		},
		{
			"finds one by RAM, disk, and VCPU",
			"foo",
			orcv1alpha1.FlavorResourceSpec{RAM: 2, Disk: 2, Vcpus: 2},
			&mockFlavorClient{[]flavors.Flavor{
				{ID: "1", Name: "foo", RAM: 1, Disk: 1, VCPUs: 1},
				{ID: "2", Name: "foo", RAM: 2, Disk: 2, VCPUs: 2},
				{ID: "3", Name: "foo", RAM: 3, Disk: 3, VCPUs: 3},
			}},
			checks(noError, findsID("2"), findsN(1)),
		},
		{
			"finds one by name, VCPUs, RAM and disk",
			"two",
			orcv1alpha1.FlavorResourceSpec{
				RAM:   2,
				Disk:  2,
				Vcpus: 2,
			},
			&mockFlavorClient{[]flavors.Flavor{
				{ID: "1", Name: "one", RAM: 1, Disk: 1, VCPUs: 1},
				{ID: "2", Name: "two", RAM: 2, Disk: 2, VCPUs: 2},
				{ID: "3", Name: "three", RAM: 3, Disk: 3, VCPUs: 3},
			}},
			checks(noError, findsID("2"), findsN(1)),
		},
		{
			"checks RAM",
			"two",
			orcv1alpha1.FlavorResourceSpec{
				RAM:  2,
				Disk: 2,
			},
			&mockFlavorClient{[]flavors.Flavor{
				{ID: "1", Name: "one", RAM: 1, Disk: 1},
				{ID: "2", Name: "two", RAM: 200, Disk: 2},
				{ID: "3", Name: "three", RAM: 3, Disk: 3},
			}},
			checks(noError, findsN(0)),
		},
		{
			"checks disk",
			"two",
			orcv1alpha1.FlavorResourceSpec{
				RAM:  2,
				Disk: 2,
			},
			&mockFlavorClient{[]flavors.Flavor{
				{ID: "1", Name: "one", RAM: 1, Disk: 1},
				{ID: "2", Name: "two", RAM: 2, Disk: -12},
				{ID: "3", Name: "three", RAM: 3, Disk: 3},
			}},
			checks(noError, findsN(0)),
		},
		{
			"returns lister errors",
			"one",
			orcv1alpha1.FlavorResourceSpec{},
			osclients.NewComputeErrorClient(errTest),
			checks(wantError(errTest)),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			actuator := flavorActuator{tc.client}
			flavorIter, canAdopt := actuator.ListOSResourcesForAdoption(ctx, &orcv1alpha1.Flavor{
				ObjectMeta: metav1.ObjectMeta{
					Name: tc.resourceName,
				},
				Spec: orcv1alpha1.FlavorSpec{
					Resource: &tc.resourceSpec,
				},
			})
			if !canAdopt {
				t.Errorf("canAdopt should be true")
			}

			var flavorResults []flavorResult
			for flavor, err := range flavorIter {
				flavorResults = append(flavorResults, flavorResult{flavor, err})
			}

			for _, check := range tc.checks {
				if e := check(flavorResults); e != nil {
					t.Error(e)
				}
			}
		})
	}
}
