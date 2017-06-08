package main

import (
	"os"
	"runtime"

	"k8s.io/apiserver/pkg/util/logs"

	"github.com/openshift/kube-projects/pkg/cmd/server"

	// install all APIs
	_ "github.com/openshift/kube-projects/pkg/project/api/install"
	_ "github.com/openshift/kube-projects/pkg/project/auth"
	_ "k8s.io/kubernetes/pkg/api/install"
)

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()
	// defer serviceability.BehaviorOnPanic(os.Getenv("OPENSHIFT_ON_PANIC"))()
	// defer serviceability.Profile(os.Getenv("OPENSHIFT_PROFILE")).Stop()

	if len(os.Getenv("GOMAXPROCS")) == 0 {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}

	cmd := server.NewCommandStartProjectServer(os.Stdout)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
