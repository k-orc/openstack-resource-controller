#!/usr/bin/env bash

set -Eeuo pipefail

find . -type f -name '*_test.sh' -exec {} \;
