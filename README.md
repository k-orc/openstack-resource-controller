# openstack-resource-controller
ORC is a set of Kubernetes controllers that manage your OpenStack tenant infrastructure.

You declare your OpenStack resource as a YAML file, you `kubectl apply` it and ORC provisions it on your OpenStack cloud.

## Description

ORC defines each OpenStack resource type as a CRD (see [./api/v1alpha1/](https://github.com/gophercloud/openstack-resource-controller/tree/main/api/v1alpha1)). Each resource type has its own controller (see [./internal/controller/](https://github.com/gophercloud/openstack-resource-controller/tree/main/internal/controller)). Controllers are responsible for creating and deleting resources in OpenStack when a CRD is created or deleted in their Kubernetes namespace.

## State of the project

**This project is currently in a prototype phase. Do NOT use in production.**

## Getting Started
You’ll need a Kubernetes cluster to run against. You can use [KIND](https://sigs.k8s.io/kind) to get a local cluster for testing, or run against a remote cluster.
**Note:** Your controller will automatically use the current context in your kubeconfig file (i.e. whatever cluster `kubectl cluster-info` shows).

### Running on the cluster
1. Deploy the controller to the cluster with the image specified by `IMG`:

```sh
make deploy IMG=quay.io/orc/orc:main
```

2. Install Instances of Custom Resources:

```sh
kubectl apply -k config/samples/
```

### Uninstall CRDs
To delete the CRDs from the cluster:

```sh
make uninstall
```

### Undeploy controller
UnDeploy the controller from the cluster:

```sh
make undeploy
```

## Contributing
See [CONTRIBUTING.md](./CONTRIBUTING.md).

### How it works
This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/).

It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/),
which provide a reconcile function responsible for synchronizing resources until the desired state is reached on the cluster.

## License

Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
