#!/bin/bash

# Copyright 2015 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit
set -o nounset
set -o pipefail

generated_files=(${CODECGEN_GENERATED_FILES})

# We only work for deps within this prefix.
my_prefix="${CODECGEN_PREFIX}"

# Register function to be called on EXIT to remove codecgen
# binary.
function cleanup {
  rm -f "${CODECGEN:-}"
}
trap cleanup EXIT

# Precompute dependencies for all directories.
# Then sort all files in the dependency order.
number=${#generated_files[@]}
result=""
for (( i=0; i<number; i++ )); do
  visited[${i}]=false
  file="${generated_files[${i}]/\.generated\.go/.go}"
  deps[${i}]=$(go list -f '{{range .Deps}}{{.}}{{"\n"}}{{end}}' ${file} | grep "^${my_prefix}")
done

# NOTE: depends function assumes that the whole repository is under
# $my_prefix - it will NOT work if that is not true.
function depends {
  rhs="$(dirname ${generated_files[$2]/#./${my_prefix}})"
  for dep in ${deps[$1]}; do
    if [[ "${dep}" == "${rhs}" ]]; then
      return 0
    fi
  done
  return 1
}

function tsort {
  visited[$1]=true
  local j=0
  for (( j=0; j<number; j++ )); do
    if ! ${visited[${j}]}; then
      if depends "$1" ${j}; then
        tsort $j
      fi
    fi
  done
  result="${result} $1"
}
echo "Building dependencies"
for (( i=0; i<number; i++ )); do
  if ! ${visited[${i}]}; then
    tsort ${i}
  fi
done
index=(${result})

haveindex=${index:-}
if [[ -z ${haveindex} ]]; then
  echo No files found for $0
  exit 1
fi

echo "Building codecgen"
CODECGEN="${PWD}/codecgen_binary"
go build -o "${CODECGEN}" ./vendor/github.com/ugorji/go/codec/codecgen

# Running codecgen fails if some of the files doesn't compile.
# Thus (since all the files are completely auto-generated and
# not required for the code to be compilable, we first remove
# them and then regenerate them.
for (( i=0; i < number; i++ )); do
  rm -f "${generated_files[${i}]}"
done

# Generate files in the dependency order.
for current in "${index[@]}"; do
  generated_file=${generated_files[${current}]}
  initial_dir=${PWD}
  file=${generated_file/\.generated\.go/.go}
  echo "processing ${file}"
  # codecgen work only if invoked from directory where the file
  # is located.
  pushd "$(dirname ${file})" > /dev/null
  base_file=$(basename "${file}")
  base_generated_file=$(basename "${generated_file}")
  # We use '-d 1234' flag to have a deterministic output every time.
  # The constant was just randomly chosen.
  ${CODECGEN} -d 1234 -o "${base_generated_file}" "${base_file}"
  popd > /dev/null
done
