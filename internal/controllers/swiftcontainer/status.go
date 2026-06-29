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

package swiftcontainer

import (
	"sort"
	"strings"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/progress"
	orcapplyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/pkg/clients/applyconfiguration/api/v1alpha1"
)

type swiftcontainerStatusWriter struct{}

type objectApplyT = orcapplyconfigv1alpha1.SwiftContainerApplyConfiguration
type statusApplyT = orcapplyconfigv1alpha1.SwiftContainerStatusApplyConfiguration

var _ interfaces.ResourceStatusWriter[*orcv1alpha1.SwiftContainer, *osContainerT, *objectApplyT, *statusApplyT] = swiftcontainerStatusWriter{}

func (swiftcontainerStatusWriter) GetApplyConfig(name, namespace string) *objectApplyT {
	return orcapplyconfigv1alpha1.SwiftContainer(name, namespace)
}

func (swiftcontainerStatusWriter) ResourceAvailableStatus(orcObject *orcv1alpha1.SwiftContainer, osResource *osContainerT) (metav1.ConditionStatus, progress.ReconcileStatus) {
	if osResource == nil {
		if orcObject.Status.ID == nil {
			return metav1.ConditionFalse, nil
		}
		return metav1.ConditionUnknown, nil
	}

	// SwiftContainer is available as soon as it exists
	return metav1.ConditionTrue, nil
}

func (swiftcontainerStatusWriter) ApplyResourceStatus(_ logr.Logger, osResource *osContainerT, statusApply *statusApplyT) {
	resourceStatus := orcapplyconfigv1alpha1.SwiftContainerResourceStatus().
		WithName(osResource.Name).
		WithBytesUsed(osResource.BytesUsed).
		WithObjectCount(osResource.ObjectCount)

	// containerRead and containerWrite are stored as []string (parsed from
	// comma-separated ACL headers); join them back for status reporting.
	// Only set the status field if the resulting ACL string is non-empty;
	// gophercloud may return []string{""} for an absent header, which
	// should not be surfaced as an empty-string ACL in status.
	if readACL := strings.Join(osResource.Read, ","); readACL != "" {
		resourceStatus.WithContainerRead(readACL)
	}
	if writeACL := strings.Join(osResource.Write, ","); writeACL != "" {
		resourceStatus.WithContainerWrite(writeACL)
	}

	if osResource.StoragePolicy != "" {
		resourceStatus.WithStoragePolicy(osResource.StoragePolicy)
	}
	if osResource.VersionsLocation != "" {
		resourceStatus.WithVersions(osResource.VersionsLocation)
	}

	// Populate observed metadata from X-Container-Meta-* headers.
	// Keys are sorted to ensure deterministic output.
	if len(osResource.Metadata) > 0 {
		keys := make([]string, 0, len(osResource.Metadata))
		for k := range osResource.Metadata {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			resourceStatus.WithMetadata(
				orcapplyconfigv1alpha1.SwiftContainerMetadataStatus().
					WithKey(k).
					WithValue(osResource.Metadata[k]),
			)
		}
	}

	statusApply.WithResource(resourceStatus)
}
