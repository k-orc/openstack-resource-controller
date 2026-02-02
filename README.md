# ORC: openstack-resource-controller

## Description

[**openstack-resource-controller**][orc], or **ORC**, is a Kubernetes API for
declarative management of OpenStack resources. By fully controlling the order
of OpenStack operations, it allows consumers to easily create, manage, and
reproduce complex deployments. ORC aims to be easily consumed both directly by
users, and by higher level controllers. ORC aims to cover all OpenStack APIs
which can be expressed declaratively.

ORC is based on [Gophercloud][gophercloud], the OpenStack Go SDK.

[orc]: https://github.com/k-orc/openstack-resource-controller
[gophercloud]: https://github.com/gophercloud/gophercloud

## Maturity

While we currently cover a limited subset of OpenStack resources, we focus on
making existing controllers as correct and predictable as possible. We
encourage you to contribute, file issues, and help improve the project as we
continue to work on it!

ORC versioning follows [semver](https://semver.org/spec/v2.0.0.html): there will be no breaking changes within a major release.

## How You Can Contribute

We welcome contributions of all kinds! Whether you’re fixing bugs, adding new features, or improving documentation, your help is greatly appreciated. To get started:

* Fork the repository.
* Create a new branch for your changes.
* Setup a [local development environment](https://k-orc.cloud/development/quickstart/).
* Read the [developers guide](https://k-orc.cloud/development/overview/).
* Make your changes and test thoroughly.
* Submit a pull request with a clear description of your changes.

For significant new features or architectural changes, please review our
[enhancement proposal process](enhancements/README.md) before starting work.

If you're unsure where to start, check out the [open issues](https://github.com/k-orc/openstack-resource-controller/issues) and feel free to ask
questions or propose ideas!

Join us on kubernetes slack, on [#gophercloud](https://kubernetes.slack.com/archives/C05G4NJ6P6X). Visit [slack.k8s.io](https://slack.k8s.io) for an invitation.

## Getting Started

### Prerequisites

ORC heavily uses [CEL validations](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#validation-rules) and thus depends on Kubernetes v1.29 or above.

### Installation

To install a released version of ORC, the simplest is probably to use the provided `install.yaml` file:

```
export ORC_RELEASE="https://github.com/k-orc/openstack-resource-controller/releases/latest/download/install.yaml"
kubectl apply --server-side -f $ORC_RELEASE
```

You may later uninstall ORC using the same `install.yaml` file:
```
kubectl delete -f $ORC_RELEASE
```

### Usage

* [Deploy your first OpenStack resource using ORC](https://k-orc.cloud/getting-started/)

## Supported OpenStack resources

| **controller**              | **1.x** | **2.x** | **main** |
|:---------------------------:|:-------:|:-------:|:--------:|
| domain                      |         |    ✔    |     ✔    |
| flavor                      |         |    ✔    |     ✔    |
| floating ip                 |         |    ◐    |     ◐    |
| group                       |         |    ✔    |     ✔    |
| image                       |    ✔    |    ✔    |     ✔    |
| keypair                     |         |    ◐    |     ◐    |
| network                     |         |    ◐    |     ◐    |
| port                        |         |    ◐    |     ◐    |
| project                     |         |    ◐    |     ◐    |
| role                        |         |    ✔    |     ✔    |
| router                      |         |    ◐    |     ◐    |
| security group (incl. rule) |         |    ✔    |     ✔    |
| server                      |         |    ◐    |     ◐    |
| server group                |         |    ✔    |     ✔    |
| service                     |         |    ✔    |     ✔    |
| subnet                      |         |    ◐    |     ◐    |
| volume                      |         |    ◐    |     ◐    |
| volume type                 |         |    ◐    |     ◐    |



✔: mostly implemented

◐: partially implemented

## License

Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

