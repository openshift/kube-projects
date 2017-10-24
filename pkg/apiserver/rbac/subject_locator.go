package rbac

import (
	rbacv1 "k8s.io/api/rbac/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/authorization/authorizer"
)

type RoleToRuleMapper interface {
	// GetRoleReferenceRules attempts to resolve the role reference of a RoleBinding or ClusterRoleBinding.  The passed namespace should be the namespace
	// of the role binding, the empty string if a cluster role binding.
	GetRoleReferenceRules(roleRef rbacv1.RoleRef, namespace string) ([]rbacv1.PolicyRule, error)
}

type SubjectLocator interface {
	AllowedSubjects(attributes authorizer.Attributes) ([]rbacv1.Subject, error)
}

var _ = SubjectLocator(&SubjectAccessEvaluator{})

type SubjectAccessEvaluator struct {
	superUser string

	roleBindingLister        RoleBindingLister
	clusterRoleBindingLister ClusterRoleBindingLister
	roleToRuleMapper         RoleToRuleMapper
}

func NewSubjectAccessEvaluator(roles RoleGetter, roleBindings RoleBindingLister, clusterRoles ClusterRoleGetter, clusterRoleBindings ClusterRoleBindingLister, superUser string) *SubjectAccessEvaluator {
	subjectLocator := &SubjectAccessEvaluator{
		superUser:                superUser,
		roleBindingLister:        roleBindings,
		clusterRoleBindingLister: clusterRoleBindings,
		roleToRuleMapper: NewDefaultRuleResolver(
			roles, roleBindings, clusterRoles, clusterRoleBindings,
		),
	}
	return subjectLocator
}

// AllowedSubjects returns the subjects that can perform an action and any errors encountered while computing the list.
// It is possible to have both subjects and errors returned if some rolebindings couldn't be resolved, but others could be.
func (r *SubjectAccessEvaluator) AllowedSubjects(requestAttributes authorizer.Attributes) ([]rbacv1.Subject, error) {
	subjects := []rbacv1.Subject{{Kind: rbacv1.GroupKind, APIGroup: rbacv1.GroupName, Name: user.SystemPrivilegedGroup}}
	if len(r.superUser) > 0 {
		subjects = append(subjects, rbacv1.Subject{Kind: rbacv1.UserKind, APIGroup: rbacv1.GroupName, Name: r.superUser})
	}
	errorlist := []error{}

	if clusterRoleBindings, err := r.clusterRoleBindingLister.ListClusterRoleBindings(); err != nil {
		errorlist = append(errorlist, err)

	} else {
		for _, clusterRoleBinding := range clusterRoleBindings {
			rules, err := r.roleToRuleMapper.GetRoleReferenceRules(clusterRoleBinding.RoleRef, "")
			if err != nil {
				// if we have an error, just keep track of it and keep processing.  Since rules are additive,
				// missing a reference is bad, but we can continue with other rolebindings and still have a list
				// that does not contain any invalid values
				errorlist = append(errorlist, err)
			}
			if RulesAllow(requestAttributes, rules...) {
				subjects = append(subjects, clusterRoleBinding.Subjects...)
			}
		}
	}

	if namespace := requestAttributes.GetNamespace(); len(namespace) > 0 {
		if roleBindings, err := r.roleBindingLister.ListRoleBindings(namespace); err != nil {
			errorlist = append(errorlist, err)

		} else {
			for _, roleBinding := range roleBindings {
				rules, err := r.roleToRuleMapper.GetRoleReferenceRules(roleBinding.RoleRef, namespace)
				if err != nil {
					// if we have an error, just keep track of it and keep processing.  Since rules are additive,
					// missing a reference is bad, but we can continue with other rolebindings and still have a list
					// that does not contain any invalid values
					errorlist = append(errorlist, err)
				}
				if RulesAllow(requestAttributes, rules...) {
					subjects = append(subjects, roleBinding.Subjects...)
				}
			}
		}
	}

	dedupedSubjects := []rbacv1.Subject{}
	for _, subject := range subjects {
		found := false
		for _, curr := range dedupedSubjects {
			if curr == subject {
				found = true
				break
			}
		}

		if !found {
			dedupedSubjects = append(dedupedSubjects, subject)
		}
	}

	return subjects, utilerrors.NewAggregate(errorlist)
}
