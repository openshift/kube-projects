package install

import (
	"github.com/openshift/kube-projects/pkg/project/api"
	"github.com/openshift/kube-projects/pkg/project/api/v1"
	"k8s.io/kubernetes/pkg/apimachinery/announced"
)

func init() {
	if err := announced.NewGroupMetaFactory(
		&announced.GroupMetaFactoryArgs{
			GroupName:                  api.GroupName,
			VersionPreferenceOrder:     []string{v1.SchemeGroupVersion.Version},
			ImportPrefix:               "github.com/openshift/kube-projects/pkg/project/api",
			AddInternalObjectsToScheme: api.AddToScheme,
		},
		announced.VersionToSchemeFunc{
			v1.SchemeGroupVersion.Version: v1.AddToScheme,
		},
	).Announce().RegisterAndEnable(); err != nil {
		panic(err)
	}
}
