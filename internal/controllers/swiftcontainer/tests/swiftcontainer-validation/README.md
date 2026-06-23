# SwiftContainer name validation

## Step 00

Create a valid SwiftContainer to confirm the controller is operational, and
wait for it to become available.

## Step 01

Attempt to create a SwiftContainer whose underlying OpenStack container name
contains a forward slash (`/`). The Swift object-storage specification forbids
forward slashes in container names because they are used as path separators in
the object-storage URL.

The `SwiftContainerName` type enforces this constraint at two layers:

1. **CRD-level validation**: The kubebuilder pattern constraint `^[^/]+$` on
   the `SwiftContainerName` type rejects the object at the Kubernetes API
   level. The `kubectl apply` command will fail with a validation error.

2. **Controller-level validation** (defense-in-depth): Even if a name with a
   forward slash somehow bypasses API-level validation (e.g., via a direct
   etcd write), the controller detects the slash before calling the Swift API
   and sets `Available=False` with reason `InvalidConfiguration` and a message
   mentioning `forward slashes`.

This test exercises layer 1: the invalid container creation is attempted with
`ignoreFailure: true` (because the API is expected to reject it), and the
assertion verifies that:
- The invalid container does **not** exist in the cluster (was rejected)
- The valid container from step 00 continues to be `Available=True`

## Validation note

The 256-byte name length limit is enforced only at the CRD level via a CEL
rule (`self.size() <= 256`). Testing this via KUTTL is impractical because:
- YAML/JSON strings of exactly 257 bytes are cumbersome to write in test files
- The validation is already exercised by the unit tests in `actuator_test.go`

## Reference

- Swift object-storage specification: container names must not contain `/`
- https://k-orc.cloud/development/writing-tests/
