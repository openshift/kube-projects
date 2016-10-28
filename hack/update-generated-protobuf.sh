#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

source "$(dirname "${BASH_SOURCE}")/lib/init.sh"

BINS=(
  vendor/k8s.io/kubernetes/cmd/libs/go2idl/go-to-protobuf
  vendor/k8s.io/kubernetes/cmd/libs/go2idl/go-to-protobuf/protoc-gen-gogo
)
make -C "${OS_ROOT}" WHAT="${BINS[*]}"

if [[ -z "$(which protoc)" || "$(protoc --version)" != "libprotoc 3."* ]]; then
  echo "Generating protobuf requires protoc 3.0.0-beta1 or newer. Please download and"
  echo "install the platform appropriate Protobuf package for your OS: "
  echo
  echo "  https://github.com/google/protobuf/releases"
  echo
  echo "WARNING: Protobuf changes are not being validated"
  exit 1
fi

gotoprotobuf=$(os::build::find-binary "go-to-protobuf" ${OS_ROOT})

# requires the 'proto' tag to build (will remove when ready)
# searches for the protoc-gen-gogo extension in the output directory
# satisfies import of github.com/gogo/protobuf/gogoproto/gogo.proto and the
# core Google protobuf types
KUBE_PACKAGES=(
-k8s.io/kubernetes/pkg/util/intstr
-k8s.io/kubernetes/pkg/api/resource
-k8s.io/kubernetes/pkg/runtime
-k8s.io/kubernetes/pkg/watch/versioned
-k8s.io/kubernetes/pkg/api/unversioned
-k8s.io/kubernetes/pkg/api/v1
-k8s.io/kubernetes/pkg/apis/policy/v1alpha1
-k8s.io/kubernetes/pkg/apis/extensions/v1beta1
-k8s.io/kubernetes/pkg/apis/autoscaling/v1
-k8s.io/kubernetes/pkg/apis/authorization/v1beta1
-k8s.io/kubernetes/pkg/apis/batch/v1
-k8s.io/kubernetes/pkg/apis/batch/v2alpha1
-k8s.io/kubernetes/pkg/apis/apps/v1alpha1
-k8s.io/kubernetes/pkg/apis/authentication/v1beta1
-k8s.io/kubernetes/pkg/apis/rbac/v1alpha1
-k8s.io/kubernetes/federation/apis/federation/v1beta1
-k8s.io/kubernetes/pkg/apis/certificates/v1alpha1
-k8s.io/kubernetes/pkg/apis/imagepolicy/v1alpha1
#-k8s.io/kubernetes/pkg/apis/storage/v1beta2
)

MY_PACKAGES=(
github.com/openshift/kube-projects/pkg/project/api/v1
)

PACKAGES=$(IFS=,; echo "${KUBE_PACKAGES[*]}"),$(IFS=,; echo "${MY_PACKAGES[*]}")

PATH=$(os::build::get-bin-output-path):${PATH}
  "${gotoprotobuf}" \
  --go-header-file ${OS_ROOT}/hack/boilerplate.txt \
	--packages "${PACKAGES}" \
  --proto-import="${OS_ROOT}/vendor/github.com/gogo/protobuf" \
  $@
  #--proto-import="${KUBE_ROOT}/third_party/protobuf" \
