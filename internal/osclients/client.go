package osclients

import (
	"context"

	"github.com/gophercloud/gophercloud/v2"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

// withReconcileID returns a copy of the ServiceClient, with an additional tracing
// header "X-OpenStack-Request-ID" bearing the reconcileID value.
func withReconcileID(client *gophercloud.ServiceClient, reconcileID string) *gophercloud.ServiceClient {
	clientCopy := *client

	moreHeadersCopy := make(map[string]string)
	for k, v := range client.MoreHeaders {
		moreHeadersCopy[k] = v
	}
	moreHeadersCopy["X-OpenStack-Request-ID"] = "req-" + reconcileID

	clientCopy.MoreHeaders = moreHeadersCopy

	return &clientCopy
}

func getClient(ctx context.Context, client *gophercloud.ServiceClient) *gophercloud.ServiceClient {
	reconcileID := controller.ReconcileIDFromContext(ctx)
	if reconcileID != "" {
		return withReconcileID(client, string(reconcileID))
	}
	return client
}
