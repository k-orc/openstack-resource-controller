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

package neutrontags

import (
	"context"
	"strings"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/attributestags"
	"k8s.io/utils/set"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/generic"
	"github.com/k-orc/openstack-resource-controller/internal/osclients"
)

type StringTag interface {
	orcv1alpha1.NeutronTag | orcv1alpha1.ServerTag | orcv1alpha1.ImageTag
}

// Join joins a slice of tags into a comma separated list of tags.
func Join[T StringTag](tags []T) string {
	var b strings.Builder
	for i := range tags {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(string(tags[i]))
	}
	return b.String()
}

func ReconcileTags[orcObjectPT, osResourceT any](
	networkClient osclients.NetworkClient,
	resourceType string, resourceID string,
	specTags []orcv1alpha1.NeutronTag,
	observedTags []string,
) generic.ResourceReconciler[orcObjectPT, osResourceT] {
	return func(ctx context.Context, _ orcObjectPT, _ *osResourceT) ([]generic.ProgressStatus, error) {
		observedTagSet := set.New(observedTags...)
		specTagSet := set.New[string]()
		for i := range specTags {
			specTagSet.Insert(string(specTags[i]))
		}
		var progressStatus []generic.ProgressStatus
		var err error
		if !specTagSet.Equal(observedTagSet) {
			opts := attributestags.ReplaceAllOpts{Tags: specTagSet.SortedList()}
			_, err = networkClient.ReplaceAllAttributesTags(ctx, resourceType, resourceID, &opts)
			if err == nil {
				// If we updated the tags we need another reconcile to refresh the resource status
				progressStatus = []generic.ProgressStatus{generic.NeedsRefresh()}
			}
		}
		return progressStatus, err
	}
}
