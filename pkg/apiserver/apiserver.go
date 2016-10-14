package apiserver

import (
	// _ "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/rest"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	"k8s.io/kubernetes/pkg/genericapiserver"

	projectapi "github.com/openshift/kube-projects/pkg/project/api"
	projectapiv1 "github.com/openshift/kube-projects/pkg/project/api/v1"
	projectrequeststorage "github.com/openshift/kube-projects/pkg/project/registry/projectrequest"
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

	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(projectapi.GroupName)
	apiGroupInfo.GroupMeta.GroupVersion = projectapiv1.SchemeGroupVersion

	v1storage := map[string]rest.Storage{}
	v1storage["projectrequests"] = projectrequeststorage.NewREST("", c.Config.GenericConfig.Authorizer, c.PrivilegedKubeClient)

	apiGroupInfo.VersionedResourcesStorageMap[projectapiv1.SchemeGroupVersion.Version] = v1storage

	if err := m.GenericAPIServer.InstallAPIGroup(&apiGroupInfo); err != nil {
		return nil, err
	}

	return m, nil
}
