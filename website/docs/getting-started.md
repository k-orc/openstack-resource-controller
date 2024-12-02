# Getting started

## In a nutshell

1. Deploy ORC to your Kubernetes cluster
1. Create a Kubernetes secret containing a `clouds.yaml`
1. Create an OpenStackCloud object pointing to that `clouds.yaml`
1. Deploy your OpenStack infrastructure as Kubernetes custom resources

## Deploy ORC to your Kubernetes cluster

From the git repository, run:

```sh
make deploy IMG=quay.io/orc/openstack-resource-controller
```

## Create a Kubernetes secret containing a `clouds.yaml`
An example for a clouds.yaml file:
```
clouds:
    mycloud:
        auth:
            auth_url: http://192.168.20.20:5000
            password: guess
            project_name: myproject
            domain_name: Default
            username: myuser
        identity_api_version: '3'
        region_name: regionOne
```
Create the secret:

```sh
kubectl create secret generic openstack-clouds \
    --from-file=${XDG_CONFIG_HOME}/openstack/clouds.yaml
```

**Note:**
> The command above will upload your entire `clouds.yaml` to your Kubernetes
> cluster! If that is not appropriate, you may want to upload a slimmed version
> of it.

## Create an OpenStackCloud object pointing to that `clouds.yaml`

Your `clouds.yaml` contains a YAML dictionary of one or more clouds. If the
name of the cloud you want to target is, for example, `openstack-one`, your
OpenStackCloud object should look like this:

```sh
kubectl apply -f- <<EOF
apiVersion: openstack.k-orc.cloud/v1alpha1
kind: OpenStackCloud
metadata:
  labels:
    app.kubernetes.io/name: openstackcloud
    app.kubernetes.io/instance: openstackcloud-gettingstarted
    app.kubernetes.io/part-of: gettingstarted
  name: osp1
spec:
  cloud: openstack-one # <-- replace with your cloud name in clouds.yaml
  credentials:
    source: secret
    secretRef:
      name: openstack-clouds
      key: clouds.yaml
EOF
```

Check that the resource is ready in Kubernetes:

```sh
kubectl get OpenStackCloud osp1
```

If the credentials are valid, you get:
```plaintext
$ kubectl get OpenStackCloud osp1
NAME   READY   ERROR   STATUS
osp1   True    False   Ready
```

## Deploy your OpenStack infrastructure as Kubernetes custom resources

This is the definition of a subnet. You can apply it to your cloud:

```yaml
kubectl apply -f- <<EOF
apiVersion: openstack.k-orc.cloud/v1alpha1
kind: OpenStackSubnet
metadata:
  labels:
    app.kubernetes.io/name: openstacksubnet
    app.kubernetes.io/instance: openstacksubnet-gettingstarted
    app.kubernetes.io/part-of: gettingstarted
  name: subnet-1
spec:
  cloud: osp1
  resource:
    name: subnet-1
    network: network-1
    allocationPools:
    - start: 192.168.1.5
      end: 192.168.1.60
    cidr: 192.168.1.0/24
    ipVersion: IPv4
EOF
```

The controller will only attempt creating the subnet when the corresponding
OpenStackNetwork exists:

```plaintext
$ kubectl get openstacksubnet
NAME       READY   ERROR   STATUS
subnet-1   False   False   Waiting for the following dependencies to be ready: network:default/network-1
```

Let's create it:

```yaml
kubectl apply -f- <<EOF
apiVersion: openstack.k-orc.cloud/v1alpha1
kind: OpenStackNetwork
metadata:
  labels:
    app.kubernetes.io/name: openstacknetwork
    app.kubernetes.io/instance: openstacknetwork-gettingstarted
    app.kubernetes.io/part-of: gettingstarted
  name: network-1
spec:
  cloud: osp1
  resource:
    name: network-1
EOF
```

After a few seconds, both resources become ready:

```plaintext
$ kubectl get openstacknetwork
NAME        READY   ERROR   STATUS
network-1   True    False   Ready

$ kubectl get openstacksubnet
NAME       READY   ERROR   STATUS
subnet-1   True    False   Ready
```

The subnet can be inspected through its Kubernetes representation, under
`.status.resource`:

```sh
kubectl get openstacksubnet subnet-1 -o yaml
```

```yaml
apiVersion: openstack.k-orc.cloud/v1alpha1
kind: OpenStackSubnet
metadata:
  # [...]
  finalizers:
  - openstacksubnet.k-orc.cloud
  labels:
    app.kubernetes.io/instance: openstacksubnet-gettingstarted
    app.kubernetes.io/name: openstacksubnet
    app.kubernetes.io/part-of: gettingstarted
    cloud.openstack.k-orc.cloud/osp1: ""
    network.openstack.k-orc.cloud/network-1: ""
  name: subnet-1
  namespace: default
  uid: 9f256fd8-91fb-4c6d-a1fa-8a4bda1a92f5
spec:
  cloud: osp1
  resource:
    allocationPools:
    - end: 192.168.1.60
      start: 192.168.1.5
    cidr: 192.168.1.0/24
    ipVersion: IPv4
    name: subnet-1
    network: network-1
status:
  conditions:
  - message: Ready
    reason: Ready
    status: "True"
    type: Ready
  - message: ""
    reason: NoError
    status: "False"
    type: Error
  resource:
    allocationPools:
    - end: 192.168.1.60
      start: 192.168.1.5
    cidr: 192.168.1.0/24
    enableDHCP: true
    gatewayIP: 192.168.1.1
    id: 0e4a1c78-d5c3-4670-88ba-880cb56c4188
    ipVersion: 4
    name: subnet-1
    networkID: 9d4ecc67-ea5b-4344-b919-2051f0255c06
    projectID: 90dce24f8e6748bfbc319a9223d0a7a6
    tenantID: 90dce24f8e6748bfbc319a9223d0a7a6
```

## Reset

To reset both Kubernetes and Openstack to their original state, delete the
resources and undeploy ORC:

```sh
kubectl delete OpenStackSubnet subnet-1
kubectl delete OpenStackNetwork network-1
kubectl delete OpenStackCloud osp1
kubectl delete secret openstack-clouds
make undeploy
```
