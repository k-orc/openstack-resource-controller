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

package port

import (
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/k-orc/openstack-resource-controller/internal/controllers/generic"
	orcapplyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/pkg/clients/applyconfiguration/api/v1alpha1"
)

type objectApplyPT = *orcapplyconfigv1alpha1.PortApplyConfiguration
type statusApplyPT = *orcapplyconfigv1alpha1.PortStatusApplyConfiguration

type portStatusWriter struct{}

var _ generic.ResourceStatusWriter[orcObjectPT, *osResourceT, objectApplyPT, statusApplyPT] = portStatusWriter{}

func (portStatusWriter) GetApplyConfigConstructor() generic.ORCApplyConfigConstructor[objectApplyPT, statusApplyPT] {
	return orcapplyconfigv1alpha1.Port
}

func (portStatusWriter) GetCommonStatus(_ orcObjectPT, osResource *osResourceT) (bool, bool) {
	// A port is available as soon as it exists
	available := osResource != nil
	return available, available
}

func (portStatusWriter) ApplyResourceStatus(log logr.Logger, osResource *osResourceT, statusApply statusApplyPT) {
	resourceStatus := orcapplyconfigv1alpha1.PortResourceStatus().
		WithName(osResource.Name).
		WithDescription(osResource.Description).
		WithAdminStateUp(osResource.AdminStateUp).
		WithMACAddress(osResource.MACAddress).
		WithDeviceID(osResource.DeviceID).
		WithDeviceOwner(osResource.DeviceOwner).
		WithStatus(osResource.Status).
		WithProjectID(osResource.ProjectID).
		WithTags(osResource.Tags...).
		WithSecurityGroups(osResource.SecurityGroups...).
		WithPropagateUplinkStatus(osResource.PropagateUplinkStatus).
		WithRevisionNumber(int64(osResource.RevisionNumber)).
		WithCreatedAt(metav1.NewTime(osResource.CreatedAt)).
		WithUpdatedAt(metav1.NewTime(osResource.UpdatedAt))

	if len(osResource.AllowedAddressPairs) > 0 {
		allowedAddressPairs := make([]*orcapplyconfigv1alpha1.AllowedAddressPairStatusApplyConfiguration, len(osResource.AllowedAddressPairs))
		for i := range osResource.AllowedAddressPairs {
			allowedAddressPairs[i] = orcapplyconfigv1alpha1.AllowedAddressPairStatus().
				WithIP(osResource.AllowedAddressPairs[i].IPAddress).
				WithMAC(osResource.AllowedAddressPairs[i].MACAddress)
		}
		resourceStatus.WithAllowedAddressPairs(allowedAddressPairs...)
	}

	if len(osResource.FixedIPs) > 0 {
		fixedIPs := make([]*orcapplyconfigv1alpha1.FixedIPStatusApplyConfiguration, len(osResource.FixedIPs))
		for i := range osResource.FixedIPs {
			fixedIPs[i] = orcapplyconfigv1alpha1.FixedIPStatus().
				WithIP(osResource.FixedIPs[i].IPAddress).
				WithSubnetID(osResource.FixedIPs[i].SubnetID)
		}
		resourceStatus.WithFixedIPs(fixedIPs...)
	}

	statusApply.WithResource(resourceStatus)
}
