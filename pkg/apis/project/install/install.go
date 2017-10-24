package install

import (
	"k8s.io/apimachinery/pkg/apimachinery/announced"
	"k8s.io/apimachinery/pkg/apimachinery/registered"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/openshift/kube-projects/pkg/apis/project"
	"github.com/openshift/kube-projects/pkg/apis/project/v1"
)

func init() {
	Install(project.GroupFactoryRegistry, project.Registry, project.Scheme)
}

func Install(groupFactoryRegistry announced.APIGroupFactoryRegistry, registry *registered.APIRegistrationManager, scheme *runtime.Scheme) {
	if err := announced.NewGroupMetaFactory(
		&announced.GroupMetaFactoryArgs{
			GroupName:                  project.GroupName,
			RootScopedKinds:            sets.NewString("ProjectRequest", "Project"),
			VersionPreferenceOrder:     []string{v1.SchemeGroupVersion.Version},
			AddInternalObjectsToScheme: project.AddToScheme,
		},
		announced.VersionToSchemeFunc{
			v1.SchemeGroupVersion.Version: v1.AddToScheme,
		},
	).Announce(groupFactoryRegistry).RegisterAndEnable(registry, scheme); err != nil {
		panic(err)
	}
}
