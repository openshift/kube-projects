package api

import (
	fmt "fmt"
	api "k8s.io/kubernetes/pkg/api"
	registered "k8s.io/kubernetes/pkg/apimachinery/registered"
	restclient "k8s.io/kubernetes/pkg/client/restclient"
	schema "k8s.io/kubernetes/pkg/runtime/schema"
	serializer "k8s.io/kubernetes/pkg/runtime/serializer"
)

type ProjectApiInterface interface {
	RESTClient() restclient.Interface
	ProjectsGetter
}

// ProjectApiClient is used to interact with features provided by the k8s.io/kubernetes/pkg/apimachinery/registered.Group group.
type ProjectApiClient struct {
	restClient restclient.Interface
}

func (c *ProjectApiClient) Projects(namespace string) ProjectInterface {
	return newProjects(c, namespace)
}

// NewForConfig creates a new ProjectApiClient for the given config.
func NewForConfig(c *restclient.Config) (*ProjectApiClient, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	client, err := restclient.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}
	return &ProjectApiClient{client}, nil
}

// NewForConfigOrDie creates a new ProjectApiClient for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *restclient.Config) *ProjectApiClient {
	client, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return client
}

// New creates a new ProjectApiClient for the given RESTClient.
func New(c restclient.Interface) *ProjectApiClient {
	return &ProjectApiClient{c}
}

func setConfigDefaults(config *restclient.Config) error {
	gv, err := schema.ParseGroupVersion("project/api")
	if err != nil {
		return err
	}
	// if project/api is not enabled, return an error
	if !registered.IsEnabledVersion(gv) {
		return fmt.Errorf("project/api is not enabled")
	}
	config.APIPath = "/apis"
	if config.UserAgent == "" {
		config.UserAgent = restclient.DefaultKubernetesUserAgent()
	}
	copyGroupVersion := gv
	config.GroupVersion = &copyGroupVersion

	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: api.Codecs}

	return nil
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *ProjectApiClient) RESTClient() restclient.Interface {
	if c == nil {
		return nil
	}
	return c.restClient
}
