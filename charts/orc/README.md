# orc

Helm chart for deploying the [OpenStack Resource Controller (ORC)](https://k-orc.cloud/).

## Prerequisites

Install the CRDs first using the [orc-crds](../orc-crds/) chart:

```bash
helm install orc-crds oci://quay.io/orc/helm/orc-crds --version <version>
```

## Install

```bash
helm install orc oci://quay.io/orc/helm/orc \
  --namespace orc-system --create-namespace \
  --version <version>
```

## Values

| Key | Default | Description |
|-----|---------|-------------|
| `image.repository` | `quay.io/orc/openstack-resource-controller` | Controller image repository |
| `image.tag` | `""` (uses appVersion) | Controller image tag |
| `image.pullPolicy` | `IfNotPresent` | Image pull policy |
| `replicaCount` | `1` | Number of controller replicas |
| `resources` | `{limits: {cpu: 500m, memory: 256Mi}, requests: {cpu: 10m, memory: 64Mi}}` | Container resource requests/limits |
| `extraArgs` | `[]` | Extra arguments passed to the manager binary |
| `serviceAccount.name` | `""` (generated) | Override service account name |
| `serviceAccount.annotations` | `{}` | Service account annotations |
| `metrics.enabled` | `true` | Enable metrics endpoint |
| `metrics.port` | `8443` | Metrics port |
| `metrics.serviceMonitor.enabled` | `false` | Create a Prometheus ServiceMonitor |
| `probes.enabled` | `true` | Enable liveness/readiness probes |
| `probes.port` | `8081` | Health probe port |
| `priorityClassName` | `system-cluster-critical` | Pod priority class |
| `nodeSelector` | `{}` | Node selector |
| `tolerations` | `[]` | Tolerations |
| `affinity` | `{}` | Affinity rules |

## RBAC

The manager ClusterRole (`templates/rbac/manager-clusterrole.yaml`) is generated
from `config/rbac/role.yaml` by `hack/gen-helm.sh`. Do not edit it by hand. Run
`make generate` to regenerate.

All other RBAC resources are hand-authored and rarely change.
