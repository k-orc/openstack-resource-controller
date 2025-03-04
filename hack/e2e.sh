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

E2E_KUTTL_CACERT_OPT=
if [ -n "$E2E_CACERT" ]; then
    E2E_KUTTL_CACERT_OPT="--from-file=cacert=${E2E_CACERT}"
fi

# Export variables referenced in kuttl tests.
export E2E_KUTTL_OSCLOUDS=${PREPARED_OSCLOUDS}
export E2E_KUTTL_CACERT_OPT

kubectl kuttl test $E2E_KUTTL_DIR --test "$E2E_KUTTL_TEST"
