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

package volumesnapshot

import (
	"context"
	"errors"
	"iter"
	"strings"
	"testing"

	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/snapshots"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
)

var errNotImplemented = errors.New("not implemented")

type testVolumeSnapshotClient struct {
	listOpts   snapshots.ListOptsBuilder
	listCalled bool
}

func (c *testVolumeSnapshotClient) ListVolumeSnapshots(_ context.Context, listOpts snapshots.ListOptsBuilder) iter.Seq2[*snapshots.Snapshot, error] {
	c.listCalled = true
	c.listOpts = listOpts
	return func(yield func(*snapshots.Snapshot, error) bool) {}
}

func (c *testVolumeSnapshotClient) CreateVolumeSnapshot(_ context.Context, _ snapshots.CreateOptsBuilder) (*snapshots.Snapshot, error) {
	return nil, errNotImplemented
}

func (c *testVolumeSnapshotClient) DeleteVolumeSnapshot(_ context.Context, _ string) error {
	return errNotImplemented
}

func (c *testVolumeSnapshotClient) GetVolumeSnapshot(_ context.Context, _ string) (*snapshots.Snapshot, error) {
	return nil, errNotImplemented
}

func (c *testVolumeSnapshotClient) UpdateVolumeSnapshot(_ context.Context, _ string, _ snapshots.UpdateOptsBuilder) (*snapshots.Snapshot, error) {
	return nil, errNotImplemented
}

func newTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	scheme := runtime.NewScheme()
	if err := orcv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed adding API scheme: %v", err)
	}
	return scheme
}

func TestNeedsUpdate(t *testing.T) {
	testCases := []struct {
		name         string
		updateOpts   snapshots.UpdateOpts
		expectChange bool
	}{
		{
			name:         "Empty base opts",
			updateOpts:   snapshots.UpdateOpts{},
			expectChange: false,
		},
		{
			name:         "Updated opts",
			updateOpts:   snapshots.UpdateOpts{Name: ptr.To("updated")},
			expectChange: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			got, err := needsUpdate(tt.updateOpts)
			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})
	}
}

func TestHandleNameUpdate(t *testing.T) {
	ptrToName := ptr.To[orcv1alpha1.OpenStackName]
	testCases := []struct {
		name          string
		newValue      *orcv1alpha1.OpenStackName
		existingValue string
		expectChange  bool
	}{
		{name: "Identical", newValue: ptrToName("name"), existingValue: "name", expectChange: false},
		{name: "Different", newValue: ptrToName("new-name"), existingValue: "name", expectChange: true},
		{name: "No value provided, existing is identical to object name", newValue: nil, existingValue: "object-name", expectChange: false},
		{name: "No value provided, existing is different from object name", newValue: nil, existingValue: "different-from-object-name", expectChange: true},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resource := &orcv1alpha1.VolumeSnapshot{}
			resource.Name = "object-name"
			resource.Spec = orcv1alpha1.VolumeSnapshotSpec{
				Resource: &orcv1alpha1.VolumeSnapshotResourceSpec{Name: tt.newValue},
			}
			osResource := &osResourceT{Name: tt.existingValue}

			updateOpts := snapshots.UpdateOpts{}
			handleNameUpdate(&updateOpts, resource, osResource)

			got, err := needsUpdate(updateOpts)
			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})

	}
}

func TestHandleDescriptionUpdate(t *testing.T) {
	ptrToDescription := ptr.To[string]
	testCases := []struct {
		name          string
		newValue      *string
		existingValue string
		expectChange  bool
	}{
		{name: "Identical", newValue: ptrToDescription("desc"), existingValue: "desc", expectChange: false},
		{name: "Different", newValue: ptrToDescription("new-desc"), existingValue: "desc", expectChange: true},
		{name: "No value provided, existing is set", newValue: nil, existingValue: "desc", expectChange: true},
		{name: "No value provided, existing is empty", newValue: nil, existingValue: "", expectChange: false},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resource := &orcv1alpha1.VolumeSnapshotResourceSpec{Description: tt.newValue}
			osResource := &osResourceT{Description: tt.existingValue}

			updateOpts := snapshots.UpdateOpts{}
			handleDescriptionUpdate(&updateOpts, resource, osResource)

			got, err := needsUpdate(updateOpts)
			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})

	}
}

func TestGetVolumeIDForImport(t *testing.T) {
	ctx := context.Background()
	namespace := "default"
	volumeRef := orcv1alpha1.KubernetesNameRef("import-volume")
	orcObject := &orcv1alpha1.VolumeSnapshot{
		ObjectMeta: v1.ObjectMeta{
			Name:      "import-snapshot",
			Namespace: namespace,
		},
	}

	t.Run("waits for missing volume", func(t *testing.T) {
		actuator := volumesnapshotActuator{
			k8sClient: fake.NewClientBuilder().
				WithScheme(newTestScheme(t)).
				Build(),
		}

		volumeID, reconcileStatus := actuator.getVolumeIDForImport(ctx, orcObject, orcv1alpha1.VolumeSnapshotFilter{
			VolumeRef: &volumeRef,
		})
		if volumeID != "" {
			t.Fatalf("expected empty volume ID, got %q", volumeID)
		}
		if reconcileStatus == nil {
			t.Fatalf("expected reconcile status, got nil")
		}
		needsReschedule, err := reconcileStatus.NeedsReschedule()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !needsReschedule {
			t.Fatalf("expected reconcile status to require reschedule")
		}
		if !strings.Contains(strings.Join(reconcileStatus.GetProgressMessages(), "\n"), "Waiting for Volume/import-volume to be created") {
			t.Fatalf("expected waiting for volume creation message, got %v", reconcileStatus.GetProgressMessages())
		}
	})

	t.Run("waits for not-ready volume", func(t *testing.T) {
		volume := &orcv1alpha1.Volume{
			ObjectMeta: v1.ObjectMeta{
				Name:      "import-volume",
				Namespace: namespace,
			},
		}

		actuator := volumesnapshotActuator{
			k8sClient: fake.NewClientBuilder().
				WithScheme(newTestScheme(t)).
				WithObjects(volume).
				Build(),
		}

		volumeID, reconcileStatus := actuator.getVolumeIDForImport(ctx, orcObject, orcv1alpha1.VolumeSnapshotFilter{
			VolumeRef: &volumeRef,
		})
		if volumeID != "" {
			t.Fatalf("expected empty volume ID, got %q", volumeID)
		}
		if reconcileStatus == nil {
			t.Fatalf("expected reconcile status, got nil")
		}
		needsReschedule, err := reconcileStatus.NeedsReschedule()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !needsReschedule {
			t.Fatalf("expected reconcile status to require reschedule")
		}
		if !strings.Contains(strings.Join(reconcileStatus.GetProgressMessages(), "\n"), "Waiting for Volume/import-volume to be ready") {
			t.Fatalf("expected waiting for volume ready message, got %v", reconcileStatus.GetProgressMessages())
		}
	})

	t.Run("returns volume ID when volume is ready", func(t *testing.T) {
		volume := &orcv1alpha1.Volume{
			ObjectMeta: v1.ObjectMeta{
				Name:      "import-volume",
				Namespace: namespace,
			},
			Status: orcv1alpha1.VolumeStatus{
				ID: ptr.To("volume-id"),
				Conditions: []v1.Condition{
					{
						Type:   orcv1alpha1.ConditionAvailable,
						Status: v1.ConditionTrue,
					},
				},
			},
		}

		actuator := volumesnapshotActuator{
			k8sClient: fake.NewClientBuilder().
				WithScheme(newTestScheme(t)).
				WithObjects(volume).
				Build(),
		}

		volumeID, reconcileStatus := actuator.getVolumeIDForImport(ctx, orcObject, orcv1alpha1.VolumeSnapshotFilter{
			VolumeRef: &volumeRef,
		})
		if reconcileStatus != nil {
			t.Fatalf("expected nil reconcile status, got %v", reconcileStatus)
		}
		if volumeID != "volume-id" {
			t.Fatalf("expected volume-id, got %q", volumeID)
		}
	})
}

func TestListOSResourcesForImport_ResolvesVolumeRefToVolumeID(t *testing.T) {
	ctx := context.Background()
	namespace := "default"
	volumeRef := orcv1alpha1.KubernetesNameRef("import-volume")

	volume := &orcv1alpha1.Volume{
		ObjectMeta: v1.ObjectMeta{
			Name:      "import-volume",
			Namespace: namespace,
		},
		Status: orcv1alpha1.VolumeStatus{
			ID: ptr.To("volume-id"),
			Conditions: []v1.Condition{
				{
					Type:   orcv1alpha1.ConditionAvailable,
					Status: v1.ConditionTrue,
				},
			},
		},
	}

	osClient := &testVolumeSnapshotClient{}
	actuator := volumesnapshotActuator{
		osClient: osClient,
		k8sClient: fake.NewClientBuilder().
			WithScheme(newTestScheme(t)).
			WithObjects(volume).
			Build(),
	}

	orcObject := &orcv1alpha1.VolumeSnapshot{
		ObjectMeta: v1.ObjectMeta{
			Name:      "import-snapshot",
			Namespace: namespace,
		},
	}

	filter := orcv1alpha1.VolumeSnapshotFilter{
		Name:      ptr.To(orcv1alpha1.OpenStackName("snapshot-name")),
		VolumeRef: &volumeRef,
	}

	_, reconcileStatus := actuator.ListOSResourcesForImport(ctx, orcObject, filter)
	if reconcileStatus != nil {
		t.Fatalf("expected nil reconcile status, got %v", reconcileStatus)
	}
	if !osClient.listCalled {
		t.Fatalf("expected ListVolumeSnapshots to be called")
	}

	listOpts, ok := osClient.listOpts.(snapshots.ListOpts)
	if !ok {
		t.Fatalf("expected snapshots.ListOpts, got %T", osClient.listOpts)
	}
	if listOpts.VolumeID != "volume-id" {
		t.Fatalf("expected ListOpts.VolumeID to be volume-id, got %q", listOpts.VolumeID)
	}
}
