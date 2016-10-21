#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

source "$(dirname "${BASH_SOURCE}")/lib/init.sh"

REQUIRED_BINS=(
  "github.com/ugorji/go/codec/codecgen"
  "k8s.io/kubernetes/cmd/libs/go2idl/client-gen"
  "k8s.io/kubernetes/cmd/libs/go2idl/conversion-gen"
  "k8s.io/kubernetes/cmd/libs/go2idl/deepcopy-gen"
  "k8s.io/kubernetes/cmd/genswaggertypedocs"
  "k8s.io/kubernetes/cmd/libs/go2idl/go-to-protobuf"
  "k8s.io/kubernetes/cmd/libs/go2idl/go-to-protobuf/protoc-gen-gogo"
  "./..."
)

godep save "${REQUIRED_BINS[@]}"
