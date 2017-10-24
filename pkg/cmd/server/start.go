package server

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	utilwait "k8s.io/apimachinery/pkg/util/wait"
	genericapiserver "k8s.io/apiserver/pkg/server"
	genericoptions "k8s.io/apiserver/pkg/server/options"

	"github.com/golang/glog"
	"github.com/openshift/kube-projects/pkg/apis/project"
	projectapiv1 "github.com/openshift/kube-projects/pkg/apis/project/v1"
	"github.com/openshift/kube-projects/pkg/apiserver"
)

const defaultConfigDir = "openshift.local.config/project-server"

type ProjectServerOptions struct {
	RecommendedOptions *genericoptions.RecommendedOptions
}

const startLong = `Start an API server hosting the project.openshift.io API.`

// NewCommandStartMaster provides a CLI handler for 'start master' command
func NewCommandStartProjectServer(out io.Writer) *cobra.Command {
	o := &ProjectServerOptions{
		RecommendedOptions: genericoptions.NewRecommendedOptions("kube-projects.openshift.io", project.Codecs.LegacyCodec(projectapiv1.SchemeGroupVersion)),
	}

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Launch a project.openshift.io server",
		Long:  startLong,
		Run: func(c *cobra.Command, args []string) {
			fmt.Printf("Starting\n")

			if err := o.Complete(); err != nil {
				glog.Fatal(err)
			}
			if err := o.Validate(args); err != nil {
				glog.Fatal(err)
			}
			if err := o.RunProjectServer(); err != nil {
				glog.Fatal(err)
			}
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

	genericAPIServerConfig := genericapiserver.NewRecommendedConfig(project.Codecs)
	if err := o.RecommendedOptions.ApplyTo(genericAPIServerConfig); err != nil {
		return err
	}

	config := apiserver.Config{
		GenericConfig: genericAPIServerConfig,
	}

	server, err := config.Complete().New()
	if err != nil {
		return err
	}
	server.GenericAPIServer.PrepareRun().Run(utilwait.NeverStop)
	return nil
}
