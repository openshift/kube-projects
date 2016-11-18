package bootstraproutes

import (
	"net/http"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/v1"
	rbac "k8s.io/kubernetes/pkg/apis/rbac"
	rbacv1alpha1 "k8s.io/kubernetes/pkg/apis/rbac/v1alpha1"
	"k8s.io/kubernetes/pkg/auth/user"
	"k8s.io/kubernetes/pkg/runtime"
)

// Index provides a webservice for the http root / listing all known paths.
type RBAC struct {
	ServerUser string
	AuthUser   string
}

// Install adds the Index webservice to the given mux.
func (i RBAC) Install(mux *http.ServeMux) {
	mux.HandleFunc("/bootstrap/rbac", func(w http.ResponseWriter, r *http.Request) {
		resourceList := i.rbacResources()

		encoder := api.Codecs.LegacyCodec(rbacv1alpha1.SchemeGroupVersion, v1.SchemeGroupVersion)
		if err := runtime.EncodeList(encoder, resourceList.Items); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		objBytes, err := runtime.Encode(encoder, resourceList)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Write(objBytes)
	})
}

func (i RBAC) rbacResources() *api.List {
	ret := &api.List{}

	rolebindings := i.clusterRoleBindings()
	for i := range rolebindings {
		ret.Items = append(ret.Items, &rolebindings[i])
	}

	roles := i.clusterRoles()
	for i := range roles {
		ret.Items = append(ret.Items, &roles[i])
	}

	return ret
}

var (
	readWrite = []string{"get", "list", "watch", "create", "update", "patch", "delete", "deletecollection"}
	read      = []string{"get", "list", "watch"}
)

const (
	projectGroup = "project.openshift.io"
	rbacGroup    = "rbac.authorization.k8s.io"
)

func (i RBAC) clusterRoles() []rbac.ClusterRole {
	return []rbac.ClusterRole{
		{
			ObjectMeta: api.ObjectMeta{Name: projectGroup + ":admin"},
			Rules: []rbac.PolicyRule{
				rbac.NewRule(readWrite...).Groups(projectGroup).Resources("projects").RuleOrDie(),
			},
		},
		{
			ObjectMeta: api.ObjectMeta{Name: projectGroup + ":editor"},
			Rules: []rbac.PolicyRule{
				rbac.NewRule(read...).Groups(projectGroup).Resources("projects").RuleOrDie(),
			},
		},
		{
			ObjectMeta: api.ObjectMeta{Name: projectGroup + ":viewer"},
			Rules: []rbac.PolicyRule{
				rbac.NewRule(read...).Groups(projectGroup).Resources("projects").RuleOrDie(),
			},
		},
		{
			ObjectMeta: api.ObjectMeta{Name: projectGroup + ":basic-user"},
			Rules: []rbac.PolicyRule{
				rbac.NewRule("list", "watch").Groups(projectGroup).Resources("projects", "projectrequests").RuleOrDie(),
			},
		},
		{
			// role bound at the cluster scope to allow users to self-provision projects
			ObjectMeta: api.ObjectMeta{Name: projectGroup + ":self-provisioner"},
			Rules: []rbac.PolicyRule{
				rbac.NewRule("create").Groups(projectGroup).Resources("projectrequests").RuleOrDie(),
			},
		},
		{
			// role bound at the cluster scope to allow this server to filter namespaces via project
			ObjectMeta: api.ObjectMeta{Name: projectGroup + ":server"},
			Rules: []rbac.PolicyRule{
				rbac.NewRule(readWrite...).Groups("").Resources("namespaces").RuleOrDie(),
				rbac.NewRule(read...).Groups(rbacGroup).Resources("clusterroles", "clusterrolebindings", "roles", "rolebindings").RuleOrDie(),
			},
		},
	}
}

func (i RBAC) clusterRoleBindings() []rbac.ClusterRoleBinding {
	// we need this role so that we can run delegated auth checks
	auth := rbac.NewClusterBinding("system:auth-delegator").Users(i.AuthUser).BindingOrDie()
	auth.Name = projectGroup + ":" + auth.Name

	// we grant admin, so we need admin across all namespaces
	admin := rbac.NewClusterBinding("admin").Users(i.ServerUser).BindingOrDie()
	admin.Name = projectGroup + ":admin"

	// we grant this role, so we need it across all namespaces
	projectAdmin := rbac.NewClusterBinding(projectGroup + ":admin").Users(i.ServerUser).BindingOrDie()
	projectAdmin.Name = projectGroup + ":" + projectGroup + ":admin"

	return []rbac.ClusterRoleBinding{
		auth,
		admin,
		projectAdmin,
		rbac.NewClusterBinding(projectGroup + ":server").Users(i.ServerUser).BindingOrDie(),
		rbac.NewClusterBinding(projectGroup + ":self-provisioner").Groups(user.AllAuthenticated).BindingOrDie(),
		rbac.NewClusterBinding(projectGroup+":basic-user").Groups(user.AllAuthenticated, user.AllUnauthenticated).BindingOrDie(),
	}
}
