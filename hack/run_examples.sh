#!/bin/bash

set -euo pipefail

# Move to top level directory
REAL_PATH=$(realpath $0)
cd "$(dirname "$REAL_PATH")/.."

source ./hack/init_test_env.sh

export OS_CLOUD=${E2E_OPENSTACK_CLOUD_NAME}

export E2E_EXTERNAL_NETWORK_NAME

cd ./examples

# Populate local config
cp "${PREPARED_OSCLOUDS}" local-config/clouds.yaml
envsubst < local-config/external-network-filter.yaml.example > local-config/external-network-filter.yaml
make local-config

# Apply the cirros server example and wait for the server to be available
kubectl apply -k apply/cirros --server-side
kubectl wait --timeout=10m --for=condition=available server ${USER}-cirros-server

openstack server show "$(kubectl get server ${USER}-cirros-server -o jsonpath='{.status.id}')"
