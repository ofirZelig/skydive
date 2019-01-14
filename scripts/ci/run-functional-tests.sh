#!/bin/bash

set -v
set -e

DIR="$(dirname "$0")"

. "$DIR/run-tests-utils.sh"
network_setup
WITH_EBPF=true tests_run
exit $RETCODE
