package main

import "testing"

func TestPluralize(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// Empty string
		{"", ""},

		// All existing ORC resources (lowercase, as used by NameLower)
		{"addressscope", "addressscopes"},
		{"applicationcredential", "applicationcredentials"},
		{"domain", "domains"},
		{"endpoint", "endpoints"},
		{"flavor", "flavors"},
		{"floatingip", "floatingips"},
		{"group", "groups"},
		{"image", "images"},
		{"keypair", "keypairs"},
		{"limit", "limits"},
		{"network", "networks"},
		{"port", "ports"},
		{"project", "projects"},
		{"region", "regions"},
		{"registeredlimit", "registeredlimits"},
		{"role", "roles"},
		{"roleassignment", "roleassignments"},
		{"router", "routers"},
		{"routerinterface", "routerinterfaces"},
		{"securitygroup", "securitygroups"},
		{"server", "servers"},
		{"servergroup", "servergroups"},
		{"service", "services"},
		{"sharenetwork", "sharenetworks"},
		{"subnet", "subnets"},
		{"trunk", "trunks"},
		{"user", "users"},
		{"volume", "volumes"},
		{"volumetype", "volumetypes"},

		// Consonant + y → ies
		{"policy", "policies"},
		{"qospolicy", "qospolicies"},

		// Vowel + y → just s
		{"key", "keys"},

		// Sibilant endings → es
		{"address", "addresses"},
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
