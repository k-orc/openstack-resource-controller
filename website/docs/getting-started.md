This guide walks you through creating your first OpenStack resources with ORC.

## Prerequisites

- ORC installed in your Kubernetes cluster (see [Installation](installation.md))
- A Kubernetes secret with your OpenStack credentials

## Set Up Credentials

Create a secret containing your OpenStack `clouds.yaml`:

```bash
kubectl create secret generic openstack-clouds \
    --from-file=clouds.yaml=/path/to/your/clouds.yaml
```

!!! tip

    You can download your `clouds.yaml` from the OpenStack dashboard under
    **API Access** → **Download OpenStack RC File** → **clouds.yaml**.

## Create Your First Resource

Let's create a network and subnet. First, create the network:

```yaml
kubectl apply --server-side -f- <<EOF
apiVersion: openstack.k-orc.cloud/v1alpha1
kind: Network
metadata:
  name: my-network
spec:
  cloudCredentialsRef:
    cloudName: openstack
    secretName: openstack-clouds
  managementPolicy: managed
  resource:
    description: My first ORC network
EOF
```

Now create a subnet on that network:

```yaml
kubectl apply --server-side -f- <<EOF
apiVersion: openstack.k-orc.cloud/v1alpha1
kind: Subnet
metadata:
  name: my-subnet
spec:
  cloudCredentialsRef:
    cloudName: openstack
    secretName: openstack-clouds
  managementPolicy: managed
  resource:
    networkRef: my-network
    cidr: 192.168.1.0/24
    ipVersion: 4
EOF
```

!!! note

    ORC automatically handles dependencies. If you create the subnet before
    the network, it will wait:

    ```
    $ kubectl get subnets
    NAME        AVAILABLE   MESSAGE
    my-subnet   False       Waiting for Network/my-network to be created
    ```

After a few seconds, both resources become ready:

```
$ kubectl get networks,subnets
NAME         AVAILABLE   MESSAGE
my-network   True        OpenStack resource is available

NAME        AVAILABLE   MESSAGE
my-subnet   True        OpenStack resource is available
```

## Inspect Resource Status

View the full status including OpenStack-assigned fields:

```bash
kubectl get subnet my-subnet -o yaml
```

The `.status.resource` field contains the observed state from OpenStack, including
fields like `projectID`, `gatewayIP`, and `revisionNumber`.

## Cleanup

Delete the resources:

```bash
kubectl delete subnet my-subnet
kubectl delete network my-network
```

ORC automatically deletes the corresponding OpenStack resources.

## Next Steps

- **[User Guide](user-guide/index.md)** - Learn core concepts like management policies and imports
- **[CRD Reference](crd-reference.md)** - Full documentation of all resource types
