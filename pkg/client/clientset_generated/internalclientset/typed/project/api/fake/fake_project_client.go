package fake

import (
	api "github.com/openshift/kube-projects/pkg/client/clientset_generated/internalclientset/typed/project/api"
	restclient "k8s.io/kubernetes/pkg/client/restclient"
	core "k8s.io/kubernetes/pkg/client/testing/core"
)

type FakeProject struct {
	*core.Fake
}

func (c *FakeProject) Projects(namespace string) api.ProjectInterface {
	return &FakeProjects{c, namespace}
}

// GetRESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeProject) GetRESTClient() *restclient.RESTClient {
	return nil
}
