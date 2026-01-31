/*
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

package networksegmentrange

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/controller"

	ctrl "sigs.k8s.io/controller-runtime"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/interfaces"
	genericreconciler "github.com/k-orc/openstack-resource-controller/v2/internal/controllers/generic/reconciler"
	"github.com/k-orc/openstack-resource-controller/v2/internal/scope"
)

type networksegmentrangeReconcilerConstructor struct {
	scopeFactory scope.Factory
}

func New(scopeFactory scope.Factory) interfaces.Controller {
	return networksegmentrangeReconcilerConstructor{scopeFactory: scopeFactory}
}

func (networksegmentrangeReconcilerConstructor) GetName() string {
	return "networksegmentrange"
}

// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=networksegmentranges,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openstack.k-orc.cloud,resources=networksegmentranges/status,verbs=get;update;patch

func (c networksegmentrangeReconcilerConstructor) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	// Temporarily disabled until code generation completes
	return nil
	/*
	reconciler := genericreconciler.NewController(
		mgr.GetClient(), c.scopeFactory, c.GetName(),
		networksegmentrangeHelperFactory{},
		networksegmentrangeStatusWriter{},
	)

	return ctrl.NewControllerManagedBy(mgr).
		For(&orcv1alpha1.NetworkSegmentRange{}).
		WithOptions(options).
		Complete(reconciler)
	*/
}
