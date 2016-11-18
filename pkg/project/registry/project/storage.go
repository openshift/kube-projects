package project

import (
	"fmt"

	kapi "k8s.io/kubernetes/pkg/api"
	kerrors "k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/api/meta"
	"k8s.io/kubernetes/pkg/api/rest"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/client/cache"
	coreclient "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset/typed/core/internalversion"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"
	nsregistry "k8s.io/kubernetes/pkg/registry/core/namespace"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/storage"
	"k8s.io/kubernetes/pkg/util/sets"
	"k8s.io/kubernetes/pkg/watch"

	projectapi "github.com/openshift/kube-projects/pkg/project/api"
	projectauth "github.com/openshift/kube-projects/pkg/project/auth"
	projectutil "github.com/openshift/kube-projects/pkg/project/util"
)

type REST struct {
	// client can modify Kubernetes namespaces
	client coreclient.NamespaceInterface
	// lister can enumerate project lists that enforce policy
	lister projectauth.Lister
	// Allows extended behavior during creation, required
	createStrategy rest.RESTCreateStrategy
	// Allows extended behavior during updates, required
	updateStrategy rest.RESTUpdateStrategy

	authCache *projectauth.AuthorizationCache
	nsLister  *cache.IndexerToNamespaceLister
}

// NewREST returns a RESTStorage object that will work against Project resources
func NewREST(client coreclient.NamespaceInterface, lister projectauth.Lister, authCache *projectauth.AuthorizationCache, nsLister *cache.IndexerToNamespaceLister) *REST {
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
func (s *REST) List(ctx kapi.Context, options *kapi.ListOptions) (runtime.Object, error) {
	user, ok := kapi.UserFrom(ctx)
	if !ok {
		return nil, kerrors.NewForbidden(projectapi.Resource("project"), "", fmt.Errorf("unable to list projects without a user on the context"))
	}
	namespaceList, err := s.lister.List(user)
	if err != nil {
		return nil, err
	}
	m := nsregistry.MatchNamespace(ListOptionsToSelectors(options))
	list, err := filterList(namespaceList, m, nil)
	if err != nil {
		return nil, err
	}
	return projectutil.ConvertNamespaceList(list.(*kapi.NamespaceList)), nil
}

func (s *REST) Watch(ctx kapi.Context, options *kapi.ListOptions) (watch.Interface, error) {
	if ctx == nil {
		return nil, fmt.Errorf("Context is nil")
	}
	userInfo, exists := kapi.UserFrom(ctx)
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
func (s *REST) Get(ctx kapi.Context, name string) (runtime.Object, error) {
	namespace, err := s.client.Get(name)
	if err != nil {
		return nil, err
	}
	return projectutil.ConvertNamespace(namespace), nil
}

var _ = rest.Creater(&REST{})

// Create registers the given Project.
func (s *REST) Create(ctx kapi.Context, obj runtime.Object) (runtime.Object, error) {
	project, ok := obj.(*projectapi.Project)
	if !ok {
		return nil, fmt.Errorf("not a project: %#v", obj)
	}
	kapi.FillObjectMetaSystemFields(ctx, &project.ObjectMeta)
	s.createStrategy.PrepareForCreate(ctx, obj)
	if errs := s.createStrategy.Validate(ctx, obj); len(errs) > 0 {
		return nil, kerrors.NewInvalid(projectapi.Kind("Project"), project.Name, errs)
	}
	namespace, err := s.client.Create(projectutil.ConvertProject(project))
	if err != nil {
		return nil, err
	}
	return projectutil.ConvertNamespace(namespace), nil
}

var _ = rest.Updater(&REST{})

func (s *REST) Update(ctx kapi.Context, name string, objInfo rest.UpdatedObjectInfo) (runtime.Object, bool, error) {
	oldObj, err := s.Get(ctx, name)
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

	namespace, err := s.client.Update(projectutil.ConvertProject(project))
	if err != nil {
		return nil, false, err
	}

	return projectutil.ConvertNamespace(namespace), false, nil
}

var _ = rest.Deleter(&REST{})

// Delete deletes a Project specified by its name
func (s *REST) Delete(ctx kapi.Context, name string) (runtime.Object, error) {
	return &unversioned.Status{Status: unversioned.StatusSuccess}, s.client.Delete(name, nil)
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

func ListOptionsToSelectors(options *kapi.ListOptions) (labels.Selector, fields.Selector) {
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
