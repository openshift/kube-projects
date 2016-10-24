package server

import (
	"fmt"
	"io"
	"net"
	"path"
	"time"

	"github.com/spf13/cobra"

	"k8s.io/kubernetes/pkg/auth/authenticator"
	"k8s.io/kubernetes/pkg/auth/authenticator/bearertoken"
	"k8s.io/kubernetes/pkg/auth/group"
	"k8s.io/kubernetes/pkg/auth/user"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	"k8s.io/kubernetes/pkg/client/unversioned/clientcmd"
	"k8s.io/kubernetes/pkg/genericapiserver"
	kcmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	certutil "k8s.io/kubernetes/pkg/util/cert"
	"k8s.io/kubernetes/plugin/pkg/auth/authenticator/request/anonymous"
	authenticationunion "k8s.io/kubernetes/plugin/pkg/auth/authenticator/request/union"
	"k8s.io/kubernetes/plugin/pkg/auth/authenticator/request/x509"
	authenticationwebhook "k8s.io/kubernetes/plugin/pkg/auth/authenticator/token/webhook"
	authorizationwebhook "k8s.io/kubernetes/plugin/pkg/auth/authorizer/webhook"

	"github.com/openshift/kube-projects/pkg/apiserver"
)

const defaultConfigDir = "openshift.local.config/project-server"

type ProjectServerOptions struct {
	StdOut io.Writer

	ConfigDir string

	// ConfigFile is the serialized config file used to launch this process.  It is optional
	ConfigFile string
	KubeConfig string
	ClientCA   string
}

const startLong = `Start an API server hosting the project.openshift.io API.`

// NewCommandStartMaster provides a CLI handler for 'start master' command
func NewCommandStartProjectServer(out io.Writer) *cobra.Command {
	o := &ProjectServerOptions{StdOut: out}

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
	flags.StringVar(&o.ConfigDir, "write-config", o.ConfigDir, "Directory to write an initial config into.  After writing, exit without starting the server.")
	flags.StringVar(&o.KubeConfig, "kubeconfig", o.KubeConfig, "Location of the master configuration file to run from. When running from a configuration file, all other command-line arguments are ignored.")
	flags.StringVar(&o.ClientCA, "client-ca-file", o.ClientCA, "If set, any request presenting a client certificate signed by one of the authorities in the client-ca-file is authenticated with an identity corresponding to the CommonName of the client certificate.")

	// autocompletion hints
	cmd.MarkFlagFilename("write-config")
	cmd.MarkFlagFilename("config", "yaml", "yml")

	GLog(cmd.PersistentFlags())

	return cmd
}

func (o ProjectServerOptions) Validate(args []string) error {
	return nil
}

func (o *ProjectServerOptions) Complete() error {
	return nil
}

// RunServer will eventually take the options and:
// 1.  Creates certs if needed
// 2.  Reads fully specified master config OR builds a fully specified master config from the args
// 3.  Writes the fully specified master config and exits if needed
// 4.  Starts the master based on the fully specified config
func (o ProjectServerOptions) RunProjectServer() error {
	secureServingInfo := genericapiserver.ServingInfo{
		BindAddress: net.JoinHostPort("0.0.0.0", "8444"),
		ServerCert: genericapiserver.CertInfo{
			Generate: true,
			CertFile: path.Join(defaultConfigDir, "apiserver.crt"),
			KeyFile:  path.Join(defaultConfigDir, "apiserver.key"),
		},
		ClientCA: o.ClientCA,
	}

	m := &ProjectServer{
		servingInfo: secureServingInfo,
		kubeConfig:  o.KubeConfig,
	}
	return m.Start()
}

// ProjectServer encapsulates starting the components of the master
type ProjectServer struct {
	// this should be part of the serializeable config
	servingInfo genericapiserver.ServingInfo
	kubeConfig  string
}

// Start launches a master. It will error if possible, but some background processes may still
// be running and the process should exit after it finishes.
func (s *ProjectServer) Start() error {
	genericAPIServerConfig := genericapiserver.NewConfig().Complete()
	genericAPIServerConfig.SecureServingInfo = &s.servingInfo
	if err := genericAPIServerConfig.MaybeGenerateServingCerts(); err != nil {
		return err
	}

	kubeClientConfig, err := clientcmd.
		NewNonInteractiveDeferredLoadingClientConfig(&clientcmd.ClientConfigLoadingRules{ExplicitPath: s.kubeConfig}, &clientcmd.ConfigOverrides{}).
		ClientConfig()
	if err != nil {
		return err
	}
	clientset, err := internalclientset.NewForConfig(kubeClientConfig)
	if err != nil {
		return err
	}
	genericAPIServerConfig.Authenticator, err = NewAuthenticator(s.servingInfo.ClientCA, clientset)
	if err != nil {
		return err
	}
	genericAPIServerConfig.Authorizer, err = authorizationwebhook.NewFromInterface(clientset.Authorization().SubjectAccessReviews(), 30*time.Second, 30*time.Second)
	if err != nil {
		return err
	}

	config := apiserver.Config{
		GenericConfig:        genericAPIServerConfig.Config,
		PrivilegedKubeClient: clientset,
	}

	server, err := config.Complete().New()
	if err != nil {
		return err
	}
	server.GenericAPIServer.Run()
	return nil
}

func NewAuthenticator(clientCAFile string, clientset internalclientset.Interface) (authenticator.Request, error) {
	certAuth, err := newAuthenticatorFromClientCAFile(clientCAFile)
	if err != nil {
		return nil, err
	}

	tokenChecker, err := authenticationwebhook.NewFromInterface(clientset.Authentication().TokenReviews(), 5*time.Minute)
	if err != nil {
		return nil, err
	}

	authenticator := authenticationunion.New(certAuth, bearertoken.New(tokenChecker))
	authenticator = group.NewGroupAdder(authenticator, []string{user.AllAuthenticated})

	// If the authenticator chain returns an error, return an error (don't consider a bad bearer token anonymous).
	authenticator = authenticationunion.NewFailOnError(authenticator, anonymous.NewAuthenticator())

	return authenticator, nil

}

func newAuthenticatorFromClientCAFile(clientCAFile string) (authenticator.Request, error) {
	roots, err := certutil.NewPool(clientCAFile)
	if err != nil {
		return nil, err
	}

	opts := x509.DefaultVerifyOptions()
	opts.Roots = roots

	return x509.New(opts, x509.CommonNameUserConversion), nil
}
