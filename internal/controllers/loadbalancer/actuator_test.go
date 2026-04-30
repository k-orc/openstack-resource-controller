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

package loadbalancer

import (
	"context"
	"iter"
	"testing"

	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/loadbalancers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v2/internal/osclients/mock"
)

// ---------------------------------------------------------------------------
// Pure unit tests (no envtest needed)
// ---------------------------------------------------------------------------

func TestNeedsUpdate(t *testing.T) {
	testCases := []struct {
		name         string
		updateOpts   loadbalancers.UpdateOpts
		expectChange bool
	}{
		{
			name:         "Empty base opts",
			updateOpts:   loadbalancers.UpdateOpts{},
			expectChange: false,
		},
		{
			name:         "Updated name",
			updateOpts:   loadbalancers.UpdateOpts{Name: ptr.To("updated")},
			expectChange: true,
		},
		{
			name:         "Updated description",
			updateOpts:   loadbalancers.UpdateOpts{Description: ptr.To("updated")},
			expectChange: true,
		},
		{
			name:         "Updated admin state",
			updateOpts:   loadbalancers.UpdateOpts{AdminStateUp: ptr.To(false)},
			expectChange: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			got, err := needsUpdate(tt.updateOpts)
			if err != nil {
				t.Fatalf("needsUpdate() unexpected error: %v", err)
			}
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})
	}
}

func TestHandleNameUpdate(t *testing.T) {
	ptrToName := ptr.To[orcv1alpha1.OpenStackName]
	testCases := []struct {
		name          string
		newValue      *orcv1alpha1.OpenStackName
		existingValue string
		expectChange  bool
	}{
		{name: "Identical", newValue: ptrToName("name"), existingValue: "name", expectChange: false},
		{name: "Different", newValue: ptrToName("new-name"), existingValue: "name", expectChange: true},
		{name: "No value provided, existing is identical to object name", newValue: nil, existingValue: "object-name", expectChange: false},
		{name: "No value provided, existing is different from object name", newValue: nil, existingValue: "different-from-object-name", expectChange: true},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resource := &orcv1alpha1.LoadBalancer{}
			resource.Name = "object-name"
			resource.Spec = orcv1alpha1.LoadBalancerSpec{
				Resource: &orcv1alpha1.LoadBalancerResourceSpec{
					Name:      tt.newValue,
					SubnetRef: ptr.To[orcv1alpha1.KubernetesNameRef]("some-subnet"),
				},
			}
			osResource := &loadbalancers.LoadBalancer{Name: tt.existingValue}

			updateOpts := loadbalancers.UpdateOpts{}
			handleNameUpdate(&updateOpts, resource, osResource)

			got, err := needsUpdate(updateOpts)
			if err != nil {
				t.Fatalf("needsUpdate() unexpected error: %v", err)
			}
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})
	}
}

func TestHandleDescriptionUpdate(t *testing.T) {
	ptrToDescription := ptr.To[orcv1alpha1.NeutronDescription]
	testCases := []struct {
		name          string
		newValue      *orcv1alpha1.NeutronDescription
		existingValue string
		expectChange  bool
	}{
		{name: "Identical", newValue: ptrToDescription("desc"), existingValue: "desc", expectChange: false},
		{name: "Different", newValue: ptrToDescription("new-desc"), existingValue: "desc", expectChange: true},
		{name: "No value provided, existing is set", newValue: nil, existingValue: "desc", expectChange: true},
		{name: "No value provided, existing is empty", newValue: nil, existingValue: "", expectChange: false},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resource := &orcv1alpha1.LoadBalancerResourceSpec{Description: tt.newValue}
			osResource := &loadbalancers.LoadBalancer{Description: tt.existingValue}

			updateOpts := loadbalancers.UpdateOpts{}
			handleDescriptionUpdate(&updateOpts, resource, osResource)

			got, err := needsUpdate(updateOpts)
			if err != nil {
				t.Fatalf("needsUpdate() unexpected error: %v", err)
			}
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})
	}
}

func TestHandleAdminStateUpUpdate(t *testing.T) {
	testCases := []struct {
		name          string
		newValue      *bool
		existingValue bool
		expectChange  bool
	}{
		{name: "Identical true", newValue: ptr.To(true), existingValue: true, expectChange: false},
		{name: "Identical false", newValue: ptr.To(false), existingValue: false, expectChange: false},
		{name: "Changed to false", newValue: ptr.To(false), existingValue: true, expectChange: true},
		{name: "Changed to true", newValue: ptr.To(true), existingValue: false, expectChange: true},
		// Default (nil) is treated as true
		{name: "Default (nil), existing is true", newValue: nil, existingValue: true, expectChange: false},
		{name: "Default (nil), existing is false", newValue: nil, existingValue: false, expectChange: true},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resource := &orcv1alpha1.LoadBalancerResourceSpec{AdminStateUp: tt.newValue}
			osResource := &loadbalancers.LoadBalancer{AdminStateUp: tt.existingValue}

			updateOpts := loadbalancers.UpdateOpts{}
			handleAdminStateUpUpdate(&updateOpts, resource, osResource)

			got, err := needsUpdate(updateOpts)
			if err != nil {
				t.Fatalf("needsUpdate() unexpected error: %v", err)
			}
			if got != tt.expectChange {
				t.Errorf("Expected change: %v, got: %v", tt.expectChange, got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ResourceAvailableStatus tests (pure unit tests)
// ---------------------------------------------------------------------------

func TestResourceAvailableStatus(t *testing.T) {
	sw := loadbalancerStatusWriter{}

	testCases := []struct {
		name                string
		orcObject           *orcv1alpha1.LoadBalancer
		osResource          *loadbalancers.LoadBalancer
		wantCondition       metav1.ConditionStatus
		wantNeedsReschedule bool
	}{
		{
			name:                "nil osResource, no status ID → ConditionFalse",
			orcObject:           &orcv1alpha1.LoadBalancer{},
			osResource:          nil,
			wantCondition:       metav1.ConditionFalse,
			wantNeedsReschedule: false,
		},
		{
			name: "nil osResource, has status ID → ConditionUnknown",
			orcObject: &orcv1alpha1.LoadBalancer{
				Status: orcv1alpha1.LoadBalancerStatus{ID: ptr.To("some-id")},
			},
			osResource:          nil,
			wantCondition:       metav1.ConditionUnknown,
			wantNeedsReschedule: false,
		},
		{
			name:      "ACTIVE → ConditionTrue",
			orcObject: &orcv1alpha1.LoadBalancer{},
			osResource: &loadbalancers.LoadBalancer{
				ProvisioningStatus: ProvisioningStatusActive,
			},
			wantCondition:       metav1.ConditionTrue,
			wantNeedsReschedule: false,
		},
		{
			name:      "PENDING_CREATE → ConditionUnknown + WaitingOnOpenStack",
			orcObject: &orcv1alpha1.LoadBalancer{},
			osResource: &loadbalancers.LoadBalancer{
				ProvisioningStatus: ProvisioningStatusPendingCreate,
			},
			wantCondition:       metav1.ConditionUnknown,
			wantNeedsReschedule: true,
		},
		{
			name:      "PENDING_UPDATE → ConditionUnknown + WaitingOnOpenStack",
			orcObject: &orcv1alpha1.LoadBalancer{},
			osResource: &loadbalancers.LoadBalancer{
				ProvisioningStatus: ProvisioningStatusPendingUpdate,
			},
			wantCondition:       metav1.ConditionUnknown,
			wantNeedsReschedule: true,
		},
		{
			name:      "PENDING_DELETE → ConditionUnknown + WaitingOnOpenStack",
			orcObject: &orcv1alpha1.LoadBalancer{},
			osResource: &loadbalancers.LoadBalancer{
				ProvisioningStatus: ProvisioningStatusPendingDelete,
			},
			wantCondition:       metav1.ConditionUnknown,
			wantNeedsReschedule: true,
		},
		{
			name:      "ERROR → ConditionFalse",
			orcObject: &orcv1alpha1.LoadBalancer{},
			osResource: &loadbalancers.LoadBalancer{
				ProvisioningStatus: ProvisioningStatusError,
			},
			wantCondition:       metav1.ConditionFalse,
			wantNeedsReschedule: false,
		},
		{
			name:      "Unknown provisioning status → ConditionUnknown + WaitingOnOpenStack",
			orcObject: &orcv1alpha1.LoadBalancer{},
			osResource: &loadbalancers.LoadBalancer{
				ProvisioningStatus: "SOME_UNKNOWN_STATUS",
			},
			wantCondition:       metav1.ConditionUnknown,
			wantNeedsReschedule: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			condStatus, rs := sw.ResourceAvailableStatus(tt.orcObject, tt.osResource)
			if condStatus != tt.wantCondition {
				t.Errorf("ResourceAvailableStatus() condition = %v, want %v", condStatus, tt.wantCondition)
			}
			needsReschedule, _ := rs.NeedsReschedule()
			if needsReschedule != tt.wantNeedsReschedule {
				t.Errorf("ResourceAvailableStatus() needsReschedule = %v, want %v", needsReschedule, tt.wantNeedsReschedule)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// DeleteResource tests (pure unit tests using mock OS client)
// ---------------------------------------------------------------------------

func TestDeleteResource(t *testing.T) {
	const lbID = "aabbccdd-1234-5678-abcd-000000000001"

	testCases := []struct {
		name                string
		provisioningStatus  string
		expectDeleteCalled  bool
		wantNeedsReschedule bool
	}{
		{
			name:                "ACTIVE → calls DeleteLoadBalancer",
			provisioningStatus:  ProvisioningStatusActive,
			expectDeleteCalled:  true,
			wantNeedsReschedule: false,
		},
		{
			name:                "PENDING_DELETE → waits, no delete call",
			provisioningStatus:  ProvisioningStatusPendingDelete,
			expectDeleteCalled:  false,
			wantNeedsReschedule: true,
		},
		{
			name:                "PENDING_CREATE → waits, no delete call",
			provisioningStatus:  ProvisioningStatusPendingCreate,
			expectDeleteCalled:  false,
			wantNeedsReschedule: true,
		},
		{
			name:                "PENDING_UPDATE → waits, no delete call",
			provisioningStatus:  ProvisioningStatusPendingUpdate,
			expectDeleteCalled:  false,
			wantNeedsReschedule: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			mockctrl := gomock.NewController(t)
			lbClient := mock.NewMockLoadBalancerClient(mockctrl)

			if tt.expectDeleteCalled {
				lbClient.EXPECT().
					DeleteLoadBalancer(gomock.Any(), lbID, gomock.Nil()).
					Return(nil)
			}

			actuator := loadbalancerActuator{osClient: lbClient}
			osResource := &loadbalancers.LoadBalancer{
				ID:                 lbID,
				ProvisioningStatus: tt.provisioningStatus,
			}

			rs := actuator.DeleteResource(context.TODO(), &orcv1alpha1.LoadBalancer{}, osResource)
			needsReschedule, err := rs.NeedsReschedule()
			if err != nil {
				t.Fatalf("DeleteResource() unexpected error: %v", err)
			}
			if needsReschedule != tt.wantNeedsReschedule {
				t.Errorf("DeleteResource() needsReschedule = %v, want %v", needsReschedule, tt.wantNeedsReschedule)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Namespace helpers for Ginkgo envtest-based tests
// ---------------------------------------------------------------------------

func createNamespace(ctx context.Context) string {
	ns := &corev1.Namespace{}
	ns.GenerateName = "lb-test-"
	Expect(k8sClient.Create(ctx, ns)).To(Succeed())
	return ns.Name
}

func deleteNamespace(ctx context.Context, name string) {
	ns := &corev1.Namespace{}
	Expect(k8sClient.Get(ctx, client.ObjectKey{Name: name}, ns)).To(Succeed())
	Expect(k8sClient.Delete(ctx, ns)).To(Succeed())
}

// ---------------------------------------------------------------------------
// Dependency object helpers
// ---------------------------------------------------------------------------

// makeAvailableSubnet creates a Subnet in the given namespace with status.ID set
// and an Available condition = True. Uses unmanaged policy with ID import to
// satisfy CRD validation requirements without needing a real network spec.
// The loadbalancer finalizer is pre-added so that EnsureFinalizer (which uses
// SSA patching) skips the patch operation in tests.
func makeAvailableSubnet(ctx context.Context, namespace, name, subnetID string) *orcv1alpha1.Subnet {
	subnet := &orcv1alpha1.Subnet{
		ObjectMeta: metav1.ObjectMeta{
			Name:       name,
			Namespace:  namespace,
			Finalizers: []string{finalizer},
		},
		Spec: orcv1alpha1.SubnetSpec{
			CloudCredentialsRef: orcv1alpha1.CloudCredentialsReference{
				SecretName: "credentials",
				CloudName:  "openstack",
			},
			ManagementPolicy: orcv1alpha1.ManagementPolicyUnmanaged,
			Import: &orcv1alpha1.SubnetImport{
				ID: ptr.To(subnetID),
			},
		},
	}
	Expect(k8sClient.Create(ctx, subnet)).To(Succeed())

	subnet.Status = orcv1alpha1.SubnetStatus{
		ID: ptr.To(subnetID),
		Conditions: []metav1.Condition{
			{
				Type:               orcv1alpha1.ConditionAvailable,
				Status:             metav1.ConditionTrue,
				Reason:             orcv1alpha1.ConditionReasonSuccess,
				LastTransitionTime: metav1.Now(),
			},
		},
	}
	Expect(k8sClient.Status().Update(ctx, subnet)).To(Succeed())
	return subnet
}

// makeAvailableNetwork creates an available Network in the given namespace.
// Uses unmanaged policy with ID import to satisfy CRD validation requirements.
// The loadbalancer finalizer is pre-added so that EnsureFinalizer skips the
// SSA patch operation in tests.
func makeAvailableNetwork(ctx context.Context, namespace, name, networkID string) *orcv1alpha1.Network {
	network := &orcv1alpha1.Network{
		ObjectMeta: metav1.ObjectMeta{
			Name:       name,
			Namespace:  namespace,
			Finalizers: []string{finalizer},
		},
		Spec: orcv1alpha1.NetworkSpec{
			CloudCredentialsRef: orcv1alpha1.CloudCredentialsReference{
				SecretName: "credentials",
				CloudName:  "openstack",
			},
			ManagementPolicy: orcv1alpha1.ManagementPolicyUnmanaged,
			Import: &orcv1alpha1.NetworkImport{
				ID: ptr.To(networkID),
			},
		},
	}
	Expect(k8sClient.Create(ctx, network)).To(Succeed())

	network.Status = orcv1alpha1.NetworkStatus{
		ID: ptr.To(networkID),
		Conditions: []metav1.Condition{
			{
				Type:               orcv1alpha1.ConditionAvailable,
				Status:             metav1.ConditionTrue,
				Reason:             orcv1alpha1.ConditionReasonSuccess,
				LastTransitionTime: metav1.Now(),
			},
		},
	}
	Expect(k8sClient.Status().Update(ctx, network)).To(Succeed())
	return network
}

// makeAvailablePort creates an available Port in the given namespace.
// Uses unmanaged policy with ID import to satisfy CRD validation requirements.
// The loadbalancer finalizer is pre-added so that EnsureFinalizer skips the
// SSA patch operation in tests.
func makeAvailablePort(ctx context.Context, namespace, name, portID string) *orcv1alpha1.Port {
	port := &orcv1alpha1.Port{
		ObjectMeta: metav1.ObjectMeta{
			Name:       name,
			Namespace:  namespace,
			Finalizers: []string{finalizer},
		},
		Spec: orcv1alpha1.PortSpec{
			CloudCredentialsRef: orcv1alpha1.CloudCredentialsReference{
				SecretName: "credentials",
				CloudName:  "openstack",
			},
			ManagementPolicy: orcv1alpha1.ManagementPolicyUnmanaged,
			Import: &orcv1alpha1.PortImport{
				ID: ptr.To(portID),
			},
		},
	}
	Expect(k8sClient.Create(ctx, port)).To(Succeed())

	port.Status = orcv1alpha1.PortStatus{
		ID: ptr.To(portID),
		Conditions: []metav1.Condition{
			{
				Type:               orcv1alpha1.ConditionAvailable,
				Status:             metav1.ConditionTrue,
				Reason:             orcv1alpha1.ConditionReasonSuccess,
				LastTransitionTime: metav1.Now(),
			},
		},
	}
	Expect(k8sClient.Status().Update(ctx, port)).To(Succeed())
	return port
}

// makeAvailableProject creates an available Project in the given namespace.
// Uses unmanaged policy with ID import to satisfy CRD validation requirements.
// The loadbalancer finalizer is pre-added so that EnsureFinalizer skips the
// SSA patch operation in tests.
func makeAvailableProject(ctx context.Context, namespace, name, projectID string) *orcv1alpha1.Project {
	project := &orcv1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name:       name,
			Namespace:  namespace,
			Finalizers: []string{finalizer},
		},
		Spec: orcv1alpha1.ProjectSpec{
			CloudCredentialsRef: orcv1alpha1.CloudCredentialsReference{
				SecretName: "credentials",
				CloudName:  "openstack",
			},
			ManagementPolicy: orcv1alpha1.ManagementPolicyUnmanaged,
			Import: &orcv1alpha1.ProjectImport{
				ID: ptr.To(projectID),
			},
		},
	}
	Expect(k8sClient.Create(ctx, project)).To(Succeed())

	project.Status = orcv1alpha1.ProjectStatus{
		ID: ptr.To(projectID),
		Conditions: []metav1.Condition{
			{
				Type:               orcv1alpha1.ConditionAvailable,
				Status:             metav1.ConditionTrue,
				Reason:             orcv1alpha1.ConditionReasonSuccess,
				LastTransitionTime: metav1.Now(),
			},
		},
	}
	Expect(k8sClient.Status().Update(ctx, project)).To(Succeed())
	return project
}

// makeUnavailableSubnet creates a Subnet with no Available condition (not ready).
// makeUnavailableSubnet creates a Subnet with no Available condition (not ready).
// Uses unmanaged policy with ID import to satisfy CRD validation requirements.
func makeUnavailableSubnet(ctx context.Context, namespace, name string) *orcv1alpha1.Subnet {
	subnet := &orcv1alpha1.Subnet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: orcv1alpha1.SubnetSpec{
			CloudCredentialsRef: orcv1alpha1.CloudCredentialsReference{
				SecretName: "credentials",
				CloudName:  "openstack",
			},
			ManagementPolicy: orcv1alpha1.ManagementPolicyUnmanaged,
			Import: &orcv1alpha1.SubnetImport{
				ID: ptr.To("aaaaaaaa-0000-0000-0000-000000000000"),
			},
		},
	}
	Expect(k8sClient.Create(ctx, subnet)).To(Succeed())
	return subnet
}

// makeUnavailableProject creates a Project with no Available condition (not ready).
// Uses unmanaged policy with ID import to satisfy CRD validation requirements.
func makeUnavailableProject(ctx context.Context, namespace, name string) *orcv1alpha1.Project {
	project := &orcv1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: orcv1alpha1.ProjectSpec{
			CloudCredentialsRef: orcv1alpha1.CloudCredentialsReference{
				SecretName: "credentials",
				CloudName:  "openstack",
			},
			ManagementPolicy: orcv1alpha1.ManagementPolicyUnmanaged,
			Import: &orcv1alpha1.ProjectImport{
				ID: ptr.To("bbbbbbbb-0000-0000-0000-000000000000"),
			},
		},
	}
	Expect(k8sClient.Create(ctx, project)).To(Succeed())
	return project
}

// ---------------------------------------------------------------------------
// Envtest-based Ginkgo tests: CreateResource
// ---------------------------------------------------------------------------

var _ = Describe("CreateResource", func() {
	var (
		ctx       context.Context
		namespace string
	)

	BeforeEach(func() {
		ctx = context.TODO()
		namespace = createNamespace(ctx)
	})

	AfterEach(func() {
		deleteNamespace(ctx, namespace)
	})

	Context("VIP subnet reference", func() {
		It("should pass vip_subnet_id to Octavia when subnetRef is set", func() {
			const (
				subnetName = "test-subnet"
				subnetID   = "aaaaaaaa-0000-0000-0000-000000000001"
				lbID       = "bbbbbbbb-0000-0000-0000-000000000002"
			)

			_ = makeAvailableSubnet(ctx, namespace, subnetName, subnetID)

			mockctrl := gomock.NewController(GinkgoT())
			lbClient := mock.NewMockLoadBalancerClient(mockctrl)

			var capturedCreateOpts *loadbalancers.CreateOpts
			lbClient.EXPECT().
				CreateLoadBalancer(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, opts loadbalancers.CreateOptsBuilder) (*loadbalancers.LoadBalancer, error) {
					co, ok := opts.(*loadbalancers.CreateOpts)
					if ok {
						capturedCreateOpts = co
					}
					return &loadbalancers.LoadBalancer{
						ID:                 lbID,
						ProvisioningStatus: ProvisioningStatusPendingCreate,
					}, nil
				})

			actuator := loadbalancerActuator{
				osClient:  lbClient,
				k8sClient: k8sClient,
			}

			orcLB := &orcv1alpha1.LoadBalancer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-lb",
					Namespace: namespace,
				},
				Spec: orcv1alpha1.LoadBalancerSpec{
					Resource: &orcv1alpha1.LoadBalancerResourceSpec{
						SubnetRef: ptr.To[orcv1alpha1.KubernetesNameRef](orcv1alpha1.KubernetesNameRef(subnetName)),
					},
				},
			}

			osResource, rs := actuator.CreateResource(ctx, orcLB)
			Expect(osResource).ToNot(BeNil(), "expected OS resource to be returned")
			Expect(osResource.ID).To(Equal(lbID))
			needsReschedule, err := rs.NeedsReschedule()
			Expect(err).ToNot(HaveOccurred())
			Expect(needsReschedule).To(BeFalse())

			// Verify the create opts contained the correct VIP subnet ID
			Expect(capturedCreateOpts).ToNot(BeNil())
			Expect(capturedCreateOpts.VipSubnetID).To(Equal(subnetID))
			Expect(capturedCreateOpts.VipNetworkID).To(BeEmpty())
			Expect(capturedCreateOpts.VipPortID).To(BeEmpty())
		})
	})

	Context("VIP network reference", func() {
		It("should pass vip_network_id to Octavia when networkRef is set", func() {
			const (
				networkName = "test-network"
				networkID   = "cccccccc-0000-0000-0000-000000000003"
				lbID        = "dddddddd-0000-0000-0000-000000000004"
			)

			_ = makeAvailableNetwork(ctx, namespace, networkName, networkID)

			mockctrl := gomock.NewController(GinkgoT())
			lbClient := mock.NewMockLoadBalancerClient(mockctrl)

			var capturedCreateOpts *loadbalancers.CreateOpts
			lbClient.EXPECT().
				CreateLoadBalancer(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, opts loadbalancers.CreateOptsBuilder) (*loadbalancers.LoadBalancer, error) {
					co, ok := opts.(*loadbalancers.CreateOpts)
					if ok {
						capturedCreateOpts = co
					}
					return &loadbalancers.LoadBalancer{
						ID:                 lbID,
						ProvisioningStatus: ProvisioningStatusPendingCreate,
					}, nil
				})

			actuator := loadbalancerActuator{
				osClient:  lbClient,
				k8sClient: k8sClient,
			}

			orcLB := &orcv1alpha1.LoadBalancer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-lb-network",
					Namespace: namespace,
				},
				Spec: orcv1alpha1.LoadBalancerSpec{
					Resource: &orcv1alpha1.LoadBalancerResourceSpec{
						NetworkRef: ptr.To[orcv1alpha1.KubernetesNameRef](orcv1alpha1.KubernetesNameRef(networkName)),
					},
				},
			}

			osResource, rs := actuator.CreateResource(ctx, orcLB)
			Expect(osResource).ToNot(BeNil())
			needsReschedule, err := rs.NeedsReschedule()
			Expect(err).ToNot(HaveOccurred())
			Expect(needsReschedule).To(BeFalse())

			Expect(capturedCreateOpts).ToNot(BeNil())
			Expect(capturedCreateOpts.VipNetworkID).To(Equal(networkID))
			Expect(capturedCreateOpts.VipSubnetID).To(BeEmpty())
			Expect(capturedCreateOpts.VipPortID).To(BeEmpty())
		})
	})

	Context("VIP port reference", func() {
		It("should pass vip_port_id to Octavia when vipPortRef is set", func() {
			const (
				portName = "test-port"
				portID   = "eeeeeeee-0000-0000-0000-000000000005"
				lbID     = "ffffffff-0000-0000-0000-000000000006"
			)

			_ = makeAvailablePort(ctx, namespace, portName, portID)

			mockctrl := gomock.NewController(GinkgoT())
			lbClient := mock.NewMockLoadBalancerClient(mockctrl)

			var capturedCreateOpts *loadbalancers.CreateOpts
			lbClient.EXPECT().
				CreateLoadBalancer(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, opts loadbalancers.CreateOptsBuilder) (*loadbalancers.LoadBalancer, error) {
					co, ok := opts.(*loadbalancers.CreateOpts)
					if ok {
						capturedCreateOpts = co
					}
					return &loadbalancers.LoadBalancer{
						ID:                 lbID,
						ProvisioningStatus: ProvisioningStatusPendingCreate,
					}, nil
				})

			actuator := loadbalancerActuator{
				osClient:  lbClient,
				k8sClient: k8sClient,
			}

			orcLB := &orcv1alpha1.LoadBalancer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-lb-port",
					Namespace: namespace,
				},
				Spec: orcv1alpha1.LoadBalancerSpec{
					Resource: &orcv1alpha1.LoadBalancerResourceSpec{
						VIPPortRef: ptr.To[orcv1alpha1.KubernetesNameRef](orcv1alpha1.KubernetesNameRef(portName)),
					},
				},
			}

			osResource, rs := actuator.CreateResource(ctx, orcLB)
			Expect(osResource).ToNot(BeNil())
			needsReschedule, err := rs.NeedsReschedule()
			Expect(err).ToNot(HaveOccurred())
			Expect(needsReschedule).To(BeFalse())

			Expect(capturedCreateOpts).ToNot(BeNil())
			Expect(capturedCreateOpts.VipPortID).To(Equal(portID))
			Expect(capturedCreateOpts.VipSubnetID).To(BeEmpty())
			Expect(capturedCreateOpts.VipNetworkID).To(BeEmpty())
		})
	})

	Context("Dependency waiting", func() {
		It("should wait when subnet dependency is not yet available", func() {
			const subnetName = "not-ready-subnet"
			_ = makeUnavailableSubnet(ctx, namespace, subnetName)

			mockctrl := gomock.NewController(GinkgoT())
			lbClient := mock.NewMockLoadBalancerClient(mockctrl)
			// No CreateLoadBalancer call expected

			actuator := loadbalancerActuator{
				osClient:  lbClient,
				k8sClient: k8sClient,
			}

			orcLB := &orcv1alpha1.LoadBalancer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-lb-wait-subnet",
					Namespace: namespace,
				},
				Spec: orcv1alpha1.LoadBalancerSpec{
					Resource: &orcv1alpha1.LoadBalancerResourceSpec{
						SubnetRef: ptr.To[orcv1alpha1.KubernetesNameRef](orcv1alpha1.KubernetesNameRef(subnetName)),
					},
				},
			}

			osResource, rs := actuator.CreateResource(ctx, orcLB)
			Expect(osResource).To(BeNil(), "expected no OS resource while waiting on dependency")
			needsReschedule, _ := rs.NeedsReschedule()
			Expect(needsReschedule).To(BeTrue(), "expected reschedule while waiting on subnet")
		})

		It("should wait when subnet dependency does not exist", func() {
			mockctrl := gomock.NewController(GinkgoT())
			lbClient := mock.NewMockLoadBalancerClient(mockctrl)
			// No CreateLoadBalancer call expected

			actuator := loadbalancerActuator{
				osClient:  lbClient,
				k8sClient: k8sClient,
			}

			orcLB := &orcv1alpha1.LoadBalancer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-lb-missing-subnet",
					Namespace: namespace,
				},
				Spec: orcv1alpha1.LoadBalancerSpec{
					Resource: &orcv1alpha1.LoadBalancerResourceSpec{
						SubnetRef: ptr.To[orcv1alpha1.KubernetesNameRef](orcv1alpha1.KubernetesNameRef("nonexistent-subnet")),
					},
				},
			}

			osResource, rs := actuator.CreateResource(ctx, orcLB)
			Expect(osResource).To(BeNil(), "expected no OS resource while waiting on dependency")
			needsReschedule, _ := rs.NeedsReschedule()
			Expect(needsReschedule).To(BeTrue(), "expected reschedule when subnet does not exist")
		})

		It("should wait when project dependency is not yet available", func() {
			const (
				subnetName  = "ready-subnet-for-project-test"
				subnetID    = "11111111-0000-0000-0000-000000000001"
				projectName = "not-ready-project"
			)
			_ = makeAvailableSubnet(ctx, namespace, subnetName, subnetID)
			_ = makeUnavailableProject(ctx, namespace, projectName)

			mockctrl := gomock.NewController(GinkgoT())
			lbClient := mock.NewMockLoadBalancerClient(mockctrl)
			// No CreateLoadBalancer call expected

			actuator := loadbalancerActuator{
				osClient:  lbClient,
				k8sClient: k8sClient,
			}

			orcLB := &orcv1alpha1.LoadBalancer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-lb-wait-project",
					Namespace: namespace,
				},
				Spec: orcv1alpha1.LoadBalancerSpec{
					Resource: &orcv1alpha1.LoadBalancerResourceSpec{
						SubnetRef:  ptr.To[orcv1alpha1.KubernetesNameRef](orcv1alpha1.KubernetesNameRef(subnetName)),
						ProjectRef: ptr.To[orcv1alpha1.KubernetesNameRef](orcv1alpha1.KubernetesNameRef(projectName)),
					},
				},
			}

			osResource, rs := actuator.CreateResource(ctx, orcLB)
			Expect(osResource).To(BeNil(), "expected no OS resource while waiting on project dependency")
			needsReschedule, _ := rs.NeedsReschedule()
			Expect(needsReschedule).To(BeTrue(), "expected reschedule while waiting on project")
		})
	})

	Context("Full configuration", func() {
		It("should pass all optional fields to Octavia", func() {
			const (
				subnetName  = "full-config-subnet"
				subnetID    = "22222222-0000-0000-0000-000000000002"
				projectName = "full-config-project"
				projectID   = "33333333-0000-0000-0000-000000000003"
				lbID        = "44444444-0000-0000-0000-000000000004"
			)

			_ = makeAvailableSubnet(ctx, namespace, subnetName, subnetID)
			_ = makeAvailableProject(ctx, namespace, projectName, projectID)

			mockctrl := gomock.NewController(GinkgoT())
			lbClient := mock.NewMockLoadBalancerClient(mockctrl)

			var capturedCreateOpts *loadbalancers.CreateOpts
			lbClient.EXPECT().
				CreateLoadBalancer(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, opts loadbalancers.CreateOptsBuilder) (*loadbalancers.LoadBalancer, error) {
					co, ok := opts.(*loadbalancers.CreateOpts)
					if ok {
						capturedCreateOpts = co
					}
					return &loadbalancers.LoadBalancer{
						ID:                 lbID,
						ProvisioningStatus: ProvisioningStatusPendingCreate,
					}, nil
				})

			actuator := loadbalancerActuator{
				osClient:  lbClient,
				k8sClient: k8sClient,
			}

			orcLB := &orcv1alpha1.LoadBalancer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-lb-full",
					Namespace: namespace,
				},
				Spec: orcv1alpha1.LoadBalancerSpec{
					Resource: &orcv1alpha1.LoadBalancerResourceSpec{
						Name:         ptr.To[orcv1alpha1.OpenStackName]("my-lb"),
						Description:  ptr.To[orcv1alpha1.NeutronDescription]("my description"),
						SubnetRef:    ptr.To[orcv1alpha1.KubernetesNameRef](orcv1alpha1.KubernetesNameRef(subnetName)),
						ProjectRef:   ptr.To[orcv1alpha1.KubernetesNameRef](orcv1alpha1.KubernetesNameRef(projectName)),
						AdminStateUp: ptr.To(false),
						VIPAddress:   ptr.To[orcv1alpha1.IPvAny]("192.168.1.100"),
						Provider:     ptr.To("amphora"),
						Tags:         []orcv1alpha1.NeutronTag{"tag1", "tag2"},
					},
				},
			}

			osResource, rs := actuator.CreateResource(ctx, orcLB)
			Expect(osResource).ToNot(BeNil())
			needsReschedule, err := rs.NeedsReschedule()
			Expect(err).ToNot(HaveOccurred())
			Expect(needsReschedule).To(BeFalse())

			Expect(capturedCreateOpts).ToNot(BeNil())
			Expect(capturedCreateOpts.Name).To(Equal("my-lb"))
			Expect(capturedCreateOpts.Description).To(Equal("my description"))
			Expect(capturedCreateOpts.VipSubnetID).To(Equal(subnetID))
			Expect(capturedCreateOpts.ProjectID).To(Equal(projectID))
			Expect(capturedCreateOpts.AdminStateUp).ToNot(BeNil())
			Expect(*capturedCreateOpts.AdminStateUp).To(BeFalse())
			Expect(capturedCreateOpts.VipAddress).To(Equal("192.168.1.100"))
			Expect(capturedCreateOpts.Provider).To(Equal("amphora"))
			Expect(capturedCreateOpts.Tags).To(ConsistOf("tag1", "tag2"))
		})
	})
})

// ---------------------------------------------------------------------------
// Envtest-based Ginkgo tests: ListOSResourcesForImport
// ---------------------------------------------------------------------------

var _ = Describe("ListOSResourcesForImport", func() {
	var (
		ctx       context.Context
		namespace string
	)

	BeforeEach(func() {
		ctx = context.TODO()
		namespace = createNamespace(ctx)
	})

	AfterEach(func() {
		deleteNamespace(ctx, namespace)
	})

	It("should filter by name", func() {
		mockctrl := gomock.NewController(GinkgoT())
		lbClient := mock.NewMockLoadBalancerClient(mockctrl)

		var capturedListOpts loadbalancers.ListOpts
		lbClient.EXPECT().
			ListLoadBalancer(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, opts loadbalancers.ListOptsBuilder) iter.Seq2[*loadbalancers.LoadBalancer, error] {
				lo, ok := opts.(loadbalancers.ListOpts)
				if ok {
					capturedListOpts = lo
				}
				return func(yield func(*loadbalancers.LoadBalancer, error) bool) {}
			})

		actuator := loadbalancerActuator{
			osClient:  lbClient,
			k8sClient: k8sClient,
		}

		orcLB := &orcv1alpha1.LoadBalancer{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-import-lb",
				Namespace: namespace,
			},
		}

		filter := orcv1alpha1.LoadBalancerFilter{
			Name: ptr.To[orcv1alpha1.OpenStackName]("my-lb"),
		}

		_, rs := actuator.ListOSResourcesForImport(ctx, orcLB, filter)
		needsReschedule, err := rs.NeedsReschedule()
		Expect(err).ToNot(HaveOccurred())
		Expect(needsReschedule).To(BeFalse())
		Expect(capturedListOpts.Name).To(Equal("my-lb"))
	})

	It("should filter by project reference", func() {
		const (
			projectName = "import-project"
			projectID   = "55555555-0000-0000-0000-000000000005"
		)

		_ = makeAvailableProject(ctx, namespace, projectName, projectID)

		mockctrl := gomock.NewController(GinkgoT())
		lbClient := mock.NewMockLoadBalancerClient(mockctrl)

		var capturedListOpts loadbalancers.ListOpts
		lbClient.EXPECT().
			ListLoadBalancer(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, opts loadbalancers.ListOptsBuilder) iter.Seq2[*loadbalancers.LoadBalancer, error] {
				lo, ok := opts.(loadbalancers.ListOpts)
				if ok {
					capturedListOpts = lo
				}
				return func(yield func(*loadbalancers.LoadBalancer, error) bool) {}
			})

		actuator := loadbalancerActuator{
			osClient:  lbClient,
			k8sClient: k8sClient,
		}

		orcLB := &orcv1alpha1.LoadBalancer{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-import-lb-project",
				Namespace: namespace,
			},
		}

		filter := orcv1alpha1.LoadBalancerFilter{
			ProjectRef: ptr.To[orcv1alpha1.KubernetesNameRef](orcv1alpha1.KubernetesNameRef(projectName)),
		}

		_, rs := actuator.ListOSResourcesForImport(ctx, orcLB, filter)
		needsReschedule, err := rs.NeedsReschedule()
		Expect(err).ToNot(HaveOccurred())
		Expect(needsReschedule).To(BeFalse())
		Expect(capturedListOpts.ProjectID).To(Equal(projectID))
	})

	It("should wait for project dependency when not available", func() {
		const projectName = "not-ready-import-project"
		_ = makeUnavailableProject(ctx, namespace, projectName)

		mockctrl := gomock.NewController(GinkgoT())
		lbClient := mock.NewMockLoadBalancerClient(mockctrl)
		// No ListLoadBalancer call expected

		actuator := loadbalancerActuator{
			osClient:  lbClient,
			k8sClient: k8sClient,
		}

		orcLB := &orcv1alpha1.LoadBalancer{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-import-lb-wait-project",
				Namespace: namespace,
			},
		}

		filter := orcv1alpha1.LoadBalancerFilter{
			ProjectRef: ptr.To[orcv1alpha1.KubernetesNameRef](orcv1alpha1.KubernetesNameRef(projectName)),
		}

		_, rs := actuator.ListOSResourcesForImport(ctx, orcLB, filter)
		needsReschedule, _ := rs.NeedsReschedule()
		Expect(needsReschedule).To(BeTrue(), "expected reschedule while waiting on project dependency")
	})

	It("should resolve vipSubnetRef dependency and pass subnet ID to list opts", func() {
		const (
			subnetName = "import-filter-subnet"
			subnetID   = "66666666-0000-0000-0000-000000000006"
		)

		_ = makeAvailableSubnet(ctx, namespace, subnetName, subnetID)

		mockctrl := gomock.NewController(GinkgoT())
		lbClient := mock.NewMockLoadBalancerClient(mockctrl)

		var capturedListOpts loadbalancers.ListOpts
		lbClient.EXPECT().
			ListLoadBalancer(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, opts loadbalancers.ListOptsBuilder) iter.Seq2[*loadbalancers.LoadBalancer, error] {
				lo, ok := opts.(loadbalancers.ListOpts)
				if ok {
					capturedListOpts = lo
				}
				return func(yield func(*loadbalancers.LoadBalancer, error) bool) {}
			})

		actuator := loadbalancerActuator{
			osClient:  lbClient,
			k8sClient: k8sClient,
		}

		orcLB := &orcv1alpha1.LoadBalancer{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-import-lb-subnet-filter",
				Namespace: namespace,
			},
		}

		filter := orcv1alpha1.LoadBalancerFilter{
			VIPSubnetRef: ptr.To[orcv1alpha1.KubernetesNameRef](orcv1alpha1.KubernetesNameRef(subnetName)),
		}

		_, rs := actuator.ListOSResourcesForImport(ctx, orcLB, filter)
		needsReschedule, err := rs.NeedsReschedule()
		Expect(err).ToNot(HaveOccurred())
		Expect(needsReschedule).To(BeFalse())
		Expect(capturedListOpts.VipSubnetID).To(Equal(subnetID))
	})
})
