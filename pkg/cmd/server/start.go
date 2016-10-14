package server

import (
	"fmt"
	"io"
	"net"
	"path"

	"github.com/golang/glog"
	"github.com/spf13/cobra"

	"k8s.io/kubernetes/pkg/genericapiserver"
	kcmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"

	"github.com/openshift/kube-projects/pkg/apiserver"
)

const defaultConfigDir = "openshift.local.config/project-server"

type ProjectServerOptions struct {
	StdOut io.Writer

	ConfigDir string

	// ConfigFile is the serialized config file used to launch this process.  It is optional
	ConfigFile string
	KubeConfig string
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

	// autocompletion hints
	cmd.MarkFlagFilename("write-config")
	cmd.MarkFlagFilename("config", "yaml", "yml")

	return cmd
}

func (o ProjectServerOptions) Validate(args []string) error {
	return nil
}

func (o *ProjectServerOptions) Complete() error {
	return nil
}

// StartMaster calls RunMaster and then waits forever
func (o ProjectServerOptions) RunProjectServer() error {
	if err := o.RunServer(); err != nil {
		return err
	}

	if o.IsWriteConfigOnly() {
		return nil
	}

	return nil
}

// RunServer will eventually take the options and:
// 1.  Creates certs if needed
// 2.  Reads fully specified master config OR builds a fully specified master config from the args
// 3.  Writes the fully specified master config and exits if needed
// 4.  Starts the master based on the fully specified config
func (o ProjectServerOptions) RunServer() error {
	startUsingConfigFile := !o.IsWriteConfigOnly() && o.IsRunFromConfig()

	if !startUsingConfigFile {
		glog.V(2).Infof("Generating master configuration")
		if err := o.CreateCerts(); err != nil {
			return err
		}
	}

	secureServingInfo := genericapiserver.ServingInfo{
		BindAddress: net.JoinHostPort("0.0.0.0", "8444"),
		ServerCert: genericapiserver.CertInfo{
			Generate: true,
			CertFile: path.Join(defaultConfigDir, "apiserver.crt"),
			KeyFile:  path.Join(defaultConfigDir, "apiserver.key"),
		},
		ClientCA: "",
	}

	// var masterConfig *configapi.MasterConfig
	// var err error
	// if startUsingConfigFile {
	// 	masterConfig, err = configapilatest.ReadAndResolveMasterConfig(o.ConfigFile)
	// } else {
	// 	masterConfig, err = o.MasterArgs.BuildSerializeableMasterConfig()
	// }
	// if err != nil {
	// 	return err
	// }

	// if o.IsWriteConfigOnly() {
	// 	// Resolve relative to CWD
	// 	cwd, err := os.Getwd()
	// 	if err != nil {
	// 		return err
	// 	}
	// 	if err := configapi.ResolveMasterConfigPaths(masterConfig, cwd); err != nil {
	// 		return err
	// 	}

	// 	// Relativize to config file dir
	// 	base, err := cmdutil.MakeAbs(filepath.Dir(o.MasterArgs.GetConfigFileToWrite()), cwd)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	if err := configapi.RelativizeMasterConfigPaths(masterConfig, base); err != nil {
	// 		return err
	// 	}

	// 	content, err := configapilatest.WriteYAML(masterConfig)
	// 	if err != nil {

	// 		return err
	// 	}

	// 	if err := os.MkdirAll(path.Dir(o.MasterArgs.GetConfigFileToWrite()), os.FileMode(0755)); err != nil {
	// 		return err
	// 	}
	// 	if err := ioutil.WriteFile(o.MasterArgs.GetConfigFileToWrite(), content, 0644); err != nil {
	// 		return err
	// 	}

	// 	fmt.Fprintf(o.Output, "Wrote master config to: %s\n", o.MasterArgs.GetConfigFileToWrite())

	// 	return nil
	// }

	m := &ProjectServer{
		servingInfo: secureServingInfo,
	}
	return m.Start()
}

func (o ProjectServerOptions) CreateCerts() error {
	// masterAddr, err := o.MasterArgs.GetMasterAddress()
	// if err != nil {
	// 	return err
	// }
	// publicMasterAddr, err := o.MasterArgs.GetMasterPublicAddress()
	// if err != nil {
	// 	return err
	// }

	// signerName := admin.DefaultSignerName()
	// hostnames, err := o.MasterArgs.GetServerCertHostnames()
	// if err != nil {
	// 	return err
	// }
	// mintAllCertsOptions := admin.CreateMasterCertsOptions{
	// 	CertDir:            o.MasterArgs.ConfigDir.Value(),
	// 	SignerName:         signerName,
	// 	Hostnames:          hostnames.List(),
	// 	APIServerURL:       masterAddr.String(),
	// 	APIServerCAFiles:   o.MasterArgs.APIServerCAFiles,
	// 	CABundleFile:       admin.DefaultCABundleFile(o.MasterArgs.ConfigDir.Value()),
	// 	PublicAPIServerURL: publicMasterAddr.String(),
	// 	Output:             cmdutil.NewGLogWriterV(3),
	// }
	// if err := mintAllCertsOptions.Validate(nil); err != nil {
	// 	return err
	// }
	// if err := mintAllCertsOptions.CreateMasterCerts(); err != nil {
	// 	return err
	// }

	return nil
}

func (o ProjectServerOptions) IsWriteConfigOnly() bool {
	return len(o.ConfigDir) > 0
}

func (o ProjectServerOptions) IsRunFromConfig() bool {
	return (len(o.ConfigFile) > 0)
}

// ProjectServer encapsulates starting the components of the master
type ProjectServer struct {
	// this should be part of the serializeable config
	servingInfo genericapiserver.ServingInfo
}

// Start launches a master. It will error if possible, but some background processes may still
// be running and the process should exit after it finishes.
func (m *ProjectServer) Start() error {
	genericAPIServerConfig := genericapiserver.NewConfig().Complete()
	genericAPIServerConfig.SecureServingInfo = &m.servingInfo
	if err := genericAPIServerConfig.MaybeGenerateServingCerts(); err != nil {
		return err
	}

	config := apiserver.Config{
		GenericConfig: genericAPIServerConfig.Config,
	}

	server, err := config.Complete().New()
	if err != nil {
		return err
	}
	server.GenericAPIServer.Run()
	return nil
}
