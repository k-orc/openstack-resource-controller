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

package network

import (
	"strconv"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/generic"
	"github.com/k-orc/openstack-resource-controller/internal/osclients"
	orcapplyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/pkg/clients/applyconfiguration/api/v1alpha1"
)

const (
	NetworkStatusActive = "ACTIVE"
)

type networkStatusWriter struct{}

var _ generic.ResourceStatusWriter[*orcv1alpha1.Network, *osclients.NetworkExt, *orcapplyconfigv1alpha1.NetworkApplyConfiguration, *orcapplyconfigv1alpha1.NetworkStatusApplyConfiguration] = networkStatusWriter{}

func (networkStatusWriter) GetApplyConfigConstructor() generic.ORCApplyConfigConstructor[*orcapplyconfigv1alpha1.NetworkApplyConfiguration, *orcapplyconfigv1alpha1.NetworkStatusApplyConfiguration] {
	return orcapplyconfigv1alpha1.Network
}

func (networkStatusWriter) GetCommonStatus(orcObject *orcv1alpha1.Network, osResource *osclients.NetworkExt) (bool, bool) {
	available := osResource != nil && osResource.Status == NetworkStatusActive
	return available, available
}

func (networkStatusWriter) ApplyResourceStatus(log logr.Logger, osResource *osclients.NetworkExt, statusApply *orcapplyconfigv1alpha1.NetworkStatusApplyConfiguration) {
	networkResourceStatus := orcapplyconfigv1alpha1.NetworkResourceStatus().
		WithName(osResource.Name).
		WithDescription(osResource.Description).
		WithAdminStateUp(osResource.AdminStateUp).
		WithAvailabilityZoneHints(osResource.AvailabilityZoneHints...).
		WithStatus(osResource.Status).
		WithProjectID(osResource.ProjectID).
		WithTags(osResource.Tags...).
		WithDNSDomain(osResource.DNSDomain).
		WithRevisionNumber(int64(osResource.RevisionNumber)).
		WithExternal(osResource.External).
		WithSubnets(osResource.Subnets...).
		WithMTU(int32(osResource.MTU)).
		WithPortSecurityEnabled(osResource.PortSecurityEnabled).
		WithShared(osResource.Shared).
		WithCreatedAt(metav1.NewTime(osResource.CreatedAt)).
		WithUpdatedAt(metav1.NewTime(osResource.UpdatedAt))

	if osResource.NetworkType != "" {
		providerProperties := orcapplyconfigv1alpha1.ProviderPropertiesStatus().
			WithNetworkType(osResource.NetworkType).
			WithPhysicalNetwork(osResource.PhysicalNetwork)

		if osResource.SegmentationID != "" {
			segmentationID, err := strconv.ParseInt(osResource.SegmentationID, 10, 32)
			if err != nil {
				log.V(3).Error(err, "Invalid segmentation ID", "segmentationID", osResource.SegmentationID)
			} else {
				providerProperties.WithSegmentationID(int32(segmentationID))
			}
		}
		networkResourceStatus.WithProvider(providerProperties)
	}

	statusApply.WithResource(networkResourceStatus)
}
