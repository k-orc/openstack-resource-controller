# ShareType Controller E2E Tests

This directory contains end-to-end tests for the ShareType controller using KUTTL.

## Test Suites

### 1. sharetype-create-minimal
Tests creating a ShareType with minimal required configuration.

**What it tests:**
- Creating ShareType with only `driverHandlesShareServers: true`
- Default `isPublic: true` behavior
- Correct extra spec propagation to status
- Resource becomes Available
- Deletion when credentials are removed

**Key validations:**
- Status has ID assigned
- Name matches expected value
- `isPublic` defaults to true
- `driver_handles_share_servers` extra spec is set to "True"

---

### 2. sharetype-create-full
Tests creating a ShareType with all available configuration options.

**What it tests:**
- Custom name override
- Setting `isPublic: false` (private share type)
- Setting `driverHandlesShareServers: false`
- Setting `snapshotSupport: true`
- All fields correctly reflected in status

**Key validations:**
- Name override works correctly
- `isPublic` is false
- `driver_handles_share_servers` extra spec is "False"
- `snapshot_support` extra spec is "True"
- Resource becomes Available

---

### 3. sharetype-import
Tests importing an existing ShareType using filter criteria.

**What it tests:**
- Unmanaged ShareType with import filter
- Filter matching by name and isPublic
- Correct filtering (doesn't match wrong resources)
- Import waits until matching resource exists

**Test flow:**
1. Create unmanaged ShareType with filter (name + `isPublic: false`)
2. Verify it waits for resource (Progressing=True)
3. Create "trap" ShareType with same name but `isPublic: true`
4. Verify import doesn't match the trap (still waiting)
5. Create actual ShareType with matching criteria
6. Verify import matches the correct resource
7. Verify imported resource has all expected fields

**Key validations:**
- Import waits when no match found
- Import doesn't match resources with different isPublic
- Import correctly identifies matching resource
- Imported resource ID matches created resource ID
- Extra specs are correctly imported

---

### 4. sharetype-import-error
Tests error handling when import filter doesn't match any resource.

**What it tests:**
- Graceful handling of non-matching import filter
- Continuous retry behavior

**Key validations:**
- Resource enters and stays in Progressing state
- No ID is assigned (resource not found)
- Appropriate condition messages

---

## Running the Tests

### Prerequisites
- Running OpenStack Manila service
- Admin credentials configured in `$E2E_KUTTL_OSCLOUDS`
- ORC controller running

### Run all ShareType tests
```bash
make test-e2e ARGS="--test sharetype"
```

### Run a specific test
```bash
kubectl kuttl test --config kuttl-test.yaml --test sharetype-create-minimal
```

## Test Design Notes

### ShareType Immutability
ShareTypes are **immutable** after creation per Manila API design. Therefore:
- No update tests are included
- All fields in `ShareTypeResourceSpec` have immutability validation
- Changes to spec require delete + recreate

### No Dependencies
ShareTypes don't depend on other ORC resources, so:
- No dependency tests are included
- Tests only need admin credentials (no other resources to set up)

### Required Fields
- `driverHandlesShareServers` is required per Manila specification
- This field determines if the storage backend manages share servers

### Extra Specs
Extra specs are stored as `map[string]interface{}` in OpenStack but are converted to `map[string]string` in the status for consistency.

Boolean values in extra specs are represented as strings:
- `true` → `"True"`
- `false` → `"False"`
