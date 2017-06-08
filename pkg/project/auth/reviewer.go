package auth

import (
	"k8s.io/apiserver/pkg/authorization/authorizer"
	kapi "k8s.io/client-go/pkg/api"
	"k8s.io/kubernetes/pkg/apis/rbac"
	rbacauthorizer "k8s.io/kubernetes/plugin/pkg/auth/authorizer/rbac"
)

// Reviewer performs access reviews for a project by name
type Reviewer interface {
	Review(name string) ([]rbac.Subject, error)
}

// reviewer performs access reviews for a project by name
type reviewer struct {
	subjectAccessEvaluator *rbacauthorizer.SubjectAccessEvaluator
}

// NewReviewer knows how to make access control reviews for a resource by name
func NewReviewer(subjectAccessEvaluator *rbacauthorizer.SubjectAccessEvaluator) Reviewer {
	return &reviewer{
		subjectAccessEvaluator: subjectAccessEvaluator,
	}
}

// Review performs a resource access review for the given resource by name
func (r *reviewer) Review(name string) ([]rbac.Subject, error) {
	action := authorizer.AttributesRecord{
		Verb:            "get",
		Namespace:       name,
		APIGroup:        kapi.GroupName,
		Resource:        "namespaces",
		Name:            name,
		ResourceRequest: true,
	}
	return r.subjectAccessEvaluator.AllowedSubjects(action)
}
