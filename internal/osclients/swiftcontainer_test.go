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

package osclients_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/containers"
	"github.com/gophercloud/utils/v2/openstack/clientconfig"

	"github.com/k-orc/openstack-resource-controller/v2/internal/osclients"
)

const fakeToken = "test-token-abc123"

// newFakeSwiftServer creates an httptest server and a corresponding SwiftContainerClient
// pointing to it. The caller is responsible for closing the server.
func newFakeSwiftServer(t *testing.T, mux *http.ServeMux) (*httptest.Server, osclients.SwiftContainerClient) {
	t.Helper()

	server := httptest.NewServer(mux)

	// Construct a gophercloud ServiceClient directly, pointing to the test server.
	// This avoids needing real OpenStack credentials.
	serviceClient := &gophercloud.ServiceClient{
		ProviderClient: &gophercloud.ProviderClient{
			TokenID: fakeToken,
		},
		Endpoint: server.URL + "/",
	}

	client := osclients.NewSwiftContainerClientFromServiceClient(serviceClient)
	return server, client
}

// TestNewSwiftContainerClient_Success verifies the constructor creates a valid client
// with a correct endpoint when given a valid provider.
func TestNewSwiftContainerClient_Success(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	// Provide a fake endpoint locator that returns the test server URL.
	providerClient := &gophercloud.ProviderClient{
		TokenID: fakeToken,
		EndpointLocator: func(eo gophercloud.EndpointOpts) (string, error) {
			return server.URL + "/", nil
		},
	}

	opts := &clientconfig.ClientOpts{}
	client, err := osclients.NewSwiftContainerClient(providerClient, opts)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

// TestNewSwiftContainerClient_InvalidEndpoint verifies that an error is returned
// when the endpoint locator returns an error (simulating an invalid or missing endpoint).
func TestNewSwiftContainerClient_InvalidEndpoint(t *testing.T) {
	endpointErr := errors.New("endpoint not found")

	providerClient := &gophercloud.ProviderClient{
		TokenID: fakeToken,
		EndpointLocator: func(eo gophercloud.EndpointOpts) (string, error) {
			return "", endpointErr
		},
	}

	opts := &clientconfig.ClientOpts{}
	client, err := osclients.NewSwiftContainerClient(providerClient, opts)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if client != nil {
		t.Fatalf("expected nil client on error, got: %v", client)
	}
}

// TestListContainers_Pagination verifies that the iterator correctly yields containers
// across paginated responses.
func TestListContainers_Pagination(t *testing.T) {
	mux := http.NewServeMux()

	// The Swift list containers API is paginated using the marker parameter.
	// First page returns two containers; second (with marker) returns one; final returns empty.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if got := r.Header.Get("X-Auth-Token"); got != fakeToken {
			t.Errorf("expected token %q, got %q", fakeToken, got)
		}

		if err := r.ParseForm(); err != nil {
			t.Errorf("failed to parse form: %v", err)
		}
		marker := r.Form.Get("marker")

		w.Header().Set("Content-Type", "application/json")
		switch marker {
		case "":
			_, _ = fmt.Fprint(w, `[{"name":"container-a","count":0,"bytes":0},{"name":"container-b","count":3,"bytes":1024}]`)
		case "container-b":
			_, _ = fmt.Fprint(w, `[]`)
		default:
			t.Errorf("unexpected marker: %q", marker)
			w.WriteHeader(http.StatusBadRequest)
		}
	})

	server, client := newFakeSwiftServer(t, mux)
	defer server.Close()

	ctx := context.Background()
	var got []string
	for container, err := range client.ListContainers(ctx, containers.ListOpts{}) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got = append(got, container.Name)
	}

	want := []string{"container-a", "container-b"}
	if len(got) != len(want) {
		t.Fatalf("expected %d containers, got %d: %v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("container[%d]: expected %q, got %q", i, want[i], got[i])
		}
	}
}

// TestListContainers_Error verifies that errors from page extraction are propagated
// through the iterator correctly.
//
// When a Swift server returns a 200 response with a non-JSON content type, the
// page body cannot be parsed as a container list, so ExtractInfo returns an error.
// The iterator must propagate this error to callers rather than silently stopping.
func TestListContainers_Error(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Return 200 with text/plain - gophercloud accepts it as a valid page
		// but ExtractInfo fails to unmarshal the body as []containers.Container,
		// propagating the error through yieldPage.
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "not-valid-json-container-data")
	})

	server, client := newFakeSwiftServer(t, mux)
	defer server.Close()

	ctx := context.Background()
	var gotErr error
	for _, err := range client.ListContainers(ctx, containers.ListOpts{}) {
		if err != nil {
			gotErr = err
			break
		}
	}

	if gotErr == nil {
		t.Fatal("expected error from iterator, got nil")
	}
}

// TestCreateContainer_Success verifies that CreateContainer sends the correct
// headers, including ACLs and custom metadata.
func TestCreateContainer_Success(t *testing.T) {
	const containerName = "my-container"

	mux := http.NewServeMux()
	mux.HandleFunc("/"+containerName, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}

		// Verify ACL headers.
		if got := r.Header.Get("X-Container-Read"); got != ".r:*" {
			t.Errorf("X-Container-Read: expected %q, got %q", ".r:*", got)
		}
		if got := r.Header.Get("X-Container-Write"); got != "myproject:myuser" {
			t.Errorf("X-Container-Write: expected %q, got %q", "myproject:myuser", got)
		}

		// Verify that custom metadata is sent with the correct prefix.
		if got := r.Header.Get("X-Container-Meta-Environment"); got != "production" {
			t.Errorf("X-Container-Meta-Environment: expected %q, got %q", "production", got)
		}

		w.Header().Set("Content-Length", "0")
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		w.Header().Set("X-Trans-Id", "tx-test-id")
		w.WriteHeader(http.StatusNoContent)
	})

	server, client := newFakeSwiftServer(t, mux)
	defer server.Close()

	ctx := context.Background()
	opts := containers.CreateOpts{
		ContainerRead:  ".r:*",
		ContainerWrite: "myproject:myuser",
		Metadata: map[string]string{
			"Environment": "production",
		},
	}
	header, err := client.CreateContainer(ctx, containerName, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if header == nil {
		t.Fatal("expected non-nil header")
	}
}

// TestGetContainer_Success verifies that GetContainer returns correct header information.
func TestGetContainer_Success(t *testing.T) {
	const containerName = "my-container"

	mux := http.NewServeMux()
	mux.HandleFunc("/"+containerName, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			t.Errorf("expected HEAD, got %s", r.Method)
		}

		w.Header().Set("X-Container-Bytes-Used", "2048")
		w.Header().Set("X-Container-Object-Count", "10")
		w.Header().Set("X-Container-Read", ".r:*")
		w.Header().Set("X-Container-Write", "admin")
		w.Header().Set("X-Storage-Policy", "gold")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("X-Trans-Id", "tx-get-test")
		w.WriteHeader(http.StatusNoContent)
	})

	server, client := newFakeSwiftServer(t, mux)
	defer server.Close()

	ctx := context.Background()
	header, err := client.GetContainer(ctx, containerName, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if header == nil {
		t.Fatal("expected non-nil header")
	}
	if header.BytesUsed != 2048 {
		t.Errorf("BytesUsed: expected 2048, got %d", header.BytesUsed)
	}
	if header.ObjectCount != 10 {
		t.Errorf("ObjectCount: expected 10, got %d", header.ObjectCount)
	}
	if header.StoragePolicy != "gold" {
		t.Errorf("StoragePolicy: expected %q, got %q", "gold", header.StoragePolicy)
	}
}

// TestGetContainerMetadata_Success verifies that metadata extraction works correctly,
// stripping the "X-Container-Meta-" prefix from header keys.
func TestGetContainerMetadata_Success(t *testing.T) {
	const containerName = "my-container"

	mux := http.NewServeMux()
	mux.HandleFunc("/"+containerName, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			t.Errorf("expected HEAD, got %s", r.Method)
		}

		// Gophercloud's ExtractMetadata strips the "X-Container-Meta-" prefix
		// and lowercases the key.
		w.Header().Set("X-Container-Meta-Env", "staging")
		w.Header().Set("X-Container-Meta-Owner", "team-infra")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusNoContent)
	})

	server, client := newFakeSwiftServer(t, mux)
	defer server.Close()

	ctx := context.Background()
	meta, err := client.GetContainerMetadata(ctx, containerName)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta == nil {
		t.Fatal("expected non-nil metadata map")
	}

	// Gophercloud strips the "X-Container-Meta-" prefix. Go's net/http
	// canonicalises header keys, so "X-Container-Meta-Env" becomes key "Env".
	if got := meta["Env"]; got != "staging" {
		t.Errorf("metadata[Env]: expected %q, got %q", "staging", got)
	}
	if got := meta["Owner"]; got != "team-infra" {
		t.Errorf("metadata[Owner]: expected %q, got %q", "team-infra", got)
	}
}

// TestUpdateContainer_ACLs verifies that ACL updates are sent as the correct headers.
func TestUpdateContainer_ACLs(t *testing.T) {
	const containerName = "my-container"
	const newReadACL = ".r:*,.rlistings"
	const newWriteACL = "myproject:*"

	mux := http.NewServeMux()
	mux.HandleFunc("/"+containerName, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		if got := r.Header.Get("X-Container-Read"); got != newReadACL {
			t.Errorf("X-Container-Read: expected %q, got %q", newReadACL, got)
		}
		if got := r.Header.Get("X-Container-Write"); got != newWriteACL {
			t.Errorf("X-Container-Write: expected %q, got %q", newWriteACL, got)
		}

		w.Header().Set("Content-Length", "0")
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		w.Header().Set("X-Trans-Id", "tx-update-test")
		w.WriteHeader(http.StatusNoContent)
	})

	server, client := newFakeSwiftServer(t, mux)
	defer server.Close()

	ctx := context.Background()
	readACL := newReadACL
	writeACL := newWriteACL
	opts := containers.UpdateOpts{
		ContainerRead:  &readACL,
		ContainerWrite: &writeACL,
	}
	header, err := client.UpdateContainer(ctx, containerName, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if header == nil {
		t.Fatal("expected non-nil header")
	}
}

// TestDeleteContainer_Success verifies that the delete call succeeds and sends
// a DELETE request to the correct path.
func TestDeleteContainer_Success(t *testing.T) {
	const containerName = "my-container"

	mux := http.NewServeMux()
	mux.HandleFunc("/"+containerName, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	server, client := newFakeSwiftServer(t, mux)
	defer server.Close()

	ctx := context.Background()
	err := client.DeleteContainer(ctx, containerName)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestSwiftContainerErrorClient verifies that the error client returns the
// configured error for every method.
func TestSwiftContainerErrorClient(t *testing.T) {
	testErr := errors.New("test configured error")
	client := osclients.NewSwiftContainerErrorClient(testErr)
	ctx := context.Background()

	t.Run("ListContainers", func(t *testing.T) {
		var gotErr error
		for _, err := range client.ListContainers(ctx, nil) {
			gotErr = err
			break
		}
		if !errors.Is(gotErr, testErr) {
			t.Errorf("expected %v, got %v", testErr, gotErr)
		}
	})

	t.Run("CreateContainer", func(t *testing.T) {
		_, err := client.CreateContainer(ctx, "any", nil)
		if !errors.Is(err, testErr) {
			t.Errorf("expected %v, got %v", testErr, err)
		}
	})

	t.Run("GetContainer", func(t *testing.T) {
		_, err := client.GetContainer(ctx, "any", nil)
		if !errors.Is(err, testErr) {
			t.Errorf("expected %v, got %v", testErr, err)
		}
	})

	t.Run("GetContainerMetadata", func(t *testing.T) {
		_, err := client.GetContainerMetadata(ctx, "any")
		if !errors.Is(err, testErr) {
			t.Errorf("expected %v, got %v", testErr, err)
		}
	})

	t.Run("DeleteContainer", func(t *testing.T) {
		err := client.DeleteContainer(ctx, "any")
		if !errors.Is(err, testErr) {
			t.Errorf("expected %v, got %v", testErr, err)
		}
	})

	t.Run("UpdateContainer", func(t *testing.T) {
		_, err := client.UpdateContainer(ctx, "any", nil)
		if !errors.Is(err, testErr) {
			t.Errorf("expected %v, got %v", testErr, err)
		}
	})
}
