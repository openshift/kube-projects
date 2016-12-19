OpenShift Project API for Kubernetes
====================================

A server to provide `projects` (ACL filtered view of namespaces) and `projectrequests` (controlled escalation to create a particular namespace and grant you rights to see/use it).


1. Start Kubernetes using the RBAC authorizer.  For testing, you can do something like: 
```bash
API_HOST=<your-ip> API_HOST_IP=<your-ip> KUBE_ENABLE_CLUSTER_DNS=true ALLOW_ANY_TOKEN=true ENABLE_RBAC=true hack/local-up-cluster.sh
```

2. Start `kubernetes-discovery`.  For testing, you can do something like:
```bash
API_HOST=<your-ip> API_HOST_IP=<your-ip> hack/local-up-discovery.sh
```

2. 
```
# start the projects API server
KUBECONFIG=/var/run/kubernetes/admin-discovery.kubeconfig hack/install.sh

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
