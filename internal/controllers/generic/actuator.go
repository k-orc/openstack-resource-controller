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

package generic

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	orcerrors "github.com/k-orc/openstack-resource-controller/internal/util/errors"
)

const (
	// The time to wait between checking if a delete was successful
	deletePollingPeriod = 1 * time.Second

	// The time to wait before reconciling again when we are waiting for some change in OpenStack
	externalUpdatePollingPeriod = 15 * time.Second
)

type BaseResourceActuator[osResourcePT any] interface {
	GetObject() client.Object

	GetManagementPolicy() orcv1alpha1.ManagementPolicy
	GetManagedOptions() *orcv1alpha1.ManagedOptions

	GetResourceID(osResource osResourcePT) string

	GetOSResourceByStatusID(ctx context.Context) (bool, osResourcePT, error)
	GetOSResourceBySpec(ctx context.Context) (osResourcePT, error)
}

type CreateResourceActuator[osResourcePT any] interface {
	BaseResourceActuator[osResourcePT]

	GetOSResourceByImportID(ctx context.Context) (bool, osResourcePT, error)
	GetOSResourceByImportFilter(ctx context.Context) (bool, osResourcePT, error)
	CreateResource(ctx context.Context) ([]WaitingOnEvent, osResourcePT, error)
}

type DeleteResourceActuator[osResourcePT any] interface {
	BaseResourceActuator[osResourcePT]

	DeleteResource(ctx context.Context, osResource osResourcePT) ([]WaitingOnEvent, error)
}

func GetOrCreateOSResource[osResourcePT *osResourceT, osResourceT any](ctx context.Context, log logr.Logger, k8sClient client.Client, actuator CreateResourceActuator[osResourcePT]) ([]WaitingOnEvent, osResourcePT, error) {
	// Get by status ID
	if hasStatusID, osResource, err := actuator.GetOSResourceByStatusID(ctx); hasStatusID {
		if orcerrors.IsNotFound(err) {
			// An OpenStack resource we previously referenced has been deleted unexpectedly. We can't recover from this.
			err = orcerrors.Terminal(orcv1alpha1.ConditionReasonUnrecoverableError, "resource has been deleted from OpenStack")
		}
		if osResource != nil {
			log.V(4).Info("Got existing OpenStack resource", "ID", actuator.GetResourceID(osResource))
		}
		return nil, osResource, err
	}

	// Import by ID
	if hasImportID, osResource, err := actuator.GetOSResourceByImportID(ctx); hasImportID {
		if orcerrors.IsNotFound(err) {
			// We assume that a resource imported by ID must already exist. It's a terminal error if it doesn't.
			err = orcerrors.Terminal(orcv1alpha1.ConditionReasonUnrecoverableError, "referenced resource does not exist in OpenStack")
		}
		if osResource != nil {
			log.V(4).Info("Imported existing OpenStack resource by ID", "ID", actuator.GetResourceID(osResource))
		}
		return nil, osResource, err
	}

	// Import by filter
	if hasImportFilter, osResource, err := actuator.GetOSResourceByImportFilter(ctx); hasImportFilter {
		var waitEvents []WaitingOnEvent
		if osResource == nil {
			waitEvents = []WaitingOnEvent{WaitingOnOpenStackExternal(externalUpdatePollingPeriod)}
		}
		return waitEvents, osResource, err
	}

	// Create
	if actuator.GetManagementPolicy() == orcv1alpha1.ManagementPolicyUnmanaged {
		// We never create an unmanaged resource
		// API validation should have ensured that one of the above functions returned
		return nil, nil, orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Not creating unmanaged resource")
	}

	osResource, err := actuator.GetOSResourceBySpec(ctx)
	if err != nil {
		return nil, nil, err
	}
	if osResource != nil {
		log.V(4).Info("Adopted previously created resource")
		return nil, osResource, nil
	}

	log.V(3).Info("Creating resource")
	return actuator.CreateResource(ctx)
}

func DeleteResource[osResourcePT *osResourceT, osResourceT any](ctx context.Context, log logr.Logger, actuator DeleteResourceActuator[osResourcePT], onComplete func() error) (osResourcePT, ctrl.Result, error) {
	obj := actuator.GetObject()

	// We always fetch the resource by ID so we can continue to report status even when waiting for a finalizer
	hasStatusID, osResource, err := actuator.GetOSResourceByStatusID(ctx)
	if err != nil {
		if !orcerrors.IsNotFound(err) {
			return osResource, ctrl.Result{}, err
		}
		// Gophercloud can return an empty non-nil object when returning errors,
		// which will confuse us below.
		osResource = nil
	}

	if len(obj.GetFinalizers()) > 1 {
		log.V(4).Info("Deferring resource cleanup due to remaining external finalizers")
		return osResource, ctrl.Result{}, nil
	}

	// We won't delete the resource for an unmanaged object, or if onDelete is detach
	managementPolicy := actuator.GetManagementPolicy()
	managedOptions := actuator.GetManagedOptions()
	if managementPolicy == orcv1alpha1.ManagementPolicyUnmanaged || managedOptions.GetOnDelete() == orcv1alpha1.OnDeleteDetach {
		logPolicy := []any{"managementPolicy", managementPolicy}
		if managementPolicy == orcv1alpha1.ManagementPolicyManaged {
			logPolicy = append(logPolicy, "onDelete", managedOptions.GetOnDelete())
		}
		log.V(4).Info("Not deleting OpenStack resource due to policy", logPolicy...)
		return osResource, ctrl.Result{}, onComplete()
	}

	// If status.ID was not set, we still need to check if there's an orphaned object.
	if osResource == nil && !hasStatusID {
		osResource, err = actuator.GetOSResourceBySpec(ctx)
		if err != nil {
			return osResource, ctrl.Result{}, err
		}
	}

	if osResource == nil {
		log.V(4).Info("Resource is no longer observed")
		return osResource, ctrl.Result{}, onComplete()
	}

	log.V(4).Info("Deleting OpenStack resource")
	waitEvents, err := actuator.DeleteResource(ctx, osResource)
	if err != nil {
		return osResource, ctrl.Result{}, err
	}

	var requeue time.Duration
	if len(waitEvents) > 0 {
		requeue = MaxRequeue(waitEvents)
	} else {
		requeue = deletePollingPeriod
	}
	return osResource, ctrl.Result{RequeueAfter: requeue}, nil
}

type WaitingOnEvent interface {
	Message() string
	Requeue() time.Duration
}

type waitingOnType int

const (
	WaitingOnCreation waitingOnType = iota
	WaitingOnReady
	WaitingOnDeletion
)

type waitingOnORC struct {
	kind      string
	name      string
	waitingOn waitingOnType
}

var _ WaitingOnEvent = waitingOnORC{}

func (e waitingOnORC) Message() string {
	var outcome string
	switch e.waitingOn {
	case WaitingOnCreation:
		outcome = "created"
	case WaitingOnReady:
		outcome = "ready"
	case WaitingOnDeletion:
		outcome = "deleted"
	}
	return fmt.Sprintf("Waiting for %s/%s to be %s", e.kind, e.name, outcome)
}

func newWaitingOnORC(kind, name string, event waitingOnType) WaitingOnEvent {
	return waitingOnORC{
		kind:      kind,
		name:      name,
		waitingOn: event,
	}
}

func WaitingOnORCExist(kind, name string) WaitingOnEvent {
	return newWaitingOnORC(kind, name, WaitingOnCreation)
}

func WaitingOnORCReady(kind, name string) WaitingOnEvent {
	return newWaitingOnORC(kind, name, WaitingOnReady)
}

func WaitingOnORCDeleted(kind, name string) WaitingOnEvent {
	return newWaitingOnORC(kind, name, WaitingOnDeletion)
}

func (e waitingOnORC) Requeue() time.Duration {
	return 0
}

type waitingOnOpenStack struct {
	waitingOn     waitingOnType
	pollingPeriod time.Duration
}

var _ WaitingOnEvent = waitingOnOpenStack{}

func newWaitingOnOpenStack(event waitingOnType, pollingPeriod time.Duration) WaitingOnEvent {
	return waitingOnOpenStack{
		waitingOn:     event,
		pollingPeriod: pollingPeriod,
	}
}

func WaitingOnOpenStackExternal(pollingPeriod time.Duration) WaitingOnEvent {
	return newWaitingOnOpenStack(WaitingOnCreation, pollingPeriod)
}

func WaitingOnOpenStackReady(kind, name string, pollingPeriod time.Duration) WaitingOnEvent {
	return newWaitingOnOpenStack(WaitingOnReady, pollingPeriod)
}

func (e waitingOnOpenStack) Message() string {
	var outcome string
	switch e.waitingOn {
	case WaitingOnCreation:
		outcome = "be created externally"
	case WaitingOnReady:
		outcome = "be ready"
	}
	return fmt.Sprintf("Waiting for OpenStack resource to %s", outcome)
}

func (e waitingOnOpenStack) Requeue() time.Duration {
	return e.pollingPeriod
}

func MaxRequeue(evts []WaitingOnEvent) time.Duration {
	var ret time.Duration
	for _, evt := range evts {
		if evt.Requeue() > ret {
			ret = evt.Requeue()
		}
	}
	return ret
}
