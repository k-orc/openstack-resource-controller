package controller

import (
	"testing"

	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/routers"
)

func TestRouterEquals(t *testing.T) {
	for _, tc := range [...]struct {
		name      string
		candidate routers.Router
		opts      routers.CreateOpts
		expected  bool
	}{
		{
			name: "same external fixed IPs",
			candidate: routers.Router{
				GatewayInfo: routers.GatewayInfo{
					ExternalFixedIPs: []routers.ExternalFixedIP{
						{IPAddress: "10.0.0.1", SubnetID: "06125cfa-fe0d-40c7-af74-d3f11140f2e0"},
						{IPAddress: "10.0.0.2", SubnetID: "06125cfa-fe0d-40c7-af74-d3f11140f2e1"},
					},
				},
			},
			opts: routers.CreateOpts{
				GatewayInfo: &routers.GatewayInfo{
					ExternalFixedIPs: []routers.ExternalFixedIP{
						{IPAddress: "10.0.0.2", SubnetID: "06125cfa-fe0d-40c7-af74-d3f11140f2e1"},
						{IPAddress: "10.0.0.1", SubnetID: "06125cfa-fe0d-40c7-af74-d3f11140f2e0"},
					},
				},
			},
			expected: true,
		},
		{
			name: "different external fixed IPs",
			candidate: routers.Router{
				GatewayInfo: routers.GatewayInfo{
					ExternalFixedIPs: []routers.ExternalFixedIP{
						{IPAddress: "10.0.0.1", SubnetID: "06125cfa-fe0d-40c7-af74-d3f11140f2e0"},
						{IPAddress: "10.0.0.2", SubnetID: "06125cfa-fe0d-40c7-af74-d3f11140f2e1"},
					},
				},
			},
			opts: routers.CreateOpts{
				GatewayInfo: &routers.GatewayInfo{
					ExternalFixedIPs: []routers.ExternalFixedIP{
						{IPAddress: "10.0.0.2", SubnetID: "06125cfa-fe0d-40c7-af74-d3f11140f2e1"},
						{IPAddress: "10.0.0.0", SubnetID: "06125cfa-fe0d-40c7-af74-d3f11140f2e0"},
					},
				},
			},
			expected: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := routerEquals(tc.candidate, tc.opts); got != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, got)
			}
		})
	}
}
