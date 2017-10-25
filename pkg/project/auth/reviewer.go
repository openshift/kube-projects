package auth

import (
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apiserver/pkg/authorization/authorizer"

	subjectlocator "github.com/openshift/kube-projects/pkg/apiserver/rbac"
)

// Reviewer performs access reviews for a project by name
type Reviewer interface {
	Review(name string) ([]rbacv1.Subject, error)
}

// reviewer performs access reviews for a project by name
type reviewer struct {
	subjectAccessEvaluator *subjectlocator.SubjectAccessEvaluator
}

// NewReviewer knows how to make access control reviews for a resource by name
func NewReviewer(subjectAccessEvaluator *subjectlocator.SubjectAccessEvaluator) Reviewer {
	return &reviewer{
		subjectAccessEvaluator: subjectAccessEvaluator,
	}
}

// Review performs a resource access review for the given resource by name
func (r *reviewer) Review(name string) ([]rbacv1.Subject, error) {
	action := authorizer.AttributesRecord{
		Verb:            "get",
		Namespace:       name,
		APIGroup:        corev1.GroupName,
		Resource:        "namespaces",
		Name:            name,
		ResourceRequest: true,
	}
	return r.subjectAccessEvaluator.AllowedSubjects(action)
}
