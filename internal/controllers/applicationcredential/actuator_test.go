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

package applicationcredential

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v3/internal/scope"
)

type lookupTestController struct {
	client.Client
	*scope.MockScopeFactory
}

func (lookupTestController) GetName() string                  { return controllerName }
func (c lookupTestController) GetK8sClient() client.Client    { return c.Client }
func (c lookupTestController) GetScopeFactory() scope.Factory { return c.MockScopeFactory }

func TestGetOSResourceByID(t *testing.T) {
	lookupError := errors.New("lookup failed")
	k8sError := errors.New("kubernetes lookup failed")
	for _, tc := range []struct {
		name         string
		importSpec   *orcv1alpha1.ApplicationCredentialImport
		missingUser  bool
		userID       *string
		getUserError error
		lookupError  error
		wantWait     bool
	}{
		{name: "managed owner need not be available", userID: ptr.To("owner-id")},
		{name: "managed owner not found", missingUser: true, wantWait: true},
		{name: "managed owner has no ID", wantWait: true},
		{name: "managed owner has empty ID", userID: ptr.To(""), wantWait: true},
		{name: "owner fetch error", getUserError: k8sError},
		{name: "owner credential not found", userID: ptr.To("owner-id"), lookupError: &gophercloud.ErrUnexpectedResponseCode{Actual: http.StatusNotFound}},
		{name: "owner credential forbidden", userID: ptr.To("owner-id"), lookupError: &gophercloud.ErrUnexpectedResponseCode{Actual: http.StatusForbidden}},
		{name: "owner credential unavailable", userID: ptr.To("owner-id"), lookupError: &gophercloud.ErrUnexpectedResponseCode{Actual: http.StatusServiceUnavailable}},
		{name: "ID import uses discovery", importSpec: &orcv1alpha1.ApplicationCredentialImport{ID: ptr.To("credential-id")}},
		{name: "filter import uses discovery without owner dependency", importSpec: &orcv1alpha1.ApplicationCredentialImport{Filter: &orcv1alpha1.ApplicationCredentialFilter{UserRef: "missing"}}},
		{name: "import discovery error", importSpec: &orcv1alpha1.ApplicationCredentialImport{ID: ptr.To("credential-id")}, lookupError: lookupError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			if err := orcv1alpha1.AddToScheme(scheme); err != nil {
				t.Fatal(err)
			}
			if err := corev1.AddToScheme(scheme); err != nil {
				t.Fatal(err)
			}
			obj := &orcv1alpha1.ApplicationCredential{
				ObjectMeta: metav1.ObjectMeta{Name: "credential", Namespace: "test"},
				Spec: orcv1alpha1.ApplicationCredentialSpec{
					CloudCredentialsRef: orcv1alpha1.CloudCredentialsReference{SecretName: "clouds", CloudName: "openstack"},
					ManagementPolicy:    orcv1alpha1.ManagementPolicyManaged,
					Resource:            &orcv1alpha1.ApplicationCredentialResourceSpec{UserRef: "owner"},
				},
			}
			if tc.importSpec != nil {
				obj.Spec.ManagementPolicy = orcv1alpha1.ManagementPolicyUnmanaged
				obj.Spec.Resource = nil
				obj.Spec.Import = tc.importSpec
			}
			secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "clouds", Namespace: "test", Finalizers: []string{finalizer}}}
			user := &orcv1alpha1.User{
				ObjectMeta: metav1.ObjectMeta{Name: "owner", Namespace: "test"},
				Status:     orcv1alpha1.UserStatus{ID: tc.userID},
			}
			builder := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret)
			if !tc.missingUser {
				builder.WithObjects(user)
			}
			backingClient := builder.Build()
			k8sClient := interceptor.NewClient(backingClient, interceptor.Funcs{
				Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
					if _, ok := obj.(*orcv1alpha1.User); ok {
						if tc.importSpec != nil {
							t.Fatal("import lookup must not fetch User")
						}
						if tc.getUserError != nil {
							return tc.getUserError
						}
					}
					return c.Get(ctx, key, obj, opts...)
				},
			})
			factory := scope.NewMockScopeFactory(gomock.NewController(t))
			actuator, rs := newActuator(t.Context(), obj, lookupTestController{k8sClient, factory})
			if needsReschedule, err := rs.NeedsReschedule(); needsReschedule {
				t.Fatalf("newActuator unexpectedly requires a dependency: %v, %v", rs.GetProgressMessages(), err)
			}
			var wantResource *osResourceT
			if !tc.wantWait && tc.getUserError == nil {
				if tc.lookupError == nil {
					wantResource = &osResourceT{ID: "credential-id"}
				}
				if tc.importSpec != nil {
					factory.ApplicationCredentialClient.EXPECT().GetApplicationCredential(gomock.Any(), "credential-id").Return(wantResource, tc.lookupError)
				} else {
					factory.ApplicationCredentialClient.EXPECT().GetApplicationCredentialForUser(gomock.Any(), "owner-id", "credential-id").Return(wantResource, tc.lookupError)
				}
			}
			resource, rs := actuator.GetOSResourceByID(t.Context(), "credential-id")
			wantError := tc.lookupError
			if tc.getUserError != nil {
				wantError = tc.getUserError
			}
			if !errors.Is(rs.GetError(), wantError) {
				t.Errorf("error = %v, want %v", rs.GetError(), wantError)
			}
			if resource != wantResource {
				t.Errorf("resource = %v, want %v", resource, wantResource)
			}
			if needsReschedule, _ := rs.NeedsReschedule(); needsReschedule != (tc.wantWait || wantError != nil) {
				t.Errorf("NeedsReschedule = %v", needsReschedule)
			}
			if !tc.missingUser {
				// Use the backing client to bypass the fetch-error interceptor.
				stored := &orcv1alpha1.User{}
				if err := backingClient.Get(t.Context(), client.ObjectKeyFromObject(user), stored); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(stored.Finalizers, user.Finalizers) {
					t.Errorf("read added finalizers: %v", stored.Finalizers)
				}
			}
		})
	}
}
