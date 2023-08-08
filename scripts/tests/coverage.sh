#!/bin/bash

set -Eeuo pipefail

SCRIPT_NAME="$(basename "$0")"
SCRIPT_DIR="$(dirname "$0")"
REPO_ROOT="$(cd "${SCRIPT_DIR}" && git rev-parse --show-toplevel)"
SCRIPTS_DIR="${REPO_ROOT}/scripts"
COVER_DIR=${REPO_ROOT}/coverage

source "${SCRIPTS_DIR}/helpers-source.sh"

echo "${SCRIPT_NAME} is running... "

export GO111MODULE=on

rm -rf "${COVER_DIR}"
mkdir -p "${COVER_DIR}"

COVERPROFILE="${COVER_DIR}/unit.cov"
GOEXPERIMENT=nocoverageredesign go test --count=1 -coverpkg=./... -coverprofile "${COVERPROFILE}" -covermode=atomic ./... >tests-report.json

# Get total from coverage report
go tool cover \
    -func "${COVERPROFILE}"

go tool cover \
    -func "${COVERPROFILE}" \
        | grep total \
        | awk '{print "Code coverage: " $3}'


echo "${SCRIPT_NAME} is done... "
