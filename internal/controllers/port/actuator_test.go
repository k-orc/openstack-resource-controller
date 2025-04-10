package port

import (
	"testing"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	"k8s.io/utils/ptr"
)

func TestPortSecurityEnabled(t *testing.T) {
	testCases := []struct {
		name              string
		portSecurityState orcv1alpha1.PortSecurityState
		networkStatus     orcv1alpha1.NetworkStatus
		expectedPortSec   bool
	}{
		{
			name:              "explicitly enabled",
			portSecurityState: orcv1alpha1.PortSecurityEnabled,
			networkStatus:     orcv1alpha1.NetworkStatus{},
			expectedPortSec:   true,
		},
		{
			name:              "explicitly disabled",
			portSecurityState: orcv1alpha1.PortSecurityDisabled,
			networkStatus:     orcv1alpha1.NetworkStatus{},
			expectedPortSec:   false,
		},
		{
			name:              "inherit with network port security enabled",
			portSecurityState: orcv1alpha1.PortSecurityInherit,
			networkStatus: orcv1alpha1.NetworkStatus{
				Resource: &orcv1alpha1.NetworkResourceStatus{
					PortSecurityEnabled: ptr.To(true),
				},
			},
			expectedPortSec: true,
		},
		{
			name:              "inherit with network port security disabled",
			portSecurityState: orcv1alpha1.PortSecurityInherit,
			networkStatus: orcv1alpha1.NetworkStatus{
				Resource: &orcv1alpha1.NetworkResourceStatus{
					PortSecurityEnabled: ptr.To(false),
				},
			},
			expectedPortSec: false,
		},
		{
			name:              "inherit with network port security nil (default enabled)",
			portSecurityState: orcv1alpha1.PortSecurityInherit,
			networkStatus:     orcv1alpha1.NetworkStatus{},
			expectedPortSec:   true,
		},
		{
			name:              "unknown state (default enabled)",
			portSecurityState: "unknown",
			networkStatus:     orcv1alpha1.NetworkStatus{},
			expectedPortSec:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := portSecurityEnabled(tc.portSecurityState, tc.networkStatus)
			if result != tc.expectedPortSec {
				t.Errorf("Expected port security to be %v, got %v", tc.expectedPortSec, result)
			}
		})
	}
}
