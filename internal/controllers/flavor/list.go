package flavor

import (
	"context"

	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/flavors"
	"github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	osclients "github.com/k-orc/openstack-resource-controller/internal/osclients"
	orcerrors "github.com/k-orc/openstack-resource-controller/internal/util/errors"
	"k8s.io/utils/ptr"
)

func specToFilter(resourceSpec v1alpha1.FlavorResourceSpec) v1alpha1.FlavorFilter {
	return v1alpha1.FlavorFilter{
		Name: resourceSpec.Name,
		RAM:  &resourceSpec.RAM,
		Disk: &resourceSpec.Disk,
	}
}

type flavorLister interface {
	ListFlavors(ctx context.Context, listOpts flavors.ListOptsBuilder) <-chan (osclients.Result[*flavors.Flavor])
}

func GetByFilter(ctx context.Context, osClient flavorLister, filter v1alpha1.FlavorFilter) (*flavors.Flavor, error) {
	filterFuncs := make([]func(*flavors.Flavor) bool, 0, 3)

	if filter.Name != nil {
		filterFuncs = append(filterFuncs, func(f *flavors.Flavor) bool { return f.Name == string(*filter.Name) })
	}

	if filter.RAM != nil {
		filterFuncs = append(filterFuncs, func(f *flavors.Flavor) bool { return f.RAM == int(*filter.RAM) })
	}

	if filter.Disk != nil {
		filterFuncs = append(filterFuncs, func(f *flavors.Flavor) bool { return f.Disk == int(*filter.Disk) })
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	return osclients.JustOne(
		osclients.Filter(
			osClient.ListFlavors(
				ctx, flavors.ListOpts{
					MinDisk: int(ptr.Deref(filter.Disk, 0)),
					MinRAM:  int(ptr.Deref(filter.RAM, 0)),
				}),
			filterFuncs...),
		orcerrors.Terminal(v1alpha1.OpenStackConditionReasonInvalidConfiguration, "found more than one matching flavor in OpenStack"),
	)
}
