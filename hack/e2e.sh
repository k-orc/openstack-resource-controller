#!/bin/bash

set -euo pipefail

# Move to top level directory
cd "$(git rev-parse --show-toplevel)"

source ./hack/init_test_env.sh

# Run kuttl tests from a specific directory.
# Defaults to empty string (all discovered kuttl directories)
E2E_KUTTL_DIR=${E2E_KUTTL_DIR:-}

# Run a specific kuttl test.
# Defaults to empty string (run all tests)
E2E_KUTTL_TEST=${E2E_KUTTL_TEST:-}

# Flavor name to use for tests
E2E_KUTTL_FLAVOR=${E2E_KUTTL_FLAVOR:-m1.tiny}

E2E_KUTTL_CACERT_OPT=
if [ -n "$E2E_CACERT" ]; then
    E2E_KUTTL_CACERT_OPT="--from-file=cacert=${E2E_CACERT}"
fi

E2E_KUTTL_TIMEOUT=${E2E_KUTTL_TIMEOUT:-}
E2E_KUTTL_TIMEOUT_OPT=
if [ -n "$E2E_KUTTL_TIMEOUT" ]; then
    E2E_KUTTL_TIMEOUT_OPT="--timeout $E2E_KUTTL_TIMEOUT"
fi

# Skip deleting resources based on an option
# all: delete all resources
# success: delete only cases that succeeded
# none: never delete
E2E_KUTTL_DELETE_OPTION=${E2E_KUTTL_DELETE_OPTION:-success}
E2E_KUTTL_DELETE_OPT=
if [ -n "$E2E_KUTTL_DELETE_OPTION" ]; then
    E2E_KUTTL_DELETE_OPT="--delete ${E2E_KUTTL_DELETE_OPTION}"
fi

# Add the OpenStack version to the test run labels if it is set.
# https://releases.openstack.org/
# 2027.1 Indri
# 2026.2 Hibiscus
# 2026.1 Gazpacho
# 2025.2 Flamingo
# 2025.1 Epoxy
KNOWN_RELEASES=("2025.1" "2025.2" "2026.1" "2026.2" "2027.1")
labels=
if [ -n "${E2E_OPENSTACK_VERSION:-}" ]; then
    OS_RELEASE="$E2E_OPENSTACK_VERSION"

    for release in "${KNOWN_RELEASES[@]}"; do
        if [[ ! "$OS_RELEASE" < "$release" ]]; then
            labels="${labels:+$labels,}min-${release}=true"
        fi
        if [[ ! "$OS_RELEASE" > "$release" ]]; then
            labels="${labels:+$labels,}max-${release}=true"
        fi
    done
fi

# Skip the tests that need admin credentials (their TestCase selects on admin)
E2E_KUTTL_SKIP_ADMIN=${E2E_KUTTL_SKIP_ADMIN:-false}
if [ "$E2E_KUTTL_SKIP_ADMIN" = "true" ]; then
    labels="${labels:+$labels,}admin=false"
fi

E2E_KUTTL_TEST_RUN_LABELS=
if [ -n "$labels" ]; then
    E2E_KUTTL_TEST_RUN_LABELS="--test-run-labels $labels"
fi

# Export variables referenced in kuttl tests.
export E2E_EXTERNAL_NETWORK_NAME
export E2E_KUTTL_OSCLOUDS=${PREPARED_OSCLOUDS}
export E2E_KUTTL_CACERT_OPT
export E2E_KUTTL_FLAVOR

set -x
kubectl kuttl test $E2E_KUTTL_DIR $E2E_KUTTL_TIMEOUT_OPT $E2E_KUTTL_DELETE_OPT $E2E_KUTTL_TEST_RUN_LABELS --test "$E2E_KUTTL_TEST"
set +x
