# Changelog

## v2.0 - Mar 28, 2025

This release introduces several new controllers, expanding ORC's capabilities
beyond the original image controller. With this update, ORC now offers
a robust, stable core and a comprehensive end-to-end (e2e) test suite, making
it easier to create new controllers while ensuring quality and reliability.

Version 2.0 highlights the capabilities of ORC and the direction the project
wants to take. The API is still alpha and may change frequently.

### New controllers

- Flavor
- Network
- Port
- Router
- Security Group
- Server
- Subnet

### Breaking changes

```
github.com/k-orc/openstack-resource-controller/api/v1alpha1
  Incompatible changes:
  - ImageFilter.Name: changed from *string to *OpenStackName
  - ImageFilter: old is comparable, new is not
  - ImageProperties.MinDiskGB: changed from *int to *int32
  - ImageProperties.MinMemoryMB: changed from *int to *int32
  - ImagePropertiesHardware.CPUCores: changed from *int to *int32
  - ImagePropertiesHardware.CPUSockets: changed from *int to *int32
  - ImagePropertiesHardware.CPUThreads: changed from *int to *int32
  - ImageResourceStatus.Status: changed from *string to string
  - ImageResourceStatus: old is comparable, new is not
  - ImageStatus.DownloadAttempts: changed from *int to *int32
  - ImageStatusExtra.DownloadAttempts: changed from *int to *int32
  - OpenStackDescription: removed
```

## v1.0 - Dec 19, 2024

First public version for a standalone ORC.

This preliminary release is not intended for general consumption. Its primary
purpose is to satisfy the existing use case of
[cluster-api-provider-openstack](https://github.com/kubernetes-sigs/cluster-api-provider-openstack)
without creating any new APIs.

ORC v1.0.0 contains an API and controller for creating and deleting Glance images.

### New controllers

- Image
