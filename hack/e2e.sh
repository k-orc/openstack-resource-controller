#!/bin/bash

set -euo pipefail

# Path to a clouds.yaml to use for e2e tests.
E2E_OSCLOUDS=${E2E_OSCLOUDS:-/etc/openstack/clouds.yaml}

# Path to a cacert file to use to connect to OpenStack.
E2E_CACERT=${E2E_CACERT:-}
E2E_KUTTL_CACERT_OPT=
if [ -n "$E2E_CACERT" ]; then
    E2E_KUTTL_CACERT_OPT="--from-file=cacert=${E2E_CACERT}"
fi

# Run kuttl tests from a specific directory.
# Defaults to empty string (all discovered kuttl directories)
E2E_KUTTL_DIR=${E2E_KUTTL_DIR:-}

# Run a specific kuttl test.
# Defaults to empty string (run all tests)
E2E_KUTTL_TEST=${E2E_KUTTL_TEST:-}

# default flavor name to use for tests
export E2E_KUTTL_FLAVOR=${E2E_KUTTL_FLAVOR:-m1.tiny}

# Define a custom external network
export E2E_EXTERNAL_NETWORK_NAME=${E2E_EXTERNAL_NETWORK_NAME:-private}

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

# Export variables referenced in kuttl tests.
export E2E_KUTTL_OSCLOUDS="${creds_dir}/clouds.yaml"
export E2E_KUTTL_CACERT_OPT

# Name of the openstack credentials to use from the E2E_OSCLOUDS file
E2E_OPENSTACK_CLOUD_NAME=${E2E_OPENSTACK_CLOUD_NAME:-devstack}
# Name of the openstack admin credentials to use from the E2E_OSCLOUDS file
E2E_OPENSTACK_ADMIN_CLOUD_NAME=${E2E_OPENSTACK_ADMIN_CLOUD_NAME:-"devstack-admin-demo"}

cp "${E2E_OSCLOUDS}" "${E2E_KUTTL_OSCLOUDS}"
sed -r -i "s/^(\s+)${E2E_OPENSTACK_CLOUD_NAME}:/\1openstack:/g" "${E2E_KUTTL_OSCLOUDS}"
sed -r -i "s/^(\s+)${E2E_OPENSTACK_ADMIN_CLOUD_NAME}:/\1openstack-admin:/g" "${E2E_KUTTL_OSCLOUDS}"

kubectl kuttl test $E2E_KUTTL_DIR --test "$E2E_KUTTL_TEST"

# Now drop admin privileges
export OS_CLOUD=devstack

cd examples

# Populate local config
sed "s/  devstack:/  openstack:/g" /etc/openstack/clouds.yaml > local-config/clouds.yaml
envsubst < local-config/external-network-filter.yaml.example > local-config/external-network-filter.yaml
make local-config

# Apply the cirros server example and wait for the server to be available
kubectl apply -k apply/cirros --server-side
kubectl wait --timeout=10m --for=condition=available server ${USER}-cirros-server

openstack server show "$(kubectl get server ${USER}-cirros-server -o jsonpath='{.status.id}')"
