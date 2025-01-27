#!/bin/bash

set -euo pipefail

kubectl kuttl test

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
