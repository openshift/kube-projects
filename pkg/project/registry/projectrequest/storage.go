package projectrequest

import (
	"errors"
	"fmt"

	"github.com/golang/glog"

	kapi "k8s.io/kubernetes/pkg/api"
	kapierror "k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/apis/rbac"

	"k8s.io/kubernetes/pkg/api/meta"
	"k8s.io/kubernetes/pkg/api/rest"
	"k8s.io/kubernetes/pkg/api/unversioned"
	authorizationapi "k8s.io/kubernetes/pkg/apis/authorization"
	"k8s.io/kubernetes/pkg/auth/authorizer"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	kclient "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/kubectl/resource"
	"k8s.io/kubernetes/pkg/runtime"
	utilerrors "k8s.io/kubernetes/pkg/util/errors"
	utilruntime "k8s.io/kubernetes/pkg/util/runtime"
	"k8s.io/kubernetes/pkg/util/wait"

	projectapi "github.com/openshift/kube-projects/pkg/project/api"
	projectutil "github.com/openshift/kube-projects/pkg/project/util"
)

type REST struct {
	message string

	authorizer authorizer.Authorizer

	privilegedKubeClient internalclientset.Interface
}

func NewREST(message string, authorizer authorizer.Authorizer, privilegedKubeClient internalclientset.Interface) *REST {
	return &REST{
		message:              message,
		authorizer:           authorizer,
		privilegedKubeClient: privilegedKubeClient,
	}
}

func (r *REST) New() runtime.Object {
	return &projectapi.ProjectRequest{}
}

func (r *REST) NewList() runtime.Object {
	return &unversioned.Status{}
}

var _ = rest.Creater(&REST{})

func (r *REST) Create(ctx kapi.Context, obj runtime.Object) (runtime.Object, error) {
	userInfo, exists := kapi.UserFrom(ctx)
	if !exists {
		return nil, errors.New("a user must be provided")
	}

	if err := rest.BeforeCreate(Strategy, ctx, obj); err != nil {
		return nil, err
	}

	projectRequest := obj.(*projectapi.ProjectRequest)

	if _, err := r.privilegedKubeClient.Core().Namespaces().Get(projectRequest.Name); err == nil {
		return nil, kapierror.NewAlreadyExists(projectapi.Resource("project"), projectRequest.Name)
	}

	ns := projectRequest.Name
	username := userInfo.GetName()

	namespace := &kapi.Namespace{}
	namespace.Name = ns
	namespace.Annotations = map[string]string{
		projectapi.ProjectDescription: projectRequest.Description,
		projectapi.ProjectDisplayName: projectRequest.DisplayName,
		projectapi.ProjectRequester:   username,
	}
	resultingNamespace, err := r.privilegedKubeClient.Core().Namespaces().Create(namespace)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("error namespace %q: %v", projectRequest.Name, err))
		return nil, kapierror.NewInternalError(err)
	}

	binding := &rbac.RoleBinding{}
	binding.Name = "admin"
	binding.Namespace = ns
	binding.Subjects = []rbac.Subject{{Kind: rbac.UserKind, Name: username}}
	binding.RoleRef.Name = "admin"
	if _, err := r.privilegedKubeClient.Rbac().RoleBindings().Create(binding); err != nil {
		utilruntime.HandleError(fmt.Errorf("error rolebinding in %q: %v", projectRequest.Name, err))
		return nil, kapierror.NewInternalError(err)
	}

	r.waitForRoleBinding(projectName, lastRoleBinding.Name)

	return projectutil.ConvertNamespace(resultingNamespace), nil
}

// waitForAccess blocks until the apiserver says the user has access to the namespace
func (r *REST) waitForAccess(namespace, username string) {
	sar := authorizationapi.SubjectAccessReview{
		Spec: authorizationapi.SubjectAccessReviewSpec{
			ResourceAttributes: &authorizationapi.ResourceAttributes{
				Namespace: namespace,
				Verb:      "get",
				Group:     kapi.GroupName,
				Resource:  "namespaces",
				Name:      namespace,
			},
			User: username,
		},
	}

	// we have a rolebinding, the we check the cache we have to see if its been updated with this rolebinding
	// if you share a cache with our authorizer (you should), then this will let you know when the authorizer is ready.
	// doesn't matter if this failed.  When the call returns, return.  If we have access great.  If not, oh well.
	backoff := kclient.DefaultBackoff
	backoff.Steps = 6 // this effectively waits for 6-ish seconds
	err := wait.ExponentialBackoff(backoff, func() (bool, error) {
		result, err := r.privilegedKubeClient.Authorization().SubjectAccessReviews().Create(sar)
		if err != nil {
			return false, err
		}

		return result.Status.Allowed, nil
	})

	if err != nil {
		glog.V(4).Infof("authorization cache failed to update for %v %v: %v", namespace, name, err)
	}
}

var _ = rest.Lister(&REST{})

func (r *REST) List(ctx kapi.Context, options *kapi.ListOptions) (runtime.Object, error) {
	userInfo, exists := kapi.UserFrom(ctx)
	if !exists {
		return nil, errors.New("a user must be provided")
	}

	accessCheck := authorizer.AttributesRecord{
		User:            attributes.GetUserInfo(),
		Verb:            "create",
		Namespace:       "",
		APIGroup:        projectapi.GroupName,
		Resource:        "projectrequests",
		Subresource:     "",
		Name:            "",
		ResourceRequest: true,
		Path:            "",
	}
	allowed, _, _ := a.authorizer.Authorize(accessCheck)
	if allowed {
		return &unversioned.Status{Status: unversioned.StatusSuccess}, nil
	}

	forbiddenError := kapierror.NewForbidden(projectapi.Resource("projectrequest"), "", errors.New("you may not request a new project via this API."))
	if len(r.message) > 0 {
		forbiddenError.ErrStatus.Message = r.message
		forbiddenError.ErrStatus.Details = &unversioned.StatusDetails{
			Group: projectapi.GroupName,
			Kind:  "ProjectRequest",
			Causes: []unversioned.StatusCause{
				{Message: r.message},
			},
		}
	} else {
		forbiddenError.ErrStatus.Message = "You may not request a new project via this API."
	}
	return nil, forbiddenError
}
