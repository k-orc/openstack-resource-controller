/*
Copyright 2025 The ORC Authors.

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

package listener

import (
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/progress"
	orcapplyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/pkg/clients/applyconfiguration/api/v1alpha1"
)
// TODO(scaffolding): these are just examples. Change them to the controller's need.
// Ideally, these constants are defined in gophercloud.
const ListenerStatusAvailable = "available"
const ListenerStatusInUse     = "in-use"
const ListenerStatusDeleting  = "deleting"

type listenerStatusWriter struct{}

type objectApplyT = orcapplyconfigv1alpha1.ListenerApplyConfiguration
type statusApplyT = orcapplyconfigv1alpha1.ListenerStatusApplyConfiguration

var _ interfaces.ResourceStatusWriter[*orcv1alpha1.Listener, *osResourceT, *objectApplyT, *statusApplyT] = listenerStatusWriter{}

func (listenerStatusWriter) GetApplyConfig(name, namespace string) *objectApplyT {
	return orcapplyconfigv1alpha1.Listener(name, namespace)
}

func (listenerStatusWriter) ResourceAvailableStatus(orcObject *orcv1alpha1.Listener, osResource *osResourceT) (metav1.ConditionStatus, progress.ReconcileStatus) {
	if osResource == nil {
		if orcObject.Status.ID == nil {
			return metav1.ConditionFalse, nil
		} else {
			return metav1.ConditionUnknown, nil
		}
	}
	// TODO(scaffolding): add conditions for returning available, for instance:

	if osResource.Status == ListenerStatusAvailable || osResource.Status == ListenerStatusInUse {
		return metav1.ConditionTrue, nil
	}

	// Otherwise we should continue to poll
	return metav1.ConditionFalse, progress.WaitingOnOpenStack(progress.WaitingOnReady, listenerAvailablePollingPeriod)
}

func (listenerStatusWriter) ApplyResourceStatus(log logr.Logger, osResource *osResourceT, statusApply *statusApplyT) {
	resourceStatus := orcapplyconfigv1alpha1.ListenerResourceStatus().
		WithLoadBalancerID(osResource.LoadBalancerID).
		WithPoolID(osResource.PoolID).
		WithName(osResource.Name)

	// TODO(scaffolding): add all of the fields supported in the ListenerResourceStatus struct
	// If a zero-value isn't expected in the response, place it behind a conditional

	if osResource.Description != "" {
		resourceStatus.WithDescription(osResource.Description)
	}

	statusApply.WithResource(resourceStatus)
}
