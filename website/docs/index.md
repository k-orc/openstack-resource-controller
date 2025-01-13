# openstack-resource-controller

_Deploy your OpenStack resources in a declarative way, using Kubernetes._

[**openstack-resource-controller**][orc], or **ORC**, is a Kubernetes API for
declarative management of OpenStack resources. By fully controlling the order
of OpenStack operations, it allows consumers to easily create, manage, and
reproduce complex deployments. ORC aims to be easily consumed both directly by
users, and by higher level controllers. ORC aims to cover all OpenStack APIs
which can be expressed declaratively.

ORC is based on [Gophercloud][gophercloud], the OpenStack Go SDK.

[orc]: https://github.com/k-orc/openstack-resource-controller
[gophercloud]: https://github.com/gophercloud/gophercloud
