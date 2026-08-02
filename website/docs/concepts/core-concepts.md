# Core Concepts

This page explains what ORC does and how its main features work. For the
reasoning behind these choices, see [Design Principles](design-principles.md). For
the full field-level API documentation, see the
[CRD Reference](../crd-reference.md).

## Management Policies

Every ORC resource has a `managementPolicy` that determines how ORC treats the
underlying OpenStack resource:

| Policy | Description |
|--------|-------------|
| `managed` | ORC creates, updates, and deletes the OpenStack resource. This is the default. |
| `unmanaged` | ORC imports an existing OpenStack resource but will not modify or delete it. |

### When to use `managed`

- The resource should be created and owned by ORC
- You want ORC to update the OpenStack resource when you change the spec
- You want ORC to delete the OpenStack resource when you remove it from
  Kubernetes

### When to use `unmanaged`

- The resource is owned by another system (e.g. created by an admin or
  another tool)
- You need to reference shared infrastructure that multiple projects use
  (external networks, public flavors)

The distinction matters most at deletion time. By default, when you delete a
`managed` ORC object, ORC deletes the corresponding OpenStack resource. When you
delete an `unmanaged` ORC object, the OpenStack resource is always left
untouched. See [Deletion Behavior](#deletion-behavior) for how to change the
default for managed resources.

### Create vs Import

A **managed** resource uses `spec.resource` to describe what should be created:

```yaml
spec:
  managementPolicy: managed
  resource:
    description: My application network
```

An **unmanaged** resource uses `spec.import` to find an existing OpenStack
resource by UUID or by a filter query:

```yaml
spec:
  managementPolicy: unmanaged
  import:
    filter:
      name: public
      external: true
```

When importing by filter, the filter must match exactly **one** resource:

- **No matches**: ORC keeps retrying (the resource stays `Progressing: True`).
  This is useful when you expect another system to create the resource soon.
- **Multiple matches**: ORC reports a terminal error. Make the filter more
  specific.

See [Troubleshooting](../troubleshooting.md#common-issues) for debugging
import filter issues. The [Tutorial](../getting-started.md) walks through
importing resources step by step.

## Resource References and Dependencies

ORC resources reference each other using `*Ref` fields (e.g. `networkRef`,
`flavorRef`, `portRef`). These references serve three purposes:

1. **Name resolution**: References are resolved by Kubernetes object name
   within the same namespace. References to other OpenStack resources always
   go through an ORC object, never by raw OpenStack UUID. (A few spec fields
   do accept raw IDs for values that aren't references to other resources,
   such as setting a resource's own ID or a port's host ID.)

2. **Automatic ordering**: ORC waits for a referenced resource to exist and
   be `Available` before proceeding. You can apply all your resources at once
   in any order and ORC will sort out the sequencing.

3. **Deletion protection**: ORC prevents deletion of a resource while other
   resources still reference it. If you delete everything at once, ORC
   automatically deletes them in the correct reverse order.

See [Design Principles](design-principles.md#orc-objects-only-reference-orc-objects)
for why ORC requires all references to go through ORC objects.

### Cross-namespace references

ORC does not allow cross-namespace references. All `*Ref` fields resolve within
the same namespace. This applies to ORC objects, credential secrets, and any
other referenced objects. See
[Design Principles](design-principles.md#no-cross-namespace-references) for the
rationale behind this choice.

## Deletion Behavior

For managed resources, the `managedOptions.onDelete` field controls what happens
when the Kubernetes object is deleted:

| Value | Description |
|-------|-------------|
| `delete` | Delete the OpenStack resource. This is the default. |
| `detach` | Keep the OpenStack resource; only remove the ORC object. |

```yaml
spec:
  managementPolicy: managed
  managedOptions:
    onDelete: detach  # Keep the OpenStack resource on deletion
  resource:
    # ...
```

Use `detach` when you want to stop managing a resource through ORC without
destroying the underlying infrastructure, for example during a migration.

## Name Reuse

Deleting an ORC object and creating a new one with the same name is safe. ORC's
dependency management ensures that the old resource is fully cleaned up before
the new one takes its place.

## Status and Conditions

Every ORC resource reports its state through two conditions: **Available** (is
the resource ready?) and **Progressing** (is ORC still working on it?). A
resource is healthy when `Available=True` and `Progressing=False`.

When something goes wrong, the condition's `reason` and `message` fields explain
what happened. See [Status Conditions Reference](../reference/conditions.md)
for the full list of condition states, reasons, and recommended actions.

ORC surfaces error messages, including potentially sensitive details from
OpenStack, directly in status conditions.

The `.status.resource` field contains the observed state from OpenStack,
including fields that OpenStack assigns (like `projectID`, `createdAt`, etc.).

## Cloud Credentials

Every ORC resource has its own `cloudCredentialsRef` that points to a Kubernetes
Secret containing OpenStack credentials. The secret holds a standard OpenStack
`clouds.yaml`, and each ORC resource specifies which cloud entry to use.
Because credentials are per-resource, you can manage OpenStack resources across
multiple clouds or projects from the same namespace.

ORC prevents deletion of credential secrets while they are still referenced by
ORC resources, ensuring credentials aren't accidentally removed from under
running infrastructure.

See [Set Up Cloud Credentials](../howto/cloud-credentials.md) for step-by-step
instructions on creating the secret, adding custom CA certificates, and
referencing credentials from ORC resources.

## Resource Naming

By default, ORC creates OpenStack resources with the same name as the Kubernetes
object. You can override this using `spec.resource.name`. See
[Design Principles](design-principles.md#resource-naming) for how ORC handles
duplicate names safely.

## Deterministic Behavior

When OpenStack would create resources behind the scenes or make arbitrary
choices, ORC requires the user to be explicit instead. See
[Design Principles](design-principles.md#deterministic-behavior) for examples.
