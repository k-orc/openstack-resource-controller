package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/registeredlimits"
)

func createRegisteredLimit(ctx context.Context, c *gophercloud.ServiceClient, svcName, resourceName string, limit int) (string, error) {
	svcId, err := getServiceId(ctx, c, svcName)
	if err != nil {
		return "", fmt.Errorf("get service id with name: %w", err)
	}

	limits, err := registeredlimits.BatchCreate(ctx, c, registeredlimits.BatchCreateOpts{
		{
			ServiceID:    svcId,
			ResourceName: resourceName,
			DefaultLimit: limit,
		},
	}).Extract()

	if err != nil {
		return "", err
	}

	if len(limits) != 1 {
		return "", fmt.Errorf("unexpected creation response")
	}

	slog.Info("registered limit created", "serviceName", svcName, "resourceName", resourceName, "limit", limit)

	return limits[0].ID, nil
}

func setupRegisteredLimits(ctx context.Context, c *gophercloud.ServiceClient) error {
	// Format of REGISTER_LIMITS:
	// ServiceName/ResourceName/Limit,ServiceName/ResourceName/Limit...
	limitEnv := os.Getenv("REGISTER_LIMITS")
	if limitEnv == "" {
		return nil
	}

	for res := range strings.SplitSeq(limitEnv, ",") {
		if res == "" {
			continue
		}

		splits := strings.Split(res, "/")
		if len(splits) != 3 {
			return fmt.Errorf("unexpect resource request %q", res)
		}

		svcName, resName, limitStr := splits[0], splits[1], splits[2]
		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			return fmt.Errorf("invalid limit %s: %w", limitStr, err)
		}

		id, err := createRegisteredLimit(ctx, c, svcName, resName, limit)
		if err != nil {
			return fmt.Errorf("create registered limit: %w", err)
		}

		registeredLimitIds = append(registeredLimitIds, id)
	}

	slog.Info("created registered limits", "ids", registeredLimitIds)

	return nil
}

func cleanUpRegisteredLimits(ctx context.Context, c *gophercloud.ServiceClient) {
	for _, id := range registeredLimitIds {
		if err := deleteRegisteredLimit(ctx, c, id); err != nil {
			slog.Error("delete registered limit", "error", err, "id", id)
			continue
		}

		slog.Info("deleted registered limit", "id", id)
	}
}

func cleanUpDomains(ctx context.Context, c *gophercloud.ServiceClient) {
	domainEnv := os.Getenv("DOMAINS")
	if domainEnv == "" {
		return
	}

	for d := range strings.SplitSeq(domainEnv, ",") {
		disableDomain(ctx, c, d)
	}
}
