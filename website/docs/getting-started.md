# Getting started

## In a nutshell

1. Deploy ORC to your Kubernetes cluster
1. Create a Kubernetes secret containing a `clouds.yaml`
1. Deploy your OpenStack infrastructure as Kubernetes custom resources

## Deploy ORC to your Kubernetes cluster

To install the latest released version of ORC, the simplest is probably to use the provided `install.yaml` file:

```sh
export ORC_RELEASE="https://github.com/k-orc/openstack-resource-controller/releases/latest/download/install.yaml"
kubectl apply --server-side -f $ORC_RELEASE
```

## Create a Kubernetes secret containing a `clouds.yaml`

```sh
kubectl create secret generic openstack-clouds \
    --from-file=clouds.yaml=${XDG_CONFIG_HOME:-~/.config}/openstack/clouds.yaml
```

!!! note

    The command above will upload your entire `clouds.yaml` to your Kubernetes
    cluster! If that is not appropriate, you may want to upload a slimmed version
    of it.

## Deploy your OpenStack infrastructure as Kubernetes custom resources

This is the definition of a subnet. You can apply it to your cloud:

```yaml
kubectl apply --server-side -f- <<EOF
apiVersion: openstack.k-orc.cloud/v1alpha1
kind: Subnet
metadata:
  labels:
    app.kubernetes.io/name: openstacksubnet
    app.kubernetes.io/instance: openstacksubnet-gettingstarted
    app.kubernetes.io/part-of: gettingstarted
  name: subnet-1
spec:
  cloudCredentialsRef:
    cloudName: openstack
    secretName: openstack-clouds
  managementPolicy: managed
  networkRef: network-1
  resource:
    description: |
      Example subnet
    tags:
    - gettingstarted
    ipVersion: 4
    allocationPools:
    - start: 192.168.1.5
      end: 192.168.1.60
    cidr: 192.168.1.0/24
EOF
```

The controller will only attempt creating the subnet when the corresponding
Network exists:

```plaintext
$ kubectl get subnets
NAME       ID    AVAILABLE   MESSAGE                                       AGE
subnet-1         False       Waiting for Network/network-1 to be created   4s
```

Let's create it:

```yaml
kubectl apply --server-side -f- <<EOF
apiVersion: openstack.k-orc.cloud/v1alpha1
kind: Network
metadata:
  labels:
    app.kubernetes.io/name: openstacknetwork
    app.kubernetes.io/instance: openstacknetwork-gettingstarted
    app.kubernetes.io/part-of: gettingstarted
  name: network-1
spec:
  cloudCredentialsRef:
    cloudName: openstack
    secretName: openstack-clouds
  managementPolicy: managed
  resource:
    description: |
      Example network
    tags:
    - gettingstarted
EOF
```

After a few seconds, both resources become ready:

```plaintext
$ kubectl get networks
NAME        ID                                     AVAILABLE   MESSAGE                           AGE
network-1   5df739fb-2cdf-4d49-ad67-95fd36d99056   True        OpenStack resource is available   96s

$ kubectl get subnets
NAME       ID                                     AVAILABLE   MESSAGE                           AGE
subnet-1   bb1f0b74-0e79-4f77-b518-6a05a61662f0   True        OpenStack resource is available   2m42s
```

The subnet can be inspected through its Kubernetes representation, under
`.status.resource`:

```sh
kubectl get subnet subnet-1 -o yaml
```

```yaml
apiVersion: openstack.k-orc.cloud/v1alpha1
kind: Subnet
metadata:
  annotations:
  creationTimestamp: "2025-01-03T16:29:05Z"
  finalizers:
  - openstack.k-orc.cloud/port
  - openstack.k-orc.cloud/subnet
  generation: 1
  labels:
    app.kubernetes.io/instance: openstacksubnet-gettingstarted
    app.kubernetes.io/name: openstacksubnet
    app.kubernetes.io/part-of: gettingstarted
  name: subnet-1
  namespace: default
  resourceVersion: "2318"
  uid: cc132c51-990d-4b4e-be80-3b72822d1a88
spec:
  cloudCredentialsRef:
    cloudName: openstack
    secretName: openstack-clouds
  managementPolicy: managed
  networkRef: network-1
  resource:
    allocationPools:
    - end: 192.168.1.60
      start: 192.168.1.5
    cidr: 192.168.1.0/24
    description: |
      Example subnet
    ipVersion: 4
    tags:
    - gettingstarted
status:
  conditions:
  - lastTransitionTime: "2025-01-03T16:34:31Z"
    message: OpenStack resource is available
    observedGeneration: 1
    reason: Success
    status: "True"
    type: Available
  - lastTransitionTime: "2025-01-03T16:34:31Z"
    message: OpenStack resource is up to date
    observedGeneration: 1
    reason: Success
    status: "False"
    type: Progressing
  id: bb1f0b74-0e79-4f77-b518-6a05a61662f0
  resource:
    allocationPools:
    - end: 192.168.1.60
      start: 192.168.1.5
    cidr: 192.168.1.0/24
    description: |
      Example subnet
    dnsPublishFixedIP: false
    enableDHCP: true
    gatewayIP: 192.168.1.1
    ipVersion: 4
    ipv6AddressMode: ""
    ipv6RAMode: ""
    name: subnet-1
    projectID: c73b7097d07c46f78eb4b4dcfbac5ca8
    revisionNumber: 1
    tags:
    - gettingstarted
```

## Reset

To reset both Kubernetes and Openstack to their original state, delete the
resources and undeploy ORC:

```sh
kubectl delete subnet subnet-1
kubectl delete network network-1
kubectl delete secret openstack-clouds
kubectl delete -f $ORC_RELEASE
```
