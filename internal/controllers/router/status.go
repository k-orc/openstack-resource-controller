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

package router

import (
	"github.com/go-logr/logr"

	"github.com/k-orc/openstack-resource-controller/internal/controllers/generic"
	orcapplyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/pkg/clients/applyconfiguration/api/v1alpha1"
)

const (
	RouterStatusActive = "ACTIVE"
)

type objectApplyPT = *orcapplyconfigv1alpha1.RouterApplyConfiguration
type statusApplyPT = *orcapplyconfigv1alpha1.RouterStatusApplyConfiguration

type routerStatusWriter struct{}

var _ generic.ResourceStatusWriter[orcObjectPT, osResourcePT, objectApplyPT, statusApplyPT] = routerStatusWriter{}

func (routerStatusWriter) GetApplyConfigConstructor() generic.ORCApplyConfigConstructor[objectApplyPT, statusApplyPT] {
	return orcapplyconfigv1alpha1.Router
}

func (routerStatusWriter) GetCommonStatus(orcObject orcObjectPT, osResource osResourcePT) (bool, bool) {
	available := orcObject.Status.ID != nil && osResource != nil && osResource.Status == RouterStatusActive
	return available, available
}

func (routerStatusWriter) ApplyResourceStatus(log logr.Logger, osResource osResourcePT, statusApply statusApplyPT) {
	status := orcapplyconfigv1alpha1.RouterResourceStatus().
		WithName(osResource.Name).
		WithDescription(osResource.Description).
		WithProjectID(osResource.ProjectID).
		WithStatus(osResource.Status).
		WithTags(osResource.Tags...).
		WithAdminStateUp(osResource.AdminStateUp).
		WithAvailabilityZoneHints(osResource.AvailabilityZoneHints...)

	if osResource.GatewayInfo.NetworkID != "" {
		status.WithExternalGateways(orcapplyconfigv1alpha1.ExternalGatewayStatus().
			WithNetworkID(osResource.GatewayInfo.NetworkID))
	}

	statusApply.WithResource(status)
}
