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

package keypair

import (
	"context"
	"iter"

	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/keypairs"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/progress"
	osclients "github.com/k-orc/openstack-resource-controller/v2/internal/osclients"
	orcerrors "github.com/k-orc/openstack-resource-controller/v2/internal/util/errors"
)

// OpenStack resource types
type (
	osResourceT = keypairs.KeyPair

	createResourceActuator = interfaces.CreateResourceActuator[orcObjectPT, orcObjectT, filterT, osResourceT]
	deleteResourceActuator = interfaces.DeleteResourceActuator[orcObjectPT, orcObjectT, osResourceT]
	resourceReconciler     = interfaces.ResourceReconciler[orcObjectPT, osResourceT]
	helperFactory          = interfaces.ResourceHelperFactory[orcObjectPT, orcObjectT, resourceSpecT, filterT, osResourceT]
)

type KeyPairClient interface {
	GetKeyPair(context.Context, string, string) (*osResourceT, error)
	ListKeyPairs(context.Context, keypairs.ListOptsBuilder) iter.Seq2[*osResourceT, error]
	CreateKeyPair(context.Context, keypairs.CreateOptsBuilder) (*osResourceT, error)
	DeleteKeyPair(context.Context, string, string) error
}

type keypairActuator struct {
	osClient  KeyPairClient
	k8sClient client.Client
}

var _ createResourceActuator = keypairActuator{}
var _ deleteResourceActuator = keypairActuator{}

func (keypairActuator) GetResourceID(osResource *keypairs.KeyPair) string {
	return osResource.Name
}

func (actuator keypairActuator) GetOSResourceByID(ctx context.Context, name string) (*osResourceT, progress.ReconcileStatus) {
	// For Keypairs, ID is the name
	// Note: We pass empty userID here, which means we get the keypair for the authenticated user.
	// This works for most cases. For admin users managing keypairs for other users,
	// the userID would need to be extracted from the ORC object, but GetOSResourceByID
	// interface only provides the ID. This is a known limitation - admin users should
	// use import by filter instead of import by ID when managing keypairs for other users.
	userID := ""
	resource, err := actuator.osClient.GetKeyPair(ctx, name, userID)
	if err != nil {
		return nil, progress.WrapError(err)
	}
	return resource, nil
}

func (actuator keypairActuator) ListOSResourcesForAdoption(ctx context.Context, orcObject orcObjectPT) (iter.Seq2[*osResourceT, error], bool) {
	resourceSpec := orcObject.Spec.Resource
	if resourceSpec == nil {
		return nil, false
	}

	// Filter by the expected resource name to avoid adopting wrong keypairs.
	// The OpenStack Keypairs API only supports filtering by userID server-side,
	// so we must use client-side filtering for the name.
	var filters []osclients.ResourceFilter[osResourceT]
	filters = append(filters, func(kp *keypairs.KeyPair) bool {
		return kp.Name == getResourceName(orcObject)
	})

	listOpts := keypairs.ListOpts{
		UserID: ptr.Deref(resourceSpec.UserID, ""),
	}

	return actuator.listOSResources(ctx, filters, listOpts), true
}

func (actuator keypairActuator) ListOSResourcesForImport(ctx context.Context, obj orcObjectPT, filter filterT) (iter.Seq2[*osResourceT, error], progress.ReconcileStatus) {
	// The OpenStack Keypairs API only supports filtering by userID server-side.
	// Client-side filtering is required for the name field.
	var filters []osclients.ResourceFilter[osResourceT]

	if filter.Name != nil {
		filters = append(filters, func(kp *keypairs.KeyPair) bool {
			return kp.Name == string(*filter.Name)
		})
	}

	listOpts := keypairs.ListOpts{
		UserID: ptr.Deref(filter.UserID, ""),
	}

	return actuator.listOSResources(ctx, filters, listOpts), nil
}

func (actuator keypairActuator) listOSResources(ctx context.Context, filters []osclients.ResourceFilter[osResourceT], listOpts keypairs.ListOptsBuilder) iter.Seq2[*osResourceT, error] {
	keypairs := actuator.osClient.ListKeyPairs(ctx, listOpts)
	return osclients.Filter(keypairs, filters...)
}

func (actuator keypairActuator) CreateResource(ctx context.Context, obj orcObjectPT) (*osResourceT, progress.ReconcileStatus) {
	resource := obj.Spec.Resource

	if resource == nil {
		// Should have been caught by API validation
		return nil, progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "Creation requested, but spec.resource is not set"))
	}

	// Validate configuration
	if resource.PublicKey == nil && resource.PrivateKeySecretRef == nil {
		return nil, progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration,
				"Either publicKey (for import) or privateKeySecretRef (for generation) must be specified"))
	}

	createOpts := keypairs.CreateOpts{
		Name:   getResourceName(obj),
		Type:   ptr.Deref(resource.Type, "ssh"),
		UserID: ptr.Deref(resource.UserID, ""),
	}

	// If publicKey is provided, import it
	if resource.PublicKey != nil {
		createOpts.PublicKey = *resource.PublicKey
	}

	osResource, err := actuator.osClient.CreateKeyPair(ctx, createOpts)
	if err != nil {
		if !orcerrors.IsRetryable(err) {
			err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration,
				"invalid configuration creating Keypair: "+err.Error(), err)
		}
		return nil, progress.WrapError(err)
	}

	// If we generated a new Keypair (no public key provided), we MUST store the private key
	// The private key is only available during creation and cannot be retrieved later
	if resource.PublicKey == nil && osResource.PrivateKey != "" {
		if err := actuator.storePrivateKey(ctx, obj, osResource.PrivateKey); err != nil {
			log := ctrl.LoggerFrom(ctx)
			log.Error(err, "Failed to store generated private key, deleting orphaned keypair to prevent unusable resource")

			// Clean up the keypair since we can't store its private key
			// Without the private key, this keypair is useless
			userID := ptr.Deref(resource.UserID, "")
			deleteErr := actuator.osClient.DeleteKeyPair(ctx, osResource.Name, userID)
			if deleteErr != nil {
				log.Error(deleteErr, "Failed to delete orphaned keypair after secret creation failure - manual cleanup may be required", "keypairName", osResource.Name)
			}

			// Return a terminal error - the user needs to fix the secret permissions/configuration
			return nil, progress.WrapError(
				orcerrors.Terminal(orcv1alpha1.ConditionReasonUnrecoverableError,
					"failed to store generated private key in secret "+resource.PrivateKeySecretRef.Name+": "+err.Error()+". The keypair was deleted to prevent orphaned resources. Please ensure the controller has permission to create/update secrets in this namespace and retry."))
		}
	}

	return osResource, nil
}

func (actuator keypairActuator) storePrivateKey(ctx context.Context, obj orcObjectPT, privateKey string) error {
	log := ctrl.LoggerFrom(ctx)
	resource := obj.Spec.Resource

	if resource.PrivateKeySecretRef == nil {
		return orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration,
			"privateKeySecretRef not specified but private key needs to be stored")
	}

	secretName := resource.PrivateKeySecretRef.Name
	secretKey := ptr.Deref(resource.PrivateKeySecretRef.Key, "private_key")

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: obj.Namespace,
		},
		StringData: map[string]string{
			secretKey: privateKey,
		},
		Type: corev1.SecretTypeOpaque,
	}

	// Set owner reference so secret is deleted with Keypair
	if err := ctrl.SetControllerReference(obj, secret, actuator.k8sClient.Scheme()); err != nil {
		return err
	}

	// Create or update the secret
	existing := &corev1.Secret{}
	err := actuator.k8sClient.Get(ctx, client.ObjectKey{
		Name:      secretName,
		Namespace: obj.Namespace,
	}, existing)

	if err == nil {
		// Secret exists, update it
		// Note: StringData is write-only, so we must update Data (bytes) instead
		if existing.Data == nil {
			existing.Data = make(map[string][]byte)
		}
		existing.Data[secretKey] = []byte(privateKey)
		if err := actuator.k8sClient.Update(ctx, existing); err != nil {
			return err
		}
		log.Info("Updated private key secret", "secret", secretName)
	} else {
		// Secret doesn't exist, create it
		if err := actuator.k8sClient.Create(ctx, secret); err != nil {
			return err
		}
		log.Info("Created private key secret", "secret", secretName)
	}

	return nil
}

func (actuator keypairActuator) DeleteResource(ctx context.Context, _ orcObjectPT, Keypair *osResourceT) progress.ReconcileStatus {
	// Use the userID from the OpenStack resource itself.
	// If empty, OpenStack will use the authenticated user.
	return progress.WrapError(actuator.osClient.DeleteKeyPair(ctx, Keypair.Name, Keypair.UserID))
}

func (actuator keypairActuator) GetResourceReconcilers(ctx context.Context, orcObject orcObjectPT, osResource *osResourceT, controller interfaces.ResourceController) ([]resourceReconciler, progress.ReconcileStatus) {
	// Keypairs are immutable - no update reconcilers needed
	return []resourceReconciler{}, nil
}

type KeypairHelperFactory struct{}

var _ helperFactory = KeypairHelperFactory{}

func newActuator(ctx context.Context, orcObject *orcv1alpha1.KeyPair, controller interfaces.ResourceController) (keypairActuator, progress.ReconcileStatus) {
	log := ctrl.LoggerFrom(ctx)

	// Ensure credential secrets exist and have our finalizer
	_, reconcileStatus := credentialsDependency.GetDependencies(ctx, controller.GetK8sClient(), orcObject, func(*corev1.Secret) bool { return true })
	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return keypairActuator{}, reconcileStatus
	}

	clientScope, err := controller.GetScopeFactory().NewClientScopeFromObject(ctx, controller.GetK8sClient(), log, orcObject)
	if err != nil {
		return keypairActuator{}, progress.WrapError(err)
	}
	osClient, err := clientScope.NewComputeClient()
	if err != nil {
		return keypairActuator{}, progress.WrapError(err)
	}

	return keypairActuator{
		osClient:  osClient,
		k8sClient: controller.GetK8sClient(),
	}, nil
}

func (KeypairHelperFactory) NewAPIObjectAdapter(obj orcObjectPT) adapterI {
	return keypairAdapter{obj}
}

func (KeypairHelperFactory) NewCreateActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (createResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, orcObject, controller)
}

func (KeypairHelperFactory) NewDeleteActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (deleteResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, orcObject, controller)
}
