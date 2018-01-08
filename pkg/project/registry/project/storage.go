package project

import (
	"fmt"

	"k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/storage"
	apistorage "k8s.io/apiserver/pkg/storage"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	corev1listers "k8s.io/client-go/listers/core/v1"

	projectapi "github.com/openshift/kube-projects/pkg/apis/project"
	projectauth "github.com/openshift/kube-projects/pkg/project/auth"
	projectutil "github.com/openshift/kube-projects/pkg/project/util"
)

type REST struct {
	// client can modify Kubernetes namespaces
	client corev1client.NamespaceInterface
	// lister can enumerate project lists that enforce policy
	lister projectauth.Lister
	// Allows extended behavior during creation, required
	createStrategy rest.RESTCreateStrategy
	// Allows extended behavior during updates, required
	updateStrategy rest.RESTUpdateStrategy

	authCache *projectauth.AuthorizationCache
	nsLister  corev1listers.NamespaceLister
}

// NewREST returns a RESTStorage object that will work against Project resources
func NewREST(client corev1client.NamespaceInterface, lister projectauth.Lister, authCache *projectauth.AuthorizationCache, nsLister corev1listers.NamespaceLister) *REST {
	return &REST{
		client:         client,
		lister:         lister,
		createStrategy: Strategy,
		updateStrategy: Strategy,

		authCache: authCache,
		nsLister:  nsLister,
	}
}

// New returns a new Project
func (s *REST) New() runtime.Object {
	return &projectapi.Project{}
}

// NewList returns a new ProjectList
func (*REST) NewList() runtime.Object {
	return &projectapi.ProjectList{}
}

var _ = rest.Lister(&REST{})

// List retrieves a list of Projects that match label.
func (s *REST) List(ctx request.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	user, ok := request.UserFrom(ctx)
	if !ok {
		return nil, kerrors.NewForbidden(projectapi.Resource("project"), "", fmt.Errorf("unable to list projects without a user on the context"))
	}
	namespaceList, err := s.lister.List(user)
	if err != nil {
		return nil, err
	}
	m := MatchNamespace(ListOptionsToSelectors(options))
	list, err := filterList(namespaceList, m, nil)
	if err != nil {
		return nil, err
	}
	return projectutil.ConvertNamespaceList(list.(*v1.NamespaceList)), nil
}

func (s *REST) Watch(ctx request.Context, options *metav1.ListOptions) (watch.Interface, error) {
	if ctx == nil {
		return nil, fmt.Errorf("Context is nil")
	}
	userInfo, exists := request.UserFrom(ctx)
	if !exists {
		return nil, fmt.Errorf("no user")
	}

	includeAllExistingProjects := (options != nil) && options.ResourceVersion == "0"

	// allowedNamespaces are the namespaces allowed by scopes.  kube has no scopess, see all
	allowedNamespaces := sets.NewString("*")

	watcher := projectauth.NewUserProjectWatcher(userInfo, allowedNamespaces, s.nsLister, s.authCache, includeAllExistingProjects)
	s.authCache.AddWatcher(watcher)

	go watcher.Watch()
	return watcher, nil
}

var _ = rest.Getter(&REST{})

// Get retrieves a Project by name
func (s *REST) Get(ctx request.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	namespace, err := s.client.Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return projectutil.ConvertNamespace(namespace), nil
}

var _ = rest.Creater(&REST{})

// Create registers the given Project.
func (s *REST) Create(ctx request.Context, obj runtime.Object, createValidation rest.ValidateObjectFunc, _ bool) (runtime.Object, error) {
	project, ok := obj.(*projectapi.Project)
	if !ok {
		return nil, fmt.Errorf("not a project: %#v", obj)
	}
	rest.FillObjectMetaSystemFields(ctx, &project.ObjectMeta)
	s.createStrategy.PrepareForCreate(ctx, obj)
	if errs := s.createStrategy.Validate(ctx, obj); len(errs) > 0 {
		return nil, kerrors.NewInvalid(projectapi.Kind("Project"), project.Name, errs)
	}
	if err := createValidation(obj); err != nil {
		return nil, err
	}
	namespace, err := s.client.Create(projectutil.ConvertProject(project))
	if err != nil {
		return nil, err
	}
	return projectutil.ConvertNamespace(namespace), nil
}

var _ = rest.Updater(&REST{})

func (s *REST) Update(ctx request.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc) (runtime.Object, bool, error) {
	oldObj, err := s.Get(ctx, name, nil)
	if err != nil {
		return nil, false, err
	}

	obj, err := objInfo.UpdatedObject(ctx, oldObj)
	if err != nil {
		return nil, false, err
	}

	project, ok := obj.(*projectapi.Project)
	if !ok {
		return nil, false, fmt.Errorf("not a project: %#v", obj)
	}

	s.updateStrategy.PrepareForUpdate(ctx, obj, oldObj)
	if errs := s.updateStrategy.ValidateUpdate(ctx, obj, oldObj); len(errs) > 0 {
		return nil, false, kerrors.NewInvalid(projectapi.Kind("Project"), project.Name, errs)
	}
	if err := updateValidation(obj, oldObj); err != nil {
		return nil, false, err
	}

	namespace, err := s.client.Update(projectutil.ConvertProject(project))
	if err != nil {
		return nil, false, err
	}

	return projectutil.ConvertNamespace(namespace), false, nil
}

var _ = rest.Deleter(&REST{})

// Delete deletes a Project specified by its name
func (s *REST) Delete(ctx request.Context, name string) (runtime.Object, error) {
	return &metav1.Status{Status: metav1.StatusSuccess}, s.client.Delete(name, nil)
}

// decoratorFunc can mutate the provided object prior to being returned.
type decoratorFunc func(obj runtime.Object) error

// filterList filters any list object that conforms to the api conventions,
// provided that 'm' works with the concrete type of list. d is an optional
// decorator for the returned functions. Only matching items are decorated.
func filterList(list runtime.Object, m storage.SelectionPredicate, d decoratorFunc) (filtered runtime.Object, err error) {
	// TODO: push a matcher down into tools.etcdHelper to avoid all this
	// nonsense. This is a lot of unnecessary copies.
	items, err := meta.ExtractList(list)
	if err != nil {
		return nil, err
	}
	var filteredItems []runtime.Object
	for _, obj := range items {
		match, err := m.Matches(obj)
		if err != nil {
			return nil, err
		}
		if match {
			if d != nil {
				if err := d(obj); err != nil {
					return nil, err
				}
			}
			filteredItems = append(filteredItems, obj)
		}
	}
	err = meta.SetList(list, filteredItems)
	if err != nil {
		return nil, err
	}
	return list, nil
}

func ListOptionsToSelectors(options *metainternalversion.ListOptions) (labels.Selector, fields.Selector) {
	label := labels.Everything()
	if options != nil && options.LabelSelector != nil {
		label = options.LabelSelector
	}
	field := fields.Everything()
	if options != nil && options.FieldSelector != nil {
		field = options.FieldSelector
	}
	return label, field
}

// GetAttrs returns labels and fields of a given object for filtering purposes.
func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, bool, error) {
	namespaceObj, ok := obj.(*v1.Namespace)
	if !ok {
		return nil, nil, false, fmt.Errorf("not a namespace")
	}
	return labels.Set(namespaceObj.Labels), NamespaceToSelectableFields(namespaceObj), namespaceObj.Initializers != nil, nil
}

// MatchNamespace returns a generic matcher for a given label and field selector.
func MatchNamespace(label labels.Selector, field fields.Selector) apistorage.SelectionPredicate {
	return apistorage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

// NamespaceToSelectableFields returns a field set that represents the object
func NamespaceToSelectableFields(namespace *v1.Namespace) fields.Set {
	objectMetaFieldsSet := generic.ObjectMetaFieldsSet(&namespace.ObjectMeta, false)
	specificFieldsSet := fields.Set{
		"status.phase": string(namespace.Status.Phase),
		// This is a bug, but we need to support it for backward compatibility.
		"name": namespace.Name,
	}
	return generic.MergeFieldsSets(objectMetaFieldsSet, specificFieldsSet)
}
