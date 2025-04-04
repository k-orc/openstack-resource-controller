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

package image

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/images"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v2/internal/util/applyconfigs"
	orcstrings "github.com/k-orc/openstack-resource-controller/v2/internal/util/strings"
	orcapplyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/pkg/clients/applyconfiguration/api/v1alpha1"
)

const (
	glanceOSHashAlgo  = "os_hash_algo"
	glanceOSHashValue = "os_hash_value"

	SSATransactionDownloadingStatus orcstrings.SSATransactionID = "downloadingstatus"
)

type objectApplyPT = *orcapplyconfigv1alpha1.ImageApplyConfiguration
type statusApplyPT = *orcapplyconfigv1alpha1.ImageStatusApplyConfiguration

type imageStatusWriter struct{}

var _ interfaces.ResourceStatusWriter[orcObjectPT, *osResourceT, objectApplyPT, statusApplyPT] = imageStatusWriter{}

func (imageStatusWriter) GetApplyConfig(name, namespace string) objectApplyPT {
	return orcapplyconfigv1alpha1.Image(name, namespace)
}

func (imageStatusWriter) ResourceAvailableStatus(orcObject orcObjectPT, osResource *osResourceT) metav1.ConditionStatus {
	if osResource == nil {
		if orcObject.Status.ID == nil {
			return metav1.ConditionFalse
		} else {
			return metav1.ConditionUnknown
		}
	}

	if osResource.Status == images.ImageStatusActive {
		return metav1.ConditionTrue
	}
	return metav1.ConditionFalse
}

func (imageStatusWriter) ApplyResourceStatus(log logr.Logger, osResource *osResourceT, statusApply statusApplyPT) {
	resourceStatus := orcapplyconfigv1alpha1.ImageResourceStatus().
		WithName(osResource.Name).
		WithStatus(string(osResource.Status)).
		WithProtected(osResource.Protected).
		WithVisibility(string(osResource.Visibility)).
		WithSizeB(osResource.SizeBytes).
		WithTags(osResource.Tags...)

	osHashAlgo, _ := osResource.Properties[glanceOSHashAlgo].(string)
	osHashValue, _ := osResource.Properties[glanceOSHashValue].(string)
	if osHashAlgo != "" && osHashValue != "" {
		resourceStatus.WithHash(orcapplyconfigv1alpha1.ImageHash().
			WithAlgorithm(orcv1alpha1.ImageHashAlgorithm(osHashAlgo)).
			WithValue(osHashValue))
	}

	statusApply.WithResource(resourceStatus)
}

func setDownloadingStatus(ctx context.Context, attempts int32, orcObject orcObjectPT, k8sClient client.Client) error {
	var status statusApplyPT = new(orcapplyconfigv1alpha1.ImageStatusApplyConfiguration)
	status.WithDownloadAttempts(attempts)

	applyConfig := orcapplyconfigv1alpha1.Image(orcObject.GetName(), orcObject.GetNamespace()).
		WithUID(orcObject.GetUID()).
		WithStatus(status)

	ssaFieldOwner := orcstrings.GetSSAFieldOwnerWithTxn(controllerName, SSATransactionDownloadingStatus)
	return k8sClient.Status().Patch(ctx, orcObject, applyconfigs.Patch(types.ApplyPatchType, applyConfig), client.ForceOwnership, ssaFieldOwner)
}
