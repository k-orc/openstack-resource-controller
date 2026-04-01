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

package user

import (
	"context"
	"testing"

	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/users"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v2/internal/osclients/mock"
)

func TestNeedsUpdate(t *testing.T) {
	testCases := []struct {
		name         string
		updateOpts   users.UpdateOpts
		expectChange bool
	}{
		{
			name:         "Empty base opts",
			updateOpts:   users.UpdateOpts{},
			expectChange: false,
		},
		{
			name:         "Updated opts",
			updateOpts:   users.UpdateOpts{Name: "updated"},
			expectChange: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := needsUpdate(tt.updateOpts)
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
			resource := &orcv1alpha1.User{}
			resource.Name = "object-name"
			resource.Spec = orcv1alpha1.UserSpec{
				Resource: &orcv1alpha1.UserResourceSpec{Name: tt.newValue},
			}
			osResource := &osResourceT{Name: tt.existingValue}

			updateOpts := users.UpdateOpts{}
			handleNameUpdate(&updateOpts, resource, osResource)

			got, _ := needsUpdate(updateOpts)
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
			resource := &orcv1alpha1.UserResourceSpec{Description: tt.newValue}
			osResource := &osResourceT{Description: tt.existingValue}

			updateOpts := users.UpdateOpts{}
			handleDescriptionUpdate(&updateOpts, resource, osResource)

			got, _ := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})

	}
}

func TestHandleEnabledUpdate(t *testing.T) {
	ptrToBool := ptr.To[bool]
	testCases := []struct {
		name          string
		newValue      *bool
		existingValue bool
		expectChange  bool
	}{
		{name: "Identical", newValue: ptrToBool(true), existingValue: true, expectChange: false},
		{name: "Different", newValue: ptrToBool(true), existingValue: false, expectChange: true},
		{name: "No value provided, existing is set", newValue: nil, existingValue: false, expectChange: true},
		{name: "No value provided, existing is default", newValue: nil, existingValue: true, expectChange: false},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resource := &orcv1alpha1.UserResourceSpec{Enabled: tt.newValue}
			osResource := &users.User{Enabled: tt.existingValue}

			updateOpts := users.UpdateOpts{}
			handleEnabledUpdate(&updateOpts, resource, osResource)

			got, _ := needsUpdate(updateOpts)
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})
	}
}

func TestReconcilePassword(t *testing.T) {
	ptrToPasswordRef := ptr.To[orcv1alpha1.KubernetesNameRef]
	testCases := []struct {
		name           string
		orcObject      *orcv1alpha1.User
		osResource     *users.User
		secret         *corev1.Secret
		setupMock      func(*mock.MockUserClientMockRecorder)
		wantReschedule bool
		wantErr        bool
	}{
		{
			name: "No password ref set",
			orcObject: &orcv1alpha1.User{
				Spec: orcv1alpha1.UserSpec{
					Resource: &orcv1alpha1.UserResourceSpec{},
				},
			},
			osResource:     &users.User{ID: "user-id"},
			wantReschedule: false,
			wantErr:        false,
		},
		{
			name: "Resource is nil",
			orcObject: &orcv1alpha1.User{
				Spec: orcv1alpha1.UserSpec{},
			},
			osResource:     &users.User{ID: "user-id"},
			wantReschedule: false,
			wantErr:        false,
		},
		{
			name: "Password ref unchanged",
			orcObject: &orcv1alpha1.User{
				Spec: orcv1alpha1.UserSpec{
					Resource: &orcv1alpha1.UserResourceSpec{
						PasswordRef: ptrToPasswordRef("my-secret"),
					},
				},
				Status: orcv1alpha1.UserStatus{
					Resource: &orcv1alpha1.UserResourceStatus{
						AppliedPasswordRef: "my-secret",
					},
				},
			},
			osResource:     &users.User{ID: "user-id"},
			wantReschedule: false,
			wantErr:        false,
		},
		{
			name: "First password set - no UpdateUser call",
			orcObject: &orcv1alpha1.User{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "test-ns",
					UID:       "test-uid",
				},
				Spec: orcv1alpha1.UserSpec{
					Resource: &orcv1alpha1.UserResourceSpec{
						PasswordRef: ptrToPasswordRef("my-secret"),
					},
				},
				Status: orcv1alpha1.UserStatus{
					Resource: &orcv1alpha1.UserResourceStatus{
						AppliedPasswordRef: "",
					},
				},
			},
			osResource: &users.User{ID: "user-id"},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-secret",
					Namespace: "test-ns",
				},
				Data: map[string][]byte{
					"password": []byte("mypassword123"),
				},
			},
			// No UpdateUser call expected on first reconcile
			setupMock:      func(recorder *mock.MockUserClientMockRecorder) {},
			wantReschedule: false,
			wantErr:        false,
		},
		{
			name: "Password changed - UpdateUser called",
			orcObject: &orcv1alpha1.User{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "test-ns",
					UID:       "test-uid",
				},
				Spec: orcv1alpha1.UserSpec{
					Resource: &orcv1alpha1.UserResourceSpec{
						PasswordRef: ptrToPasswordRef("my-secret"),
					},
				},
				Status: orcv1alpha1.UserStatus{
					Resource: &orcv1alpha1.UserResourceStatus{
						AppliedPasswordRef: "old-secret",
					},
				},
			},
			osResource: &users.User{ID: "user-id"},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-secret",
					Namespace: "test-ns",
				},
				Data: map[string][]byte{
					"password": []byte("newpassword456"),
				},
			},
			setupMock: func(recorder *mock.MockUserClientMockRecorder) {
				recorder.UpdateUser(gomock.Any(), "user-id", gomock.Any()).Return(&users.User{}, nil)
			},
			wantReschedule: true, // NeedsRefresh returns true
			wantErr:        false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			mockctrl := gomock.NewController(t)
			userClient := mock.NewMockUserClient(mockctrl)

			// Create fake k8s client
			scheme := runtime.NewScheme()
			_ = corev1.AddToScheme(scheme)
			_ = orcv1alpha1.AddToScheme(scheme)

			objects := []client.Object{tt.orcObject}
			if tt.secret != nil {
				objects = append(objects, tt.secret)
			}

			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(objects...).
				WithStatusSubresource(&orcv1alpha1.User{}).
				Build()

			actuator := userActuator{
				osClient:  userClient,
				k8sClient: k8sClient,
			}

			if tt.setupMock != nil {
				tt.setupMock(userClient.EXPECT())
			}

			reconcileStatus := actuator.reconcilePassword(context.TODO(), tt.orcObject, tt.osResource)

			needsReschedule, err := reconcileStatus.NeedsReschedule()
			if (err != nil) != tt.wantErr {
				t.Errorf("reconcilePassword() error = %v, wantErr %v", err, tt.wantErr)
			}
			if needsReschedule != tt.wantReschedule {
				t.Errorf("reconcilePassword() needsReschedule = %v, want %v", needsReschedule, tt.wantReschedule)
			}
		})
	}
}
