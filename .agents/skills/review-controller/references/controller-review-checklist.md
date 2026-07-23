# ORC Controller Review Checklist

This checklist is distilled from recurring review feedback on ORC controller and API PRs.

## API Shape

- Do not expose raw OpenStack IDs in spec references. If an ID is the resource's own creation ID, document that and add the linter exception inline.
- Check OpenStack API docs, OpenStack service schema/source, and gophercloud behavior before adding validation. Avoid over-restricting fields when OpenStack accepts empty strings, shorter values, spaces, numeric protocol aliases, or service-specific formats.
- Copy OpenStack constants and terminology verbatim unless the API contract has a clear reason to use a Kubernetes-facing name.
- Use resource-specific types when services enforce different rules. Avoid premature common types for fields like extra specs or metadata when Nova, Cinder, Neutron, etc. differ.
- For key-value lists with unique keys, use `+listType=map` and `+listMapKey=<key-field>`. Add validation tests when CEL or list-map behavior is non-trivial.
- Prefer `int32` or `int64` for Kubernetes API integer fields. Do not introduce narrower integer types even when OpenStack has a smaller effective range.
- Optional spec fields should usually be pointers with `omitempty` when the controller must distinguish unspecified from zero, empty string, or `false`. Non-pointer optional fields can create API compatibility problems later.
- Keep unsupported fields out of the API/status until the controller can populate or reconcile them. A smaller robust controller is preferred over a broad half-implemented one.
- If a mutable OpenStack field is not implemented yet, mark it immutable or document the limitation. Do not imply mutability without reconciliation and tests.
- If fields are mutually exclusive or exactly one must be set, enforce that in API validation instead of relying on actuator checks.
- Treat generated clients and public API changes as compatibility surface. Avoid breaking type changes unless the release plan allows a major version bump.
- API comments must describe the actual field and status behavior. Generated docs should be regenerated, not hand-edited.

## Actuator And Reconciliation

- Separate OpenStack API calls into separate reconcilers. Do not hide secondary operations such as extra specs, ACLs, tags, or metadata inside `CreateResource` if they can fail independently and need drift reconciliation.
- If OpenStack supports a field only through a separate update API, create a separate reconciler rather than folding it into the generic update path.
- Name reconcilers by concern, for example `reconcileExtraSpecs` or `reconcileMetadata`, not generic names when there is only one concern.
- Return `nil` when a reconciler finds no work, matching existing reconcile functions.
- Before making a field mutable, prove the controller can converge for set, update, and clear/unset cases. If gophercloud cannot clear a value or OpenStack defaults make desired state unknowable, keep the field immutable and document why.
- Check nil, empty, defaulted, and environment-dependent values carefully. Avoid logic that constantly reconciles because OpenStack returns defaults that differ from an omitted spec field.
- For Neutron updates, pass the resource revision number where the API supports it. When deciding whether an update is needed, prefer the local `needsUpdate`/update-map pattern used by nearby controllers.
- For non-retryable OpenStack or gophercloud errors, wrap with `orcerrors.Terminal(...)` using `ConditionReasonInvalidConfiguration`; leave retryable errors retryable.
- If custom osclient or unmarshalling code works around a gophercloud gap, add a short TODO/comment with the upstream issue or expected removal condition. Prefer fixing or waiting for gophercloud when the workaround would be broad or fragile.
- Do not add osclient wrapper tests just because a wrapper exists. Unit-test custom parsing, custom API behavior, or non-trivial actuator logic; rely on KUTTL for ordinary gophercloud wrapper calls.
- Avoid unused helpers and commented-out test/code blocks. If deferred behavior is important, reference an issue or add a short explanatory comment.
- Interface assertions are useful as developer feedback, but they are not functional behavior.

## Dependencies

- Use ORC dependency mechanisms for references to other ORC resources; never resolve OpenStack resources directly from user-provided IDs.
- Filter APIs should use ORC refs for ORC-managed resources, for example `ProjectRef` or `PortRef`, rather than exposing OpenStack IDs or legacy aliases such as tenant IDs unless the resource API specifically needs them.
- Resolve dependencies only where needed and only after checking reconcile status. If a dependency may be unavailable, guard before dereferencing status IDs.
- When multiple dependencies are required, merge their `ReconcileStatus` values so Progressing can report all missing or unavailable dependencies, not only the first one encountered.
- When dependencies are mutable, prove finalizers on old dependencies are handled. If not covered, prefer making dependency fields immutable for the first implementation.
- Choose deletion guards deliberately. Some relationships, such as flavor-like or server-group-like references, may need ordering without preventing deletion; verify OpenStack semantics before using a deletion guard.
- Use current dependency predicates such as `orcv1alpha1.IsAvailable` where that is the local convention.

## Status

- Status should reflect what OpenStack returns. Do not over-validate or normalize observed status beyond deterministic ordering and comparable representations.
- Status validation should usually be more relaxed than spec validation. Avoid status regex/enums unless needed to keep kube-apiserver validation safe; otherwise return OpenStack values verbatim.
- Optional empty OpenStack values should usually be omitted from status unless the project has a convention for surfacing zero values.
- Every status field added by the API should have a status writer mapping and a deterministic test, or be removed/deferred.
- For imported resources, ensure the status is refreshed enough to report the actual OpenStack state.
- Watch for stale or misleading status fields on resources that progress asynchronously. If a status field can remain stale without another reconcile, either poll/resync appropriately or leave it out.

## KUTTL And Tests

- KUTTL is the main confidence surface for controller behavior. Prefer meaningful KUTTL coverage over excessive actuator unit tests.
- Create-minimal tests should assert both expected values and important absent fields. The resulting resource should be exactly what the controller promises.
- Create-full tests should set each supported spec field to a non-default value when the environment can support it, then assert the corresponding status. Use a different `spec.resource.name` than `metadata.name` to prove OpenStack name override behavior when the resource supports names.
- Update tests should prove mutable fields actually changed in OpenStack and can be cleared when supported.
- Import tests should exercise all supported filter fields and prove the imported object is the intended resource, not a similarly named trap resource.
- Adoption tests should cover the case where an existing OpenStack resource matches the managed spec and can be adopted, including supplied identifiers such as addresses when relevant.
- Use YAML status matchers for fixed scalar status. Reserve CEL for dynamic IDs, cross-resource equality, list membership/size, and absence checks.
- Avoid redundant CEL expressions that duplicate normal KUTTL YAML matching. Use CEL when normal matching cannot assert the condition, especially absence checks.
- Add API validation tests for complex CEL, list-map uniqueness, or exact OpenStack regex constraints.
- If a field cannot be reliably asserted in e2e because the backend is environment-specific, document that in the test or PR instead of adding a flaky assertion.
- If a test needs admin credentials for a field such as `adminStateUp`, use the admin cloud only where required and add a short comment explaining why.
- Follow current KUTTL naming conventions, including `00-create-resource.yaml` for creation steps and the established `*-update`, `*-import`, and dependency suite layouts.
- Use `resource: {}` in minimal create/update tests when the only required resource name can come from `metadata.name`.

## Scope And PR Hygiene

- Keep unrelated generated or packaging churn out of controller PRs unless the change requires it.
- Regenerate after API changes: CRD, OpenAPI, applyconfig/client artifacts, examples/docs as applicable.
- Check examples and samples for API field renames.
- Prefer small, robust first implementations. Leave broader field support, optimization, or service-specific enhancements for follow-up PRs when they would need substantial extra dependency or drift handling.
- Remove scaffolding leftovers, stale TODOs, boilerplate comments, `.gitkeep` files in populated test directories, and outdated downstream references.
- Use current project boilerplate in new files, including `Copyright The ORC Authors.`
