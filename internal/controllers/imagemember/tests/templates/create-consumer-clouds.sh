#!/bin/bash
# Creates the secret consumer-clouds with a cloud "consumer" for the user and
# project created by create-consumer.tmpl.
set -euo pipefail
auth_url=$(grep -m1 'auth_url:' "${E2E_KUTTL_OSCLOUDS}" | awk '{print $2}')
region=$(grep -m1 'region_name:' "${E2E_KUTTL_OSCLOUDS}" | awk '{print $2}' || true)
clouds=$(mktemp)
cat > "$clouds" <<CLOUDS
clouds:
  consumer:
    auth:
      auth_url: ${auth_url}
      username: ${E2E_KUTTL_CURRENT_TEST}
      password: ${E2E_KUTTL_CURRENT_TEST}-Pa55
      project_name: ${E2E_KUTTL_CURRENT_TEST}
      user_domain_name: Default
      project_domain_name: Default
    region_name: ${region:-RegionOne}
    identity_api_version: 3
CLOUDS
kubectl create secret generic consumer-clouds -n "${NAMESPACE}" --from-file=clouds.yaml="$clouds" ${E2E_KUTTL_CACERT_OPT}
rm -f "$clouds"
