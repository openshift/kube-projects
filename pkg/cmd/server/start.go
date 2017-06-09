package server

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	utilwait "k8s.io/apimachinery/pkg/util/wait"
	genericapiserver "k8s.io/apiserver/pkg/server"
	genericoptions "k8s.io/apiserver/pkg/server/options"
	"k8s.io/client-go/rest"
	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	kcmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"

	"github.com/openshift/kube-projects/pkg/apiserver"
	projectapiv1 "github.com/openshift/kube-projects/pkg/project/api/v1"
)

const defaultConfigDir = "openshift.local.config/project-server"

type ProjectServerOptions struct {
	RecommendedOptions *genericoptions.RecommendedOptions
}

const startLong = `Start an API server hosting the project.openshift.io API.`

// NewCommandStartMaster provides a CLI handler for 'start master' command
func NewCommandStartProjectServer(out io.Writer) *cobra.Command {
	o := &ProjectServerOptions{
		RecommendedOptions: genericoptions.NewRecommendedOptions("kube-projects.openshift.io", kapi.Scheme, kapi.Codecs.LegacyCodec(projectapiv1.SchemeGroupVersion)),
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

	o.RecommendedOptions.AddFlags(cmd.Flags())

	return cmd
}

func (o ProjectServerOptions) Validate(args []string) error {
	return nil
}

func (o *ProjectServerOptions) Complete() error {
	return nil
}

func (o ProjectServerOptions) RunProjectServer() error {
	// TODO have a "real" external address
	if err := o.RecommendedOptions.SecureServing.MaybeDefaultWithSelfSignedCerts("localhost", nil, nil); err != nil {
		return fmt.Errorf("error creating self-signed certificates: %v", err)
	}

	genericAPIServerConfig := genericapiserver.NewConfig(kapi.Codecs)
	if err := o.RecommendedOptions.ApplyTo(genericAPIServerConfig); err != nil {
		return err
	}

	// read the kubeconfig file to use for proxying requests
	kubeClientConfig, err := rest.InClusterConfig()
	if err != nil {
		return err
	}
	internalclientset, err := internalclientset.NewForConfig(kubeClientConfig)
	if err != nil {
		return err
	}

	config := apiserver.Config{
		GenericConfig: genericAPIServerConfig,
		KubeClient:    internalclientset,
	}

	server, err := config.Complete().New()
	if err != nil {
		return err
	}
	server.GenericAPIServer.PrepareRun().Run(utilwait.NeverStop)
	return nil
}
