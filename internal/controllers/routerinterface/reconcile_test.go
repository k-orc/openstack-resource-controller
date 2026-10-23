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

package routerinterface

import (
	"context"
	"iter"
	"testing"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/routers"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v3/internal/osclients"
	"github.com/k-orc/openstack-resource-controller/v3/internal/scope"
)

// A router that has an ID but isn't Available must not block deleting its interfaces (#197).
func TestReconcile_DeletesInterfaceOfNotAvailableRouter(t *testing.T) {
	const (
		namespace = "test"
		routerID  = "router-id"
		subnetID  = "subnet-id"
	)
	credentials := orcv1alpha1.CloudCredentialsReference{SecretName: "openstack-clouds", CloudName: "openstack"}

	router := &orcv1alpha1.Router{
		ObjectMeta: metav1.ObjectMeta{Name: "router", Namespace: namespace, UID: types.UID("router-uid"), Finalizers: []string{finalizer}},
		Spec:       orcv1alpha1.RouterSpec{CloudCredentialsRef: credentials},
		Status:     orcv1alpha1.RouterStatus{ID: ptr.To(routerID)},
	}
	subnet := &orcv1alpha1.Subnet{
		ObjectMeta: metav1.ObjectMeta{Name: "subnet", Namespace: namespace},
		Status:     orcv1alpha1.SubnetStatus{ID: ptr.To(subnetID)},
	}
	deleting := &orcv1alpha1.RouterInterface{
		ObjectMeta: metav1.ObjectMeta{
			Name: "deleting", Namespace: namespace, UID: types.UID("deleting-uid"),
			Finalizers:        []string{finalizer},
			DeletionTimestamp: ptr.To(metav1.Now()),
		},
		Spec: orcv1alpha1.RouterInterfaceSpec{
			Type:      orcv1alpha1.RouterInterfaceTypeSubnet,
			RouterRef: "router",
			SubnetRef: ptr.To[orcv1alpha1.KubernetesNameRef]("subnet"),
		},
	}

	scheme := runtime.NewScheme()
	if err := orcv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&orcv1alpha1.RouterInterface{}).
		WithObjects(router, subnet, deleting).
		WithIndex(&orcv1alpha1.RouterInterface{}, "spec.routerRef", func(obj client.Object) []string {
			return []string{string(obj.(*orcv1alpha1.RouterInterface).Spec.RouterRef)}
		}).
		Build()

	mockCtrl := gomock.NewController(t)
	scopeFactory := scope.NewMockScopeFactory(mockCtrl)
	interfacePort := &osclients.PortExt{Port: ports.Port{
		ID:          "port-id",
		DeviceOwner: "network:router_interface",
		FixedIPs:    []ports.IP{{SubnetID: subnetID}},
	}}
	network := scopeFactory.NetworkClient.EXPECT()
	network.ListPort(gomock.Any(), gomock.Any()).Return(iter.Seq2[*osclients.PortExt, error](
		func(yield func(*osclients.PortExt, error) bool) { yield(interfacePort, nil) }))
	network.RemoveRouterInterface(gomock.Any(), routerID, &routers.RemoveInterfaceOpts{SubnetID: subnetID}).
		Return(&routers.InterfaceInfo{}, nil)

	r := orcRouterInterfaceReconciler{client: k8sClient, scopeFactory: scopeFactory}
	if _, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(router)}); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
}
