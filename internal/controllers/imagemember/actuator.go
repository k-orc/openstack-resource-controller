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

package imagemember

import (
	"context"
	"iter"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/members"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v3/internal/controllers/generic/interfaces"
	"github.com/k-orc/openstack-resource-controller/v3/internal/controllers/generic/progress"
	"github.com/k-orc/openstack-resource-controller/v3/internal/logging"
	"github.com/k-orc/openstack-resource-controller/v3/internal/osclients"
	"github.com/k-orc/openstack-resource-controller/v3/internal/util/dependency"
	orcerrors "github.com/k-orc/openstack-resource-controller/v3/internal/util/errors"
)

const (
	// statusPending is the status of a member which has not accepted or
	// rejected the image. A pending member is treated as not existing:
	// setting accepted or rejected creates the resource, and deleting it
	// resets the status to pending.
	statusPending = "pending"

	// The time to wait for the image owner to add the member
	memberPollingPeriod = 30 * time.Second
)

// OpenStack resource types
type (
	osResourceT = members.Member

	createResourceActuator    = interfaces.CreateResourceActuator[orcObjectPT, orcObjectT, filterT, osResourceT]
	deleteResourceActuator    = interfaces.DeleteResourceActuator[orcObjectPT, orcObjectT, osResourceT]
	reconcileResourceActuator = interfaces.ReconcileResourceActuator[orcObjectPT, osResourceT]
	resourceReconciler        = interfaces.ResourceReconciler[orcObjectPT, osResourceT]
	helperFactory             = interfaces.ResourceHelperFactory[orcObjectPT, orcObjectT, resourceSpecT, filterT, osResourceT]
)

type imagememberActuator struct {
	osClient  osclients.ImageClient
	k8sClient client.Client

	// imageID is the ID of the shared image. It is empty if the image no
	// longer exists, in which case neither does the member.
	imageID string
	// memberID is the ID of the project of our credentials
	memberID string
}

var _ createResourceActuator = imagememberActuator{}
var _ deleteResourceActuator = imagememberActuator{}
var _ reconcileResourceActuator = imagememberActuator{}

func (imagememberActuator) GetResourceID(osResource *osResourceT) string {
	return osResource.MemberID
}

func (actuator imagememberActuator) notFound() error {
	return gophercloud.ErrResourceNotFound{Name: actuator.memberID, ResourceType: "image member"}
}

func (actuator imagememberActuator) GetOSResourceByID(ctx context.Context, id string) (*osResourceT, progress.ReconcileStatus) {
	if actuator.imageID == "" {
		return nil, progress.WrapError(actuator.notFound())
	}
	member, err := actuator.osClient.GetImageMember(ctx, actuator.imageID, id)
	if err != nil {
		return nil, progress.WrapError(err)
	}
	if member.Status == statusPending {
		return nil, progress.WrapError(actuator.notFound())
	}
	return member, nil
}

// getMember returns our member if it has accepted or rejected the image
func (actuator imagememberActuator) getMember(ctx context.Context) iter.Seq2[*osResourceT, error] {
	return func(yield func(*osResourceT, error) bool) {
		member, err := actuator.osClient.GetImageMember(ctx, actuator.imageID, actuator.memberID)
		if err != nil {
			if !orcerrors.IsNotFound(err) {
				yield(nil, err)
			}
			return
		}
		if member.Status != statusPending {
			yield(member, nil)
		}
	}
}

func (actuator imagememberActuator) ListOSResourcesForAdoption(ctx context.Context, _ orcObjectPT) (iter.Seq2[*osResourceT, error], bool) {
	return actuator.getMember(ctx), true
}

func (actuator imagememberActuator) ListOSResourcesForImport(ctx context.Context, _ orcObjectPT, _ filterT) (iter.Seq2[*osResourceT, error], progress.ReconcileStatus) {
	return actuator.getMember(ctx), nil
}

func (actuator imagememberActuator) CreateResource(ctx context.Context, obj orcObjectPT) (*osResourceT, progress.ReconcileStatus) {
	return actuator.setStatus(ctx, desiredStatus(obj))
}

func (actuator imagememberActuator) setStatus(ctx context.Context, status orcv1alpha1.ImageMembershipStatus) (*osResourceT, progress.ReconcileStatus) {
	member, err := actuator.osClient.UpdateImageMember(ctx, actuator.imageID, actuator.memberID, members.UpdateOpts{Status: string(status)})
	if err != nil {
		// The image owner has not added us as a member yet, or the image is
		// not shared
		if orcerrors.IsNotFound(err) || gophercloud.ResponseCodeIs(err, 403) {
			return nil, progress.NewReconcileStatus().
				WithProgressMessage("Waiting for the image owner to add the project as a member").
				WithRequeue(memberPollingPeriod)
		}
		return nil, progress.WrapError(err)
	}
	return member, nil
}

func (actuator imagememberActuator) DeleteResource(ctx context.Context, _ orcObjectPT, _ *osResourceT) progress.ReconcileStatus {
	_, err := actuator.osClient.UpdateImageMember(ctx, actuator.imageID, actuator.memberID, members.UpdateOpts{Status: statusPending})
	if orcerrors.IsNotFound(err) {
		return nil
	}
	return progress.WrapError(err)
}

func (actuator imagememberActuator) reconcileStatus(ctx context.Context, obj orcObjectPT, osResource *osResourceT) progress.ReconcileStatus {
	status := desiredStatus(obj)
	if osResource.Status == string(status) {
		return nil
	}

	ctrl.LoggerFrom(ctx).V(logging.Verbose).Info("Updating member status", "status", status)
	if _, reconcileStatus := actuator.setStatus(ctx, status); reconcileStatus != nil {
		return reconcileStatus
	}
	return progress.NeedsRefresh()
}

func desiredStatus(obj orcObjectPT) orcv1alpha1.ImageMembershipStatus {
	return ptr.Deref(obj.Spec.Resource.Status, orcv1alpha1.ImageMembershipStatusAccepted)
}

func (actuator imagememberActuator) GetResourceReconcilers(ctx context.Context, orcObject orcObjectPT, osResource *osResourceT, controller interfaces.ResourceController) ([]resourceReconciler, progress.ReconcileStatus) {
	return []resourceReconciler{
		actuator.reconcileStatus,
	}, nil
}

type imagememberHelperFactory struct{}

var _ helperFactory = imagememberHelperFactory{}

func imageRef(orcObject *orcv1alpha1.ImageMember) *orcv1alpha1.KubernetesNameRef {
	if orcObject.Spec.Resource != nil {
		return &orcObject.Spec.Resource.ImageRef
	}
	if orcObject.Spec.Import != nil && orcObject.Spec.Import.Filter != nil {
		return orcObject.Spec.Import.Filter.ImageRef
	}
	return nil
}

func newActuator(ctx context.Context, orcObject *orcv1alpha1.ImageMember, controller interfaces.ResourceController, deleting bool) (imagememberActuator, progress.ReconcileStatus) {
	log := ctrl.LoggerFrom(ctx)

	// Ensure credential secrets exist and have our finalizer
	_, reconcileStatus := credentialsDependency.RequireDependencies(ctx, controller.GetK8sClient(), orcObject, func(*corev1.Secret) bool { return true })
	if needsReschedule, _ := reconcileStatus.NeedsReschedule(); needsReschedule {
		return imagememberActuator{}, reconcileStatus
	}

	ref := imageRef(orcObject)
	if ref == nil {
		return imagememberActuator{}, progress.WrapError(
			orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "import by ID is not supported: use import.filter.imageRef"))
	}

	// We don't add a finalizer to the image: its owner must always be able to
	// delete it.
	image, imageRS := dependency.FetchDependency(
		ctx, controller.GetK8sClient(), orcObject.Namespace, ref, "Image",
		func(image *orcv1alpha1.Image) bool {
			return orcv1alpha1.IsAvailable(image) && image.Status.ID != nil
		},
	)
	if needsReschedule, _ := imageRS.NeedsReschedule(); needsReschedule && !deleting {
		return imagememberActuator{}, imageRS
	}

	clientScope, err := controller.GetScopeFactory().NewClientScopeFromObject(ctx, controller.GetK8sClient(), log, orcObject)
	if err != nil {
		return imagememberActuator{}, progress.WrapError(err)
	}
	osClient, err := clientScope.NewImageClient()
	if err != nil {
		return imagememberActuator{}, progress.WrapError(err)
	}
	memberID, err := osClient.GetAuthenticatedProjectID()
	if err != nil {
		return imagememberActuator{}, progress.WrapError(err)
	}

	actuator := imagememberActuator{
		osClient:  osClient,
		k8sClient: controller.GetK8sClient(),
		memberID:  memberID,
	}
	if image != nil {
		actuator.imageID = ptr.Deref(image.Status.ID, "")
	}
	return actuator, nil
}

func (imagememberHelperFactory) NewAPIObjectAdapter(obj orcObjectPT) adapterI {
	return imagememberAdapter{obj}
}

func (imagememberHelperFactory) NewCreateActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (createResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, orcObject, controller, false)
}

func (imagememberHelperFactory) NewDeleteActuator(ctx context.Context, orcObject orcObjectPT, controller interfaces.ResourceController) (deleteResourceActuator, progress.ReconcileStatus) {
	return newActuator(ctx, orcObject, controller, true)
}
