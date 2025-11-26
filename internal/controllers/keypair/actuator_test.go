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

package keypair

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"testing"

	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/keypairs"
	"k8s.io/utils/ptr"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
)

var (
	errNotImplemented = errors.New("not implemented")
)

type mockKeyPairClient struct {
	keypairs []keypairs.KeyPair
}

var _ KeyPairClient = mockKeyPairClient{}

func (m mockKeyPairClient) ListKeyPairs(_ context.Context, _ keypairs.ListOptsBuilder) iter.Seq2[*keypairs.KeyPair, error] {
	return func(yield func(*keypairs.KeyPair, error) bool) {
		for i := range m.keypairs {
			if !yield(&m.keypairs[i], nil) {
				return
			}
		}
	}
}

func (m mockKeyPairClient) GetKeyPair(_ context.Context, _ string) (*keypairs.KeyPair, error) {
	return nil, errNotImplemented
}

func (m mockKeyPairClient) CreateKeyPair(_ context.Context, _ keypairs.CreateOptsBuilder) (*keypairs.KeyPair, error) {
	return nil, errNotImplemented
}

func (m mockKeyPairClient) DeleteKeyPair(_ context.Context, _ string) error {
	return errNotImplemented
}

type keypairResult struct {
	keypair *keypairs.KeyPair
	err     error
}

type checkFunc func([]keypairResult) error

func checks(fns ...checkFunc) []checkFunc { return fns }

func noError(results []keypairResult) error {
	for _, result := range results {
		if result.err != nil {
			return fmt.Errorf("unexpected error: %w", result.err)
		}
	}
	return nil
}

func findsN(wantN int) checkFunc {
	return func(results []keypairResult) error {
		found := len(results)
		if found != wantN {
			return fmt.Errorf("expected %d results, got %d", wantN, found)
		}
		return nil
	}
}

func findsName(wantName string) checkFunc {
	return func(results []keypairResult) error {
		for _, result := range results {
			if result.keypair == nil {
				continue
			}
			if result.keypair.Name == wantName {
				return nil
			}
		}
		return fmt.Errorf("did not find keypair with name %s", wantName)
	}
}

func TestListOSResourcesForImport(t *testing.T) {
	ctx := context.TODO()

	testCases := []struct {
		name     string
		keypairs []keypairs.KeyPair
		filter   orcv1alpha1.KeyPairFilter
		checks   []checkFunc
	}{
		{
			name: "Filter by name - match",
			keypairs: []keypairs.KeyPair{
				{Name: "test-keypair", Fingerprint: "aa:bb:cc"},
				{Name: "other-keypair", Fingerprint: "dd:ee:ff"},
			},
			filter: orcv1alpha1.KeyPairFilter{
				Name: ptr.To(orcv1alpha1.OpenStackName("test-keypair")),
			},
			checks: checks(
				noError,
				findsN(1),
				findsName("test-keypair"),
			),
		},
		{
			name: "Filter by name - no match",
			keypairs: []keypairs.KeyPair{
				{Name: "test-keypair", Fingerprint: "aa:bb:cc"},
				{Name: "other-keypair", Fingerprint: "dd:ee:ff"},
			},
			filter: orcv1alpha1.KeyPairFilter{
				Name: ptr.To(orcv1alpha1.OpenStackName("nonexistent")),
			},
			checks: checks(
				noError,
				findsN(0),
			),
		},
		{
			name: "No filter - returns all",
			keypairs: []keypairs.KeyPair{
				{Name: "keypair1", Fingerprint: "aa:bb:cc"},
				{Name: "keypair2", Fingerprint: "dd:ee:ff"},
				{Name: "keypair3", Fingerprint: "11:22:33"},
			},
			filter: orcv1alpha1.KeyPairFilter{},
			checks: checks(
				noError,
				findsN(3),
			),
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			actuator := keypairActuator{
				osClient: mockKeyPairClient{keypairs: tt.keypairs},
			}

			orcObject := &orcv1alpha1.KeyPair{}
			results := []keypairResult{}

			iter, _ := actuator.ListOSResourcesForImport(ctx, orcObject, tt.filter)
			for kp, err := range iter {
				results = append(results, keypairResult{keypair: kp, err: err})
			}

			for _, check := range tt.checks {
				if err := check(results); err != nil {
					t.Error(err)
				}
			}
		})
	}
}

func TestListOSResourcesForAdoption(t *testing.T) {
	ctx := context.TODO()

	testCases := []struct {
		name       string
		keypairs   []keypairs.KeyPair
		objectName string
		checks     []checkFunc
	}{
		{
			name: "Finds keypair with matching name",
			keypairs: []keypairs.KeyPair{
				{Name: "my-keypair", Fingerprint: "aa:bb:cc"},
				{Name: "other-keypair", Fingerprint: "dd:ee:ff"},
			},
			objectName: "my-keypair",
			checks: checks(
				noError,
				findsN(1),
				findsName("my-keypair"),
			),
		},
		{
			name: "No matching keypair",
			keypairs: []keypairs.KeyPair{
				{Name: "keypair1", Fingerprint: "aa:bb:cc"},
				{Name: "keypair2", Fingerprint: "dd:ee:ff"},
			},
			objectName: "nonexistent",
			checks: checks(
				noError,
				findsN(0),
			),
		},
		{
			name:       "Empty list",
			keypairs:   []keypairs.KeyPair{},
			objectName: "my-keypair",
			checks: checks(
				noError,
				findsN(0),
			),
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			actuator := keypairActuator{
				osClient: mockKeyPairClient{keypairs: tt.keypairs},
			}

			orcObject := &orcv1alpha1.KeyPair{}
			orcObject.Name = tt.objectName
			orcObject.Spec = orcv1alpha1.KeyPairSpec{
				Resource: &orcv1alpha1.KeyPairResourceSpec{},
			}

			results := []keypairResult{}
			iter, ok := actuator.ListOSResourcesForAdoption(ctx, orcObject)
			if !ok {
				t.Fatal("Expected ListOSResourcesForAdoption to return true")
			}

			for kp, err := range iter {
				results = append(results, keypairResult{keypair: kp, err: err})
			}

			for _, check := range tt.checks {
				if err := check(results); err != nil {
					t.Error(err)
				}
			}
		})
	}
}
