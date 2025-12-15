# Troubleshooting

This guide helps you diagnose and resolve common issues with ORC resources.

## Understanding Resource Status

### Checking Resource Status

Every ORC resource reports its status through conditions. To check the status of a resource:

```bash
# Quick overview
kubectl get openstack

# Detailed status for a specific resource
kubectl get network my-network -o yaml

# Just the conditions
kubectl get network my-network -o jsonpath='{.status.conditions}' | jq
```

### Status Conditions Explained

ORC resources have two primary conditions:

| Condition | Status | Meaning |
|-----------|--------|---------|
| `Available` | `True` | Resource is ready for use |
| `Available` | `False` | Resource is not ready (check message for details) |
| `Progressing` | `True` | ORC is still working on the resource |
| `Progressing` | `False` | ORC has finished (either success or terminal error) |

### Condition Reasons

The `reason` field categorizes what's happening:

| Reason | Description | Action |
|--------|-------------|--------|
| `Success` | Resource is reconciled successfully | None needed |
| `Progressing` | Normal operation in progress | Wait for completion |
| `TransientError` | Temporary error, will retry | Check if it persists |
| `InvalidConfiguration` | Spec has invalid values | Fix the resource spec |
| `UnrecoverableError` | Permanent error, won't retry | Fix the underlying issue |

## Common Issues

### Resource Won't Delete

**Symptoms:**
```bash
$ kubectl delete subnet my-subnet
# Command hangs or resource stays in "Terminating" state
```

**Causes and Solutions:**

??? note "Resource still in use"

    Another resource depends on this one. Check for finalizers:

    ```bash
    kubectl get subnet my-subnet -o jsonpath='{.metadata.finalizers}'
    ```

    Delete the dependent resources first:

    ```bash
    # Example: ports using the subnet must be deleted first
    kubectl delete port -l app=my-app
    ```

??? note "OpenStack deletion failed"

    Check the controller logs for errors:

    ```bash
    kubectl logs -n orc-system deployment/orc-controller-manager | grep my-subnet
    ```

    The OpenStack resource might have dependencies that are not reflected in ORC. Check in OpenStack:

    ```bash
    openstack subnet show my-subnet
    openstack port list --fixed-ip subnet=my-subnet
    ```

??? danger "Stuck finalizer (emergency only)"

    Only remove finalizers manually as a last resort. This can leave orphaned resources in OpenStack.

    ```bash
    kubectl patch subnet my-subnet -p '{"metadata":{"finalizers":null}}' --type=merge
    ```

### Credentials Secret Cannot Be Deleted

**Symptoms:**
```bash
$ kubectl delete secret openstack-clouds
# Hangs or shows finalizers
```

**Cause:** ORC resources still reference this secret.

**Solution:** Delete all ORC resources using this secret first. Find them all with:
```bash
kubectl get openstack -o jsonpath='{range .items[?(@.spec.cloudCredentialsRef.secretName=="openstack-clouds")]}{.kind}/{.metadata.name}{"\n"}{end}'
```

### Import Filter Matches Multiple Resources

**Symptoms:**
```yaml
conditions:
  - type: Progressing
    status: "False"
    reason: UnrecoverableError
    message: "filter returned 3 results, expected 1"
```

**Solution:** Make the filter more specific:

```yaml
# Too broad
import:
  filter:
    name: network

# More specific
import:
  filter:
    name: my-specific-network
    external: true
    tags:
      - production
```

### Import Filter Matches No Resources

**Symptoms:**
```yaml
conditions:
  - type: Progressing
    status: "True"
    reason: Progressing
    message: "Waiting for OpenStack resource to be created externally"
```

**This is normal if:**

- The resource doesn't exist yet and you're waiting for external creation

**If the resource should exist:**

- Verify it exists in OpenStack: `openstack network list --name my-network`
- Check filter criteria match exactly (case-sensitive)
- Verify you have permission to see the resource

## Controller Issues

### Viewing Controller Logs

```bash
# Follow logs
kubectl logs -n orc-system deployment/orc-controller-manager -f

# Filter for specific resource
kubectl logs -n orc-system deployment/orc-controller-manager | grep "my-network"
```

To increase log verbosity, patch the deployment to add the `-zap-log-level` flag:

```bash
kubectl patch deployment -n orc-system orc-controller-manager --type='json' -p='[
  {"op": "add", "path": "/spec/template/spec/containers/0/args/-", "value": "-zap-log-level=5"}
]'
```

### Controller Not Reconciling Resources

**Symptoms:** Resource status doesn't change, no activity in logs.

**Checks:**

```bash
# Controller is running
kubectl get pods -n orc-system

# CRDs are installed
kubectl get crd | grep openstack.k-orc.cloud

# Controller has correct RBAC
kubectl auth can-i list networks.openstack.k-orc.cloud --as=system:serviceaccount:orc-system:orc-controller-manager
```

### Controller Crashlooping

**Symptoms:**
```bash
$ kubectl get pods -n orc-system
NAME                                      READY   STATUS             RESTARTS
orc-controller-manager-xxx                0/1     CrashLoopBackOff   5
```

**Check logs:**
```bash
kubectl logs -n orc-system deployment/orc-controller-manager --previous
```

**Common causes:**

- Invalid configuration flags
- Cannot connect to Kubernetes API
- Memory limits too low ([increase in deployment](installation.md#resource-limits))

## OpenStack-Specific Issues

### Authentication Failures

**Symptoms:**
```yaml
conditions:
  - type: Progressing
    status: "True"
    reason: TransientError
    message: "no cloud named: mycloud, in the provided config"
```

**Checks:**

```bash
# Check the secret contains valid clouds.yaml
kubectl get secret openstack-clouds --template='{{index .data "clouds.yaml"|base64decode}}'
```

Verify the cloud name in your secret matches `cloudCredentialsRef.cloudName` in your resources:

```yaml
# In clouds.yaml
clouds:
  mycloud:    # ← This name...
    auth:
      ...

# Must match in your resource
cloudCredentialsRef:
  cloudName: mycloud    # ← ...this name
  secretName: openstack-clouds
```

### SSL/TLS Certificate Errors

**Symptoms in logs:**
```
x509: certificate signed by unknown authority
```

**Solutions:**

1. Add CA certificate to the secret:
   ```bash
   kubectl create secret generic openstack-clouds \
       --from-file=clouds.yaml=clouds.yaml \
       --from-file=cacert=ca-bundle.crt
   ```

2. Or use the controller's default CA certs flag:
   ```bash
   # In controller deployment
   args:
     - --default-ca-certs=/etc/ssl/certs/ca-certificates.crt
   ```

## Debugging Tips

### Correlating ORC and OpenStack Logs

ORC passes a Request ID to OpenStack for every API call. Find it in ORC logs:
```
"requestID": "req-abc123..."
```

Search for this ID in OpenStack service logs to trace the request.

### Checking OpenStack State

When ORC status doesn't match expectations, verify directly in OpenStack:

```bash
openstack server list
openstack server show <id>
```

### Testing Connectivity

Verify the controller can reach OpenStack:

```bash
# From controller pod
kubectl debug -n orc-system orc-controller-manager-5d4b8c9f7-xxxxx --attach \
    --image chainguard/curl -- curl -k <the openstack identity endpoint>
```

## Getting Help

If you're still stuck:

1. **Check existing issues:** [GitHub Issues](https://github.com/k-orc/openstack-resource-controller/issues)
2. **Join the community:** [#gophercloud on Kubernetes Slack](https://kubernetes.slack.com/archives/C05G4NJ6P6X)
3. **File a bug** with: ORC version, resource YAML (sanitized), `kubectl describe` output, and controller logs
