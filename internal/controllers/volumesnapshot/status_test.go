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
	"errors"
	"testing"

	"github.com/go-logr/logr"
	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	orcerrors "github.com/k-orc/openstack-resource-controller/v2/internal/util/errors"
	orcapplyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/pkg/clients/applyconfiguration/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestApplyResourceStatus_ConsumesQuotaFalseIsPreserved(t *testing.T) {
	writer := volumesnapshotStatusWriter{}
	statusApply := orcapplyconfigv1alpha1.VolumeSnapshotStatus()

	writer.ApplyResourceStatus(logr.Discard(), &osResourceT{
		Name:          "snapshot-1",
		VolumeID:      "volume-1",
		Status:        SnapshotStatusAvailable,
		Size:          1,
		ConsumesQuota: false,
	}, statusApply)

	if statusApply.Resource == nil {
		t.Fatalf("expected status resource to be set")
	}
	if statusApply.Resource.ConsumesQuota == nil {
		t.Fatalf("expected consumesQuota to be present, got nil")
	}
	if *statusApply.Resource.ConsumesQuota {
		t.Fatalf("expected consumesQuota=false, got true")
	}
}

func TestApplyResourceStatus_MetadataOrderIsDeterministic(t *testing.T) {
	writer := volumesnapshotStatusWriter{}
	statusApply := orcapplyconfigv1alpha1.VolumeSnapshotStatus()

	writer.ApplyResourceStatus(logr.Discard(), &osResourceT{
		Name:     "snapshot-1",
		VolumeID: "volume-1",
		Status:   SnapshotStatusAvailable,
		Size:     1,
		Metadata: map[string]string{
			"z-key": "z-value",
			"a-key": "a-value",
		},
	}, statusApply)

	if statusApply.Resource == nil {
		t.Fatalf("expected status resource to be set")
	}
	if len(statusApply.Resource.Metadata) != 2 {
		t.Fatalf("expected 2 metadata items, got %d", len(statusApply.Resource.Metadata))
	}

	firstName := statusApply.Resource.Metadata[0].Name
	secondName := statusApply.Resource.Metadata[1].Name
	if firstName == nil || secondName == nil {
		t.Fatalf("expected metadata names to be set")
	}
	if *firstName != "a-key" || *secondName != "z-key" {
		t.Fatalf("expected metadata sorted by key, got %q then %q", *firstName, *secondName)
	}
}

func TestResourceAvailableStatus_ErrorIsTerminal(t *testing.T) {
	writer := volumesnapshotStatusWriter{}
	orcObject := &orcv1alpha1.VolumeSnapshot{}

	available, reconcileStatus := writer.ResourceAvailableStatus(orcObject, &osResourceT{
		Status: SnapshotStatusError,
	})
	if available != metav1.ConditionFalse {
		t.Fatalf("expected available condition false, got %q", available)
	}
	if reconcileStatus == nil {
		t.Fatalf("expected terminal reconcile status, got nil")
	}

	err := reconcileStatus.GetError()
	if err == nil {
		t.Fatalf("expected terminal error, got nil")
	}

	var terminalErr *orcerrors.TerminalError
	if !errors.As(err, &terminalErr) {
		t.Fatalf("expected terminal error type, got %T", err)
	}
	if terminalErr.Reason != orcv1alpha1.ConditionReasonUnrecoverableError {
		t.Fatalf("expected reason %q, got %q", orcv1alpha1.ConditionReasonUnrecoverableError, terminalErr.Reason)
	}
}
