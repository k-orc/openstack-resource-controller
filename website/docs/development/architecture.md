# Architecture

This document covers architectural details relevant to controller developers.
For user-facing concepts (management policies, dependencies, deletion behavior),
see [Core Concepts](../concepts/core-concepts.md). For design rationale, see
[Design Principles](../concepts/design-principles.md).

## Adoption

Adoption is a mechanism to recover from partial failures during resource
creation. If ORC creates an OpenStack resource but fails to record its ID in
`status.id` (e.g. due to a controller crash), the resource would appear
orphaned on the next reconcile.

To handle this, before creating a new resource, ORC searches OpenStack for
existing resources that match the spec. If a match is found, ORC adopts it
instead of creating a duplicate. This makes resource creation idempotent.

Adoption is an internal implementation detail, not a user-facing feature. It
differs from [import](../concepts/core-concepts.md#create-vs-import), which is
an intentional mechanism for users to bring existing resources under ORC
management.

## Documenting implicit behavior overrides

When ORC overrides OpenStack's default behavior to ensure
[deterministic behavior](../concepts/design-principles.md#deterministic-behavior)
(e.g. not creating IP addresses for ports unless explicitly specified), the
behavior should be documented in the API's Godoc.
