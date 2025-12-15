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

# Export variables referenced in kuttl tests.
export E2E_EXTERNAL_NETWORK_NAME
export E2E_KUTTL_OSCLOUDS=${PREPARED_OSCLOUDS}
export E2E_KUTTL_CACERT_OPT
export E2E_KUTTL_FLAVOR

kubectl kuttl test $E2E_KUTTL_DIR $E2E_KUTTL_TIMEOUT_OPT --test "$E2E_KUTTL_TEST"
