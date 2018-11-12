#!/bin/bash

if [ -z "${TEST_ARG_1+x}" ]; then 
    echo "TEST_ARG_1 isn't set in the called script environment"
    exit 1
fi

set -eu -o pipefail

echo "TEST_ENV_1 is set to: ${TEST_ENV_1}"

