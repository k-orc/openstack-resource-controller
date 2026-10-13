# Changelog

## v3.0-pre1 - October 14, 2026

This is the first pre-release of ORC v3. This release is safe to use in
production, but additional backward-incompatible changes may land before
v3.0.0. The Go module path has changed from `v2` to `v3`, and the release
contains several backward-incompatible API changes. See the
[upgrade guide](howto/upgrading.md#from-v2-to-v3) for migration instructions.

### New controllers

- **DNSZone**: Manage Designate DNS zones
- **Limit**: Manage Keystone unified limits
- **Region**: Manage Keystone regions
- **RegisteredLimit**: Manage Keystone registered limits
- **RoleAssignment**: Manage Keystone role assignments (relationship resource, no OpenStack ID)
- **ShareNetwork**: Manage Manila share networks
- **SubnetPool**: Manage Neutron subnet pools

### New features

- **Drift detection and resync**: Every resource now has an optional `spec.resyncPeriod` field
  and a `status.lastSyncTime` timestamp. A new `--default-resync-period` manager flag sets the
  global default. When enabled, ORC periodically re-reads the OpenStack resource and updates
  status accordingly. External deletion of managed resources is detected and the resource is
  re-created automatically. Drift detection is not yet enabled by default; see
  [Enable Drift Detection](howto/drift-detection.md) for setup instructions and
  [Drift Detection Explained](concepts/drift-detection.md) for how it works.
- **Helm charts**: New `orc` and `orc-crds` Helm charts for installation.
- **OLM bundle**: Enriched metadata for OperatorHub.io.
- **RBAC aggregation**: ORC CRDs now contribute rules to the standard Kubernetes `view`, `edit`,
  and `admin` [aggregated ClusterRoles](https://kubernetes.io/docs/reference/access-authn-authz/rbac/#user-facing-roles).
- **Server**: Boot-from-volume support via `spec.resource.bootVolume`. New
  `spec.resource.schedulerHints` with support for server groups, affinity/anti-affinity hints,
  cell targeting, and arbitrary additional properties.
- **Flavor**: Mutable extra specs via `spec.resource.extraSpecs`.
- **Router**: Static routes support via `spec.resource.routes`.
- **RouterInterface**: Support attaching by port (new `portRef` field and `Port` type).
- **Network**: Provider segments in spec (`spec.resource.segments`) and status.
- **Port**: New fields: `trustedVIF`, `valueSpecs`, `propagateUplinkStatus`.
- **Endpoint**: `regionRef` in spec, filter, and status.
- **Image**: `firmwareType` hardware property.

### Breaking changes

For detailed migration instructions, see the [upgrade guide](howto/upgrading.md#from-v2-to-v3).

- **Go module path**: Changed from
  `github.com/k-orc/openstack-resource-controller/v2` to
  `github.com/k-orc/openstack-resource-controller/v3`. All Go imports must be updated.

- **Server: `imageRef` changed from required to optional**: The type changed from
  `KubernetesNameRef` (value) to `*KubernetesNameRef` (pointer). A new CEL rule requires
  exactly one of `imageRef` or `bootVolume` to be set. Existing manifests that already
  specify `imageRef` are not affected, but Go code referencing this field must handle the
  pointer.

- **Server: `serverGroupRef` moved to `schedulerHints`**: The field
  `spec.resource.serverGroupRef` has been removed. Use
  `spec.resource.schedulerHints.serverGroupRef` instead.

- **Network: `status.resource.provider` replaced by `status.resource.segments`**: The
  single `provider` object in status has been replaced by a `segments` list that can
  represent multi-segment provider networks. Any code or tooling reading
  `status.resource.provider` must switch to `status.resource.segments`.

- **Port: `securityGroupRefs` changed from `[]OpenStackName` to `[]KubernetesNameRef`**:
  Values were previously raw OpenStack security group names. They are now references to ORC
  SecurityGroup objects. Existing Port manifests must be updated to reference ORC
  SecurityGroup object names. (Fixes [#438](https://github.com/k-orc/openstack-resource-controller/issues/438))

- **Status `bool` fields changed to `*bool`**: Five status fields changed from bare `bool`
  to `*bool` so that `false` is distinguishable from unset in JSON. The affected fields are:
    - `ApplicationCredentialResourceStatus.Unrestricted`
    - `ImageResourceStatus.Protected`
    - `SecurityGroupResourceStatus.Stateful`
    - `ServerResourceStatus.ConfigDrive`
    - `UserResourceStatus.Enabled`

    This affects Go code that reads these fields directly (must dereference the pointer).
    Existing YAML manifests are not affected since these are status-only.

### Bug fixes

- Flavor/Domain: Relaxed import ID validation from UUID-only to a broader pattern, allowing
  non-UUID identifiers used by some OpenStack deployments (Fixes [#869](https://github.com/k-orc/openstack-resource-controller/issues/869))
- SecurityGroup: Fixed deliberately-empty rules list never reaching Available (Fixes [#960](https://github.com/k-orc/openstack-resource-controller/issues/960))
- SecurityGroup: Fixed incorrect index used for PortRange when creating rules
- Status: Fixed `CreatedAt`/`UpdatedAt` set to literal JSON null on a zero timestamp
- RouterInterface: Fixed status conditions when Router is not Available (Fixes [#838](https://github.com/k-orc/openstack-resource-controller/issues/838))
- Port: Fixed endless update loop when MAC is empty in AllowedAddressPairs (Fixes [#852](https://github.com/k-orc/openstack-resource-controller/issues/852))
- Port/Volume: Poll for status transition after server attachment (port ACTIVE, volume in-use) (Fixes [#842](https://github.com/k-orc/openstack-resource-controller/issues/842))
- Invalidate cached provider scope when an OpenStack endpoint is missing from the catalog (Fixes [#941](https://github.com/k-orc/openstack-resource-controller/issues/941))
- Network transport errors are now retryable (Fixes [#879](https://github.com/k-orc/openstack-resource-controller/issues/879))

### Infrastructure improvements

- Go bumped to 1.25.13
- gophercloud bumped to v2.15.0
- kuttl bumped to v0.27.0
- Website migrated from mkdocs to zensical, with ORC logo and branding
- Various dependency bumps (controller-runtime, cel-go, grpc, otel, etc.)

---

## v2.6 - June 10, 2026

### New features

- Flavor: Added `id` field for creation

### Bug fixes

- Tightened adoption filters across multiple controllers (AddressScope, FloatingIP, Group, Network, Port, Project, Router, SecurityGroup, ServerGroup, Subnet, Trunk, User) to prevent adopting resources that don't fully match the spec
- Fixed terminal error classification: use `IsRetryable` instead of `IsConflict` so non-HTTP gophercloud errors (e.g., client-side validation failures) are no longer retried indefinitely (Fixes [#241](https://github.com/k-orc/openstack-resource-controller/issues/241))
- Treated Neutron quota-exceeded 409 errors as retryable so controllers retry when quota becomes available (Fixes [#667](https://github.com/k-orc/openstack-resource-controller/issues/667))
- Fixed port status not updating to ACTIVE after server interface attachment
- Fixed volume status not updating to in-use after server attachment

### Infrastructure improvements

- Bumped Go to 1.25.11
- Bumped kuttl to v0.26.0
- Bumped golang.org/x/net to v0.53.0 and other dependency updates
- Added CI verification for `generate-bundle`

## v2.5 - April 16, 2026

This release adds five new controllers spanning Neutron and Keystone services,
plus significant improvements to existing controllers. This is expected to be
the last v2 release before work begins on v3, which will include minor breaking
API changes.

### New controllers
- Trunk: Manage Neutron trunk ports
- AddressScope: Manage Neutron address scopes
- Endpoint: Manage Keystone endpoints
- User: Manage Keystone users
- ApplicationCredential: Manage Keystone application credentials

### New features

* Port: Added `adminStateUp` and `macAddress` fields
* Port: Added support for port binding
* Server: Added `configdrive` and `metadata` fields
* Volume: Added ability to create bootable volumes from images
* Project: Added ability to specify domainRef
* We now have a process for [lightweight enhancement proposals](https://github.com/k-orc/openstack-resource-controller/tree/main/enhancements). We even got a proposal for [drift detection](https://github.com/k-orc/openstack-resource-controller/blob/main/enhancements/drift-detection.md).

### Bug fixes

- SecurityGroup: Fixed inverted error handling for rule creation where retryable errors were marked terminal ([#672](https://github.com/k-orc/openstack-resource-controller/pull/672))
- SecurityGroup: Fixed availability status by counting security group rules (Fixes [#120](https://github.com/k-orc/openstack-resource-controller/issues/120))
- RouterInterface: Fixed missing status conditions when routerRef does not exist (Fixes [#314](https://github.com/k-orc/openstack-resource-controller/issues/314))
- Role: Fixed adoption of domain-scoped roles failing with 409 Conflict ([#733](https://github.com/k-orc/openstack-resource-controller/pull/733))

### Infrastructure improvements

- Go: Bumped to version 1.25.9
- Bumped dependencies, most notably gophercloud to v2.11.1
- Added ORC API linter to enforce API design philosophy (no OpenStack IDs in spec fields)
- Added API validation tests for all controllers, with scaffolding support for new controllers
- Added AI agent instructions and skills for assisted development
- CI: Hardened GitHub Actions security (pinned SHAs, scoped permissions, zizmor scanning)
- CI: Added gazpacho, dropped dalmatian testing
* Restored development container image expiration in Quay

## v2.4 - December 17, 2025

### New controllers

- Keypair: Manage Nova SSH keypairs
- Domain: Manage Keystone identity domains
- Group: Manage Keystone identity groups
- Role: Manage Keystone identity roles
- Service: Manage Keystone identity services

### New features

- Server: Added ability to specify SSH keypair
- Volume: Added support for availability zone

### Bug fixes

- Allow to use application credentials with access rules (Fixes [#596](https://github.com/k-orc/openstack-resource-controller/issues/596))
- Fix actuator errors going silently (Fixes [#587](https://github.com/k-orc/openstack-resource-controller/issues/587))
- Improve error message when cloud credentials can't be found in secret (Fixes [#589](https://github.com/k-orc/openstack-resource-controller/issues/589))

### Infrastructure improvements

- Go: Bumped to version 1.24.11
- Bumped dependencies, most notably k8s libs to v0.34, controller-runtime to v0.22.4, and gophercloud to v2.9.0.
- Added timeout environment variable for kuttl tests
- Logging: Added log levels to more messages
- Documentation: Major improvements to user and developer docs

## v2.3 - November 7, 2025

### New controllers

- Volume: Manage Cinder block storage volumes
- VolumeType: Manage Cinder volume types

### New features

- Added support for updating resources (mutability) for all relevant controllers
- Server: Expose network interfaces status
- Server: Added ability to attach and detach volumes and interfaces
- Server: Added support for Availability Zone
- Image: Added visibility filter support for image import
- Added support for generating and publishing OLM bundle images
- Added controller scaffolding tool for easier development

### Bug fixes

- Attachable resource now have their status more closely reflecting reality (Fixes [#282](https://github.com/k-orc/openstack-resource-controller/issues/282))
- Documentation: Fixed examples in getting-started guide
- Make ORC more suited for production environment by increasing memory limit to 256Mi and adjusing the API max items limits (Fixes [#527](https://github.com/k-orc/openstack-resource-controller/issues/527))

### Infrastructure improvements

- CI: Dropped caracal support, added epoxy and flamingo OpenStack versions
- CI: Enabled openstack-cli-server for testing
- Go: Bumped to version 1.24

## v2.2 - June 11, 2025

### New controllers

- Floating IP
- Server Group

### New features

- The subnet controller gained support for updating subnets. The rest of the controllers will follow in the next releases.
- Added ability to specify which project to create resources in, for all networking controllers.

### Bug fixes

- Add an optional dependency of subnet on router: a subnet will now wait for the referenced router to be available before proceeding with creation. Likewise, the router won't be deleted until all subnets that depend on it are themselves deleted (Fixes [#316](https://github.com/k-orc/openstack-resource-controller/issues/316)).
- Fix the import of images, where the status may not reflect the real status of the resource.
- Make the deletion of router interface more robust (Fixes [#378](https://github.com/k-orc/openstack-resource-controller/issues/378)).
- Add API validation to limit ExternalGateways to one until multiple gateways is effectively implemented in ORC (Fixes [#416](https://github.com/k-orc/openstack-resource-controller/issues/416)).
- Selectively stop populating status fields with zero values (Fixes [#188](https://github.com/k-orc/openstack-resource-controller/issues/188)).

### Update considerations

Although we don't guarantee that ORC runs fine on OpenStack versions that are no longer maintained by the OpenStack community, we've merged a change in this release that requires Nova from Stein. To the best of our knowledge, OpenStack Stein now becomes the minimum required version of OpenStack.

## v2.1 - May 02, 2025

Release 2.1 marks the continuation of our efforts to stabilize and consolidate ORC.

We are now building multi-platform container images, supporting `amd64`, `arm64`, `ppc64le` and `s390x` platforms.

This release also brings initial support for Keystone resources in the form of
the Project controller.

### New controllers

- Project

### New features

- ORC now passes the Kubernetes ReconcileID as the Request ID for all OpenStack API calls allowing to correlate the ORC and OpenStack logs, making troubleshooting much easier.
- The image controller is now more consistent with other controllers.
- Support setting new properties in the image controller: the `hw_rng_model` and `hw_qemu_guest_agent` hardware properties, and the `os_distro`, `os_version`, `architecture` and `hypervisor_type` properties.
- The port controller now has the ability to set port security and vnic type.

### Bug fixes

- The user-agent header now includes the ORC versions, helping identify specific versions in logs for better support and debugging.
- Add ability to create router interfaces for HA or DVR routers (Fixes [#330](https://github.com/k-orc/openstack-resource-controller/issues/330)).
- The status of servers is now reflected more accurately (Fixes [#280](https://github.com/k-orc/openstack-resource-controller/issues/280)).


## v2.0 - Mar 28, 2025

This release introduces several new controllers, expanding ORC's capabilities
beyond the original image controller. With this update, ORC now offers
a robust, stable core and a comprehensive end-to-end (e2e) test suite, making
it easier to create new controllers while ensuring quality and reliability.

Version 2.0 highlights the capabilities of ORC and the direction the project
wants to take. The API is still alpha and may change frequently.

### New controllers

- Flavor
- Network
- Port
- Router
- Security Group
- Server
- Subnet

### Breaking changes

```
github.com/k-orc/openstack-resource-controller/api/v1alpha1
  Incompatible changes:
  - ImageFilter.Name: changed from *string to *OpenStackName
  - ImageFilter: old is comparable, new is not
  - ImageProperties.MinDiskGB: changed from *int to *int32
  - ImageProperties.MinMemoryMB: changed from *int to *int32
  - ImagePropertiesHardware.CPUCores: changed from *int to *int32
  - ImagePropertiesHardware.CPUSockets: changed from *int to *int32
  - ImagePropertiesHardware.CPUThreads: changed from *int to *int32
  - ImageResourceStatus.Status: changed from *string to string
  - ImageResourceStatus: old is comparable, new is not
  - ImageStatus.DownloadAttempts: changed from *int to *int32
  - ImageStatusExtra.DownloadAttempts: changed from *int to *int32
  - OpenStackDescription: removed
```

## v1.0 - Dec 19, 2024

First public version for a standalone ORC.

This preliminary release is not intended for general consumption. Its primary
purpose is to satisfy the existing use case of
[cluster-api-provider-openstack](https://github.com/kubernetes-sigs/cluster-api-provider-openstack)
without creating any new APIs.

ORC v1.0.0 contains an API and controller for creating and deleting Glance images.

### New controllers

- Image
