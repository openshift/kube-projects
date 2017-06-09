package auth

import (
	"fmt"
	"strconv"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/client-go/tools/cache"
	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apis/rbac"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset/fake"
	informers "k8s.io/kubernetes/pkg/client/informers/informers_generated/internalversion"
	corelisters "k8s.io/kubernetes/pkg/client/listers/core/internalversion"
	"k8s.io/kubernetes/pkg/controller"
)

// common test users
var (
	alice = &user.DefaultInfo{
		Name:   "Alice",
		UID:    "alice-uid",
		Groups: []string{},
	}
	bob = &user.DefaultInfo{
		Name:   "Bob",
		UID:    "bob-uid",
		Groups: []string{"employee"},
	}
	eve = &user.DefaultInfo{
		Name:   "Eve",
		UID:    "eve-uid",
		Groups: []string{"employee"},
	}
	frank = &user.DefaultInfo{
		Name:   "Frank",
		UID:    "frank-uid",
		Groups: []string{},
	}
)

// mockReviewer returns the specified values for each supplied resource
type mockReviewer struct {
	expectedResults map[string][]rbac.Subject
}

// Review returns the mapped review from the mock object, or an error if none exists
func (mr *mockReviewer) Review(name string) ([]rbac.Subject, error) {
	review := mr.expectedResults[name]
	if review == nil {
		return nil, fmt.Errorf("Item %s does not exist", name)
	}
	return review, nil
}

func validateList(t *testing.T, lister Lister, user user.Info, expectedSet sets.String) {
	namespaceList, err := lister.List(user)
	if err != nil {
		t.Errorf("Unexpected error %v", err)
	}
	results := sets.String{}
	for _, namespace := range namespaceList.Items {
		results.Insert(namespace.Name)
	}
	if results.Len() != expectedSet.Len() || !results.HasAll(expectedSet.List()...) {
		t.Errorf("User %v, Expected: %v, Actual: %v", user.GetName(), expectedSet, results)
	}
}

func TestSyncNamespace(t *testing.T) {
	namespaceList := []*kapi.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "foo", ResourceVersion: "1"},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "bar", ResourceVersion: "2"},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "car", ResourceVersion: "3"},
		},
	}
	informerFactory := informers.NewSharedInformerFactory(fake.NewSimpleClientset(), controller.NoResyncPeriodFunc())

	reviewer := &mockReviewer{
		expectedResults: map[string][]rbac.Subject{
			"foo": []rbac.Subject{
				{Kind: rbac.UserKind, Name: alice.GetName()},
				{Kind: rbac.UserKind, Name: bob.GetName()},
				{Kind: rbac.GroupKind, Name: "employee"},
			},
			"bar": []rbac.Subject{
				{Kind: rbac.UserKind, Name: frank.GetName()},
				{Kind: rbac.UserKind, Name: eve.GetName()},
				{Kind: rbac.GroupKind, Name: "random"},
			},
			"car": []rbac.Subject{},
		},
	}

	nsIndexer := cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})

	authorizationCache := NewAuthorizationCache(
		reviewer,
		informerFactory.Core().InternalVersion().Namespaces(),
		informerFactory.Rbac().InternalVersion().ClusterRoles(),
		informerFactory.Rbac().InternalVersion().ClusterRoleBindings(),
		informerFactory.Rbac().InternalVersion().Roles(),
		informerFactory.Rbac().InternalVersion().RoleBindings(),
	)
	authorizationCache.namespaceLister = corelisters.NewNamespaceLister(nsIndexer)
	// we prime the data we need here since we are not running reflectors
	for _, ns := range namespaceList {
		obj, _ := kapi.Scheme.Copy(ns)
		nsIndexer.Add(obj.(*kapi.Namespace))
	}
	authorizationCache.skip = &neverSkipSynchronizer{}

	// synchronize the cache
	authorizationCache.synchronize()

	validateList(t, authorizationCache, alice, sets.NewString("foo"))
	validateList(t, authorizationCache, bob, sets.NewString("foo"))
	validateList(t, authorizationCache, eve, sets.NewString("foo", "bar"))
	validateList(t, authorizationCache, frank, sets.NewString("bar"))

	// modify access rules
	reviewer.expectedResults["foo"] = []rbac.Subject{
		{Kind: rbac.UserKind, Name: bob.GetName()},
		{Kind: rbac.GroupKind, Name: "random"},
	}
	reviewer.expectedResults["bar"] = []rbac.Subject{
		{Kind: rbac.UserKind, Name: alice.GetName()},
		{Kind: rbac.UserKind, Name: eve.GetName()},
		{Kind: rbac.GroupKind, Name: "employee"},
	}
	reviewer.expectedResults["car"] = []rbac.Subject{
		{Kind: rbac.UserKind, Name: bob.GetName()},
		{Kind: rbac.UserKind, Name: eve.GetName()},
		{Kind: rbac.GroupKind, Name: "employee"},
	}

	// modify resource version on each namespace to simulate a change had occurred to force cache refresh
	for i := range namespaceList {
		namespace := namespaceList[i]
		oldVersion, err := strconv.Atoi(namespace.ResourceVersion)
		if err != nil {
			t.Errorf("Bad test setup, resource versions should be numbered, %v", err)
		}
		newVersion := strconv.Itoa(oldVersion + 1)
		namespace.ResourceVersion = newVersion
		nsIndexer.Add(namespace)
	}

	// now refresh the cache (which is resource version aware)
	authorizationCache.synchronize()

	// make sure new rights hold
	validateList(t, authorizationCache, alice, sets.NewString("bar"))
	validateList(t, authorizationCache, bob, sets.NewString("foo", "bar", "car"))
	validateList(t, authorizationCache, eve, sets.NewString("bar", "car"))
	validateList(t, authorizationCache, frank, sets.NewString())
}
