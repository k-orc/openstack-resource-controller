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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	applyconfigv1 "k8s.io/client-go/applyconfigurations/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-logr/logr"
	"github.com/k-orc/openstack-resource-controller/internal/util/applyconfigs"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
)

type ORCApplyConfig[objectApplyPT any, statusApplyPT ORCStatusApplyConfig[statusApplyPT]] interface {
	WithUID(types.UID) objectApplyPT
	WithStatus(statusApplyPT) objectApplyPT
}

type ORCStatusApplyConfig[statusApplyPT any] interface {
	WithConditions(...*applyconfigv1.ConditionApplyConfiguration) statusApplyPT
	WithID(id string) statusApplyPT
}

type ORCApplyConfigConstructor[objectApplyPT ORCApplyConfig[objectApplyPT, statusApplyPT], statusApplyPT ORCStatusApplyConfig[statusApplyPT]] func(name, namespace string) objectApplyPT

type ResourceStatusWriter[objectPT orcv1alpha1.ObjectWithConditions, osResourcePT any, objectApplyPT ORCApplyConfig[objectApplyPT, statusApplyPT], statusApplyPT ORCStatusApplyConfig[statusApplyPT]] interface {
	GetApplyConfigConstructor() ORCApplyConfigConstructor[objectApplyPT, statusApplyPT]
	ResourceIsAvailable(orcObject objectPT, osResource osResourcePT) bool
	ApplyResourceStatus(log logr.Logger, osResource osResourcePT, statusApply statusApplyPT)
}

func SetStatusID[
	orcObjectPT interface {
		client.Object
		orcv1alpha1.ObjectWithConditions
	},
	objectApplyPT ORCApplyConfig[objectApplyPT, statusApplyPT],
	statusApplyPT interface {
		*statusApplyT
		ORCStatusApplyConfig[statusApplyPT]
	},
	statusApplyT any,
	osResourcePT any,
](
	ctx context.Context,
	controller ResourceController,
	orcObject orcObjectPT,
	resourceID string,
	statusWriter ResourceStatusWriter[orcObjectPT, osResourcePT, objectApplyPT, statusApplyPT],
) error {
	var status statusApplyPT = new(statusApplyT)
	status.WithID(resourceID)

	applyConfig := statusWriter.GetApplyConfigConstructor()(orcObject.GetName(), orcObject.GetNamespace()).
		WithUID(orcObject.GetUID()).
		WithStatus(status)

	return controller.GetK8sClient().Status().Patch(ctx, orcObject, applyconfigs.Patch(types.MergePatchType, applyConfig))
}

func UpdateStatus[
	orcObjectPT interface {
		client.Object
		orcv1alpha1.ObjectWithConditions
	},
	osResourcePT *osResourceT,
	objectApplyPT ORCApplyConfig[objectApplyPT, statusApplyPT],
	statusApplyPT interface {
		ORCStatusApplyConfig[statusApplyPT]
		*statusApply
	},
	statusApply any,
	osResourceT any,
](
	ctx context.Context,
	controller ResourceController,
	statusWriter ResourceStatusWriter[orcObjectPT, osResourcePT, objectApplyPT, statusApplyPT],
	orcObject orcObjectPT, osResource osResourcePT, progressStatus []ProgressStatus, err error,
) error {
	log := ctrl.LoggerFrom(ctx)
	now := metav1.NewTime(time.Now())

	// Create a new apply configuration for this status transaction
	var applyConfigStatus statusApplyPT = new(statusApply)
	applyConfig := statusWriter.GetApplyConfigConstructor()(orcObject.GetName(), orcObject.GetNamespace()).
		WithStatus(applyConfigStatus)

	// Write resource status to the apply configuration
	if osResource != nil {
		statusWriter.ApplyResourceStatus(log, osResource, applyConfigStatus)
	}

	// Set common conditions
	available := statusWriter.ResourceIsAvailable(orcObject, osResource)
	SetCommonConditions(orcObject, applyConfigStatus, available, progressStatus, err, now)

	// Patch orcObject with the status transaction
	k8sClient := controller.GetK8sClient()
	ssaFieldOwner := GetSSAFieldOwnerWithTxn(controller.GetName(), SSATransactionStatus)
	return k8sClient.Status().Patch(ctx, orcObject, applyconfigs.Patch(types.ApplyPatchType, applyConfig), client.ForceOwnership, ssaFieldOwner)
}
