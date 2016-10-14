#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

source "$(dirname "${BASH_SOURCE}")/lib/init.sh"

# Register function to be called on EXIT to remove generated binary.
function cleanup {
  rm -f "${CLIENTGEN:-}"
}
trap cleanup EXIT

echo "Building client-gen"
CLIENTGEN="${PWD}/client-gen-binary"
go build -o "${CLIENTGEN}" ./vendor/k8s.io/kubernetes/cmd/libs/go2idl/client-gen

PREFIX=github.com/openshift/kube-projects
INPUT_BASE="--input-base ${PREFIX}"
INPUT_APIS=(
./pkg/project/api
)
INPUT="--input ${INPUT_APIS[@]}"
CLIENTSET_PATH="--clientset-path ${PREFIX}/pkg/client/clientset_generated"
BOILERPLATE="--go-header-file ${OS_ROOT}/hack/boilerplate.txt"

# regular client code
${CLIENTGEN} ${INPUT_BASE} ${INPUT} ${CLIENTSET_PATH} ${BOILERPLATE}
# client code for testdata
${CLIENTGEN} -t ${INPUT_BASE} ${INPUT} ${CLIENTSET_PATH} ${BOILERPLATE}
