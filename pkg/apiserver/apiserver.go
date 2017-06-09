package apiserver

import (
	"fmt"
	"time"

	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	kubeinformers "k8s.io/kubernetes/pkg/client/informers/informers_generated/internalversion"
	rbacauthorizer "k8s.io/kubernetes/plugin/pkg/auth/authorizer/rbac"

	projectapi "github.com/openshift/kube-projects/pkg/project/api"
	projectapiv1 "github.com/openshift/kube-projects/pkg/project/api/v1"
	authcache "github.com/openshift/kube-projects/pkg/project/auth"
	projectstorage "github.com/openshift/kube-projects/pkg/project/registry/project"
	projectrequeststorage "github.com/openshift/kube-projects/pkg/project/registry/projectrequest"
)

type Config struct {
	GenericConfig *genericapiserver.Config

	KubeClient internalclientset.Interface
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
	c.GenericConfig.Version = nil

	c.GenericConfig.Complete()

	return completedConfig{c}
}

// SkipComplete provides a way to construct a server instance without config completion.
func (c *Config) SkipComplete() completedConfig {
	return completedConfig{c}
}

// New returns a new instance of ProjectServer from the given config.
func (c completedConfig) New() (*ProjectServer, error) {
	if c.KubeClient == nil {
		return nil, fmt.Errorf("missing KubeClient")
	}

	s, err := c.Config.GenericConfig.SkipComplete().New("kube-project-apiserver", genericapiserver.EmptyDelegate) // completion is done in Complete, no need for a second time
	if err != nil {
		return nil, err
	}

	m := &ProjectServer{
		GenericAPIServer: s,
	}

	informerFactory := kubeinformers.NewSharedInformerFactory(c.KubeClient, 10*time.Minute)
	subjectAccessEvaluator := rbacauthorizer.NewSubjectAccessEvaluator(
		&roleGetter{informerFactory.Rbac().InternalVersion().Roles().Lister()},
		&roleBindingLister{informerFactory.Rbac().InternalVersion().RoleBindings().Lister()},
		&clusterRoleGetter{informerFactory.Rbac().InternalVersion().ClusterRoles().Lister()},
		&clusterRoleBindingLister{informerFactory.Rbac().InternalVersion().ClusterRoleBindings().Lister()},
		"",
	)
	authCache := authcache.NewAuthorizationCache(
		authcache.NewReviewer(subjectAccessEvaluator),
		informerFactory.Core().InternalVersion().Namespaces(),
		informerFactory.Rbac().InternalVersion().ClusterRoles(),
		informerFactory.Rbac().InternalVersion().ClusterRoleBindings(),
		informerFactory.Rbac().InternalVersion().Roles(),
		informerFactory.Rbac().InternalVersion().RoleBindings(),
	)

	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(projectapi.GroupName, kapi.Registry, kapi.Scheme, kapi.ParameterCodec, kapi.Codecs)
	apiGroupInfo.GroupMeta.GroupVersion = projectapiv1.SchemeGroupVersion

	v1storage := map[string]rest.Storage{}
	v1storage["projectrequests"] = projectrequeststorage.NewREST("", c.Config.GenericConfig.Authorizer, c.KubeClient)
	v1storage["projects"] = projectstorage.NewREST(c.KubeClient.Core().Namespaces(), authCache, authCache, informerFactory.Core().InternalVersion().Namespaces().Lister())

	apiGroupInfo.VersionedResourcesStorageMap[projectapiv1.SchemeGroupVersion.Version] = v1storage

	if err := m.GenericAPIServer.InstallAPIGroup(&apiGroupInfo); err != nil {
		return nil, err
	}

	m.GenericAPIServer.AddPostStartHook("start-informers", func(context genericapiserver.PostStartHookContext) error {
		informerFactory.Start(context.StopCh)
		return nil
	})
	m.GenericAPIServer.AddPostStartHook("start-authorization-cache", func(context genericapiserver.PostStartHookContext) error {
		go authCache.Run(1 * time.Second)
		return nil
	})

	return m, nil
}
