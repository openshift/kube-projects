package projectrequest

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/endpoints/request"
	kapi "k8s.io/kubernetes/pkg/api"

	projectapi "github.com/openshift/kube-projects/pkg/apis/project"
	projectvalidation "github.com/openshift/kube-projects/pkg/apis/project/validation"
)

type strategy struct {
	runtime.ObjectTyper
}

var Strategy = strategy{kapi.Scheme}

func (strategy) PrepareForUpdate(ctx request.Context, obj, old runtime.Object) {}

func (strategy) NamespaceScoped() bool {
	return false
}

func (strategy) GenerateName(base string) string {
	return base
}

func (strategy) PrepareForCreate(ctx request.Context, obj runtime.Object) {
}

// Validate validates a new client
func (strategy) Validate(ctx request.Context, obj runtime.Object) field.ErrorList {
	projectrequest := obj.(*projectapi.ProjectRequest)
	return projectvalidation.ValidateProjectRequest(projectrequest)
}

// ValidateUpdate validates a client update
func (strategy) ValidateUpdate(ctx request.Context, obj runtime.Object, old runtime.Object) field.ErrorList {
	return nil
}

// Canonicalize normalizes the object after validation.
func (strategy) Canonicalize(obj runtime.Object) {
}

// AllowCreateOnUpdate is false for OAuth objects
func (strategy) AllowCreateOnUpdate() bool {
	return false
}
