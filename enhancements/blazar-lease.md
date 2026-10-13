# Enhancement: Blazar Lease Controller

|       Field        |     Value     |
| ------------------ | ------------- |
| **Status**         | implementable |
| **Author(s)**      | @barnes-c     |
| **Created**        | 2026-10-13    |
| **Last Updated**   | 2026-10-13    |
| **Tracking Issue** | TBD           |

## Summary

This enhancement adds a `Lease` kind which manages leases in [Blazar][blazar].

With this, a user could reserve compute capacity for a time window, similar to GCP's Dynamic Workload Scheduler or AWS's EC2 Capacity Blocks.

In this time window, a user could then reserve a host, capacity for instances, a specific amount of instances of one flavor or instances of a given size.
Blazar then exposes leases via a scheduler hint or flavor ID that needs to be consumed to "claim" the lease.

[blazar]: https://docs.openstack.org/blazar/latest/

## Motivation

The `Lease` kind is really interesting in ORC because users could then use tools such as Cluster API Provider OpenStack (CAPO) to schedule their workloads in advance.

This is interesting because things like GPUs or other expensive infra are expensive to hold and you only want to own it when you are actually using it.
When the point comes, though, to use it, it might not be available due to the scarcity of these resources nowadays.

ATM, one would need to create a lease with Blazar outside of ORC or CAPO and then bring the resulting scheduler hint or flavor ID into the Kubernetes world.

It's already possible with gophercloud to use Blazar resources, so the support is there (once `v2.16` is released).

## Goals

- A `Lease` kind supporting the three compute reservation types of Blazar: `physical:host`, `virtual:instance`, and `flavor:instance`.
- Create, delete, import and adoption, like the other controllers.
- Report what instances need to consume to "activate" the lease (reservation IDs, flavor IDs from Blazar).
- Conditions which reflect the time-bound nature of a lease. E.g. `Available` only while the lease is active or `Progressing` while waiting?

## Non-Goals

- **Floating IP reservations** (`virtual:floatingip`): these are less interesting and can wait.
- **Lease events and notifications**: before and after lease end, for example.

## Proposal

### API

A lease contains a list of reservations.
Blazar would create all reservations of a lease together, so they belong in one object.

```yaml
apiVersion: openstack.k-orc.cloud/v1alpha1
kind: Lease
metadata:
  name: gpu-training
spec:
  cloudCredentialsRef:
    cloudName: openstack
    secretName: openstack-clouds
  managementPolicy: managed
  resource:
    startDate: "2027-01-01T08:00:00Z" # optional, defaults to "now"
    endDate: "2027-12-12T18:00:00Z"
    reservations:
      - host:
          min: 1
          max: 2
          resourceProperties: '["==", "$gpu", "a100"]'
      - instance:
          amount: 4
          vcpus: 128
          memory: 10TB
          disk: 1TB
      - flavorInstance:
          amount: 2
          flavorRef: nvidia.b300 # ORC Flavor
```

The idea of how statuses would then be reported:

```yaml
status:
  id: 6f0c...
  resource:
    name: gpu-training
    status: PENDING
    startDate: "2027-01-01T08:00:00Z"
    endDate: "2027-12-12T18:00:00Z"
    reservations:
      - id: 1b2c...                 # host: pass as the `reservation` scheduler hint
        resourceType: physical:host
        status: pending
      - id: 3d4e...
        resourceType: virtual:instance
        status: pending
        flavorID: 3d4e...           # instance: boot with this flavor
        serverGroupID: 5f6a...
      - id: 7b8c...
        resourceType: flavor:instance
        status: pending
        flavorID: 7b8c...           # flavor: boot with this flavor
```

## Alternatives Considered

- Calling Blazar from CAPO directly
  CAPO would work, but ORC is the better place to implement it since this project aims to declaratively manage OpenStack resources.

- Leases created out of band
  Users can create leases with the Blazar CLI and pass the reservation ID as a scheduler hint.
  This works today but is not IaC.
  The lease lifetime is not tied to the cluster; one has to copy IDs manually.

## Implementation History

- 2026-10-13: Enhancement proposed
