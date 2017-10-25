package project

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/storage/names"

	projectapi "github.com/openshift/kube-projects/pkg/apis/project"
	projectapiv1 "github.com/openshift/kube-projects/pkg/apis/project/v1"
	"github.com/openshift/kube-projects/pkg/apis/project/validation"
)

// projectStrategy implements behavior for projects
type projectStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

// Strategy is the default logic that applies when creating and updating Project
// objects via the REST API.
var Strategy = projectStrategy{projectapi.Scheme, names.SimpleNameGenerator}

// NamespaceScoped is false for projects.
func (projectStrategy) NamespaceScoped() bool {
	return false
}

// PrepareForCreate clears fields that are not allowed to be set by end users on creation.
func (projectStrategy) PrepareForCreate(ctx request.Context, obj runtime.Object) {
}

// PrepareForUpdate clears fields that are not allowed to be set by end users on update.
func (projectStrategy) PrepareForUpdate(ctx request.Context, obj, old runtime.Object) {
	newProject := obj.(*projectapiv1.Project)
	oldProject := old.(*projectapiv1.Project)
	newProject.Spec.Finalizers = oldProject.Spec.Finalizers
	newProject.Status = oldProject.Status
}

// Validate validates a new project.
func (projectStrategy) Validate(ctx request.Context, obj runtime.Object) field.ErrorList {
	return validation.ValidateProject(obj.(*projectapi.Project))
}

// AllowCreateOnUpdate is false for project.
func (projectStrategy) AllowCreateOnUpdate() bool {
	return false
}

func (projectStrategy) AllowUnconditionalUpdate() bool {
	return false
}

// Canonicalize normalizes the object after validation.
func (projectStrategy) Canonicalize(obj runtime.Object) {
}

// ValidateUpdate is the default update validation for an end user.
func (projectStrategy) ValidateUpdate(ctx request.Context, obj, old runtime.Object) field.ErrorList {
	return validation.ValidateProjectUpdate(obj.(*projectapi.Project), old.(*projectapi.Project))
}
