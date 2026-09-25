package main

import "testing"

func TestPluralize(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// Empty string
		{"", ""},

		// Default: append "s"
		{"Flavor", "Flavors"},
		{"Network", "Networks"},
		{"Server", "Servers"},
		{"Image", "Images"},
		{"Port", "Ports"},
		{"Router", "Routers"},
		{"Subnet", "Subnets"},
		{"Trunk", "Trunks"},
		{"Volume", "Volumes"},
		{"User", "Users"},
		{"Role", "Roles"},
		{"Domain", "Domains"},
		{"Endpoint", "Endpoints"},
		{"Limit", "Limits"},
		{"Region", "Regions"},
		{"Group", "Groups"},
		{"Service", "Services"},
		{"FloatingIP", "FloatingIPs"},
		{"KeyPair", "KeyPairs"},

		// Compound names (still just +s)
		{"SecurityGroup", "SecurityGroups"},
		{"ServerGroup", "ServerGroups"},
		{"VolumeType", "VolumeTypes"},
		{"ShareNetwork", "ShareNetworks"},
		{"AddressScope", "AddressScopes"},
		{"ApplicationCredential", "ApplicationCredentials"},
		{"RoleAssignment", "RoleAssignments"},
		{"RegisteredLimit", "RegisteredLimits"},
		{"RouterInterface", "RouterInterfaces"},

		// Consonant + y → ies
		{"Policy", "Policies"},
		{"policy", "policies"},
		{"QoSPolicy", "QoSPolicies"},
		{"qospolicy", "qospolicies"},

		// Vowel + y → just s (not ies)
		{"Key", "Keys"},
		{"key", "keys"},

		// Sibilant endings → es
		{"address", "addresses"},
		{"class", "classes"},
		{"bus", "buses"},
		{"match", "matches"},
		{"box", "boxes"},
		{"buzz", "buzzes"},
		{"mesh", "meshes"},

		// Lowercase defaults
		{"flavor", "flavors"},
		{"network", "networks"},
		{"securitygroup", "securitygroups"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := pluralize(tt.input)
			if got != tt.want {
				t.Errorf("pluralize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
