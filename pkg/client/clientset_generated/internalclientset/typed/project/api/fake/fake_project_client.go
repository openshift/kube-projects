package fake

import (
	api "github.com/openshift/kube-projects/pkg/client/clientset_generated/internalclientset/typed/project/api"
	restclient "k8s.io/kubernetes/pkg/client/restclient"
	core "k8s.io/kubernetes/pkg/client/testing/core"
)

type FakeProjectApi struct {
	*core.Fake
}

func (c *FakeProjectApi) Projects(namespace string) api.ProjectInterface {
	return &FakeProjects{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeProjectApi) RESTClient() restclient.Interface {
	var ret *restclient.RESTClient
	return ret
}
