package server

import (
	"context"

	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	osclients "github.com/k-orc/openstack-resource-controller/internal/osclients"
	orcerrors "github.com/k-orc/openstack-resource-controller/internal/util/errors"
	"k8s.io/utils/ptr"
)

func specToFilter(orcObject *orcv1alpha1.Server) orcv1alpha1.ServerFilter {
	// gosimple thinks this should be a type conversion (S1016), but we intend
	// these structs to diverge soon so this is better
	return orcv1alpha1.ServerFilter{ //nolint:gosimple
		Name: ptr.To(getResourceName(orcObject)),
	}
}

type serverLister interface {
	ListServers(ctx context.Context, listOpts servers.ListOptsBuilder) <-chan (osclients.Result[*servers.Server])
}

func GetByFilter(ctx context.Context, osClient serverLister, filter orcv1alpha1.ServerFilter) (*servers.Server, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	return osclients.JustOne(
		osClient.ListServers(ctx, servers.ListOpts{
			Name: string(ptr.Deref(filter.Name, "")),
		}),
		orcerrors.Terminal(orcv1alpha1.OpenStackConditionReasonInvalidConfiguration, "found more than one matching server in OpenStack"),
	)
}
