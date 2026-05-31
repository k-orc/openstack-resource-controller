package flavor

import (
	"testing"

	"k8s.io/utils/ptr"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
)

func TestFlavorAdapterIsImported(t *testing.T) {
	tests := []struct {
		name string
		spec orcv1alpha1.FlavorSpec
		want bool
	}{
		{
			name: "no import - created by ORC",
			spec: orcv1alpha1.FlavorSpec{
				Resource: &orcv1alpha1.FlavorResourceSpec{},
			},
			want: false,
		},
		{
			name: "import is nil - created by ORC",
			spec: orcv1alpha1.FlavorSpec{
				Import: nil,
			},
			want: false,
		},
		{
			name: "import with non-nil ID",
			spec: orcv1alpha1.FlavorSpec{
				Import: &orcv1alpha1.FlavorImport{
					ID: ptr.To("some-uuid-1234"),
				},
			},
			want: true,
		},
		{
			name: "import with non-nil filter",
			spec: orcv1alpha1.FlavorSpec{
				Import: &orcv1alpha1.FlavorImport{
					Filter: &orcv1alpha1.FlavorFilter{
						Name: (*orcv1alpha1.OpenStackName)(ptr.To("my-flavor")),
					},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := flavorAdapter{
				Flavor: &orcv1alpha1.Flavor{
					Spec: tt.spec,
				},
			}
			got := adapter.IsImported()
			if got != tt.want {
				t.Errorf("IsImported() = %v, want %v", got, tt.want)
			}
		})
	}
}
