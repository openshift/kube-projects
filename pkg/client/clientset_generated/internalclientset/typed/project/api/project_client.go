package api

import (
	api "k8s.io/kubernetes/pkg/api"
	registered "k8s.io/kubernetes/pkg/apimachinery/registered"
	restclient "k8s.io/kubernetes/pkg/client/restclient"
	serializer "k8s.io/kubernetes/pkg/runtime/serializer"
)

type ProjectInterface interface {
	GetRESTClient() *restclient.RESTClient
	ProjectsGetter
}

// ProjectClient is used to interact with features provided by the Project group.
type ProjectClient struct {
	*restclient.RESTClient
}

func (c *ProjectClient) Projects(namespace string) ProjectInterface {
	return newProjects(c, namespace)
}

// NewForConfig creates a new ProjectClient for the given config.
func NewForConfig(c *restclient.Config) (*ProjectClient, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	client, err := restclient.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}
	return &ProjectClient{client}, nil
}

// NewForConfigOrDie creates a new ProjectClient for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *restclient.Config) *ProjectClient {
	client, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return client
}

// New creates a new ProjectClient for the given RESTClient.
func New(c *restclient.RESTClient) *ProjectClient {
	return &ProjectClient{c}
}

func setConfigDefaults(config *restclient.Config) error {
	// if project group is not registered, return an error
	g, err := registered.Group("project")
	if err != nil {
		return err
	}
	config.APIPath = "/apis"
	if config.UserAgent == "" {
		config.UserAgent = restclient.DefaultKubernetesUserAgent()
	}
	// TODO: Unconditionally set the config.Version, until we fix the config.
	//if config.Version == "" {
	copyGroupVersion := g.GroupVersion
	config.GroupVersion = &copyGroupVersion
	//}

	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: api.Codecs}

	return nil
}

// GetRESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *ProjectClient) GetRESTClient() *restclient.RESTClient {
	if c == nil {
		return nil
	}
	return c.RESTClient
}
