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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/internal/controllers/common"
	"github.com/k-orc/openstack-resource-controller/internal/scope"
	orcerrors "github.com/k-orc/openstack-resource-controller/internal/util/errors"
)

const (
	// The time to wait between checking if a delete was successful
	deletePollingPeriod = 1 * time.Second

	// The time to wait before reconciling again when we are waiting for some change in OpenStack
	externalUpdatePollingPeriod = 15 * time.Second
)

const ORCK8SPrefix = "openstack.k-orc.cloud"

type SSATransactionID string

const (
	// Field owner of the object finalizer.
	SSATransactionFinalizer SSATransactionID = "finalizer"
	SSATransactionStatus    SSATransactionID = "status"
)

type ActuatorFactory[orcObjectPT any, osResourcePT any] interface {
	NewCreateActuator(ctx context.Context, orcObject orcObjectPT, controller ResourceController) ([]WaitingOnEvent, CreateResourceActuator[osResourcePT], error)
	NewDeleteActuator(ctx context.Context, orcObject orcObjectPT, controller ResourceController) ([]WaitingOnEvent, DeleteResourceActuator[osResourcePT], error)
}

type ResourceController interface {
	GetName() string

	GetK8sClient() client.Client
	GetScopeFactory() scope.Factory
}

type BaseResourceActuator[osResourcePT any] interface {
	GetObject() client.Object
	GetController() ResourceController

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

type ResourceUpdater[orcObjectPT, osResourcePT any] func(ctx context.Context, orcObject orcObjectPT, osResource osResourcePT) ([]WaitingOnEvent, orcObjectPT, osResourcePT, error)

type UpdateResourceActuator[orcObjectPT, osResourcePT any] interface {
	GetResourceUpdaters(ctx context.Context, orcObject orcObjectPT, osResource osResourcePT, controller ResourceController) ([]ResourceUpdater[orcObjectPT, osResourcePT], error)
}

type DeleteResourceActuator[osResourcePT any] interface {
	BaseResourceActuator[osResourcePT]

	DeleteResource(ctx context.Context, osResource osResourcePT) ([]WaitingOnEvent, error)
}

func getSSAFieldOwnerString(controller ResourceController) string {
	return ORCK8SPrefix + "/" + controller.GetName() + "controller"
}

// GetSSAFieldOwner returns the field owner for a specific named SSA transaction.
func GetSSAFieldOwner(controller ResourceController) client.FieldOwner {
	return client.FieldOwner(getSSAFieldOwnerString(controller))
}

func GetSSAFieldOwnerWithTxn(controller ResourceController, txn SSATransactionID) client.FieldOwner {
	return client.FieldOwner(getSSAFieldOwnerString(controller) + "/" + string(txn))
}

// GetFinalizerName returns the finalizer to be used for the given actuator
func GetFinalizerName(controller ResourceController) string {
	return ORCK8SPrefix + "/" + controller.GetName()
}

func GetOrCreateOSResource[osResourcePT *osResourceT, osResourceT any](ctx context.Context, log logr.Logger, k8sClient client.Client, actuator CreateResourceActuator[osResourcePT]) ([]WaitingOnEvent, osResourcePT, error) {
	orcObject := actuator.GetObject()
	controller := actuator.GetController()

	finalizer := GetFinalizerName(controller)
	if !controllerutil.ContainsFinalizer(orcObject, finalizer) {
		patch := common.SetFinalizerPatch(orcObject, finalizer)
		if err := k8sClient.Patch(ctx, orcObject, patch, client.ForceOwnership, GetSSAFieldOwnerWithTxn(controller, SSATransactionFinalizer)); err != nil {
			return nil, nil, fmt.Errorf("setting finalizer: %w", err)
		}
	}

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
			waitEvents = []WaitingOnEvent{WaitingOnOpenStackCreate(externalUpdatePollingPeriod)}
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

func DeleteResource[osResourcePT *osResourceT, osResourceT any](ctx context.Context, log logr.Logger, k8sClient client.Client, actuator DeleteResourceActuator[osResourcePT]) (bool, []WaitingOnEvent, osResourcePT, error) {
	obj := actuator.GetObject()
	controller := actuator.GetController()

	// We always fetch the resource by ID so we can continue to report status even when waiting for a finalizer
	hasStatusID, osResource, err := actuator.GetOSResourceByStatusID(ctx)
	if err != nil {
		if !orcerrors.IsNotFound(err) {
			return false, nil, osResource, err
		}
		// Gophercloud can return an empty non-nil object when returning errors,
		// which will confuse us below.
		osResource = nil
	}

	finalizer := GetFinalizerName(controller)

	var waitEvents []WaitingOnEvent
	var foundFinalizer bool
	for _, f := range obj.GetFinalizers() {
		if f == finalizer {
			foundFinalizer = true
		} else {
			waitEvents = append(waitEvents, WaitingOnFinalizer(f))
		}
	}

	// Cleanup not required if our finalizer is not present
	if !foundFinalizer {
		return true, waitEvents, osResource, nil
	}

	if len(waitEvents) > 0 {
		log.V(4).Info("Deferring resource cleanup due to remaining external finalizers")
		return false, waitEvents, osResource, nil
	}

	removeFinalizer := func() error {
		if err := k8sClient.Patch(ctx, obj, common.RemoveFinalizerPatch(obj), GetSSAFieldOwnerWithTxn(controller, SSATransactionFinalizer)); err != nil {
			return fmt.Errorf("removing finalizer: %w", err)
		}
		return nil
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
		return true, waitEvents, osResource, removeFinalizer()
	}

	// If status.ID was not set, we still need to check if there's an orphaned object.
	if osResource == nil && !hasStatusID {
		osResource, err = actuator.GetOSResourceBySpec(ctx)
		if err != nil {
			return false, waitEvents, osResource, err
		}
	}

	if osResource == nil {
		log.V(4).Info("Resource is no longer observed")

		return true, waitEvents, osResource, removeFinalizer()
	}

	log.V(4).Info("Deleting OpenStack resource")
	waitEvents, err = actuator.DeleteResource(ctx, osResource)

	// If there are no other wait events, we still need to poll for the deletion
	// of the OpenStack resource
	if len(waitEvents) == 0 {
		waitEvents = []WaitingOnEvent{WaitingOnOpenStackDeleted(deletePollingPeriod)}
	}
	return false, waitEvents, osResource, err
}
