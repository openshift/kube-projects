OpenShift Project API for Kubernetes
====================================

A server to provide `projects` (ACL filtered view of namespaces) and `projectrequests` (controlled escalation to create a particular namespace and grant you rights to see/use it).


1. Start up the `kube-aggregator`: https://github.com/openshift/kube-aggregator

2. 
```
# start the projects API server
nice make && hack/local-up.sh

# create bootstrap rbac resources
echo `curl -k https://localhost:8445/bootstrap/rbac`  | kubectl create -f - --token=root/system:masters --server=https://localhost:6443

# register with the API federator
echo `curl -k https://localhost:8445/bootstrap/apifederation` | kubectl create -f - --token=federation-editor --server=https://localhost:8444

# log into the API federator as  yourself
# TODO requires https://github.com/openshift/origin/pull/11340
oc login https://localhost:8444 --token deads

kubectl get projects

# create a new project request
sed 's/PROJECT_NAME/my-project/g' test/artifacts/project-request.yaml | kubectl create -f -

# see the new project
kubectl get projects

oc project my-project

# see the service accounts
kubectl get sa
```
