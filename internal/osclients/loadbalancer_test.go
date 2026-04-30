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

package osclients_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/loadbalancers"
	"github.com/k-orc/openstack-resource-controller/v2/internal/osclients"
)

const (
	// lbTokenID is a fake auth token used in tests.
	lbTokenID = "cbc36478b0bd8e67e89469c7749d4127"

	// lbID is the UUID of the test load balancer.
	lbID = "36e08a3e-a78f-4b40-a229-1e7e23eee1ab"
)

// lbSingleBody is the canned response body for a single load balancer.
const lbSingleBody = `{
	"loadbalancer": {
		"id": "36e08a3e-a78f-4b40-a229-1e7e23eee1ab",
		"project_id": "54030507-44f7-473c-9342-b4d14a95f692",
		"name": "test-lb",
		"description": "test load balancer",
		"vip_subnet_id": "9cedb85d-0759-4898-8a4b-fa5a5ea10086",
		"vip_address": "10.30.176.48",
		"vip_port_id": "2bf413c8-41a9-4477-b505-333d5cbe8b55",
		"provider": "haproxy",
		"admin_state_up": true,
		"provisioning_status": "ACTIVE",
		"operating_status": "ONLINE",
		"tags": ["test"]
	}
}`

// lbListBody is the canned response body for a load balancer list.
const lbListBody = `{
	"loadbalancers": [
		{
			"id": "c331058c-6a40-4144-948e-b9fb1df9db4b",
			"project_id": "54030507-44f7-473c-9342-b4d14a95f692",
			"name": "web-lb",
			"vip_subnet_id": "8a49c438-848f-467b-9655-ea1548708154",
			"vip_address": "10.30.176.47",
			"vip_port_id": "2a22e552-a347-44fd-b530-1f2b1b2a6735",
			"provider": "haproxy",
			"admin_state_up": true,
			"provisioning_status": "ACTIVE",
			"operating_status": "ONLINE",
			"tags": ["web"]
		},
		{
			"id": "36e08a3e-a78f-4b40-a229-1e7e23eee1ab",
			"project_id": "54030507-44f7-473c-9342-b4d14a95f692",
			"name": "test-lb",
			"vip_subnet_id": "9cedb85d-0759-4898-8a4b-fa5a5ea10086",
			"vip_address": "10.30.176.48",
			"vip_port_id": "2bf413c8-41a9-4477-b505-333d5cbe8b55",
			"provider": "haproxy",
			"admin_state_up": true,
			"provisioning_status": "ACTIVE",
			"operating_status": "ONLINE",
			"tags": ["test"]
		}
	]
}`

// lbUpdatedBody is the canned response body for an updated load balancer.
const lbUpdatedBody = `{
	"loadbalancer": {
		"id": "36e08a3e-a78f-4b40-a229-1e7e23eee1ab",
		"project_id": "54030507-44f7-473c-9342-b4d14a95f692",
		"name": "updated-lb",
		"description": "updated load balancer",
		"vip_subnet_id": "9cedb85d-0759-4898-8a4b-fa5a5ea10086",
		"vip_address": "10.30.176.48",
		"vip_port_id": "2bf413c8-41a9-4477-b505-333d5cbe8b55",
		"provider": "haproxy",
		"admin_state_up": true,
		"provisioning_status": "ACTIVE",
		"operating_status": "ONLINE",
		"tags": ["updated"]
	}
}`

// newLBTestClient creates an osclients.loadBalancerClient pointing at the given test server URL.
// It constructs the gophercloud.ServiceClient manually so that no real OpenStack endpoint
// discovery is needed.
func newLBTestClient(serverURL string) osclients.LoadBalancerClient {
	providerClient := &gophercloud.ProviderClient{TokenID: lbTokenID}
	serviceClient := &gophercloud.ServiceClient{
		ProviderClient: providerClient,
		// The loadbalancers package builds URLs as: Endpoint + "lbaas/loadbalancers[/id]"
		// We set Endpoint to serverURL+"/" so that URLs are <serverURL>/lbaas/loadbalancers...
		Endpoint: serverURL + "/",
	}
	return osclients.NewLoadBalancerClientFromServiceClient(serviceClient)
}

// --- GetLoadBalancer tests ---

func TestGetLoadBalancer_Success(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	mux.HandleFunc("/lbaas/loadbalancers/"+lbID, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.Header.Get("X-Auth-Token") != lbTokenID {
			t.Errorf("expected token %q, got %q", lbTokenID, r.Header.Get("X-Auth-Token"))
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, lbSingleBody)
	})

	client := newLBTestClient(server.URL)
	lb, err := client.GetLoadBalancer(context.Background(), lbID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lb == nil {
		t.Fatal("expected non-nil load balancer")
	}
	if lb.ID != lbID {
		t.Errorf("expected ID %q, got %q", lbID, lb.ID)
	}
	if lb.Name != "test-lb" {
		t.Errorf("expected name %q, got %q", "test-lb", lb.Name)
	}
	if lb.ProvisioningStatus != "ACTIVE" {
		t.Errorf("expected provisioning status %q, got %q", "ACTIVE", lb.ProvisioningStatus)
	}
}

func TestGetLoadBalancer_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	mux.HandleFunc("/lbaas/loadbalancers/"+lbID, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"itemNotFound": {"message": "Load balancer not found", "code": 404}}`)
	})

	client := newLBTestClient(server.URL)
	_, err := client.GetLoadBalancer(context.Background(), lbID)
	if err == nil {
		t.Fatal("expected error for 404 response, got nil")
	}
	if !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		t.Errorf("expected 404 error, got: %v", err)
	}
}

func TestGetLoadBalancer_ServerError(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	mux.HandleFunc("/lbaas/loadbalancers/"+lbID, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"error": {"message": "Internal server error", "code": 500}}`)
	})

	client := newLBTestClient(server.URL)
	_, err := client.GetLoadBalancer(context.Background(), lbID)
	if err == nil {
		t.Fatal("expected error for 500 response, got nil")
	}
}

// --- CreateLoadBalancer tests ---

func TestCreateLoadBalancer_Success(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	mux.HandleFunc("/lbaas/loadbalancers", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("X-Auth-Token") != lbTokenID {
			t.Errorf("expected token %q, got %q", lbTokenID, r.Header.Get("X-Auth-Token"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %q", r.Header.Get("Content-Type"))
		}

		// Verify the request body
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}
		lb, ok := body["loadbalancer"].(map[string]any)
		if !ok {
			t.Errorf("expected 'loadbalancer' key in request body, got: %v", body)
		}
		if lb["name"] != "test-lb" {
			t.Errorf("expected name %q in body, got %q", "test-lb", lb["name"])
		}
		if lb["vip_subnet_id"] != "9cedb85d-0759-4898-8a4b-fa5a5ea10086" {
			t.Errorf("expected vip_subnet_id in body, got %q", lb["vip_subnet_id"])
		}
		if lb["provider"] != "haproxy" {
			t.Errorf("expected provider %q in body, got %q", "haproxy", lb["provider"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, lbSingleBody)
	})

	client := newLBTestClient(server.URL)
	adminStateUp := true
	lb, err := client.CreateLoadBalancer(context.Background(), loadbalancers.CreateOpts{
		Name:         "test-lb",
		Description:  "test load balancer",
		VipSubnetID:  "9cedb85d-0759-4898-8a4b-fa5a5ea10086",
		Provider:     "haproxy",
		AdminStateUp: &adminStateUp,
		Tags:         []string{"test"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lb == nil {
		t.Fatal("expected non-nil load balancer")
	}
	if lb.ID != lbID {
		t.Errorf("expected ID %q, got %q", lbID, lb.ID)
	}
	if lb.Name != "test-lb" {
		t.Errorf("expected name %q, got %q", "test-lb", lb.Name)
	}
}

func TestCreateLoadBalancer_WithVipNetworkID(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	mux.HandleFunc("/lbaas/loadbalancers", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}
		lb, ok := body["loadbalancer"].(map[string]any)
		if !ok {
			t.Errorf("expected 'loadbalancer' key in request body, got: %v", body)
		}
		if lb["vip_network_id"] != "d0d217df-3958-4fbf-a3c2-8dad2908c709" {
			t.Errorf("expected vip_network_id in body, got %q", lb["vip_network_id"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, lbSingleBody)
	})

	client := newLBTestClient(server.URL)
	lb, err := client.CreateLoadBalancer(context.Background(), loadbalancers.CreateOpts{
		Name:         "test-lb",
		VipNetworkID: "d0d217df-3958-4fbf-a3c2-8dad2908c709",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lb == nil {
		t.Fatal("expected non-nil load balancer")
	}
}

func TestCreateLoadBalancer_WithAllParameters(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	mux.HandleFunc("/lbaas/loadbalancers", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}
		lb, ok := body["loadbalancer"].(map[string]any)
		if !ok {
			t.Errorf("expected 'loadbalancer' key in request body")
		}
		// Verify all key fields are present
		if lb["name"] != "full-lb" {
			t.Errorf("expected name %q, got %q", "full-lb", lb["name"])
		}
		if lb["description"] != "fully populated lb" {
			t.Errorf("expected description, got %q", lb["description"])
		}
		if lb["vip_port_id"] != "2bf413c8-41a9-4477-b505-333d5cbe8b55" {
			t.Errorf("expected vip_port_id in body")
		}
		if lb["flavor_id"] != "bba40eb2-ee8c-11e9-81b4-2a2ae2dbcce4" {
			t.Errorf("expected flavor_id in body")
		}
		if lb["availability_zone"] != "test-az" {
			t.Errorf("expected availability_zone in body")
		}
		if lb["project_id"] != "e3cd678b11784734bc366148aa37580e" {
			t.Errorf("expected project_id in body")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, lbSingleBody)
	})

	client := newLBTestClient(server.URL)
	adminStateUp := true
	lb, err := client.CreateLoadBalancer(context.Background(), loadbalancers.CreateOpts{
		Name:             "full-lb",
		Description:      "fully populated lb",
		VipPortID:        "2bf413c8-41a9-4477-b505-333d5cbe8b55",
		VipSubnetID:      "9cedb85d-0759-4898-8a4b-fa5a5ea10086",
		VipAddress:       "10.30.176.48",
		FlavorID:         "bba40eb2-ee8c-11e9-81b4-2a2ae2dbcce4",
		AvailabilityZone: "test-az",
		Provider:         "haproxy",
		ProjectID:        "e3cd678b11784734bc366148aa37580e",
		AdminStateUp:     &adminStateUp,
		Tags:             []string{"test", "stage"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lb == nil {
		t.Fatal("expected non-nil load balancer")
	}
}

func TestCreateLoadBalancer_Conflict(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	mux.HandleFunc("/lbaas/loadbalancers", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		fmt.Fprint(w, `{"NeutronError": {"message": "Quota exceeded for loadbalancers", "code": 409}}`)
	})

	client := newLBTestClient(server.URL)
	_, err := client.CreateLoadBalancer(context.Background(), loadbalancers.CreateOpts{
		Name:        "test-lb",
		VipSubnetID: "9cedb85d-0759-4898-8a4b-fa5a5ea10086",
	})
	if err == nil {
		t.Fatal("expected error for 409 response, got nil")
	}
	if !gophercloud.ResponseCodeIs(err, http.StatusConflict) {
		t.Errorf("expected 409 error, got: %v", err)
	}
}

// --- UpdateLoadBalancer tests ---

func TestUpdateLoadBalancer_Success(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	mux.HandleFunc("/lbaas/loadbalancers/"+lbID, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.Header.Get("X-Auth-Token") != lbTokenID {
			t.Errorf("expected token %q, got %q", lbTokenID, r.Header.Get("X-Auth-Token"))
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}
		lb, ok := body["loadbalancer"].(map[string]any)
		if !ok {
			t.Errorf("expected 'loadbalancer' key in request body, got: %v", body)
		}
		if lb["name"] != "updated-lb" {
			t.Errorf("expected name %q in body, got %q", "updated-lb", lb["name"])
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, lbUpdatedBody)
	})

	client := newLBTestClient(server.URL)
	newName := "updated-lb"
	newDesc := "updated load balancer"
	newTags := []string{"updated"}
	lb, err := client.UpdateLoadBalancer(context.Background(), lbID, loadbalancers.UpdateOpts{
		Name:        &newName,
		Description: &newDesc,
		Tags:        &newTags,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lb == nil {
		t.Fatal("expected non-nil load balancer")
	}
	if lb.Name != "updated-lb" {
		t.Errorf("expected name %q, got %q", "updated-lb", lb.Name)
	}
	if lb.Description != "updated load balancer" {
		t.Errorf("expected description %q, got %q", "updated load balancer", lb.Description)
	}
}

func TestUpdateLoadBalancer_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	mux.HandleFunc("/lbaas/loadbalancers/"+lbID, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"itemNotFound": {"message": "Load balancer not found", "code": 404}}`)
	})

	client := newLBTestClient(server.URL)
	newName := "updated-lb"
	_, err := client.UpdateLoadBalancer(context.Background(), lbID, loadbalancers.UpdateOpts{
		Name: &newName,
	})
	if err == nil {
		t.Fatal("expected error for 404 response, got nil")
	}
	if !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		t.Errorf("expected 404 error, got: %v", err)
	}
}

// --- DeleteLoadBalancer tests ---

func TestDeleteLoadBalancer_Success(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	mux.HandleFunc("/lbaas/loadbalancers/"+lbID, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.Header.Get("X-Auth-Token") != lbTokenID {
			t.Errorf("expected token %q, got %q", lbTokenID, r.Header.Get("X-Auth-Token"))
		}
		// Verify no cascade param when not set
		if r.URL.Query().Get("cascade") != "" {
			t.Errorf("expected no cascade param, got: %q", r.URL.Query().Get("cascade"))
		}
		w.WriteHeader(http.StatusNoContent)
	})

	client := newLBTestClient(server.URL)
	err := client.DeleteLoadBalancer(context.Background(), lbID, loadbalancers.DeleteOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteLoadBalancer_Cascade(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	mux.HandleFunc("/lbaas/loadbalancers/"+lbID, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		// Verify cascade query parameter is set to true
		if r.URL.Query().Get("cascade") != "true" {
			t.Errorf("expected cascade=true, got: %q", r.URL.Query().Get("cascade"))
		}
		w.WriteHeader(http.StatusNoContent)
	})

	client := newLBTestClient(server.URL)
	err := client.DeleteLoadBalancer(context.Background(), lbID, loadbalancers.DeleteOpts{
		Cascade: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteLoadBalancer_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	mux.HandleFunc("/lbaas/loadbalancers/"+lbID, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"itemNotFound": {"message": "Load balancer not found", "code": 404}}`)
	})

	client := newLBTestClient(server.URL)
	err := client.DeleteLoadBalancer(context.Background(), lbID, loadbalancers.DeleteOpts{})
	if err == nil {
		t.Fatal("expected error for 404 response, got nil")
	}
	if !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		t.Errorf("expected 404 error, got: %v", err)
	}
}

func TestDeleteLoadBalancer_Conflict(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	mux.HandleFunc("/lbaas/loadbalancers/"+lbID, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		fmt.Fprint(w, `{"NeutronError": {"message": "Load balancer is in use", "code": 409}}`)
	})

	client := newLBTestClient(server.URL)
	err := client.DeleteLoadBalancer(context.Background(), lbID, loadbalancers.DeleteOpts{})
	if err == nil {
		t.Fatal("expected error for 409 response, got nil")
	}
	if !gophercloud.ResponseCodeIs(err, http.StatusConflict) {
		t.Errorf("expected 409 error, got: %v", err)
	}
}

func TestDeleteLoadBalancer_NilOpts(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	mux.HandleFunc("/lbaas/loadbalancers/"+lbID, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	client := newLBTestClient(server.URL)
	// nil opts should also work (no query params appended)
	err := client.DeleteLoadBalancer(context.Background(), lbID, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- ListLoadBalancer tests ---

func TestListLoadBalancer_Success(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	mux.HandleFunc("/lbaas/loadbalancers", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.Header.Get("X-Auth-Token") != lbTokenID {
			t.Errorf("expected token %q, got %q", lbTokenID, r.Header.Get("X-Auth-Token"))
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, lbListBody)
	})

	client := newLBTestClient(server.URL)
	var results []*loadbalancers.LoadBalancer
	var iterErr error
	for lb, err := range client.ListLoadBalancer(context.Background(), loadbalancers.ListOpts{}) {
		if err != nil {
			iterErr = err
			break
		}
		results = append(results, lb)
	}
	if iterErr != nil {
		t.Fatalf("unexpected error during iteration: %v", iterErr)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 load balancers, got %d", len(results))
	}
	if results[0].ID != "c331058c-6a40-4144-948e-b9fb1df9db4b" {
		t.Errorf("expected first ID %q, got %q", "c331058c-6a40-4144-948e-b9fb1df9db4b", results[0].ID)
	}
	if results[1].ID != lbID {
		t.Errorf("expected second ID %q, got %q", lbID, results[1].ID)
	}
}

func TestListLoadBalancer_Empty(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	mux.HandleFunc("/lbaas/loadbalancers", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"loadbalancers": []}`)
	})

	client := newLBTestClient(server.URL)
	var results []*loadbalancers.LoadBalancer
	for lb, err := range client.ListLoadBalancer(context.Background(), loadbalancers.ListOpts{}) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		results = append(results, lb)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 load balancers, got %d", len(results))
	}
}

func TestListLoadBalancer_WithFilter(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	mux.HandleFunc("/lbaas/loadbalancers", func(w http.ResponseWriter, r *http.Request) {
		// Verify the name filter is in the query
		name := r.URL.Query().Get("name")
		if name != "test-lb" {
			t.Errorf("expected name filter %q, got %q", "test-lb", name)
		}
		w.Header().Set("Content-Type", "application/json")
		// Return only the matching lb
		fmt.Fprint(w, `{"loadbalancers": [{
			"id": "36e08a3e-a78f-4b40-a229-1e7e23eee1ab",
			"name": "test-lb",
			"provisioning_status": "ACTIVE",
			"operating_status": "ONLINE",
			"admin_state_up": true
		}]}`)
	})

	client := newLBTestClient(server.URL)
	var results []*loadbalancers.LoadBalancer
	for lb, err := range client.ListLoadBalancer(context.Background(), loadbalancers.ListOpts{Name: "test-lb"}) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		results = append(results, lb)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 load balancer, got %d", len(results))
	}
	if results[0].Name != "test-lb" {
		t.Errorf("expected name %q, got %q", "test-lb", results[0].Name)
	}
}

func TestListLoadBalancer_Pagination(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	// Return two pages: first page has next link, second has empty list
	mux.HandleFunc("/lbaas/loadbalancers", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		marker := r.URL.Query().Get("marker")
		switch marker {
		case "":
			// First page: return 2 lbs with a next link
			fmt.Fprintf(w, `{
				"loadbalancers": [
					{"id": "c331058c-6a40-4144-948e-b9fb1df9db4b", "name": "web-lb", "admin_state_up": true},
					{"id": "36e08a3e-a78f-4b40-a229-1e7e23eee1ab", "name": "test-lb", "admin_state_up": true}
				],
				"loadbalancers_links": [
					{"rel": "next", "href": "%s/lbaas/loadbalancers?marker=36e08a3e-a78f-4b40-a229-1e7e23eee1ab"}
				]
			}`, server.URL)
		case "36e08a3e-a78f-4b40-a229-1e7e23eee1ab":
			// Second page: empty
			fmt.Fprint(w, `{"loadbalancers": []}`)
		default:
			t.Errorf("unexpected marker: %q", marker)
			w.WriteHeader(http.StatusBadRequest)
		}
	})

	client := newLBTestClient(server.URL)
	var results []*loadbalancers.LoadBalancer
	for lb, err := range client.ListLoadBalancer(context.Background(), loadbalancers.ListOpts{}) {
		if err != nil {
			t.Fatalf("unexpected error during iteration: %v", err)
		}
		results = append(results, lb)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 load balancers across pages, got %d", len(results))
	}
}

func TestListLoadBalancer_IterationBreak(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	called := 0
	mux.HandleFunc("/lbaas/loadbalancers", func(w http.ResponseWriter, r *http.Request) {
		called++
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, lbListBody)
	})

	client := newLBTestClient(server.URL)
	// Only collect first result then break
	count := 0
	for lb, err := range client.ListLoadBalancer(context.Background(), loadbalancers.ListOpts{}) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		count++
		_ = lb
		break // stop after first item
	}
	if count != 1 {
		t.Errorf("expected to break after 1 item, got %d", count)
	}
}
