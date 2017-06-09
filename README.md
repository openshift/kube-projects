OpenShift Project API for Kubernetes
====================================

A server to provide `projects` (ACL filtered view of namespaces) and `projectrequests` (controlled escalation to create a particular namespace and grant you rights to see/use it).


1. Start Kubernetes using the RBAC authorizer.  For testing, you can do something like: 
```bash
ALLOW_ANY_TOKEN=true ENABLE_RBAC=true hack/local-up-cluster.sh
```

3. 
```
# create the required namespace
kubectl create ns project-openshift-io

# run the project.openshift.io apiserver
kubectl create -f https://raw.githubusercontent.com/openshift/kube-projects/master/bootstrap-resources/apiregistration.k8s.io.yaml
kubectl create -f https://raw.githubusercontent.com/openshift/kube-projects/master/bootstrap-resources/rbac.authorization.k8s.io.yaml
kubectl create -f https://raw.githubusercontent.com/openshift/kube-projects/master/bootstrap-resources/core.k8s.io.yaml

# try as the cluster-admin and see them all
kubectl get projects

# try as david and see none
kubectl get projects --as david

# create a new project request as david
# TODO make this curl a URL or something first
sed 's/PROJECT_NAME/my-project/g' test/artifacts/project-request.yaml | kubectl create --as david -f -

# see the new project as david and see your project
kubectl get projects --as david
```
