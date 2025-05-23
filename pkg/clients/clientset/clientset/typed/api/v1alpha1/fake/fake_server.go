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

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	v1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	apiv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/pkg/clients/applyconfiguration/api/v1alpha1"
	typedapiv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/pkg/clients/clientset/clientset/typed/api/v1alpha1"
	gentype "k8s.io/client-go/gentype"
)

// fakeServers implements ServerInterface
type fakeServers struct {
	*gentype.FakeClientWithListAndApply[*v1alpha1.Server, *v1alpha1.ServerList, *apiv1alpha1.ServerApplyConfiguration]
	Fake *FakeOpenstackV1alpha1
}

func newFakeServers(fake *FakeOpenstackV1alpha1, namespace string) typedapiv1alpha1.ServerInterface {
	return &fakeServers{
		gentype.NewFakeClientWithListAndApply[*v1alpha1.Server, *v1alpha1.ServerList, *apiv1alpha1.ServerApplyConfiguration](
			fake.Fake,
			namespace,
			v1alpha1.SchemeGroupVersion.WithResource("servers"),
			v1alpha1.SchemeGroupVersion.WithKind("Server"),
			func() *v1alpha1.Server { return &v1alpha1.Server{} },
			func() *v1alpha1.ServerList { return &v1alpha1.ServerList{} },
			func(dst, src *v1alpha1.ServerList) { dst.ListMeta = src.ListMeta },
			func(list *v1alpha1.ServerList) []*v1alpha1.Server { return gentype.ToPointerSlice(list.Items) },
			func(list *v1alpha1.ServerList, items []*v1alpha1.Server) {
				list.Items = gentype.FromPointerSlice(items)
			},
		),
		fake,
	}
}
