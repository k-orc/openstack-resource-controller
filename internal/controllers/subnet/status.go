/*
Copyright 2024 The ORC Authors.

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

package subnet

import (
	"github.com/go-logr/logr"

	"github.com/k-orc/openstack-resource-controller/internal/controllers/generic/interfaces"
	orcapplyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/pkg/clients/applyconfiguration/api/v1alpha1"
)

type objectApplyPT = *orcapplyconfigv1alpha1.SubnetApplyConfiguration
type statusApplyPT = *orcapplyconfigv1alpha1.SubnetStatusApplyConfiguration

type subnetStatusWriter struct{}

var _ interfaces.ResourceStatusWriter[orcObjectPT, *osResourceT, objectApplyPT, statusApplyPT] = subnetStatusWriter{}

func (subnetStatusWriter) GetApplyConfigConstructor() interfaces.ORCApplyConfigConstructor[objectApplyPT, statusApplyPT] {
	return orcapplyconfigv1alpha1.Subnet
}

func (subnetStatusWriter) ResourceIsAvailable(orcObject orcObjectPT, osResource *osResourceT) bool {
	// Subnet is available as soon as it exists
	return osResource != nil
}

func (subnetStatusWriter) ApplyResourceStatus(log logr.Logger, osResource *osResourceT, statusApply statusApplyPT) {
	status := orcapplyconfigv1alpha1.SubnetResourceStatus().
		WithName(osResource.Name).
		WithDescription(osResource.Description).
		WithIPVersion(int32(osResource.IPVersion)).
		WithCIDR(osResource.CIDR).
		WithGatewayIP(osResource.GatewayIP).
		WithDNSPublishFixedIP(osResource.DNSPublishFixedIP).
		WithEnableDHCP(osResource.EnableDHCP).
		WithProjectID(osResource.ProjectID).
		WithRevisionNumber(int64(osResource.RevisionNumber)).
		WithIPv6AddressMode(osResource.IPv6AddressMode).
		WithIPv6RAMode(osResource.IPv6RAMode).
		WithTags(osResource.Tags...).
		WithDNSNameservers(osResource.DNSNameservers...)

	for i := range osResource.AllocationPools {
		status.WithAllocationPools(orcapplyconfigv1alpha1.AllocationPoolStatus().
			WithStart(osResource.AllocationPools[i].Start).
			WithEnd(osResource.AllocationPools[i].End))
	}

	for i := range osResource.HostRoutes {
		status.WithHostRoutes(orcapplyconfigv1alpha1.HostRouteStatus().
			WithDestination(osResource.HostRoutes[i].DestinationCIDR).
			WithNextHop(osResource.HostRoutes[i].NextHop))
	}

	statusApply.WithResource(status)
}
