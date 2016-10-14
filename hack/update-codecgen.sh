#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

source "$(dirname "${BASH_SOURCE}")/lib/init.sh"

export CODECGEN_GENERATED_FILES="
pkg/project/api/types.generated.go
pkg/project/api/v1/types.generated.go
"

export CODECGEN_PREFIX=github.com/openshift/kube-projects

${OS_ROOT}/hack/run-codecgen.sh
