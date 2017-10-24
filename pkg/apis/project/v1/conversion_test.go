package v1_test

import (
	"testing"

	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/registry/core/namespace"

	// install all APIs
	_ "github.com/openshift/kube-projects/pkg/apis/project/install"
)

func CheckFieldLabelConversions(t *testing.T, version, kind string, expectedLabels map[string]string, customLabels ...string) {
	for label := range expectedLabels {
		_, _, err := kapi.Scheme.ConvertFieldLabel(version, kind, label, "")
		if err != nil {
			t.Errorf("No conversion registered for %s for %s %s", label, version, kind)
		}
	}
	for _, label := range customLabels {
		_, _, err := kapi.Scheme.ConvertFieldLabel(version, kind, label, "")
		if err != nil {
			t.Errorf("No conversion registered for %s for %s %s", label, version, kind)
		}
	}
}

func TestFieldSelectorConversions(t *testing.T) {
	CheckFieldLabelConversions(t, "v1", "Project",
		// Ensure all currently returned labels are supported
		namespace.NamespaceToSelectableFields(&kapi.Namespace{}),
		// Ensure previously supported labels have conversions. DO NOT REMOVE THINGS FROM THIS LIST
		"status.phase",
	)
}
