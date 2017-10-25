#!/bin/bash

set -e

OS_ROOT=$(dirname ${BASH_SOURCE})/..

# Register function to be called on EXIT to remove generated binary.
function cleanup {
  rm "${OS_ROOT}/docker/kube-projects"
}
trap cleanup EXIT

cp -v ${OS_ROOT}/_output/bin/kube-projects "${OS_ROOT}/docker/kube-projects"
docker build -t openshift/origin-kube-projects:latest ${OS_ROOT}/docker
