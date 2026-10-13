# Upgrading

When upgrading across any major version, CRDs **must** be upgraded before the
controller.

Using kubectl, the `install.yaml` manifest includes both CRDs and the
controller:

```bash
export NEW_VERSION=<target_version>
kubectl apply --server-side -f \
    https://github.com/k-orc/openstack-resource-controller/releases/download/${NEW_VERSION}/install.yaml
```

Using Helm, upgrade the CRDs chart first, then the controller:

```bash
helm upgrade orc-crds oci://quay.io/orc/helm/orc-crds
helm upgrade orc oci://quay.io/orc/helm/orc
```

## From v2 to v3

This section covers the backward-incompatible changes in ORC v3 and how to
migrate from v2. For the full list of changes, see the
[changelog](../changelog.md#v30-pre1-october-14-2026).

### Go module path

The module path changed from `github.com/k-orc/openstack-resource-controller/v2`
to `github.com/k-orc/openstack-resource-controller/v3`. Update all Go imports:

```bash
find . -name '*.go' -exec sed -i \
  's|k-orc/openstack-resource-controller/v2|k-orc/openstack-resource-controller/v3|g' {} +
```

### Server: imageRef

`imageRef` changed from a required value field (`KubernetesNameRef`) to an
optional pointer (`*KubernetesNameRef`). A new CEL rule requires exactly one of
`imageRef` or `bootVolume` to be set.

No YAML changes are required if `imageRef` is already set. Go code referencing
this field must handle the pointer (e.g. using `ptr.Deref`).

### Server: serverGroupRef

The field `spec.resource.serverGroupRef` has been removed. Use
`spec.resource.schedulerHints.serverGroupRef` instead.

```yaml
# Before (v2)
spec:
  resource:
    serverGroupRef: my-server-group

# After (v3)
spec:
  resource:
    schedulerHints:
      serverGroupRef: my-server-group
```

### Network: provider to segments

The single `status.resource.provider` object has been replaced by a
`status.resource.segments` list that can represent multi-segment provider
networks. Any code or tooling reading `status.resource.provider` must switch to
`status.resource.segments`.

```yaml
# Before (v2)
status:
  resource:
    provider:
      networkType: vlan
      physicalNetwork: provider-net
      segmentationID: 100

# After (v3)
status:
  resource:
    segments:
      - networkType: vlan
        physicalNetwork: provider-net
        segmentationID: 100
```

A new `spec.resource.segments` field allows setting provider segments at
creation time.

### Port: securityGroupRefs

The Go type of `securityGroupRefs` changed from `[]OpenStackName` to
`[]KubernetesNameRef`. The YAML representation is identical (a list of
strings), so no manifest changes are needed. The values were always intended to
be ORC SecurityGroup object names.

Go code that references this field type must be updated accordingly.

### Status bool fields (Go code only)

Five status fields changed from bare `bool` to `*bool` so that `false` is
distinguishable from unset in JSON:

- `ApplicationCredentialResourceStatus.Unrestricted`
- `ImageResourceStatus.Protected`
- `SecurityGroupResourceStatus.Stateful`
- `ServerResourceStatus.ConfigDrive`
- `UserResourceStatus.Enabled`

Existing YAML manifests are not affected since these are status-only fields.

```go
// Before (v2)
if status.Resource.Protected {

// After (v3)
if ptr.Deref(status.Resource.Protected, false) {
```
