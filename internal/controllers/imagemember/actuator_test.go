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

package imagemember

import (
	"testing"

	"k8s.io/utils/ptr"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/api/v1alpha1"
)

func TestImageRef(t *testing.T) {
	testCases := []struct {
		name string
		spec orcv1alpha1.ImageMemberSpec
		want *orcv1alpha1.KubernetesNameRef
	}{
		{
			name: "Managed",
			spec: orcv1alpha1.ImageMemberSpec{Resource: &orcv1alpha1.ImageMemberResourceSpec{ImageRef: "image"}},
			want: ptr.To[orcv1alpha1.KubernetesNameRef]("image"),
		},
		{
			name: "Import by filter",
			spec: orcv1alpha1.ImageMemberSpec{Import: &orcv1alpha1.ImageMemberImport{Filter: &orcv1alpha1.ImageMemberFilter{ImageRef: ptr.To[orcv1alpha1.KubernetesNameRef]("image")}}},
			want: ptr.To[orcv1alpha1.KubernetesNameRef]("image"),
		},
		{
			name: "Import by ID",
			spec: orcv1alpha1.ImageMemberSpec{Import: &orcv1alpha1.ImageMemberImport{ID: ptr.To("member")}},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			got := imageRef(&orcv1alpha1.ImageMember{Spec: tt.spec})
			if ptr.Deref(got, "") != ptr.Deref(tt.want, "") {
				t.Errorf("Expected %v, got %v", ptr.Deref(tt.want, ""), ptr.Deref(got, ""))
			}
		})
	}
}

func TestDesiredStatus(t *testing.T) {
	member := func(status *orcv1alpha1.ImageMembershipStatus) *orcv1alpha1.ImageMember {
		return &orcv1alpha1.ImageMember{Spec: orcv1alpha1.ImageMemberSpec{Resource: &orcv1alpha1.ImageMemberResourceSpec{Status: status}}}
	}
	if got := desiredStatus(member(nil)); got != orcv1alpha1.ImageMembershipStatusAccepted {
		t.Errorf("Expected accepted by default, got %s", got)
	}
	if got := desiredStatus(member(ptr.To(orcv1alpha1.ImageMembershipStatusRejected))); got != orcv1alpha1.ImageMembershipStatusRejected {
		t.Errorf("Expected rejected, got %s", got)
	}
}
