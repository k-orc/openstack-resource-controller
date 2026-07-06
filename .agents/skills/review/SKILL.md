---
name: review
description: Review ORC controller code for Kubernetes best practices and ORC conventions. Use after implementing or modifying a controller.
disable-model-invocation: true
---

# ORC Code Review Guide

Review ORC controller code for correctness, Kubernetes best practices, and ORC conventions. Produce a structured report at the end.

## Step 1: Identify Review Scope

Determine what changed and which checklists apply:

1. Run `git diff` (or `git diff --cached`, or diff against the base branch) to identify modified files.
2. Categorize changes by file type:
   - `api/v1alpha1/*_types.go` -> API Types checklist
   - `internal/controllers/*/controller.go` -> Controller Setup checklist
   - `internal/controllers/*/actuator.go` -> Actuator Logic checklist
   - `internal/controllers/*/status.go` -> Status Writer checklist
   - `internal/controllers/*/tests/` -> Test Coverage checklist
   - Any `.go` file -> Code Style checklist
3. Always apply the Kubernetes Best Practices checklist.
4. Read every changed file in full before reviewing. Also read surrounding context (e.g., the full `*_types.go` for the resource, even if only part changed).

## Step 2: API Types (`*_types.go`)

Review any `api/v1alpha1/*_types.go` file against these rules:

### Structure

- [ ] Three hand-written types exist: `<Resource>ResourceSpec`, `<Resource>Filter`, `<Resource>ResourceStatus`.
- [ ] Top-level types (`<Resource>`, `<Resource>Spec`, `<Resource>Status`, `<Resource>List`) are code-generated in `zz_generated.*` files and NOT hand-edited.
- [ ] `ResourceSpec` contains fields mapping to OpenStack create API parameters.
- [ ] `Filter` contains a subset of identifying fields, all optional pointers.
- [ ] `ResourceStatus` contains observed state from OpenStack.

### Validation Markers

- [ ] String fields use `+kubebuilder:validation:MinLength` / `MaxLength` constraints.
- [ ] Numeric fields use `+kubebuilder:validation:Minimum` / `Maximum` where appropriate.
- [ ] Enum types use `+kubebuilder:validation:Enum` listing all valid values.
- [ ] Filter structs have `+kubebuilder:validation:MinProperties:=1` (at least one criterion required).
- [ ] Slice fields have `+listType` annotations (`set` for unique items like tags, `atomic` for ordered/opaque lists, `map` with `+listMapKey` for keyed lists like conditions).

### Immutability

- [ ] Fully immutable resources (rare, e.g., ServerGroup) apply `+kubebuilder:validation:XValidation:rule="self == oldSelf"` at the struct level.
- [ ] Partially mutable resources apply `rule="self == oldSelf"` on individual immutable fields, leaving mutable fields unmarked.
- [ ] Immutability validation messages are descriptive (e.g., `"imageRef is immutable"`).

### Field Conventions

- [ ] `+required` fields use value types (e.g., `RAM int32`) but still have `json:"...,omitempty"`.
- [ ] `+optional` fields use pointer types (e.g., `*OpenStackName`, `*bool`) to distinguish "not set" from zero.
- [ ] In `ResourceStatus`, fields are `+optional` with pointers or plain strings with `omitempty`.
- [ ] Status string fields use `string` with `+kubebuilder:validation:MaxLength=1024`, not the strongly-typed wrapper.

### Shared Types

- [ ] Resource names use `OpenStackName` (not raw `string`). Check the specific OpenStack project for the correct max length (Keystone: 64 chars, Neutron: 255 chars, etc.).
- [ ] References to other ORC objects use `KubernetesNameRef` with a `Ref` suffix (e.g., `projectRef`, `networkRef`).
- [ ] References NEVER point to raw OpenStack resource IDs. OpenStack IDs appear only in status.
- [ ] IP addresses use `IPvAny`, CIDRs use `CIDR`, MACs use `MAC`.
- [ ] Neutron resources use shared types: `NeutronDescription`, `NeutronTag`, `FilterByNeutronTags`, `NeutronStatusMetadata`.
- [ ] Non-Neutron resources define their own tag types with appropriate length constraints.

### Sub-resources

- [ ] Nested sub-resources have separate Spec and Status types (e.g., `SecurityGroupRule` vs `SecurityGroupRuleStatus`).
- [ ] Sub-resource Status types include an `ID` field when the sub-resource has its own OpenStack ID.
- [ ] Complex cross-field validation uses `XValidation` rules on the sub-resource struct.

## Step 3: Controller Setup (`controller.go`)

### Basic Setup

- [ ] RBAC markers are present and minimal (only the verbs actually needed).
- [ ] Controller name is lowercase, may contain hyphens, and is unique across all controllers.
- [ ] `GetName()` returns the controller name constant.
- [ ] `SetupWithManager` follows the standard pattern: builder -> watches -> dependency registration -> reconciler creation -> complete.

### Dependencies

- [ ] Dependencies are declared as **package-level variables**, not inside functions.
- [ ] `DeletionGuardDependency` is used when deleting the dependency would either fail or cause the dependent to fail.
- [ ] Regular `Dependency` (no deletion guard) is used for import-only dependencies and cases where OpenStack allows the deletion.
- [ ] Each dependency has a descriptive name (e.g., `vipSubnetDependency` not `subnetDependency` when multiple subnet types exist).
- [ ] Field path strings in dependency declarations match the actual API field paths.
- [ ] Extraction functions correctly handle nil checks for optional references.

### Watches

- [ ] Each dependency has a corresponding `Watches` call in `SetupWithManager`.
- [ ] Watch handlers use `predicates.NewBecameAvailable` to avoid unnecessary reconciles.
- [ ] Credential dependency watch is always registered.
- [ ] All dependency registrations use `errors.Join` with `AddToManager`.

## Step 4: Actuator Logic (`actuator.go`)

### Structure

- [ ] Type aliases defined at the top of the file for `osResourceT`, actuator interfaces, and `helperFactory`.
- [ ] Compile-time interface assertions present (`var _ createResourceActuator = myActuator{}`).
- [ ] OS client interface defined locally with only the methods the actuator needs.
- [ ] Actuator struct holds the OS client and optionally `k8sClient` (when dependencies are used).

### Resource Name

- [ ] `getResourceName` helper exists: returns `spec.resource.name` if set, otherwise falls back to the ORC object name.

### GetOSResourceByID

- [ ] Wraps errors with `progress.WrapError`.
- [ ] Handles "not found" correctly (returns `nil` resource, not an error).

### ListOSResourcesForAdoption

- [ ] Returns `false` (second return value) when `spec.resource` is nil (no spec to match against).
- [ ] Builds client-side filters matching the **full** resource spec for accurate adoption.

### ListOSResourcesForImport

- [ ] Builds filters from the import filter spec only.
- [ ] All filter fields are mapped.

### CreateResource

- [ ] Translates ORC spec into OpenStack `CreateOpts` completely.
- [ ] **MUST NOT** perform any action after the Create API call (idempotency requirement).
- [ ] Any actions before Create are idempotent (Create may be called many times).
- [ ] Non-retryable errors are wrapped with `orcerrors.Terminal`.
- [ ] Lists (tags, etc.) are sorted before passing to Create for deterministic state.
- [ ] Finalizers on dependencies are added immediately before the Create call, not earlier.

### DeleteResource

- [ ] **MUST NOT** perform any action after the Delete API call.
- [ ] Handles "not found" gracefully (resource already deleted).
- [ ] For resources with intermediate states: checks provisioning status before deleting, handles 409 Conflict by waiting.
- [ ] Minimal dependency requirements -- does not require dependencies that aren't strictly needed for deletion.

### ReconcileResourceActuator (if implemented)

- [ ] `GetResourceReconcilers` returns reconciler functions for post-creation tasks (e.g., setting Neutron tags, handling mutable field updates).
- [ ] Reconcilers that modify the OpenStack resource return a `progress.ProgressStatus` to force a status refresh.
- [ ] Reconcilers are independent and don't rely on side effects of other reconcilers.
- [ ] `updateResource` is used only for general mutable field updates via the resource's Update API (building `UpdateOpts`, single API call). Operations using a separate API have a descriptive name (e.g., `reconcileExtraSpecs`, `reconcileSubports`, `reconcilePassword`, `updateRules`).
- [ ] Single-concern reconcilers return `nil` (not a terminal error) when `spec.resource` is nil. Only `updateResource` returns a terminal error for nil `spec.resource`.
- [ ] `CreateResource` does not duplicate work that is handled by a reconciler. The `CreateResource` contract forbids actions that can fail after creating the primary resource.

### Error Handling

- [ ] All errors from OpenStack API calls are checked.
- [ ] Non-retryable errors (400, invalid config) wrapped with `orcerrors.Terminal` and an appropriate `ConditionReason`.
- [ ] Transient errors (5xx, network) left as default (automatic retry with backoff).
- [ ] `ReconcileStatus` return values are **never discarded** -- always assigned and propagated.
- [ ] When wrapping errors, use `progress.WrapError(err)` (not bare `fmt.Errorf`).

### Dependency Resolution

- [ ] Dependencies resolved **as late as possible**, close to the point of use.
- [ ] Dependencies not required for deletion unless strictly necessary (e.g., don't require Network to delete a Subnet with `status.ID` already set).
- [ ] Dependencies not required for import-by-ID.
- [ ] `GetDependency` results checked: if `needsReschedule` is true, return early.
- [ ] Readiness predicate uses `orcv1alpha1.IsAvailable` (the standard helper from `api/v1alpha1/conditions.go`). `Status.ID` is always set before a resource becomes Available, so checking `dep.Status.ID != nil` separately is unnecessary.

## Step 5: Status Writer (`status.go`)

### Structure

- [ ] Type aliases for `objectApplyT` and `statusApplyT` (SSA apply configuration types).
- [ ] Compile-time interface assertion for `ResourceStatusWriter`.

### ResourceAvailableStatus

- [ ] Returns `ConditionTrue` only when the resource is completely ready for use.
- [ ] Returns `ConditionFalse` when `osResource` is nil and no `status.ID` exists (not yet created).
- [ ] Returns `ConditionUnknown` when `osResource` is nil but `status.ID` exists (can't verify current state).
- [ ] For resources with intermediate states (BUILD, PENDING_CREATE): returns `ConditionFalse` until the resource reaches a stable, usable state (e.g., ACTIVE).
- [ ] For resources in ERROR state: returns `ConditionFalse`.

### ApplyResourceStatus

- [ ] Maps **all** OpenStack resource fields to ORC status fields.
- [ ] Zero/empty values handled correctly: only include swap, ephemeral, description, etc., when non-zero/non-empty.
- [ ] Does NOT attempt to preserve previous status when the OpenStack resource can't be fetched (status.resource is cleared intentionally).
- [ ] Pointer fields in status use `ptr.To()` for conversion.

## Step 6: Kubernetes Best Practices

### Conditions

- [ ] **Progressing=True** means status doesn't yet reflect spec AND controller expects more reconciles.
- [ ] **Progressing=False** means the object will NOT be reconciled again until the spec changes. This covers both success (Available=True) and terminal errors.
- [ ] **Available=True** means the resource is ready for use by consumers.
- [ ] Condition reasons use defined constants from `orcv1alpha1` (e.g., `ConditionReasonInvalidConfiguration`, `ConditionReasonTransientError`).
- [ ] Conditions are not set directly by the actuator -- the generic reconciler handles this based on `ReconcileStatus` and `ResourceStatusWriter` return values.

### Finalizers

- [ ] Finalizers on dependency objects are added only immediately before the OpenStack create/update call that references them (not during initialization).
- [ ] Deletion guard finalizers are managed by `DeletionGuardDependency` -- the controller doesn't manually add/remove them.
- [ ] The controller's own finalizer is managed by the generic reconciler framework.

### Server-Side Apply

- [ ] Status is written via SSA apply configurations (not direct status updates).
- [ ] `GetApplyConfig` returns a fresh apply configuration each time.
- [ ] Status is written in a single SSA transaction per reconcile.

### Resource Safety

- [ ] No cascade deletes unless the user explicitly requested them.
- [ ] No auto-correction of invalid states that might cause data loss.
- [ ] Prefer failing safely over making assumptions.

## Step 7: Code Style

See AGENTS.md for conventions (import ordering, logging levels, pointer handling). Only flag deviations that affect correctness:

- [ ] Generated files (`zz_generated.*`) are not hand-edited.
- [ ] `make generate` has been run after any API type changes.
- [ ] Constants from gophercloud are preferred over locally defined string constants (e.g., `ports.StatusActive` instead of `"ACTIVE"`).

## Step 8: Test Coverage

### E2E Test Directories

For each controller, verify the following test directories exist under `internal/controllers/<kind>/tests/`:

| Required Test | Purpose |
|---------------|---------|
| `<kind>-create-minimal/` | Create with only required fields, verify status matches |
| `<kind>-create-full/` | Create with all fields populated |
| `<kind>-import/` | Import an existing OpenStack resource |
| `<kind>-import-error/` | Import with no matches, verify error handling |
| `<kind>-dependency/` | Test dependency waiting and deletion guard protection |

| Conditional Test | When Required |
|------------------|---------------|
| `<kind>-update/` | Resource has mutable fields |
| `<kind>-import-dependency/` | Import filter references other ORC objects |

### E2E Test Quality

- [ ] Each test directory has a `README.md` describing each step.
- [ ] Step files use zero-padded numeric prefixes (`00-`, `01-`, etc.).
- [ ] Cloud credentials secret created via `TestStep` command (not a manifest) using `E2E_KUTTL_OSCLOUDS`.
- [ ] Assertions verify `status.resource` fields match the spec.
- [ ] Conditions (`Available`, `Progressing`) are asserted with correct `status`, `reason`, and `message`.
- [ ] Dependency tests verify: (1) waiting state with `Progressing=True`, (2) availability after dep created, (3) finalizer blocks dep deletion, (4) dep deleted after resource deleted.
- [ ] Update tests use `kubectl replace` (not KUTTL patch) to test field removal.
- [ ] CEL expressions (`celExpr`) used for complex assertions (e.g., checking `deletionTimestamp`, finalizer membership, field absence with `!has(...)`).

### Unit / API Validation Tests

- [ ] API validation tests exist at `test/apivalidations/<resource>_test.go` for non-trivial validation rules.
- [ ] Unit tests cover any complex helper logic.

## Step 9: Produce Review Report

After running through all applicable checklists, produce a structured report:

### Report Format

```
## Review Summary

**Scope**: <list of files reviewed>
**Overall**: <PASS / PASS with suggestions / NEEDS CHANGES>

## Blockers
Items that MUST be fixed before merge. These are correctness issues, violations
of idempotency/safety invariants, or missing required functionality.

- [file:line] Description of the issue and why it's a blocker.

## Warnings
Items that SHOULD be fixed. These are convention violations, missing edge case
handling, or patterns that may cause issues in production.

- [file:line] Description and recommendation.

## Suggestions
Items that COULD be improved. Style preferences, minor optimizations, or
additional test coverage that would be nice to have.

- [file:line] Description and suggestion.

## Positive Observations
Notable good practices observed in the code (keep brief, 2-3 items max).
```

### Severity Guidelines

**Blocker** -- any of:
- Violates CreateResource/DeleteResource idempotency invariant (actions after the API call)
- Incorrect Progressing/Available condition semantics (could cause reconciliation to hang)
- Missing `ReconcileStatus` propagation (discarded return value)
- Terminal error not marked terminal (infinite retry of unfixable error)
- Missing finalizer or finalizer added too early
- Security issue (RBAC too broad, secrets leaked in logs)
- Data loss risk (cascade delete without explicit user intent)

**Warning** -- any of:
- Missing validation markers on API types
- Dependency resolved too early (unnecessary coupling)
- Missing error wrapping (`progress.WrapError`)
- Incomplete status mapping (fields not reflected in status)
- Missing E2E test for a standard scenario
- Wrong logging level
- Missing interface assertion

**Suggestion** -- any of:
- Import ordering
- Naming could be more descriptive
- Additional test coverage beyond the standard set
- Code could be simplified
- Comment could be clearer
