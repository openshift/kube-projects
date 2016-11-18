#!/bin/bash

source "$(dirname "${BASH_SOURCE}")/lib/init.sh"


${OS_ROOT}/_output/local/bin/linux/amd64/project-server \
  --authentication-kubeconfig=${OS_ROOT}/test/artifacts/local-secure-anytoken.kubeconfig \
  --authorization-kubeconfig=${OS_ROOT}/test/artifacts/local-secure-anytoken.kubeconfig \
  --auth-user=project-server \
  --server-user=project-server \
  --kubeconfig=${OS_ROOT}/test/artifacts/local-secure-anytoken.kubeconfig \
  --client-ca-file=/var/run/kubernetes/apiserver.crt \
  --tls-ca-file=/var/run/kubernetes/apiserver.crt \
  --secure-port=8445
