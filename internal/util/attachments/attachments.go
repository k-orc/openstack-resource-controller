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
package attachments

import (
	"context"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v2/internal/logging"
	"github.com/k-orc/openstack-resource-controller/v2/internal/util/applyconfigs"
	orcstrings "github.com/k-orc/openstack-resource-controller/v2/internal/util/strings"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/set"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const SSATransactionAttachment orcstrings.SSATransactionID = "attachment"

func NewAttachableResource[
	objectTP ObjectType[objectT],
	objectT any,
	applyConfigTP ApplyConfigWithSpec[applyConfigTP, specApplyConfigTP],
	specApplyConfigTP ApplyConfigSpecWithAttachments[specApplyConfigTP],
](
	obj objectTP,
	createApplyConfig func(name, namespace string) applyConfigTP,
	createSpecApplyConfig func() specApplyConfigTP,
) AttachableResource[objectTP, objectT, applyConfigTP, specApplyConfigTP] {
	return AttachableResource[objectTP, objectT, applyConfigTP, specApplyConfigTP]{
		obj:                   obj,
		createApplyConfig:     createApplyConfig,
		createSpecApplyConfig: createSpecApplyConfig,
	}
}

// ObjectType is a type constraint for Kubernetes objects that can be attached.
type ObjectType[objectT any] interface {
	*objectT
	client.Object
	GetAttachments() []orcv1alpha1.KubernetesNameRef
}

type ApplyConfigWithSpec[applyConfigTP any, specApplyConfigTP any] interface {
	WithSpec(specApplyConfigTP) applyConfigTP
}

type ApplyConfigSpecWithAttachments[specApplyConfigTP any] interface {
	WithAttachments(...orcv1alpha1.KubernetesNameRef) specApplyConfigTP
}

type AttachableResource[
	objectTP ObjectType[objectT],
	objectT any,
	applyConfigTP ApplyConfigWithSpec[applyConfigTP, specApplyConfigTP],
	specApplyConfigTP ApplyConfigSpecWithAttachments[specApplyConfigTP],
] struct {
	obj                   objectTP
	createApplyConfig     func(name, namespace string) applyConfigTP
	createSpecApplyConfig func() specApplyConfigTP
}

// AttachTo adds the target of the attachment to the list of attachments for the resource.
func (a *AttachableResource[objectTP, objectT, applyConfigTP, specApplyConfigTP]) AttachTo(ctx context.Context, k8sClient client.Client, controllerName, target string) {
	log := ctrl.LoggerFrom(ctx)
	applyConfig := a.createApplyConfig(a.obj.GetName(), a.obj.GetNamespace())

	attachments := set.New(a.obj.GetAttachments()...)
	attachments.Insert(orcv1alpha1.KubernetesNameRef(target))

	spec := a.createSpecApplyConfig().WithAttachments(attachments.SortedList()...)

	applyConfig = applyConfig.WithSpec(spec)

	ssaFieldOwner := orcstrings.GetSSAFieldOwnerWithTxn(controllerName, SSATransactionAttachment)
	err := k8sClient.Patch(ctx, a.obj, applyconfigs.Patch(types.ApplyPatchType, applyConfig), client.ForceOwnership, ssaFieldOwner)
	if err != nil {
		// This is best effort, we're not returning the error
		log.V(logging.Debug).Info("error patching volume", "key", a.obj.GetName(), "error", err)
	}
}

// DetachFrom removes the target of the attachment from the list of attachments for the resource.
func (a *AttachableResource[objectTP, objectT, applyConfigTP, specApplyConfigTP]) DetachFrom(ctx context.Context, k8sClient client.Client, controllerName, target string) {
	log := ctrl.LoggerFrom(ctx)
	applyConfig := a.createApplyConfig(a.obj.GetName(), a.obj.GetNamespace())

	attachments := set.New(a.obj.GetAttachments()...)
	attachments.Delete(orcv1alpha1.KubernetesNameRef(target))

	spec := a.createSpecApplyConfig().WithAttachments(attachments.SortedList()...)

	applyConfig = applyConfig.WithSpec(spec)

	ssaFieldOwner := orcstrings.GetSSAFieldOwnerWithTxn(controllerName, SSATransactionAttachment)
	err := k8sClient.Patch(ctx, a.obj, applyconfigs.Patch(types.ApplyPatchType, applyConfig), client.ForceOwnership, ssaFieldOwner)
	if err != nil {
		// This is best effort, we're not returning the error
		log.V(logging.Debug).Info("error patching volume", "key", a.obj.GetName(), "error", err)
	}
}
