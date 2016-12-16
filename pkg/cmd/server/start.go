package server

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	"k8s.io/kubernetes/pkg/client/unversioned/clientcmd"
	"k8s.io/kubernetes/pkg/genericapiserver"
	"k8s.io/kubernetes/pkg/genericapiserver/filters"
	genericoptions "k8s.io/kubernetes/pkg/genericapiserver/options"
	kcmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/kubernetes/pkg/util/sets"
	utilwait "k8s.io/kubernetes/pkg/util/wait"

	"github.com/openshift/kube-projects/pkg/apiserver"
)

const defaultConfigDir = "openshift.local.config/project-server"

type ProjectServerOptions struct {
	SecureServing  *genericoptions.SecureServingOptions
	Authentication *genericoptions.DelegatingAuthenticationOptions
	Authorization  *genericoptions.DelegatingAuthorizationOptions

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
	if err := o.SecureServing.MaybeDefaultWithSelfSignedCerts("localhost"); err != nil {
		return fmt.Errorf("error creating self-signed certificates: %v", err)
	}

	genericAPIServerConfig := genericapiserver.NewConfig()
	if _, err := genericAPIServerConfig.ApplySecureServingOptions(o.SecureServing); err != nil {
		return err
	}
	if _, err := genericAPIServerConfig.ApplyDelegatingAuthenticationOptions(o.Authentication); err != nil {
		return err
	}
	if _, err := genericAPIServerConfig.ApplyDelegatingAuthorizationOptions(o.Authorization); err != nil {
		return err
	}
	genericAPIServerConfig.LongRunningFunc = filters.BasicLongRunningRequestCheck(
		sets.NewString("watch", "proxy"),
		sets.NewString(),
	)

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
