package apiserver

import (
	"fmt"
	"net/http"
	"os"
	"time"

	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/rest"
	apiserverfilters "k8s.io/kubernetes/pkg/apiserver/filters"
	authhandlers "k8s.io/kubernetes/pkg/auth/handlers"
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/clientset"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	kubeinformers "k8s.io/kubernetes/pkg/client/informers/informers_generated"
	"k8s.io/kubernetes/pkg/controller/informers"
	"k8s.io/kubernetes/pkg/genericapiserver"
	genericfilters "k8s.io/kubernetes/pkg/genericapiserver/filters"
	"k8s.io/kubernetes/pkg/util/wait"
	"k8s.io/kubernetes/pkg/version"
	rbacauthorizer "k8s.io/kubernetes/plugin/pkg/auth/authorizer/rbac"

	"github.com/openshift/kube-projects/pkg/apiserver/bootstraproutes"
	projectapi "github.com/openshift/kube-projects/pkg/project/api"
	projectapiv1 "github.com/openshift/kube-projects/pkg/project/api/v1"
	authcache "github.com/openshift/kube-projects/pkg/project/auth"
	projectstorage "github.com/openshift/kube-projects/pkg/project/registry/project"
	projectrequeststorage "github.com/openshift/kube-projects/pkg/project/registry/projectrequest"
)

type Config struct {
	GenericConfig *genericapiserver.Config

	PrivilegedKubeClient         internalclientset.Interface
	PrivilegedExternalKubeClient clientset.Interface

	AuthUser   string
	ServerUser string
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
	c.GenericConfig.Version = &version.Info{Major: "1"}

	c.GenericConfig.Complete()

	return completedConfig{c}
}

// SkipComplete provides a way to construct a server instance without config completion.
func (c *Config) SkipComplete() completedConfig {
	return completedConfig{c}
}

// New returns a new instance of ProjectServer from the given config.
func (c completedConfig) New() (*ProjectServer, error) {
	if c.PrivilegedKubeClient == nil {
		return nil, fmt.Errorf("missing PrivilegedKubeClient")
	}
	if c.PrivilegedExternalKubeClient == nil {
		return nil, fmt.Errorf("missing PrivilegedExternalKubeClient")
	}

	unprotectedMux := http.NewServeMux()
	c.Config.GenericConfig.BuildHandlerChainsFunc = (&handlerChainConfig{
		unprotectedMux: unprotectedMux,
	}).handlerChain

	s, err := c.Config.GenericConfig.SkipComplete().New() // completion is done in Complete, no need for a second time
	if err != nil {
		return nil, err
	}

	m := &ProjectServer{
		GenericAPIServer: s,
	}

	bootstraproutes.RBAC{AuthUser: c.AuthUser, ServerUser: c.ServerUser}.Install(unprotectedMux)
	bootstraproutes.APIFederation{
		Namespace:   "projects.openshift.io",
		ServiceName: "api",
	}.Install(unprotectedMux)

	kubeInformers := kubeinformers.NewSharedInformerFactory(c.PrivilegedKubeClient, c.PrivilegedExternalKubeClient, 10*time.Minute)

	informerFactory := informers.NewSharedInformerFactory(c.PrivilegedExternalKubeClient, c.PrivilegedKubeClient, 10*time.Minute)
	subjectAccessEvaluator := rbacauthorizer.NewSubjectAccessEvaluator(
		informerFactory.Roles().Lister(),
		informerFactory.RoleBindings().Lister(),
		informerFactory.ClusterRoles().Lister(),
		informerFactory.ClusterRoleBindings().Lister(),
		"",
	)
	authCache := authcache.NewAuthorizationCache(
		authcache.NewReviewer(subjectAccessEvaluator),
		kubeInformers.Core().InternalVersion().Namespaces(),
		informerFactory.ClusterRoles(), informerFactory.ClusterRoleBindings(), informerFactory.Roles(), informerFactory.RoleBindings(),
	)

	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(projectapi.GroupName)
	apiGroupInfo.GroupMeta.GroupVersion = projectapiv1.SchemeGroupVersion

	v1storage := map[string]rest.Storage{}
	v1storage["projectrequests"] = projectrequeststorage.NewREST("", c.Config.GenericConfig.Authorizer, c.PrivilegedKubeClient)
	v1storage["projects"] = projectstorage.NewREST(c.PrivilegedKubeClient.Core().Namespaces(), authCache, authCache, informerFactory.Namespaces().Lister())

	apiGroupInfo.VersionedResourcesStorageMap[projectapiv1.SchemeGroupVersion.Version] = v1storage

	if err := m.GenericAPIServer.InstallAPIGroup(&apiGroupInfo); err != nil {
		return nil, err
	}

	m.GenericAPIServer.AddPostStartHook("start-informers", func(context genericapiserver.PostStartHookContext) error {
		informerFactory.Start(wait.NeverStop)
		kubeInformers.Start(wait.NeverStop)
		return nil
	})
	m.GenericAPIServer.AddPostStartHook("start-authorization-cache", func(context genericapiserver.PostStartHookContext) error {
		go authCache.Run(1 * time.Second)
		return nil
	})

	return m, nil
}

type handlerChainConfig struct {
	unprotectedMux *http.ServeMux
}

func (h *handlerChainConfig) handlerChain(apiHandler http.Handler, c *genericapiserver.Config) (secure, insecure http.Handler) {
	handler := apiserverfilters.WithAuthorization(apiHandler, c.RequestContextMapper, c.Authorizer)

	// this mux is NOT protected by authorization, but DOES have authentication information
	// this is so that everyone can hit these endpoints, but we have the user information for proxy cases
	handler = WithUnprotectedMux(handler, h.unprotectedMux)

	handler = apiserverfilters.WithImpersonation(handler, c.RequestContextMapper, c.Authorizer)
	handler = apiserverfilters.WithAudit(handler, c.RequestContextMapper, os.Stdout)
	handler = authhandlers.WithAuthentication(handler, c.RequestContextMapper, c.Authenticator, authhandlers.Unauthorized(c.SupportsBasicAuth))

	handler = genericfilters.WithCORS(handler, c.CorsAllowedOriginList, nil, nil, nil, "true")
	handler = genericfilters.WithPanicRecovery(handler, c.RequestContextMapper)
	handler = genericfilters.WithTimeoutForNonLongRunningRequests(handler, c.RequestContextMapper, c.LongRunningFunc)
	handler = genericfilters.WithMaxInFlightLimit(handler, c.MaxRequestsInFlight, c.MaxMutatingRequestsInFlight, c.RequestContextMapper, c.LongRunningFunc)
	handler = apiserverfilters.WithRequestInfo(handler, genericapiserver.NewRequestInfoResolver(c), c.RequestContextMapper)
	handler = kapi.WithRequestContext(handler, c.RequestContextMapper)

	return handler, nil
}

func WithUnprotectedMux(handler http.Handler, mux *http.ServeMux) http.Handler {
	if mux == nil {
		return handler
	}

	// register the handler at this stage against everything under slash.  More specific paths that get registered will take precedence
	mux.Handle("/", handler)

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		mux.ServeHTTP(w, req)
	})
}
