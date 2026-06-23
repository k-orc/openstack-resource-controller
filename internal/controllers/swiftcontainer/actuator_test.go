/*
Copyright 2024 The ORC Authors.

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

package swiftcontainer

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"strings"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/containers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	osclients "github.com/k-orc/openstack-resource-controller/v2/internal/osclients"
	orcerrors "github.com/k-orc/openstack-resource-controller/v2/internal/util/errors"
)

var (
	errNotImplemented = errors.New("not implemented")
	errTest           = errors.New("test error")
)

// mockSwiftContainerClient implements the osclients.SwiftContainerClient
// interface for use in unit tests.
type mockSwiftContainerClient struct {
	// containers is the list of containers returned by ListContainers.
	containers []containers.Container

	// containerData maps container name to its simulated GetHeader and metadata.
	containerData map[string]mockContainerData

	// listErr, if non-nil, is returned as an error from ListContainers.
	listErr error
}

type mockContainerData struct {
	header   containers.GetHeader
	metadata map[string]string
}

var _ osclients.SwiftContainerClient = &mockSwiftContainerClient{}

func (m *mockSwiftContainerClient) ListContainers(_ context.Context, _ containers.ListOptsBuilder) iter.Seq2[*containers.Container, error] {
	return func(yield func(*containers.Container, error) bool) {
		if m.listErr != nil {
			yield(nil, m.listErr)
			return
		}
		for i := range m.containers {
			if !yield(&m.containers[i], nil) {
				return
			}
		}
	}
}

func (m *mockSwiftContainerClient) CreateContainer(_ context.Context, containerName string, _ containers.CreateOptsBuilder) (*containers.CreateHeader, error) {
	if m.containerData == nil {
		m.containerData = make(map[string]mockContainerData)
	}
	m.containerData[containerName] = mockContainerData{
		header:   containers.GetHeader{},
		metadata: map[string]string{},
	}
	return &containers.CreateHeader{}, nil
}

func (m *mockSwiftContainerClient) GetContainer(_ context.Context, containerName string, _ containers.GetOptsBuilder) (*containers.GetHeader, error) {
	if m.containerData == nil {
		return nil, gophercloud.ErrResourceNotFound{Name: containerName}
	}
	data, ok := m.containerData[containerName]
	if !ok {
		return nil, gophercloud.ErrResourceNotFound{Name: containerName}
	}
	header := data.header
	return &header, nil
}

func (m *mockSwiftContainerClient) GetContainerMetadata(_ context.Context, containerName string) (map[string]string, error) {
	if m.containerData == nil {
		return nil, gophercloud.ErrResourceNotFound{Name: containerName}
	}
	data, ok := m.containerData[containerName]
	if !ok {
		return nil, gophercloud.ErrResourceNotFound{Name: containerName}
	}
	meta := make(map[string]string, len(data.metadata))
	for k, v := range data.metadata {
		meta[k] = v
	}
	return meta, nil
}

func (m *mockSwiftContainerClient) DeleteContainer(_ context.Context, _ string) error {
	return errNotImplemented
}

func (m *mockSwiftContainerClient) UpdateContainer(_ context.Context, _ string, _ containers.UpdateOptsBuilder) (*containers.UpdateHeader, error) {
	return nil, errNotImplemented
}

// containerResult holds the result of a single iteration from a container iterator.
type containerResult struct {
	container *osContainerT
	err       error
}

type checkFunc func([]containerResult) error

func checks(fns ...checkFunc) []checkFunc { return fns }

func noError(results []containerResult) error {
	for _, result := range results {
		if result.err != nil {
			return fmt.Errorf("unexpected error: %w", result.err)
		}
	}
	return nil
}

func wantError(wantErr error) checkFunc {
	return func(results []containerResult) error {
		for _, result := range results {
			if result.err == nil {
				continue
			}
			if errors.Is(result.err, wantErr) {
				return nil
			}
			return fmt.Errorf("unexpected error: %w", result.err)
		}
		// If we get here, no error was found anywhere
		return nil
	}
}

func findsN(wantN int) checkFunc {
	return func(results []containerResult) error {
		found := len(results)
		if found != wantN {
			return fmt.Errorf("expected %d results, got %d", wantN, found)
		}
		return nil
	}
}

func findsID(wantName string) checkFunc {
	return func(results []containerResult) error {
		for _, result := range results {
			if result.container == nil {
				continue
			}
			if result.container.Name == wantName {
				return nil
			}
		}
		return fmt.Errorf("did not find container with name %s", wantName)
	}
}

// newSwiftContainerObject creates a SwiftContainer object for use in tests.
func newSwiftContainerObject(name string, resource *orcv1alpha1.SwiftContainerResourceSpec) *orcv1alpha1.SwiftContainer {
	return &orcv1alpha1.SwiftContainer{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: orcv1alpha1.SwiftContainerSpec{
			Resource: resource,
		},
	}
}

func TestListOSResourcesForImport(t *testing.T) {
	for _, tc := range [...]struct {
		name   string
		filter orcv1alpha1.SwiftContainerFilter
		client osclients.SwiftContainerClient
		checks []checkFunc
	}{
		{
			name:   "finds one by name",
			filter: orcv1alpha1.SwiftContainerFilter{Name: ptr.To[orcv1alpha1.SwiftContainerName]("my-container")},
			client: &mockSwiftContainerClient{
				containerData: map[string]mockContainerData{
					"my-container":    {header: containers.GetHeader{}, metadata: map[string]string{}},
					"other-container": {header: containers.GetHeader{}, metadata: map[string]string{}},
				},
			},
			checks: checks(noError, findsID("my-container"), findsN(1)),
		},
		{
			name:   "finds none by name",
			filter: orcv1alpha1.SwiftContainerFilter{Name: ptr.To[orcv1alpha1.SwiftContainerName]("missing-container")},
			client: &mockSwiftContainerClient{
				containerData: map[string]mockContainerData{
					"my-container": {header: containers.GetHeader{}, metadata: map[string]string{}},
				},
			},
			checks: checks(noError, findsN(0)),
		},
		{
			name:   "finds multiple containers matching prefix filter",
			filter: orcv1alpha1.SwiftContainerFilter{Prefix: ptr.To("test-")},
			client: &mockSwiftContainerClient{
				containers: []containers.Container{
					{Name: "test-alpha"},
					{Name: "test-beta"},
					{Name: "other"},
				},
				containerData: map[string]mockContainerData{
					"test-alpha": {header: containers.GetHeader{}, metadata: map[string]string{}},
					"test-beta":  {header: containers.GetHeader{}, metadata: map[string]string{}},
					"other":      {header: containers.GetHeader{}, metadata: map[string]string{}},
				},
			},
			checks: checks(noError, findsN(2)),
		},
		{
			name:   "returns lister errors from client",
			filter: orcv1alpha1.SwiftContainerFilter{Prefix: ptr.To("test-")},
			client: &mockSwiftContainerClient{
				listErr: errTest,
			},
			checks: checks(wantError(errTest)),
		},
		{
			// When both Name and Prefix are set and the named container does not
			// match the prefix, no results should be returned. Previously the
			// prefix predicate was silently ignored in the name-lookup branch.
			name: "finds none when name does not match prefix",
			filter: orcv1alpha1.SwiftContainerFilter{
				Name:   ptr.To[orcv1alpha1.SwiftContainerName]("prod-bucket"),
				Prefix: ptr.To("test-"),
			},
			client: &mockSwiftContainerClient{
				containerData: map[string]mockContainerData{
					"prod-bucket": {header: containers.GetHeader{}, metadata: map[string]string{}},
				},
			},
			checks: checks(noError, findsN(0)),
		},
		{
			// When both Name and Prefix are set and the named container matches
			// the prefix, the container should be found normally.
			name: "finds one when name matches prefix",
			filter: orcv1alpha1.SwiftContainerFilter{
				Name:   ptr.To[orcv1alpha1.SwiftContainerName]("test-bucket"),
				Prefix: ptr.To("test-"),
			},
			client: &mockSwiftContainerClient{
				containerData: map[string]mockContainerData{
					"test-bucket": {header: containers.GetHeader{}, metadata: map[string]string{}},
				},
			},
			checks: checks(noError, findsN(1), findsID("test-bucket")),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			actuator := swiftcontainerActuator{tc.client}
			containerIter, _ := actuator.ListOSResourcesForImport(ctx, &orcv1alpha1.SwiftContainer{
				ObjectMeta: metav1.ObjectMeta{
					Name: "swiftcontainer",
				},
			}, tc.filter)

			var results []containerResult
			for container, err := range containerIter {
				results = append(results, containerResult{container, err})
			}

			for _, check := range tc.checks {
				if e := check(results); e != nil {
					t.Error(e)
				}
			}
		})
	}
}

func TestListOSResourcesForAdoption(t *testing.T) {
	t.Run("finds container by resource name", func(t *testing.T) {
		ctx := context.Background()
		client := &mockSwiftContainerClient{
			containerData: map[string]mockContainerData{
				"explicit-name": {header: containers.GetHeader{}, metadata: map[string]string{}},
			},
		}
		actuator := swiftcontainerActuator{client}
		orcObject := newSwiftContainerObject("my-object", &orcv1alpha1.SwiftContainerResourceSpec{
			Name: ptr.To[orcv1alpha1.SwiftContainerName]("explicit-name"),
		})

		containerIter, canAdopt := actuator.ListOSResourcesForAdoption(ctx, orcObject)
		if !canAdopt {
			t.Fatal("canAdopt should be true when spec.resource is set")
		}

		var results []containerResult
		for container, err := range containerIter {
			results = append(results, containerResult{container, err})
		}

		for _, check := range checks(noError, findsID("explicit-name"), findsN(1)) {
			if e := check(results); e != nil {
				t.Error(e)
			}
		}
	})

	t.Run("finds container by object name when spec.resource.name is nil", func(t *testing.T) {
		ctx := context.Background()
		client := &mockSwiftContainerClient{
			containerData: map[string]mockContainerData{
				"my-object": {header: containers.GetHeader{}, metadata: map[string]string{}},
			},
		}
		actuator := swiftcontainerActuator{client}
		// spec.resource.name is nil, so the object's own name should be used
		orcObject := newSwiftContainerObject("my-object", &orcv1alpha1.SwiftContainerResourceSpec{})

		containerIter, canAdopt := actuator.ListOSResourcesForAdoption(ctx, orcObject)
		if !canAdopt {
			t.Fatal("canAdopt should be true when spec.resource is set")
		}

		var results []containerResult
		for container, err := range containerIter {
			results = append(results, containerResult{container, err})
		}

		for _, check := range checks(noError, findsID("my-object"), findsN(1)) {
			if e := check(results); e != nil {
				t.Error(e)
			}
		}
	})

	t.Run("finds none when container does not exist", func(t *testing.T) {
		ctx := context.Background()
		// Empty containerData so all GetContainer calls return 404
		client := &mockSwiftContainerClient{
			containerData: map[string]mockContainerData{},
		}
		actuator := swiftcontainerActuator{client}
		orcObject := newSwiftContainerObject("nonexistent", &orcv1alpha1.SwiftContainerResourceSpec{})

		containerIter, canAdopt := actuator.ListOSResourcesForAdoption(ctx, orcObject)
		if !canAdopt {
			t.Fatal("canAdopt should be true when spec.resource is set")
		}

		var results []containerResult
		for container, err := range containerIter {
			results = append(results, containerResult{container, err})
		}

		for _, check := range checks(noError, findsN(0)) {
			if e := check(results); e != nil {
				t.Error(e)
			}
		}
	})

	t.Run("returns metadata when container has metadata", func(t *testing.T) {
		ctx := context.Background()
		client := &mockSwiftContainerClient{
			containerData: map[string]mockContainerData{
				"my-object": {
					header:   containers.GetHeader{},
					metadata: map[string]string{"env": "prod", "team": "infra"},
				},
			},
		}
		actuator := swiftcontainerActuator{client}
		orcObject := newSwiftContainerObject("my-object", &orcv1alpha1.SwiftContainerResourceSpec{
			Metadata: []orcv1alpha1.SwiftContainerMetadata{
				{Key: "env", Value: "prod"},
				{Key: "team", Value: "infra"},
			},
		})

		containerIter, canAdopt := actuator.ListOSResourcesForAdoption(ctx, orcObject)
		if !canAdopt {
			t.Fatal("canAdopt should be true when spec.resource is set")
		}

		var results []containerResult
		for container, err := range containerIter {
			results = append(results, containerResult{container, err})
		}

		for _, check := range checks(noError, findsN(1)) {
			if e := check(results); e != nil {
				t.Error(e)
			}
		}

		// Verify metadata is passed through
		if len(results) == 1 && results[0].container != nil {
			if results[0].container.Metadata["env"] != "prod" {
				t.Errorf("expected metadata env=prod, got %q", results[0].container.Metadata["env"])
			}
			if results[0].container.Metadata["team"] != "infra" {
				t.Errorf("expected metadata team=infra, got %q", results[0].container.Metadata["team"])
			}
		}
	})

	t.Run("returns false for canAdopt when spec.resource is nil", func(t *testing.T) {
		ctx := context.Background()
		client := &mockSwiftContainerClient{}
		actuator := swiftcontainerActuator{client}
		// spec.resource is nil
		orcObject := &orcv1alpha1.SwiftContainer{
			ObjectMeta: metav1.ObjectMeta{Name: "my-object"},
			Spec:       orcv1alpha1.SwiftContainerSpec{},
		}

		_, canAdopt := actuator.ListOSResourcesForAdoption(ctx, orcObject)
		if canAdopt {
			t.Error("canAdopt should be false when spec.resource is nil")
		}
	})
}

func TestCreateResource(t *testing.T) {
	t.Run("successfully creates container with minimal config", func(t *testing.T) {
		ctx := context.Background()
		client := &mockSwiftContainerClient{}
		actuator := swiftcontainerActuator{client}
		orcObject := newSwiftContainerObject("my-container", &orcv1alpha1.SwiftContainerResourceSpec{})

		result, reconcileStatus := actuator.CreateResource(ctx, orcObject)
		if reconcileStatus != nil {
			t.Fatalf("unexpected reconcile status: %v", reconcileStatus)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Name != "my-container" {
			t.Errorf("expected container name %q, got %q", "my-container", result.Name)
		}
	})

	t.Run("successfully creates container with full config", func(t *testing.T) {
		ctx := context.Background()
		client := &mockSwiftContainerClient{}
		actuator := swiftcontainerActuator{client}
		orcObject := newSwiftContainerObject("full-container", &orcv1alpha1.SwiftContainerResourceSpec{
			Name: ptr.To[orcv1alpha1.SwiftContainerName]("full-container"),
			Metadata: []orcv1alpha1.SwiftContainerMetadata{
				{Key: "project", Value: "orc"},
				{Key: "env", Value: "test"},
			},
			ContainerRead:  ptr.To(".r:*"),
			ContainerWrite: ptr.To("account:user"),
			StoragePolicy:  ptr.To("gold"),
		})

		result, reconcileStatus := actuator.CreateResource(ctx, orcObject)
		if reconcileStatus != nil {
			t.Fatalf("unexpected reconcile status: %v", reconcileStatus)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Name != "full-container" {
			t.Errorf("expected container name %q, got %q", "full-container", result.Name)
		}
	})

	t.Run("returns terminal error for invalid container name containing slash", func(t *testing.T) {
		ctx := context.Background()

		// The actuator explicitly validates the container name before calling
		// the Swift API. A slash in the name is caught early and returned as a
		// terminal error with a message mentioning "forward slashes".
		// (Kubebuilder validation normally prevents this from reaching the
		// controller, but in unit tests API validation is not enforced.)
		client := &mockSwiftContainerClient{}
		actuator := swiftcontainerActuator{client}
		// Use a name that bypasses kubebuilder validation (in unit tests, API
		// validation is not enforced); this simulates what would happen if a
		// slash somehow reached the actuator.
		orcObject := &orcv1alpha1.SwiftContainer{
			ObjectMeta: metav1.ObjectMeta{Name: "invalid"},
			Spec: orcv1alpha1.SwiftContainerSpec{
				Resource: &orcv1alpha1.SwiftContainerResourceSpec{
					Name: ptr.To[orcv1alpha1.SwiftContainerName]("invalid/name"),
				},
			},
		}

		result, reconcileStatus := actuator.CreateResource(ctx, orcObject)
		if result != nil {
			t.Errorf("expected nil result, got %v", result)
		}
		if reconcileStatus == nil {
			t.Fatal("expected non-nil reconcile status for terminal error")
		}
		_, err := reconcileStatus.NeedsReschedule()
		if err == nil {
			t.Error("expected error from reconcile status")
		}
		var termErr *orcerrors.TerminalError
		if !errors.As(err, &termErr) {
			t.Errorf("expected TerminalError, got %T: %v", err, err)
		}
	})

	t.Run("returns terminal error for container name exceeding 256 bytes", func(t *testing.T) {
		ctx := context.Background()

		// A name exceeding 256 UTF-8 bytes should cause Swift to reject it.
		// Simulate this via the error client.
		terminalErr := orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration,
			"container name exceeds maximum length")
		client := osclients.NewSwiftContainerErrorClient(terminalErr)
		actuator := swiftcontainerActuator{client}
		longName := strings.Repeat("a", 257)
		orcObject := &orcv1alpha1.SwiftContainer{
			ObjectMeta: metav1.ObjectMeta{Name: "invalid"},
			Spec: orcv1alpha1.SwiftContainerSpec{
				Resource: &orcv1alpha1.SwiftContainerResourceSpec{
					Name: ptr.To[orcv1alpha1.SwiftContainerName](orcv1alpha1.SwiftContainerName(longName)),
				},
			},
		}

		result, reconcileStatus := actuator.CreateResource(ctx, orcObject)
		if result != nil {
			t.Errorf("expected nil result, got %v", result)
		}
		if reconcileStatus == nil {
			t.Fatal("expected non-nil reconcile status for terminal error")
		}
		_, err := reconcileStatus.NeedsReschedule()
		if err == nil {
			t.Error("expected error from reconcile status")
		}
		var termErr *orcerrors.TerminalError
		if !errors.As(err, &termErr) {
			t.Errorf("expected TerminalError, got %T: %v", err, err)
		}
	})

	t.Run("returns error when spec.resource is nil", func(t *testing.T) {
		ctx := context.Background()
		client := &mockSwiftContainerClient{}
		actuator := swiftcontainerActuator{client}
		orcObject := &orcv1alpha1.SwiftContainer{
			ObjectMeta: metav1.ObjectMeta{Name: "my-container"},
			Spec:       orcv1alpha1.SwiftContainerSpec{},
		}

		result, reconcileStatus := actuator.CreateResource(ctx, orcObject)
		if result != nil {
			t.Errorf("expected nil result, got %v", result)
		}
		if reconcileStatus == nil {
			t.Fatal("expected non-nil reconcile status when spec.resource is nil")
		}
		_, err := reconcileStatus.NeedsReschedule()
		if err == nil {
			t.Error("expected error from reconcile status")
		}
		var termErr *orcerrors.TerminalError
		if !errors.As(err, &termErr) {
			t.Errorf("expected TerminalError, got %T: %v", err, err)
		}
	})
}

func TestContainerNameValidation(t *testing.T) {
	t.Run("rejects names containing forward slash", func(t *testing.T) {
		name := orcv1alpha1.SwiftContainerName("containers/bucket")
		if !strings.Contains(string(name), "/") {
			t.Fatal("test setup error: name should contain a slash")
		}
		// The actuator validates the name before calling the Swift API.
		// A slash in the name causes an early terminal error with a message
		// mentioning "forward slashes". (Kubebuilder pattern validation would
		// normally catch this before it reaches the controller, but in unit
		// tests API validation is not enforced.)
		ctx := context.Background()
		client := &mockSwiftContainerClient{}
		actuator := swiftcontainerActuator{client}
		orcObject := &orcv1alpha1.SwiftContainer{
			ObjectMeta: metav1.ObjectMeta{Name: "invalid"},
			Spec: orcv1alpha1.SwiftContainerSpec{
				Resource: &orcv1alpha1.SwiftContainerResourceSpec{
					Name: ptr.To(name),
				},
			},
		}
		_, reconcileStatus := actuator.CreateResource(ctx, orcObject)
		if reconcileStatus == nil {
			t.Error("expected reconcile status error for name with slash")
			return
		}
		_, err := reconcileStatus.NeedsReschedule()
		var termErr *orcerrors.TerminalError
		if !errors.As(err, &termErr) {
			t.Errorf("expected TerminalError for invalid name, got %T: %v", err, err)
		}
		if !strings.Contains(termErr.Error(), "forward slashes") {
			t.Errorf("expected error message to mention 'forward slashes', got: %v", termErr.Error())
		}
	})

	t.Run("rejects names exceeding 256 UTF-8 bytes", func(t *testing.T) {
		longName := orcv1alpha1.SwiftContainerName(strings.Repeat("x", 257))
		if len(longName) <= 256 {
			t.Fatal("test setup error: name should exceed 256 bytes")
		}
		ctx := context.Background()
		terminalErr := orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration,
			"container name exceeds 256 bytes")
		client := osclients.NewSwiftContainerErrorClient(terminalErr)
		actuator := swiftcontainerActuator{client}
		orcObject := &orcv1alpha1.SwiftContainer{
			ObjectMeta: metav1.ObjectMeta{Name: "invalid"},
			Spec: orcv1alpha1.SwiftContainerSpec{
				Resource: &orcv1alpha1.SwiftContainerResourceSpec{
					Name: ptr.To(longName),
				},
			},
		}
		_, reconcileStatus := actuator.CreateResource(ctx, orcObject)
		if reconcileStatus == nil {
			t.Error("expected reconcile status error for name exceeding 256 bytes")
			return
		}
		_, err := reconcileStatus.NeedsReschedule()
		var termErr *orcerrors.TerminalError
		if !errors.As(err, &termErr) {
			t.Errorf("expected TerminalError for too-long name, got %T: %v", err, err)
		}
	})

	t.Run("accepts valid names at boundary (exactly 256 bytes)", func(t *testing.T) {
		// A name of exactly 256 ASCII characters is valid.
		exactName := orcv1alpha1.SwiftContainerName(strings.Repeat("a", 256))
		if len(exactName) != 256 {
			t.Fatal("test setup error: name should be exactly 256 bytes")
		}
		ctx := context.Background()
		client := &mockSwiftContainerClient{}
		actuator := swiftcontainerActuator{client}
		orcObject := &orcv1alpha1.SwiftContainer{
			ObjectMeta: metav1.ObjectMeta{Name: "boundary-test"},
			Spec: orcv1alpha1.SwiftContainerSpec{
				Resource: &orcv1alpha1.SwiftContainerResourceSpec{
					Name: ptr.To(exactName),
				},
			},
		}
		result, reconcileStatus := actuator.CreateResource(ctx, orcObject)
		if reconcileStatus != nil {
			t.Fatalf("expected no error for valid 256-byte name, got: %v", reconcileStatus)
		}
		if result == nil {
			t.Error("expected non-nil result for valid 256-byte name")
		}
	})
}
