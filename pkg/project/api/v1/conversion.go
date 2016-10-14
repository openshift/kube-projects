package v1

import (
	"fmt"

	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/registry/core/namespace"
	"k8s.io/kubernetes/pkg/runtime"
)

// LIFTED FROM OPENSHIFT

// GetFieldLabelConversionFunc returns a field label conversion func, which does the following:
// * returns overrideLabels[label], value, nil if the specified label exists in the overrideLabels map
// * returns label, value, nil if the specified label exists as a key in the supportedLabels map (values in this map are unused, it is intended to be a prototypical label/value map)
// * otherwise, returns an error
func GetFieldLabelConversionFunc(supportedLabels map[string]string, overrideLabels map[string]string) func(label, value string) (string, string, error) {
	return func(label, value string) (string, string, error) {
		if label, overridden := overrideLabels[label]; overridden {
			return label, value, nil
		}
		if _, supported := supportedLabels[label]; supported {
			return label, value, nil
		}
		return "", "", fmt.Errorf("field label not supported: %s", label)
	}
}

func addConversionFuncs(scheme *runtime.Scheme) error {
	return scheme.AddFieldLabelConversionFunc("v1", "Project",
		GetFieldLabelConversionFunc(namespace.NamespaceToSelectableFields(&kapi.Namespace{}), nil),
	)
}
