OpenShift Project API for Kubernetes
====================================

Start the Kubernetes API server using: `ALLOW_ANY_TOKEN=true ENABLE_RBAC=true hack/local-up-cluster.sh`, then start this by running `project-server --kubeconfig=test/artifacts/local-secure-anytoken-kubeconfig --client-ca-file=/var/run/kubernetes/apiserver.crt --loglevel=8`.

You can use `kubectl` or `oc` against the server with `oc login localhost:8444 --token=your-user`.
