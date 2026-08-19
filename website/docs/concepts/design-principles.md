# Design Principles

This page explains why ORC is designed the way it is. It covers the principles
and trade-offs that guide ORC's behavior. Understanding these principles helps
you predict how ORC will behave in situations the documentation doesn't
explicitly cover. For a practical introduction to ORC's features, see
[Core Concepts](core-concepts.md).

## ORC objects only reference ORC objects

In order to fully manage dependencies, ORC only ever references other ORC
objects in spec fields. You cannot put a raw OpenStack UUID in a `*Ref` field.
Instead, you create an ORC object (even an `unmanaged` import) and reference
that.

This may seem like extra work for simple cases, but it gives ORC a complete
picture of the dependency graph. That's what makes automatic ordering and
deletion protection possible: ORC knows what depends on what because every
relationship goes through a Kubernetes object it can watch.

Status fields *do* contain OpenStack UUIDs. When reporting observed state, ORC
directly reports what OpenStack returned, including resource IDs.

## No implicit resource creation

ORC avoids API options that would cause OpenStack to create resources behind the
scenes. For example, OpenStack's server create API accepts networks by network
ID, port ID, or fixed IP, all inline. Passing a network or IP address inline
would cause OpenStack to create a port implicitly, invisible to ORC. ORC
requires you to create a separate Port object and reference it via `portRef`,
so every resource has a clear owner.

## Deterministic behavior

Given a spec, the resulting OpenStack state should be predictable. ORC honors
OpenStack's default values, but where OpenStack would make an arbitrary choice,
ORC requires the user to be explicit.

For example, creating a port on a network with multiple subnets would let
OpenStack pick a subnet arbitrarily. ORC will not create IP addresses for ports
unless the user specifies them in the spec.

## No cross-namespace references

ORC does not allow cross-namespace references. This applies to all references,
including references to other ORC objects and to non-ORC objects such as
credential secrets and user-data secrets.

This design principle:

- **Reduces security risk**: a bug in the controller cannot accidentally leak
  resources from other namespaces
- **Enables namespace-scoped operation**: ORC can run in a single namespace
  without any ClusterRoles

## Resource naming

By default, ORC creates OpenStack resources with the same name as the Kubernetes
object. Since Kubernetes enforces unique names within a namespace, OpenStack
resources created by ORC will have distinct names by default.

ORC also allows overriding the OpenStack name via `spec.resource.name`, which
makes it possible to create OpenStack resources with duplicate names. ORC
handles this correctly because it always tracks resources by their OpenStack ID
(stored in `status.id`), not by name.

## Error reporting

ORC considers itself an agent of the user: error messages, including
potentially sensitive details from OpenStack, are surfaced directly in status
conditions because the user would have received the same response calling the
API directly.
