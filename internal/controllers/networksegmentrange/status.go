/*
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

package networksegmentrange

import (
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	//orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	//orcapplyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/applyconfiguration/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/progress"
)

// Temporarily disabled until code generation completes
/*
type (
	statusApplyPT = *orcapplyconfigv1alpha1.NetworkSegmentRangeStatusApplyConfiguration
	objectApplyPT = *orcapplyconfigv1alpha1.NetworkSegmentRangeApplyConfiguration
)

type networksegmentrangeStatusWriter struct{}

func (networksegmentrangeStatusWriter) GetApplyConfig(name, namespace string) objectApplyPT {
	return orcapplyconfigv1alpha1.NetworkSegmentRange(name, namespace)
}

func (networksegmentrangeStatusWriter) ResourceAvailableStatus(orcObject orcObjectPT, osResource *osResourceT) (metav1.ConditionStatus, progress.ReconcileStatus) {
	return metav1.ConditionTrue, nil
}

func (networksegmentrangeStatusWriter) ApplyResourceStatus(log logr.Logger, osResource *osResourceT, statusApply statusApplyPT) {
	statusApply.WithName(osResource.Name).
		WithNetworkType(osResource.NetworkType).
		WithPhysicalNetwork(osResource.PhysicalNetwork).
		WithMinimum(osResource.Minimum).
		WithMaximum(osResource.Maximum).
		WithShared(osResource.Shared).
		WithDefault(osResource.Default).
		WithProjectID(osResource.ProjectID)
}
*/
