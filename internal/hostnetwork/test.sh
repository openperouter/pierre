#!/usr/bin/env sh
#
# Run CNI plugin tests.
#
# This needs sudo, as we'll be creating net interfaces.
#
set -e

sudo -E sh -c "umask 0; PATH=${GOPATH}/bin:$(pwd)/bin:${PATH} go test -v -race $*"
