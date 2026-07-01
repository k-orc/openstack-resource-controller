# DNS Zones (DNSZone)

The `DNSZone` resource manages DNS zones in OpenStack Designate. It allows you to declaratively create, update, and delete primary and secondary DNS zones, or import existing zones for read-only access.

---

## Core Concepts

In OpenStack Designate, a DNS zone holds DNS records (such as A, AAAA, MX, TXT, etc.). ORC supports both **PRIMARY** and **SECONDARY** zones:
* **PRIMARY** zones are master zones where DNS records are managed directly within OpenStack Designate.
* **SECONDARY** zones are read-only copies of zones that automatically perform zone transfers from external master DNS servers.

### Domain Name Syntax
All DNS zone names in OpenStack Designate and ORC **must end with a trailing period** (e.g., `example.com.`). This is enforced by Kubernetes API validation rules.

---

## Management Policies

Like all ORC resources, `DNSZone` supports two management policies: `managed` and `unmanaged`.

### 1. Managed Zone (Default)

In the `managed` policy, ORC handles the entire lifecycle of the DNS zone in OpenStack Designate.
* **Creation**: ORC creates the zone if it does not exist.
* **Update**: ORC synchronizes specifications (like description, email, TTL, and masters) to Designate. (Note: `name` and `type` are immutable).
* **Deletion**: On deletion of the Kubernetes resource, the corresponding Designate zone is deleted (unless `managedOptions.onDelete` is set to `detach`).

#### Option A: Managed Primary Zone

```yaml
apiVersion: openstack.k-orc.cloud/v1alpha1
kind: DNSZone
metadata:
  name: primary-zone
spec:
  cloudCredentialsRef:
    secretName: openstack-clouds
    cloudName: openstack
  managementPolicy: managed
  resource:
    # name specifies the name of the DNS Zone. Must end with a period.
    # Defaults to the ORC object name if not specified.
    # Immutable after creation.
    name: primary.example.com.

    # email is the email address of the administrator for the zone.
    # Required for PRIMARY zones. Must be omitted for SECONDARY zones.
    email: admin@example.com

    # description is a human-readable description for the DNS Zone.
    description: "Complete managed primary DNS zone example"

    # ttl is the Time To Live for the zone in seconds.
    ttl: 3600

    # type specifies the type of the zone. Can be 'PRIMARY' or 'SECONDARY'.
    # Immutable after creation.
    type: PRIMARY
```

#### Option B: Managed Secondary Zone

```yaml
apiVersion: openstack.k-orc.cloud/v1alpha1
kind: DNSZone
metadata:
  name: secondary-zone
spec:
  cloudCredentialsRef:
    secretName: openstack-clouds
    cloudName: openstack
  managementPolicy: managed
  resource:
    # name specifies the name of the DNS Zone. Must end with a period.
    name: secondary.example.com.

    # type specifies the type of the zone.
    type: SECONDARY

    # masters specifies zone masters from which zone transfers are performed.
    # Required for SECONDARY zones. Must be omitted for PRIMARY zones.
    masters:
      - 192.0.2.1
      - 192.0.2.2

    description: "Complete managed secondary DNS zone example"
    ttl: 3600
```

### 2. Unmanaged Import Workflow

In the `unmanaged` policy, ORC imports an existing DNS zone from OpenStack Designate. ORC will **never** modify or delete the OpenStack resource; it is imported for read-only status propagation.

You can import an existing zone using either its **ID (UUID)** or by matching its **properties (filters)**.

#### Option A: Import by ID
Use this option when you know the exact UUID of the existing zone.

```yaml
apiVersion: openstack.k-orc.cloud/v1alpha1
kind: DNSZone
metadata:
  name: imported-zone-by-id
spec:
  cloudCredentialsRef:
    secretName: openstack-clouds
    cloudName: openstack
  managementPolicy: unmanaged
  import:
    id: "12345678-1234-1234-1234-1234567890ab"
```

#### Option B: Import by Filter
Use this option when you want to look up an existing zone based on its properties. The filter **must resolve to exactly one resource** in OpenStack Designate, otherwise ORC will enter an error state.

```yaml
apiVersion: openstack.k-orc.cloud/v1alpha1
kind: DNSZone
metadata:
  name: imported-zone-by-filter
spec:
  cloudCredentialsRef:
    secretName: openstack-clouds
    cloudName: openstack
  managementPolicy: unmanaged
  import:
    filter:
      name: existing-zone.example.com.
      email: admin@example.com
      ttl: 3600
```

---

## Validation Rules & Immutability

The `DNSZone` Custom Resource Definition (CRD) implements strict validation via Common Expression Language (CEL) and OpenAPI schemas:

* **Name Validation**:
  * Must end with a trailing period (`.`).
  * Immutable. Once created, you cannot change the zone name in the specification.
  * Defaults to the ORC object name (with a trailing period appended by the user) if not explicitly set.
* **Type Validation**:
  * Allowed values are `PRIMARY` and `SECONDARY`.
  * Immutable. Once created, you cannot change the zone type.
* **Email Validation**:
  * Required when `type` is `PRIMARY`.
  * Must be omitted (not specified) when `type` is `SECONDARY`.
  * Must be a valid email format.
  * Maximum length of `255` characters.
* **Masters Validation**:
  * Required when `type` is `SECONDARY` (must specify at least one master IP address).
  * Must be omitted (not specified) when `type` is `PRIMARY`.
  * Maximum of `32` items, each with a maximum length of `255` characters.
* **TTL Validation**:
  * Must be an integer between `1` and `2147483647` (inclusive).
* **Description Validation**:
  * Length must be between `1` and `255` characters.

---

## Status Conditions

To check the reconciliation and readiness of your `DNSZone`, inspect its status conditions:

```bash
kubectl get dnszone primary-zone -o yaml
```

### 1. `Available` Condition
* `True`: The DNS zone is created and active (`ACTIVE` status in Designate).
* `False`: The DNS zone is not ready for use. Common reasons include:
  * `Progressing`: The zone is currently in a `PENDING` state in Designate. ORC is actively polling until it transitions to `ACTIVE`.
  * `UnrecoverableError`: The zone is in an `ERROR` state in Designate, or the import configuration points to a non-existent ID or filter.
  * `InvalidConfiguration`: The configuration is invalid or there was an authentication/client issue.

### 2. `Progressing` Condition
* `True`: ORC is still performing operations or polling for updates (e.g., waiting for the zone to become `ACTIVE` from a `PENDING` state).
* `False`: ORC has completed reconciliation. The spec matches the observed status or a terminal error has been reached.

---

## Troubleshooting

Here are common issues you might encounter with `DNSZone` resources and how to solve them:

### 1. Validation Failures on Creation

**Symptom:**
When applying the YAML, you get an API rejection error:
```
Error from server (BadRequest): error when creating "dnszone.yaml": DNSZone.openstack.k-orc.cloud "my-zone" is invalid: spec.resource.name: Invalid value: "my-zone": name must end with a period
```

**Solution:**
Ensure that your `spec.resource.name` ends with a period (e.g., `my-zone.example.com.`). If you omit `spec.resource.name`, the controller will default to the Kubernetes object name, but the name must still be valid as a DNS Zone name (ending in a `.`). Therefore, if you omit `spec.resource.name`, the Kubernetes object's name itself is used, but because Kubernetes resource names cannot end with a period, you must explicitly specify `spec.resource.name` with the trailing period.

---

### 2. Zone Creation Hangs or Enters Terminal Error (409 Conflict)

**Symptom:**
The resource reports `Available=False` and `Progressing=False` with a message containing:
```
Conflicting zone already exists in OpenStack Designate
```

**Cause:**
A DNS Zone with the same domain name already exists in the target OpenStack tenant. Designate does not allow duplicate zone names.

**Solution:**
* If you want to take over and manage this existing zone, you should change your `managementPolicy` to `unmanaged` and use the `import` mechanism.
* If you want to create a new managed zone, choose a unique domain name.

---

### 3. Unmanaged Import Fails to Find Zone

**Symptom:**
The imported `DNSZone` is stuck in an error state with the message:
```
referenced resource does not exist in OpenStack
```

**Solution:**
* Verify that the ID in `spec.import.id` is correct and matches an existing zone in Designate.
* If importing by `spec.import.filter`, verify that the filter matches **exactly one** zone. If it matches zero or multiple zones, the import will fail. Use the OpenStack CLI to verify:
  ```bash
  openstack zone list --name existing-zone.example.com.
  ```

---

### 4. Zone stuck in PENDING status

**Symptom:**
`Available` is `False`, `Progressing` is `True`, and the message is `Waiting for resource to become available in OpenStack`.

**Cause:**
Designate is asynchronously processing the zone creation/update or communicating with backend DNS nameservers.

**Solution:**
This is normal behavior. ORC is actively polling OpenStack. Wait a few moments for Designate to transition the zone status to `ACTIVE`. If it remains in this state for a prolonged period, check the Designate API service status and your OpenStack logs.
