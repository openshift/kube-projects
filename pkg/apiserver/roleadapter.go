package apiserver

import (
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/labels"
	rbacv1listers "k8s.io/client-go/listers/rbac/v1"
)

type roleGetter struct {
	lister rbacv1listers.RoleLister
}

func (g *roleGetter) GetRole(namespace, name string) (*rbacv1.Role, error) {
	return g.lister.Roles(namespace).Get(name)
}

type roleBindingLister struct {
	lister rbacv1listers.RoleBindingLister
}

func (l *roleBindingLister) ListRoleBindings(namespace string) ([]*rbacv1.RoleBinding, error) {
	return l.lister.RoleBindings(namespace).List(labels.Everything())
}

type clusterRoleGetter struct {
	lister rbacv1listers.ClusterRoleLister
}

func (g *clusterRoleGetter) GetClusterRole(name string) (*rbacv1.ClusterRole, error) {
	return g.lister.Get(name)
}

type clusterRoleBindingLister struct {
	lister rbacv1listers.ClusterRoleBindingLister
}

func (l *clusterRoleBindingLister) ListClusterRoleBindings() ([]*rbacv1.ClusterRoleBinding, error) {
	return l.lister.List(labels.Everything())
}
