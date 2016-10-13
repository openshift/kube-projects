package main

import (
	"fmt"
	"os"
	"runtime"

	"k8s.io/kubernetes/pkg/util/logs"

	// "github.com/openshift/origin/pkg/cmd/util/serviceability"

	// install all APIs
	_ "github.com/openshift/kube-projects/pkg/apis/project/install"
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

	fmt.Printf("Starting")
}
