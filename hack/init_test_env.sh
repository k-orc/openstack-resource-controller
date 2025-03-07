#!/bin/bash

set -euo pipefail

# Path to a clouds.yaml to use for e2e tests.
E2E_OSCLOUDS=${E2E_OSCLOUDS:-/etc/openstack/clouds.yaml}

# Path to a cacert file to use to connect to OpenStack.
E2E_CACERT=${E2E_CACERT:-}

# Name of the openstack credentials to use from the E2E_OSCLOUDS file
E2E_OPENSTACK_CLOUD_NAME=${E2E_OPENSTACK_CLOUD_NAME:-devstack}

# Name of the openstack admin credentials to use from the E2E_OSCLOUDS file
E2E_OPENSTACK_ADMIN_CLOUD_NAME=${E2E_OPENSTACK_ADMIN_CLOUD_NAME:-"devstack-admin-demo"}

# Define a custom external network
E2E_EXTERNAL_NETWORK_NAME=${E2E_EXTERNAL_NETWORK_NAME:-public}

creds_dir=$(mktemp -d)

function logresources() {
    # Log all resources if exiting with an error
    if [ $? != 0 ]; then
        kubectl get openstack -o yaml -A
    fi
}

function cleanup() {
    # logresources must be called first as it checks the exit code
    logresources
    rm -rf -- "$creds_dir"
}

trap cleanup EXIT

PREPARED_OSCLOUDS="${creds_dir}/clouds.yaml"

cp "${E2E_OSCLOUDS}" "${PREPARED_OSCLOUDS}"
if sed --version 2>/dev/null | grep -q GNU; then
    sed -E -i "s/^([[:space:]]+)${E2E_OPENSTACK_CLOUD_NAME}:/\1openstack:/g" "${PREPARED_OSCLOUDS}"
    sed -E -i "s/^([[:space:]]+)${E2E_OPENSTACK_ADMIN_CLOUD_NAME}:/\1openstack-admin:/g" "${PREPARED_OSCLOUDS}"
else
    sed -E -i' ' "s/^([[:space:]]+)${E2E_OPENSTACK_CLOUD_NAME}:/\1openstack:/g" "${PREPARED_OSCLOUDS}"
    sed -E -i' ' "s/^([[:space:]]+)${E2E_OPENSTACK_ADMIN_CLOUD_NAME}:/\1openstack-admin:/g" "${PREPARED_OSCLOUDS}"
fi
