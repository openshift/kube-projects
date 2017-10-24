package apiserver

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/client-go/kubernetes"

	projectapi "github.com/openshift/kube-projects/pkg/apis/project"
	projectapiv1 "github.com/openshift/kube-projects/pkg/apis/project/v1"
	"github.com/openshift/kube-projects/pkg/apiserver/rbac"
	authcache "github.com/openshift/kube-projects/pkg/project/auth"
	projectstorage "github.com/openshift/kube-projects/pkg/project/registry/project"
	projectrequeststorage "github.com/openshift/kube-projects/pkg/project/registry/projectrequest"
)

type Config struct {
	GenericConfig *genericapiserver.RecommendedConfig

	ExtraConfig ExtraConfig
}

type ExtraConfig struct {
}

type completedConfig struct {
	GenericConfig genericapiserver.CompletedConfig
	ExtraConfig   *ExtraConfig
}

type CompletedConfig struct {
	// Embed a private pointer that cannot be instantiated outside of this package.
	*completedConfig
}

// ProjectServer contains state for a Kubernetes cluster master/api server.
type ProjectServer struct {
	GenericAPIServer *genericapiserver.GenericAPIServer
}

// Complete fills in any fields not set that are required to have valid data. It's mutating the receiver.
func (c *Config) Complete() completedConfig {
	c.GenericConfig.Version = nil

	return completedConfig{
		GenericConfig: c.GenericConfig.Complete(),
		ExtraConfig:   &c.ExtraConfig,
	}
}

// New returns a new instance of ProjectServer from the given config.
func (c completedConfig) New() (*ProjectServer, error) {
	kubeClient, err := kubernetes.NewForConfig(c.GenericConfig.LoopbackClientConfig)
	if err != nil {
		return nil, err
	}
	s, err := c.GenericConfig.New("kube-project-apiserver", genericapiserver.EmptyDelegate) // completion is done in Complete, no need for a second time
	if err != nil {
		return nil, err
	}

	m := &ProjectServer{
		GenericAPIServer: s,
	}

	informerFactory := c.GenericConfig.SharedInformerFactory
	subjectAccessEvaluator := rbac.NewSubjectAccessEvaluator(
		&roleGetter{informerFactory.Rbac().V1().Roles().Lister()},
		&roleBindingLister{informerFactory.Rbac().V1().RoleBindings().Lister()},
		&clusterRoleGetter{informerFactory.Rbac().V1().ClusterRoles().Lister()},
		&clusterRoleBindingLister{informerFactory.Rbac().V1().ClusterRoleBindings().Lister()},
		"",
	)
	authCache := authcache.NewAuthorizationCache(
		authcache.NewReviewer(subjectAccessEvaluator),
		informerFactory.Core().V1().Namespaces(),
		informerFactory.Rbac().V1().ClusterRoles(),
		informerFactory.Rbac().V1().ClusterRoleBindings(),
		informerFactory.Rbac().V1().Roles(),
		informerFactory.Rbac().V1().RoleBindings(),
	)

	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(projectapi.GroupName, projectapi.Registry, projectapi.Scheme, metav1.ParameterCodec, projectapi.Codecs)
	apiGroupInfo.GroupMeta.GroupVersion = projectapiv1.SchemeGroupVersion

	v1storage := map[string]rest.Storage{}
	v1storage["projectrequests"] = projectrequeststorage.NewREST("", c.GenericConfig.Authorizer, kubeClient)
	v1storage["projects"] = projectstorage.NewREST(kubeClient.Core().Namespaces(), authCache, authCache, informerFactory.Core().V1().Namespaces().Lister())

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
