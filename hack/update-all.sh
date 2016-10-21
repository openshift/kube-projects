#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

source "$(dirname "${BASH_SOURCE}")/lib/init.sh"

scripts=($(find ${OS_ROOT}/hack -not -name update-all.sh -name 'update-*.sh'))
for script in ${scripts[@]}; do
  echo "Executing ${script}"
  $script
done
