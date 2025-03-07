#!/bin/bash

set -euo pipefail

# Move to top level directory
cd "$(git rev-parse --show-toplevel)"

source ./hack/init_test_env.sh

export OS_CLOUD=${E2E_OPENSTACK_CLOUD_NAME}

# External network to be used in the examples
# NOTE: we should rely on E2E_EXTERNAL_NETWORK_NAME instead
export EXAMPLE_EXTERNAL_NETWORK_NAME=${EXAMPLE_EXTERNAL_NETWORK_NAME:-private}

cd ./examples

# Populate local config
cp "${PREPARED_OSCLOUDS}" local-config/clouds.yaml
envsubst < local-config/external-network-filter.yaml.example > local-config/external-network-filter.yaml
make local-config

# Apply the cirros server example and wait for the server to be available
kubectl apply -k apply/cirros --server-side
kubectl wait --timeout=10m --for=condition=available server ${USER}-cirros-server

openstack server show "$(kubectl get server ${USER}-cirros-server -o jsonpath='{.status.id}')"
