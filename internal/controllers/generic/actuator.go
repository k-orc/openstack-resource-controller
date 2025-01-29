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
	"iter"
	"time"

	"github.com/go-logr/logr"
	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/api/v1alpha1"
	orcerrors "github.com/k-orc/openstack-resource-controller/internal/util/errors"
	"github.com/k-orc/openstack-resource-controller/internal/util/finalizers"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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

type ResourceHelperFactory[
	orcObjectPT interface {
		*orcObjectT
		client.Object
		orcv1alpha1.ObjectWithConditions
	}, orcObjectT any,
	resourceSpecT any, filterT any,
	osResourceT any,
] interface {
	NewAPIObjectAdapter(orcObject orcObjectPT) APIObjectAdapter[orcObjectPT, resourceSpecT, filterT]

	NewCreateActuator(ctx context.Context, orcObject orcObjectPT, controller ResourceController) ([]ProgressStatus, CreateResourceActuator[orcObjectPT, orcObjectT, filterT, osResourceT], error)
	NewDeleteActuator(ctx context.Context, orcObject orcObjectPT, controller ResourceController) ([]ProgressStatus, DeleteResourceActuator[orcObjectPT, orcObjectT, osResourceT], error)
}

type BaseResourceActuator[
	orcObjectPT interface {
		*orcObjectT
		client.Object
		orcv1alpha1.ObjectWithConditions
	}, orcObjectT any,
	osResourceT any,
] interface {
	GetResourceID(osResource *osResourceT) string

	GetOSResourceByID(ctx context.Context, id string) (*osResourceT, error)
	ListOSResourcesForAdoption(ctx context.Context, orcObject orcObjectPT) (iter.Seq2[*osResourceT, error], bool)
}

type CreateResourceActuator[
	orcObjectPT interface {
		*orcObjectT
		client.Object
		orcv1alpha1.ObjectWithConditions
	}, orcObjectT any,
	filterT any,
	osResourceT any,
] interface {
	BaseResourceActuator[orcObjectPT, orcObjectT, osResourceT]

	CreateResource(ctx context.Context, orcObject orcObjectPT) ([]ProgressStatus, *osResourceT, error)
	ListOSResourcesForImport(ctx context.Context, filter filterT) iter.Seq2[*osResourceT, error]
}

type DeleteResourceActuator[
	orcObjectPT interface {
		*orcObjectT
		client.Object
		orcv1alpha1.ObjectWithConditions
	}, orcObjectT any,
	osResourceT any,
] interface {
	BaseResourceActuator[orcObjectPT, orcObjectT, osResourceT]

	DeleteResource(ctx context.Context, orcObject orcObjectPT, osResource *osResourceT) ([]ProgressStatus, error)
}

type ResourceReconciler[orcObjectPT, osResourceT any] func(ctx context.Context, orcObject orcObjectPT, osResource *osResourceT) ([]ProgressStatus, error)

type ReconcileResourceActuator[orcObjectPT, osResourceT any] interface {
	GetResourceReconcilers(ctx context.Context, orcObject orcObjectPT, osResource *osResourceT, controller ResourceController) ([]ResourceReconciler[orcObjectPT, osResourceT], error)
}

func GetOrCreateOSResource[
	orcObjectPT interface {
		*orcObjectT
		client.Object
		orcv1alpha1.ObjectWithConditions
	}, orcObjectT any,
	resourceSpecT any, filterT any,
	osResourceT any,
](
	ctx context.Context, log logr.Logger, controller ResourceController,
	objAdapter APIObjectAdapter[orcObjectPT, resourceSpecT, filterT],
	actuator CreateResourceActuator[orcObjectPT, orcObjectT, filterT, osResourceT],
) ([]ProgressStatus, *osResourceT, error) {
	k8sClient := controller.GetK8sClient()

	finalizer := GetFinalizerName(controller)
	if !controllerutil.ContainsFinalizer(objAdapter.GetObject(), finalizer) {
		patch := finalizers.SetFinalizerPatch(objAdapter.GetObject(), finalizer)
		if err := k8sClient.Patch(ctx, objAdapter.GetObject(), patch, client.ForceOwnership, GetSSAFieldOwnerWithTxn(controller, SSATransactionFinalizer)); err != nil {
			return nil, nil, fmt.Errorf("setting finalizer: %w", err)
		}
	}

	if resourceID := objAdapter.GetStatusID(); resourceID != nil {
		osResource, err := actuator.GetOSResourceByID(ctx, *resourceID)
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
	if resourceID := objAdapter.GetImportID(); resourceID != nil {
		osResource, err := actuator.GetOSResourceByID(ctx, *resourceID)
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
	if filter := objAdapter.GetImportFilter(); filter != nil {
		osResource, err := getResourceForImport(ctx, actuator, *filter)
		if err != nil {
			return nil, nil, err
		}
		if osResource == nil {
			// Poll until we find a resource
			progressStatus := []ProgressStatus{WaitingOnOpenStackCreate(externalUpdatePollingPeriod)}
			return progressStatus, nil, nil
		}
		return nil, osResource, nil
	}

	// Create
	if objAdapter.GetManagementPolicy() == orcv1alpha1.ManagementPolicyUnmanaged {
		// We never create an unmanaged resource
		// API validation should have ensured that one of the above functions returned
		return nil, nil, orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Not creating unmanaged resource")
	}

	osResource, err := getResourceForAdoption(ctx, actuator, objAdapter)
	if err != nil {
		return nil, nil, err
	}
	if osResource != nil {
		log.V(4).Info("Adopted previously created resource")
		return nil, osResource, nil
	}

	log.V(3).Info("Creating resource")
	return actuator.CreateResource(ctx, objAdapter.GetObject())
}

func atMostOne[osResourceT any](resourceIter iter.Seq2[*osResourceT, error], multipleErr error) (*osResourceT, error) {
	next, stop := iter.Pull2(resourceIter)
	defer stop()

	// Try to fetch the first result
	osResource, err, ok := next()
	if err != nil {
		return nil, err
	} else if !ok {
		// No first result
		return nil, nil
	}

	// Check that there are no other results
	_, err, ok = next()
	if err != nil {
		return nil, err
	} else if ok {
		return nil, multipleErr
	}

	return osResource, nil
}

func getResourceForAdoption[
	orcObjectPT interface {
		*orcObjectT
		client.Object
		orcv1alpha1.ObjectWithConditions
	}, orcObjectT any,
	resourceSpecT any, filterT any,
	osResourceT any,
](
	ctx context.Context,
	actuator BaseResourceActuator[orcObjectPT, orcObjectT, osResourceT],
	objAdapter APIObjectAdapter[orcObjectPT, resourceSpecT, filterT],
) (*osResourceT, error) {
	resourceIter, canAdopt := actuator.ListOSResourcesForAdoption(ctx, objAdapter.GetObject())
	if !canAdopt {
		return nil, nil
	}

	return atMostOne(resourceIter, orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "found more than one matching OpenStack resource during adoption"))
}

func getResourceForImport[
	orcObjectPT interface {
		*orcObjectT
		client.Object
		orcv1alpha1.ObjectWithConditions
	}, orcObjectT any,
	filterT any,
	osResourceT any,
](
	ctx context.Context,
	actuator CreateResourceActuator[orcObjectPT, orcObjectT, filterT, osResourceT],
	filter filterT,
) (*osResourceT, error) {
	resourceIter := actuator.ListOSResourcesForImport(ctx, filter)
	return atMostOne(resourceIter, orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "found more than one matching OpenStack resource during import"))
}

func DeleteResource[
	orcObjectPT interface {
		*orcObjectT
		client.Object
		orcv1alpha1.ObjectWithConditions
	}, orcObjectT any,
	resourceSpecT any, filterT any,
	osResourceT any,
](
	ctx context.Context, log logr.Logger, controller ResourceController,
	objAdapter APIObjectAdapter[orcObjectPT, resourceSpecT, filterT],
	actuator DeleteResourceActuator[orcObjectPT, orcObjectT, osResourceT],
) (bool, []ProgressStatus, *osResourceT, error) {
	var osResource *osResourceT

	// We always fetch the resource by ID so we can continue to report status even when waiting for a finalizer
	statusID := objAdapter.GetStatusID()
	if statusID != nil {
		var err error
		osResource, err = actuator.GetOSResourceByID(ctx, *statusID)
		if err != nil {
			if !orcerrors.IsNotFound(err) {
				return false, nil, osResource, err
			}
			// Gophercloud can return an empty non-nil object when returning errors,
			// which will confuse us below.
			osResource = nil
		}
	}

	finalizer := GetFinalizerName(controller)

	var progressStatus []ProgressStatus
	var foundFinalizer bool
	for _, f := range objAdapter.GetFinalizers() {
		if f == finalizer {
			foundFinalizer = true
		} else {
			progressStatus = append(progressStatus, WaitingOnFinalizer(f))
		}
	}

	// Cleanup not required if our finalizer is not present
	if !foundFinalizer {
		return true, progressStatus, osResource, nil
	}

	if len(progressStatus) > 0 {
		log.V(4).Info("Deferring resource cleanup due to remaining external finalizers")
		return false, progressStatus, osResource, nil
	}

	removeFinalizer := func() error {
		if err := controller.GetK8sClient().Patch(ctx, objAdapter.GetObject(), finalizers.RemoveFinalizerPatch(objAdapter.GetObject()), GetSSAFieldOwnerWithTxn(controller, SSATransactionFinalizer)); err != nil {
			return fmt.Errorf("removing finalizer: %w", err)
		}
		return nil
	}

	// We won't delete the resource for an unmanaged object, or if onDelete is detach
	managementPolicy := objAdapter.GetManagementPolicy()
	managedOptions := objAdapter.GetManagedOptions()
	if managementPolicy == orcv1alpha1.ManagementPolicyUnmanaged || managedOptions.GetOnDelete() == orcv1alpha1.OnDeleteDetach {
		logPolicy := []any{"managementPolicy", managementPolicy}
		if managementPolicy == orcv1alpha1.ManagementPolicyManaged {
			logPolicy = append(logPolicy, "onDelete", managedOptions.GetOnDelete())
		}
		log.V(4).Info("Not deleting OpenStack resource due to policy", logPolicy...)
		return true, progressStatus, osResource, removeFinalizer()
	}

	// If status.ID was not set, we still need to check if there's an orphaned object.
	if osResource == nil && statusID == nil {
		var err error
		osResource, err = getResourceForAdoption(ctx, actuator, objAdapter)
		if err != nil {
			return false, progressStatus, osResource, err
		}
	}

	if osResource == nil {
		log.V(4).Info("Resource is no longer observed")

		return true, progressStatus, osResource, removeFinalizer()
	}

	log.V(4).Info("Deleting OpenStack resource")
	progressStatus, err := actuator.DeleteResource(ctx, objAdapter.GetObject(), osResource)

	// If there are no other wait events, we still need to poll for the deletion
	// of the OpenStack resource
	if len(progressStatus) == 0 {
		progressStatus = []ProgressStatus{WaitingOnOpenStackDeleted(deletePollingPeriod)}
	}
	return false, progressStatus, osResource, err
}
