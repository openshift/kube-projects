#!/bin/bash


source "$(dirname "${BASH_SOURCE}")/lib/init.sh"
source "$(dirname "${BASH_SOURCE}")/lib/util.sh"

CERT_DIR=${CERT_DIR:-"openshift.local.config/certs"}
KUBE_CERT_DIR=/var/run/kubernetes

# Ensure CERT_DIR is created for auto-generated crt/key and kubeconfig
mkdir -p "${CERT_DIR}" &>/dev/null || mkdir -p "${CERT_DIR}"


# start_apiserver relies on certificates created by start_apiserver
function start_projects {
	${OS_ROOT}/hack/build-image.sh

	kube::util::create_signing_certkey "" "${CERT_DIR}" "apiserver" '"server auth"'
	# sign the apiserver cert to be good for the local node too, so that we can trust it
	kube::util::create_serving_certkey "" "${CERT_DIR}" "apiserver-ca" apiserver api.projects.openshift.io.svc "localhost"

	kubectl create namespace project-openshift-io || true
	kubectl -n project-openshift-io delete secret serving-apiserver > /dev/null 2>&1 || true
	kubectl -n project-openshift-io delete configmap apiserver-ca client-ca request-header-ca > /dev/null 2>&1 || true
	kubectl -n project-openshift-io delete -f ${OS_ROOT}/bootstrap-resources > /dev/null 2>&1 || true

	kubectl -n project-openshift-io create secret tls serving-apiserver --cert="${CERT_DIR}/serving-apiserver.crt" --key="${CERT_DIR}/serving-apiserver.key"
	kubectl -n project-openshift-io create configmap apiserver-ca --from-file="ca.crt=${CERT_DIR}/apiserver-ca.crt" 
	kubectl -n project-openshift-io create configmap client-ca --from-file="ca.crt=${KUBE_CERT_DIR}/client-ca.crt" 
	kubectl -n project-openshift-io create configmap request-header-ca --from-file="ca.crt=${KUBE_CERT_DIR}/request-header-ca.crt" 

	kubectl create -n project-openshift-io -f ${OS_ROOT}/bootstrap-resources --validate=false  || true
}

kube::util::test_openssl_installed
kube::util::test_cfssl_installed

start_projects

echo "projects.openshift.io created"
