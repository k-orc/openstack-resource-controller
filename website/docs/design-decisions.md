# Design decisions

## API

In order to fully manage dependencies, ORC only ever references other ORC
objects. ORC does not directly reference OpenStack objects in *spec*. However,
when reporting status it directly reports what OpenStack returned, including
direct references to OpenStack objects by UUID.

ORC does not map OpenStack API 1 to 1. We're taking some liberties when
implementing the API:

- we reserve the right to rename some fields (for instance, ports `Fixed IPs`
  changed to `IP Addresses`)
- there should be one and only one way of creating any resource in ORC

## ORC favors deterministic behavior

OpenStack sometimes exhibits non-deterministic behavior. A typical example
would be the creation of a port on a network that has multiple subnets: when
not specifying which subnet to use, OpenStack will pick one for you. ORC tries
to limit this behavior, and in this particular example, does not create IP
addresses for ports unless the user specifies some in the port's spec.
These behaviours should be documented in the API's Godoc.

## Importing existing resources

ORC provides different mechanisms for importing existing resources:

- Import: allows the creation of ORC objects for resources that ORC does not
  manage. We can import via an import filter that returns only one OpenStack
  resource. If the resource does not exist we will poll OpenStack until it
  does. If a filter returns more than one OpenStack resource, this is
  a terminal error.
- Adoption: a mechanism to create ORC objects for orphaned resources that were
  not recorded properly. Adoption is intended to be an implementation detail
  which improves robustness and idempotency of resource creation in the event
  of failures. It is not intended to be exposed as ‘intentional’ behaviour.

## Dependencies management

ORC is a declarative tool that offloads the burden of creating resources in
order from the user. As a result, it needs to handle dependencies.

### At resource creation

An object which references another object in its spec will automatically wait
for referenced objects to both exist and be available before proceeding.

### At resource deletion

ORC will ensure that an ORC object cannot be deleted while it still has other
objects which depend on it. This ensures that when deleting a large number of
objects they are automatically deleted in the correct order.

## Cross-namespace references

ORC does not allow cross-namespace references. This applies to all references,
including references to other ORC objects and to non-ORC objects such as
credential secrets and user-data secrets.

This design decision reduces the potential security impact of bugs in the ORC
controller by reducing the chances to leak resources from other namespaces. It
also opens the opportunity to run ORC in a single namespace without any
ClusterRoles.

## Duplicate resource names

ORC always references existing OpenStack resources by their ID, stored in the
ORC object status. Because of this, it correctly handles cases where OpenStack
resources have the same name.

By default, ORC creates OpenStack resources with the same name as the ORC
resource. Because Kubernetes does not allow objects with the same name, this
means that by default OpenStack resources will all have distinct names.
However, ORC also allows the OpenStack resource name to be overridden in the
resource spec, so it remains possible to create OpenStack resources with the
same name.

## Name reuse

We allow name reuse of ORC objects. This is safe thanks to our dependency
management.

## Error reporting

ORC is an OpenStack agent. Whatever the OpenStack API returns to us, we
consider that it would be returned to the user anyway so it’s safe to reflect
the same output into ORC (even if it contains secrets…). However other errors,
including e.g. Kubernetes errors should be summarised rather than copied
verbatim to reduce the potential for accidental secret leaks.
