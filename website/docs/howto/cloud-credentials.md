# Set Up Cloud Credentials

This guide covers how to create and configure the credentials secret that ORC
uses to authenticate with OpenStack. For background on how credentials fit into
ORC's architecture, see
[Core Concepts](../concepts/core-concepts.md#cloud-credentials).

## Create the credentials secret

The secret must contain a `clouds.yaml` file:

```bash
kubectl create secret generic openstack-clouds \
    --from-file=clouds.yaml=/path/to/your/clouds.yaml
```

Example `clouds.yaml`:

```yaml
clouds:
  openstack:
    auth:
      auth_url: https://keystone.example.com:5000/v3
      project_name: my-project
      username: my-user
      password: my-password
      user_domain_name: Default
      project_domain_name: Default
    region_name: RegionOne
```

!!! tip

    You can download your `clouds.yaml` from the OpenStack dashboard under
    **API Access** → **Download OpenStack RC File** → **clouds.yaml**.

## Add custom CA certificates

If your OpenStack deployment uses a custom CA, there are two options:

**Per-secret CA certificate**: include the CA bundle in the secret. This takes
precedence over the global default. Use this when different secrets need
different CA certificates (e.g. when managing resources across multiple clouds).

```bash
kubectl create secret generic openstack-clouds \
    --from-file=clouds.yaml=/path/to/clouds.yaml \
    --from-file=cacert=/path/to/ca-bundle.crt
```

The `cacert` is picked up automatically. Your `clouds.yaml` does not need to
reference it.

**Global default CA certificate**: set the `--default-ca-certs` flag on the
controller. This applies to all secrets that don't include their own `cacert`.
Use this when all your clouds share the same CA, to avoid duplicating the
certificate in every secret. See
[Controller Configuration](../reference/configuration.md) for details.

## Reference credentials from ORC resources

Every ORC resource references credentials like this:

```yaml
spec:
  cloudCredentialsRef:
    secretName: openstack-clouds  # Name of the secret
    cloudName: openstack          # Name of the cloud in clouds.yaml
```

The `cloudName` must match an entry in your `clouds.yaml`. If it doesn't, ORC
will report a `TransientError`. See
[Troubleshooting](../troubleshooting.md#authentication-failures) for how to
diagnose this.

## Manage resources across multiple clouds

A single `clouds.yaml` can contain multiple cloud entries, and each ORC resource
selects which cloud to use via `cloudName`. This lets you manage OpenStack
resources across different clouds from the same namespace:

```yaml
clouds:
  production:
    auth:
      auth_url: https://keystone.prod.example.com:5000/v3
      project_name: prod-project
      username: prod-user
      password: prod-password
      user_domain_name: Default
      project_domain_name: Default
  staging:
    auth:
      auth_url: https://keystone.staging.example.com:5000/v3
      project_name: staging-project
      username: staging-user
      password: staging-password
      user_domain_name: Default
      project_domain_name: Default
```

Then point each resource at the appropriate cloud:

```yaml
# Network on the production cloud
spec:
  cloudCredentialsRef:
    secretName: openstack-clouds
    cloudName: production
```

```yaml
# Network on the staging cloud
spec:
  cloudCredentialsRef:
    secretName: openstack-clouds
    cloudName: staging
```

You can also use separate secrets for different clouds. This is required when
each cloud needs a different CA certificate, because a secret can only contain
one `cacert` entry. It's also useful to grant access to each cloud
independently.

## Delete a credentials secret

ORC prevents deletion of credential secrets while ORC resources still reference
them. Delete the ORC resources first:

```bash
# Find all resources using a specific secret
kubectl get openstack -o jsonpath='{range .items[?(@.spec.cloudCredentialsRef.secretName=="openstack-clouds")]}{.kind}/{.metadata.name}{"\n"}{end}'

# Delete them, then delete the secret
kubectl delete secret openstack-clouds
```
