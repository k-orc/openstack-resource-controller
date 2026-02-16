---
name: testing
description: Run ORC tests (unit tests, linting, and E2E tests). Use after making changes to verify correctness.
disable-model-invocation: true
---

# ORC Testing Guide

Run unit tests, linting, and E2E tests for ORC controllers.

## Unit Tests and Linting

Before running E2E tests, ensure code compiles and passes linting:

```bash
make generate
make lint
make test
```

## E2E Test Prerequisites

E2E tests require `E2E_OSCLOUDS` environment variable pointing to a `clouds.yaml` file containing:
- A cloud named `openstack` - regular user credentials
- A cloud named `devstack-admin` - admin credentials

If the user did not provide `E2E_OSCLOUDS`, tell them local E2E testing will be skipped and they should run it manually later or in CI.

## Running E2E Tests

If `E2E_OSCLOUDS` is provided, execute each step in order:

**Step 1: Create kind cluster (if not already running)**
```bash
# Check if cluster exists
kind get clusters

# Create only if no cluster exists
kind create cluster
```
If a cluster already exists, skip creation and proceed to Step 2.

**Step 2: Verify cluster is ready**
```bash
kubectl get nodes
```
Ensure node shows `Ready` status.

**Step 3: Install CRDs**
```bash
kubectl apply -k config/crd --server-side
```

**Step 4: Stop any existing manager, rebuild, and start**
```bash
# Stop any existing manager to ensure we're running latest code
pkill -f orc-manager || true

# Build and start fresh
go build -o /tmp/orc-manager ./cmd/manager
/tmp/orc-manager -zap-log-level 5 > /tmp/manager.log 2>&1 &
```

**Step 5: Wait for manager to start and verify it's running**
```bash
sleep 5
ps aux | grep "[o]rc-manager"
```
If no process found, check `/tmp/manager.log` for errors.

**Step 6: Run E2E tests**
Replace `/path/to/clouds.yaml` with the actual path and `<kind>` with the controller name:
```bash
E2E_OSCLOUDS=/path/to/clouds.yaml E2E_KUTTL_DIR=internal/controllers/<kind>/tests make test-e2e
```

**Step 7: If tests fail, review manager logs**
```bash
# Search for errors first (logs are verbose at level 5)
grep -i error /tmp/manager.log | tail -50

# Or view more context
tail -500 /tmp/manager.log
```
Use these logs to diagnose and fix issues, then re-run the tests.

**Step 8: Cleanup**
After tests pass (or when done debugging):
```bash
pkill -f "orc-manager" || true
kind delete cluster
rm -f /tmp/manager.log /tmp/orc-manager
```

## E2E Test Directory Structure

Tests are located in `internal/controllers/<kind>/tests/`:

| Directory | Purpose |
|-----------|---------|
| `<kind>-create-minimal/` | Create with minimum required fields |
| `<kind>-create-full/` | Create with all fields |
| `<kind>-import/` | Import existing resource |
| `<kind>-import-error/` | Import with no matches |
| `<kind>-dependency/` | Test dependency waiting and deletion guards |
| `<kind>-import-dependency/` | Test import with dependency references |
| `<kind>-update/` | Test mutable field updates |
