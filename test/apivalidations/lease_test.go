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

package apivalidations

import (
	"context"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/api/v1alpha1"
	applyconfigv1alpha1 "github.com/k-orc/openstack-resource-controller/v3/pkg/clients/applyconfiguration/api/v1alpha1"
)

const (
	leaseName = "lease"
	leaseID   = "265c9e4f-0f5a-46e4-9f3f-fb8de25ae120"
)

func leaseStub(namespace *corev1.Namespace) *orcv1alpha1.Lease {
	obj := &orcv1alpha1.Lease{}
	obj.Name = leaseName
	obj.Namespace = namespace.Name
	return obj
}

var (
	leaseStartDate = metav1.NewTime(time.Date(2030, 1, 1, 10, 0, 0, 0, time.UTC))
	leaseEndDate   = metav1.NewTime(time.Date(2030, 1, 1, 11, 0, 0, 0, time.UTC))
)

func testLeaseHostReservation() *applyconfigv1alpha1.LeaseReservationApplyConfiguration {
	return applyconfigv1alpha1.LeaseReservation().
		WithHost(applyconfigv1alpha1.LeaseHostReservation().WithMin(1).WithMax(1))
}

func testLeaseInstanceReservation() *applyconfigv1alpha1.LeaseReservationApplyConfiguration {
	return applyconfigv1alpha1.LeaseReservation().
		WithInstance(applyconfigv1alpha1.LeaseInstanceReservation().
			WithAmount(1).WithVcpus(1).WithMemoryMB(512).WithDiskGB(0))
}

func testLeaseFlavorInstanceReservation() *applyconfigv1alpha1.LeaseReservationApplyConfiguration {
	return applyconfigv1alpha1.LeaseReservation().
		WithFlavorInstance(applyconfigv1alpha1.LeaseFlavorInstanceReservation().
			WithAmount(1).WithFlavorRef("flavor"))
}

func testLeaseResource() *applyconfigv1alpha1.LeaseResourceSpecApplyConfiguration {
	return applyconfigv1alpha1.LeaseResourceSpec().
		WithEndDate(leaseEndDate).
		WithReservations(testLeaseHostReservation())
}

func baseLeasePatch(obj client.Object) *applyconfigv1alpha1.LeaseApplyConfiguration {
	return applyconfigv1alpha1.Lease(obj.GetName(), obj.GetNamespace()).
		WithSpec(applyconfigv1alpha1.LeaseSpec().
			WithCloudCredentialsRef(testCredentials()))
}

func testLeaseImport() *applyconfigv1alpha1.LeaseImportApplyConfiguration {
	return applyconfigv1alpha1.LeaseImport().WithID(leaseID)
}

var _ = Describe("ORC Lease API validations", func() {
	var namespace *corev1.Namespace
	BeforeEach(func() {
		namespace = createNamespace()
	})

	runManagementPolicyTests(func() *corev1.Namespace { return namespace }, managementPolicyTestArgs[*applyconfigv1alpha1.LeaseApplyConfiguration]{
		createObject: func(ns *corev1.Namespace) client.Object { return leaseStub(ns) },
		basePatch: func(obj client.Object) *applyconfigv1alpha1.LeaseApplyConfiguration {
			return baseLeasePatch(obj)
		},
		applyResource: func(p *applyconfigv1alpha1.LeaseApplyConfiguration) {
			p.Spec.WithResource(testLeaseResource())
		},
		applyImport: func(p *applyconfigv1alpha1.LeaseApplyConfiguration) {
			p.Spec.WithImport(testLeaseImport())
		},
		applyEmptyImport: func(p *applyconfigv1alpha1.LeaseApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.LeaseImport())
		},
		applyEmptyFilter: func(p *applyconfigv1alpha1.LeaseApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.LeaseImport().WithFilter(applyconfigv1alpha1.LeaseFilter()))
		},
		applyValidFilter: func(p *applyconfigv1alpha1.LeaseApplyConfiguration) {
			p.Spec.WithImport(applyconfigv1alpha1.LeaseImport().WithFilter(applyconfigv1alpha1.LeaseFilter().WithName("foo")))
		},
		applyManaged: func(p *applyconfigv1alpha1.LeaseApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyManaged)
		},
		applyUnmanaged: func(p *applyconfigv1alpha1.LeaseApplyConfiguration) {
			p.Spec.WithManagementPolicy(orcv1alpha1.ManagementPolicyUnmanaged)
		},
		applyManagedOptions: func(p *applyconfigv1alpha1.LeaseApplyConfiguration) {
			p.Spec.WithManagedOptions(applyconfigv1alpha1.ManagedOptions().WithOnDelete(orcv1alpha1.OnDeleteDetach))
		},
		getManagementPolicy: func(obj client.Object) orcv1alpha1.ManagementPolicy {
			return obj.(*orcv1alpha1.Lease).Spec.ManagementPolicy
		},
		getOnDelete: func(obj client.Object) orcv1alpha1.OnDelete {
			return obj.(*orcv1alpha1.Lease).Spec.ManagedOptions.OnDelete
		},
	})

	It("should permit valid host, instance and flavor instance reservations", func(ctx context.Context) {
		lease := leaseStub(namespace)
		patch := baseLeasePatch(lease)
		patch.Spec.WithResource(applyconfigv1alpha1.LeaseResourceSpec().
			WithStartDate(leaseStartDate).
			WithEndDate(leaseEndDate).
			WithReservations(testLeaseHostReservation(), testLeaseInstanceReservation(), testLeaseFlavorInstanceReservation()))
		Expect(applyObj(ctx, lease, patch)).To(Succeed())
	})

	It("should reject a lease without required field endDate", func(ctx context.Context) {
		lease := leaseStub(namespace)
		patch := baseLeasePatch(lease)
		patch.Spec.WithResource(applyconfigv1alpha1.LeaseResourceSpec().
			WithReservations(testLeaseHostReservation()))
		Expect(applyObj(ctx, lease, patch)).To(MatchError(ContainSubstring("spec.resource.endDate")))
	})

	It("should reject a lease without reservations", func(ctx context.Context) {
		lease := leaseStub(namespace)
		patch := baseLeasePatch(lease)
		patch.Spec.WithResource(applyconfigv1alpha1.LeaseResourceSpec().
			WithEndDate(leaseEndDate))
		Expect(applyObj(ctx, lease, patch)).To(MatchError(ContainSubstring("spec.resource.reservations")))
	})

	It("should reject an endDate before the startDate", func(ctx context.Context) {
		lease := leaseStub(namespace)
		patch := baseLeasePatch(lease)
		patch.Spec.WithResource(testLeaseResource().
			WithStartDate(leaseEndDate).
			WithEndDate(leaseStartDate))
		Expect(applyObj(ctx, lease, patch)).To(MatchError(ContainSubstring("endDate must be later than startDate")))
	})

	It("should reject a lease name longer than 80 characters", func(ctx context.Context) {
		lease := leaseStub(namespace)
		patch := baseLeasePatch(lease)
		patch.Spec.WithResource(testLeaseResource().
			WithName(orcv1alpha1.BlazarName(strings.Repeat("a", 81))))
		Expect(applyObj(ctx, lease, patch)).To(MatchError(ContainSubstring("spec.resource.name")))
	})

	DescribeTable("should require exactly one reservation type",
		func(ctx context.Context, reservation *applyconfigv1alpha1.LeaseReservationApplyConfiguration) {
			lease := leaseStub(namespace)
			patch := baseLeasePatch(lease)
			patch.Spec.WithResource(applyconfigv1alpha1.LeaseResourceSpec().
				WithEndDate(leaseEndDate).
				WithReservations(reservation))
			Expect(applyObj(ctx, lease, patch)).To(MatchError(ContainSubstring("exactly one of host, instance or flavorInstance must be set")))
		},
		Entry("none", applyconfigv1alpha1.LeaseReservation()),
		Entry("host and instance", applyconfigv1alpha1.LeaseReservation().
			WithHost(applyconfigv1alpha1.LeaseHostReservation().WithMin(1).WithMax(1)).
			WithInstance(applyconfigv1alpha1.LeaseInstanceReservation().
				WithAmount(1).WithVcpus(1).WithMemoryMB(512).WithDiskGB(0))),
		Entry("instance and flavor instance", applyconfigv1alpha1.LeaseReservation().
			WithInstance(applyconfigv1alpha1.LeaseInstanceReservation().
				WithAmount(1).WithVcpus(1).WithMemoryMB(512).WithDiskGB(0)).
			WithFlavorInstance(applyconfigv1alpha1.LeaseFlavorInstanceReservation().
				WithAmount(1).WithFlavorRef("flavor"))),
	)

	It("should reject a host reservation with max lower than min", func(ctx context.Context) {
		lease := leaseStub(namespace)
		patch := baseLeasePatch(lease)
		patch.Spec.WithResource(applyconfigv1alpha1.LeaseResourceSpec().
			WithEndDate(leaseEndDate).
			WithReservations(applyconfigv1alpha1.LeaseReservation().
				WithHost(applyconfigv1alpha1.LeaseHostReservation().WithMin(2).WithMax(1))))
		Expect(applyObj(ctx, lease, patch)).To(MatchError(ContainSubstring("max must be greater than or equal to min")))
	})

	DescribeTable("should reject values below the minimum",
		func(ctx context.Context, reservation *applyconfigv1alpha1.LeaseReservationApplyConfiguration, field string) {
			lease := leaseStub(namespace)
			patch := baseLeasePatch(lease)
			patch.Spec.WithResource(applyconfigv1alpha1.LeaseResourceSpec().
				WithEndDate(leaseEndDate).
				WithReservations(reservation))
			Expect(applyObj(ctx, lease, patch)).To(MatchError(ContainSubstring(field)))
		},
		Entry("host min", applyconfigv1alpha1.LeaseReservation().
			WithHost(applyconfigv1alpha1.LeaseHostReservation().WithMin(0).WithMax(1)), "min"),
		Entry("instance amount", applyconfigv1alpha1.LeaseReservation().
			WithInstance(applyconfigv1alpha1.LeaseInstanceReservation().
				WithAmount(0).WithVcpus(1).WithMemoryMB(512).WithDiskGB(0)), "amount"),
		Entry("instance diskGB", applyconfigv1alpha1.LeaseReservation().
			WithInstance(applyconfigv1alpha1.LeaseInstanceReservation().
				WithAmount(1).WithVcpus(1).WithMemoryMB(512).WithDiskGB(-1)), "diskGB"),
		Entry("flavor instance amount", applyconfigv1alpha1.LeaseReservation().
			WithFlavorInstance(applyconfigv1alpha1.LeaseFlavorInstanceReservation().
				WithAmount(0).WithFlavorRef("flavor")), "amount"),
	)

	It("should reject a flavor instance reservation without flavorRef", func(ctx context.Context) {
		lease := leaseStub(namespace)
		patch := baseLeasePatch(lease)
		patch.Spec.WithResource(applyconfigv1alpha1.LeaseResourceSpec().
			WithEndDate(leaseEndDate).
			WithReservations(applyconfigv1alpha1.LeaseReservation().
				WithFlavorInstance(applyconfigv1alpha1.LeaseFlavorInstanceReservation().WithAmount(1))))
		Expect(applyObj(ctx, lease, patch)).To(MatchError(ContainSubstring("flavorRef")))
	})

	It("should be immutable", func(ctx context.Context) {
		lease := leaseStub(namespace)
		patch := baseLeasePatch(lease)
		patch.Spec.WithResource(testLeaseResource())
		Expect(applyObj(ctx, lease, patch)).To(Succeed())

		patch.Spec.WithResource(testLeaseResource().
			WithEndDate(metav1.NewTime(leaseEndDate.Add(time.Hour))))
		Expect(applyObj(ctx, lease, patch)).To(MatchError(ContainSubstring("LeaseResourceSpec is immutable")))
	})
})
