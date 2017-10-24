package projectrequest

import (
	"errors"
	"fmt"

	"github.com/golang/glog"

	kapierror "k8s.io/apimachinery/pkg/api/errors"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	kapi "k8s.io/kubernetes/pkg/api"
	authorizationapi "k8s.io/kubernetes/pkg/apis/authorization"
	"k8s.io/kubernetes/pkg/apis/rbac"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	"k8s.io/kubernetes/pkg/client/retry"

	projectapi "github.com/openshift/kube-projects/pkg/apis/project"
	projectutil "github.com/openshift/kube-projects/pkg/project/util"
)

type REST struct {
	message string

	authorizer authorizer.Authorizer

	kubeClient internalclientset.Interface
}

func NewREST(message string, authorizer authorizer.Authorizer, kubeClient internalclientset.Interface) *REST {
	return &REST{
		message:    message,
		authorizer: authorizer,
		kubeClient: kubeClient,
	}
}

func (r *REST) New() runtime.Object {
	return &projectapi.ProjectRequest{}
}

func (r *REST) NewList() runtime.Object {
	return &metav1.Status{}
}

var _ = rest.Creater(&REST{})

func (r *REST) Create(ctx request.Context, obj runtime.Object, includeUninitialized bool) (runtime.Object, error) {
	userInfo, exists := request.UserFrom(ctx)
	if !exists {
		return nil, errors.New("a user must be provided")
	}

	if err := rest.BeforeCreate(Strategy, ctx, obj); err != nil {
		return nil, err
	}

	projectRequest := obj.(*projectapi.ProjectRequest)

	if _, err := r.kubeClient.Core().Namespaces().Get(projectRequest.Name, metav1.GetOptions{}); err == nil {
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
	resultingNamespace, err := r.kubeClient.Core().Namespaces().Create(namespace)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("error namespace %q: %v", projectRequest.Name, err))
		return nil, kapierror.NewInternalError(err)
	}

	binding := &rbac.RoleBinding{}
	binding.Name = "admin"
	binding.Namespace = ns
	binding.Subjects = []rbac.Subject{{Kind: rbac.UserKind, Name: username}}
	binding.RoleRef.Kind = "ClusterRole"
	binding.RoleRef.Name = "admin"
	if _, err := r.kubeClient.Rbac().RoleBindings(ns).Create(binding); err != nil {
		utilruntime.HandleError(fmt.Errorf("error rolebinding in %q: %v", projectRequest.Name, err))
		return nil, kapierror.NewInternalError(err)
	}

	binding.Name = projectapi.GroupName + ":admin"
	binding.RoleRef.Name = projectapi.GroupName + ":admin"
	if _, err := r.kubeClient.Rbac().RoleBindings(ns).Create(binding); err != nil {
		utilruntime.HandleError(fmt.Errorf("error rolebinding in %q: %v", projectRequest.Name, err))
		return nil, kapierror.NewInternalError(err)
	}

	r.waitForAccess(ns, username)

	return projectutil.ConvertNamespace(resultingNamespace), nil
}

// waitForAccess blocks until the apiserver says the user has access to the namespace
func (r *REST) waitForAccess(namespace, username string) {
	sar := &authorizationapi.SubjectAccessReview{
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
	backoff := retry.DefaultBackoff
	backoff.Steps = 6 // this effectively waits for 6-ish seconds
	err := wait.ExponentialBackoff(backoff, func() (bool, error) {
		result, err := r.kubeClient.Authorization().SubjectAccessReviews().Create(sar)
		if err != nil {
			return false, err
		}

		return result.Status.Allowed, nil
	})

	if err != nil {
		glog.V(4).Infof("authorization cache failed to update for %v %v: %v", namespace, username, err)
	}
}

var _ = rest.Lister(&REST{})

func (r *REST) List(ctx request.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	userInfo, exists := request.UserFrom(ctx)
	if !exists {
		return nil, errors.New("a user must be provided")
	}

	accessCheck := authorizer.AttributesRecord{
		User:            userInfo,
		Verb:            "create",
		Namespace:       "",
		APIGroup:        projectapi.GroupName,
		Resource:        "projectrequests",
		Subresource:     "",
		Name:            "",
		ResourceRequest: true,
		Path:            "",
	}
	allowed, _, _ := r.authorizer.Authorize(accessCheck)
	if allowed {
		return &metav1.Status{Status: metav1.StatusSuccess}, nil
	}

	forbiddenError := kapierror.NewForbidden(projectapi.Resource("projectrequest"), "", errors.New("you may not request a new project via this API."))
	if len(r.message) > 0 {
		forbiddenError.ErrStatus.Message = r.message
		forbiddenError.ErrStatus.Details = &metav1.StatusDetails{
			Group: projectapi.GroupName,
			Kind:  "ProjectRequest",
			Causes: []metav1.StatusCause{
				{Message: r.message},
			},
		}
	} else {
		forbiddenError.ErrStatus.Message = "You may not request a new project via this API."
	}
	return nil, forbiddenError
}
