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

package interfaces

import (
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
	applyconfigv1 "k8s.io/client-go/applyconfigurations/meta/v1"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
)

type ORCApplyConfig[objectApplyPT any, statusApplyPT ORCStatusApplyConfig[statusApplyPT]] interface {
	WithUID(types.UID) objectApplyPT
	WithStatus(statusApplyPT) objectApplyPT
}

type ORCStatusApplyConfig[statusApplyPT any] interface {
	WithConditions(...*applyconfigv1.ConditionApplyConfiguration) statusApplyPT
	WithID(id string) statusApplyPT
}

type ResourceStatusWriter[objectPT orcv1alpha1.ObjectWithConditions, osResourcePT any, objectApplyPT ORCApplyConfig[objectApplyPT, statusApplyPT], statusApplyPT ORCStatusApplyConfig[statusApplyPT]] interface {
	ResourceIsAvailable(orcObject objectPT, osResource osResourcePT) bool
	GetApplyConfig(name, namespace string) objectApplyPT
	ApplyResourceStatus(log logr.Logger, osResource osResourcePT, statusApply statusApplyPT)
}
