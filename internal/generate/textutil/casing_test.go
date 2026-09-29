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

func TestCamelToSnake(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"Flavor", "flavor"},
		{"Network", "network"},
		{"FloatingIP", "floating_ip"},
		{"SecurityGroup", "security_group"},
		{"ServerGroup", "server_group"},
		{"VolumeType", "volume_type"},
		{"ShareNetwork", "share_network"},
		{"AddressScope", "address_scope"},
		{"ApplicationCredential", "application_credential"},
		{"RoleAssignment", "role_assignment"},
		{"RegisteredLimit", "registered_limit"},
		{"RouterInterface", "router_interface"},
		{"KeyPair", "key_pair"},
		{"QoSPolicy", "qo_s_policy"},
		{"HTTPRequest", "http_request"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := CamelToSnake(tt.input)
			if got != tt.want {
				t.Errorf("CamelToSnake(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestToCamelCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"Network", "network"},
		{"SecurityGroup", "securityGroup"},
		{"Flavor", "flavor"},
		{"Project", "project"},
		{"Router", "router"},
		{"Subnet", "subnet"},
		{"Port", "port"},
		{"Image", "image"},
		{"Server", "server"},
		{"Volume", "volume"},
		{"FloatingIP", "floatingIP"},
		{"VolumeType", "volumeType"},
		{"ServerGroup", "serverGroup"},
		{"ShareNetwork", "shareNetwork"},
		{"AddressScope", "addressScope"},
		{"ApplicationCredential", "applicationCredential"},
		{"RoleAssignment", "roleAssignment"},
		{"RegisteredLimit", "registeredLimit"},
		{"KeyPair", "keyPair"},

		// snake_case input
		{"floating_ip", "floatingIp"},
		{"security_group", "securityGroup"},

		// Already camelCase
		{"securityGroup", "securityGroup"},

		// Single character
		{"A", "a"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ToCamelCase(tt.input)
			if got != tt.want {
				t.Errorf("ToCamelCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
