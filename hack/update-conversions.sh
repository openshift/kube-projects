#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

source "$(dirname "${BASH_SOURCE}")/lib/init.sh"

# Register function to be called on EXIT to remove generated binary.
function cleanup {
  rm -f "${CONVERSIONGEN:-}"
}
trap cleanup EXIT

echo "Building conversion-gen"
CONVERSIONGEN="${PWD}/conversion-gen-binary"
go build -o "${CONVERSIONGEN}" ./vendor/k8s.io/kubernetes/cmd/libs/go2idl/conversion-gen

PREFIX=github.com/openshift/kube-projects

INPUT_DIRS=$(
  find . -not  \( \( -wholename '*/vendor/*' \) -prune \) -name '*.go' | \
  xargs grep --color=never -l '+k8s:conversion-gen=' | \
  xargs -n1 dirname | \
  sed "s,^\.,${PREFIX}," | \
  sort -u | \
  paste -sd,
)

${CONVERSIONGEN} --logtostderr \
  --go-header-file ${OS_ROOT}/hack/boilerplate.txt \
  --output-file-base zz_generated.conversion \
  --build-tag ignore_autogenerated_openshift \
  --input-dirs ${INPUT_DIRS[@]}
