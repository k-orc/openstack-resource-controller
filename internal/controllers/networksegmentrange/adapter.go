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
	"k8s.io/utils/ptr"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
)

type apiObjectAdapter struct {
	orcObject orcObjectPT
}

var _ adapterI = apiObjectAdapter{}

func (obj apiObjectAdapter) GetResourceSpec() *resourceSpecT {
	return obj.orcObject.Spec.Resource
}

func getResourceName(obj orcObjectPT) string {
	if obj.Spec.Resource != nil && obj.Spec.Resource.Name != nil {
		return string(*obj.Spec.Resource.Name)
	}
	return obj.Name
}
