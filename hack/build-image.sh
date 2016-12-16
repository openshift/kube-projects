#!/bin/bash

source "$(dirname "${BASH_SOURCE}")/lib/init.sh"

# Register function to be called on EXIT to remove generated binary.
function cleanup {
  rm "${OS_ROOT}/docker/project-server"
}
trap cleanup EXIT

cp -v ${OS_ROOT}/_output/local/bin/linux/amd64/project-server "${OS_ROOT}/docker/project-server"
docker build -t openshift/origin-kube-projects:latest ${OS_ROOT}/docker
