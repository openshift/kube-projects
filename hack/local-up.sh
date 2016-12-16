#!/bin/bash

source "$(dirname "${BASH_SOURCE}")/lib/init.sh"


${OS_ROOT}/_output/local/bin/linux/amd64/project-server \
  --authentication-kubeconfig=${OS_ROOT}/test/artifacts/local-secure-anytoken-auth.kubeconfig \
  --authorization-kubeconfig=${OS_ROOT}/test/artifacts/local-secure-anytoken-auth.kubeconfig \
  --requestheader-username-headers=X-Remote-User \
  --requestheader-group-headers=X-Remote-Group \
  --requestheader-extra-headers-prefix=X-Remote-Extra- \
  --requestheader-client-ca-file=/var/run/kubernetes/request-header-ca.crt \
  --requestheader-allowed-names=system:auth-proxy \
  --auth-user=project-server \
  --server-user=project-server \
  --kubeconfig=${OS_ROOT}/test/artifacts/local-secure-anytoken-server.kubeconfig \
  --client-ca-file=/var/run/kubernetes/client-ca.crt \
  --tls-ca-file=/var/run/kubernetes/apiserver.crt \
  --secure-port=8445
