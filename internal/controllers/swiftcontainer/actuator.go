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

package swiftcontainer

import (
	"context"
	"iter"
	"strings"

	"github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/containers"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	generic "github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/progress"
	"github.com/k-orc/openstack-resource-controller/v2/internal/logging"
	osclients "github.com/k-orc/openstack-resource-controller/v2/internal/osclients"
	orcerrors "github.com/k-orc/openstack-resource-controller/v2/internal/util/errors"
)

// osContainerT wraps containers.GetHeader with the container name and
// metadata, since GetHeader does not include the container name and the
// X-Container-Meta-* headers are returned separately via ExtractMetadata.
type osContainerT struct {
	Name     string
	Metadata map[string]string
	containers.GetHeader
}

// OpenStack resource types
type (
	osResourceT = osContainerT

	createResourceActuator    = generic.CreateResourceActuator[orcObjectPT, orcObjectT, filterT, osResourceT]
	deleteResourceActuator    = generic.DeleteResourceActuator[orcObjectPT, orcObjectT, osResourceT]
	reconcileResourceActuator = generic.ReconcileResourceActuator[orcObjectPT, osResourceT]
	resourceReconciler        = generic.ResourceReconciler[orcObjectPT, osResourceT]
	helperFactory             = generic.ResourceHelperFactory[orcObjectPT, orcObjectT, resourceSpecT, filterT, osResourceT]
)

type swiftcontainerActuator struct {
	osClient osclients.SwiftContainerClient
}

var _ createResourceActuator = swiftcontainerActuator{}
var _ deleteResourceActuator = swiftcontainerActuator{}
var _ reconcileResourceActuator = swiftcontainerActuator{}

func (swiftcontainerActuator) GetResourceID(osResource *osContainerT) string {
	// Swift containers are identified by name
	return osResource.Name
}

func (actuator swiftcontainerActuator) GetOSResourceByID(ctx context.Context, id string) (*osContainerT, progress.ReconcileStatus) {
	header, err := actuator.osClient.GetContainer(ctx, id, nil)
	if err != nil {
		return nil, progress.WrapError(err)
	}
	metadata, err := actuator.osClient.GetContainerMetadata(ctx, id)
	if err != nil {
		return nil, progress.WrapError(err)
	}
	return &osContainerT{Name: id, Metadata: metadata, GetHeader: *header}, nil
}

func (actuator swiftcontainerActuator) ListOSResourcesForAdoption(ctx context.Context, orcObject orcObjectPT) (iter.Seq2[*osContainerT, error], bool) {
	resourceSpec := orcObject.Spec.Resource
	if resourceSpec == nil {
		return nil, false
	}

	name := getResourceName(orcObject)
	return func(yield func(*osContainerT, error) bool) {
		header, err := actuator.osClient.GetContainer(ctx, name, nil)
		if err != nil {
			if !orcerrors.IsNotFound(err) {
				yield(nil, err)
			}
			return
		}
		metadata, err := actuator.osClient.GetContainerMetadata(ctx, name)
		if err != nil {
			yield(nil, err)
			return
		}
		yield(&osContainerT{Name: name, Metadata: metadata, GetHeader: *header}, nil)
	}, true
}

func (actuator swiftcontainerActuator) ListOSResourcesForImport(ctx context.Context, _ orcObjectPT, filter filterT) (iter.Seq2[*osResourceT, error], progress.ReconcileStatus) {
	return func(yield func(*osContainerT, error) bool) {
		// List all containers and filter by prefix.
		listOpts := containers.ListOpts{}
		for container, err := range actuator.osClient.ListContainers(ctx, listOpts) {
			if err != nil {
				yield(nil, err)
				return
			}

			if filter.Prefix != nil && !strings.HasPrefix(container.Name, *filter.Prefix) {
				continue
			}

			header, err := actuator.osClient.GetContainer(ctx, container.Name, nil)
			if err != nil {
				yield(nil, err)
				return
			}
			metadata, err := actuator.osClient.GetContainerMetadata(ctx, container.Name)
			if err != nil {
				yield(nil, err)
				return
			}
			if !yield(&osContainerT{Name: container.Name, Metadata: metadata, GetHeader: *header}, nil) {
				return
			}
		}
	}, nil
}

func (actuator swiftcontainerActuator) CreateResource(ctx context.Context, obj orcObjectPT) (*osContainerT, progress.ReconcileStatus) {
	resource := obj.Spec.Resource

	if resource == nil {
		// Should have been caught by API validation
		return nil, progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Creation requested, but spec.resource is not set"))
	}

	name := getResourceName(obj)

	createOpts := containers.CreateOpts{
		ContainerRead:  resource.ContainerRead,
		ContainerWrite: resource.ContainerWrite,
		StoragePolicy:  resource.StoragePolicy,
	}

	if len(resource.Metadata) > 0 {
		metadata := make(map[string]string, len(resource.Metadata))
		for _, m := range resource.Metadata {
			metadata[m.Key] = m.Value
		}
		createOpts.Metadata = metadata
	}

	_, err := actuator.osClient.CreateContainer(ctx, name, createOpts)
	if err != nil {
		if !orcerrors.IsRetryable(err) {
			err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration creating resource: "+err.Error(), err)
		}
		return nil, progress.WrapError(err)
	}

	// Fetch the created container to return its header and metadata
	header, err := actuator.osClient.GetContainer(ctx, name, nil)
	if err != nil {
		return nil, progress.WrapError(err)
	}
	fetchedMetadata, err := actuator.osClient.GetContainerMetadata(ctx, name)
	if err != nil {
		return nil, progress.WrapError(err)
	}

	return &osContainerT{Name: name, Metadata: fetchedMetadata, GetHeader: *header}, nil
}

func (actuator swiftcontainerActuator) DeleteResource(ctx context.Context, _ orcObjectPT, osResource *osContainerT) progress.ReconcileStatus {
	err := actuator.osClient.DeleteContainer(ctx, osResource.Name)
	if orcerrors.IsNotFound(err) {
		return nil
	}
	return progress.WrapError(err)
}

func (actuator swiftcontainerActuator) GetResourceReconcilers(_ context.Context, _ orcObjectPT, _ *osResourceT, _ generic.ResourceController) ([]resourceReconciler, progress.ReconcileStatus) {
	return []resourceReconciler{
		actuator.reconcileACLs,
		actuator.reconcileMetadata,
	}, nil
}

// reconcileACLs compares the desired ACLs from the spec with the current
// container ACLs and calls UpdateContainer if they differ.
func (actuator swiftcontainerActuator) reconcileACLs(ctx context.Context, orcObject orcObjectPT, osResource *osResourceT) progress.ReconcileStatus {
	log := ctrl.LoggerFrom(ctx)
	resource := orcObject.Spec.Resource
	if resource == nil {
		return nil
	}

	// GetHeader.Read and GetHeader.Write are []string parsed from the ACL
	// headers. We join them back to a comma-separated string for comparison
	// with the spec which stores ACLs as a single string.
	currentRead := strings.Join(osResource.Read, ",")
	currentWrite := strings.Join(osResource.Write, ",")

	desiredRead := resource.ContainerRead
	desiredWrite := resource.ContainerWrite

	if currentRead == desiredRead && currentWrite == desiredWrite {
		log.V(logging.Debug).Info("Container ACLs are up to date")
		return nil
	}

	log.V(logging.Info).Info("Updating container ACLs")
	updateOpts := containers.UpdateOpts{
		ContainerRead:  &desiredRead,
		ContainerWrite: &desiredWrite,
	}
	_, err := actuator.osClient.UpdateContainer(ctx, osResource.Name, updateOpts)
	if err != nil {
		if !orcerrors.IsRetryable(err) {
			err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration updating container ACLs: "+err.Error(), err)
		}
		return progress.WrapError(err)
	}

	return progress.NeedsRefresh()
}

// reconcileMetadata compares the desired metadata from the spec with the
// current container metadata and calls UpdateContainer if they differ.
func (actuator swiftcontainerActuator) reconcileMetadata(ctx context.Context, orcObject orcObjectPT, osResource *osResourceT) progress.ReconcileStatus {
	log := ctrl.LoggerFrom(ctx)
	resource := orcObject.Spec.Resource
	if resource == nil {
		return nil
	}

	// Fetch current metadata via GetContainerMetadata. Keys are returned in
	// canonical HTTP header form (e.g. "Env", not "env") because Go's net/http
	// canonicalises header keys on retrieval.
	currentMetadata, err := actuator.osClient.GetContainerMetadata(ctx, osResource.Name)
	if err != nil {
		return progress.WrapError(err)
	}

	// Build a lowercase-keyed view of current metadata for case-insensitive
	// comparison. We also track the original canonical key so we can form
	// correct X-Remove-Container-Meta-<Key> headers when removing entries.
	currentLower := make(map[string]string, len(currentMetadata))     // lowercase key -> value
	currentCanonical := make(map[string]string, len(currentMetadata)) // lowercase key -> canonical key
	for k, v := range currentMetadata {
		lk := strings.ToLower(k)
		currentLower[lk] = v
		currentCanonical[lk] = k
	}

	// Build the desired metadata map from the spec with lowercase keys so that
	// comparisons are case-insensitive (metadata key casing is not significant
	// in Swift).
	desiredMetadata := make(map[string]string, len(resource.Metadata))
	for _, m := range resource.Metadata {
		desiredMetadata[strings.ToLower(m.Key)] = m.Value
	}

	// Find keys to add/update and keys to remove.
	var toSet map[string]string
	var toRemove []string

	for key, desiredVal := range desiredMetadata {
		if currentVal, exists := currentLower[key]; !exists || currentVal != desiredVal {
			if toSet == nil {
				toSet = make(map[string]string)
			}
			toSet[key] = desiredVal
		}
	}
	for lk := range currentLower {
		if _, desired := desiredMetadata[lk]; !desired {
			// Use the canonical key so that the X-Remove-Container-Meta-<Key>
			// header matches what Swift has stored.
			toRemove = append(toRemove, currentCanonical[lk])
		}
	}

	if len(toSet) == 0 && len(toRemove) == 0 {
		log.V(logging.Debug).Info("Container metadata is up to date")
		return nil
	}

	log.V(logging.Info).Info("Updating container metadata")
	updateOpts := containers.UpdateOpts{
		Metadata:       toSet,
		RemoveMetadata: toRemove,
	}
	_, err = actuator.osClient.UpdateContainer(ctx, osResource.Name, updateOpts)
	if err != nil {
		if !orcerrors.IsRetryable(err) {
			err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "invalid configuration updating container metadata: "+err.Error(), err)
		}
		return progress.WrapError(err)
	}

	return progress.NeedsRefresh()
}

type swiftcontainerHelperFactory struct{}

var _ helperFactory = swiftcontainerHelperFactory{}

func newActuator(ctx context.Context, orcObject *orcv1alpha1.SwiftContainer, controller generic.ResourceController) (swiftcontainerActuator, progress.ReconcileStatus) {
	log := ctrl.LoggerFrom(ctx)

	// Ensure credential secrets exist and have our finalizer
	_, reconcileStatus := credentialsDependency.GetDependencies(ctx, controller.GetK8sClient(), orcObject, func(*corev1.Secret) bool { return true })
	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return swiftcontainerActuator{}, reconcileStatus
	}

	clientScope, err := controller.GetScopeFactory().NewClientScopeFromObject(ctx, controller.GetK8sClient(), log, orcObject)
	if err != nil {
		return swiftcontainerActuator{}, progress.WrapError(err)
	}
	osClient, err := clientScope.NewSwiftContainerClient()
	if err != nil {
		return swiftcontainerActuator{}, progress.WrapError(err)
	}

	return swiftcontainerActuator{
		osClient: osClient,
	}, nil
}

func (swiftcontainerHelperFactory) NewAPIObjectAdapter(obj orcObjectPT) adapterI {
	return swiftcontainerAdapter{obj}
}

func (swiftcontainerHelperFactory) NewCreateActuator(ctx context.Context, orcObject orcObjectPT, controller generic.ResourceController) (createResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, orcObject, controller)
}

func (swiftcontainerHelperFactory) NewDeleteActuator(ctx context.Context, orcObject orcObjectPT, controller generic.ResourceController) (deleteResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, orcObject, controller)
}
