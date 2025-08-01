# API Reference

## Packages
- [openstack.k-orc.cloud/v1alpha1](#openstackk-orccloudv1alpha1)


## openstack.k-orc.cloud/v1alpha1

Package v1alpha1 contains API Schema definitions for the openstack v1alpha1 API group


### Resource Types
- [Flavor](#flavor)
- [FlavorList](#flavorlist)
- [FloatingIP](#floatingip)
- [FloatingIPList](#floatingiplist)
- [Image](#image)
- [ImageList](#imagelist)
- [Network](#network)
- [NetworkList](#networklist)
- [Port](#port)
- [PortList](#portlist)
- [Project](#project)
- [ProjectList](#projectlist)
- [Router](#router)
- [RouterInterface](#routerinterface)
- [RouterInterfaceList](#routerinterfacelist)
- [RouterList](#routerlist)
- [SecurityGroup](#securitygroup)
- [SecurityGroupList](#securitygrouplist)
- [Server](#server)
- [ServerGroup](#servergroup)
- [ServerGroupList](#servergrouplist)
- [ServerList](#serverlist)
- [Subnet](#subnet)
- [SubnetList](#subnetlist)
- [Volume](#volume)
- [VolumeList](#volumelist)



#### Address







_Appears in:_
- [PortResourceSpec](#portresourcespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `ip` _[IPvAny](#ipvany)_ | ip contains a fixed IP address assigned to the port. It must belong<br />to the referenced subnet's CIDR. If not specified, OpenStack<br />allocates an available IP from the referenced subnet. |  | MaxLength: 45 <br />MinLength: 1 <br /> |
| `subnetRef` _[KubernetesNameRef](#kubernetesnameref)_ | subnetRef references the subnet from which to allocate the IP<br />address. |  | MaxLength: 253 <br />MinLength: 1 <br /> |


#### AllocationPool







_Appears in:_
- [SubnetResourceSpec](#subnetresourcespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `start` _[IPvAny](#ipvany)_ | start is the first IP address in the allocation pool. |  | MaxLength: 45 <br />MinLength: 1 <br /> |
| `end` _[IPvAny](#ipvany)_ | end is the last IP address in the allocation pool. |  | MaxLength: 45 <br />MinLength: 1 <br /> |


#### AllocationPoolStatus







_Appears in:_
- [SubnetResourceStatus](#subnetresourcestatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `start` _string_ | start is the first IP address in the allocation pool. |  | MaxLength: 1024 <br /> |
| `end` _string_ | end is the last IP address in the allocation pool. |  | MaxLength: 1024 <br /> |


#### AllowedAddressPair







_Appears in:_
- [PortResourceSpec](#portresourcespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `ip` _[IPvAny](#ipvany)_ | ip contains an IP address which a server connected to the port can<br />send packets with. It can be an IP Address or a CIDR (if supported<br />by the underlying extension plugin). |  | MaxLength: 45 <br />MinLength: 1 <br /> |
| `mac` _[MAC](#mac)_ | mac contains a MAC address which a server connected to the port can<br />send packets with. Defaults to the MAC address of the port. |  | MaxLength: 17 <br />MinLength: 1 <br /> |


#### AllowedAddressPairStatus







_Appears in:_
- [PortResourceStatus](#portresourcestatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `ip` _string_ | ip contains an IP address which a server connected to the port can<br />send packets with. |  | MaxLength: 1024 <br /> |
| `mac` _string_ | mac contains a MAC address which a server connected to the port can<br />send packets with. |  | MaxLength: 1024 <br /> |


#### AvailabilityZoneHint

_Underlying type:_ _string_



_Validation:_
- MaxLength: 255
- MinLength: 1

_Appears in:_
- [NetworkResourceSpec](#networkresourcespec)
- [RouterResourceSpec](#routerresourcespec)



#### CIDR

_Underlying type:_ _string_



_Validation:_
- Format: cidr
- MaxLength: 49
- MinLength: 1

_Appears in:_
- [HostRoute](#hostroute)
- [SecurityGroupRule](#securitygrouprule)
- [SubnetFilter](#subnetfilter)
- [SubnetResourceSpec](#subnetresourcespec)





#### CloudCredentialsReference



CloudCredentialsReference is a reference to a secret containing OpenStack credentials.



_Appears in:_
- [FlavorSpec](#flavorspec)
- [FloatingIPSpec](#floatingipspec)
- [ImageSpec](#imagespec)
- [NetworkSpec](#networkspec)
- [PortSpec](#portspec)
- [ProjectSpec](#projectspec)
- [RouterSpec](#routerspec)
- [SecurityGroupSpec](#securitygroupspec)
- [ServerGroupSpec](#servergroupspec)
- [ServerSpec](#serverspec)
- [SubnetSpec](#subnetspec)
- [VolumeSpec](#volumespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretName` _string_ | secretName is the name of a secret in the same namespace as the resource being provisioned.<br />The secret must contain a key named `clouds.yaml` which contains an OpenStack clouds.yaml file.<br />The secret may optionally contain a key named `cacert` containing a PEM-encoded CA certificate. |  | MaxLength: 253 <br />MinLength: 1 <br /> |
| `cloudName` _string_ | cloudName specifies the name of the entry in the clouds.yaml file to use. |  | MaxLength: 256 <br />MinLength: 1 <br /> |


#### DNSDomain

_Underlying type:_ _string_



_Validation:_
- MaxLength: 255
- MinLength: 1
- Pattern: `^[A-Za-z0-9]{1,63}(.[A-Za-z0-9-]{1,63})*(.[A-Za-z]{2,63})*.?$`

_Appears in:_
- [NetworkResourceSpec](#networkresourcespec)



#### Ethertype

_Underlying type:_ _string_



_Validation:_
- Enum: [IPv4 IPv6]

_Appears in:_
- [SecurityGroupRule](#securitygrouprule)

| Field | Description |
| --- | --- |
| `IPv4` |  |
| `IPv6` |  |


#### ExternalGateway







_Appears in:_
- [RouterResourceSpec](#routerresourcespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `networkRef` _[KubernetesNameRef](#kubernetesnameref)_ | networkRef is a reference to the ORC Network which the external<br />gateway is on. |  | MaxLength: 253 <br />MinLength: 1 <br /> |


#### ExternalGatewayStatus







_Appears in:_
- [RouterResourceStatus](#routerresourcestatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `networkID` _string_ | networkID is the ID of the network the gateway is on. |  | MaxLength: 1024 <br /> |


#### FilterByKeystoneTags







_Appears in:_
- [ProjectFilter](#projectfilter)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `tags` _[KeystoneTag](#keystonetag) array_ | tags is a list of tags to filter by. If specified, the resource must<br />have all of the tags specified to be included in the result. |  | MaxItems: 80 <br />MaxLength: 255 <br />MinLength: 1 <br /> |
| `tagsAny` _[KeystoneTag](#keystonetag) array_ | tagsAny is a list of tags to filter by. If specified, the resource<br />must have at least one of the tags specified to be included in the<br />result. |  | MaxItems: 80 <br />MaxLength: 255 <br />MinLength: 1 <br /> |
| `notTags` _[KeystoneTag](#keystonetag) array_ | notTags is a list of tags to filter by. If specified, resources which<br />contain all of the given tags will be excluded from the result. |  | MaxItems: 80 <br />MaxLength: 255 <br />MinLength: 1 <br /> |
| `notTagsAny` _[KeystoneTag](#keystonetag) array_ | notTagsAny is a list of tags to filter by. If specified, resources<br />which contain any of the given tags will be excluded from the result. |  | MaxItems: 80 <br />MaxLength: 255 <br />MinLength: 1 <br /> |


#### FilterByNeutronTags







_Appears in:_
- [FloatingIPFilter](#floatingipfilter)
- [NetworkFilter](#networkfilter)
- [PortFilter](#portfilter)
- [RouterFilter](#routerfilter)
- [SecurityGroupFilter](#securitygroupfilter)
- [SubnetFilter](#subnetfilter)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `tags` _[NeutronTag](#neutrontag) array_ | tags is a list of tags to filter by. If specified, the resource must<br />have all of the tags specified to be included in the result. |  | MaxItems: 32 <br />MaxLength: 255 <br />MinLength: 1 <br /> |
| `tagsAny` _[NeutronTag](#neutrontag) array_ | tagsAny is a list of tags to filter by. If specified, the resource<br />must have at least one of the tags specified to be included in the<br />result. |  | MaxItems: 32 <br />MaxLength: 255 <br />MinLength: 1 <br /> |
| `notTags` _[NeutronTag](#neutrontag) array_ | notTags is a list of tags to filter by. If specified, resources which<br />contain all of the given tags will be excluded from the result. |  | MaxItems: 32 <br />MaxLength: 255 <br />MinLength: 1 <br /> |
| `notTagsAny` _[NeutronTag](#neutrontag) array_ | notTagsAny is a list of tags to filter by. If specified, resources<br />which contain any of the given tags will be excluded from the result. |  | MaxItems: 32 <br />MaxLength: 255 <br />MinLength: 1 <br /> |


#### FilterByServerTags







_Appears in:_
- [ServerFilter](#serverfilter)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `tags` _[ServerTag](#servertag) array_ | tags is a list of tags to filter by. If specified, the resource must<br />have all of the tags specified to be included in the result. |  | MaxItems: 32 <br />MaxLength: 80 <br />MinLength: 1 <br /> |
| `tagsAny` _[ServerTag](#servertag) array_ | tagsAny is a list of tags to filter by. If specified, the resource<br />must have at least one of the tags specified to be included in the<br />result. |  | MaxItems: 32 <br />MaxLength: 80 <br />MinLength: 1 <br /> |
| `notTags` _[ServerTag](#servertag) array_ | notTags is a list of tags to filter by. If specified, resources which<br />contain all of the given tags will be excluded from the result. |  | MaxItems: 32 <br />MaxLength: 80 <br />MinLength: 1 <br /> |
| `notTagsAny` _[ServerTag](#servertag) array_ | notTagsAny is a list of tags to filter by. If specified, resources<br />which contain any of the given tags will be excluded from the result. |  | MaxItems: 32 <br />MaxLength: 80 <br />MinLength: 1 <br /> |


#### FixedIPStatus







_Appears in:_
- [PortResourceStatus](#portresourcestatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `ip` _string_ | ip contains a fixed IP address assigned to the port. |  | MaxLength: 1024 <br /> |
| `subnetID` _string_ | subnetID is the ID of the subnet this IP is allocated from. |  | MaxLength: 1024 <br /> |


#### Flavor



Flavor is the Schema for an ORC resource.



_Appears in:_
- [FlavorList](#flavorlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openstack.k-orc.cloud/v1alpha1` | | |
| `kind` _string_ | `Flavor` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[FlavorSpec](#flavorspec)_ | spec specifies the desired state of the resource. |  |  |
| `status` _[FlavorStatus](#flavorstatus)_ | status defines the observed state of the resource. |  |  |


#### FlavorFilter



FlavorFilter defines an existing resource by its properties

_Validation:_
- MinProperties: 1

_Appears in:_
- [FlavorImport](#flavorimport)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _[OpenStackName](#openstackname)_ | name of the existing resource |  | MaxLength: 255 <br />MinLength: 1 <br />Pattern: `^[^,]+$` <br /> |
| `ram` _integer_ | ram is the memory of the flavor, measured in MB. |  | Minimum: 1 <br /> |
| `vcpus` _integer_ | vcpus is the number of vcpus for the flavor. |  | Minimum: 1 <br /> |
| `disk` _integer_ | disk is the size of the root disk in GiB. |  | Minimum: 0 <br /> |


#### FlavorImport



FlavorImport specifies an existing resource which will be imported instead of
creating a new one

_Validation:_
- MaxProperties: 1
- MinProperties: 1

_Appears in:_
- [FlavorSpec](#flavorspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `id` _string_ | id contains the unique identifier of an existing OpenStack resource. Note<br />that when specifying an import by ID, the resource MUST already exist.<br />The ORC object will enter an error state if the resource does not exist. |  | Format: uuid <br /> |
| `filter` _[FlavorFilter](#flavorfilter)_ | filter contains a resource query which is expected to return a single<br />result. The controller will continue to retry if filter returns no<br />results. If filter returns multiple results the controller will set an<br />error state and will not continue to retry. |  | MinProperties: 1 <br /> |


#### FlavorList



FlavorList contains a list of Flavor.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openstack.k-orc.cloud/v1alpha1` | | |
| `kind` _string_ | `FlavorList` | | |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[Flavor](#flavor) array_ | items contains a list of Flavor. |  |  |


#### FlavorResourceSpec



FlavorResourceSpec contains the desired state of a flavor



_Appears in:_
- [FlavorSpec](#flavorspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _[OpenStackName](#openstackname)_ | name will be the name of the created resource. If not specified, the<br />name of the ORC object will be used. |  | MaxLength: 255 <br />MinLength: 1 <br />Pattern: `^[^,]+$` <br /> |
| `description` _string_ | description contains a free form description of the flavor. |  | MaxLength: 65535 <br />MinLength: 1 <br /> |
| `ram` _integer_ | ram is the memory of the flavor, measured in MB. |  | Minimum: 1 <br /> |
| `vcpus` _integer_ | vcpus is the number of vcpus for the flavor. |  | Minimum: 1 <br /> |
| `disk` _integer_ | disk is the size of the root disk that will be created in GiB. If 0<br />the root disk will be set to exactly the size of the image used to<br />deploy the instance. However, in this case the scheduler cannot<br />select the compute host based on the virtual image size. Therefore,<br />0 should only be used for volume booted instances or for testing<br />purposes. Volume-backed instances can be enforced for flavors with<br />zero root disk via the<br />os_compute_api:servers:create:zero_disk_flavor policy rule. |  | Minimum: 0 <br /> |
| `swap` _integer_ | swap is the size of a dedicated swap disk that will be allocated, in<br />MiB. If 0 (the default), no dedicated swap disk will be created. |  | Minimum: 0 <br /> |
| `isPublic` _boolean_ | isPublic flags a flavor as being available to all projects or not. |  |  |
| `ephemeral` _integer_ | ephemeral is the size of the ephemeral disk that will be created, in GiB.<br />Ephemeral disks may be written over on server state changes. So should only<br />be used as a scratch space for applications that are aware of its<br />limitations. Defaults to 0. |  | Minimum: 0 <br /> |


#### FlavorResourceStatus



FlavorResourceStatus represents the observed state of the resource.



_Appears in:_
- [FlavorStatus](#flavorstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | name is a Human-readable name for the flavor. Might not be unique. |  | MaxLength: 1024 <br /> |
| `description` _string_ | description is a human-readable description for the resource. |  | MaxLength: 65535 <br /> |
| `ram` _integer_ | ram is the memory of the flavor, measured in MB. |  |  |
| `vcpus` _integer_ | vcpus is the number of vcpus for the flavor. |  |  |
| `disk` _integer_ | disk is the size of the root disk that will be created in GiB. |  |  |
| `swap` _integer_ | swap is the size of a dedicated swap disk that will be allocated, in<br />MiB. |  |  |
| `isPublic` _boolean_ | isPublic flags a flavor as being available to all projects or not. |  |  |
| `ephemeral` _integer_ | ephemeral is the size of the ephemeral disk, in GiB. |  |  |


#### FlavorSpec



FlavorSpec defines the desired state of an ORC object.



_Appears in:_
- [Flavor](#flavor)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `import` _[FlavorImport](#flavorimport)_ | import refers to an existing OpenStack resource which will be imported instead of<br />creating a new one. |  | MaxProperties: 1 <br />MinProperties: 1 <br /> |
| `resource` _[FlavorResourceSpec](#flavorresourcespec)_ | resource specifies the desired state of the resource.<br /><br />resource may not be specified if the management policy is `unmanaged`.<br /><br />resource must be specified if the management policy is `managed`. |  |  |
| `managementPolicy` _[ManagementPolicy](#managementpolicy)_ | managementPolicy defines how ORC will treat the object. Valid values are<br />`managed`: ORC will create, update, and delete the resource; `unmanaged`:<br />ORC will import an existing resource, and will not apply updates to it or<br />delete it. | managed | Enum: [managed unmanaged] <br /> |
| `managedOptions` _[ManagedOptions](#managedoptions)_ | managedOptions specifies options which may be applied to managed objects. |  |  |
| `cloudCredentialsRef` _[CloudCredentialsReference](#cloudcredentialsreference)_ | cloudCredentialsRef points to a secret containing OpenStack credentials |  |  |


#### FlavorStatus



FlavorStatus defines the observed state of an ORC resource.



_Appears in:_
- [Flavor](#flavor)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#condition-v1-meta) array_ | conditions represents the observed status of the object.<br />Known .status.conditions.type are: "Available", "Progressing"<br /><br />Available represents the availability of the OpenStack resource. If it is<br />true then the resource is ready for use.<br /><br />Progressing indicates whether the controller is still attempting to<br />reconcile the current state of the OpenStack resource to the desired<br />state. Progressing will be False either because the desired state has<br />been achieved, or because some terminal error prevents it from ever being<br />achieved and the controller is no longer attempting to reconcile. If<br />Progressing is True, an observer waiting on the resource should continue<br />to wait. |  | MaxItems: 32 <br /> |
| `id` _string_ | id is the unique identifier of the OpenStack resource. |  |  |
| `resource` _[FlavorResourceStatus](#flavorresourcestatus)_ | resource contains the observed state of the OpenStack resource. |  |  |


#### FloatingIP



FloatingIP is the Schema for an ORC resource.



_Appears in:_
- [FloatingIPList](#floatingiplist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openstack.k-orc.cloud/v1alpha1` | | |
| `kind` _string_ | `FloatingIP` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[FloatingIPSpec](#floatingipspec)_ | spec specifies the desired state of the resource. |  |  |
| `status` _[FloatingIPStatus](#floatingipstatus)_ | status defines the observed state of the resource. |  |  |


#### FloatingIPFilter



FloatingIPFilter specifies a query to select an OpenStack floatingip. At least one property must be set.

_Validation:_
- MinProperties: 1

_Appears in:_
- [FloatingIPImport](#floatingipimport)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `floatingIP` _[IPvAny](#ipvany)_ | floatingIP is the floatingip address. |  | MaxLength: 45 <br />MinLength: 1 <br /> |
| `description` _[NeutronDescription](#neutrondescription)_ | description of the existing resource |  | MaxLength: 255 <br />MinLength: 1 <br /> |
| `floatingNetworkRef` _[KubernetesNameRef](#kubernetesnameref)_ | floatingNetworkRef is a reference to the ORC Network which this resource is associated with. |  | MaxLength: 253 <br />MinLength: 1 <br /> |
| `portRef` _[KubernetesNameRef](#kubernetesnameref)_ | portRef is a reference to the ORC Port which this resource is associated with. |  | MaxLength: 253 <br />MinLength: 1 <br /> |
| `projectRef` _[KubernetesNameRef](#kubernetesnameref)_ | projectRef is a reference to the ORC Project this resource is associated with.<br />Typically, only used by admin. |  | MaxLength: 253 <br />MinLength: 1 <br /> |
| `status` _string_ | status is the status of the floatingip. |  | MaxLength: 1024 <br /> |
| `tags` _[NeutronTag](#neutrontag) array_ | tags is a list of tags to filter by. If specified, the resource must<br />have all of the tags specified to be included in the result. |  | MaxItems: 32 <br />MaxLength: 255 <br />MinLength: 1 <br /> |
| `tagsAny` _[NeutronTag](#neutrontag) array_ | tagsAny is a list of tags to filter by. If specified, the resource<br />must have at least one of the tags specified to be included in the<br />result. |  | MaxItems: 32 <br />MaxLength: 255 <br />MinLength: 1 <br /> |
| `notTags` _[NeutronTag](#neutrontag) array_ | notTags is a list of tags to filter by. If specified, resources which<br />contain all of the given tags will be excluded from the result. |  | MaxItems: 32 <br />MaxLength: 255 <br />MinLength: 1 <br /> |
| `notTagsAny` _[NeutronTag](#neutrontag) array_ | notTagsAny is a list of tags to filter by. If specified, resources<br />which contain any of the given tags will be excluded from the result. |  | MaxItems: 32 <br />MaxLength: 255 <br />MinLength: 1 <br /> |


#### FloatingIPImport



FloatingIPImport specifies an existing resource which will be imported instead of
creating a new one

_Validation:_
- MaxProperties: 1
- MinProperties: 1

_Appears in:_
- [FloatingIPSpec](#floatingipspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `id` _string_ | id contains the unique identifier of an existing OpenStack resource. Note<br />that when specifying an import by ID, the resource MUST already exist.<br />The ORC object will enter an error state if the resource does not exist. |  | Format: uuid <br /> |
| `filter` _[FloatingIPFilter](#floatingipfilter)_ | filter contains a resource query which is expected to return a single<br />result. The controller will continue to retry if filter returns no<br />results. If filter returns multiple results the controller will set an<br />error state and will not continue to retry. |  | MinProperties: 1 <br /> |


#### FloatingIPList



FloatingIPList contains a list of FloatingIP.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openstack.k-orc.cloud/v1alpha1` | | |
| `kind` _string_ | `FloatingIPList` | | |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[FloatingIP](#floatingip) array_ | items contains a list of FloatingIP. |  |  |


#### FloatingIPResourceSpec



FloatingIPResourceSpec contains the desired state of a floating IP



_Appears in:_
- [FloatingIPSpec](#floatingipspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `description` _[NeutronDescription](#neutrondescription)_ | description is a human-readable description for the resource. |  | MaxLength: 255 <br />MinLength: 1 <br /> |
| `tags` _[NeutronTag](#neutrontag) array_ | tags is a list of tags which will be applied to the floatingip. |  | MaxItems: 32 <br />MaxLength: 255 <br />MinLength: 1 <br /> |
| `floatingNetworkRef` _[KubernetesNameRef](#kubernetesnameref)_ | floatingNetworkRef references the network to which the floatingip is associated. |  | MaxLength: 253 <br />MinLength: 1 <br /> |
| `floatingSubnetRef` _[KubernetesNameRef](#kubernetesnameref)_ | floatingSubnetRef references the subnet to which the floatingip is associated. |  | MaxLength: 253 <br />MinLength: 1 <br /> |
| `floatingIP` _[IPvAny](#ipvany)_ | floatingIP is the IP that will be assigned to the floatingip. If not set, it will<br />be assigned automatically. |  | MaxLength: 45 <br />MinLength: 1 <br /> |
| `portRef` _[KubernetesNameRef](#kubernetesnameref)_ | portRef is a reference to the ORC Port which this resource is associated with. |  | MaxLength: 253 <br />MinLength: 1 <br /> |
| `fixedIP` _[IPvAny](#ipvany)_ | fixedIP is the IP address of the port to which the floatingip is associated. |  | MaxLength: 45 <br />MinLength: 1 <br /> |
| `projectRef` _[KubernetesNameRef](#kubernetesnameref)_ | projectRef is a reference to the ORC Project this resource is associated with.<br />Typically, only used by admin. |  | MaxLength: 253 <br />MinLength: 1 <br /> |


#### FloatingIPResourceStatus







_Appears in:_
- [FloatingIPStatus](#floatingipstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `description` _string_ | description is a human-readable description for the resource. |  | MaxLength: 1024 <br /> |
| `floatingNetworkID` _string_ | floatingNetworkID is the ID of the network to which the floatingip is associated. |  | MaxLength: 1024 <br /> |
| `floatingIP` _string_ | floatingIP is the IP address of the floatingip. |  | MaxLength: 1024 <br /> |
| `portID` _string_ | portID is the ID of the port to which the floatingip is associated. |  | MaxLength: 1024 <br /> |
| `fixedIP` _string_ | fixedIP is the IP address of the port to which the floatingip is associated. |  | MaxLength: 1024 <br /> |
| `tenantID` _string_ | tenantID is the project owner of the resource. |  | MaxLength: 1024 <br /> |
| `projectID` _string_ | projectID is the project owner of the resource. |  | MaxLength: 1024 <br /> |
| `status` _string_ | status indicates the current status of the resource. |  | MaxLength: 1024 <br /> |
| `routerID` _string_ | routerID is the ID of the router to which the floatingip is associated. |  | MaxLength: 1024 <br /> |
| `tags` _string array_ | tags is the list of tags on the resource. |  | MaxItems: 32 <br /> |
| `createdAt` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta)_ | createdAt shows the date and time when the resource was created. The date and time stamp format is ISO 8601 |  |  |
| `updatedAt` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta)_ | updatedAt shows the date and time when the resource was updated. The date and time stamp format is ISO 8601 |  |  |
| `revisionNumber` _integer_ | revisionNumber optionally set via extensions/standard-attr-revisions |  |  |


#### FloatingIPSpec



FloatingIPSpec defines the desired state of an ORC object.



_Appears in:_
- [FloatingIP](#floatingip)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `import` _[FloatingIPImport](#floatingipimport)_ | import refers to an existing OpenStack resource which will be imported instead of<br />creating a new one. |  | MaxProperties: 1 <br />MinProperties: 1 <br /> |
| `resource` _[FloatingIPResourceSpec](#floatingipresourcespec)_ | resource specifies the desired state of the resource.<br /><br />resource may not be specified if the management policy is `unmanaged`.<br /><br />resource must be specified if the management policy is `managed`. |  |  |
| `managementPolicy` _[ManagementPolicy](#managementpolicy)_ | managementPolicy defines how ORC will treat the object. Valid values are<br />`managed`: ORC will create, update, and delete the resource; `unmanaged`:<br />ORC will import an existing resource, and will not apply updates to it or<br />delete it. | managed | Enum: [managed unmanaged] <br /> |
| `managedOptions` _[ManagedOptions](#managedoptions)_ | managedOptions specifies options which may be applied to managed objects. |  |  |
| `cloudCredentialsRef` _[CloudCredentialsReference](#cloudcredentialsreference)_ | cloudCredentialsRef points to a secret containing OpenStack credentials |  |  |


#### FloatingIPStatus



FloatingIPStatus defines the observed state of an ORC resource.



_Appears in:_
- [FloatingIP](#floatingip)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#condition-v1-meta) array_ | conditions represents the observed status of the object.<br />Known .status.conditions.type are: "Available", "Progressing"<br /><br />Available represents the availability of the OpenStack resource. If it is<br />true then the resource is ready for use.<br /><br />Progressing indicates whether the controller is still attempting to<br />reconcile the current state of the OpenStack resource to the desired<br />state. Progressing will be False either because the desired state has<br />been achieved, or because some terminal error prevents it from ever being<br />achieved and the controller is no longer attempting to reconcile. If<br />Progressing is True, an observer waiting on the resource should continue<br />to wait. |  | MaxItems: 32 <br /> |
| `id` _string_ | id is the unique identifier of the OpenStack resource. |  |  |
| `resource` _[FloatingIPResourceStatus](#floatingipresourcestatus)_ | resource contains the observed state of the OpenStack resource. |  |  |


#### HostRoute







_Appears in:_
- [SubnetResourceSpec](#subnetresourcespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `destination` _[CIDR](#cidr)_ | destination for the additional route. |  | Format: cidr <br />MaxLength: 49 <br />MinLength: 1 <br /> |
| `nextHop` _[IPvAny](#ipvany)_ | nextHop for the additional route. |  | MaxLength: 45 <br />MinLength: 1 <br /> |


#### HostRouteStatus







_Appears in:_
- [SubnetResourceStatus](#subnetresourcestatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `destination` _string_ | destination for the additional route. |  | MaxLength: 1024 <br /> |
| `nextHop` _string_ | nextHop for the additional route. |  | MaxLength: 1024 <br /> |


#### IPVersion

_Underlying type:_ _integer_



_Validation:_
- Enum: [4 6]

_Appears in:_
- [SubnetFilter](#subnetfilter)
- [SubnetResourceSpec](#subnetresourcespec)



#### IPv6AddressMode

_Underlying type:_ _string_



_Validation:_
- Enum: [slaac dhcpv6-stateful dhcpv6-stateless]

_Appears in:_
- [IPv6Options](#ipv6options)



#### IPv6Options





_Validation:_
- MinProperties: 1

_Appears in:_
- [SubnetFilter](#subnetfilter)
- [SubnetResourceSpec](#subnetresourcespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `addressMode` _[IPv6AddressMode](#ipv6addressmode)_ | addressMode specifies mechanisms for assigning IPv6 IP addresses. |  | Enum: [slaac dhcpv6-stateful dhcpv6-stateless] <br /> |
| `raMode` _[IPv6RAMode](#ipv6ramode)_ | raMode specifies the IPv6 router advertisement mode. It specifies whether<br />the networking service should transmit ICMPv6 packets. |  | Enum: [slaac dhcpv6-stateful dhcpv6-stateless] <br /> |


#### IPv6RAMode

_Underlying type:_ _string_



_Validation:_
- Enum: [slaac dhcpv6-stateful dhcpv6-stateless]

_Appears in:_
- [IPv6Options](#ipv6options)



#### IPvAny

_Underlying type:_ _string_



_Validation:_
- MaxLength: 45
- MinLength: 1

_Appears in:_
- [Address](#address)
- [AllocationPool](#allocationpool)
- [AllowedAddressPair](#allowedaddresspair)
- [FloatingIPFilter](#floatingipfilter)
- [FloatingIPResourceSpec](#floatingipresourcespec)
- [HostRoute](#hostroute)
- [SubnetFilter](#subnetfilter)
- [SubnetGateway](#subnetgateway)
- [SubnetResourceSpec](#subnetresourcespec)



#### Image



Image is the Schema for an ORC resource.



_Appears in:_
- [ImageList](#imagelist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openstack.k-orc.cloud/v1alpha1` | | |
| `kind` _string_ | `Image` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[ImageSpec](#imagespec)_ | spec specifies the desired state of the resource. |  |  |
| `status` _[ImageStatus](#imagestatus)_ | status defines the observed state of the resource. |  |  |


#### ImageCompression

_Underlying type:_ _string_



_Validation:_
- Enum: [xz gz bz2]

_Appears in:_
- [ImageContentSourceDownload](#imagecontentsourcedownload)

| Field | Description |
| --- | --- |
| `xz` |  |
| `gz` |  |
| `bz2` |  |


#### ImageContainerFormat

_Underlying type:_ _string_



_Validation:_
- Enum: [ami ari aki bare ovf ova docker compressed]

_Appears in:_
- [ImageContent](#imagecontent)

| Field | Description |
| --- | --- |
| `aki` |  |
| `ami` |  |
| `ari` |  |
| `bare` |  |
| `compressed` |  |
| `docker` |  |
| `ova` |  |
| `ovf` |  |


#### ImageContent







_Appears in:_
- [ImageResourceSpec](#imageresourcespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `containerFormat` _[ImageContainerFormat](#imagecontainerformat)_ | containerFormat is the format of the image container.<br />qcow2 and raw images do not usually have a container. This is specified as "bare", which is also the default.<br />Permitted values are ami, ari, aki, bare, compressed, ovf, ova, and docker. | bare | Enum: [ami ari aki bare ovf ova docker compressed] <br /> |
| `diskFormat` _[ImageDiskFormat](#imagediskformat)_ | diskFormat is the format of the disk image.<br />Normal values are "qcow2", or "raw". Glance may be configured to support others. |  | Enum: [ami ari aki vhd vhdx vmdk raw qcow2 vdi ploop iso] <br /> |
| `download` _[ImageContentSourceDownload](#imagecontentsourcedownload)_ | download describes how to obtain image data by downloading it from a URL.<br />Must be set when creating a managed image. |  |  |


#### ImageContentSourceDownload







_Appears in:_
- [ImageContent](#imagecontent)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `url` _string_ | url containing image data |  | Format: uri <br />MaxLength: 2048 <br /> |
| `decompress` _[ImageCompression](#imagecompression)_ | decompress specifies that the source data must be decompressed with the<br />given compression algorithm before being stored. Specifying Decompress<br />will disable the use of Glance's web-download, as web-download cannot<br />currently deterministically decompress downloaded content. |  | Enum: [xz gz bz2] <br /> |
| `hash` _[ImageHash](#imagehash)_ | hash is a hash which will be used to verify downloaded data, i.e.<br />before any decompression. If not specified, no hash verification will be<br />performed. Specifying a Hash will disable the use of Glance's<br />web-download, as web-download cannot currently deterministically verify<br />the hash of downloaded content. |  |  |


#### ImageDiskFormat

_Underlying type:_ _string_



_Validation:_
- Enum: [ami ari aki vhd vhdx vmdk raw qcow2 vdi ploop iso]

_Appears in:_
- [ImageContent](#imagecontent)

| Field | Description |
| --- | --- |
| `ami` |  |
| `ari` |  |
| `aki` |  |
| `vhd` |  |
| `vhdx` |  |
| `vmdk` |  |
| `raw` |  |
| `qcow2` |  |
| `vdi` |  |
| `ploop` |  |
| `iso` |  |


#### ImageFilter



ImageFilter defines a Glance query

_Validation:_
- MinProperties: 1

_Appears in:_
- [ImageImport](#imageimport)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _[OpenStackName](#openstackname)_ | name specifies the name of a Glance image |  | MaxLength: 255 <br />MinLength: 1 <br />Pattern: `^[^,]+$` <br /> |
| `visibility` _[ImageVisibility](#imagevisibility)_ | visibility specifies the visibility of a Glance image. |  | Enum: [public private shared community] <br /> |
| `tags` _[ImageTag](#imagetag) array_ | tags is the list of tags on the resource. |  | MaxItems: 32 <br />MaxLength: 255 <br />MinLength: 1 <br /> |


#### ImageHWBus

_Underlying type:_ _string_

ImageHWBus is a type of hardware bus.


Permitted values are scsi, virtio, uml, xen, ide, usb, and lxc.

_Validation:_
- Enum: [scsi virtio uml xen ide usb lxc]

_Appears in:_
- [ImagePropertiesHardware](#imagepropertieshardware)



#### ImageHash







_Appears in:_
- [ImageContentSourceDownload](#imagecontentsourcedownload)
- [ImageResourceStatus](#imageresourcestatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `algorithm` _[ImageHashAlgorithm](#imagehashalgorithm)_ | algorithm is the hash algorithm used to generate value. |  | Enum: [md5 sha1 sha256 sha512] <br /> |
| `value` _string_ | value is the hash of the image data using Algorithm. It must be hex encoded using lowercase letters. |  | MaxLength: 1024 <br />MinLength: 1 <br />Pattern: `^[0-9a-f]+$` <br /> |


#### ImageHashAlgorithm

_Underlying type:_ _string_



_Validation:_
- Enum: [md5 sha1 sha256 sha512]

_Appears in:_
- [ImageHash](#imagehash)

| Field | Description |
| --- | --- |
| `md5` |  |
| `sha1` |  |
| `sha256` |  |
| `sha512` |  |


#### ImageImport



ImageImport specifies an existing resource which will be imported instead of
creating a new one

_Validation:_
- MaxProperties: 1
- MinProperties: 1

_Appears in:_
- [ImageSpec](#imagespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `id` _string_ | id contains the unique identifier of an existing OpenStack resource. Note<br />that when specifying an import by ID, the resource MUST already exist.<br />The ORC object will enter an error state if the resource does not exist. |  | Format: uuid <br /> |
| `filter` _[ImageFilter](#imagefilter)_ | filter contains a resource query which is expected to return a single<br />result. The controller will continue to retry if filter returns no<br />results. If filter returns multiple results the controller will set an<br />error state and will not continue to retry. |  | MinProperties: 1 <br /> |


#### ImageList



ImageList contains a list of Image.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openstack.k-orc.cloud/v1alpha1` | | |
| `kind` _string_ | `ImageList` | | |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[Image](#image) array_ | items contains a list of Image. |  |  |


#### ImageProperties







_Appears in:_
- [ImageResourceSpec](#imageresourcespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `architecture` _string_ | architecture is the CPU architecture that must be supported by the hypervisor. |  | Enum: [aarch64 alpha armv7l cris i686 ia64 lm32 m68k microblaze microblazeel mips mipsel mips64 mips64el openrisc parisc parisc64 ppc ppc64 ppcemb s390 s390x sh4 sh4eb sparc sparc64 unicore32 x86_64 xtensa xtensaeb] <br /> |
| `hypervisorType` _string_ | hypervisorType is the hypervisor type |  | Enum: [hyperv ironic lxc qemu uml vmware xen] <br /> |
| `minDiskGB` _integer_ | minDiskGB is the minimum amount of disk space in GB that is required to boot the image |  | Minimum: 1 <br /> |
| `minMemoryMB` _integer_ | minMemoryMB is the minimum amount of RAM in MB that is required to boot the image. |  | Minimum: 1 <br /> |
| `hardware` _[ImagePropertiesHardware](#imagepropertieshardware)_ | hardware is a set of properties which control the virtual hardware<br />created by Nova. |  |  |
| `operatingSystem` _[ImagePropertiesOperatingSystem](#imagepropertiesoperatingsystem)_ | operatingSystem is a set of properties that specify and influence the behavior<br />of the operating system within the virtual machine. |  |  |


#### ImagePropertiesHardware







_Appears in:_
- [ImageProperties](#imageproperties)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `cpuSockets` _integer_ | cpuSockets is the preferred number of sockets to expose to the guest |  | Minimum: 1 <br /> |
| `cpuCores` _integer_ | cpuCores is the preferred number of cores to expose to the guest |  | Minimum: 1 <br /> |
| `cpuThreads` _integer_ | cpuThreads is the preferred number of threads to expose to the guest |  | Minimum: 1 <br /> |
| `cpuPolicy` _string_ | cpuPolicy is used to pin the virtual CPUs (vCPUs) of instances to the<br />host's physical CPU cores (pCPUs). Host aggregates should be used to<br />separate these pinned instances from unpinned instances as the latter<br />will not respect the resourcing requirements of the former.<br /><br />Permitted values are shared (the default), and dedicated.<br /><br />shared: The guest vCPUs will be allowed to freely float across host<br />pCPUs, albeit potentially constrained by NUMA policy.<br /><br />dedicated: The guest vCPUs will be strictly pinned to a set of host<br />pCPUs. In the absence of an explicit vCPU topology request, the<br />drivers typically expose all vCPUs as sockets with one core and one<br />thread. When strict CPU pinning is in effect the guest CPU topology<br />will be setup to match the topology of the CPUs to which it is<br />pinned. This option implies an overcommit ratio of 1.0. For example,<br />if a two vCPU guest is pinned to a single host core with two threads,<br />then the guest will get a topology of one socket, one core, two<br />threads. |  | Enum: [shared dedicated] <br /> |
| `cpuThreadPolicy` _string_ | cpuThreadPolicy further refines a CPUPolicy of 'dedicated' by stating<br />how hardware CPU threads in a simultaneous multithreading-based (SMT)<br />architecture be used. SMT-based architectures include Intel<br />processors with Hyper-Threading technology. In these architectures,<br />processor cores share a number of components with one or more other<br />cores. Cores in such architectures are commonly referred to as<br />hardware threads, while the cores that a given core share components<br />with are known as thread siblings.<br /><br />Permitted values are prefer (the default), isolate, and require.<br /><br />prefer: The host may or may not have an SMT architecture. Where an<br />SMT architecture is present, thread siblings are preferred.<br /><br />isolate: The host must not have an SMT architecture or must emulate a<br />non-SMT architecture. If the host does not have an SMT architecture,<br />each vCPU is placed on a different core as expected. If the host does<br />have an SMT architecture - that is, one or more cores have thread<br />siblings - then each vCPU is placed on a different physical core. No<br />vCPUs from other guests are placed on the same core. All but one<br />thread sibling on each utilized core is therefore guaranteed to be<br />unusable.<br /><br />require: The host must have an SMT architecture. Each vCPU is<br />allocated on thread siblings. If the host does not have an SMT<br />architecture, then it is not used. If the host has an SMT<br />architecture, but not enough cores with free thread siblings are<br />available, then scheduling fails. |  | Enum: [prefer isolate require] <br /> |
| `cdromBus` _[ImageHWBus](#imagehwbus)_ | cdromBus specifies the type of disk controller to attach CD-ROM devices to. |  | Enum: [scsi virtio uml xen ide usb lxc] <br /> |
| `diskBus` _[ImageHWBus](#imagehwbus)_ | diskBus specifies the type of disk controller to attach disk devices to. |  | Enum: [scsi virtio uml xen ide usb lxc] <br /> |
| `scsiModel` _string_ | scsiModel enables the use of VirtIO SCSI (virtio-scsi) to provide<br />block device access for compute instances; by default, instances use<br />VirtIO Block (virtio-blk). VirtIO SCSI is a para-virtualized SCSI<br />controller device that provides improved scalability and performance,<br />and supports advanced SCSI hardware.<br /><br />The only permitted value is virtio-scsi. |  | Enum: [virtio-scsi] <br /> |
| `vifModel` _string_ | vifModel specifies the model of virtual network interface device to use.<br /><br />Permitted values are e1000, e1000e, ne2k_pci, pcnet, rtl8139, virtio,<br />and vmxnet3. |  | Enum: [e1000 e1000e ne2k_pci pcnet rtl8139 virtio vmxnet3] <br /> |
| `rngModel` _string_ | rngModel adds a random-number generator device to the image’s instances.<br />This image property by itself does not guarantee that a hardware RNG will be used;<br />it expresses a preference that may or may not be satisfied depending upon Nova configuration. |  | MaxLength: 255 <br /> |
| `qemuGuestAgent` _boolean_ | qemuGuestAgent enables QEMU guest agent. |  |  |


#### ImagePropertiesOperatingSystem







_Appears in:_
- [ImageProperties](#imageproperties)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `distro` _string_ | distro is the common name of the operating system distribution in lowercase. |  | Enum: [arch centos debian fedora freebsd gentoo mandrake mandriva mes msdos netbsd netware openbsd opensolaris opensuse rocky rhel sled ubuntu windows] <br /> |
| `version` _string_ | version is the operating system version as specified by the distributor. |  | MaxLength: 255 <br /> |


#### ImageResourceSpec



ImageResourceSpec contains the desired state of a Glance image



_Appears in:_
- [ImageSpec](#imagespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _[OpenStackName](#openstackname)_ | name will be the name of the created Glance image. If not specified, the<br />name of the Image object will be used. |  | MaxLength: 255 <br />MinLength: 1 <br />Pattern: `^[^,]+$` <br /> |
| `protected` _boolean_ | protected specifies that the image is protected from deletion.<br />If not specified, the default is false. |  |  |
| `tags` _[ImageTag](#imagetag) array_ | tags is a list of tags which will be applied to the image. A tag has a maximum length of 255 characters. |  | MaxItems: 32 <br />MaxLength: 255 <br />MinLength: 1 <br /> |
| `visibility` _[ImageVisibility](#imagevisibility)_ | visibility of the image |  | Enum: [public private shared community] <br /> |
| `properties` _[ImageProperties](#imageproperties)_ | properties is metadata available to consumers of the image |  |  |
| `content` _[ImageContent](#imagecontent)_ | content specifies how to obtain the image content. |  |  |


#### ImageResourceStatus



ImageResourceStatus represents the observed state of a Glance image



_Appears in:_
- [ImageStatus](#imagestatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | name is a Human-readable name for the image. Might not be unique. |  | MaxLength: 1024 <br /> |
| `status` _string_ | status is the image status as reported by Glance |  | MaxLength: 1024 <br /> |
| `protected` _boolean_ | protected specifies that the image is protected from deletion. |  |  |
| `visibility` _string_ | visibility of the image |  | MaxLength: 1024 <br /> |
| `hash` _[ImageHash](#imagehash)_ | hash is the hash of the image data published by Glance. Note that this is<br />a hash of the data stored internally by Glance, which will have been<br />decompressed and potentially format converted depending on server-side<br />configuration which is not visible to clients. It is expected that this<br />hash will usually differ from the download hash. |  |  |
| `sizeB` _integer_ | sizeB is the size of the image data, in bytes |  |  |
| `virtualSizeB` _integer_ | virtualSizeB is the size of the disk the image data represents, in bytes |  |  |
| `tags` _string array_ | tags is the list of tags on the resource. |  | MaxItems: 32 <br /> |


#### ImageSpec



ImageSpec defines the desired state of an ORC object.



_Appears in:_
- [Image](#image)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `import` _[ImageImport](#imageimport)_ | import refers to an existing OpenStack resource which will be imported instead of<br />creating a new one. |  | MaxProperties: 1 <br />MinProperties: 1 <br /> |
| `resource` _[ImageResourceSpec](#imageresourcespec)_ | resource specifies the desired state of the resource.<br /><br />resource may not be specified if the management policy is `unmanaged`.<br /><br />resource must be specified if the management policy is `managed`. |  |  |
| `managementPolicy` _[ManagementPolicy](#managementpolicy)_ | managementPolicy defines how ORC will treat the object. Valid values are<br />`managed`: ORC will create, update, and delete the resource; `unmanaged`:<br />ORC will import an existing resource, and will not apply updates to it or<br />delete it. | managed | Enum: [managed unmanaged] <br /> |
| `managedOptions` _[ManagedOptions](#managedoptions)_ | managedOptions specifies options which may be applied to managed objects. |  |  |
| `cloudCredentialsRef` _[CloudCredentialsReference](#cloudcredentialsreference)_ | cloudCredentialsRef points to a secret containing OpenStack credentials |  |  |


#### ImageStatus



ImageStatus defines the observed state of an ORC resource.



_Appears in:_
- [Image](#image)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#condition-v1-meta) array_ | conditions represents the observed status of the object.<br />Known .status.conditions.type are: "Available", "Progressing"<br /><br />Available represents the availability of the OpenStack resource. If it is<br />true then the resource is ready for use.<br /><br />Progressing indicates whether the controller is still attempting to<br />reconcile the current state of the OpenStack resource to the desired<br />state. Progressing will be False either because the desired state has<br />been achieved, or because some terminal error prevents it from ever being<br />achieved and the controller is no longer attempting to reconcile. If<br />Progressing is True, an observer waiting on the resource should continue<br />to wait. |  | MaxItems: 32 <br /> |
| `id` _string_ | id is the unique identifier of the OpenStack resource. |  |  |
| `resource` _[ImageResourceStatus](#imageresourcestatus)_ | resource contains the observed state of the OpenStack resource. |  |  |
| `downloadAttempts` _integer_ | downloadAttempts is the number of times the controller has attempted to download the image contents |  |  |


#### ImageStatusExtra







_Appears in:_
- [ImageStatus](#imagestatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `downloadAttempts` _integer_ | downloadAttempts is the number of times the controller has attempted to download the image contents |  |  |


#### ImageTag

_Underlying type:_ _string_



_Validation:_
- MaxLength: 255
- MinLength: 1

_Appears in:_
- [ImageFilter](#imagefilter)
- [ImageResourceSpec](#imageresourcespec)



#### ImageVisibility

_Underlying type:_ _string_



_Validation:_
- Enum: [public private shared community]

_Appears in:_
- [ImageFilter](#imagefilter)
- [ImageResourceSpec](#imageresourcespec)

| Field | Description |
| --- | --- |
| `public` |  |
| `private` |  |
| `shared` |  |
| `community` |  |


#### KeystoneName

_Underlying type:_ _string_



_Validation:_
- MaxLength: 64
- MinLength: 1

_Appears in:_
- [ProjectFilter](#projectfilter)
- [ProjectResourceSpec](#projectresourcespec)



#### KeystoneTag

_Underlying type:_ _string_



_Validation:_
- MaxLength: 255
- MinLength: 1

_Appears in:_
- [FilterByKeystoneTags](#filterbykeystonetags)
- [ProjectFilter](#projectfilter)
- [ProjectResourceSpec](#projectresourcespec)



#### KubernetesNameRef

_Underlying type:_ _string_



_Validation:_
- MaxLength: 253
- MinLength: 1

_Appears in:_
- [Address](#address)
- [ExternalGateway](#externalgateway)
- [FloatingIPFilter](#floatingipfilter)
- [FloatingIPResourceSpec](#floatingipresourcespec)
- [NetworkFilter](#networkfilter)
- [NetworkResourceSpec](#networkresourcespec)
- [PortFilter](#portfilter)
- [PortResourceSpec](#portresourcespec)
- [RouterFilter](#routerfilter)
- [RouterInterfaceSpec](#routerinterfacespec)
- [RouterResourceSpec](#routerresourcespec)
- [SecurityGroupFilter](#securitygroupfilter)
- [SecurityGroupResourceSpec](#securitygroupresourcespec)
- [ServerPortSpec](#serverportspec)
- [ServerResourceSpec](#serverresourcespec)
- [SubnetFilter](#subnetfilter)
- [SubnetResourceSpec](#subnetresourcespec)
- [UserDataSpec](#userdataspec)



#### MAC

_Underlying type:_ _string_



_Validation:_
- MaxLength: 17
- MinLength: 1

_Appears in:_
- [AllowedAddressPair](#allowedaddresspair)



#### MTU

_Underlying type:_ _integer_



_Validation:_
- Maximum: 9216
- Minimum: 68

_Appears in:_
- [NetworkResourceSpec](#networkresourcespec)



#### ManagedOptions







_Appears in:_
- [FlavorSpec](#flavorspec)
- [FloatingIPSpec](#floatingipspec)
- [ImageSpec](#imagespec)
- [NetworkSpec](#networkspec)
- [PortSpec](#portspec)
- [ProjectSpec](#projectspec)
- [RouterSpec](#routerspec)
- [SecurityGroupSpec](#securitygroupspec)
- [ServerGroupSpec](#servergroupspec)
- [ServerSpec](#serverspec)
- [SubnetSpec](#subnetspec)
- [VolumeSpec](#volumespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `onDelete` _[OnDelete](#ondelete)_ | onDelete specifies the behaviour of the controller when the ORC<br />object is deleted. Options are `delete` - delete the OpenStack resource;<br />`detach` - do not delete the OpenStack resource. If not specified, the<br />default is `delete`. | delete | Enum: [delete detach] <br /> |


#### ManagementPolicy

_Underlying type:_ _string_



_Validation:_
- Enum: [managed unmanaged]

_Appears in:_
- [FlavorSpec](#flavorspec)
- [FloatingIPSpec](#floatingipspec)
- [ImageSpec](#imagespec)
- [NetworkSpec](#networkspec)
- [PortSpec](#portspec)
- [ProjectSpec](#projectspec)
- [RouterSpec](#routerspec)
- [SecurityGroupSpec](#securitygroupspec)
- [ServerGroupSpec](#servergroupspec)
- [ServerSpec](#serverspec)
- [SubnetSpec](#subnetspec)
- [VolumeSpec](#volumespec)

| Field | Description |
| --- | --- |
| `managed` | ManagementPolicyManaged specifies that the controller will reconcile the<br />state of the referenced OpenStack resource with the state of the ORC<br />object.<br /> |
| `unmanaged` | ManagementPolicyUnmanaged specifies that the controller will expect the<br />resource to either exist already or to be created externally. The<br />controller will not make any changes to the referenced OpenStack<br />resource.<br /> |


#### Network



Network is the Schema for an ORC resource.



_Appears in:_
- [NetworkList](#networklist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openstack.k-orc.cloud/v1alpha1` | | |
| `kind` _string_ | `Network` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[NetworkSpec](#networkspec)_ | spec specifies the desired state of the resource. |  |  |
| `status` _[NetworkStatus](#networkstatus)_ | status defines the observed state of the resource. |  |  |


#### NetworkFilter



NetworkFilter defines an existing resource by its properties

_Validation:_
- MinProperties: 1

_Appears in:_
- [NetworkImport](#networkimport)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _[OpenStackName](#openstackname)_ | name of the existing resource |  | MaxLength: 255 <br />MinLength: 1 <br />Pattern: `^[^,]+$` <br /> |
| `description` _[NeutronDescription](#neutrondescription)_ | description of the existing resource |  | MaxLength: 255 <br />MinLength: 1 <br /> |
| `external` _boolean_ | external indicates whether the network has an external routing<br />facility that’s not managed by the networking service. |  |  |
| `projectRef` _[KubernetesNameRef](#kubernetesnameref)_ | projectRef is a reference to the ORC Project this resource is associated with.<br />Typically, only used by admin. |  | MaxLength: 253 <br />MinLength: 1 <br /> |
| `tags` _[NeutronTag](#neutrontag) array_ | tags is a list of tags to filter by. If specified, the resource must<br />have all of the tags specified to be included in the result. |  | MaxItems: 32 <br />MaxLength: 255 <br />MinLength: 1 <br /> |
| `tagsAny` _[NeutronTag](#neutrontag) array_ | tagsAny is a list of tags to filter by. If specified, the resource<br />must have at least one of the tags specified to be included in the<br />result. |  | MaxItems: 32 <br />MaxLength: 255 <br />MinLength: 1 <br /> |
| `notTags` _[NeutronTag](#neutrontag) array_ | notTags is a list of tags to filter by. If specified, resources which<br />contain all of the given tags will be excluded from the result. |  | MaxItems: 32 <br />MaxLength: 255 <br />MinLength: 1 <br /> |
| `notTagsAny` _[NeutronTag](#neutrontag) array_ | notTagsAny is a list of tags to filter by. If specified, resources<br />which contain any of the given tags will be excluded from the result. |  | MaxItems: 32 <br />MaxLength: 255 <br />MinLength: 1 <br /> |


#### NetworkImport



NetworkImport specifies an existing resource which will be imported instead of
creating a new one

_Validation:_
- MaxProperties: 1
- MinProperties: 1

_Appears in:_
- [NetworkSpec](#networkspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `id` _string_ | id contains the unique identifier of an existing OpenStack resource. Note<br />that when specifying an import by ID, the resource MUST already exist.<br />The ORC object will enter an error state if the resource does not exist. |  | Format: uuid <br /> |
| `filter` _[NetworkFilter](#networkfilter)_ | filter contains a resource query which is expected to return a single<br />result. The controller will continue to retry if filter returns no<br />results. If filter returns multiple results the controller will set an<br />error state and will not continue to retry. |  | MinProperties: 1 <br /> |


#### NetworkList



NetworkList contains a list of Network.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openstack.k-orc.cloud/v1alpha1` | | |
| `kind` _string_ | `NetworkList` | | |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[Network](#network) array_ | items contains a list of Network. |  |  |


#### NetworkResourceSpec



NetworkResourceSpec contains the desired state of a network



_Appears in:_
- [NetworkSpec](#networkspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _[OpenStackName](#openstackname)_ | name will be the name of the created resource. If not specified, the<br />name of the ORC object will be used. |  | MaxLength: 255 <br />MinLength: 1 <br />Pattern: `^[^,]+$` <br /> |
| `description` _[NeutronDescription](#neutrondescription)_ | description is a human-readable description for the resource. |  | MaxLength: 255 <br />MinLength: 1 <br /> |
| `tags` _[NeutronTag](#neutrontag) array_ | tags is a list of tags which will be applied to the network. |  | MaxItems: 64 <br />MaxLength: 255 <br />MinLength: 1 <br /> |
| `adminStateUp` _boolean_ | adminStateUp is the administrative state of the network, which is up (true) or down (false) |  |  |
| `dnsDomain` _[DNSDomain](#dnsdomain)_ | dnsDomain is the DNS domain of the network |  | MaxLength: 255 <br />MinLength: 1 <br />Pattern: `^[A-Za-z0-9]\{1,63\}(.[A-Za-z0-9-]\{1,63\})*(.[A-Za-z]\{2,63\})*.?$` <br /> |
| `mtu` _[MTU](#mtu)_ | mtu is the the maximum transmission unit value to address<br />fragmentation. Minimum value is 68 for IPv4, and 1280 for IPv6.<br />Defaults to 1500. |  | Maximum: 9216 <br />Minimum: 68 <br /> |
| `portSecurityEnabled` _boolean_ | portSecurityEnabled is the port security status of the network.<br />Valid values are enabled (true) and disabled (false). This value is<br />used as the default value of port_security_enabled field of a newly<br />created port. |  |  |
| `external` _boolean_ | external indicates whether the network has an external routing<br />facility that’s not managed by the networking service. |  |  |
| `shared` _boolean_ | shared indicates whether this resource is shared across all<br />projects. By default, only administrative users can change this<br />value. |  |  |
| `availabilityZoneHints` _[AvailabilityZoneHint](#availabilityzonehint) array_ | availabilityZoneHints is the availability zone candidate for the network. |  | MaxItems: 32 <br />MaxLength: 255 <br />MinLength: 1 <br /> |
| `projectRef` _[KubernetesNameRef](#kubernetesnameref)_ | projectRef is a reference to the ORC Project this resource is associated with.<br />Typically, only used by admin. |  | MaxLength: 253 <br />MinLength: 1 <br /> |


#### NetworkResourceStatus



NetworkResourceStatus represents the observed state of the resource.



_Appears in:_
- [NetworkStatus](#networkstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | name is a Human-readable name for the network. Might not be unique. |  | MaxLength: 1024 <br /> |
| `description` _string_ | description is a human-readable description for the resource. |  | MaxLength: 1024 <br /> |
| `projectID` _string_ | projectID is the project owner of the network. |  | MaxLength: 1024 <br /> |
| `status` _string_ | status indicates whether network is currently operational. Possible values<br />include `ACTIVE', `DOWN', `BUILD', or `ERROR'. Plug-ins might define<br />additional values. |  | MaxLength: 1024 <br /> |
| `tags` _string array_ | tags is the list of tags on the resource. |  | MaxItems: 32 <br /> |
| `createdAt` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta)_ | createdAt shows the date and time when the resource was created. The date and time stamp format is ISO 8601 |  |  |
| `updatedAt` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta)_ | updatedAt shows the date and time when the resource was updated. The date and time stamp format is ISO 8601 |  |  |
| `revisionNumber` _integer_ | revisionNumber optionally set via extensions/standard-attr-revisions |  |  |
| `adminStateUp` _boolean_ | adminStateUp is the administrative state of the network,<br />which is up (true) or down (false). |  |  |
| `availabilityZoneHints` _string array_ | availabilityZoneHints is the availability zone candidate for the<br />network. |  | MaxItems: 32 <br /> |
| `dnsDomain` _string_ | dnsDomain is the DNS domain of the network |  | MaxLength: 1024 <br /> |
| `mtu` _integer_ | mtu is the the maximum transmission unit value to address<br />fragmentation. Minimum value is 68 for IPv4, and 1280 for IPv6. |  |  |
| `portSecurityEnabled` _boolean_ | portSecurityEnabled is the port security status of the network.<br />Valid values are enabled (true) and disabled (false). This value is<br />used as the default value of port_security_enabled field of a newly<br />created port. |  |  |
| `provider` _[ProviderPropertiesStatus](#providerpropertiesstatus)_ | provider contains provider-network properties. |  |  |
| `external` _boolean_ | external defines whether the network may be used for creation of<br />floating IPs. Only networks with this flag may be an external<br />gateway for routers. The network must have an external routing<br />facility that is not managed by the networking service. If the<br />network is updated from external to internal the unused floating IPs<br />of this network are automatically deleted when extension<br />floatingip-autodelete-internal is present. |  |  |
| `shared` _boolean_ | shared specifies whether the network resource can be accessed by any<br />tenant. |  |  |
| `subnets` _string array_ | subnets associated with this network. |  | MaxItems: 32 <br /> |


#### NetworkSpec



NetworkSpec defines the desired state of an ORC object.



_Appears in:_
- [Network](#network)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `import` _[NetworkImport](#networkimport)_ | import refers to an existing OpenStack resource which will be imported instead of<br />creating a new one. |  | MaxProperties: 1 <br />MinProperties: 1 <br /> |
| `resource` _[NetworkResourceSpec](#networkresourcespec)_ | resource specifies the desired state of the resource.<br /><br />resource may not be specified if the management policy is `unmanaged`.<br /><br />resource must be specified if the management policy is `managed`. |  |  |
| `managementPolicy` _[ManagementPolicy](#managementpolicy)_ | managementPolicy defines how ORC will treat the object. Valid values are<br />`managed`: ORC will create, update, and delete the resource; `unmanaged`:<br />ORC will import an existing resource, and will not apply updates to it or<br />delete it. | managed | Enum: [managed unmanaged] <br /> |
| `managedOptions` _[ManagedOptions](#managedoptions)_ | managedOptions specifies options which may be applied to managed objects. |  |  |
| `cloudCredentialsRef` _[CloudCredentialsReference](#cloudcredentialsreference)_ | cloudCredentialsRef points to a secret containing OpenStack credentials |  |  |


#### NetworkStatus



NetworkStatus defines the observed state of an ORC resource.



_Appears in:_
- [Network](#network)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#condition-v1-meta) array_ | conditions represents the observed status of the object.<br />Known .status.conditions.type are: "Available", "Progressing"<br /><br />Available represents the availability of the OpenStack resource. If it is<br />true then the resource is ready for use.<br /><br />Progressing indicates whether the controller is still attempting to<br />reconcile the current state of the OpenStack resource to the desired<br />state. Progressing will be False either because the desired state has<br />been achieved, or because some terminal error prevents it from ever being<br />achieved and the controller is no longer attempting to reconcile. If<br />Progressing is True, an observer waiting on the resource should continue<br />to wait. |  | MaxItems: 32 <br /> |
| `id` _string_ | id is the unique identifier of the OpenStack resource. |  |  |
| `resource` _[NetworkResourceStatus](#networkresourcestatus)_ | resource contains the observed state of the OpenStack resource. |  |  |


#### NeutronDescription

_Underlying type:_ _string_



_Validation:_
- MaxLength: 255
- MinLength: 1

_Appears in:_
- [FloatingIPFilter](#floatingipfilter)
- [FloatingIPResourceSpec](#floatingipresourcespec)
- [NetworkFilter](#networkfilter)
- [NetworkResourceSpec](#networkresourcespec)
- [PortFilter](#portfilter)
- [PortResourceSpec](#portresourcespec)
- [RouterFilter](#routerfilter)
- [RouterResourceSpec](#routerresourcespec)
- [SecurityGroupFilter](#securitygroupfilter)
- [SecurityGroupResourceSpec](#securitygroupresourcespec)
- [SecurityGroupRule](#securitygrouprule)
- [SubnetFilter](#subnetfilter)
- [SubnetResourceSpec](#subnetresourcespec)



#### NeutronStatusMetadata







_Appears in:_
- [FloatingIPResourceStatus](#floatingipresourcestatus)
- [NetworkResourceStatus](#networkresourcestatus)
- [PortResourceStatus](#portresourcestatus)
- [SecurityGroupResourceStatus](#securitygroupresourcestatus)
- [SubnetResourceStatus](#subnetresourcestatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `createdAt` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta)_ | createdAt shows the date and time when the resource was created. The date and time stamp format is ISO 8601 |  |  |
| `updatedAt` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta)_ | updatedAt shows the date and time when the resource was updated. The date and time stamp format is ISO 8601 |  |  |
| `revisionNumber` _integer_ | revisionNumber optionally set via extensions/standard-attr-revisions |  |  |


#### NeutronTag

_Underlying type:_ _string_

NeutronTag represents a tag on a Neutron resource.
It may not be empty and may not contain commas.

_Validation:_
- MaxLength: 255
- MinLength: 1

_Appears in:_
- [FilterByNeutronTags](#filterbyneutrontags)
- [FloatingIPFilter](#floatingipfilter)
- [FloatingIPResourceSpec](#floatingipresourcespec)
- [NetworkFilter](#networkfilter)
- [NetworkResourceSpec](#networkresourcespec)
- [PortFilter](#portfilter)
- [PortResourceSpec](#portresourcespec)
- [RouterFilter](#routerfilter)
- [RouterResourceSpec](#routerresourcespec)
- [SecurityGroupFilter](#securitygroupfilter)
- [SecurityGroupResourceSpec](#securitygroupresourcespec)
- [SubnetFilter](#subnetfilter)
- [SubnetResourceSpec](#subnetresourcespec)





#### OnDelete

_Underlying type:_ _string_



_Validation:_
- Enum: [delete detach]

_Appears in:_
- [ManagedOptions](#managedoptions)

| Field | Description |
| --- | --- |
| `delete` | OnDeleteDelete specifies that the OpenStack resource will be deleted<br />when the managed ORC object is deleted.<br /> |
| `detach` | OnDeleteDetach specifies that the OpenStack resource will not be<br />deleted when the managed ORC object is deleted.<br /> |


#### OpenStackName

_Underlying type:_ _string_



_Validation:_
- MaxLength: 255
- MinLength: 1
- Pattern: `^[^,]+$`

_Appears in:_
- [FlavorFilter](#flavorfilter)
- [FlavorResourceSpec](#flavorresourcespec)
- [ImageFilter](#imagefilter)
- [ImageResourceSpec](#imageresourcespec)
- [NetworkFilter](#networkfilter)
- [NetworkResourceSpec](#networkresourcespec)
- [PortFilter](#portfilter)
- [PortResourceSpec](#portresourcespec)
- [RouterFilter](#routerfilter)
- [RouterResourceSpec](#routerresourcespec)
- [SecurityGroupFilter](#securitygroupfilter)
- [SecurityGroupResourceSpec](#securitygroupresourcespec)
- [ServerFilter](#serverfilter)
- [ServerGroupFilter](#servergroupfilter)
- [ServerGroupResourceSpec](#servergroupresourcespec)
- [ServerResourceSpec](#serverresourcespec)
- [SubnetFilter](#subnetfilter)
- [SubnetResourceSpec](#subnetresourcespec)
- [VolumeFilter](#volumefilter)
- [VolumeResourceSpec](#volumeresourcespec)



#### Port



Port is the Schema for an ORC resource.



_Appears in:_
- [PortList](#portlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openstack.k-orc.cloud/v1alpha1` | | |
| `kind` _string_ | `Port` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[PortSpec](#portspec)_ | spec specifies the desired state of the resource. |  |  |
| `status` _[PortStatus](#portstatus)_ | status defines the observed state of the resource. |  |  |


#### PortFilter



PortFilter specifies a filter to select a port. At least one parameter must be specified.

_Validation:_
- MinProperties: 1

_Appears in:_
- [PortImport](#portimport)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _[OpenStackName](#openstackname)_ | name of the existing resource |  | MaxLength: 255 <br />MinLength: 1 <br />Pattern: `^[^,]+$` <br /> |
| `description` _[NeutronDescription](#neutrondescription)_ | description of the existing resource |  | MaxLength: 255 <br />MinLength: 1 <br /> |
| `networkRef` _[KubernetesNameRef](#kubernetesnameref)_ | networkRef is a reference to the ORC Network which this port is associated with. |  | MaxLength: 253 <br />MinLength: 1 <br /> |
| `projectRef` _[KubernetesNameRef](#kubernetesnameref)_ | projectRef is a reference to the ORC Project this resource is associated with.<br />Typically, only used by admin. |  | MaxLength: 253 <br />MinLength: 1 <br /> |
| `tags` _[NeutronTag](#neutrontag) array_ | tags is a list of tags to filter by. If specified, the resource must<br />have all of the tags specified to be included in the result. |  | MaxItems: 32 <br />MaxLength: 255 <br />MinLength: 1 <br /> |
| `tagsAny` _[NeutronTag](#neutrontag) array_ | tagsAny is a list of tags to filter by. If specified, the resource<br />must have at least one of the tags specified to be included in the<br />result. |  | MaxItems: 32 <br />MaxLength: 255 <br />MinLength: 1 <br /> |
| `notTags` _[NeutronTag](#neutrontag) array_ | notTags is a list of tags to filter by. If specified, resources which<br />contain all of the given tags will be excluded from the result. |  | MaxItems: 32 <br />MaxLength: 255 <br />MinLength: 1 <br /> |
| `notTagsAny` _[NeutronTag](#neutrontag) array_ | notTagsAny is a list of tags to filter by. If specified, resources<br />which contain any of the given tags will be excluded from the result. |  | MaxItems: 32 <br />MaxLength: 255 <br />MinLength: 1 <br /> |


#### PortImport



PortImport specifies an existing resource which will be imported instead of
creating a new one

_Validation:_
- MaxProperties: 1
- MinProperties: 1

_Appears in:_
- [PortSpec](#portspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `id` _string_ | id contains the unique identifier of an existing OpenStack resource. Note<br />that when specifying an import by ID, the resource MUST already exist.<br />The ORC object will enter an error state if the resource does not exist. |  | Format: uuid <br /> |
| `filter` _[PortFilter](#portfilter)_ | filter contains a resource query which is expected to return a single<br />result. The controller will continue to retry if filter returns no<br />results. If filter returns multiple results the controller will set an<br />error state and will not continue to retry. |  | MinProperties: 1 <br /> |


#### PortList



PortList contains a list of Port.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openstack.k-orc.cloud/v1alpha1` | | |
| `kind` _string_ | `PortList` | | |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[Port](#port) array_ | items contains a list of Port. |  |  |


#### PortNumber

_Underlying type:_ _integer_



_Validation:_
- Maximum: 65535
- Minimum: 0

_Appears in:_
- [PortRangeSpec](#portrangespec)



#### PortRangeSpec







_Appears in:_
- [SecurityGroupRule](#securitygrouprule)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `min` _[PortNumber](#portnumber)_ | min is the minimum port number in the range that is matched by the security group rule.<br />If the protocol is TCP, UDP, DCCP, SCTP or UDP-Lite this value must be less than or equal<br />to the port_range_max attribute value. If the protocol is ICMP, this value must be an ICMP type |  | Maximum: 65535 <br />Minimum: 0 <br /> |
| `max` _[PortNumber](#portnumber)_ | max is the maximum port number in the range that is matched by the security group rule.<br />If the protocol is TCP, UDP, DCCP, SCTP or UDP-Lite this value must be greater than or equal<br />to the port_range_min attribute value. If the protocol is ICMP, this value must be an ICMP code. |  | Maximum: 65535 <br />Minimum: 0 <br /> |


#### PortRangeStatus







_Appears in:_
- [SecurityGroupRuleStatus](#securitygrouprulestatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `min` _integer_ | min is the minimum port number in the range that is matched by the security group rule.<br />If the protocol is TCP, UDP, DCCP, SCTP or UDP-Lite this value must be less than or equal<br />to the port_range_max attribute value. If the protocol is ICMP, this value must be an ICMP type |  |  |
| `max` _integer_ | max is the maximum port number in the range that is matched by the security group rule.<br />If the protocol is TCP, UDP, DCCP, SCTP or UDP-Lite this value must be greater than or equal<br />to the port_range_min attribute value. If the protocol is ICMP, this value must be an ICMP code. |  |  |


#### PortResourceSpec







_Appears in:_
- [PortSpec](#portspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _[OpenStackName](#openstackname)_ | name is a human-readable name of the port. If not set, the object's name will be used. |  | MaxLength: 255 <br />MinLength: 1 <br />Pattern: `^[^,]+$` <br /> |
| `description` _[NeutronDescription](#neutrondescription)_ | description is a human-readable description for the resource. |  | MaxLength: 255 <br />MinLength: 1 <br /> |
| `networkRef` _[KubernetesNameRef](#kubernetesnameref)_ | networkRef is a reference to the ORC Network which this port is associated with. |  | MaxLength: 253 <br />MinLength: 1 <br /> |
| `tags` _[NeutronTag](#neutrontag) array_ | tags is a list of tags which will be applied to the port. |  | MaxItems: 32 <br />MaxLength: 255 <br />MinLength: 1 <br /> |
| `allowedAddressPairs` _[AllowedAddressPair](#allowedaddresspair) array_ | allowedAddressPairs are allowed addresses associated with this port. |  | MaxItems: 32 <br /> |
| `addresses` _[Address](#address) array_ | addresses are the IP addresses for the port. |  | MaxItems: 32 <br /> |
| `securityGroupRefs` _[OpenStackName](#openstackname) array_ | securityGroupRefs are the names of the security groups associated<br />with this port. |  | MaxItems: 32 <br />MaxLength: 255 <br />MinLength: 1 <br />Pattern: `^[^,]+$` <br /> |
| `vnicType` _string_ | vnicType specifies the type of vNIC which this port should be<br />attached to. This is used to determine which mechanism driver(s) to<br />be used to bind the port. The valid values are normal, macvtap,<br />direct, baremetal, direct-physical, virtio-forwarder, smart-nic and<br />remote-managed, although these values will not be validated in this<br />API to ensure compatibility with future neutron changes or custom<br />implementations. What type of vNIC is actually available depends on<br />deployments. If not specified, the Neutron default value is used. |  | MaxLength: 64 <br /> |
| `portSecurity` _[PortSecurityState](#portsecuritystate)_ | portSecurity controls port security for this port.<br />When set to Enabled, port security is enabled.<br />When set to Disabled, port security is disabled and SecurityGroupRefs must be empty.<br />When set to Inherit (default), it takes the value from the network level. | Inherit | Enum: [Enabled Disabled Inherit] <br /> |
| `projectRef` _[KubernetesNameRef](#kubernetesnameref)_ | projectRef is a reference to the ORC Project this resource is associated with.<br />Typically, only used by admin. |  | MaxLength: 253 <br />MinLength: 1 <br /> |


#### PortResourceStatus







_Appears in:_
- [PortStatus](#portstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | name is the human-readable name of the resource. Might not be unique. |  | MaxLength: 1024 <br /> |
| `description` _string_ | description is a human-readable description for the resource. |  | MaxLength: 1024 <br /> |
| `networkID` _string_ | networkID is the ID of the attached network. |  | MaxLength: 1024 <br /> |
| `projectID` _string_ | projectID is the project owner of the resource. |  | MaxLength: 1024 <br /> |
| `status` _string_ | status indicates the current status of the resource. |  | MaxLength: 1024 <br /> |
| `tags` _string array_ | tags is the list of tags on the resource. |  | MaxItems: 32 <br /> |
| `adminStateUp` _boolean_ | adminStateUp is the administrative state of the port,<br />which is up (true) or down (false). |  |  |
| `macAddress` _string_ | macAddress is the MAC address of the port. |  | MaxLength: 1024 <br /> |
| `deviceID` _string_ | deviceID is the ID of the device that uses this port. |  | MaxLength: 1024 <br /> |
| `deviceOwner` _string_ | deviceOwner is the entity type that uses this port. |  | MaxLength: 1024 <br /> |
| `allowedAddressPairs` _[AllowedAddressPairStatus](#allowedaddresspairstatus) array_ | allowedAddressPairs is a set of zero or more allowed address pair<br />objects each where address pair object contains an IP address and<br />MAC address. |  | MaxItems: 32 <br /> |
| `fixedIPs` _[FixedIPStatus](#fixedipstatus) array_ | fixedIPs is a set of zero or more fixed IP objects each where fixed<br />IP object contains an IP address and subnet ID from which the IP<br />address is assigned. |  | MaxItems: 32 <br /> |
| `securityGroups` _string array_ | securityGroups contains the IDs of security groups applied to the port. |  | MaxItems: 32 <br /> |
| `propagateUplinkStatus` _boolean_ | propagateUplinkStatus represents the uplink status propagation of<br />the port. |  |  |
| `vnicType` _string_ | vnicType is the type of vNIC which this port is attached to. |  | MaxLength: 64 <br /> |
| `portSecurityEnabled` _boolean_ | portSecurityEnabled indicates whether port security is enabled or not. |  |  |
| `createdAt` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta)_ | createdAt shows the date and time when the resource was created. The date and time stamp format is ISO 8601 |  |  |
| `updatedAt` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta)_ | updatedAt shows the date and time when the resource was updated. The date and time stamp format is ISO 8601 |  |  |
| `revisionNumber` _integer_ | revisionNumber optionally set via extensions/standard-attr-revisions |  |  |


#### PortSecurityState

_Underlying type:_ _string_

PortSecurityState represents the security state of a port

_Validation:_
- Enum: [Enabled Disabled Inherit]

_Appears in:_
- [PortResourceSpec](#portresourcespec)

| Field | Description |
| --- | --- |
| `Enabled` | PortSecurityEnabled means port security is enabled<br /> |
| `Disabled` | PortSecurityDisabled means port security is disabled<br /> |
| `Inherit` | PortSecurityInherit means port security settings are inherited from the network<br /> |


#### PortSpec



PortSpec defines the desired state of an ORC object.



_Appears in:_
- [Port](#port)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `import` _[PortImport](#portimport)_ | import refers to an existing OpenStack resource which will be imported instead of<br />creating a new one. |  | MaxProperties: 1 <br />MinProperties: 1 <br /> |
| `resource` _[PortResourceSpec](#portresourcespec)_ | resource specifies the desired state of the resource.<br /><br />resource may not be specified if the management policy is `unmanaged`.<br /><br />resource must be specified if the management policy is `managed`. |  |  |
| `managementPolicy` _[ManagementPolicy](#managementpolicy)_ | managementPolicy defines how ORC will treat the object. Valid values are<br />`managed`: ORC will create, update, and delete the resource; `unmanaged`:<br />ORC will import an existing resource, and will not apply updates to it or<br />delete it. | managed | Enum: [managed unmanaged] <br /> |
| `managedOptions` _[ManagedOptions](#managedoptions)_ | managedOptions specifies options which may be applied to managed objects. |  |  |
| `cloudCredentialsRef` _[CloudCredentialsReference](#cloudcredentialsreference)_ | cloudCredentialsRef points to a secret containing OpenStack credentials |  |  |


#### PortStatus



PortStatus defines the observed state of an ORC resource.



_Appears in:_
- [Port](#port)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#condition-v1-meta) array_ | conditions represents the observed status of the object.<br />Known .status.conditions.type are: "Available", "Progressing"<br /><br />Available represents the availability of the OpenStack resource. If it is<br />true then the resource is ready for use.<br /><br />Progressing indicates whether the controller is still attempting to<br />reconcile the current state of the OpenStack resource to the desired<br />state. Progressing will be False either because the desired state has<br />been achieved, or because some terminal error prevents it from ever being<br />achieved and the controller is no longer attempting to reconcile. If<br />Progressing is True, an observer waiting on the resource should continue<br />to wait. |  | MaxItems: 32 <br /> |
| `id` _string_ | id is the unique identifier of the OpenStack resource. |  |  |
| `resource` _[PortResourceStatus](#portresourcestatus)_ | resource contains the observed state of the OpenStack resource. |  |  |


#### Project



Project is the Schema for an ORC resource.



_Appears in:_
- [ProjectList](#projectlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openstack.k-orc.cloud/v1alpha1` | | |
| `kind` _string_ | `Project` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[ProjectSpec](#projectspec)_ | spec specifies the desired state of the resource. |  |  |
| `status` _[ProjectStatus](#projectstatus)_ | status defines the observed state of the resource. |  |  |


#### ProjectFilter



ProjectFilter defines an existing resource by its properties

_Validation:_
- MinProperties: 1

_Appears in:_
- [ProjectImport](#projectimport)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _[KeystoneName](#keystonename)_ | name of the existing resource |  | MaxLength: 64 <br />MinLength: 1 <br /> |
| `tags` _[KeystoneTag](#keystonetag) array_ | tags is a list of tags to filter by. If specified, the resource must<br />have all of the tags specified to be included in the result. |  | MaxItems: 80 <br />MaxLength: 255 <br />MinLength: 1 <br /> |
| `tagsAny` _[KeystoneTag](#keystonetag) array_ | tagsAny is a list of tags to filter by. If specified, the resource<br />must have at least one of the tags specified to be included in the<br />result. |  | MaxItems: 80 <br />MaxLength: 255 <br />MinLength: 1 <br /> |
| `notTags` _[KeystoneTag](#keystonetag) array_ | notTags is a list of tags to filter by. If specified, resources which<br />contain all of the given tags will be excluded from the result. |  | MaxItems: 80 <br />MaxLength: 255 <br />MinLength: 1 <br /> |
| `notTagsAny` _[KeystoneTag](#keystonetag) array_ | notTagsAny is a list of tags to filter by. If specified, resources<br />which contain any of the given tags will be excluded from the result. |  | MaxItems: 80 <br />MaxLength: 255 <br />MinLength: 1 <br /> |


#### ProjectImport



ProjectImport specifies an existing resource which will be imported instead of
creating a new one

_Validation:_
- MaxProperties: 1
- MinProperties: 1

_Appears in:_
- [ProjectSpec](#projectspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `id` _string_ | id contains the unique identifier of an existing OpenStack resource. Note<br />that when specifying an import by ID, the resource MUST already exist.<br />The ORC object will enter an error state if the resource does not exist. |  | Format: uuid <br /> |
| `filter` _[ProjectFilter](#projectfilter)_ | filter contains a resource query which is expected to return a single<br />result. The controller will continue to retry if filter returns no<br />results. If filter returns multiple results the controller will set an<br />error state and will not continue to retry. |  | MinProperties: 1 <br /> |


#### ProjectList



ProjectList contains a list of Project.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openstack.k-orc.cloud/v1alpha1` | | |
| `kind` _string_ | `ProjectList` | | |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[Project](#project) array_ | items contains a list of Project. |  |  |


#### ProjectResourceSpec



ProjectResourceSpec contains the desired state of a project



_Appears in:_
- [ProjectSpec](#projectspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _[KeystoneName](#keystonename)_ | name will be the name of the created resource. If not specified, the<br />name of the ORC object will be used. |  | MaxLength: 64 <br />MinLength: 1 <br /> |
| `description` _string_ | description contains a free form description of the project. |  | MaxLength: 65535 <br />MinLength: 1 <br /> |
| `enabled` _boolean_ | enabled defines whether a project is enabled or not. Default is true. |  |  |
| `tags` _[KeystoneTag](#keystonetag) array_ | tags is list of simple strings assigned to a project.<br />Tags can be used to classify projects into groups. |  | MaxItems: 80 <br />MaxLength: 255 <br />MinLength: 1 <br /> |


#### ProjectResourceStatus



ProjectResourceStatus represents the observed state of the resource.



_Appears in:_
- [ProjectStatus](#projectstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | name is a Human-readable name for the project. Might not be unique. |  | MaxLength: 1024 <br /> |
| `description` _string_ | description is a human-readable description for the resource. |  | MaxLength: 65535 <br /> |
| `enabled` _boolean_ | enabled represents whether a project is enabled or not. |  |  |
| `tags` _string array_ | tags is the list of tags on the resource. |  | MaxItems: 80 <br /> |


#### ProjectSpec



ProjectSpec defines the desired state of an ORC object.



_Appears in:_
- [Project](#project)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `import` _[ProjectImport](#projectimport)_ | import refers to an existing OpenStack resource which will be imported instead of<br />creating a new one. |  | MaxProperties: 1 <br />MinProperties: 1 <br /> |
| `resource` _[ProjectResourceSpec](#projectresourcespec)_ | resource specifies the desired state of the resource.<br /><br />resource may not be specified if the management policy is `unmanaged`.<br /><br />resource must be specified if the management policy is `managed`. |  |  |
| `managementPolicy` _[ManagementPolicy](#managementpolicy)_ | managementPolicy defines how ORC will treat the object. Valid values are<br />`managed`: ORC will create, update, and delete the resource; `unmanaged`:<br />ORC will import an existing resource, and will not apply updates to it or<br />delete it. | managed | Enum: [managed unmanaged] <br /> |
| `managedOptions` _[ManagedOptions](#managedoptions)_ | managedOptions specifies options which may be applied to managed objects. |  |  |
| `cloudCredentialsRef` _[CloudCredentialsReference](#cloudcredentialsreference)_ | cloudCredentialsRef points to a secret containing OpenStack credentials |  |  |


#### ProjectStatus



ProjectStatus defines the observed state of an ORC resource.



_Appears in:_
- [Project](#project)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#condition-v1-meta) array_ | conditions represents the observed status of the object.<br />Known .status.conditions.type are: "Available", "Progressing"<br /><br />Available represents the availability of the OpenStack resource. If it is<br />true then the resource is ready for use.<br /><br />Progressing indicates whether the controller is still attempting to<br />reconcile the current state of the OpenStack resource to the desired<br />state. Progressing will be False either because the desired state has<br />been achieved, or because some terminal error prevents it from ever being<br />achieved and the controller is no longer attempting to reconcile. If<br />Progressing is True, an observer waiting on the resource should continue<br />to wait. |  | MaxItems: 32 <br /> |
| `id` _string_ | id is the unique identifier of the OpenStack resource. |  |  |
| `resource` _[ProjectResourceStatus](#projectresourcestatus)_ | resource contains the observed state of the OpenStack resource. |  |  |


#### Protocol

_Underlying type:_ _string_



_Validation:_
- Enum: [ah dccp egp esp gre icmp icmpv6 igmp ipip ipv6-encap ipv6-frag ipv6-icmp ipv6-nonxt ipv6-opts ipv6-route ospf pgm rsvp sctp tcp udp udplite vrrp]

_Appears in:_
- [SecurityGroupRule](#securitygrouprule)

| Field | Description |
| --- | --- |
| `ah` |  |
| `dccp` |  |
| `egp` |  |
| `esp` |  |
| `gre` |  |
| `icmp` |  |
| `icmpv6` |  |
| `igmp` |  |
| `ipip` |  |
| `ipv6-encap` |  |
| `ipv6-frag` |  |
| `ipv6-icmp` |  |
| `ipv6-nonxt` |  |
| `ipv6-opts` |  |
| `ipv6-route` |  |
| `ospf` |  |
| `pgm` |  |
| `rsvp` |  |
| `sctp` |  |
| `tcp` |  |
| `udp` |  |
| `udplite` |  |
| `vrrp` |  |


#### ProviderPropertiesStatus







_Appears in:_
- [NetworkResourceStatus](#networkresourcestatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `networkType` _string_ | networkType is the type of physical network that this<br />network should be mapped to. Supported values are flat, vlan, vxlan, and gre.<br />Valid values depend on the networking back-end. |  | MaxLength: 1024 <br /> |
| `physicalNetwork` _string_ | physicalNetwork is the physical network where this network<br />should be implemented. The Networking API v2.0 does not provide a<br />way to list available physical networks. For example, the Open<br />vSwitch plug-in configuration file defines a symbolic name that maps<br />to specific bridges on each compute host. |  | MaxLength: 1024 <br /> |
| `segmentationID` _integer_ | segmentationID is the ID of the isolated segment on the<br />physical network. The network_type attribute defines the<br />segmentation model. For example, if the network_type value is vlan,<br />this ID is a vlan identifier. If the network_type value is gre, this<br />ID is a gre key. |  |  |


#### Router



Router is the Schema for an ORC resource.



_Appears in:_
- [RouterList](#routerlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openstack.k-orc.cloud/v1alpha1` | | |
| `kind` _string_ | `Router` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[RouterSpec](#routerspec)_ | spec specifies the desired state of the resource. |  |  |
| `status` _[RouterStatus](#routerstatus)_ | status defines the observed state of the resource. |  |  |


#### RouterFilter



RouterFilter specifies a query to select an OpenStack router. At least one property must be set.

_Validation:_
- MinProperties: 1

_Appears in:_
- [RouterImport](#routerimport)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _[OpenStackName](#openstackname)_ | name of the existing resource |  | MaxLength: 255 <br />MinLength: 1 <br />Pattern: `^[^,]+$` <br /> |
| `description` _[NeutronDescription](#neutrondescription)_ | description of the existing resource |  | MaxLength: 255 <br />MinLength: 1 <br /> |
| `projectRef` _[KubernetesNameRef](#kubernetesnameref)_ | projectRef is a reference to the ORC Project this resource is associated with.<br />Typically, only used by admin. |  | MaxLength: 253 <br />MinLength: 1 <br /> |
| `tags` _[NeutronTag](#neutrontag) array_ | tags is a list of tags to filter by. If specified, the resource must<br />have all of the tags specified to be included in the result. |  | MaxItems: 32 <br />MaxLength: 255 <br />MinLength: 1 <br /> |
| `tagsAny` _[NeutronTag](#neutrontag) array_ | tagsAny is a list of tags to filter by. If specified, the resource<br />must have at least one of the tags specified to be included in the<br />result. |  | MaxItems: 32 <br />MaxLength: 255 <br />MinLength: 1 <br /> |
| `notTags` _[NeutronTag](#neutrontag) array_ | notTags is a list of tags to filter by. If specified, resources which<br />contain all of the given tags will be excluded from the result. |  | MaxItems: 32 <br />MaxLength: 255 <br />MinLength: 1 <br /> |
| `notTagsAny` _[NeutronTag](#neutrontag) array_ | notTagsAny is a list of tags to filter by. If specified, resources<br />which contain any of the given tags will be excluded from the result. |  | MaxItems: 32 <br />MaxLength: 255 <br />MinLength: 1 <br /> |


#### RouterImport



RouterImport specifies an existing resource which will be imported instead of
creating a new one

_Validation:_
- MaxProperties: 1
- MinProperties: 1

_Appears in:_
- [RouterSpec](#routerspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `id` _string_ | id contains the unique identifier of an existing OpenStack resource. Note<br />that when specifying an import by ID, the resource MUST already exist.<br />The ORC object will enter an error state if the resource does not exist. |  | Format: uuid <br /> |
| `filter` _[RouterFilter](#routerfilter)_ | filter contains a resource query which is expected to return a single<br />result. The controller will continue to retry if filter returns no<br />results. If filter returns multiple results the controller will set an<br />error state and will not continue to retry. |  | MinProperties: 1 <br /> |


#### RouterInterface



RouterInterface is the Schema for an ORC resource.



_Appears in:_
- [RouterInterfaceList](#routerinterfacelist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openstack.k-orc.cloud/v1alpha1` | | |
| `kind` _string_ | `RouterInterface` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[RouterInterfaceSpec](#routerinterfacespec)_ | spec specifies the desired state of the resource. |  |  |
| `status` _[RouterInterfaceStatus](#routerinterfacestatus)_ | status defines the observed state of the resource. |  |  |


#### RouterInterfaceList



RouterInterfaceList contains a list of RouterInterface.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openstack.k-orc.cloud/v1alpha1` | | |
| `kind` _string_ | `RouterInterfaceList` | | |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[RouterInterface](#routerinterface) array_ | items contains a list of RouterInterface. |  | MaxItems: 32 <br /> |


#### RouterInterfaceSpec







_Appears in:_
- [RouterInterface](#routerinterface)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _[RouterInterfaceType](#routerinterfacetype)_ | type specifies the type of the router interface. |  | Enum: [Subnet] <br />MaxLength: 8 <br />MinLength: 1 <br /> |
| `routerRef` _[KubernetesNameRef](#kubernetesnameref)_ | routerRef references the router to which this interface belongs. |  | MaxLength: 253 <br />MinLength: 1 <br /> |
| `subnetRef` _[KubernetesNameRef](#kubernetesnameref)_ | subnetRef references the subnet the router interface is created on. |  | MaxLength: 253 <br />MinLength: 1 <br /> |


#### RouterInterfaceStatus







_Appears in:_
- [RouterInterface](#routerinterface)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#condition-v1-meta) array_ | conditions represents the observed status of the object.<br />Known .status.conditions.type are: "Available", "Progressing"<br /><br />Available represents the availability of the OpenStack resource. If it is<br />true then the resource is ready for use.<br /><br />Progressing indicates whether the controller is still attempting to<br />reconcile the current state of the OpenStack resource to the desired<br />state. Progressing will be False either because the desired state has<br />been achieved, or because some terminal error prevents it from ever being<br />achieved and the controller is no longer attempting to reconcile. If<br />Progressing is True, an observer waiting on the resource should continue<br />to wait. |  | MaxItems: 32 <br /> |
| `id` _string_ | id is the unique identifier of the port created for the router interface |  | MaxLength: 1024 <br /> |


#### RouterInterfaceType

_Underlying type:_ _string_



_Validation:_
- Enum: [Subnet]
- MaxLength: 8
- MinLength: 1

_Appears in:_
- [RouterInterfaceSpec](#routerinterfacespec)

| Field | Description |
| --- | --- |
| `Subnet` |  |


#### RouterList



RouterList contains a list of Router.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openstack.k-orc.cloud/v1alpha1` | | |
| `kind` _string_ | `RouterList` | | |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[Router](#router) array_ | items contains a list of Router. |  |  |


#### RouterResourceSpec







_Appears in:_
- [RouterSpec](#routerspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _[OpenStackName](#openstackname)_ | name is a human-readable name of the router. If not set, the<br />object's name will be used. |  | MaxLength: 255 <br />MinLength: 1 <br />Pattern: `^[^,]+$` <br /> |
| `description` _[NeutronDescription](#neutrondescription)_ | description is a human-readable description for the resource. |  | MaxLength: 255 <br />MinLength: 1 <br /> |
| `tags` _[NeutronTag](#neutrontag) array_ | tags is a list of tags which will be applied to the router. |  | MaxItems: 32 <br />MaxLength: 255 <br />MinLength: 1 <br /> |
| `adminStateUp` _boolean_ | adminStateUp represents the administrative state of the resource,<br />which is up (true) or down (false). Default is true. |  |  |
| `externalGateways` _[ExternalGateway](#externalgateway) array_ | externalGateways is a list of external gateways for the router.<br />Multiple gateways are not currently supported by ORC. |  | MaxItems: 1 <br /> |
| `distributed` _boolean_ | distributed indicates whether the router is distributed or not. It<br />is available when dvr extension is enabled. |  |  |
| `availabilityZoneHints` _[AvailabilityZoneHint](#availabilityzonehint) array_ | availabilityZoneHints is the availability zone candidate for the router. |  | MaxItems: 32 <br />MaxLength: 255 <br />MinLength: 1 <br /> |
| `projectRef` _[KubernetesNameRef](#kubernetesnameref)_ | projectRef is a reference to the ORC Project this resource is associated with.<br />Typically, only used by admin. |  | MaxLength: 253 <br />MinLength: 1 <br /> |


#### RouterResourceStatus







_Appears in:_
- [RouterStatus](#routerstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | name is the human-readable name of the resource. Might not be unique. |  | MaxLength: 1024 <br /> |
| `description` _string_ | description is a human-readable description for the resource. |  | MaxLength: 1024 <br /> |
| `projectID` _string_ | projectID is the project owner of the resource. |  | MaxLength: 1024 <br /> |
| `status` _string_ | status indicates the current status of the resource. |  | MaxLength: 1024 <br /> |
| `tags` _string array_ | tags is the list of tags on the resource. |  | MaxItems: 32 <br /> |
| `adminStateUp` _boolean_ | adminStateUp is the administrative state of the router,<br />which is up (true) or down (false). |  |  |
| `externalGateways` _[ExternalGatewayStatus](#externalgatewaystatus) array_ | externalGateways is a list of external gateways for the router. |  | MaxItems: 32 <br /> |
| `availabilityZoneHints` _string array_ | availabilityZoneHints is the availability zone candidate for the<br />router. |  | MaxItems: 32 <br /> |


#### RouterSpec



RouterSpec defines the desired state of an ORC object.



_Appears in:_
- [Router](#router)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `import` _[RouterImport](#routerimport)_ | import refers to an existing OpenStack resource which will be imported instead of<br />creating a new one. |  | MaxProperties: 1 <br />MinProperties: 1 <br /> |
| `resource` _[RouterResourceSpec](#routerresourcespec)_ | resource specifies the desired state of the resource.<br /><br />resource may not be specified if the management policy is `unmanaged`.<br /><br />resource must be specified if the management policy is `managed`. |  |  |
| `managementPolicy` _[ManagementPolicy](#managementpolicy)_ | managementPolicy defines how ORC will treat the object. Valid values are<br />`managed`: ORC will create, update, and delete the resource; `unmanaged`:<br />ORC will import an existing resource, and will not apply updates to it or<br />delete it. | managed | Enum: [managed unmanaged] <br /> |
| `managedOptions` _[ManagedOptions](#managedoptions)_ | managedOptions specifies options which may be applied to managed objects. |  |  |
| `cloudCredentialsRef` _[CloudCredentialsReference](#cloudcredentialsreference)_ | cloudCredentialsRef points to a secret containing OpenStack credentials |  |  |


#### RouterStatus



RouterStatus defines the observed state of an ORC resource.



_Appears in:_
- [Router](#router)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#condition-v1-meta) array_ | conditions represents the observed status of the object.<br />Known .status.conditions.type are: "Available", "Progressing"<br /><br />Available represents the availability of the OpenStack resource. If it is<br />true then the resource is ready for use.<br /><br />Progressing indicates whether the controller is still attempting to<br />reconcile the current state of the OpenStack resource to the desired<br />state. Progressing will be False either because the desired state has<br />been achieved, or because some terminal error prevents it from ever being<br />achieved and the controller is no longer attempting to reconcile. If<br />Progressing is True, an observer waiting on the resource should continue<br />to wait. |  | MaxItems: 32 <br /> |
| `id` _string_ | id is the unique identifier of the OpenStack resource. |  |  |
| `resource` _[RouterResourceStatus](#routerresourcestatus)_ | resource contains the observed state of the OpenStack resource. |  |  |


#### RuleDirection

_Underlying type:_ _string_



_Validation:_
- Enum: [ingress egress]

_Appears in:_
- [SecurityGroupRule](#securitygrouprule)



#### SecurityGroup



SecurityGroup is the Schema for an ORC resource.



_Appears in:_
- [SecurityGroupList](#securitygrouplist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openstack.k-orc.cloud/v1alpha1` | | |
| `kind` _string_ | `SecurityGroup` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[SecurityGroupSpec](#securitygroupspec)_ | spec specifies the desired state of the resource. |  |  |
| `status` _[SecurityGroupStatus](#securitygroupstatus)_ | status defines the observed state of the resource. |  |  |


#### SecurityGroupFilter



SecurityGroupFilter defines an existing resource by its properties

_Validation:_
- MinProperties: 1

_Appears in:_
- [SecurityGroupImport](#securitygroupimport)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _[OpenStackName](#openstackname)_ | name of the existing resource |  | MaxLength: 255 <br />MinLength: 1 <br />Pattern: `^[^,]+$` <br /> |
| `description` _[NeutronDescription](#neutrondescription)_ | description of the existing resource |  | MaxLength: 255 <br />MinLength: 1 <br /> |
| `projectRef` _[KubernetesNameRef](#kubernetesnameref)_ | projectRef is a reference to the ORC Project this resource is associated with.<br />Typically, only used by admin. |  | MaxLength: 253 <br />MinLength: 1 <br /> |
| `tags` _[NeutronTag](#neutrontag) array_ | tags is a list of tags to filter by. If specified, the resource must<br />have all of the tags specified to be included in the result. |  | MaxItems: 32 <br />MaxLength: 255 <br />MinLength: 1 <br /> |
| `tagsAny` _[NeutronTag](#neutrontag) array_ | tagsAny is a list of tags to filter by. If specified, the resource<br />must have at least one of the tags specified to be included in the<br />result. |  | MaxItems: 32 <br />MaxLength: 255 <br />MinLength: 1 <br /> |
| `notTags` _[NeutronTag](#neutrontag) array_ | notTags is a list of tags to filter by. If specified, resources which<br />contain all of the given tags will be excluded from the result. |  | MaxItems: 32 <br />MaxLength: 255 <br />MinLength: 1 <br /> |
| `notTagsAny` _[NeutronTag](#neutrontag) array_ | notTagsAny is a list of tags to filter by. If specified, resources<br />which contain any of the given tags will be excluded from the result. |  | MaxItems: 32 <br />MaxLength: 255 <br />MinLength: 1 <br /> |


#### SecurityGroupImport



SecurityGroupImport specifies an existing resource which will be imported instead of
creating a new one

_Validation:_
- MaxProperties: 1
- MinProperties: 1

_Appears in:_
- [SecurityGroupSpec](#securitygroupspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `id` _string_ | id contains the unique identifier of an existing OpenStack resource. Note<br />that when specifying an import by ID, the resource MUST already exist.<br />The ORC object will enter an error state if the resource does not exist. |  | Format: uuid <br /> |
| `filter` _[SecurityGroupFilter](#securitygroupfilter)_ | filter contains a resource query which is expected to return a single<br />result. The controller will continue to retry if filter returns no<br />results. If filter returns multiple results the controller will set an<br />error state and will not continue to retry. |  | MinProperties: 1 <br /> |


#### SecurityGroupList



SecurityGroupList contains a list of SecurityGroup.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openstack.k-orc.cloud/v1alpha1` | | |
| `kind` _string_ | `SecurityGroupList` | | |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[SecurityGroup](#securitygroup) array_ | items contains a list of SecurityGroup. |  |  |


#### SecurityGroupResourceSpec



SecurityGroupResourceSpec contains the desired state of a security group



_Appears in:_
- [SecurityGroupSpec](#securitygroupspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _[OpenStackName](#openstackname)_ | name will be the name of the created resource. If not specified, the<br />name of the ORC object will be used. |  | MaxLength: 255 <br />MinLength: 1 <br />Pattern: `^[^,]+$` <br /> |
| `description` _[NeutronDescription](#neutrondescription)_ | description is a human-readable description for the resource. |  | MaxLength: 255 <br />MinLength: 1 <br /> |
| `tags` _[NeutronTag](#neutrontag) array_ | tags is a list of tags which will be applied to the security group. |  | MaxItems: 32 <br />MaxLength: 255 <br />MinLength: 1 <br /> |
| `stateful` _boolean_ | stateful indicates if the security group is stateful or stateless. |  |  |
| `rules` _[SecurityGroupRule](#securitygrouprule) array_ | rules is a list of security group rules belonging to this SG. |  | MaxItems: 256 <br />MinProperties: 1 <br /> |
| `projectRef` _[KubernetesNameRef](#kubernetesnameref)_ | projectRef is a reference to the ORC Project this resource is associated with.<br />Typically, only used by admin. |  | MaxLength: 253 <br />MinLength: 1 <br /> |


#### SecurityGroupResourceStatus



SecurityGroupResourceStatus represents the observed state of the resource.



_Appears in:_
- [SecurityGroupStatus](#securitygroupstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | name is a Human-readable name for the security group. Might not be unique. |  | MaxLength: 1024 <br /> |
| `description` _string_ | description is a human-readable description for the resource. |  | MaxLength: 1024 <br /> |
| `projectID` _string_ | projectID is the project owner of the security group. |  | MaxLength: 1024 <br /> |
| `tags` _string array_ | tags is the list of tags on the resource. |  | MaxItems: 32 <br /> |
| `stateful` _boolean_ | stateful indicates if the security group is stateful or stateless. |  |  |
| `rules` _[SecurityGroupRuleStatus](#securitygrouprulestatus) array_ | rules is a list of security group rules belonging to this SG. |  | MaxItems: 256 <br /> |
| `createdAt` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta)_ | createdAt shows the date and time when the resource was created. The date and time stamp format is ISO 8601 |  |  |
| `updatedAt` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta)_ | updatedAt shows the date and time when the resource was updated. The date and time stamp format is ISO 8601 |  |  |
| `revisionNumber` _integer_ | revisionNumber optionally set via extensions/standard-attr-revisions |  |  |


#### SecurityGroupRule



SecurityGroupRule defines a Security Group rule

_Validation:_
- MinProperties: 1

_Appears in:_
- [SecurityGroupResourceSpec](#securitygroupresourcespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `description` _[NeutronDescription](#neutrondescription)_ | description is a human-readable description for the resource. |  | MaxLength: 255 <br />MinLength: 1 <br /> |
| `direction` _[RuleDirection](#ruledirection)_ | direction represents the direction in which the security group rule<br />is applied. Can be ingress or egress. |  | Enum: [ingress egress] <br /> |
| `remoteIPPrefix` _[CIDR](#cidr)_ | remoteIPPrefix is an IP address block. Should match the Ethertype (IPv4 or IPv6) |  | Format: cidr <br />MaxLength: 49 <br />MinLength: 1 <br /> |
| `protocol` _[Protocol](#protocol)_ | protocol is the IP protocol is represented by a string |  | Enum: [ah dccp egp esp gre icmp icmpv6 igmp ipip ipv6-encap ipv6-frag ipv6-icmp ipv6-nonxt ipv6-opts ipv6-route ospf pgm rsvp sctp tcp udp udplite vrrp] <br /> |
| `ethertype` _[Ethertype](#ethertype)_ | ethertype must be IPv4 or IPv6, and addresses represented in CIDR<br />must match the ingress or egress rules. |  | Enum: [IPv4 IPv6] <br /> |
| `portRange` _[PortRangeSpec](#portrangespec)_ | portRange sets the minimum and maximum ports range that the security group rule<br />matches. If the protocol is [tcp, udp, dccp sctp,udplite] PortRange.Min must be less than<br />or equal to the PortRange.Max attribute value.<br />If the protocol is ICMP, this PortRamge.Min must be an ICMP code and PortRange.Max<br />should be an ICMP type |  |  |


#### SecurityGroupRuleStatus







_Appears in:_
- [SecurityGroupResourceStatus](#securitygroupresourcestatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `id` _string_ | id is the ID of the security group rule. |  | MaxLength: 1024 <br /> |
| `description` _string_ | description is a human-readable description for the resource. |  | MaxLength: 1024 <br /> |
| `direction` _string_ | direction represents the direction in which the security group rule<br />is applied. Can be ingress or egress. |  | MaxLength: 1024 <br /> |
| `remoteGroupID` _string_ | remoteGroupID is the remote group UUID to associate with this security group rule<br />RemoteGroupID |  | MaxLength: 1024 <br /> |
| `remoteIPPrefix` _string_ | remoteIPPrefix is an IP address block. Should match the Ethertype (IPv4 or IPv6) |  | MaxLength: 1024 <br /> |
| `protocol` _string_ | protocol is the IP protocol can be represented by a string, an<br />integer, or null |  | MaxLength: 1024 <br /> |
| `ethertype` _string_ | ethertype must be IPv4 or IPv6, and addresses represented in CIDR<br />must match the ingress or egress rules. |  | MaxLength: 1024 <br /> |
| `portRange` _[PortRangeStatus](#portrangestatus)_ | portRange sets the minimum and maximum ports range that the security group rule<br />matches. If the protocol is [tcp, udp, dccp sctp,udplite] PortRange.Min must be less than<br />or equal to the PortRange.Max attribute value.<br />If the protocol is ICMP, this PortRamge.Min must be an ICMP code and PortRange.Max<br />should be an ICMP type |  |  |


#### SecurityGroupSpec



SecurityGroupSpec defines the desired state of an ORC object.



_Appears in:_
- [SecurityGroup](#securitygroup)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `import` _[SecurityGroupImport](#securitygroupimport)_ | import refers to an existing OpenStack resource which will be imported instead of<br />creating a new one. |  | MaxProperties: 1 <br />MinProperties: 1 <br /> |
| `resource` _[SecurityGroupResourceSpec](#securitygroupresourcespec)_ | resource specifies the desired state of the resource.<br /><br />resource may not be specified if the management policy is `unmanaged`.<br /><br />resource must be specified if the management policy is `managed`. |  |  |
| `managementPolicy` _[ManagementPolicy](#managementpolicy)_ | managementPolicy defines how ORC will treat the object. Valid values are<br />`managed`: ORC will create, update, and delete the resource; `unmanaged`:<br />ORC will import an existing resource, and will not apply updates to it or<br />delete it. | managed | Enum: [managed unmanaged] <br /> |
| `managedOptions` _[ManagedOptions](#managedoptions)_ | managedOptions specifies options which may be applied to managed objects. |  |  |
| `cloudCredentialsRef` _[CloudCredentialsReference](#cloudcredentialsreference)_ | cloudCredentialsRef points to a secret containing OpenStack credentials |  |  |


#### SecurityGroupStatus



SecurityGroupStatus defines the observed state of an ORC resource.



_Appears in:_
- [SecurityGroup](#securitygroup)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#condition-v1-meta) array_ | conditions represents the observed status of the object.<br />Known .status.conditions.type are: "Available", "Progressing"<br /><br />Available represents the availability of the OpenStack resource. If it is<br />true then the resource is ready for use.<br /><br />Progressing indicates whether the controller is still attempting to<br />reconcile the current state of the OpenStack resource to the desired<br />state. Progressing will be False either because the desired state has<br />been achieved, or because some terminal error prevents it from ever being<br />achieved and the controller is no longer attempting to reconcile. If<br />Progressing is True, an observer waiting on the resource should continue<br />to wait. |  | MaxItems: 32 <br /> |
| `id` _string_ | id is the unique identifier of the OpenStack resource. |  |  |
| `resource` _[SecurityGroupResourceStatus](#securitygroupresourcestatus)_ | resource contains the observed state of the OpenStack resource. |  |  |


#### Server



Server is the Schema for an ORC resource.



_Appears in:_
- [ServerList](#serverlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openstack.k-orc.cloud/v1alpha1` | | |
| `kind` _string_ | `Server` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[ServerSpec](#serverspec)_ | spec specifies the desired state of the resource. |  |  |
| `status` _[ServerStatus](#serverstatus)_ | status defines the observed state of the resource. |  |  |


#### ServerFilter



ServerFilter defines an existing resource by its properties

_Validation:_
- MinProperties: 1

_Appears in:_
- [ServerImport](#serverimport)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _[OpenStackName](#openstackname)_ | name of the existing resource |  | MaxLength: 255 <br />MinLength: 1 <br />Pattern: `^[^,]+$` <br /> |
| `tags` _[ServerTag](#servertag) array_ | tags is a list of tags to filter by. If specified, the resource must<br />have all of the tags specified to be included in the result. |  | MaxItems: 32 <br />MaxLength: 80 <br />MinLength: 1 <br /> |
| `tagsAny` _[ServerTag](#servertag) array_ | tagsAny is a list of tags to filter by. If specified, the resource<br />must have at least one of the tags specified to be included in the<br />result. |  | MaxItems: 32 <br />MaxLength: 80 <br />MinLength: 1 <br /> |
| `notTags` _[ServerTag](#servertag) array_ | notTags is a list of tags to filter by. If specified, resources which<br />contain all of the given tags will be excluded from the result. |  | MaxItems: 32 <br />MaxLength: 80 <br />MinLength: 1 <br /> |
| `notTagsAny` _[ServerTag](#servertag) array_ | notTagsAny is a list of tags to filter by. If specified, resources<br />which contain any of the given tags will be excluded from the result. |  | MaxItems: 32 <br />MaxLength: 80 <br />MinLength: 1 <br /> |


#### ServerGroup



ServerGroup is the Schema for an ORC resource.



_Appears in:_
- [ServerGroupList](#servergrouplist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openstack.k-orc.cloud/v1alpha1` | | |
| `kind` _string_ | `ServerGroup` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[ServerGroupSpec](#servergroupspec)_ | spec specifies the desired state of the resource. |  |  |
| `status` _[ServerGroupStatus](#servergroupstatus)_ | status defines the observed state of the resource. |  |  |


#### ServerGroupFilter



ServerGroupFilter defines an existing resource by its properties

_Validation:_
- MinProperties: 1

_Appears in:_
- [ServerGroupImport](#servergroupimport)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _[OpenStackName](#openstackname)_ | name of the existing resource |  | MaxLength: 255 <br />MinLength: 1 <br />Pattern: `^[^,]+$` <br /> |


#### ServerGroupImport



ServerGroupImport specifies an existing resource which will be imported instead of
creating a new one

_Validation:_
- MaxProperties: 1
- MinProperties: 1

_Appears in:_
- [ServerGroupSpec](#servergroupspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `id` _string_ | id contains the unique identifier of an existing OpenStack resource. Note<br />that when specifying an import by ID, the resource MUST already exist.<br />The ORC object will enter an error state if the resource does not exist. |  | Format: uuid <br /> |
| `filter` _[ServerGroupFilter](#servergroupfilter)_ | filter contains a resource query which is expected to return a single<br />result. The controller will continue to retry if filter returns no<br />results. If filter returns multiple results the controller will set an<br />error state and will not continue to retry. |  | MinProperties: 1 <br /> |


#### ServerGroupList



ServerGroupList contains a list of ServerGroup.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openstack.k-orc.cloud/v1alpha1` | | |
| `kind` _string_ | `ServerGroupList` | | |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[ServerGroup](#servergroup) array_ | items contains a list of ServerGroup. |  |  |


#### ServerGroupPolicy

_Underlying type:_ _string_



_Validation:_
- Enum: [affinity anti-affinity soft-affinity soft-anti-affinity]

_Appears in:_
- [ServerGroupResourceSpec](#servergroupresourcespec)

| Field | Description |
| --- | --- |
| `affinity` | ServerGroupPolicyAffinity is a server group policy that restricts instances belonging to the server group to the same host.<br /> |
| `anti-affinity` | ServerGroupPolicyAntiAffinity is a server group policy that restricts instances belonging to the server group to separate hosts.<br /> |
| `soft-affinity` | ServerGroupPolicySoftAffinity is a server group policy that attempts to restrict instances belonging to the server group to the same host.<br />Where it is not possible to schedule all instances on one host, they will be scheduled together on as few hosts as possible.<br /> |
| `soft-anti-affinity` | ServerGroupPolicySoftAntiAffinity is a server group policy that attempts to restrict instances belonging to the server group to separate hosts.<br /> Where it is not possible to schedule all instances to separate hosts, they will be scheduled on as many separate hosts as possible.<br /> |


#### ServerGroupResourceSpec



ServerGroupResourceSpec contains the desired state of a servergroup



_Appears in:_
- [ServerGroupSpec](#servergroupspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _[OpenStackName](#openstackname)_ | name will be the name of the created resource. If not specified, the<br />name of the ORC object will be used. |  | MaxLength: 255 <br />MinLength: 1 <br />Pattern: `^[^,]+$` <br /> |
| `policy` _[ServerGroupPolicy](#servergrouppolicy)_ | policy is the policy to use for the server group. |  | Enum: [affinity anti-affinity soft-affinity soft-anti-affinity] <br /> |
| `rules` _[ServerGroupRules](#servergrouprules)_ | rules is the rules to use for the server group. |  |  |


#### ServerGroupResourceStatus



ServerGroupResourceStatus represents the observed state of the resource.



_Appears in:_
- [ServerGroupStatus](#servergroupstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | name is a Human-readable name for the servergroup. Might not be unique. |  | MaxLength: 1024 <br /> |
| `policy` _string_ | policy is the policy of the servergroup. |  | MaxLength: 1024 <br /> |
| `projectID` _string_ | projectID is the project owner of the resource. |  | MaxLength: 1024 <br /> |
| `userID` _string_ | userID of the server group. |  | MaxLength: 1024 <br /> |
| `rules` _[ServerGroupRulesStatus](#servergrouprulesstatus)_ | rules is the rules of the server group. |  |  |


#### ServerGroupRules







_Appears in:_
- [ServerGroupResourceSpec](#servergroupresourcespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `maxServerPerHost` _integer_ | maxServerPerHost specifies how many servers can reside on a single compute host.<br />It can be used only with the "anti-affinity" policy. |  |  |


#### ServerGroupRulesStatus







_Appears in:_
- [ServerGroupResourceStatus](#servergroupresourcestatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `maxServerPerHost` _integer_ | maxServerPerHost specifies how many servers can reside on a single compute host.<br />It can be used only with the "anti-affinity" policy. |  |  |


#### ServerGroupSpec



ServerGroupSpec defines the desired state of an ORC object.



_Appears in:_
- [ServerGroup](#servergroup)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `import` _[ServerGroupImport](#servergroupimport)_ | import refers to an existing OpenStack resource which will be imported instead of<br />creating a new one. |  | MaxProperties: 1 <br />MinProperties: 1 <br /> |
| `resource` _[ServerGroupResourceSpec](#servergroupresourcespec)_ | resource specifies the desired state of the resource.<br /><br />resource may not be specified if the management policy is `unmanaged`.<br /><br />resource must be specified if the management policy is `managed`. |  |  |
| `managementPolicy` _[ManagementPolicy](#managementpolicy)_ | managementPolicy defines how ORC will treat the object. Valid values are<br />`managed`: ORC will create, update, and delete the resource; `unmanaged`:<br />ORC will import an existing resource, and will not apply updates to it or<br />delete it. | managed | Enum: [managed unmanaged] <br /> |
| `managedOptions` _[ManagedOptions](#managedoptions)_ | managedOptions specifies options which may be applied to managed objects. |  |  |
| `cloudCredentialsRef` _[CloudCredentialsReference](#cloudcredentialsreference)_ | cloudCredentialsRef points to a secret containing OpenStack credentials |  |  |


#### ServerGroupStatus



ServerGroupStatus defines the observed state of an ORC resource.



_Appears in:_
- [ServerGroup](#servergroup)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#condition-v1-meta) array_ | conditions represents the observed status of the object.<br />Known .status.conditions.type are: "Available", "Progressing"<br /><br />Available represents the availability of the OpenStack resource. If it is<br />true then the resource is ready for use.<br /><br />Progressing indicates whether the controller is still attempting to<br />reconcile the current state of the OpenStack resource to the desired<br />state. Progressing will be False either because the desired state has<br />been achieved, or because some terminal error prevents it from ever being<br />achieved and the controller is no longer attempting to reconcile. If<br />Progressing is True, an observer waiting on the resource should continue<br />to wait. |  | MaxItems: 32 <br /> |
| `id` _string_ | id is the unique identifier of the OpenStack resource. |  |  |
| `resource` _[ServerGroupResourceStatus](#servergroupresourcestatus)_ | resource contains the observed state of the OpenStack resource. |  |  |


#### ServerImport



ServerImport specifies an existing resource which will be imported instead of
creating a new one

_Validation:_
- MaxProperties: 1
- MinProperties: 1

_Appears in:_
- [ServerSpec](#serverspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `id` _string_ | id contains the unique identifier of an existing OpenStack resource. Note<br />that when specifying an import by ID, the resource MUST already exist.<br />The ORC object will enter an error state if the resource does not exist. |  | Format: uuid <br /> |
| `filter` _[ServerFilter](#serverfilter)_ | filter contains a resource query which is expected to return a single<br />result. The controller will continue to retry if filter returns no<br />results. If filter returns multiple results the controller will set an<br />error state and will not continue to retry. |  | MinProperties: 1 <br /> |


#### ServerList



ServerList contains a list of Server.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openstack.k-orc.cloud/v1alpha1` | | |
| `kind` _string_ | `ServerList` | | |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[Server](#server) array_ | items contains a list of Server. |  |  |


#### ServerPortSpec





_Validation:_
- MaxProperties: 1
- MinProperties: 1

_Appears in:_
- [ServerResourceSpec](#serverresourcespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `portRef` _[KubernetesNameRef](#kubernetesnameref)_ | portRef is a reference to a Port object. Server creation will wait for<br />this port to be created and available. |  | MaxLength: 253 <br />MinLength: 1 <br /> |


#### ServerResourceSpec



ServerResourceSpec contains the desired state of a server



_Appears in:_
- [ServerSpec](#serverspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _[OpenStackName](#openstackname)_ | name will be the name of the created resource. If not specified, the<br />name of the ORC object will be used. |  | MaxLength: 255 <br />MinLength: 1 <br />Pattern: `^[^,]+$` <br /> |
| `imageRef` _[KubernetesNameRef](#kubernetesnameref)_ | imageRef references the image to use for the server instance.<br />NOTE: This is not required in case of boot from volume. |  | MaxLength: 253 <br />MinLength: 1 <br /> |
| `flavorRef` _[KubernetesNameRef](#kubernetesnameref)_ | flavorRef references the flavor to use for the server instance. |  | MaxLength: 253 <br />MinLength: 1 <br /> |
| `userData` _[UserDataSpec](#userdataspec)_ | userData specifies data which will be made available to the server at<br />boot time, either via the metadata service or a config drive. It is<br />typically read by a configuration service such as cloud-init or ignition. |  | MaxProperties: 1 <br />MinProperties: 1 <br /> |
| `ports` _[ServerPortSpec](#serverportspec) array_ | ports defines a list of ports which will be attached to the server. |  | MaxItems: 32 <br />MaxProperties: 1 <br />MinProperties: 1 <br /> |
| `serverGroupRef` _[KubernetesNameRef](#kubernetesnameref)_ | serverGroupRef is a reference to a ServerGroup object. The server<br />will be created in the server group. |  | MaxLength: 253 <br />MinLength: 1 <br /> |
| `tags` _[ServerTag](#servertag) array_ | tags is a list of tags which will be applied to the server. |  | MaxItems: 32 <br />MaxLength: 80 <br />MinLength: 1 <br /> |


#### ServerResourceStatus



ServerResourceStatus represents the observed state of the resource.



_Appears in:_
- [ServerStatus](#serverstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | name is the human-readable name of the resource. Might not be unique. |  | MaxLength: 1024 <br /> |
| `hostID` _string_ | hostID is the host where the server is located in the cloud. |  | MaxLength: 1024 <br /> |
| `status` _string_ | status contains the current operational status of the server,<br />such as IN_PROGRESS or ACTIVE. |  | MaxLength: 1024 <br /> |
| `imageID` _string_ | imageID indicates the OS image used to deploy the server. |  | MaxLength: 1024 <br /> |
| `serverGroups` _string array_ | serverGroups is a slice of strings containing the UUIDs of the<br />server groups to which the server belongs. Currently this can<br />contain at most one entry. |  | MaxItems: 32 <br /> |
| `tags` _string array_ | tags is the list of tags on the resource. |  | MaxItems: 32 <br /> |


#### ServerSpec



ServerSpec defines the desired state of an ORC object.



_Appears in:_
- [Server](#server)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `import` _[ServerImport](#serverimport)_ | import refers to an existing OpenStack resource which will be imported instead of<br />creating a new one. |  | MaxProperties: 1 <br />MinProperties: 1 <br /> |
| `resource` _[ServerResourceSpec](#serverresourcespec)_ | resource specifies the desired state of the resource.<br /><br />resource may not be specified if the management policy is `unmanaged`.<br /><br />resource must be specified if the management policy is `managed`. |  |  |
| `managementPolicy` _[ManagementPolicy](#managementpolicy)_ | managementPolicy defines how ORC will treat the object. Valid values are<br />`managed`: ORC will create, update, and delete the resource; `unmanaged`:<br />ORC will import an existing resource, and will not apply updates to it or<br />delete it. | managed | Enum: [managed unmanaged] <br /> |
| `managedOptions` _[ManagedOptions](#managedoptions)_ | managedOptions specifies options which may be applied to managed objects. |  |  |
| `cloudCredentialsRef` _[CloudCredentialsReference](#cloudcredentialsreference)_ | cloudCredentialsRef points to a secret containing OpenStack credentials |  |  |


#### ServerStatus



ServerStatus defines the observed state of an ORC resource.



_Appears in:_
- [Server](#server)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#condition-v1-meta) array_ | conditions represents the observed status of the object.<br />Known .status.conditions.type are: "Available", "Progressing"<br /><br />Available represents the availability of the OpenStack resource. If it is<br />true then the resource is ready for use.<br /><br />Progressing indicates whether the controller is still attempting to<br />reconcile the current state of the OpenStack resource to the desired<br />state. Progressing will be False either because the desired state has<br />been achieved, or because some terminal error prevents it from ever being<br />achieved and the controller is no longer attempting to reconcile. If<br />Progressing is True, an observer waiting on the resource should continue<br />to wait. |  | MaxItems: 32 <br /> |
| `id` _string_ | id is the unique identifier of the OpenStack resource. |  |  |
| `resource` _[ServerResourceStatus](#serverresourcestatus)_ | resource contains the observed state of the OpenStack resource. |  |  |


#### ServerTag

_Underlying type:_ _string_



_Validation:_
- MaxLength: 80
- MinLength: 1

_Appears in:_
- [FilterByServerTags](#filterbyservertags)
- [ServerFilter](#serverfilter)
- [ServerResourceSpec](#serverresourcespec)



#### Subnet



Subnet is the Schema for an ORC resource.



_Appears in:_
- [SubnetList](#subnetlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openstack.k-orc.cloud/v1alpha1` | | |
| `kind` _string_ | `Subnet` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[SubnetSpec](#subnetspec)_ | spec specifies the desired state of the resource. |  |  |
| `status` _[SubnetStatus](#subnetstatus)_ | status defines the observed state of the resource. |  |  |


#### SubnetFilter



SubnetFilter specifies a filter to select a subnet. At least one parameter must be specified.

_Validation:_
- MinProperties: 1

_Appears in:_
- [SubnetImport](#subnetimport)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _[OpenStackName](#openstackname)_ | name of the existing resource |  | MaxLength: 255 <br />MinLength: 1 <br />Pattern: `^[^,]+$` <br /> |
| `description` _[NeutronDescription](#neutrondescription)_ | description of the existing resource |  | MaxLength: 255 <br />MinLength: 1 <br /> |
| `ipVersion` _[IPVersion](#ipversion)_ | ipVersion of the existing resource |  | Enum: [4 6] <br /> |
| `gatewayIP` _[IPvAny](#ipvany)_ | gatewayIP is the IP address of the gateway of the existing resource |  | MaxLength: 45 <br />MinLength: 1 <br /> |
| `cidr` _[CIDR](#cidr)_ | cidr of the existing resource |  | Format: cidr <br />MaxLength: 49 <br />MinLength: 1 <br /> |
| `ipv6` _[IPv6Options](#ipv6options)_ | ipv6 options of the existing resource |  | MinProperties: 1 <br /> |
| `networkRef` _[KubernetesNameRef](#kubernetesnameref)_ | networkRef is a reference to the ORC Network which this subnet is associated with. |  | MaxLength: 253 <br />MinLength: 1 <br /> |
| `projectRef` _[KubernetesNameRef](#kubernetesnameref)_ | projectRef is a reference to the ORC Project this resource is associated with.<br />Typically, only used by admin. |  | MaxLength: 253 <br />MinLength: 1 <br /> |
| `tags` _[NeutronTag](#neutrontag) array_ | tags is a list of tags to filter by. If specified, the resource must<br />have all of the tags specified to be included in the result. |  | MaxItems: 32 <br />MaxLength: 255 <br />MinLength: 1 <br /> |
| `tagsAny` _[NeutronTag](#neutrontag) array_ | tagsAny is a list of tags to filter by. If specified, the resource<br />must have at least one of the tags specified to be included in the<br />result. |  | MaxItems: 32 <br />MaxLength: 255 <br />MinLength: 1 <br /> |
| `notTags` _[NeutronTag](#neutrontag) array_ | notTags is a list of tags to filter by. If specified, resources which<br />contain all of the given tags will be excluded from the result. |  | MaxItems: 32 <br />MaxLength: 255 <br />MinLength: 1 <br /> |
| `notTagsAny` _[NeutronTag](#neutrontag) array_ | notTagsAny is a list of tags to filter by. If specified, resources<br />which contain any of the given tags will be excluded from the result. |  | MaxItems: 32 <br />MaxLength: 255 <br />MinLength: 1 <br /> |


#### SubnetGateway







_Appears in:_
- [SubnetResourceSpec](#subnetresourcespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _[SubnetGatewayType](#subnetgatewaytype)_ | type specifies how the default gateway will be created. `Automatic`<br />specifies that neutron will automatically add a default gateway. This is<br />also the default if no Gateway is specified. `None` specifies that the<br />subnet will not have a default gateway. `IP` specifies that the subnet<br />will use a specific address as the default gateway, which must be<br />specified in `IP`. |  | Enum: [None Automatic IP] <br /> |
| `ip` _[IPvAny](#ipvany)_ | ip is the IP address of the default gateway, which must be specified if<br />Type is `IP`. It must be a valid IP address, either IPv4 or IPv6,<br />matching the IPVersion in SubnetResourceSpec. |  | MaxLength: 45 <br />MinLength: 1 <br /> |


#### SubnetGatewayType

_Underlying type:_ _string_





_Appears in:_
- [SubnetGateway](#subnetgateway)



#### SubnetImport



SubnetImport specifies an existing resource which will be imported instead of
creating a new one

_Validation:_
- MaxProperties: 1
- MinProperties: 1

_Appears in:_
- [SubnetSpec](#subnetspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `id` _string_ | id contains the unique identifier of an existing OpenStack resource. Note<br />that when specifying an import by ID, the resource MUST already exist.<br />The ORC object will enter an error state if the resource does not exist. |  | Format: uuid <br /> |
| `filter` _[SubnetFilter](#subnetfilter)_ | filter contains a resource query which is expected to return a single<br />result. The controller will continue to retry if filter returns no<br />results. If filter returns multiple results the controller will set an<br />error state and will not continue to retry. |  | MinProperties: 1 <br /> |


#### SubnetList



SubnetList contains a list of Subnet.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openstack.k-orc.cloud/v1alpha1` | | |
| `kind` _string_ | `SubnetList` | | |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[Subnet](#subnet) array_ | items contains a list of Subnet. |  |  |


#### SubnetResourceSpec







_Appears in:_
- [SubnetSpec](#subnetspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _[OpenStackName](#openstackname)_ | name is a human-readable name of the subnet. If not set, the object's name will be used. |  | MaxLength: 255 <br />MinLength: 1 <br />Pattern: `^[^,]+$` <br /> |
| `description` _[NeutronDescription](#neutrondescription)_ | description is a human-readable description for the resource. |  | MaxLength: 255 <br />MinLength: 1 <br /> |
| `networkRef` _[KubernetesNameRef](#kubernetesnameref)_ | networkRef is a reference to the ORC Network which this subnet is associated with. |  | MaxLength: 253 <br />MinLength: 1 <br /> |
| `tags` _[NeutronTag](#neutrontag) array_ | tags is a list of tags which will be applied to the subnet. |  | MaxItems: 32 <br />MaxLength: 255 <br />MinLength: 1 <br /> |
| `ipVersion` _[IPVersion](#ipversion)_ | ipVersion is the IP version for the subnet. |  | Enum: [4 6] <br /> |
| `cidr` _[CIDR](#cidr)_ | cidr is the address CIDR of the subnet. It must match the IP version specified in IPVersion. |  | Format: cidr <br />MaxLength: 49 <br />MinLength: 1 <br /> |
| `allocationPools` _[AllocationPool](#allocationpool) array_ | allocationPools are IP Address pools that will be available for DHCP. IP<br />addresses must be in CIDR. |  | MaxItems: 32 <br /> |
| `gateway` _[SubnetGateway](#subnetgateway)_ | gateway specifies the default gateway of the subnet. If not specified,<br />neutron will add one automatically. To disable this behaviour, specify a<br />gateway with a type of None. |  |  |
| `enableDHCP` _boolean_ | enableDHCP will either enable to disable the DHCP service. |  |  |
| `dnsNameservers` _[IPvAny](#ipvany) array_ | dnsNameservers are the nameservers to be set via DHCP. |  | MaxItems: 16 <br />MaxLength: 45 <br />MinLength: 1 <br /> |
| `dnsPublishFixedIP` _boolean_ | dnsPublishFixedIP will either enable or disable the publication of<br />fixed IPs to the DNS. Defaults to false. |  |  |
| `hostRoutes` _[HostRoute](#hostroute) array_ | hostRoutes are any static host routes to be set via DHCP. |  | MaxItems: 256 <br /> |
| `ipv6` _[IPv6Options](#ipv6options)_ | ipv6 contains IPv6-specific options. It may only be set if IPVersion is 6. |  | MinProperties: 1 <br /> |
| `routerRef` _[KubernetesNameRef](#kubernetesnameref)_ | routerRef specifies a router to attach the subnet to |  | MaxLength: 253 <br />MinLength: 1 <br /> |
| `projectRef` _[KubernetesNameRef](#kubernetesnameref)_ | projectRef is a reference to the ORC Project this resource is associated with.<br />Typically, only used by admin. |  | MaxLength: 253 <br />MinLength: 1 <br /> |


#### SubnetResourceStatus







_Appears in:_
- [SubnetStatus](#subnetstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | name is the human-readable name of the subnet. Might not be unique. |  | MaxLength: 1024 <br /> |
| `description` _string_ | description is a human-readable description for the resource. |  | MaxLength: 1024 <br /> |
| `ipVersion` _integer_ | ipVersion specifies IP version, either `4' or `6'. |  |  |
| `cidr` _string_ | cidr representing IP range for this subnet, based on IP version. |  | MaxLength: 1024 <br /> |
| `gatewayIP` _string_ | gatewayIP is the default gateway used by devices in this subnet, if any. |  | MaxLength: 1024 <br /> |
| `dnsNameservers` _string array_ | dnsNameservers is a list of name servers used by hosts in this subnet. |  | MaxItems: 16 <br /> |
| `dnsPublishFixedIP` _boolean_ | dnsPublishFixedIP specifies whether the fixed IP addresses are published to the DNS. |  |  |
| `allocationPools` _[AllocationPoolStatus](#allocationpoolstatus) array_ | allocationPools is a list of sub-ranges within CIDR available for dynamic<br />allocation to ports. |  | MaxItems: 32 <br /> |
| `hostRoutes` _[HostRouteStatus](#hostroutestatus) array_ | hostRoutes is a list of routes that should be used by devices with IPs<br />from this subnet (not including local subnet route). |  | MaxItems: 256 <br /> |
| `enableDHCP` _boolean_ | enableDHCP specifies whether DHCP is enabled for this subnet or not. |  |  |
| `networkID` _string_ | networkID is the ID of the network to which the subnet belongs. |  | MaxLength: 1024 <br /> |
| `projectID` _string_ | projectID is the project owner of the subnet. |  | MaxLength: 1024 <br /> |
| `ipv6AddressMode` _string_ | ipv6AddressMode specifies mechanisms for assigning IPv6 IP addresses. |  | MaxLength: 1024 <br /> |
| `ipv6RAMode` _string_ | ipv6RAMode is the IPv6 router advertisement mode. It specifies<br />whether the networking service should transmit ICMPv6 packets. |  | MaxLength: 1024 <br /> |
| `subnetPoolID` _string_ | subnetPoolID is the id of the subnet pool associated with the subnet. |  | MaxLength: 1024 <br /> |
| `tags` _string array_ | tags optionally set via extensions/attributestags |  | MaxItems: 32 <br /> |
| `createdAt` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta)_ | createdAt shows the date and time when the resource was created. The date and time stamp format is ISO 8601 |  |  |
| `updatedAt` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta)_ | updatedAt shows the date and time when the resource was updated. The date and time stamp format is ISO 8601 |  |  |
| `revisionNumber` _integer_ | revisionNumber optionally set via extensions/standard-attr-revisions |  |  |


#### SubnetSpec



SubnetSpec defines the desired state of an ORC object.



_Appears in:_
- [Subnet](#subnet)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `import` _[SubnetImport](#subnetimport)_ | import refers to an existing OpenStack resource which will be imported instead of<br />creating a new one. |  | MaxProperties: 1 <br />MinProperties: 1 <br /> |
| `resource` _[SubnetResourceSpec](#subnetresourcespec)_ | resource specifies the desired state of the resource.<br /><br />resource may not be specified if the management policy is `unmanaged`.<br /><br />resource must be specified if the management policy is `managed`. |  |  |
| `managementPolicy` _[ManagementPolicy](#managementpolicy)_ | managementPolicy defines how ORC will treat the object. Valid values are<br />`managed`: ORC will create, update, and delete the resource; `unmanaged`:<br />ORC will import an existing resource, and will not apply updates to it or<br />delete it. | managed | Enum: [managed unmanaged] <br /> |
| `managedOptions` _[ManagedOptions](#managedoptions)_ | managedOptions specifies options which may be applied to managed objects. |  |  |
| `cloudCredentialsRef` _[CloudCredentialsReference](#cloudcredentialsreference)_ | cloudCredentialsRef points to a secret containing OpenStack credentials |  |  |


#### SubnetStatus



SubnetStatus defines the observed state of an ORC resource.



_Appears in:_
- [Subnet](#subnet)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#condition-v1-meta) array_ | conditions represents the observed status of the object.<br />Known .status.conditions.type are: "Available", "Progressing"<br /><br />Available represents the availability of the OpenStack resource. If it is<br />true then the resource is ready for use.<br /><br />Progressing indicates whether the controller is still attempting to<br />reconcile the current state of the OpenStack resource to the desired<br />state. Progressing will be False either because the desired state has<br />been achieved, or because some terminal error prevents it from ever being<br />achieved and the controller is no longer attempting to reconcile. If<br />Progressing is True, an observer waiting on the resource should continue<br />to wait. |  | MaxItems: 32 <br /> |
| `id` _string_ | id is the unique identifier of the OpenStack resource. |  |  |
| `resource` _[SubnetResourceStatus](#subnetresourcestatus)_ | resource contains the observed state of the OpenStack resource. |  |  |




#### UserDataSpec





_Validation:_
- MaxProperties: 1
- MinProperties: 1

_Appears in:_
- [ServerResourceSpec](#serverresourcespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretRef` _[KubernetesNameRef](#kubernetesnameref)_ | secretRef is a reference to a Secret containing the user data for this server. |  | MaxLength: 253 <br />MinLength: 1 <br /> |


#### Volume



Volume is the Schema for an ORC resource.



_Appears in:_
- [VolumeList](#volumelist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openstack.k-orc.cloud/v1alpha1` | | |
| `kind` _string_ | `Volume` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[VolumeSpec](#volumespec)_ | spec specifies the desired state of the resource. |  |  |
| `status` _[VolumeStatus](#volumestatus)_ | status defines the observed state of the resource. |  |  |


#### VolumeFilter



VolumeFilter defines an existing resource by its properties

_Validation:_
- MinProperties: 1

_Appears in:_
- [VolumeImport](#volumeimport)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _[OpenStackName](#openstackname)_ | name of the existing resource |  | MaxLength: 255 <br />MinLength: 1 <br />Pattern: `^[^,]+$` <br /> |
| `description` _string_ | description contains a free form description of the volume. |  | MaxLength: 255 <br />MinLength: 1 <br /> |
| `size` _integer_ | size is the size of the volume in GiB. |  | Minimum: 1 <br /> |


#### VolumeImport



VolumeImport specifies an existing resource which will be imported instead of
creating a new one

_Validation:_
- MaxProperties: 1
- MinProperties: 1

_Appears in:_
- [VolumeSpec](#volumespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `id` _string_ | id contains the unique identifier of an existing OpenStack resource. Note<br />that when specifying an import by ID, the resource MUST already exist.<br />The ORC object will enter an error state if the resource does not exist. |  | Format: uuid <br /> |
| `filter` _[VolumeFilter](#volumefilter)_ | filter contains a resource query which is expected to return a single<br />result. The controller will continue to retry if filter returns no<br />results. If filter returns multiple results the controller will set an<br />error state and will not continue to retry. |  | MinProperties: 1 <br /> |


#### VolumeList



VolumeList contains a list of Volume.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `openstack.k-orc.cloud/v1alpha1` | | |
| `kind` _string_ | `VolumeList` | | |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[Volume](#volume) array_ | items contains a list of Volume. |  |  |


#### VolumeResourceSpec



VolumeResourceSpec contains the desired state of a volume



_Appears in:_
- [VolumeSpec](#volumespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _[OpenStackName](#openstackname)_ | name will be the name of the created resource. If not specified, the<br />name of the ORC object will be used. |  | MaxLength: 255 <br />MinLength: 1 <br />Pattern: `^[^,]+$` <br /> |
| `description` _string_ | description contains a free form description of the volume. |  | MaxLength: 255 <br />MinLength: 1 <br /> |
| `size` _integer_ | size is the size of the volume, in gibibytes (GiB). |  | Minimum: 1 <br /> |


#### VolumeResourceStatus



VolumeResourceStatus represents the observed state of the resource.



_Appears in:_
- [VolumeStatus](#volumestatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | name is a Human-readable name for the volume. Might not be unique. |  | MaxLength: 1024 <br /> |
| `description` _string_ | description is a human-readable description for the resource. |  | MaxLength: 255 <br /> |
| `size` _integer_ | size is the size of the volume in GiB. |  |  |
| `status` _string_ | status represents the current status of the volume. |  | MaxLength: 1024 <br /> |
| `availabilityZone` _string_ | availabilityZone is which availability zone the volume is in. |  | MaxLength: 1024 <br /> |
| `volumeType` _string_ | volumeType is the type of volume to create, either SATA or SSD. |  | MaxLength: 1024 <br /> |
| `snapshotID` _string_ | snapshotID is the ID of the snapshot from which the volume was created |  | MaxLength: 1024 <br /> |
| `sourceVolID` _string_ | sourceVolID is the ID of another block storage volume from which the current volume was created |  | MaxLength: 1024 <br /> |
| `backupID` _string_ | backupID is the ID of the backup from which the volume was restored |  | MaxLength: 1024 <br /> |
| `userID` _string_ | userID is the ID of the user who created the volume. |  | MaxLength: 1024 <br /> |
| `bootable` _boolean_ | bootable indicates whether this is a bootable volume. |  |  |
| `encrypted` _boolean_ | encrypted denotes if the volume is encrypted. |  |  |
| `replicationStatus` _string_ | replicationStatus is the status of replication. |  | MaxLength: 1024 <br /> |
| `consistencyGroupID` _string_ | consistencyGroupID is the consistency group ID. |  | MaxLength: 1024 <br /> |
| `multiattach` _boolean_ | multiattach denotes if the volume is multi-attach capable. |  |  |
| `host` _string_ | host is the identifier of the host holding the volume. |  | MaxLength: 1024 <br /> |
| `tenantID` _string_ | tenantID is the ID of the project that owns the volume. |  | MaxLength: 1024 <br /> |
| `createdAt` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta)_ | createdAt shows the date and time when the resource was created. The date and time stamp format is ISO 8601 |  |  |
| `updatedAt` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta)_ | updatedAt shows the date and time when the resource was updated. The date and time stamp format is ISO 8601 |  |  |


#### VolumeSpec



VolumeSpec defines the desired state of an ORC object.



_Appears in:_
- [Volume](#volume)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `import` _[VolumeImport](#volumeimport)_ | import refers to an existing OpenStack resource which will be imported instead of<br />creating a new one. |  | MaxProperties: 1 <br />MinProperties: 1 <br /> |
| `resource` _[VolumeResourceSpec](#volumeresourcespec)_ | resource specifies the desired state of the resource.<br /><br />resource may not be specified if the management policy is `unmanaged`.<br /><br />resource must be specified if the management policy is `managed`. |  |  |
| `managementPolicy` _[ManagementPolicy](#managementpolicy)_ | managementPolicy defines how ORC will treat the object. Valid values are<br />`managed`: ORC will create, update, and delete the resource; `unmanaged`:<br />ORC will import an existing resource, and will not apply updates to it or<br />delete it. | managed | Enum: [managed unmanaged] <br /> |
| `managedOptions` _[ManagedOptions](#managedoptions)_ | managedOptions specifies options which may be applied to managed objects. |  |  |
| `cloudCredentialsRef` _[CloudCredentialsReference](#cloudcredentialsreference)_ | cloudCredentialsRef points to a secret containing OpenStack credentials |  |  |


#### VolumeStatus



VolumeStatus defines the observed state of an ORC resource.



_Appears in:_
- [Volume](#volume)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#condition-v1-meta) array_ | conditions represents the observed status of the object.<br />Known .status.conditions.type are: "Available", "Progressing"<br /><br />Available represents the availability of the OpenStack resource. If it is<br />true then the resource is ready for use.<br /><br />Progressing indicates whether the controller is still attempting to<br />reconcile the current state of the OpenStack resource to the desired<br />state. Progressing will be False either because the desired state has<br />been achieved, or because some terminal error prevents it from ever being<br />achieved and the controller is no longer attempting to reconcile. If<br />Progressing is True, an observer waiting on the resource should continue<br />to wait. |  | MaxItems: 32 <br /> |
| `id` _string_ | id is the unique identifier of the OpenStack resource. |  |  |
| `resource` _[VolumeResourceStatus](#volumeresourcestatus)_ | resource contains the observed state of the OpenStack resource. |  |  |


