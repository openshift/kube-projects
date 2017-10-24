package project

import (
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/endpoints/request"
	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset/fake"

	"github.com/openshift/kube-projects/pkg/apis/project"
)

// mockLister returns the namespaces in the list
type mockLister struct {
	namespaceList *kapi.NamespaceList
}

func (ml *mockLister) List(user user.Info) (*kapi.NamespaceList, error) {
	return ml.namespaceList, nil
}

func TestListProjects(t *testing.T) {
	namespaceList := kapi.NamespaceList{
		Items: []kapi.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "foo"},
			},
		},
	}
	mockClient := fake.NewSimpleClientset(&namespaceList)
	storage := REST{
		client: mockClient.Core().Namespaces(),
		lister: &mockLister{&namespaceList},
	}
	user := &user.DefaultInfo{
		Name:   "test-user",
		UID:    "test-uid",
		Groups: []string{"test-groups"},
	}
	ctx := request.WithUser(request.NewContext(), user)
	response, err := storage.List(ctx, nil)
	if err != nil {
		t.Errorf("%#v should be nil.", err)
	}
	projects := response.(*api.ProjectList)
	if len(projects.Items) != 1 {
		t.Errorf("%#v projects.Items should have len 1.", projects.Items)
	}
	responseProject := projects.Items[0]
	if e, r := responseProject.Name, "foo"; e != r {
		t.Errorf("%#v != %#v.", e, r)
	}
}

func TestCreateProjectBadObject(t *testing.T) {
	storage := REST{}

	obj, err := storage.Create(request.NewContext(), &api.ProjectList{}, false)
	if obj != nil {
		t.Errorf("Expected nil, got %v", obj)
	}
	if strings.Index(err.Error(), "not a project:") == -1 {
		t.Errorf("Expected 'not an project' error, got %v", err)
	}
}

func TestCreateInvalidProject(t *testing.T) {
	mockClient := &fake.Clientset{}
	storage := NewREST(mockClient.Core().Namespaces(), &mockLister{}, nil, nil)
	_, err := storage.Create(request.NewContext(), &api.Project{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{"openshift.io/display-name": "h\t\ni"},
		},
	}, false)
	if !errors.IsInvalid(err) {
		t.Errorf("Expected 'invalid' error, got %v", err)
	}
}

func TestCreateProjectOK(t *testing.T) {
	mockClient := &fake.Clientset{}
	storage := NewREST(mockClient.Core().Namespaces(), &mockLister{}, nil, nil)
	_, err := storage.Create(request.NewContext(), &api.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "foo"},
	}, false)
	if err != nil {
		t.Errorf("Unexpected non-nil error: %#v", err)
	}
	if len(mockClient.Actions()) != 1 {
		t.Errorf("Expected client action for create")
	}
	if !mockClient.Actions()[0].Matches("create", "namespaces") {
		t.Errorf("Expected call to create-namespace")
	}
}

func TestGetProjectOK(t *testing.T) {
	mockClient := fake.NewSimpleClientset(&kapi.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "foo"}})
	storage := NewREST(mockClient.Core().Namespaces(), &mockLister{}, nil, nil)
	project, err := storage.Get(request.NewContext(), "foo", nil)
	if project == nil {
		t.Error("Unexpected nil project")
	}
	if err != nil {
		t.Errorf("Unexpected non-nil error: %v", err)
	}
	if project.(*api.Project).Name != "foo" {
		t.Errorf("Unexpected project: %#v", project)
	}
}

func TestDeleteProject(t *testing.T) {
	mockClient := &fake.Clientset{}
	storage := REST{
		client: mockClient.Core().Namespaces(),
	}
	obj, err := storage.Delete(request.NewContext(), "foo")
	if obj == nil {
		t.Error("Unexpected nil obj")
	}
	if err != nil {
		t.Errorf("Unexpected non-nil error: %#v", err)
	}
	status, ok := obj.(*metav1.Status)
	if !ok {
		t.Errorf("Expected status type, got: %#v", obj)
	}
	if status.Status != metav1.StatusSuccess {
		t.Errorf("Expected status=success, got: %#v", status)
	}
	if len(mockClient.Actions()) != 1 {
		t.Errorf("Expected client action for delete")
	}
	if !mockClient.Actions()[0].Matches("delete", "namespaces") {
		t.Errorf("Expected call to delete-namespace")
	}
}
