// Code generated by resource-generator. DO NOT EDIT.
/*
Copyright 2025 The ORC Authors.

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

package servergroup

import (
	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/interfaces"
)

// Fundamental types
type (
	orcObjectT     = orcv1alpha1.ServerGroup
	orcObjectListT = orcv1alpha1.ServerGroupList
	resourceSpecT  = orcv1alpha1.ServerGroupResourceSpec
	filterT        = orcv1alpha1.ServerGroupFilter
)

// Derived types
type (
	orcObjectPT = *orcObjectT
	adapterI    = interfaces.APIObjectAdapter[orcObjectPT, resourceSpecT, filterT]
	adapterT    = servergroupAdapter
)

type servergroupAdapter struct {
	*orcv1alpha1.ServerGroup
}

var _ adapterI = &adapterT{}

func (f adapterT) GetObject() orcObjectPT {
	return f.ServerGroup
}

func (f adapterT) GetManagementPolicy() orcv1alpha1.ManagementPolicy {
	return f.Spec.ManagementPolicy
}

func (f adapterT) GetManagedOptions() *orcv1alpha1.ManagedOptions {
	return f.Spec.ManagedOptions
}

func (f adapterT) GetStatusID() *string {
	return f.Status.ID
}

func (f adapterT) GetResourceSpec() *resourceSpecT {
	return f.Spec.Resource
}

func (f adapterT) GetImportID() *string {
	if f.Spec.Import == nil {
		return nil
	}
	return f.Spec.Import.ID
}

func (f adapterT) GetImportFilter() *filterT {
	if f.Spec.Import == nil {
		return nil
	}
	return f.Spec.Import.Filter
}

// getResourceName returns the name of the OpenStack resource we should use.
// This method is not implemented as part of APIObjectAdapter as it is intended
// to be used by resource actuators, which don't use the adapter.
func getResourceName(orcObject orcObjectPT) string {
	if orcObject.Spec.Resource.Name != nil {
		return string(*orcObject.Spec.Resource.Name)
	}
	return orcObject.Name
}
