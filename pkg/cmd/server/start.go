package server

import (
	"fmt"
	"io"

	"github.com/pborman/uuid"
	"github.com/spf13/cobra"

	"k8s.io/kubernetes/pkg/apiserver/authenticator"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	"k8s.io/kubernetes/pkg/client/unversioned/clientcmd"
	"k8s.io/kubernetes/pkg/genericapiserver"
	genericoptions "k8s.io/kubernetes/pkg/genericapiserver/options"
	kcmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	utilwait "k8s.io/kubernetes/pkg/util/wait"
	unionauth "k8s.io/kubernetes/plugin/pkg/auth/authenticator/request/union"

	"github.com/openshift/kube-projects/pkg/apiserver"
)

const defaultConfigDir = "openshift.local.config/project-server"

type ProjectServerOptions struct {
	SecureServing  *genericoptions.SecureServingOptions
	Authentication *genericoptions.DelegatingAuthenticationOptions
	Authorization  *genericoptions.DelegatingAuthorizationOptions
	AuthProxy      *genericoptions.RequestHeaderAuthenticationOptions

	AuthUser string

	ServerUser string
	KubeConfig string
}

const startLong = `Start an API server hosting the project.openshift.io API.`

// NewCommandStartMaster provides a CLI handler for 'start master' command
func NewCommandStartProjectServer(out io.Writer) *cobra.Command {
	o := &ProjectServerOptions{
		SecureServing:  genericoptions.NewSecureServingOptions(),
		Authentication: genericoptions.NewDelegatingAuthenticationOptions(),
		Authorization:  genericoptions.NewDelegatingAuthorizationOptions(),
		AuthProxy:      &genericoptions.RequestHeaderAuthenticationOptions{},
	}

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Launch a project.openshift.io server",
		Long:  startLong,
		Run: func(c *cobra.Command, args []string) {
			fmt.Printf("Starting\n")

			kcmdutil.CheckErr(o.Complete())
			kcmdutil.CheckErr(o.Validate(args))
			kcmdutil.CheckErr(o.RunProjectServer())
		},
	}

	flags := cmd.Flags()
	o.SecureServing.AddFlags(flags)
	o.Authentication.AddFlags(flags)
	o.Authorization.AddFlags(flags)
	o.AuthProxy.AddFlags(flags)
	flags.StringVar(&o.AuthUser, "auth-user", o.AuthUser, "username of the user used for delegating authentication and authorization.  Primes /bootstrap/rbac endpoint.")
	flags.StringVar(&o.ServerUser, "server-user", o.ServerUser, "username of the user used for accessing resources for this API server.  Primes /bootstrap/rbac endpoint.")
	flags.StringVar(&o.KubeConfig, "kubeconfig", o.KubeConfig, "kubeconfig file for access resources for this API server.")

	GLog(cmd.PersistentFlags())

	return cmd
}

func (o ProjectServerOptions) Validate(args []string) error {
	return nil
}

func (o *ProjectServerOptions) Complete() error {
	return nil
}

func (o ProjectServerOptions) RunProjectServer() error {
	var err error
	genericAPIServerConfig := genericapiserver.NewConfig().ApplySecureServingOptions(o.SecureServing)
	// TODO remove this, it should be applied some other way
	genericAPIServerConfig.PublicAddress, _ = o.SecureServing.ServingOptions.DefaultExternalAddress()

	if err := genericAPIServerConfig.MaybeGenerateServingCerts(); err != nil {
		return err
	}

	privilegedLoopbackToken := uuid.NewRandom().String()
	if genericAPIServerConfig.LoopbackClientConfig, err = genericoptions.NewSelfClientConfig(o.SecureServing, nil, privilegedLoopbackToken); err != nil {
		return err
	}

	authenticatorConfig, err := o.Authentication.ToAuthenticationConfig(o.SecureServing.ClientCA)
	if err != nil {
		return err
	}
	if genericAPIServerConfig.Authenticator, _, err = authenticatorConfig.New(); err != nil {
		return err
	}
	// TODO make this a lot easier
	proxyConfig := o.AuthProxy.ToAuthenticationRequestHeaderConfig()
	proxyAuthenticator, _, err := authenticator.New(authenticator.AuthenticatorConfig{RequestHeaderConfig: proxyConfig})
	if err != nil {
		return err
	}
	genericAPIServerConfig.Authenticator = unionauth.New(proxyAuthenticator, genericAPIServerConfig.Authenticator)

	authorizerConfig, err := o.Authorization.ToAuthorizationConfig()
	if err != nil {
		return err
	}
	if genericAPIServerConfig.Authorizer, err = authorizerConfig.New(); err != nil {
		return err
	}

	// read the kubeconfig file to use for proxying requests
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.ExplicitPath = o.KubeConfig
	loader := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{})
	kubeClientConfig, err := loader.ClientConfig()
	if err != nil {
		return err
	}
	clientset, err := internalclientset.NewForConfig(kubeClientConfig)
	if err != nil {
		return err
	}

	config := apiserver.Config{
		GenericConfig:        genericAPIServerConfig,
		PrivilegedKubeClient: clientset,
		AuthUser:             o.AuthUser,
		ServerUser:           o.ServerUser,
	}

	server, err := config.Complete().New()
	if err != nil {
		return err
	}
	server.GenericAPIServer.PrepareRun().Run(utilwait.NeverStop)
	return nil
}
