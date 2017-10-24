package util

import (
	"k8s.io/api/core/v1"

	projectapi "github.com/openshift/kube-projects/pkg/apis/project"
)

// ConvertNamespace transforms a Namespace into a Project
func ConvertNamespace(namespace *v1.Namespace) *projectapi.Project {
	return &projectapi.Project{
		ObjectMeta: namespace.ObjectMeta,
		Spec: projectapi.ProjectSpec{
			Finalizers: namespace.Spec.Finalizers,
		},
		Status: projectapi.ProjectStatus{
			Phase: namespace.Status.Phase,
		},
	}
}

// convertProject transforms a Project into a Namespace
func ConvertProject(project *projectapi.Project) *v1.Namespace {
	namespace := &v1.Namespace{
		ObjectMeta: project.ObjectMeta,
		Spec: v1.NamespaceSpec{
			Finalizers: project.Spec.Finalizers,
		},
		Status: v1.NamespaceStatus{
			Phase: project.Status.Phase,
		},
	}
	if namespace.Annotations == nil {
		namespace.Annotations = map[string]string{}
	}
	namespace.Annotations[projectapi.ProjectDisplayName] = project.Annotations[projectapi.ProjectDisplayName]
	return namespace
}

// ConvertNamespaceList transforms a NamespaceList into a ProjectList
func ConvertNamespaceList(namespaceList *v1.NamespaceList) *projectapi.ProjectList {
	projects := &projectapi.ProjectList{}
	for _, n := range namespaceList.Items {
		projects.Items = append(projects.Items, *ConvertNamespace(&n))
	}
	return projects
}
