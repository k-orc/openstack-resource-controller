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

package flavor

import (
	"github.com/go-logr/logr"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/flavors"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/generic"
	orcapplyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/pkg/clients/applyconfiguration/api/v1alpha1"
)

type flavorStatusWriter struct{}

type objectApplyT = orcapplyconfigv1alpha1.FlavorApplyConfiguration
type statusApplyT = orcapplyconfigv1alpha1.FlavorStatusApplyConfiguration

var _ generic.ResourceStatusWriter[*orcv1alpha1.Flavor, *flavors.Flavor, *objectApplyT, *statusApplyT] = flavorStatusWriter{}

func (flavorStatusWriter) GetApplyConfigConstructor() generic.ORCApplyConfigConstructor[*objectApplyT, *statusApplyT] {
	return orcapplyconfigv1alpha1.Flavor
}

func (flavorStatusWriter) GetCommonStatus(_ *orcv1alpha1.Flavor, osResource *flavors.Flavor) (bool, bool) {
	available := osResource != nil
	return available, available
}

func (flavorStatusWriter) ApplyResourceStatus(_ logr.Logger, osResource *flavors.Flavor, statusApply *statusApplyT) {
	statusApply.WithResource(orcapplyconfigv1alpha1.FlavorResourceStatus().
		WithName(osResource.Name).
		WithDescription(osResource.Description).
		WithRAM(int32(osResource.RAM)).
		WithDisk(int32(osResource.Disk)).
		WithVcpus(int32(osResource.VCPUs)).
		WithSwap(int32(osResource.Swap)).
		WithIsPublic(osResource.IsPublic).
		WithEphemeral(int32(osResource.Ephemeral)))
}
