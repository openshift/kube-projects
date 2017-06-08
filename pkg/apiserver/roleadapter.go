package apiserver

import (
	"k8s.io/apimachinery/pkg/labels"
	rbacapi "k8s.io/kubernetes/pkg/apis/rbac"
	rbaclisters "k8s.io/kubernetes/pkg/client/listers/rbac/internalversion"
)

type roleGetter struct {
	lister rbaclisters.RoleLister
}

func (g *roleGetter) GetRole(namespace, name string) (*rbacapi.Role, error) {
	return g.lister.Roles(namespace).Get(name)
}

type roleBindingLister struct {
	lister rbaclisters.RoleBindingLister
}

func (l *roleBindingLister) ListRoleBindings(namespace string) ([]*rbacapi.RoleBinding, error) {
	return l.lister.RoleBindings(namespace).List(labels.Everything())
}

type clusterRoleGetter struct {
	lister rbaclisters.ClusterRoleLister
}

func (g *clusterRoleGetter) GetClusterRole(name string) (*rbacapi.ClusterRole, error) {
	return g.lister.Get(name)
}

type clusterRoleBindingLister struct {
	lister rbaclisters.ClusterRoleBindingLister
}

func (l *clusterRoleBindingLister) ListClusterRoleBindings() ([]*rbacapi.ClusterRoleBinding, error) {
	return l.lister.List(labels.Everything())
}
