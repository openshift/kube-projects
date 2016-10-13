package apiserver

import (
	// _ "k8s.io/kubernetes/pkg/api"
	// _ "k8s.io/kubernetes/pkg/api/rest"
	// _ "k8s.io/kubernetes/pkg/apis/rbac"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	"k8s.io/kubernetes/pkg/genericapiserver"
)

type Config struct {
	GenericConfig *genericapiserver.Config

	PrivilegedKubeClient internalclientset.Interface
}

// ProjectServer contains state for a Kubernetes cluster master/api server.
type ProjectServer struct {
	GenericAPIServer *genericapiserver.GenericAPIServer
}

type completedConfig struct {
	*Config
}

// Complete fills in any fields not set that are required to have valid data. It's mutating the receiver.
func (c *Config) Complete() completedConfig {
	c.GenericConfig.Complete()

	return completedConfig{c}
}

// SkipComplete provides a way to construct a server instance without config completion.
func (c *Config) SkipComplete() completedConfig {
	return completedConfig{c}
}

// New returns a new instance of ProjectServer from the given config.
func (c completedConfig) New() (*ProjectServer, error) {
	// if c.PrivilegedKubeClient == nil {
	// 	return nil, fmt.Errorf("missing PrivilegedKubeClient")
	// }

	s, err := c.Config.GenericConfig.SkipComplete().New() // completion is done in Complete, no need for a second time
	if err != nil {
		return nil, err
	}

	m := &ProjectServer{
		GenericAPIServer: s,
	}

	if false {
		apiGroupInfo := &genericapiserver.APIGroupInfo{}
		if err := m.GenericAPIServer.InstallAPIGroup(apiGroupInfo); err != nil {
			return nil, err
		}
	}

	return m, nil
}
