package validation

import (
	"reflect"
	"strings"

	"k8s.io/apimachinery/pkg/api/validation"
	apimachineryvalidation "k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/api/validation/path"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/openshift/kube-projects/pkg/apis/project"
	projectapi "github.com/openshift/kube-projects/pkg/apis/project"
)

func ValidateProjectName(name string, prefix bool) []string {
	if reasons := path.ValidatePathSegmentName(name, prefix); len(reasons) != 0 {
		return reasons
	}

	if len(name) < 2 {
		return []string{"must be at least 2 characters long"}
	}

	if reasons := apimachineryvalidation.ValidateNamespaceName(name, false); len(reasons) != 0 {
		return reasons
	}
	return nil
}

// ValidateProject tests required fields for a Project.
// This should only be called when creating a project (not on update),
// since its name validation is more restrictive than default namespace name validation
func ValidateProject(project *project.Project) field.ErrorList {
	result := validation.ValidateObjectMeta(&project.ObjectMeta, false, ValidateProjectName, field.NewPath("metadata"))

	if !validateNoNewLineOrTab(project.Annotations[projectapi.ProjectDisplayName]) {
		result = append(result, field.Invalid(field.NewPath("metadata", "annotations").Key(projectapi.ProjectDisplayName),
			project.Annotations[projectapi.ProjectDisplayName], "may not contain a new line or tab"))
	}

	return result
}

// validateNoNewLineOrTab ensures a string has no new-line or tab
func validateNoNewLineOrTab(s string) bool {
	return !(strings.Contains(s, "\n") || strings.Contains(s, "\t"))
}

// ValidateProjectUpdate tests to make sure a project update can be applied.  Modifies newProject with immutable fields.
func ValidateProjectUpdate(newProject *project.Project, oldProject *project.Project) field.ErrorList {
	allErrs := validation.ValidateObjectMetaUpdate(&newProject.ObjectMeta, &oldProject.ObjectMeta, field.NewPath("metadata"))
	allErrs = append(allErrs, ValidateProject(newProject)...)

	if !reflect.DeepEqual(newProject.Spec.Finalizers, oldProject.Spec.Finalizers) {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "finalizers"), oldProject.Spec.Finalizers, "field is immutable"))
	}
	if !reflect.DeepEqual(newProject.Status, oldProject.Status) {
		allErrs = append(allErrs, field.Invalid(field.NewPath("status"), oldProject.Spec.Finalizers, "field is immutable"))
	}

	// TODO this restriction exists because our authorizer/admission cannot properly express and restrict mutation on the field level.
	for name, value := range newProject.Annotations {
		if name == projectapi.ProjectDisplayName || name == projectapi.ProjectDescription {
			continue
		}

		if value != oldProject.Annotations[name] {
			allErrs = append(allErrs, field.Invalid(field.NewPath("metadata", "annotations").Key(name), value, "field is immutable, try updating the namespace"))
		}
	}
	// check for deletions
	for name, value := range oldProject.Annotations {
		if name == projectapi.ProjectDisplayName || name == projectapi.ProjectDescription {
			continue
		}
		if _, inNew := newProject.Annotations[name]; !inNew {
			allErrs = append(allErrs, field.Invalid(field.NewPath("metadata", "annotations").Key(name), value, "field is immutable, try updating the namespace"))
		}
	}

	for name, value := range newProject.Labels {
		if value != oldProject.Labels[name] {
			allErrs = append(allErrs, field.Invalid(field.NewPath("metadata", "labels").Key(name), value, "field is immutable, try updating the namespace"))
		}
	}
	for name, value := range oldProject.Labels {
		if _, inNew := newProject.Labels[name]; !inNew {
			allErrs = append(allErrs, field.Invalid(field.NewPath("metadata", "labels").Key(name), value, "field is immutable, try updating the namespace"))
		}
	}

	return allErrs
}

func ValidateProjectRequest(request *project.ProjectRequest) field.ErrorList {
	project := &project.Project{}
	project.ObjectMeta = request.ObjectMeta

	return ValidateProject(project)
}
