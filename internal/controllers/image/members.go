/*
Copyright The ORC Authors.

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

package image

import (
	"context"
	"slices"

	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/images"
	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/members"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v3/internal/controllers/generic/progress"
	"github.com/k-orc/openstack-resource-controller/v3/internal/logging"
	orcerrors "github.com/k-orc/openstack-resource-controller/v3/internal/util/errors"
)

// reconcileMembers makes the members of the image match spec.resource.members.
func (actuator imageActuator) reconcileMembers(ctx context.Context, obj orcObjectPT, osResource *osResourceT) progress.ReconcileStatus {
	log := ctrl.LoggerFrom(ctx)
	resource := obj.Spec.Resource
	if resource == nil {
		return nil
	}

	if osResource.Visibility != images.ImageVisibilityShared {
		if len(resource.Members) == 0 {
			return nil
		}
		// Wait for UpdateResource to change the visibility
		if ptr.Deref(resource.Visibility, "") == orcv1alpha1.ImageVisibilityShared {
			return progress.NeedsRefresh()
		}
		return progress.WrapError(orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "members requires visibility shared"))
	}

	for i := range resource.Members {
		if resource.Members[i].UserRef != nil {
			return progress.WrapError(orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "members.userRef is not supported yet"))
		}
	}

	projects, reconcileStatus := projectDependency.RequireDependencies(ctx, actuator.k8sClient, obj, func(project *orcv1alpha1.Project) bool {
		return orcv1alpha1.IsAvailable(project) && project.Status.ID != nil
	})
	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return reconcileStatus
	}

	desired := make([]string, 0, len(resource.Members))
	for i := range resource.Members {
		desired = append(desired, *projects[string(*resource.Members[i].ProjectRef)].Status.ID)
	}

	current := osResource.Members
	if current == nil {
		var err error
		current, err = actuator.listMembers(ctx, osResource.ID)
		if err != nil {
			return progress.WrapError(err)
		}
	}

	add, remove := diffMembers(desired, current)
	for _, memberID := range add {
		log.V(logging.Verbose).Info("Adding image member", "member", memberID)
		if _, err := actuator.osClient.CreateImageMember(ctx, osResource.ID, memberID); err != nil {
			return progress.WrapError(err)
		}
	}
	for _, memberID := range remove {
		log.V(logging.Verbose).Info("Removing image member", "member", memberID)
		if err := actuator.osClient.DeleteImageMember(ctx, osResource.ID, memberID); err != nil && !orcerrors.IsNotFound(err) {
			return progress.WrapError(err)
		}
	}

	if len(add) > 0 || len(remove) > 0 || osResource.Members == nil {
		return progress.NeedsRefresh()
	}
	return nil
}

// diffMembers returns the member IDs to add to and remove from current.
func diffMembers(desired []string, current []members.Member) (add, remove []string) {
	currentIDs := make([]string, len(current))
	for i := range current {
		currentIDs[i] = current[i].MemberID
	}
	for _, memberID := range desired {
		if !slices.Contains(currentIDs, memberID) && !slices.Contains(add, memberID) {
			add = append(add, memberID)
		}
	}
	for _, memberID := range currentIDs {
		if !slices.Contains(desired, memberID) {
			remove = append(remove, memberID)
		}
	}
	return add, remove
}
