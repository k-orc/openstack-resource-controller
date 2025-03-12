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
	"fmt"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/k-orc/openstack-resource-controller/internal/controllers/generic/interfaces"
	orcapplyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/pkg/clients/applyconfiguration/api/v1alpha1"
)

const (
	ServerStatusActive = "ACTIVE"
	ServerStatusError  = "ERROR"
)

type objectApplyPT = *orcapplyconfigv1alpha1.ServerApplyConfiguration
type statusApplyPT = *orcapplyconfigv1alpha1.ServerStatusApplyConfiguration

type serverStatusWriter struct{}

var _ interfaces.ResourceStatusWriter[orcObjectPT, *osResourceT, objectApplyPT, statusApplyPT] = serverStatusWriter{}

func (serverStatusWriter) GetApplyConfig(name, namespace string) objectApplyPT {
	return orcapplyconfigv1alpha1.Server(name, namespace)
}

func (serverStatusWriter) ResourceAvailableStatus(orcObject orcObjectPT, osResource *osResourceT) metav1.ConditionStatus {
	if osResource == nil {
		if orcObject.Status.ID == nil {
			return metav1.ConditionFalse
		} else {
			return metav1.ConditionUnknown
		}
	}

	if osResource.Status == ServerStatusActive {
		return metav1.ConditionTrue
	}
	return metav1.ConditionFalse
}

func (serverStatusWriter) ApplyResourceStatus(log logr.Logger, osResource *osResourceT, statusApply statusApplyPT) {
	// TODO: Add the rest of the OpenStack data to Status
	status := orcapplyconfigv1alpha1.ServerResourceStatus().
		WithName(osResource.Name).
		WithStatus(osResource.Status).
		WithHostID(osResource.HostID).
		WithTags(ptr.Deref(osResource.Tags, []string{})...)

	if imageID, ok := osResource.Image["id"]; ok {
		status.WithImageID(fmt.Sprintf("%s", imageID))
	}

	statusApply.WithResource(status)
}
