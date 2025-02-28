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

package server

import (
	"github.com/go-logr/logr"
	"k8s.io/utils/ptr"

	"github.com/k-orc/openstack-resource-controller/internal/controllers/generic"
	orcapplyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/pkg/clients/applyconfiguration/api/v1alpha1"
)

const (
	ServerStatusActive = "ACTIVE"
	ServerStatusError  = "ERROR"
)

type objectApplyPT = *orcapplyconfigv1alpha1.ServerApplyConfiguration
type statusApplyPT = *orcapplyconfigv1alpha1.ServerStatusApplyConfiguration

type serverStatusWriter struct{}

var _ generic.ResourceStatusWriter[orcObjectPT, *osResourceT, objectApplyPT, statusApplyPT] = serverStatusWriter{}

func (serverStatusWriter) GetApplyConfigConstructor() generic.ORCApplyConfigConstructor[objectApplyPT, statusApplyPT] {
	return orcapplyconfigv1alpha1.Server
}

func (serverStatusWriter) ResourceIsAvailable(orcObject orcObjectPT, osResource *osResourceT) bool {
	return orcObject.Status.ID != nil && osResource != nil && osResource.Status == ServerStatusActive
}

func (serverStatusWriter) ApplyResourceStatus(log logr.Logger, osResource *osResourceT, statusApply statusApplyPT) {
	// TODO: Add the rest of the OpenStack data to Status
	status := orcapplyconfigv1alpha1.ServerResourceStatus().
		WithName(osResource.Name).
		WithStatus(osResource.Status).
		WithHostID(osResource.HostID).
		WithAccessIPv4(osResource.AccessIPv4).
		WithAccessIPv6(osResource.AccessIPv6).
		WithTags(ptr.Deref(osResource.Tags, []string{})...)

	statusApply.WithResource(status)
}
