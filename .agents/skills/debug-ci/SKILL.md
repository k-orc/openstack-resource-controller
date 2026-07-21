---
name: debug-ci
description: "ORC CI: Debug E2E test failures in GitHub Actions. Download artifacts, analyze KUTTL output, trace through ORC controller and OpenStack logs to find root cause."
---

# ORC CI E2E Failure Debugging

Debug failures in the ORC (OpenStack Resource Controller) E2E test suite run via GitHub Actions.

## Prerequisites

- `gh` CLI authenticated with access to `k-orc/openstack-resource-controller`
- Working directory: the ORC repository root

## Overview

ORC E2E tests run KUTTL tests against live OpenStack environments in GitHub Actions. Each environment runs a devstack instance, a Kind cluster, and the ORC controller manager. When a test fails, the CI uploads artifacts containing OpenStack service logs, ORC controller logs, and managed resource snapshots.

## Step 1: Identify the Failure

Given a CI run URL or job ID, identify which job(s) failed and which test(s) failed:

```bash
# List jobs and their status
gh run view <RUN_ID> --repo k-orc/openstack-resource-controller \
  --json jobs --jq '.jobs[] | {name: .name, conclusion: .conclusion, id: .databaseId}'

# Get logs and find the failing test(s)
gh run view <RUN_ID> --repo k-orc/openstack-resource-controller \
  --log --job <JOB_ID> 2>&1 | grep "FAIL: kuttl"

# Find the assertion diff (KUTTL shows expected vs actual)
gh run view <RUN_ID> --repo k-orc/openstack-resource-controller \
  --log --job <JOB_ID> 2>&1 | grep -A30 "case.go:335:"

# Find the test namespace (needed for correlating logs)
gh run view <RUN_ID> --repo k-orc/openstack-resource-controller \
  --log --job <JOB_ID> 2>&1 | grep "<TEST_NAME>.*Creating namespace"
```

## Step 2: Download Artifacts

```bash
gh run download <RUN_ID> --repo k-orc/openstack-resource-controller \
  --name e2e-<ENVIRONMENT>-<RUN_ID> --dir /tmp/ci-artifacts
```

### Artifact Contents

**ORC controller:**

| File | Description |
|---|---|
| `orc-pod.log` | ORC controller manager stdout/stderr (may be empty — see Caveats) |
| `orc-pod.txt` | `kubectl describe pod` output for the controller |
| `orc-resources.yaml` | All resources in `orc-system` namespace |
| `orc-managed-resources/*.yaml` | All ORC CRs across all namespaces (may be empty if KUTTL already cleaned up) |

**OpenStack service logs** (in `devstack-logs/`):

| File | Service | ORC Resources |
|---|---|---|
| `neutron-api.log` | Neutron | Network, Subnet, Port, Router, RouterInterface, SecurityGroup, FloatingIP, Trunk, AddressScope |
| `n-api.log` | Nova API | Server, ServerGroup, KeyPair, Flavor |
| `n-cpu.log` | Nova Compute | Server build/rebuild failures |
| `n-sch.log` | Nova Scheduler | Server scheduling failures |
| `g-api.log` | Glance | Image |
| `c-api.log` | Cinder API | Volume, VolumeType |
| `c-vol.log` | Cinder Volume | Volume attach/detach, backend errors |
| `keystone.log` | Keystone | Project, User, Group, Role, Domain, ApplicationCredential, Service, Endpoint |

**System:**

| File | Description |
|---|---|
| `journal.log` | Full system journal from the runner |
| `free.txt` | Memory usage at time of log collection |

## Step 3: Analyze the Failure

### 3a. Understand the Test

KUTTL tests live in `internal/controllers/<resource>/tests/<test-name>/`. Each test directory contains a `README.md` describing the test's purpose and what each step does. Steps are numbered pairs of files:
- `NN-<name>.yaml` — a KUTTL `TestStep` or raw resources to apply (e.g., `00-secret.yaml` creates the cloud credentials secret, `01-create-resource.yaml` applies the ORC object)
- `NN-assert.yaml` — expected state (KUTTL polls until match or timeout)

Start by reading the test's `README.md` for an overview before examining individual step files. Then read the failing step's assert file and the corresponding resource file to understand what was expected vs. what was applied.

### 3b. Check ORC Controller Logs

If `orc-pod.log` has content, search it for the test namespace:

```bash
grep "<KUTTL_NAMESPACE>" /tmp/ci-artifacts/orc-pod.log
```

If the log is empty, you must reconstruct the failure from OpenStack-side logs (see Step 3c).

### 3c. Trace OpenStack Operations

#### Request ID correlation

ORC injects the controller-runtime reconcile ID as an `X-OpenStack-Request-ID` header on every API call to OpenStack (see `internal/scope/client.go`). This ID appears in every OpenStack service log line as the **second** `req-` field:

```
[req-<OPENSTACK_INTERNAL_ID> req-<ORC_RECONCILE_ID> <user> <project>]
```

This lets you trace every OpenStack operation back to a specific ORC reconcile loop. If you have the ORC controller log, find the reconcile ID there and grep for it in any OpenStack service log to see exactly what API calls that reconcile made. Conversely, when you spot an error in an OpenStack log, the second `req-` field tells you which ORC reconcile triggered it.

```bash
# Find ALL OpenStack operations from one ORC reconcile (works in any service log)
grep "req-<ORC_RECONCILE_ID>" /tmp/ci-artifacts/devstack-logs/*.log

# Find which ORC reconcile triggered a specific OpenStack-internal request
grep "req-<OPENSTACK_INTERNAL_ID>" /tmp/ci-artifacts/devstack-logs/neutron-api.log
# → the second req- field on that line is the ORC reconcile ID
```

#### Which log file for which resource

| ORC Resource | OpenStack Service | Log File |
|---|---|---|
| Network, Subnet, Port, Router, RouterInterface, SecurityGroup, FloatingIP, Trunk, AddressScope | Neutron | `devstack-logs/neutron-api.log` |
| Server, ServerGroup, KeyPair | Nova | `devstack-logs/n-api.log` |
| Flavor | Nova | `devstack-logs/n-api.log` |
| Image | Glance | `devstack-logs/g-api.log` |
| Volume, VolumeType | Cinder | `devstack-logs/c-api.log` |
| Project, User, Group, Role, Domain, ApplicationCredential, Service, Endpoint | Keystone | `devstack-logs/keystone.log` |

#### General techniques

```bash
# Find operations for a specific resource by name or ID
grep "<RESOURCE_NAME_OR_ID>" /tmp/ci-artifacts/devstack-logs/<service>.log

# Find HTTP requests/responses (shows method, URL, status code, timing)
grep "\[pid:" /tmp/ci-artifacts/devstack-logs/<service>.log | grep "<RESOURCE_ID_OR_NAME>"

# Find errors in a time window
grep "14:38\|14:39\|14:40" /tmp/ci-artifacts/devstack-logs/<service>.log | grep "ERROR"

# Find resource creation requests (Neutron/Keystone format)
grep "Request body.*<resource_type>" /tmp/ci-artifacts/devstack-logs/<service>.log

# Find HTTP responses by status code in a time window
grep "\[pid:" /tmp/ci-artifacts/devstack-logs/<service>.log | grep "14:38" | grep "HTTP/1.1 4"
```

### 3d. Check for OpenStack-Side Errors

Search for errors across all service logs in the test time window:

```bash
grep "ERROR" /tmp/ci-artifacts/devstack-logs/{neutron-api,n-api,g-api,c-api,keystone}.log | grep "14:38\|14:39"
```

Common patterns:

- **Phantom resource (201 → 404)**: An API returns 201 (created) but the resource is immediately destroyed internally. Seen with Neutron/OVN race conditions (`ERROR ovsdbapp` in the same request ID as the create), but can happen in any service. ORC then gets 404 on the next GET and marks the resource as a terminal error.
- **Permission errors (403)**: Non-admin operations on admin-only resources (e.g., creating subnets on external networks, manipulating other projects' resources).
- **Resource conflicts (409)**: Concurrent modifications to the same resource.
- **Resource not found (404)**: On GET after a previous successful create — the resource was deleted externally, by another test's cleanup, or by an internal OpenStack error.
- **Server build failures (Nova)**: Server stuck in ERROR state. Check `n-cpu.log` and `n-sch.log` for scheduler or hypervisor errors.
- **Volume attach/detach failures (Cinder)**: Volume stuck in `attaching`/`detaching` state. Check `c-vol.log` for backend errors.

### 3e. Understand ORC's Behavior

Key code paths when tracing failures:

| Situation | ORC Behavior | Code Location |
|---|---|---|
| `GetOSResourceByID` returns 404 after create | Terminal error: "resource has been deleted from OpenStack" | `internal/controllers/generic/reconciler/resource_actions.go` |
| Dependency not Available | Waits with progress message | `internal/util/dependency/` |
| Non-retryable OpenStack error | Terminal error, no retry | `internal/util/errors/errors.go` |
| RouterInterface with no status | routerinterface controller skipped it (Router not Available) | `internal/controllers/routerinterface/reconcile.go` |

## Step 4: Determine Test Timing

KUTTL tests run concurrently. To correlate ORC activity with OpenStack logs, establish the time window:

1. Find the test start time from the KUTTL log: `Creating namespace "<NAMESPACE>"`
2. Find the test end time from: `test step failed` or `test step completed`
3. Search OpenStack logs within that time window

```bash
# Save the full CI log for repeated searching
gh run view <RUN_ID> --repo k-orc/openstack-resource-controller \
  --log --job <JOB_ID> 2>&1 > /tmp/ci-logs.txt

# Find the time window for a specific test
grep "<TEST_NAME>" /tmp/ci-logs.txt | head -5   # start time
grep "<TEST_NAME>" /tmp/ci-logs.txt | tail -5   # end time
```

## Step 5: Classify the Failure

| Category | Description | Action |
|---|---|---|
| **OpenStack bug** | Neutron/Nova/etc. returned success but resource is broken | File upstream or document as known flake |
| **ORC bug** | Controller logic error, race condition, or incorrect error handling | File ORC issue with fix plan |
| **Test bug** | Wrong credentials, wrong assertions, missing dependencies | Fix the test |
| **Infrastructure flake** | Timeout, resource exhaustion, network issue | Document and consider retry logic |

### Classification Guardrails

Before classifying a failure as **Infrastructure flake**, verify:

1. You have **positive evidence** of an infrastructure problem (OOM in `free.txt`, network timeout in `journal.log`, resource exhaustion, etc.) — not just "I can't find another explanation".
2. You have explicitly ruled out **ORC bug** (reviewed the relevant controller code path), **Test bug** (verified assertions and test setup are correct), and **OpenStack bug** (checked service logs for API-level errors) with specific reasoning for why each doesn't apply.
3. State the specific evidence for your classification — if you cannot point to a concrete log line or metric, reconsider your conclusion.

"Unknown" is a valid classification. Do not default to "Infrastructure flake" when uncertain.

## Caveats

### Empty Controller Pod Log

The `hack/collectlogs` script may produce an empty `orc-pod.log`. When this happens, reconstruct the failure from OpenStack service logs and the controller source code. The pod description (`orc-pod.txt`) confirms whether the pod was healthy.

### Managed Resources May Be Empty

KUTTL deletes the test namespace before `collectlogs` runs (the script runs in the `if: failure()` step after the test step). The `orc-managed-resources/` files will show empty lists if cleanup completed before log collection.

### OpenStack Environments

The environment name and OpenStack release are visible in the job name (e.g., "Run acceptance tests against OpenStack **flamingo**"). To see the current list of environments and their OpenStack versions:

```bash
yq '.jobs.e2e.strategy.matrix.include[] | .name + ": " + .openstack_version' .github/workflows/e2e.yaml
```

Failures on only one environment may indicate an OpenStack version-specific bug rather than an ORC bug.
