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

package generic

import (
	"context"

	"github.com/k-orc/openstack-resource-controller/internal/util/applyconfigs"
	"k8s.io/apimachinery/pkg/types"
)

type ORCApplyConfig[objectApplyPT any, statusApplyPT ORCStatusApplyConfig[statusApplyPT]] interface {
	WithUID(types.UID) objectApplyPT
	WithStatus(statusApplyPT) objectApplyPT
}

type ORCStatusApplyConfig[statusApplyPT any] interface {
	WithID(id string) statusApplyPT
}

type ORCApplyConfigConstructor[objectApplyPT ORCApplyConfig[objectApplyPT, statusApplyPT], statusApplyPT ORCStatusApplyConfig[statusApplyPT]] func(name, namespace string) objectApplyPT

func SetStatusID[
	osResourcePT any,
	objectApplyPT ORCApplyConfig[objectApplyPT, statusApplyPT],
	statusApplyPT interface {
		*statusApplyT
		ORCStatusApplyConfig[statusApplyPT]
	},
	statusApplyT any,
](ctx context.Context, actuator interface {
	BaseResourceActuator[osResourcePT]
	ResourceStatusWriter[objectApplyPT, statusApplyPT]
}, osResource osResourcePT) error {
	var status statusApplyPT = new(statusApplyT)
	status.WithID(actuator.GetResourceID(osResource))

	orcObject := actuator.GetObject()
	applyConfig := actuator.GetApplyConfigConstructor()(orcObject.GetName(), orcObject.GetNamespace()).
		WithUID(orcObject.GetUID()).
		WithStatus(status.
			WithID(actuator.GetResourceID(osResource)))

	k8sClient := actuator.GetController().GetK8sClient()
	return k8sClient.Status().Patch(ctx, orcObject, applyconfigs.Patch(types.MergePatchType, applyConfig))
}

type ResourceStatusWriter[objectApplyPT ORCApplyConfig[objectApplyPT, statusApplyPT], statusApplyPT ORCStatusApplyConfig[statusApplyPT]] interface {
	GetApplyConfigConstructor() ORCApplyConfigConstructor[objectApplyPT, statusApplyPT]
	//GetStatusUpdateConfig(ctx context.Context, now metav1.Time, orcObject orcObjectPT, osResource osResourcePT, waitEvents []WaitingOnEvent, err error) applyConfig
}

/*
func UpdateStatus[orcObjectPT client.Object, osResourcePT any, applyConfig any](ctx context.Context, controller ResourceControllerCommon, statusWriter ResourceStatusWriter[orcObjectPT, osResourcePT, applyConfig], orcObject orcObjectPT, osResource osResourcePT, waitEvents []WaitingOnEvent, err error) error {
	now := metav1.NewTime(time.Now())

	statusUpdate := statusWriter.GetStatusUpdateConfig(ctx, now, orcObject, osResource, waitEvents, err)

	k8sClient := controller.GetK8sClient()
	ssaFieldOwner := GetSSAFieldOwnerWithTxn(controller, SSATransactionStatus)

	return k8sClient.Status().Patch(ctx, orcObject, applyconfigs.Patch(types.ApplyPatchType, statusUpdate), client.ForceOwnership, ssaFieldOwner)
}
*/
