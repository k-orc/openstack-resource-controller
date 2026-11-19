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

package kuttl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const adminTestCase = "00-test-case.yaml"

// TestAdminTestsCanBeSkipped checks that every kuttl test using the admin
// credentials has the TestCase that lets E2E_KUTTL_SKIP_ADMIN skip it.
func TestAdminTestsCanBeSkipped(t *testing.T) {
	testDirs, err := filepath.Glob("../../internal/controllers/*/tests/*")
	if err != nil {
		t.Fatal(err)
	}

	for _, dir := range testDirs {
		files, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
		if err != nil {
			t.Fatal(err)
		}

		usesAdmin := false
		for _, f := range files {
			content, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(content), "openstack-admin") {
				usesAdmin = true
				break
			}
		}
		if !usesAdmin {
			continue
		}

		content, err := os.ReadFile(filepath.Join(dir, adminTestCase))
		if err != nil || !strings.Contains(string(content), "key: admin") {
			t.Errorf("%s uses openstack-admin but has no %s selecting on admin", dir, adminTestCase)
		}
	}
}
