#!/bin/bash

set -euo pipefail

# Path to a clouds.yaml to use for e2e tests.
# Exported because it is referenced in kuttl tests.
export E2E_OSCLOUDS=${E2E_OSCLOUDS:-/etc/openstack/clouds.yaml}

# Path to a cacert file to use to connect to OpenStack.
E2E_CACERT=${E2E_CACERT:-}

E2E_CACERT_OPT=
if [ -n "$E2E_CACERT" ]; then
    export E2E_CACERT_OPT="--from-file=cacert=${E2E_CACERT}"
fi
export E2E_CACERT_OPT

# Run kuttl tests from a specific directory.
# Defaults to empty string (all discovered kuttl directories)
E2E_KUTTL_DIR=${E2E_KUTTL_DIR:-}

# Run a specific kuttl test.
# Defaults to empty string (run all tests)
E2E_KUTTL_TEST=${E2E_KUTTL_TEST:-}

kubectl kuttl test $E2E_KUTTL_DIR --test "$E2E_KUTTL_TEST"

# HACK: Update the devstack default provider network name to match the one
# hardcoded in the cirros example
export OS_CLOUD=devstack-admin
openstack network set --name provider_net_dualstack_1 private

# Now drop admin privileges
export OS_CLOUD=devstack

cd examples

# Populate example credentials
sed "s/  devstack:/  openstack:/g" /etc/openstack/clouds.yaml > credentials/clouds.yaml
make load-credentials

# Apply the cirros server example and wait for the server to be available
kubectl apply -k apply/cirros --server-side
kubectl wait --timeout=10m --for=condition=available server ${USER}-cirros-server

openstack server show "$(kubectl get server ${USER}-cirros-server -o jsonpath='{.status.id}')"
