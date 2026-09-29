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

package textutil

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
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := Pluralize(tt.input)
			if got != tt.want {
				t.Errorf("Pluralize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
