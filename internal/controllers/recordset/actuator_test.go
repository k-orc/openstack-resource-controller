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

package recordset

import (
	"context"
	"errors"
	"testing"

	"github.com/gophercloud/gophercloud/v2/openstack/dns/v2/recordsets"
	"go.uber.org/mock/gomock"
	"k8s.io/utils/ptr"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v3/internal/osclients/mock"
)

func Test_recordsetActuator_updateResource(t *testing.T) {
	const (
		zoneID      = "939da9ca-27c2-4fa6-881f-17f9038f8107"
		recordsetID = "265c9e4f-0f5a-46e4-9f3f-fb8de25ae120"
	)

	updateError := errors.New("test update error")

	orcObjectWith := func(description *string, ttl *int32, records []string) orcObjectPT {
		return &orcv1alpha1.RecordSet{
			Spec: orcv1alpha1.RecordSetSpec{
				Resource: &orcv1alpha1.RecordSetResourceSpec{
					Type:        "A",
					Description: description,
					TTL:         ttl,
					Records:     records,
				},
			},
		}
	}

	osResourceWith := func(description string, ttl int, records []string) *osResourceT {
		return &recordsets.RecordSet{
			ID:          recordsetID,
			ZoneID:      zoneID,
			Type:        "A",
			Description: description,
			TTL:         ttl,
			Records:     records,
		}
	}

	tests := []struct {
		name           string
		orcObject      orcObjectPT
		osResource     *osResourceT
		expect         func(*mock.MockRecordSetClientMockRecorder)
		wantReschedule bool
		wantErr        error
	}{
		{
			name:       "no changes, no update call",
			orcObject:  orcObjectWith(ptr.To("desc"), ptr.To(int32(300)), []string{"192.0.2.1"}),
			osResource: osResourceWith("desc", 300, []string{"192.0.2.1"}),
		},
		{
			name:       "records changed (as a set, order-independent), calls UpdateRecordSet",
			orcObject:  orcObjectWith(nil, nil, []string{"192.0.2.2", "192.0.2.1"}),
			osResource: osResourceWith("", 0, []string{"192.0.2.1"}),
			expect: func(recorder *mock.MockRecordSetClientMockRecorder) {
				recorder.UpdateRecordSet(gomock.Any(), zoneID, recordsetID, recordsets.UpdateOpts{Records: []string{"192.0.2.2", "192.0.2.1"}}).
					Return(nil, nil)
			},
			wantReschedule: true,
		},
		{
			name:       "records unchanged despite different order, no update call",
			orcObject:  orcObjectWith(nil, nil, []string{"192.0.2.2", "192.0.2.1"}),
			osResource: osResourceWith("", 0, []string{"192.0.2.1", "192.0.2.2"}),
		},
		{
			name:       "ttl changed, calls UpdateRecordSet",
			orcObject:  orcObjectWith(nil, ptr.To(int32(600)), nil),
			osResource: osResourceWith("", 300, nil),
			expect: func(recorder *mock.MockRecordSetClientMockRecorder) {
				ttl := 600
				recorder.UpdateRecordSet(gomock.Any(), zoneID, recordsetID, recordsets.UpdateOpts{TTL: &ttl}).
					Return(nil, nil)
			},
			wantReschedule: true,
		},
		{
			name:       "update error is propagated",
			orcObject:  orcObjectWith(ptr.To("new-desc"), nil, nil),
			osResource: osResourceWith("old-desc", 0, nil),
			expect: func(recorder *mock.MockRecordSetClientMockRecorder) {
				desc := "new-desc"
				recorder.UpdateRecordSet(gomock.Any(), zoneID, recordsetID, recordsets.UpdateOpts{Description: &desc}).
					Return(nil, updateError)
			},
			// A plain error (not classified non-retryable by orcerrors.IsRetryable) defaults to
			// retryable, so the generic reconciler still reschedules.
			wantReschedule: true,
			wantErr:        updateError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockctrl := gomock.NewController(t)
			dnsClient := mock.NewMockRecordSetClient(mockctrl)

			actuator := recordsetActuator{osClient: dnsClient}

			recorder := dnsClient.EXPECT()
			if tt.expect != nil {
				tt.expect(recorder)
			}

			reconcileStatus := actuator.updateResource(context.TODO(), tt.orcObject, tt.osResource)
			needsReschedule, err := reconcileStatus.NeedsReschedule()

			if tt.wantErr == nil && err != nil {
				t.Errorf("updateResource() error = %v, want no error", err)
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Errorf("updateResource() error = %v, want %v", err, tt.wantErr)
			}
			if needsReschedule != tt.wantReschedule {
				t.Errorf("updateResource() needsReschedule = %v, want %v", needsReschedule, tt.wantReschedule)
			}
		})
	}
}
