---
name: review-controller
description: Review OpenStack Resource Controller controller/API PRs and patches. Use when reviewing new or changed ORC controllers, actuator/status logic, CRD API types, dependencies, import/adoption behavior, error handling, or KUTTL/controller tests, especially before responding to maintainer review comments.
---

# Review Controller

Use this skill as a review workflow, not as an implementation recipe. Start from the diff and judge whether the controller is robust, convention-compatible, and sufficiently covered by KUTTL.

## Workflow

1. Read the changed API type, actuator, status writer, controller setup, generated files, examples, and KUTTL suites for the affected resource.
2. Compare the change to nearby controllers with the same pattern: simple immutable resources, resources with dependencies, mutable-field reconcilers, import/adoption, or extra API calls.
3. Read [references/controller-review-checklist.md](references/controller-review-checklist.md) before writing findings or making fixes.
4. Prioritize findings in this order: incorrect API contract, unsafe lifecycle/dependency behavior, broken reconciliation semantics, missing status mapping, weak KUTTL coverage, then style/convention issues.
5. Prefer concrete fixes over broad advice. If you change code, regenerate and run focused tests plus lint; run KUTTL when `E2E_OSCLOUDS` is available.

## Review Output

Lead with findings, ordered by severity, using file and line references. For each finding, explain the observable failure or maintainer convention it violates. Keep summaries secondary.

When addressing existing review comments, map each comment to a code/test change or explicitly explain why no change is needed.
