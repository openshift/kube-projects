#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

source "$(dirname "${BASH_SOURCE}")/lib/init.sh"

REQUIRED_BINS=(
  "github.com/ugorji/go/codec/codecgen"
  "./..."
)

godep save "${REQUIRED_BINS[@]}"
